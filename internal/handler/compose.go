package handler

import (
	"net/http"
	"strings"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/session"
	smtppkg "go-cubemail/internal/smtp"
	"github.com/labstack/echo/v5"
	goimap "github.com/emersion/go-imap/v2"
)

type ComposeHandler struct {
	cfg *config.Config
}

func (h *ComposeHandler) New(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"mode": "new"})
}

func (h *ComposeHandler) Reply(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"mode":    "reply",
		"mailbox": c.Param("mailbox"),
		"uid":     c.Param("uid"),
	})
}

func (h *ComposeHandler) Forward(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"mode":    "forward",
		"mailbox": c.Param("mailbox"),
		"uid":     c.Param("uid"),
	})
}

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
		files := form.File["attachments"]
		for _, file := range files {
			src, err := file.Open()
			if err != nil {
				continue
			}
			data := make([]byte, file.Size)
			src.Read(data)
			src.Close()

			contentType := file.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "application/octet-stream"
			}

			msg.Attachments = append(msg.Attachments, smtppkg.Attachment{
				Filename:    file.Filename,
				ContentType: contentType,
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

	// Tenta salvar na pasta Sent
	conn, err := imap.Connect(s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second, s.Username, pass, h.cfg.Server.Debug)
	if err == nil {
		defer conn.Close()
		// Pode variar dependendo do servidor ("Sent", "Sent Items", "Enviados")
		// Usaremos "Sent" como padrão
		_ = conn.AppendMessage("Sent", []goimap.Flag{goimap.FlagSeen}, raw)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "sent"})
}

func (h *ComposeHandler) SaveDraft(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ComposeHandler) UploadAttachment(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"files": []interface{}{}})
}

func (h *ComposeHandler) ServeAttachment(c *echo.Context) error {
	return c.String(http.StatusNotImplemented, "TODO")
}

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
