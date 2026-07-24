package middleware

import (
	"net/http"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"github.com/labstack/echo/v5"
)

// EASAuth validates HTTP Basic Auth credentials against the configured IMAP server.
//
// Validations are cached for authCacheTTL under an HMAC of the credential pair,
// the same mechanism CalDAVAuth uses. ActiveSync needs it even more: Ping is a
// long poll, so a device reconnects the moment each Ping returns, and every
// Sync, GetItemEstimate, ItemOperations and SendMail is another request. Without
// the cache each one opens a TCP+TLS connection and a LOGIN, which overloads the
// mail server and can trip its brute-force protection against this host.
func EASAuth(cfg *config.Config) echo.MiddlewareFunc {
	timeout := time.Duration(cfg.IMAP.TimeoutSec) * time.Second
	cache := newCredentialCache(authCacheTTL)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			user, pass, ok := c.Request().BasicAuth()
			if !ok || user == "" {
				return easUnauthorized(c)
			}

			if _, hit := cache.get(user, pass); !hit {
				conn, err := imap.Connect(cfg.IMAP.Host, cfg.IMAP.Port, cfg.IMAP.TLS,
					timeout, user, pass, cfg.Server.Debug)
				if err != nil {
					// Never cache a failure: a password that stopped working must
					// not keep riding a warm entry.
					cache.invalidate(user, pass)
					return easUnauthorized(c)
				}
				conn.Close()
				// The EAS handlers resolve the user themselves, so there is no
				// database ID to memoise — only the fact that the pair is valid.
				cache.put(user, pass, 0)
			}

			c.Set("eas_user", user)
			c.Set("eas_password", pass)
			return next(c)
		}
	}
}

func easUnauthorized(c *echo.Context) error {
	c.Response().Header().Set("WWW-Authenticate", `Basic realm="go-cubemail"`)
	return c.NoContent(http.StatusUnauthorized)
}
