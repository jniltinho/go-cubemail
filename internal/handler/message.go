package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-cubemail/internal/config"
	imappkg "go-cubemail/internal/imap"
	"go-cubemail/internal/session"

	"github.com/emersion/go-imap/v2"
	"github.com/labstack/echo/v5"
	"github.com/microcosm-cc/bluemonday"
)

// MessageHandler handles reading, flagging, moving, and deleting individual email messages.
type MessageHandler struct {
	cfg *config.Config
}

// messageDownloadName returns a filesystem-safe filename for downloading a message as .eml.
// Special characters that are invalid in filenames are replaced or removed.
func messageDownloadName(subject string, uid uint64) string {
	name := strings.TrimSpace(subject)
	if name == "" {
		return fmt.Sprintf("message-%d.eml", uid)
	}

	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "-",
	)
	name = replacer.Replace(name)
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return fmt.Sprintf("message-%d.eml", uid)
	}
	return name + ".eml"
}

// imapConn opens an authenticated IMAP connection using the current session credentials.
func (h *MessageHandler) imapConn(s *session.IMAPSession) (*imappkg.Client, error) {
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return nil, err
	}
	return imappkg.Connect(s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second, s.Username, pass, h.cfg.Server.Debug)
}

// findTrashFolder resolves the real Trash folder name from IMAP mailbox attributes.
// Returns "Trash" as a fallback when no folder is flagged with the \Trash attribute.
func (h *MessageHandler) findTrashFolder(conn *imappkg.Client) string {
	trashFolder := "Trash"
	if boxes, err := conn.ListMailboxes(); err == nil {
		for _, mb := range boxes {
			if mb.IsTrash {
				return mb.Name
			}
		}
	}
	return trashFolder
}

// Read fetches a message's envelope, sanitized HTML body, plain-text body, and attachment list.
// The message is marked as seen. The HTML body is sanitized via bluemonday before returning.
func (h *MessageHandler) Read(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		return echo.ErrBadRequest
	}

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}

	envelopes, err := conn.FetchEnvelopes([]imap.UID{imap.UID(uid)})
	if err != nil || len(envelopes) == 0 {
		return echo.ErrNotFound
	}

	conn.MarkSeen(imap.UID(uid))

	// Parse the raw message once and reuse the result for body, attachments, and plain text.
	var parsed *imappkg.ParsedMessage
	rawMsg, rawErr := conn.FetchRawMessage(imap.UID(uid))
	if rawErr == nil {
		parsed, _ = imappkg.ParseMessage(rawMsg)
	}

	// Build sanitized HTML body allowing common email HTML elements.
	safeHTML := template.HTML("<p>The message body is empty.</p>")
	if parsed != nil {
		bodyPolicy := bluemonday.NewPolicy()
		bodyPolicy.AllowElements(
			"html", "head", "body", "div", "span", "p", "br", "hr",
			"h1", "h2", "h3", "h4", "h5", "h6",
			"b", "strong", "i", "em", "u", "s", "strike", "sup", "sub",
			"ul", "ol", "li", "blockquote", "pre", "code",
			"table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption",
			"img", "a", "font",
		)
		bodyPolicy.AllowAttrs("style").Globally()
		bodyPolicy.AllowAttrs("class").Globally()
		bodyPolicy.AllowAttrs("align", "valign", "bgcolor", "color", "width", "height", "border", "cellpadding", "cellspacing").OnElements("table", "td", "th", "tr")
		bodyPolicy.AllowAttrs("src", "alt", "width", "height", "border").OnElements("img")
		bodyPolicy.AllowAttrs("href", "target").OnElements("a")
		bodyPolicy.AllowAttrs("face", "size", "color").OnElements("font")
		bodyPolicy.AllowAttrs("colspan", "rowspan").OnElements("td", "th")
		// Allow cid: and data: schemes for inline images embedded in the message.
		bodyPolicy.AllowURLSchemes("http", "https", "cid", "data", "mailto")

		if parsed.TextHTML != "" {
			safeHTML = template.HTML(bodyPolicy.Sanitize(parsed.TextHTML))
		} else if parsed.TextPlain != "" {
			safeHTML = template.HTML("<pre class='whitespace-pre-wrap font-sans text-sm'>" + bodyPolicy.Sanitize(parsed.TextPlain) + "</pre>")
		}
	}

	// Build attachment list with human-readable size labels.
	type attachmentView struct {
		Filename    string `json:"filename"`
		Part        int    `json:"part"`
		SizeLabel   string `json:"size_label"`
		ContentType string `json:"content_type"`
	}
	var attViews []attachmentView
	if parsed != nil {
		for _, a := range parsed.Attachments {
			var label string
			switch {
			case a.Size >= 1048576:
				label = fmt.Sprintf("%.1fMB", float64(a.Size)/1048576)
			case a.Size >= 1024:
				label = fmt.Sprintf("%.1fKB", float64(a.Size)/1024)
			default:
				label = fmt.Sprintf("%dB", a.Size)
			}
			attViews = append(attViews, attachmentView{
				Filename:    a.Filename,
				Part:        a.Part,
				SizeLabel:   label,
				ContentType: a.ContentType,
			})
		}
	}

	plainBody := ""
	if parsed != nil {
		plainBody = parsed.TextPlain
	}

	return c.JSON(http.StatusOK, map[string]any{
		"mailbox":             mailbox,
		"uid":                 uid,
		"envelope":            envelopes[0],
		"html_body":           string(safeHTML),
		"plain_body":          plainBody,
		"attachments":         attViews,
		"is_calendar_request": parsed != nil && parsed.CalendarInfo != nil, // ← novo
		"calendar_info":       func() *imappkg.CalendarInfo {                  // ← novo
			if parsed != nil {
				return parsed.CalendarInfo
			}
			return nil
		}(),
	})
}

