package handler

import (
	"net/http"
	"strings"
	"testing"

	"go-cubemail/internal/model"
)

// richVCard carries properties the flat Contact model cannot represent: a
// postal address, a birthday, a second phone number and an X-* extension.
// Any code path that regenerates the card from the indexed columns loses them,
// which is the single most reported CardDAV data-loss bug.
const richVCard = "BEGIN:VCARD\r\n" +
	"VERSION:3.0\r\n" +
	"UID:contact-001@test\r\n" +
	"FN:Ana Ribeiro\r\n" +
	"N:Ribeiro;Ana;;;\r\n" +
	"EMAIL;TYPE=INTERNET:ana@example.com\r\n" +
	"EMAIL;TYPE=WORK:ana.ribeiro@company.com\r\n" +
	"TEL;TYPE=CELL:+55 11 90000-0000\r\n" +
	"TEL;TYPE=WORK:+55 11 3000-0000\r\n" +
	"ADR;TYPE=HOME:;;Rua das Flores 100;Sao Paulo;SP;01000-000;Brazil\r\n" +
	"BDAY:19850312\r\n" +
	"ORG:Criarenet\r\n" +
	"TITLE:Engineer\r\n" +
	"X-CUSTOM-TAG:preserve-this\r\n" +
	"END:VCARD\r\n"

func contactsPath(resource string) string {
	p := "/dav/" + davTestUser + "/contacts/default/"
	if resource != "" {
		p += resource
	}
	return p
}

func TestCardDAVPropfindListsAddressBooks(t *testing.T) {
	env := newTestEnv(t)
	env.defaultAddressBook()

	doc := parseMultistatus(t, bodyBytes(
		env.propfind("/dav/"+davTestUser+"/contacts/", "1", "")))
	if len(doc.Responses) != 2 {
		t.Fatalf("expected home + 1 address book, got %d responses", len(doc.Responses))
	}
	book, ok := doc.find("/contacts/default/")
	if !ok {
		t.Fatal("default address book missing from the home set listing")
	}
	props := book.okProps()
	assertContains(t, props, "addressbook", "resourcetype")
	assertContains(t, props, "getctag", "ctag")
	assertContains(t, props, "sync-collection", "supported-report-set")
}

// RFC 6352 §6.2.3 names the child element address-data-type. With the wrong
// name a strict client ignores the property and may fall back to vCard 3.0 only.
func TestSupportedAddressDataUsesSpecElementName(t *testing.T) {
	env := newTestEnv(t)
	env.defaultAddressBook()

	body := `<?xml version="1.0" encoding="utf-8"?>
	<D:propfind xmlns:D="DAV:" xmlns:CR="urn:ietf:params:xml:ns:carddav">
	  <D:prop><CR:supported-address-data/></D:prop>
	</D:propfind>`

	doc := parseMultistatus(t, bodyBytes(env.propfind(contactsPath(""), "0", body)))
	props := doc.Responses[0].okProps()
	assertContains(t, props, "address-data-type", "supported-address-data")
	for _, version := range []string{`version="3.0"`, `version="4.0"`} {
		assertContains(t, props, version, "advertised vCard versions")
	}
}

// A payload above the limit must be refused. Truncating it would store a
// corrupted card and still answer 201 with a valid-looking ETag.
func TestOversizedVCardIsRejectedNotTruncated(t *testing.T) {
	env := newTestEnv(t)
	book := env.defaultAddressBook()

	huge := strings.Replace(richVCard, "X-CUSTOM-TAG:preserve-this",
		"NOTE:"+strings.Repeat("x", maxResourceSize+1024), 1)

	rec := env.do(http.MethodPut, contactsPath("huge.vcf"), huge,
		map[string]string{"Content-Type": "text/vcard"})
	assertStatus(t, rec, http.StatusRequestEntityTooLarge)
	assertContains(t, rec.Body.String(), "max-resource-size", "precondition element")

	if _, err := env.h.CardDAV.contactRepo.GetByResourceURI(env.userID, book.ID, "huge.vcf"); err == nil {
		t.Fatal("a rejected card must not be stored")
	}
}

// An oversized PUT must not clobber the resource that is already there.
func TestOversizedPutLeavesExistingResourceIntact(t *testing.T) {
	env := newTestEnv(t)
	env.defaultAddressBook()

	env.do(http.MethodPut, contactsPath("ana.vcf"), richVCard,
		map[string]string{"Content-Type": "text/vcard"})

	huge := strings.Replace(richVCard, "X-CUSTOM-TAG:preserve-this",
		"NOTE:"+strings.Repeat("x", maxResourceSize+1024), 1)
	assertStatus(t, env.do(http.MethodPut, contactsPath("ana.vcf"), huge,
		map[string]string{"Content-Type": "text/vcard"}), http.StatusRequestEntityTooLarge)

	get := env.do(http.MethodGet, contactsPath("ana.vcf"), "", nil)
	if get.Body.String() != richVCard {
		t.Fatal("the stored card was altered by a rejected oversized PUT")
	}
}

