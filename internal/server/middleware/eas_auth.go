package middleware

import (
	"net/http"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"github.com/labstack/echo/v5"
)

// EASAuth validates HTTP Basic Auth credentials against the configured IMAP server.
func EASAuth(cfg *config.Config) echo.MiddlewareFunc {
	timeout := time.Duration(cfg.IMAP.TimeoutSec) * time.Second
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			user, pass, ok := c.Request().BasicAuth()
			if !ok {
				c.Response().Header().Set("WWW-Authenticate", `Basic realm="go-cubemail"`)
				return c.NoContent(http.StatusUnauthorized)
			}
			conn, err := imap.Connect(cfg.IMAP.Host, cfg.IMAP.Port, cfg.IMAP.TLS, timeout, user, pass, cfg.Server.Debug)
			if err != nil {
				c.Response().Header().Set("WWW-Authenticate", `Basic realm="go-cubemail"`)
				return c.NoContent(http.StatusUnauthorized)
			}
			conn.Close()
			c.Set("eas_user", user)
			c.Set("eas_password", pass)
			return next(c)
		}
	}
}
