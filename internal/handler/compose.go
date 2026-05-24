// Package handler implements the HTTP request handlers for the go-cubemail API.
// Each handler type is responsible for a specific domain (auth, mailbox, messages, etc.)
// and is wired to routes in the server package.
package handler

import (
	"net/http"
	"strings"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/labstack/echo/v5"
	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/session"
	smtppkg "go-cubemail/internal/smtp"
)

// ComposeHandler handles email composition and sending.
type ComposeHandler struct {
	cfg *config.Config
}

// imapConn opens an authenticated IMAP connection using the current session credentials.
func (h *ComposeHandler) imapConn(s *session.IMAPSession) (*imap.Client, error) {
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return nil, err
	}
	return imap.Connect(s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second, s.Username, pass, h.cfg.Server.Debug)
}

// Send composes and delivers an email via SMTP, then appends a copy to the Sent folder.
// The Sent folder name is resolved from IMAP mailbox attributes to support servers with
// non-standard folder names (e.g. "[Gmail]/Sent Mail").
func (h *ComposeHandler) Send(c *echo.Context) error {
	s := c.Get("imap_session").(*session.IMAPSession)
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return err
	}

	msg := &smtppkg.Message{
		From:      s.Username,
		To:        splitAddrs(c.FormValue("to")),
		Cc:        splitAddrs(c.FormValue("cc")),
		Bcc:       splitAddrs(c.FormValue("bcc")),
		Subject:   c.FormValue("subject"),
		TextHTML:  c.FormValue("body_html"),
		TextPlain: c.FormValue("body_plain"),
	}

	form, err := c.MultipartForm()
	if err == nil && form != nil {
		for _, file := range form.File["attachments"] {
			src, err := file.Open()
			if err != nil {
				continue
			}
			data := make([]byte, file.Size)
			src.Read(data)
			src.Close()

			ct := file.Header.Get("Content-Type")
			if ct == "" {
				ct = "application/octet-stream"
			}
			msg.Attachments = append(msg.Attachments, smtppkg.Attachment{
				Filename:    file.Filename,
				ContentType: ct,
				Data:        data,
			})
		}
	}

	smtpCfg := smtppkg.Config{
		Host:       h.cfg.SMTP.Host,
		Port:       h.cfg.SMTP.Port,
		StartTLS:   h.cfg.SMTP.StartTLS,
		TimeoutSec: h.cfg.SMTP.TimeoutSec,
	}

	raw, err := smtppkg.Send(smtpCfg, s.Username, pass, msg)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}

	// Save a copy to the Sent folder; resolve the real folder name via IMAP attributes.
	if conn, imapErr := h.imapConn(s); imapErr == nil {
		defer conn.Close()
		sentFolder := "Sent"
		if boxes, lerr := conn.ListMailboxes(); lerr == nil {
			for _, mb := range boxes {
				if mb.IconType == "sent" {
					sentFolder = mb.Name
					break
				}
			}
		}
		_ = conn.AppendMessage(sentFolder, []goimap.Flag{goimap.FlagSeen}, raw)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "sent"})
}

// SaveDraft is a stub endpoint reserved for future draft persistence support.
func (h *ComposeHandler) SaveDraft(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// UploadAttachment is a stub endpoint reserved for future attachment upload support.
func (h *ComposeHandler) UploadAttachment(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{"files": []any{}})
}

// splitAddrs splits a comma-separated list of email addresses into a trimmed slice.
// Returns nil for an empty input.
func splitAddrs(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}
