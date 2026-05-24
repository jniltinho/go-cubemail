// Package smtp provides email composition and delivery via SMTP/STARTTLS.
// It handles inline image extraction (data: URIs → cid: references) and
// returns the raw RFC822 bytes so the caller can save a copy to the Sent folder.
package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"regexp"
	"strings"

	"go-cubemail/pkg/email"
)

// Message holds all fields needed to compose an outgoing email.
type Message struct {
	From        string
	DisplayName string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	TextPlain   string
	TextHTML    string
	Attachments []Attachment
}

// Attachment holds the binary content of a file to attach to an outgoing message.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Config holds SMTP server connection settings.
type Config struct {
	Host       string
	Port       int
	StartTLS   bool
	TimeoutSec int
}

// dataURIRe matches src="data:<mime>;base64,<data>" attributes inside HTML.
var dataURIRe = regexp.MustCompile(`src="data:([^;]+);base64,([^"]+)"`)

// extractInlineImages replaces data: URI image src attributes with cid: references,
// returning the modified HTML and the list of inline attachments to embed.
func extractInlineImages(html string) (string, []inlineImage) {
	var inlines []inlineImage
	idx := 0
	result := dataURIRe.ReplaceAllStringFunc(html, func(match string) string {
		parts := dataURIRe.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		mimeType := parts[1]
		b64data := parts[2]

		// Strip whitespace that TinyMCE may insert inside long base64 strings.
		b64data = strings.ReplaceAll(b64data, "\n", "")
		b64data = strings.ReplaceAll(b64data, " ", "")

		data, err := base64.StdEncoding.DecodeString(b64data)
		if err != nil {
			return match
		}

		idx++
		cid := fmt.Sprintf("inline-img-%d", idx)
		ext := "bin"
		switch mimeType {
		case "image/png":
			ext = "png"
		case "image/jpeg", "image/jpg":
			ext = "jpg"
		case "image/gif":
			ext = "gif"
		case "image/webp":
			ext = "webp"
		}
		filename := fmt.Sprintf("inline-%d.%s", idx, ext)

		inlines = append(inlines, inlineImage{cid: cid, filename: filename, mimeType: mimeType, data: data})
		return fmt.Sprintf(`src="cid:%s"`, cid)
	})
	return result, inlines
}

type inlineImage struct {
	cid      string
	filename string
	mimeType string
	data     []byte
}

// Send delivers msg via SMTP (or STARTTLS when cfg.StartTLS is true) and returns
// the raw RFC822 bytes so the caller can append a copy to the Sent folder.
func Send(cfg Config, user, pass string, msg *Message) ([]byte, error) {
	e := email.NewEmail()
	if msg.DisplayName != "" {
		e.From = fmt.Sprintf("%s <%s>", msg.DisplayName, msg.From)
	} else {
		e.From = msg.From
	}
	e.To = msg.To
	e.Cc = msg.Cc
	e.Bcc = msg.Bcc
	e.Subject = msg.Subject
	e.Text = []byte(msg.TextPlain)

	// Convert data: URI images to cid: inline attachments before sending.
	htmlBody := msg.TextHTML
	var inlines []inlineImage
	if htmlBody != "" {
		htmlBody, inlines = extractInlineImages(htmlBody)
		e.HTML = []byte(htmlBody)
	}

	for _, img := range inlines {
		part, err := e.Attach(bytes.NewReader(img.data), img.filename, img.mimeType)
		if err != nil {
			return nil, fmt.Errorf("attach inline %s: %w", img.filename, err)
		}
		part.Header.Set("Content-ID", fmt.Sprintf("<%s>", img.cid))
		part.Header.Set("Content-Disposition", "inline")
	}

	for _, a := range msg.Attachments {
		if _, err := e.Attach(bytes.NewReader(a.Data), a.Filename, a.ContentType); err != nil {
			return nil, fmt.Errorf("attach %s: %w", a.Filename, err)
		}
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", user, pass, cfg.Host)

	raw, err := e.Bytes()
	if err != nil {
		return nil, fmt.Errorf("build email: %w", err)
	}

	if cfg.StartTLS {
		if err := e.SendWithStartTLS(addr, auth, nil); err != nil {
			return nil, err
		}
		return raw, nil
	}
	if err := e.Send(addr, auth); err != nil {
		return nil, err
	}
	return raw, nil
}

