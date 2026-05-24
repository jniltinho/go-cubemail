package imap

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"strings"

	"github.com/emersion/go-message/mail"
)

// ParsedMessage holds the decoded body parts and attachments of a MIME message.
type ParsedMessage struct {
	TextPlain   string
	TextHTML    string
	Attachments []Attachment
}

// Attachment describes a single file attached to an email message.
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
	Part        int
	Data        []byte
}

// ParseMessage decodes a raw RFC822 message into its text bodies and attachments.
// Inline images referenced by cid: URIs are replaced with base64 data: URIs so
// they render correctly without a separate HTTP request.
func ParseMessage(raw []byte) (*ParsedMessage, error) {
	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	pm := &ParsedMessage{}
	part := 0
	// Map Content-ID → data URI for resolving cid: references in HTML.
	cidMap := make(map[string]string)

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		part++

		ctRaw := p.Header.Get("Content-Type")
		dispRaw := p.Header.Get("Content-Disposition")
		contentID := strings.Trim(p.Header.Get("Content-ID"), "<>")

		mediatype, _, _ := mime.ParseMediaType(ctRaw)
		dispType, dispParams, _ := mime.ParseMediaType(dispRaw)

		data, _ := io.ReadAll(p.Body)
		filename := dispParams["filename"]
		if filename == "" {
			_, ctParams, _ := mime.ParseMediaType(ctRaw)
			filename = ctParams["name"]
		}

		switch {
		case dispType == "attachment" || (filename != "" && dispType != "inline"):
			pm.Attachments = append(pm.Attachments, Attachment{
				Filename:    filename,
				ContentType: mediatype,
				Size:        int64(len(data)),
				Part:        part,
				Data:        data,
			})

		case contentID != "" && strings.HasPrefix(mediatype, "image/"):
			// Inline image referenced by cid: — convert to a data URI.
			b64 := base64.StdEncoding.EncodeToString(data)
			cidMap[contentID] = fmt.Sprintf("data:%s;base64,%s", mediatype, b64)

		case strings.HasPrefix(mediatype, "text/html"):
			pm.TextHTML = string(data)

		case strings.HasPrefix(mediatype, "text/plain"):
			pm.TextPlain = string(data)
		}
	}

	// Replace cid: references in the HTML body with the corresponding data URIs.
	if pm.TextHTML != "" && len(cidMap) > 0 {
		for cid, dataURI := range cidMap {
			pm.TextHTML = strings.ReplaceAll(pm.TextHTML, "cid:"+cid, dataURI)
		}
	}

	// Fallback for non-MIME messages: extract the raw body after the header separator.
	if pm.TextPlain == "" && pm.TextHTML == "" {
		if _, body, ok := bytes.Cut(raw, []byte("\r\n\r\n")); ok {
			pm.TextPlain = strings.TrimSpace(string(body))
		} else if _, body, ok := bytes.Cut(raw, []byte("\n\n")); ok {
			pm.TextPlain = strings.TrimSpace(string(body))
		}
	}

	return pm, nil
}
