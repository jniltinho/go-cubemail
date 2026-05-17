package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/session"
	"github.com/labstack/echo/v5"
)

type AuthHandler struct {
	cfg *config.Config
}

func (h *AuthHandler) LoginPage(c *echo.Context) error {
	return c.Render(http.StatusOK, "auth/login.html", map[string]interface{}{
		"DefaultHost":   h.cfg.IMAP.Host,
		"ShowHostInput": h.cfg.IMAP.ShowHostInput,
	})
}

func (h *AuthHandler) DoLogin(c *echo.Context) error {
	imapHost := c.FormValue("imap_host")
	username := c.FormValue("username")
	password := c.FormValue("password")

	if imapHost == "" {
		imapHost = h.cfg.IMAP.Host
	}

	conn, err := imap.Connect(
		imapHost,
		h.cfg.IMAP.Port,
		h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second,
		username,
		password,
		h.cfg.Server.Debug,
	)
	if err != nil {
		c.Logger().Error("IMAP Login failed", "user", username, "host", imapHost, "port", h.cfg.IMAP.Port, "error", err)
		return c.Render(http.StatusOK, "auth/login.html", map[string]interface{}{
			"DefaultHost":   imapHost,
			"ShowHostInput": h.cfg.IMAP.ShowHostInput,
			"Error":         "Invalid credentials or server unreachable.",
		})
	}
	conn.Close()

	sessID := newSessionID()
	s := &session.IMAPSession{
		IMAPHost: imapHost,
		IMAPPort: h.cfg.IMAP.Port,
		Username: username,
	}
	if err := s.SetPassword(password, h.cfg.Server.SecretKey); err != nil {
		return err
	}
	session.Set(sessID, s)

	c.SetCookie(&http.Cookie{
		Name:     h.cfg.Session.Name,
		Value:    sessID,
		Path:     "/",
		MaxAge:   h.cfg.Session.MaxAge,
		HttpOnly: h.cfg.Session.HTTPOnly,
		Secure:   h.cfg.Session.Secure,
	})
	return c.Redirect(http.StatusSeeOther, "/mail/INBOX")
}

func (h *AuthHandler) DoLogout(c *echo.Context) error {
	cookie, err := c.Cookie(h.cfg.Session.Name)
	if err == nil {
		session.Delete(cookie.Value)
	}
	c.SetCookie(&http.Cookie{
		Name:   h.cfg.Session.Name,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	return c.Redirect(http.StatusSeeOther, "/login")
}

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
