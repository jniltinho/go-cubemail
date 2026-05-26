package handler

import (
	"net/http"
	"strconv"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/labstack/echo/v5"
	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/session"
)

// MailboxHandler handles IMAP folder and message-list operations.
type MailboxHandler struct {
	cfg *config.Config
}

// imapConn opens an authenticated IMAP connection using the current session credentials.
// NOTE: This helper is duplicated (with minor type variations) in message.go and compose.go.
// Kept local to each handler for package-level isolation and to avoid introducing
// cross-handler dependencies for this thin wrapper around session + imap.Connect.
func (h *MailboxHandler) imapConn(s *session.IMAPSession) (*imap.Client, error) {
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return nil, err
	}
	return imap.Connect(s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second, s.Username, pass, h.cfg.Server.Debug)
}

// resolveCreateDelimiter returns the hierarchy delimiter to use when creating a subfolder.
// It queries the parent folder's delimiter from the server; falls back to the client-supplied
// value or "/" when neither is available.
func (h *MailboxHandler) resolveCreateDelimiter(conn *imap.Client, parent, requested string) string {
	if parent == "" {
		if requested != "" {
			return requested
		}
		return "/"
	}

	folders, err := conn.ListMailboxes()
	if err == nil {
		for _, folder := range folders {
			if folder.Name == parent && folder.Delim != "" {
				return folder.Delim
			}
		}
		for _, folder := range folders {
			if folder.Delim != "" {
				return folder.Delim
			}
		}
	}

	if requested != "" {
		return requested
	}
	return "/"
}

// List returns a paginated list of message envelopes for the given mailbox, ordered newest first.
func (h *MailboxHandler) List(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	s := c.Get("imap_session").(*session.IMAPSession)

	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Mailbox not found"})
	}

	uids, err := conn.Search(&imap.SearchCriteria{})
	if err != nil {
		return err
	}

	// Reverse UIDs so newest messages appear first.
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

	// Rebuild in the original reversed order since FetchEnvelopes may reorder results.
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

	return c.JSON(http.StatusOK, map[string]any{
		"mailbox":  mailbox,
		"messages": envelopes,
		"page":     page,
		"total":    len(uids),
		"username": s.Username,
	})
}

// FoldersJSON returns the full list of IMAP folders with tree-structure metadata.
func (h *MailboxHandler) FoldersJSON(c *echo.Context) error {
	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
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

// CreateSubfolder creates a new IMAP folder, optionally nested under a parent folder.
func (h *MailboxHandler) CreateSubfolder(c *echo.Context) error {
	parent := c.FormValue("parent")
	name := c.FormValue("name")
	if name == "" {
		return echo.ErrBadRequest
	}
	requestedDelim := c.FormValue("delim")

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer conn.Close()

	delim := h.resolveCreateDelimiter(conn, parent, requestedDelim)
	fullName := name
	if parent != "" {
		fullName = parent + delim + name
	}

	if err := conn.CreateMailbox(fullName); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "name": fullName})
}

// RenameFolder renames an existing IMAP folder.
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

// DeleteFolder recursively deletes an IMAP folder and all its subfolders.
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
	if err := conn.DeleteMailboxRecursive(name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// UnreadCountJSON returns the number of unseen messages in the named folder.
func (h *MailboxHandler) UnreadCountJSON(c *echo.Context) error {
	name := c.Param("name")
	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
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
