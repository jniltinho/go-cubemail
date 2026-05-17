package handler

import (
	"net/http"
	"strconv"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/session"
	goimap "github.com/emersion/go-imap/v2"
	"github.com/labstack/echo/v5"
)

func (h *MailboxHandler) imapConn(s *session.IMAPSession) (*imap.Client, error) {
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return nil, err
	}
	return imap.Connect(s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second, s.Username, pass, h.cfg.Server.Debug)
}

type MailboxHandler struct {
	cfg *config.Config
}

func (h *MailboxHandler) List(c *echo.Context) error {
	mailbox := c.Param("mailbox")
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

	folders, err := conn.ListMailboxes()
	if err != nil {
		return err
	}

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}

	uids, err := conn.Search(&imap.SearchCriteria{})
	if err != nil {
		return err
	}

	// Inverter para mostrar os mais recentes primeiro
	for i, j := 0, len(uids)-1; i < j; i, j = i+1, j-1 {
		uids[i], uids[j] = uids[j], uids[i]
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage := h.cfg.UI.RowsPerPage
	start := (page - 1) * perPage
	end := start + perPage
	if start > len(uids) {
		start = len(uids)
	}
	if end > len(uids) {
		end = len(uids)
	}

	fetched, err := conn.FetchEnvelopes(uids[start:end])
	if err != nil {
		return err
	}

	// Reordenar para manter a ordem requisitada (mais recente primeiro)
	envMap := make(map[goimap.UID]imap.Envelope)
	for _, e := range fetched {
		envMap[e.UID] = e
	}
	envelopes := make([]imap.Envelope, 0, len(uids[start:end]))
	for _, uid := range uids[start:end] {
		if e, ok := envMap[uid]; ok {
			envelopes = append(envelopes, e)
		}
	}

	return c.Render(http.StatusOK, "mailbox/index.html", map[string]interface{}{
		"Mailbox":   mailbox,
		"Folders":   folders,
		"Messages":  envelopes,
		"Page":      page,
		"TotalMsgs": len(uids),
		"Username":  s.Username,
	})
}

func (h *MailboxHandler) FoldersJSON(c *echo.Context) error {
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

	folders, err := conn.ListMailboxes()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, folders)
}

func (h *MailboxHandler) CreateSubfolder(c *echo.Context) error {
	parent := c.FormValue("parent")
	name := c.FormValue("name")
	delim := c.FormValue("delim")
	if name == "" {
		return echo.ErrBadRequest
	}
	if delim == "" {
		delim = "/"
	}

	fullName := name
	if parent != "" {
		fullName = parent + delim + name
	}

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer conn.Close()

	if err := conn.CreateMailbox(fullName); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "name": fullName})
}

func (h *MailboxHandler) RenameFolder(c *echo.Context) error {
	name := c.FormValue("name")
	newname := c.FormValue("newname")
	if name == "" || newname == "" {
		return echo.ErrBadRequest
	}
	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer conn.Close()
	if err := conn.RenameMailbox(name, newname); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MailboxHandler) DeleteFolder(c *echo.Context) error {
	name := c.FormValue("name")
	if name == "" {
		return echo.ErrBadRequest
	}
	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer conn.Close()
	if err := conn.DeleteMailbox(name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MailboxHandler) UnreadCountJSON(c *echo.Context) error {
	name := c.Param("name")
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

	count, err := conn.UnreadCount(name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]uint32{"unseen": count})
}
