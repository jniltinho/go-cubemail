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

// List godoc
// @Summary      List emails in mailbox
// @Description  Returns a paginated list of email message envelopes for the specified folder, ordered newest first.
// @Tags         mailbox
// @Produce      json
// @Param        mailbox  path   string  true  "Mailbox folder name (e.g. INBOX, Drafts, Sent, Trash)"
// @Param        page     query  int     false "Page number (defaults to 1)"
// @Success      200  {object}  map[string]any "Success response containing mailbox, messages, page, and total count"
// @Failure      404  {object}  map[string]string "Mailbox not found"
// @Security     CookieAuth
// @Router       /mail/{mailbox} [get]
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

// FoldersJSON godoc
// @Summary      List IMAP folders
// @Description  Returns the complete hierarchical list of IMAP mailboxes/folders with unseen/total counts and tree structure metadata.
// @Tags         mailbox
// @Produce      json
// @Success      200  {array}   imap.MailboxInfo "Success response containing the list of folders"
// @Security     CookieAuth
// @Router       /folders [get]
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

// CreateSubfolder godoc
// @Summary      Create subfolder
// @Description  Creates a new IMAP mailbox folder, optionally nested under a parent folder.
// @Tags         mailbox
// @Accept       x-www-form-urlencoded
// @Produce      json
// @Param        name    formData  string  true   "Folder name"
// @Param        parent  formData  string  false  "Parent folder name (for nesting)"
// @Param        delim   formData  string  false  "Hierarchy delimiter (default: /)"
// @Success      200  {object}  map[string]string "status and full folder name"
// @Failure      400  {object}  map[string]string "name is required"
// @Failure      500  {object}  map[string]string "IMAP or connection error"
// @Security     CookieAuth
// @Router       /folders [post]
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

// RenameFolder godoc
// @Summary      Rename folder
// @Description  Renames an existing IMAP mailbox folder.
// @Tags         mailbox
// @Accept       x-www-form-urlencoded
// @Produce      json
// @Param        name     formData  string  true  "Current folder name"
// @Param        newname  formData  string  true  "New folder name"
// @Success      200  {object}  map[string]string "status ok"
// @Failure      400  {object}  map[string]string "name or newname missing"
// @Failure      500  {object}  map[string]string "IMAP error"
// @Security     CookieAuth
// @Router       /folders/rename [post]
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

// DeleteFolder godoc
// @Summary      Delete folder
// @Description  Recursively deletes an IMAP folder and all its subfolders.
// @Tags         mailbox
// @Accept       x-www-form-urlencoded
// @Produce      json
// @Param        name  formData  string  true  "Folder name to delete"
// @Success      200  {object}  map[string]string "status ok"
// @Failure      400  {object}  map[string]string "name missing"
// @Failure      500  {object}  map[string]string "IMAP error"
// @Security     CookieAuth
// @Router       /folders/delete [post]
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

// UnreadCountJSON godoc
// @Summary      Unread message count
// @Description  Returns the number of unseen (unread) messages in the specified folder.
// @Tags         mailbox
// @Produce      json
// @Param        name  path  string  true  "Folder name"
// @Success      200  {object}  map[string]uint32 "unseen count"
// @Security     CookieAuth
// @Router       /folders/{name}/count [get]
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