// The card must survive a full round trip untouched.
func TestCardDAVPutThenGetPreservesRawVCard(t *testing.T) {
	env := newTestEnv(t)
	env.defaultAddressBook()

	put := env.do(http.MethodPut, contactsPath("ana.vcf"), richVCard,
		map[string]string{"Content-Type": "text/vcard"})
	assertStatus(t, put, http.StatusCreated)

	get := env.do(http.MethodGet, contactsPath("ana.vcf"), "", nil)
	assertStatus(t, get, http.StatusOK)
	if got := get.Body.String(); got != richVCard {
		t.Fatalf("stored vCard was rewritten.\nwant:\n%s\ngot:\n%s", richVCard, got)
	}

	// The indexed columns must still be filled from the parsed card.
	ct, err := env.h.CardDAV.contactRepo.GetByResourceURI(
		env.userID, env.defaultAddressBook().ID, "ana.vcf")
	if err != nil {
		t.Fatalf("contact not indexed: %v", err)
	}
	if ct.FirstName != "Ana" || ct.LastName != "Ribeiro" {
		t.Fatalf("name index = %q %q", ct.FirstName, ct.LastName)
	}
	if ct.Email != "ana@example.com" {
		t.Fatalf("email index = %q, want the first EMAIL entry", ct.Email)
	}
	if ct.Company != "Criarenet" {
		t.Fatalf("org index = %q", ct.Company)
	}
}

// Editing a contact in the web UI must patch the stored card, not replace it.
func TestWebUIEditKeepsUnknownVCardProperties(t *testing.T) {
	env := newTestEnv(t)
	book := env.defaultAddressBook()

	env.do(http.MethodPut, contactsPath("ana.vcf"), richVCard,
		map[string]string{"Content-Type": "text/vcard"})

	ct, err := env.h.CardDAV.contactRepo.GetByResourceURI(env.userID, book.ID, "ana.vcf")
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the web UI changing the fields it knows about.
	ct.Title = "Principal Engineer"
	ct.Email = "ana.new@example.com"
	if err := env.h.CardDAV.contactRepo.Update(ct); err != nil {
		t.Fatal(err)
	}

	card := env.do(http.MethodGet, contactsPath("ana.vcf"), "", nil).Body.String()
	assertContains(t, card, "TITLE:Principal Engineer", "edited field")
	assertContains(t, card, "ana.new@example.com", "edited e-mail")
	// Everything the UI does not know about must still be there.
	for _, keep := range []string{"X-CUSTOM-TAG:preserve-this", "ADR;TYPE=HOME", "BDAY:19850312",
		"ana.ribeiro@company.com", "TEL;TYPE=WORK"} {
		assertContains(t, card, keep, "property outside the UI model")
	}
}

func TestCardDAVSyncCollectionReportsDeletions(t *testing.T) {
	env := newTestEnv(t)
	env.defaultAddressBook()

	const syncBody = `<?xml version="1.0" encoding="utf-8"?>
	<D:sync-collection xmlns:D="DAV:">
	  <D:sync-token>%s</D:sync-token>
	  <D:prop><D:getetag/></D:prop>
	</D:sync-collection>`

	env.do(http.MethodPut, contactsPath("a.vcf"), richVCard,
		map[string]string{"Content-Type": "text/vcard"})
	env.do(http.MethodPut, contactsPath("b.vcf"),
		strings.ReplaceAll(richVCard, "contact-001@test", "contact-002@test"),
		map[string]string{"Content-Type": "text/vcard"})

	initial := parseMultistatus(t, bodyBytes(
		env.report(contactsPath(""), strings.Replace(syncBody, "%s", "", 1))))
	if len(initial.Responses) != 2 {
		t.Fatalf("initial sync should list 2 cards, got %d", len(initial.Responses))
	}

	assertStatus(t, env.do(http.MethodDelete, contactsPath("a.vcf"), "", nil), http.StatusNoContent)

	delta := parseMultistatus(t, bodyBytes(
		env.report(contactsPath(""), strings.Replace(syncBody, "%s", initial.SyncToken, 1))))
	if len(delta.Responses) != 1 {
		t.Fatalf("delta should hold exactly the deletion, got %d", len(delta.Responses))
	}
	if !delta.Responses[0].isRemoved() {
		t.Fatalf("deletion should be reported with 404, got %q", delta.Responses[0].Status)
	}
	assertContains(t, delta.Responses[0].Href, "a.vcf", "tombstone href")
}

