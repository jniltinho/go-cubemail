package imap

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"strings"
	"time"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset" // registers ISO-8859-*, Windows-125x, etc.
)

// ParsedMessage holds the decoded body parts and attachments of a MIME message.
type ParsedMessage struct {
	TextPlain    string
	TextHTML     string
	Attachments  []Attachment
	CalendarInfo *CalendarInfo
}

// CalendarInfo holds iCalendar event data (RFC 5545).
type CalendarInfo struct {
	Method    string   `json:"method"`    // REQUEST, CANCEL, REPLY
	Summary   string   `json:"summary"`
	Organizer string   `json:"organizer"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	Location  string   `json:"location"`
	UID       string   `json:"uid"`
	Attendees []string `json:"attendees"`
}

// Attachment describes a single file attached to an email message.
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
	Part        int
	Data        []byte
}

// formatICalDate converts an iCalendar DTSTART/DTEND value to a readable string.
// Handles: 20260525T150000, 20260525T150000Z, 20260525 (date only).
func formatICalDate(s string) string {
	s = strings.TrimRight(s, "Z")
	for _, layout := range []string{"20060102T150405", "20060102T1504", "20060102"} {
		if t, err := time.Parse(layout, s); err == nil {
			if layout == "20060102" {
				return t.Format("02/01/2006")
			}
			return t.Format("02/01/2006 15:04")
		}
	}
	return s
}

func parseICS(data []byte) *CalendarInfo {
	info := &CalendarInfo{}
	inVEvent := false
	for raw := range strings.SplitSeq(string(data), "\n") {
		line := strings.TrimRight(raw, "\r")
		upper := strings.ToUpper(strings.TrimSpace(line))
		if upper == "BEGIN:VEVENT" {
			inVEvent = true
			continue
		}
		if upper == "END:VEVENT" {
			inVEvent = false
			continue
		}

		keyRaw, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key := strings.ToUpper(strings.SplitN(keyRaw, ";", 2)[0])
		val = strings.TrimSpace(val)

		if !inVEvent {
			if key == "METHOD" {
				info.Method = strings.ToUpper(val)
			}
			continue
		}
		switch key {
		case "SUMMARY":
			info.Summary = val
		case "ORGANIZER":
			info.Organizer = strings.TrimPrefix(strings.ToLower(val), "mailto:")
		case "DTSTART":
			info.StartTime = formatICalDate(val)
		case "DTEND":
			info.EndTime = formatICalDate(val)
		case "LOCATION":
			info.Location = val
		case "UID":
			info.UID = val
		case "ATTENDEE":
			email := strings.TrimPrefix(strings.ToLower(val), "mailto:")
			if email != "" {
				info.Attendees = append(info.Attendees, email)
			}
		}
	}
	return info
}

// ParseMessage decodes a raw RFC822 message into its text bodies and attachments.
// Inline images referenced by cid: URIs are replaced with base64 data: URIs so
// they render correctly without a separate HTTP request.
func ParseMessage(raw []byte) (*ParsedMessage, error) {
	e, err := message.Read(bytes.NewReader(raw))
	if err != nil && !message.IsUnknownCharset(err) && !message.IsUnknownEncoding(err) {
		return rawFallback(raw), nil
	}
	if e == nil {
		return rawFallback(raw), nil
	}

	pm := &ParsedMessage{}
	cidMap := make(map[string]string)
	partNum := 0

	e.Walk(func(_ []int, entity *message.Entity, walkErr error) error {
		// Skip parts with unrecoverable errors; continue for unknown charset/encoding.
		if walkErr != nil && !message.IsUnknownCharset(walkErr) && !message.IsUnknownEncoding(walkErr) {
			return nil
		}
		if entity == nil {
			return nil
		}
		// Skip multipart container nodes — only process leaf parts.
		if entity.MultipartReader() != nil {
			return nil
		}

		partNum++
		processLeaf(entity, pm, partNum, cidMap)
		return nil
	})

	// Replace cid: references in the HTML body with the corresponding data URIs.
	if pm.TextHTML != "" && len(cidMap) > 0 {
		for cid, dataURI := range cidMap {
			pm.TextHTML = strings.ReplaceAll(pm.TextHTML, "cid:"+cid, dataURI)
		}
	}

	// Last-resort fallback when no text parts were extracted and no calendar data.
	// If CalendarInfo was parsed, the frontend can render the calendar bar even
	// without a text body.
	if pm.TextPlain == "" && pm.TextHTML == "" && pm.CalendarInfo == nil {
		return rawFallback(raw), nil
	}

	return pm, nil
}

// processLeaf extracts content from a single non-multipart MIME entity.
func processLeaf(e *message.Entity, pm *ParsedMessage, partNum int, cidMap map[string]string) {
	ctRaw := e.Header.Get("Content-Type")
	dispRaw := e.Header.Get("Content-Disposition")
	contentID := strings.Trim(e.Header.Get("Content-ID"), "<>")

	mediaType, ctParams, _ := mime.ParseMediaType(ctRaw)
	dispType, dispParams, _ := mime.ParseMediaType(dispRaw)

	// go-message decodes quoted-printable/base64 and handles charset conversion.
	data, err := io.ReadAll(e.Body)
	if err != nil {
		return
	}

	filename := dispParams["filename"]
	if filename == "" {
		filename = ctParams["name"]
	}

	switch {
	case dispType == "attachment" || (filename != "" && dispType != "inline"):
		pm.Attachments = append(pm.Attachments, Attachment{
			Filename:    filename,
			ContentType: mediaType,
			Size:        int64(len(data)),
			Part:        partNum,
			Data:        data,
		})

	case contentID != "" && strings.HasPrefix(mediaType, "image/"):
		b64 := base64.StdEncoding.EncodeToString(data)
		cidMap[contentID] = fmt.Sprintf("data:%s;base64,%s", mediaType, b64)

	case strings.HasPrefix(mediaType, "text/html"):
		if pm.TextHTML == "" {
			pm.TextHTML = string(data)
		}

	case strings.HasPrefix(mediaType, "text/plain"):
		if pm.TextPlain == "" {
			pm.TextPlain = strings.TrimSpace(string(data))
		}

	case strings.HasPrefix(mediaType, "text/calendar"),
		strings.HasSuffix(mediaType, "ics"),
		strings.EqualFold(mediaType, "application/ics"):
		pm.CalendarInfo = parseICS(data)
	}
}

// rawFallback extracts the body after the header block for non-MIME messages.
// This is a last resort — the result may include raw MIME if the message is
// malformed multipart.
func rawFallback(raw []byte) *ParsedMessage {
	pm := &ParsedMessage{}
	if _, body, ok := bytes.Cut(raw, []byte("\r\n\r\n")); ok {
		pm.TextPlain = strings.TrimSpace(string(body))
	} else if _, body, ok := bytes.Cut(raw, []byte("\n\n")); ok {
		pm.TextPlain = strings.TrimSpace(string(body))
	}
	return pm
}
