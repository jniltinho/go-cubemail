package middleware

import (
	"net/http"

	"go-cubemail/internal/session"
	"github.com/labstack/echo/v5"
)

// RequireAuth bloqueia requests de usuários não autenticados.
func RequireAuth(cookieName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			sessID, err := c.Cookie(cookieName)
			if err != nil || sessID.Value == "" {
				return c.Redirect(http.StatusSeeOther, "/login")
			}
			s := session.Get(sessID.Value)
			if s == nil {
				return c.Redirect(http.StatusSeeOther, "/login")
			}
			c.Set("imap_session", s)
			return next(c)
		}
	}
}
