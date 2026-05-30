// Package calendar provides iCalendar helpers for the calendar module.
package calendar

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go-cubemail/internal/model"
)

const defaultDomain = "go-cubemail"

// NewUID generates a unique VEVENT UID in the form "{hex16}@{domain}".
// Uses crypto/rand; domain defaults to "go-cubemail" when empty.
func NewUID(domain string) string {
	if domain == "" {
		domain = defaultDomain
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s@%s", hex.EncodeToString(b), domain)
}

// BuildICalContent generates a complete VCALENDAR document for a single VEVENT,
// including DTSTART/DTEND, organizer, attendees, RRULE, and STATUS fields.
func BuildICalContent(event *model.Event, attendees []model.EventAttendee) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//go-cubemail//Calendar//EN\r\n")
	b.WriteString("BEGIN:VEVENT\r\n")
	b.WriteString("UID:" + foldLine(event.UID) + "\r\n")
	b.WriteString("SUMMARY:" + foldLine(event.Summary) + "\r\n")
	if event.Description != "" {
		b.WriteString("DESCRIPTION:" + foldLine(event.Description) + "\r\n")
	}
	if event.Location != "" {
		b.WriteString("LOCATION:" + foldLine(event.Location) + "\r\n")
	}
	if event.IsAllDay {
		b.WriteString("DTSTART;VALUE=DATE:" + event.StartAt.UTC().Format("20060102") + "\r\n")
		b.WriteString("DTEND;VALUE=DATE:" + event.EndAt.UTC().Format("20060102") + "\r\n")
	} else {
		b.WriteString("DTSTART:" + event.StartAt.UTC().Format("20060102T150405Z") + "\r\n")
		b.WriteString("DTEND:" + event.EndAt.UTC().Format("20060102T150405Z") + "\r\n")
	}
	if event.OrganizerEmail != "" {
		org := "ORGANIZER"
		if event.OrganizerName != "" {
			org += ";CN=" + event.OrganizerName
		}
		b.WriteString(org + ":mailto:" + event.OrganizerEmail + "\r\n")
	}
	for _, a := range attendees {
		line := "ATTENDEE"
		if a.PartStat != "" {
			line += ";PARTSTAT=" + a.PartStat
		}
		if a.Role != "" {
			line += ";ROLE=" + a.Role
		}
		if a.RSVP {
			line += ";RSVP=TRUE"
		}
		if a.Name != "" {
			line += ";CN=" + a.Name
		}
		b.WriteString(line + ":mailto:" + a.Email + "\r\n")
	}
	if event.RRule != "" {
		b.WriteString("RRULE:" + event.RRule + "\r\n")
	}
	if event.Status != "" {
		b.WriteString("STATUS:" + event.Status + "\r\n")
	}
	b.WriteString("SEQUENCE:" + fmt.Sprintf("%d", event.Sequence) + "\r\n")
	b.WriteString("DTSTAMP:" + time.Now().UTC().Format("20060102T150405Z") + "\r\n")
	b.WriteString("END:VEVENT\r\n")
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}

// BuildCalendarExport merges multiple events into one VCALENDAR suitable for download.
func BuildCalendarExport(events []model.Event) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//go-cubemail//Calendar//EN\r\n")
	for _, event := range events {
		content := BuildICalContent(&event, event.Attendees)
		lines := strings.Split(content, "\r\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "BEGIN:VEVENT") ||
				strings.HasPrefix(line, "END:VEVENT") ||
				(!strings.HasPrefix(line, "BEGIN:VCALENDAR") &&
					!strings.HasPrefix(line, "END:VCALENDAR") &&
					!strings.HasPrefix(line, "VERSION:") &&
					!strings.HasPrefix(line, "PRODID:")) {
				if line != "" {
					b.WriteString(line + "\r\n")
				}
			}
		}
	}
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}

// ImportEvent holds parsed event data from an ICS file.
type ImportEvent struct {
	UID         string
	Summary     string
	Description string
	Location    string
	StartAt     time.Time
	EndAt       time.Time
	IsAllDay    bool
	Status      string
	RRule       string
	Attendees   []model.EventAttendee
}

