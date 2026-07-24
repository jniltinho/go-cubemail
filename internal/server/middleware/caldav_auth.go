package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/model"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// authCacheTTL is how long a successful IMAP validation is trusted.
//
// DAV clients poll every few minutes and each cycle issues several requests;
// without a cache every one of them opens a TCP+TLS connection and a LOGIN.
// That turns the IMAP server into a bottleneck and, on hosts running fail2ban,
// gets the webmail's own address banned. Five minutes keeps the load flat while
// bounding how long a revoked password stays usable over DAV.
const authCacheTTL = 5 * time.Minute

// authCacheSweep is how often expired entries are discarded.
const authCacheSweep = time.Minute

// credentialCache stores validated credentials keyed by an HMAC of
// user + password. The password itself is never held: only a keyed digest,
// with the key generated per process and kept in memory.
type credentialCache struct {
	mu     sync.RWMutex
	key    []byte
	items  map[string]cacheEntry
	ttl    time.Duration
	swept  time.Time
}

type cacheEntry struct {
	userID    uint
	expiresAt time.Time
}

func newCredentialCache(ttl time.Duration) *credentialCache {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		// A predictable key would only make the digests guessable, and the
		// cache never leaves this process; fall back rather than fail startup.
		key = []byte(time.Now().String())
	}
	return &credentialCache{
		key:   key,
		items: make(map[string]cacheEntry),
		ttl:   ttl,
		swept: time.Now(),
	}
}

// digest derives the cache key for a credential pair.
func (c *credentialCache) digest(user, pass string) string {
	mac := hmac.New(sha256.New, c.key)
	mac.Write([]byte(user))
	mac.Write([]byte{0})
	mac.Write([]byte(pass))
	return hex.EncodeToString(mac.Sum(nil))
}

// get returns the cached user ID for a credential pair, if still valid.
func (c *credentialCache) get(user, pass string) (uint, bool) {
	k := c.digest(user, pass)
	c.mu.RLock()
	entry, ok := c.items[k]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return 0, false
	}
	return entry.userID, true
}

// put records a successful validation.
func (c *credentialCache) put(user, pass string, userID uint) {
	k := c.digest(user, pass)
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[k] = cacheEntry{userID: userID, expiresAt: now.Add(c.ttl)}
	if now.Sub(c.swept) >= authCacheSweep {
		for key, e := range c.items {
			if now.After(e.expiresAt) {
				delete(c.items, key)
			}
		}
		c.swept = now
	}
}

// invalidate drops an entry after a failed authentication, so a password that
// stopped working cannot keep riding a warm cache entry.
func (c *credentialCache) invalidate(user, pass string) {
	k := c.digest(user, pass)
	c.mu.Lock()
	delete(c.items, k)
	c.mu.Unlock()
}

// CalDAVAuth validates HTTP Basic Auth for CalDAV/CardDAV clients and resolves
// the database user ID, setting "caldav_user_id" and "caldav_username" on the
// context. The model.User row is created on first access, matching the web
// app's RequireAuth behaviour.
func CalDAVAuth(cfg *config.Config, db *gorm.DB) echo.MiddlewareFunc {
	timeout := time.Duration(cfg.IMAP.TimeoutSec) * time.Second
	cache := newCredentialCache(authCacheTTL)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			user, pass, ok := c.Request().BasicAuth()
			if !ok || user == "" {
				return davUnauthorized(c)
			}

			if userID, hit := cache.get(user, pass); hit {
				c.Set("caldav_user_id", userID)
				c.Set("caldav_username", user)
				return next(c)
			}

			// Validate credentials against IMAP.
			conn, err := imap.Connect(cfg.IMAP.Host, cfg.IMAP.Port, cfg.IMAP.TLS,
				timeout, user, pass, cfg.Server.Debug)
			if err != nil {
				cache.invalidate(user, pass)
				return davUnauthorized(c)
			}
			conn.Close()

			// Resolve (or create) the user record.
			var u model.User
			if err := db.Where("imap_user = ?", user).
				FirstOrCreate(&u, model.User{ImapUser: user}).Error; err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}

			cache.put(user, pass, u.ID)
			c.Set("caldav_user_id", u.ID)
			c.Set("caldav_username", user)
			return next(c)
		}
	}
}

func davUnauthorized(c *echo.Context) error {
	c.Response().Header().Set("WWW-Authenticate", `Basic realm="go-cubemail DAV"`)
	return c.NoContent(http.StatusUnauthorized)
}
