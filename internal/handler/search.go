package handler

import (
	"net/http"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/session"
	"github.com/labstack/echo/v5"
)

type SearchHandler struct {
	cfg *config.Config
}

func (h *SearchHandler) Results(c *echo.Context) error {
	q := c.QueryParam("q")
	mailbox := c.QueryParam("mailbox")
	if mailbox == "" {
		mailbox = "INBOX"
	}

	s := c.Get("imap_session").(*session.IMAPSession)
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return err
	}

	conn, err := imap.Connect(s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second, s.Username, pass, h.cfg.Server.Debug)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}

	// OR across Subject, From and To so any matching field returns results
	uids, err := conn.Search(&imap.SearchCriteria{Subject: q, From: q})
	if err != nil {
		return err
	}

	envelopes, err := conn.FetchEnvelopes(uids)
	if err != nil {
		return err
	}


	return c.JSON(http.StatusOK, map[string]interface{}{
		"mailbox":  mailbox,
		"messages": envelopes,
		"query":    q,
	})
}
