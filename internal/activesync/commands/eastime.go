package commands

import (
	"strconv"
	"strings"
	"time"
)

// easTime formats a time value for MS-ASCAL / MS-ASEMAIL wire fields.
//
// All-day events use YYYYMMDDTHHMMSSZ with time 000000; timed events use the full UTC timestamp.
func easTime(t time.Time, allDay bool) string {
	if allDay {
		return t.UTC().Format("20060102T000000Z")
	}
	return t.UTC().Format("20060102T150405Z")
}

// parseEasTime parses an EAS datetime string into UTC time.
// Supports 20060102T150405Z, 20060102T000000Z, and RFC3339 layouts.
func parseEasTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"20060102T150405Z",
		"20060102T000000Z",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// serverIDForUint converts a database primary key to an EAS ServerId string (decimal).
// Used for calendar events and contacts; mail uses IMAP UID via serverIDForUID.
func serverIDForUint(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

// parseServerID parses a decimal EAS ServerId back to a uint database primary key.
func parseServerID(serverID string) (uint, bool) {
	n, err := strconv.ParseUint(serverID, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(n), true
}