// Download serves the raw RFC822 message as a downloadable .eml attachment.
func (h *MessageHandler) Download(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		return echo.ErrBadRequest
	}

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}

	rawMsg, err := conn.FetchRawMessage(imap.UID(uid))
	if err != nil || len(rawMsg) == 0 {
		return echo.ErrNotFound
	}

	filename := messageDownloadName("", uid)
	if envelopes, envErr := conn.FetchEnvelopes([]imap.UID{imap.UID(uid)}); envErr == nil && len(envelopes) > 0 {
		filename = messageDownloadName(envelopes[0].Subject, uid)
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	return c.Blob(http.StatusOK, "message/rfc822", rawMsg)
}

// Raw serves the raw RFC822 message bytes as plain text.
func (h *MessageHandler) Raw(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		return echo.ErrBadRequest
	}

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}

	rawMsg, err := conn.FetchRawMessage(imap.UID(uid))
	if err != nil || len(rawMsg) == 0 {
		return echo.ErrNotFound
	}

	return c.Blob(http.StatusOK, "text/plain; charset=utf-8", rawMsg)
}

// Flag sets or clears the "seen" or "flagged" IMAP flag on a message.
// The request must include form fields "flag" ("seen"|"flagged") and "value" ("1"|"0").
func (h *MessageHandler) Flag(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		return echo.ErrBadRequest
	}

	flag := c.FormValue("flag")
	value := c.FormValue("value") == "1"

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}

	imapUID := imap.UID(uid)
	switch flag {
	case "seen":
		if value {
			err = conn.MarkSeen(imapUID)
		} else {
			err = conn.MarkUnseen(imapUID)
		}
	case "flagged":
		err = conn.MarkFlagged(imapUID, value)
	}
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Move moves a message to the folder specified in the "dest" form field.
func (h *MessageHandler) Move(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		return echo.ErrBadRequest
	}
	dest := c.FormValue("dest")

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}
	if err := conn.MoveMessage(imap.UID(uid), dest); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Delete moves a message to the Trash folder.
// If the message is already in Trash, it is permanently deleted instead.
func (h *MessageHandler) Delete(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		return echo.ErrBadRequest
	}

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}

	trashFolder := h.findTrashFolder(conn)

	if mailbox == trashFolder {
		err = conn.DeleteMessage(imap.UID(uid))
	} else {
		err = conn.MoveMessage(imap.UID(uid), trashFolder)
	}
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// EmptyTrash permanently deletes all messages in the Trash folder,
// or moves all messages from the given mailbox to Trash if it is not the Trash folder.
func (h *MessageHandler) EmptyTrash(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	trashFolder := h.findTrashFolder(conn)

	if mailbox == trashFolder {
		if err := conn.EmptyMailbox(mailbox); err != nil {
			return err
		}
	} else {
		if err := conn.MoveAllMessages(mailbox, trashFolder); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Attachment downloads a single attachment identified by its MIME part number.
func (h *MessageHandler) Attachment(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		return echo.ErrBadRequest
	}
	part, err := strconv.Atoi(c.Param("part"))
	if err != nil {
		return echo.ErrBadRequest
	}

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}

	rawMsg, err := conn.FetchRawMessage(imap.UID(uid))
	if err != nil {
		return echo.ErrNotFound
	}

	parsed, err := imappkg.ParseMessage(rawMsg)
	if err != nil || parsed == nil {
		return echo.ErrNotFound
	}

	for _, att := range parsed.Attachments {
		if att.Part == part {
			ct := att.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			c.Response().Header().Set("Content-Disposition",
				"attachment; filename=\""+att.Filename+"\"")
			return c.Blob(http.StatusOK, ct, att.Data)
		}
	}
	return echo.ErrNotFound
}
