// Package middleware provides Echo middleware functions for go-cubemail:
// authentication, CSRF protection, rate limiting, and security headers.
package middleware

import (
	"net/http"

	"go-cubemail/internal/session"
	"github.com/labstack/echo/v5"
)

// RequireAuth blocks unauthenticated requests with a 401 JSON response.
// The Vue SPA intercepts 401 responses via axios and redirects to /app/login.
func RequireAuth(cookieName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			sessID, err := c.Cookie(cookieName)
			if err != nil || sessID.Value == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Not authenticated"})
			}
			s := session.Get(sessID.Value)
			if s == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Session expired"})
			}
			c.Set("imap_session", s)
			return next(c)
		}
	}
}