func TestAddressbookQueryFiltersByProperty(t *testing.T) {
	env := newTestEnv(t)
	env.defaultAddressBook()

	env.do(http.MethodPut, contactsPath("ana.vcf"), richVCard,
		map[string]string{"Content-Type": "text/vcard"})
	other := strings.NewReplacer(
		"contact-001@test", "contact-002@test",
		"Ana Ribeiro", "Bruno Costa",
		"Ribeiro;Ana", "Costa;Bruno",
		"ana@example.com", "bruno@example.com",
	).Replace(richVCard)
	env.do(http.MethodPut, contactsPath("bruno.vcf"), other,
		map[string]string{"Content-Type": "text/vcard"})

	body := `<?xml version="1.0" encoding="utf-8"?>
	<CR:addressbook-query xmlns:D="DAV:" xmlns:CR="urn:ietf:params:xml:ns:carddav">
	  <D:prop><D:getetag/><CR:address-data/></D:prop>
	  <CR:filter>
	    <CR:prop-filter name="FN">
	      <CR:text-match>Bruno</CR:text-match>
	    </CR:prop-filter>
	  </CR:filter>
	</CR:addressbook-query>`

	doc := parseMultistatus(t, bodyBytes(env.report(contactsPath(""), body)))
	if len(doc.Responses) != 1 {
		t.Fatalf("filter should match exactly 1 card, got %d", len(doc.Responses))
	}
	assertContains(t, doc.Responses[0].Href, "bruno.vcf", "matched href")
	assertContains(t, doc.Responses[0].okProps(), "Bruno Costa", "address-data")
}

func TestCardDAVMultigetAndConditionalWrites(t *testing.T) {
	env := newTestEnv(t)
	env.defaultAddressBook()

	created := env.do(http.MethodPut, contactsPath("ana.vcf"), richVCard,
		map[string]string{"Content-Type": "text/vcard", "If-None-Match": "*"})
	assertStatus(t, created, http.StatusCreated)
	etag := created.Header().Get("ETag")

	// Re-creating the same resource must be refused.
	assertStatus(t, env.do(http.MethodPut, contactsPath("ana.vcf"), richVCard,
		map[string]string{"Content-Type": "text/vcard", "If-None-Match": "*"}),
		http.StatusPreconditionFailed)

	// A stale If-Match must be refused.
	assertStatus(t, env.do(http.MethodPut, contactsPath("ana.vcf"), richVCard,
		map[string]string{"Content-Type": "text/vcard", "If-Match": `"outdated"`}),
		http.StatusPreconditionFailed)

	body := `<?xml version="1.0" encoding="utf-8"?>
	<CR:addressbook-multiget xmlns:D="DAV:" xmlns:CR="urn:ietf:params:xml:ns:carddav">
	  <D:prop><D:getetag/><CR:address-data/></D:prop>
	  <D:href>` + contactsPath("ana.vcf") + `</D:href>
	</CR:addressbook-multiget>`

	doc := parseMultistatus(t, bodyBytes(env.report(contactsPath(""), body)))
	if len(doc.Responses) != 1 {
		t.Fatalf("multiget returned %d responses", len(doc.Responses))
	}
	props := doc.Responses[0].okProps()
	assertContains(t, props, "X-CUSTOM-TAG", "address-data must carry the raw card")
	assertContains(t, props, strings.Trim(etag, `"`), "getetag")
}

// Contacts created through the web API must reach CardDAV clients.
func TestRestCreatedContactAppearsInAddressBook(t *testing.T) {
	env := newTestEnv(t)
	env.defaultAddressBook()

	ct := model.Contact{UserID: env.userID, FirstName: "Carla", LastName: "Dias",
		Email: "carla@example.com"}
	if err := env.h.CardDAV.contactRepo.Create(&ct); err != nil {
		t.Fatal(err)
	}
	if ct.ResourceURI == "" || ct.VCardContent == "" || ct.ETag == "" {
		t.Fatalf("repository must assign resource name, card and ETag: %+v", ct)
	}

	doc := parseMultistatus(t, bodyBytes(env.propfind(contactsPath(""), "1", "")))
	if _, ok := doc.find(ct.ResourceURI); !ok {
		t.Fatal("contact created through the repository is invisible to CardDAV clients")
	}
}

func TestMkcolCreatesAddressBook(t *testing.T) {
	env := newTestEnv(t)
	env.defaultAddressBook()

	body := `<?xml version="1.0" encoding="utf-8"?>
	<D:mkcol xmlns:D="DAV:" xmlns:CR="urn:ietf:params:xml:ns:carddav">
	  <D:set><D:prop>
	    <D:displayname>Team</D:displayname>
	    <CR:addressbook-description>Shared team book</CR:addressbook-description>
	  </D:prop></D:set>
	</D:mkcol>`

	assertStatus(t, env.do("MKCOL", "/dav/"+davTestUser+"/contacts/team/", body,
		map[string]string{"Content-Type": "application/xml"}), http.StatusCreated)

	book, err := env.h.CardDAV.bookRepo.GetByURI(env.userID, "team")
	if err != nil {
		t.Fatalf("address book was not created: %v", err)
	}
	if book.DisplayName != "Team" || book.Description != "Shared team book" {
		t.Fatalf("MKCOL properties ignored: %+v", book)
	}
}
