package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
)

// CSRF implements the double-submit cookie pattern.
// GET/HEAD/OPTIONS: issues a csrf_token cookie if absent.
// POST/PUT/PATCH/DELETE: validates that X-CSRF-Token header matches the cookie.
func CSRF() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			path := c.Request().URL.Path
			if strings.HasPrefix(path, "/Microsoft-Server-ActiveSync") ||
				strings.HasPrefix(path, "/autodiscover") ||
				strings.HasPrefix(path, "/dav") ||
				strings.HasPrefix(path, "/.well-known/caldav") ||
				strings.HasPrefix(path, "/.well-known/carddav") {
				return next(c)
			}

			method := c.Request().Method

			switch method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				if _, err := c.Cookie(csrfCookieName); err != nil {
					token, err := newCSRFToken()
					if err != nil {
						return err
					}
					c.SetCookie(&http.Cookie{
						Name:     csrfCookieName,
						Value:    token,
						Path:     "/",
						SameSite: http.SameSiteLaxMode,
						HttpOnly: false, // JS must read it to send as request header
					})
				}
			default:
				cookie, err := c.Cookie(csrfCookieName)
				if err != nil || cookie.Value == "" {
					return c.JSON(http.StatusForbidden, map[string]string{"error": "CSRF token missing"})
				}
				header := c.Request().Header.Get(csrfHeaderName)
				if header == "" || header != cookie.Value {
					return c.JSON(http.StatusForbidden, map[string]string{"error": "CSRF token invalid"})
				}
			}
			return next(c)
		}
	}
}

// newCSRFToken generates a cryptographically random 16-byte token encoded as hex.
func newCSRFToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
