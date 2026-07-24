package middleware

// Tests for the DAV Basic-auth middleware.
//
// These run against a real IMAP server — the in-memory one that ships with
// go-imap (imapserver/imapmemserver) — bound to a random loopback port. That
// exercises the actual imap.Connect path, including the LOGIN handshake, with
// no external mail server and no network access.

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/labstack/echo/v5"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testIMAPUser = "nilton@example.com"
	testIMAPPass = "correct-horse"
)

// countingListener wraps a listener to count accepted connections, which is how
// the tests observe whether the credential cache actually avoided a login.
type countingListener struct {
	net.Listener
	accepted atomic.Int64
}

func (l *countingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err == nil {
		l.accepted.Add(1)
	}
	return conn, err
}

// startMemIMAP boots an in-memory IMAP server holding a single user and returns
// its host, port and connection counter.
func startMemIMAP(t *testing.T) (host string, port int, counter *countingListener) {
	t.Helper()

	mem := imapmemserver.New()
	mem.AddUser(imapmemserver.NewUser(testIMAPUser, testIMAPPass))

	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}, imap.CapIMAP4rev2: {}},
		InsecureAuth: true, // plain loopback connection, no TLS in tests
		Logger:       log.New(io.Discard, "", 0),
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	counter = &countingListener{Listener: ln}

	go func() { _ = srv.Serve(counter) }()
	t.Cleanup(func() { _ = srv.Close() })

	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port, counter
}

// newAuthTestEnv wires the middleware to the in-memory IMAP server and a
// throwaway database.
func newAuthTestEnv(t *testing.T) (*echo.Echo, *countingListener, *gorm.DB) {
	t.Helper()

	host, port, counter := startMemIMAP(t)

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{}
	cfg.IMAP.Host = host
	cfg.IMAP.Port = port
	cfg.IMAP.TLS = false
	cfg.IMAP.TimeoutSec = 5

	e := echo.New()
	e.Add(http.MethodGet, "/dav/*", func(c *echo.Context) error {
		id, _ := c.Get("caldav_user_id").(uint)
		name, _ := c.Get("caldav_username").(string)
		return c.JSON(http.StatusOK, map[string]any{"user_id": id, "username": name})
	}, CalDAVAuth(cfg, db))

	return e, counter, db
}

func request(e *echo.Echo, user, pass string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/dav/whoami", nil)
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestCalDAVAuthRequiresCredentials(t *testing.T) {
	e, _, _ := newAuthTestEnv(t)

	rec := request(e, "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	// Without this header clients never prompt for a password.
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("401 must carry a WWW-Authenticate challenge")
	}
}

func TestCalDAVAuthRejectsWrongPassword(t *testing.T) {
	e, _, _ := newAuthTestEnv(t)

	if rec := request(e, testIMAPUser, "wrong"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if rec := request(e, "nobody@example.com", testIMAPPass); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unknown user status = %d, want 401", rec.Code)
	}
}

func TestCalDAVAuthAcceptsValidCredentialsAndCreatesUser(t *testing.T) {
	e, _, db := newAuthTestEnv(t)

	rec := request(e, testIMAPUser, testIMAPPass)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var user model.User
	if err := db.Where("imap_user = ?", testIMAPUser).First(&user).Error; err != nil {
		t.Fatalf("the user row should be created on first access: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("user has no ID")
	}
}

// DAV clients poll every few minutes and issue several requests per cycle.
// Without the cache each one opens an IMAP connection, which overloads the mail
// server and can trip its brute-force protection.
func TestCalDAVAuthCachesSuccessfulLogins(t *testing.T) {
	e, counter, _ := newAuthTestEnv(t)

	for i := 0; i < 10; i++ {
		if rec := request(e, testIMAPUser, testIMAPPass); rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d", i, rec.Code)
		}
	}

	if got := counter.accepted.Load(); got != 1 {
		t.Fatalf("10 authenticated requests opened %d IMAP connections, want 1", got)
	}
}

// A password that stops working must not keep riding a warm cache entry.
func TestCalDAVAuthDoesNotCacheFailures(t *testing.T) {
	e, counter, _ := newAuthTestEnv(t)

	for i := 0; i < 3; i++ {
		if rec := request(e, testIMAPUser, "wrong"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("request %d: status = %d, want 401", i, rec.Code)
		}
	}
	if got := counter.accepted.Load(); got != 3 {
		t.Fatalf("failed logins opened %d connections, want 3 (failures must not be cached)", got)
	}

	// A different password for the same user is a different cache key.
	if rec := request(e, testIMAPUser, testIMAPPass); rec.Code != http.StatusOK {
		t.Fatalf("valid credentials after failures: status = %d", rec.Code)
	}
}

func TestCredentialCacheExpiresEntries(t *testing.T) {
	cache := newCredentialCache(20 * time.Millisecond)
	cache.put("user", "pass", 7)

	if id, ok := cache.get("user", "pass"); !ok || id != 7 {
		t.Fatalf("fresh entry = (%d, %v)", id, ok)
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := cache.get("user", "pass"); ok {
		t.Fatal("entry should have expired")
	}
}

func TestCredentialCacheKeyCoversUserAndPassword(t *testing.T) {
	cache := newCredentialCache(time.Minute)
	cache.put("alice", "secret", 1)

	if _, ok := cache.get("alice", "other"); ok {
		t.Fatal("a different password must not hit the cache entry")
	}
	if _, ok := cache.get("bob", "secret"); ok {
		t.Fatal("a different user must not hit the cache entry")
	}
	// The digest must not be reversible to the password.
	digest := cache.digest("alice", "secret")
	if digest == "" || len(digest) != 64 {
		t.Fatalf("unexpected digest: %q", digest)
	}
	for _, plaintext := range []string{"alice", "secret"} {
		if digest == plaintext {
			t.Fatal("the cache key must be a digest, not the credential itself")
		}
	}
}
