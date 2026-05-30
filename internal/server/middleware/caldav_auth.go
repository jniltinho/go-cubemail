package middleware

import (
	"net/http"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/model"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// CalDAVAuth validates HTTP Basic Auth for CalDAV clients and resolves the
// database user ID, setting "caldav_user_id" and "caldav_username" on the context.
// Creates the model.User row on first access (same as web app RequireAuth logic).
func CalDAVAuth(cfg *config.Config, db *gorm.DB) echo.MiddlewareFunc {
	timeout := time.Duration(cfg.IMAP.TimeoutSec) * time.Second
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			user, pass, ok := c.Request().BasicAuth()
			if !ok {
				c.Response().Header().Set("WWW-Authenticate", `Basic realm="go-cubemail CalDAV"`)
				return c.NoContent(http.StatusUnauthorized)
			}

			// Validate credentials against IMAP.
			conn, err := imap.Connect(cfg.IMAP.Host, cfg.IMAP.Port, cfg.IMAP.TLS,
				timeout, user, pass, cfg.Server.Debug)
			if err != nil {
				c.Response().Header().Set("WWW-Authenticate", `Basic realm="go-cubemail CalDAV"`)
				return c.NoContent(http.StatusUnauthorized)
			}
			conn.Close()

			// Resolve (or create) the user record.
			var u model.User
			if err := db.Where("imap_user = ?", user).
				FirstOrCreate(&u, model.User{ImapUser: user}).Error; err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}

			c.Set("caldav_user_id", u.ID)
			c.Set("caldav_username", user)
			return next(c)
		}
	}
}
