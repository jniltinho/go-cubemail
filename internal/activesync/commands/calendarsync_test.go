package commands

import (
	"testing"
	"time"

	"go-cubemail/internal/model"
)

func TestEasTimeRoundTrip(t *testing.T) {
	start := time.Date(2026, 5, 30, 14, 30, 0, 0, time.UTC)
	formatted := easTime(start, false)
	parsed, ok := parseEasTime(formatted)
	if !ok || !parsed.Equal(start) {
		t.Fatalf("round trip failed: %q -> %v ok=%v", formatted, parsed, ok)
	}
}

func TestEventToAppointment(t *testing.T) {
	event := &model.Event{
		ID:       7,
		UID:      "abc@go-cubemail",
		Summary:  "Meeting",
		Location: "Room A",
		StartAt:  time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		EndAt:    time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC),
		UpdatedAt: time.Now().UTC(),
		Attendees: []model.EventAttendee{
			{Email: "bob@example.com", Name: "Bob", PartStat: "ACCEPTED"},
		},
	}
	appt := eventToAppointment(event)
	if appt.Subject != "Meeting" || appt.UID != "abc@go-cubemail" {
		t.Fatalf("unexpected appointment: %+v", appt)
	}
	if appt.Attendees == nil || len(appt.Attendees.Attendee) != 1 {
		t.Fatal("expected attendees")
	}
}

func TestModelContactToEas(t *testing.T) {
	ct := modelContactToEas(model.Contact{
		FirstName: "Alice",
		LastName:  "Smith",
		Email:     "alice@example.com",
		Phone:     "+5511999999999",
	})
	if ct.Email1Address != "alice@example.com" || ct.FirstName != "Alice" {
		t.Fatalf("unexpected contact: %+v", ct)
	}
}

func TestCollectionURI(t *testing.T) {
	tests := []struct {
		name         string
		collectionID string
		prefix       string
		wantURI      string
		wantOK       bool
	}{
		{name: "default calendar", collectionID: "vevent/personal", prefix: prefixCalendar, wantURI: "personal", wantOK: true},
		{name: "named calendar", collectionID: "vevent/work", prefix: prefixCalendar, wantURI: "work", wantOK: true},
		{name: "default contacts", collectionID: "vcard/personal", prefix: prefixContacts, wantURI: "personal", wantOK: true},
		{name: "wrong class", collectionID: "vcard/personal", prefix: prefixCalendar, wantOK: false},
		{name: "missing collection", collectionID: "vevent/", prefix: prefixCalendar, wantOK: false},
		// A nested segment could otherwise be used to address another user.
		{name: "path traversal", collectionID: "vevent/../other", prefix: prefixCalendar, wantOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uri, ok := collectionURI(tc.collectionID, tc.prefix)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && uri != tc.wantURI {
				t.Fatalf("uri = %q, want %q", uri, tc.wantURI)
			}
		})
	}
}

func TestServerIDForUint(t *testing.T) {
	id, ok := parseServerID(serverIDForUint(99))
	if !ok || id != 99 {
		t.Fatalf("round trip failed: id=%d ok=%v", id, ok)
	}
}
