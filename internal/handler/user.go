package handler

import (
	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/session"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// getUserID resolves the database user ID from the IMAP session username,
// creating a User record on first access if one does not yet exist.
func getUserID(c *echo.Context, db *gorm.DB) (uint, error) {
	s := c.Get("imap_session").(*session.IMAPSession)
	var user model.User
	err := db.Where("imap_user = ?", s.Username).
		FirstOrCreate(&user, model.User{ImapUser: s.Username}).Error
	return user.ID, err
}

// getSessionUsername returns the IMAP username from the authenticated session.
func getSessionUsername(c *echo.Context) string {
	s := c.Get("imap_session").(*session.IMAPSession)
	return s.Username
}

// getSessionPassword returns the decrypted IMAP/SMTP password from the session.
// Returns empty string on error (callers treat this as unauthenticated SMTP).
func getSessionPassword(c *echo.Context, cfg *config.Config) string {
	s := c.Get("imap_session").(*session.IMAPSession)
	pass, err := s.Password(cfg.Server.SecretKey)
	if err != nil {
		return ""
	}
	return pass
}
