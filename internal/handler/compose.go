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
// NOTE: This helper is duplicated (with minor type variations) in mailbox.go and message.go.
// Kept local to each handler for package-level isolation and to avoid introducing
// cross-handler dependencies for this thin wrapper around session + imap.Connect.
func (h *ComposeHandler) imapConn(s *session.IMAPSession) (*imap.Client, error) {
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return nil, err
	}
	return imap.Connect(s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second, s.Username, pass, h.cfg.Server.Debug)
}

// Send godoc
// @Summary      Send email
// @Description  Composes and delivers an email via SMTP, then saves a copy to the Sent folder.
// @Tags         compose
// @Accept       multipart/form-data
// @Produce      json
// @Param        from_email    formData  string  false  "Sender email (defaults to session user)"
// @Param        from_name     formData  string  false  "Sender display name"
// @Param        to            formData  string  true   "Comma-separated recipient list"
// @Param        cc            formData  string  false  "Comma-separated CC list"
// @Param        bcc           formData  string  false  "Comma-separated BCC list"
// @Param        subject       formData  string  true   "Email subject"
// @Param        body_html     formData  string  false  "HTML body"
// @Param        body_plain    formData  string  false  "Plain-text body"
// @Param        reply_to      formData  string  false  "Reply-To address"
// @Param        attachments   formData  file    false  "File attachments"
// @Success      200  {object}  map[string]string "status sent"
// @Failure      502  {object}  map[string]string "SMTP delivery error"
// @Security     CookieAuth
// @Router       /compose/send [post]
// Send composes and delivers an email via SMTP, then appends a copy to the Sent folder.
// The Sent folder name is resolved from IMAP mailbox attributes to support servers with
// non-standard folder names (e.g. "[Gmail]/Sent Mail").
func (h *ComposeHandler) Send(c *echo.Context) error {
	s := c.Get("imap_session").(*session.IMAPSession)
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return err
	}

	fromEmail := c.FormValue("from_email")
	if fromEmail == "" {
		fromEmail = s.Username
	}
	msg := &smtppkg.Message{
		From:        fromEmail,
		DisplayName: c.FormValue("from_name"),
		To:          splitAddrs(c.FormValue("to")),
		Cc:          splitAddrs(c.FormValue("cc")),
		Bcc:         splitAddrs(c.FormValue("bcc")),
		Subject:     c.FormValue("subject"),
		TextHTML:    c.FormValue("body_html"),
		TextPlain:   c.FormValue("body_plain"),
	}
	if replyTo := strings.TrimSpace(c.FormValue("reply_to")); replyTo != "" {
		msg.ReplyTo = replyTo
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

// SaveDraft godoc
// @Summary      Save draft
// @Description  Saves an email draft (currently a stub; returns ok immediately).
// @Tags         compose
// @Produce      json
// @Success      200  {object}  map[string]string "status ok"
// @Security     CookieAuth
// @Router       /compose/draft [post]
func (h *ComposeHandler) SaveDraft(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// UploadAttachment godoc
// @Summary      Upload attachment
// @Description  Pre-uploads an attachment for use in a composed email (currently a stub).
// @Tags         compose
// @Accept       multipart/form-data
// @Produce      json
// @Success      200  {object}  map[string]any "files list (empty)"
// @Security     CookieAuth
// @Router       /compose/upload [post]
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
