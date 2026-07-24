package middleware

// Tests for the ActiveSync Basic-auth middleware. They reuse the in-memory IMAP
// server from caldav_auth_test.go, so no external mail server is involved.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-cubemail/internal/config"
	"github.com/labstack/echo/v5"
)

// newEASTestEnv wires EASAuth to the in-memory IMAP server.
func newEASTestEnv(t *testing.T) (*echo.Echo, *countingListener) {
	t.Helper()

	host, port, counter := startMemIMAP(t)

	cfg := &config.Config{}
	cfg.IMAP.Host = host
	cfg.IMAP.Port = port
	cfg.IMAP.TLS = false
	cfg.IMAP.TimeoutSec = 5

	e := echo.New()
	e.Add(http.MethodPost, "/Microsoft-Server-ActiveSync", func(c *echo.Context) error {
		user, _ := c.Get("eas_user").(string)
		pass, _ := c.Get("eas_password").(string)
		return c.JSON(http.StatusOK, map[string]string{"user": user, "has_password": boolLabel(pass != "")})
	}, EASAuth(cfg))

	return e, counter
}

func boolLabel(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func easRequest(e *echo.Echo, user, pass string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/Microsoft-Server-ActiveSync?Cmd=Ping", nil)
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestEASAuthRejectsMissingAndWrongCredentials(t *testing.T) {
	e, _ := newEASTestEnv(t)

	rec := easRequest(e, "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no credentials: status = %d, want 401", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("401 must carry a WWW-Authenticate challenge")
	}
	if rec := easRequest(e, testIMAPUser, "wrong"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: status = %d, want 401", rec.Code)
	}
}

// The handlers need the plaintext password to open their own IMAP connection,
// so it must still reach the context on a cache hit.
func TestEASAuthExposesCredentialsOnCacheHit(t *testing.T) {
	e, _ := newEASTestEnv(t)

	for i := 0; i < 2; i++ {
		rec := easRequest(e, testIMAPUser, testIMAPPass)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d", i, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), testIMAPUser) {
			t.Fatalf("request %d did not receive eas_user: %s", i, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"has_password":"yes"`) {
			t.Fatalf("request %d lost eas_password: %s", i, rec.Body.String())
		}
	}
}

// Ping reconnects the instant each long poll returns, and every Sync,
// GetItemEstimate and SendMail is another request. One IMAP login per request
// is what overloads the mail server.
func TestEASAuthCachesSuccessfulLogins(t *testing.T) {
	e, counter := newEASTestEnv(t)

	for i := 0; i < 20; i++ {
		if rec := easRequest(e, testIMAPUser, testIMAPPass); rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d", i, rec.Code)
		}
	}
	if got := counter.accepted.Load(); got != 1 {
		t.Fatalf("20 authenticated requests opened %d IMAP connections, want 1", got)
	}
}

func TestEASAuthDoesNotCacheFailures(t *testing.T) {
	e, counter := newEASTestEnv(t)

	for i := 0; i < 3; i++ {
		if rec := easRequest(e, testIMAPUser, "wrong"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("request %d: status = %d, want 401", i, rec.Code)
		}
	}
	if got := counter.accepted.Load(); got != 3 {
		t.Fatalf("failed logins opened %d connections, want 3 (failures must not be cached)", got)
	}
	// A revoked password must not keep working through a warm entry.
	if rec := easRequest(e, testIMAPUser, testIMAPPass); rec.Code != http.StatusOK {
		t.Fatalf("valid credentials after failures: status = %d", rec.Code)
	}
}

// The DAV and EAS middlewares must not share a cache instance: a hit on one
// must never authenticate a request to the other.
func TestEASAndCalDAVCachesAreIndependent(t *testing.T) {
	easEcho, easCounter := newEASTestEnv(t)
	davEcho, davCounter, _ := newAuthTestEnv(t)

	if rec := easRequest(easEcho, testIMAPUser, testIMAPPass); rec.Code != http.StatusOK {
		t.Fatalf("EAS status = %d", rec.Code)
	}
	if rec := request(davEcho, testIMAPUser, testIMAPPass); rec.Code != http.StatusOK {
		t.Fatalf("DAV status = %d", rec.Code)
	}
	if easCounter.accepted.Load() != 1 || davCounter.accepted.Load() != 1 {
		t.Fatalf("each middleware should have validated once: eas=%d dav=%d",
			easCounter.accepted.Load(), davCounter.accepted.Load())
	}
}
