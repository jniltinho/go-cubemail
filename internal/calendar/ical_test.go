package calendar

import (
	"strings"
	"testing"
	"time"

	"go-cubemail/internal/model"
)

// TestParseICalImport verifies that a minimal ICS VEVENT is parsed correctly.
func TestParseICalImport(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"BEGIN:VEVENT",
		"UID:test-uid@example.com",
		"SUMMARY:Team standup",
		"DTSTART:20260530T090000Z",
		"DTEND:20260530T093000Z",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n")

	events, err := ParseICalImport([]byte(ics))
	if err != nil {
		t.Fatalf("ParseICalImport() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Summary != "Team standup" {
		t.Fatalf("unexpected summary: %q", events[0].Summary)
	}
	if events[0].UID != "test-uid@example.com" {
		t.Fatalf("unexpected uid: %q", events[0].UID)
	}
}

// TestBuildICalContent verifies that a model event produces a valid VEVENT block.
func TestBuildICalContent(t *testing.T) {
	start := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 30, 9, 30, 0, 0, time.UTC)
	event := &model.Event{
		UID:     "abc@go-cubemail",
		Summary: "Demo",
		StartAt: start,
		EndAt:   end,
		Status:  "CONFIRMED",
	}

	content := BuildICalContent(event, nil)
	if !strings.Contains(content, "BEGIN:VEVENT") {
		t.Fatalf("missing VEVENT block")
	}
	if !strings.Contains(content, "SUMMARY:Demo") {
		t.Fatalf("missing summary in ICS output")
	}
}

// TestNewUID verifies that generated UIDs use the expected domain suffix.
func TestNewUID(t *testing.T) {
	uid := NewUID("example.com")
	if !strings.HasSuffix(uid, "@example.com") {
		t.Fatalf("unexpected uid suffix: %q", uid)
	}
}
