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

func TestParseCollectionIDs(t *testing.T) {
	if !parseVEventCollectionID("vevent/personal") {
		t.Fatal("expected vevent collection")
	}
	if !parseVCardCollectionID("vcard/personal") {
		t.Fatal("expected vcard collection")
	}
}

func TestServerIDForUint(t *testing.T) {
	id, ok := parseServerID(serverIDForUint(99))
	if !ok || id != 99 {
		t.Fatalf("round trip failed: id=%d ok=%v", id, ok)
	}
}
