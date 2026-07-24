package commands

import (
	"strings"
	"testing"
	"time"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	calpkg "go-cubemail/internal/calendar"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openPhase4TestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Calendar{},
		&model.Event{},
		&model.EventAttendee{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSettingsHandlerDefaultUserInformation(t *testing.T) {
	h := NewSettingsHandler(nil)
	out, err := h.Handle(&Context{Username: "alice@example.com", DeviceID: "dev1", DeviceType: "iPhone"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var resp easSettingsResponse
	if err := wbxml.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != eas.StatusSuccess {
		t.Fatalf("status=%d", resp.Status)
	}
	if resp.Get == nil || resp.Get.UserInformation == nil {
		t.Fatal("expected UserInformation")
	}
	if resp.Get.UserInformation.Set.EmailAddresses != "alice@example.com" {
		t.Fatalf("email=%q", resp.Get.UserInformation.Set.EmailAddresses)
	}
	if resp.Get.UserInformation.Set.SmtpAddress != "alice@example.com" {
		t.Fatalf("smtp=%q", resp.Get.UserInformation.Set.SmtpAddress)
	}
}

func TestSettingsHandlerDeviceInformation(t *testing.T) {
	reqBody, err := wbxml.Marshal(easSettingsRequest{
		Get: &struct {
			UserInformation   *struct{} `wbxml:"Settings.UserInformation,omitempty"`
			DeviceInformation *struct{} `wbxml:"Settings.DeviceInformation,omitempty"`
			OOF               *struct{} `wbxml:"Settings.Oof,omitempty"`
		}{
			DeviceInformation: &struct{}{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	h := NewSettingsHandler(nil)
	out, err := h.Handle(&Context{
		Username:   "bob@example.com",
		DeviceID:     "device-abc",
		DeviceType:   "Android",
	}, reqBody)
	if err != nil {
		t.Fatal(err)
	}
	var resp easSettingsResponse
	if err := wbxml.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Get == nil || resp.Get.DeviceInformation == nil {
		t.Fatal("expected DeviceInformation")
	}
	set := resp.Get.DeviceInformation.Set
	if set.Model != "Android" || set.UserAgent != "Android" || set.FriendlyName != "device-abc" {
		t.Fatalf("unexpected device info: %+v", set)
	}
}

func TestMeetingResponseUpdatesAttendeePartStat(t *testing.T) {
	db := openPhase4TestDB(t)
	eventRepo := repository.NewEventRepo(db)

	userEmail := "attendee@example.com"
	start := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	event := &model.Event{
		UserID:     1,
		CalendarID: 1,
		UID:        "meeting-uid-001@go-cubemail",
		Summary:    "Team sync",
		StartAt:    start,
		EndAt:      end,
		Attendees: []model.EventAttendee{
			{Email: userEmail, Name: "Attendee", PartStat: "NEEDS-ACTION"},
		},
	}
	event.ICalContent = calpkg.BuildICalContent(event, event.Attendees)
	if err := eventRepo.Create(event); err != nil {
		t.Fatal(err)
	}

	reqBody, err := wbxml.Marshal(easMeetingResponseRequest{
		Request: struct {
			UserResponse int32  `wbxml:"MeetingResponse.UserResponse"`
			CollectionID string `wbxml:"MeetingResponse.CollectionId,omitempty"`
			CalendarID   string `wbxml:"MeetingResponse.CalendarId,omitempty"`
			RequestID    string `wbxml:"MeetingResponse.RequestId,omitempty"`
		}{
			UserResponse: 1,
			CalendarID:   serverIDForUint(event.ID),
			RequestID:    event.UID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	h := NewMeetingResponseHandler(nil, eventRepo)
	out, err := h.Handle(&Context{UserID: 1, Username: userEmail}, reqBody)
	if err != nil {
		t.Fatal(err)
	}
	var resp easMeetingResponseReply
	if err := wbxml.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != eas.StatusSuccess {
		t.Fatalf("status=%d", resp.Status)
	}
	if resp.Result == nil || resp.Result.Status != eas.StatusSuccess {
		t.Fatalf("unexpected result: %+v", resp.Result)
	}

	updated, err := eventRepo.Get(1, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Attendees) != 1 || updated.Attendees[0].PartStat != "ACCEPTED" {
		t.Fatalf("attendees=%+v", updated.Attendees)
	}
}

func TestParseSendMailBodyRawMIME(t *testing.T) {
	raw := []byte("From: alice@example.com\r\nTo: bob@example.com\r\nSubject: Hi\r\n\r\nBody")
	got, saveSent, err := parseSendMailBody(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !saveSent {
		t.Fatal("expected saveSent for raw MIME")
	}
	if string(got) != string(raw) {
		t.Fatalf("raw mismatch")
	}
}

func TestParseSendMailBodyWBXML(t *testing.T) {
	mime := []byte("From: alice@example.com\r\nTo: bob@example.com\r\n\r\nHello")
	body, err := wbxml.Marshal(easSendMailRequest{
		SaveInSentItems: 1,
		MIME:            mime,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, saveSent, err := parseSendMailBody(body)
	if err != nil {
		t.Fatal(err)
	}
	if !saveSent {
		t.Fatal("expected saveSent")
	}
	if string(got) != string(mime) {
		t.Fatalf("mime mismatch: %q", got)
	}
}

func TestParseSendMailBodyEmpty(t *testing.T) {
	_, _, err := parseSendMailBody(nil)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty error, got %v", err)
	}
}
