package commands

// Interaction between ActiveSync and the DAV layer.
//
// EAS and CardDAV write to the same rows through the same repositories. The EAS
// contact payload is far narrower than a vCard, so an edit made on a phone must
// not erase the properties only CardDAV knows about — and it must advance the
// DAV sync token so the change reaches CalDAV/CardDAV clients too.

import (
	"strings"
	"testing"

	"go-cubemail/internal/contacts"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// richCard carries properties the EAS Contact payload cannot express.
const richCard = "BEGIN:VCARD\r\n" +
	"VERSION:3.0\r\n" +
	"UID:contact-eas@test\r\n" +
	"FN:Ana Ribeiro\r\n" +
	"N:Ribeiro;Ana;;;\r\n" +
	"EMAIL;TYPE=INTERNET:ana@example.com\r\n" +
	"TEL;TYPE=CELL:+5511900000000\r\n" +
	"ADR;TYPE=HOME:;;Rua das Flores 100;Sao Paulo;SP;01000-000;Brazil\r\n" +
	"PHOTO;ENCODING=b;TYPE=JPEG:/9j/4AAQSkZJRg==\r\n" +
	"BDAY:19850312\r\n" +
	"X-CUSTOM:keep-me\r\n" +
	"END:VCARD\r\n"

func openDAVTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Contact{}, &model.AddressBook{}, &model.DAVChange{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

// A contact edited from a phone keeps everything the EAS payload cannot carry.
func TestEasContactEditPreservesVCardProperties(t *testing.T) {
	db := openDAVTestDB(t)
	repo := repository.NewContactRepo(db)

	parsed, err := contacts.Parse(richCard)
	if err != nil {
		t.Fatal(err)
	}
	parsed.UserID = 1
	parsed.VCardContent = richCard
	if err := repo.SaveRaw(parsed); err != nil {
		t.Fatal(err)
	}

	// The phone changes the two fields it knows about.
	stored, err := repo.Get(1, parsed.ID)
	if err != nil {
		t.Fatal(err)
	}
	// A device sends the complete item on Change, so the payload repeats the
	// fields it is not touching.
	applyEasContactToModel(&easContactPayload{
		FirstName:         "Ana",
		LastName:          "Ribeiro",
		Email1Address:     "ana@example.com",
		CompanyName:       "Criarenet",
		MobilePhoneNumber: "+5511911111111",
	}, stored)
	if err := repo.Update(stored); err != nil {
		t.Fatal(err)
	}

	after, err := repo.Get(1, parsed.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, keep := range []string{
		"ADR;TYPE=HOME", "PHOTO;ENCODING=b", "BDAY:19850312",
		"X-CUSTOM:keep-me", "UID:contact-eas@test",
	} {
		if !strings.Contains(after.VCardContent, keep) {
			t.Errorf("an ActiveSync edit destroyed %q:\n%s", keep, after.VCardContent)
		}
	}
	if !strings.Contains(after.VCardContent, "+5511911111111") {
		t.Errorf("the phone's edit was not applied:\n%s", after.VCardContent)
	}
	if after.Company != "Criarenet" {
		t.Errorf("company index = %q", after.Company)
	}
}

// A write from a phone must advance the collection revision, otherwise the
// change is invisible to CardDAV clients until something else touches the book.
func TestEasContactWriteAdvancesDAVSyncToken(t *testing.T) {
	db := openDAVTestDB(t)
	repo := repository.NewContactRepo(db)
	books := repository.NewAddressBookRepo(db)

	book, err := books.EnsureDefault(1)
	if err != nil {
		t.Fatal(err)
	}
	before := book.SyncToken

	contact := easContactToModel(&easContactPayload{
		FirstName: "Bruno", LastName: "Costa", Email1Address: "bruno@example.com",
	}, 1)
	if err := repo.Create(contact); err != nil {
		t.Fatal(err)
	}

	after, err := books.Get(1, book.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.SyncToken <= before {
		t.Fatalf("sync token did not advance: %d → %d", before, after.SyncToken)
	}
	if contact.ETag == "" || contact.ResourceURI == "" || contact.VCardContent == "" {
		t.Fatalf("EAS-created contact is missing DAV fields: %+v", contact)
	}

	// And the deletion must leave a tombstone.
	if err := repo.Delete(1, contact.ID); err != nil {
		t.Fatal(err)
	}
	var tombstones int64
	db.Model(&model.DAVChange{}).
		Where("collection_kind = ? AND uri = ? AND deleted = ?",
			model.CollectionAddressBook, contact.ResourceURI, true).
		Count(&tombstones)
	if tombstones != 1 {
		t.Fatalf("expected 1 tombstone for the EAS deletion, got %d", tombstones)
	}
}
