package commands

import (
	"testing"

	goimap "github.com/emersion/go-imap/v2"
)

func TestParseMailCollectionID(t *testing.T) {
	guid, ok := parseMailCollectionID("mail/abc123")
	if !ok || guid != "abc123" {
		t.Fatalf("parse mail collection: ok=%v guid=%q", ok, guid)
	}
	if _, ok := parseMailCollectionID("vevent/personal"); ok {
		t.Fatal("expected false for calendar collection")
	}
}

func TestServerIDRoundTrip(t *testing.T) {
	id := serverIDForUID(goimap.UID(42))
	uid, ok := parseServerUID(id)
	if !ok || uid != 42 {
		t.Fatalf("round trip failed: id=%q uid=%v ok=%v", id, uid, ok)
	}
}

func TestEasDateFromEnvelope(t *testing.T) {
	got := easDateFromEnvelope("02/01/2006 15:04")
	if got == "" {
		t.Fatal("expected formatted date")
	}
}
