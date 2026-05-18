package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/labstack/echo/v5"
	"github.com/microcosm-cc/bluemonday"
	"go-cubemail/internal/config"
	imappkg "go-cubemail/internal/imap"
	"go-cubemail/internal/session"
)

type MessageHandler struct {
	cfg *config.Config
}

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

func (h *MessageHandler) imapConn(s *session.IMAPSession) (*imappkg.Client, error) {
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return nil, err
	}
	return imappkg.Connect(s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second, s.Username, pass, h.cfg.Server.Debug)
}

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

	policy := bluemonday.UGCPolicy()
	safeSubject := policy.Sanitize(envelopes[0].Subject)

	var safeHTML template.HTML = template.HTML("<p class='text-gray-500 italic'>O corpo da mensagem está vazio.</p>")
	rawMsg, err := conn.FetchRawMessage(imap.UID(uid))
	if err == nil {
		parsedMsg, parseErr := imappkg.ParseMessage(rawMsg)
		if parseErr == nil {
			bodyPolicy := bluemonday.NewPolicy()
			// Permitir todos os elementos comuns de e-mail HTML
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
			// Permitir src com cid: (imagens inline) e https/http
			bodyPolicy.AllowURLSchemes("http", "https", "cid", "data", "mailto")

			if parsedMsg.TextHTML != "" {
				safeHTML = template.HTML(bodyPolicy.Sanitize(parsedMsg.TextHTML))
			} else if parsedMsg.TextPlain != "" {
				safeHTML = template.HTML("<pre class='whitespace-pre-wrap font-sans text-sm'>" + bodyPolicy.Sanitize(parsedMsg.TextPlain) + "</pre>")
			}
		}
	}

	// Monta lista de anexos com tamanho pré-formatado
	type attachmentView struct {
		Filename  string
		Part      int
		SizeLabel string
	}
	var attViews []attachmentView
	if rawMsg != nil {
		if p, e := imappkg.ParseMessage(rawMsg); e == nil {
			for _, a := range p.Attachments {
				label := ""
				switch {
				case a.Size >= 1048576:
					label = fmt.Sprintf("%.1fMB", float64(a.Size)/1048576)
				case a.Size >= 1024:
					label = fmt.Sprintf("%.1fKB", float64(a.Size)/1024)
				default:
					label = fmt.Sprintf("%dB", a.Size)
				}
				attViews = append(attViews, attachmentView{
					Filename:  a.Filename,
					Part:      a.Part,
					SizeLabel: label,
				})
			}
		}
	}

	return c.Render(http.StatusOK, "mailbox/message.html", map[string]interface{}{
		"Mailbox":     mailbox,
		"UID":         uid,
		"Envelope":    envelopes[0],
		"SafeSubject": safeSubject,
		"SafeHTML":    safeHTML,
		"Attachments": attViews,
	})
}

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

	if mailbox == "Trash" {
		err = conn.DeleteMessage(imap.UID(uid))
	} else {
		err = conn.MoveMessage(imap.UID(uid), "Trash")
	}
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MessageHandler) EmptyTrash(c *echo.Context) error {
	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.EmptyMailbox(c.Param("mailbox")); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

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