// ParseICalImport parses one or more VEVENT blocks from raw ICS bytes.
// Returns an error when no valid events are found.
func ParseICalImport(data []byte) ([]ImportEvent, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")

	var unfolded []string
	for _, line := range lines {
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && len(unfolded) > 0 {
			unfolded[len(unfolded)-1] += line[1:]
		} else {
			unfolded = append(unfolded, line)
		}
	}

	var events []ImportEvent
	var cur *ImportEvent
	inEvent := false

	for _, line := range unfolded {
		line = strings.TrimRight(line, "\r")
		upper := strings.ToUpper(strings.TrimSpace(line))
		if upper == "BEGIN:VEVENT" {
			cur = &ImportEvent{Status: "CONFIRMED"}
			inEvent = true
			continue
		}
		if upper == "END:VEVENT" {
			if cur != nil && cur.Summary != "" && !cur.StartAt.IsZero() {
				if cur.EndAt.IsZero() {
					if cur.IsAllDay {
						cur.EndAt = cur.StartAt.Add(24 * time.Hour)
					} else {
						cur.EndAt = cur.StartAt.Add(time.Hour)
					}
				}
				if cur.UID == "" {
					cur.UID = NewUID("")
				}
				events = append(events, *cur)
			}
			cur = nil
			inEvent = false
			continue
		}
		if !inEvent || cur == nil {
			continue
		}

		propRaw, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		prop := strings.ToUpper(strings.SplitN(propRaw, ";", 2)[0])
		val = strings.TrimSpace(val)

		switch prop {
		case "UID":
			cur.UID = val
		case "SUMMARY":
			cur.Summary = val
		case "DESCRIPTION":
			cur.Description = val
		case "LOCATION":
			cur.Location = val
		case "DTSTART":
			t, allDay := parseICalTime(propRaw, val)
			cur.StartAt = t
			cur.IsAllDay = allDay
		case "DTEND":
			t, allDay := parseICalTime(propRaw, val)
			cur.EndAt = t
			if allDay {
				cur.IsAllDay = true
			}
		case "RRULE":
			cur.RRule = val
		case "STATUS":
			cur.Status = val
		case "ATTENDEE":
			email := strings.TrimPrefix(strings.ToLower(val), "mailto:")
			name := extractParam(propRaw, "CN")
			cur.Attendees = append(cur.Attendees, model.EventAttendee{
				Name:     name,
				Email:    email,
				PartStat: extractParam(propRaw, "PARTSTAT"),
				Role:     extractParam(propRaw, "ROLE"),
				RSVP:     strings.EqualFold(extractParam(propRaw, "RSVP"), "TRUE"),
			})
		}
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no events found in ICS file")
	}
	return events, nil
}

// parseICalTime parses a DTSTART or DTEND value from an ICS property line.
// Returns the parsed instant and whether the value is an all-day DATE.
func parseICalTime(propRaw, val string) (time.Time, bool) {
	val = strings.TrimRight(val, "Z")
	if strings.Contains(strings.ToUpper(propRaw), "VALUE=DATE") || len(val) == 8 {
		t, err := time.Parse("20060102", val)
		if err == nil {
			return t.UTC(), true
		}
	}
	for _, layout := range []string{"20060102T150405", "20060102T1504"} {
		if t, err := time.Parse(layout, val); err == nil {
			return t.UTC(), false
		}
	}
	return time.Time{}, false
}

// extractParam reads a semicolon parameter (e.g. CN, PARTSTAT) from an ICS property line.
func extractParam(propRaw, key string) string {
	parts := strings.Split(propRaw, ";")
	for _, p := range parts[1:] {
		k, v, ok := strings.Cut(p, "=")
		if ok && strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}

// FoldLine escapes newline characters for safe inclusion in ICS text properties.
func FoldLine(s string) string {
	return strings.ReplaceAll(s, "\n", "\\n")
}

// foldLine is the unexported alias kept for internal use.
func foldLine(s string) string { return FoldLine(s) }
