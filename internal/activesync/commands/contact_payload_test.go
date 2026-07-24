package commands

// Tests for the contact payload mapping: notes, occupation and the semantics of
// a client Change.

import (
	"strings"
	"testing"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/model"
)

// Notes live in the AirSyncBase body, which the library's Contact type does not
// model — without the extended payload they never reach the device.
func TestContactNotesRoundTripThroughWBXML(t *testing.T) {
	source := model.Contact{
		FirstName: "Ana", LastName: "Ribeiro", Email: "ana@example.com",
		Notes: "Prefers e-mail.\nSpeaks Portuguese and English.",
	}

	payload := modelContactToEas(source)
	if payload.Body == nil {
		t.Fatal("notes were not attached to the payload")
	}
	if payload.Body.Type != bodyTypePlain {
		t.Fatalf("body type = %d, want plain text", payload.Body.Type)
	}

	// Encode and decode the way a real Sync response and request would.
	body, err := marshalApplicationDataBody(&payload)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := eas.UnmarshalApplicationData[easContactPayload](
		&wbxml.RawElement{Page: wbxml.PageAirSync, Bytes: body})
	if err != nil {
		t.Fatal(err)
	}

	if decoded.notes() != source.Notes {
		t.Fatalf("notes did not survive the round trip:\nwant %q\ngot  %q",
			source.Notes, decoded.notes())
	}
	if decoded.FirstName != "Ana" || decoded.Email1Address != "ana@example.com" {
		t.Fatalf("scalar fields were lost: %+v", decoded)
	}
}

// MS-ASCNTC splits what this application calls Title: JobTitle is the
// occupation, Title is the honorific.
func TestOccupationUsesJobTitle(t *testing.T) {
	payload := modelContactToEas(model.Contact{FirstName: "Ana", Title: "Engineer"})
	if payload.JobTitle != "Engineer" {
		t.Fatalf("occupation should be sent as JobTitle, got %+v", payload)
	}
	if payload.Title != "" {
		t.Fatalf("Title is the honorific and must stay empty, got %q", payload.Title)
	}

	// Reading accepts either, so contacts written by earlier builds — which put
	// the occupation in Title — are not lost.
	fromNew := easContactToModel(&easContactPayload{JobTitle: "Engineer"}, 1)
	fromOld := easContactToModel(&easContactPayload{Title: "Engineer"}, 1)
	if fromNew.Title != "Engineer" || fromOld.Title != "Engineer" {
		t.Fatalf("occupation not read back: new=%q old=%q", fromNew.Title, fromOld.Title)
	}
}

// A field cleared on the phone must stay cleared. Merging only non-empty values
// meant the old value came back on the next sync and could never be deleted.
func TestClearedFieldsAreCleared(t *testing.T) {
	stored := &model.Contact{
		FirstName: "Ana", LastName: "Ribeiro", Email: "ana@example.com",
		Company: "Criarenet", Title: "Engineer", Phone: "+5511900000000",
		Notes: "Some note",
	}

	// The device sends the complete item with company, title and notes removed.
	applyEasContactToModel(&easContactPayload{
		FirstName:         "Ana",
		LastName:          "Ribeiro",
		Email1Address:     "ana@example.com",
		MobilePhoneNumber: "+5511900000000",
	}, stored)

	for name, value := range map[string]string{
		"company": stored.Company, "title": stored.Title, "notes": stored.Notes,
	} {
		if value != "" {
			t.Errorf("%s was not cleared: %q", name, value)
		}
	}
	if stored.FirstName != "Ana" || stored.Phone != "+5511900000000" {
		t.Fatalf("fields the device kept were altered: %+v", stored)
	}
}

// Clearing a field through EAS must not damage the parts of the stored card the
// flat model cannot express.
func TestClearingAFieldKeepsUnrelatedVCardProperties(t *testing.T) {
	stored := &model.Contact{
		FirstName: "Ana", LastName: "Ribeiro", Email: "ana@example.com",
		Company: "Criarenet", VCardContent: richCard,
	}
	applyEasContactToModel(&easContactPayload{
		FirstName: "Ana", LastName: "Ribeiro", Email1Address: "ana@example.com",
	}, stored)

	if stored.Company != "" {
		t.Fatalf("company was not cleared: %q", stored.Company)
	}
	// The blob is only rewritten by the repository, and then as a patch.
	for _, keep := range []string{"ADR;TYPE=HOME", "PHOTO;ENCODING=b", "BDAY:19850312"} {
		if !strings.Contains(stored.VCardContent, keep) {
			t.Errorf("clearing a field touched %q", keep)
		}
	}
}

// The e-mail falls back through the alternate slots a device may use.
func TestEmailFallsBackAcrossSlots(t *testing.T) {
	tests := []struct {
		name    string
		payload easContactPayload
		want    string
	}{
		{name: "primary", payload: easContactPayload{Email1Address: "a@x.com"}, want: "a@x.com"},
		{name: "secondary", payload: easContactPayload{Email2Address: "b@x.com"}, want: "b@x.com"},
		{name: "tertiary", payload: easContactPayload{Email3Address: "c@x.com"}, want: "c@x.com"},
		{
			name: "primary wins",
			payload: easContactPayload{
				Email1Address: "a@x.com", Email2Address: "b@x.com",
			},
			want: "a@x.com",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := easContactToModel(&tc.payload, 1)
			if got.Email != tc.want {
				t.Fatalf("email = %q, want %q", got.Email, tc.want)
			}
		})
	}
}
