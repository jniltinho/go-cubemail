package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/session"
	"github.com/labstack/echo/v5"
)

type AuthHandler struct {
	cfg *config.Config
}

// DoLogin authenticates the user via IMAP and sets an encrypted session cookie.
// Returns JSON {"username": "..."} on success or {"error": "..."} on failure.
func (h *AuthHandler) DoLogin(c *echo.Context) error {
	imapHost := c.FormValue("imap_host")
	username := strings.TrimSpace(c.FormValue("username"))
	password := c.FormValue("password")

	// Input validation: reject obviously bad inputs early
	if username == "" || password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Username and password are required."})
	}
	if len(username) > 254 || len(password) > 512 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Input too long."})
	}

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
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials or server unreachable."})
	}
	conn.Close()

	sessID := newSessionID()
	s := &session.IMAPSession{
		IMAPHost: imapHost,
		IMAPPort: h.cfg.IMAP.Port,
		Username: username,
	}
	if err := s.SetPassword(password, h.cfg.Server.SecretKey); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Session error."})
	}
	session.Set(sessID, s)

	c.SetCookie(&http.Cookie{
		Name:     h.cfg.Session.Name,
		Value:    sessID,
		Path:     "/",
		MaxAge:   h.cfg.Session.MaxAge,
		HttpOnly: h.cfg.Session.HTTPOnly,
		Secure:   h.cfg.Session.Secure,
		SameSite: http.SameSiteStrictMode,
	})
	return c.JSON(http.StatusOK, map[string]string{"username": username})
}

// DoLogout invalidates the current session and clears the cookie.
func (h *AuthHandler) DoLogout(c *echo.Context) error {
	cookie, err := c.Cookie(h.cfg.Session.Name)
	if err == nil {
		session.Delete(cookie.Value)
	}
	c.SetCookie(&http.Cookie{
		Name:     h.cfg.Session.Name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteStrictMode,
	})
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// Me returns the current session username for the Vue router auth guard.
func (h *AuthHandler) Me(c *echo.Context) error {
	s := c.Get("imap_session").(*session.IMAPSession)
	return c.JSON(http.StatusOK, map[string]string{"username": s.Username})
}

// Quota returns IMAP storage quota in bytes: {"used": N, "limit": N}.
// Returns zeros without error when the server does not support QUOTA.
func (h *AuthHandler) Quota(c *echo.Context) error {
	s := c.Get("imap_session").(*session.IMAPSession)
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return err
	}
	conn, err := imap.Connect(s.IMAPHost, h.cfg.IMAP.Port, h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second, s.Username, pass, h.cfg.Server.Debug)
	if err != nil {
		return err
	}
	defer conn.Close()

	q, err := conn.GetQuota()
	if err != nil {
		return err
	}
	if q == nil {
		return c.JSON(http.StatusOK, map[string]int64{"used": 0, "limit": 0})
	}
	return c.JSON(http.StatusOK, map[string]int64{"used": q.UsageBytes, "limit": q.LimitBytes})
}

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
