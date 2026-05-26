package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	imappkg "go-cubemail/internal/imap"
	"go-cubemail/internal/session"

	"github.com/emersion/go-imap/v2"
	"github.com/labstack/echo/v5"
	"github.com/microcosm-cc/bluemonday"
)

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
	// Empty string signals the frontend to use its own empty/calendar state rendering.
	var safeHTML template.HTML
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
		"is_calendar_request": parsed != nil && parsed.CalendarInfo != nil,
		"calendar_info": func() *imappkg.CalendarInfo {
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
