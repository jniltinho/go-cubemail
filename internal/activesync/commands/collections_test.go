package commands

// Tests for multi-collection support: every calendar and address book is a
// device folder, and a collection created after the device was set up still
// reaches it.

import (
	"testing"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openCollectionsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Contact{}, &model.AddressBook{},
		&model.Calendar{}, &model.Event{}, &model.EventAttendee{}, &model.DAVChange{},
		&state.EasDevice{}, &state.EasFolderState{}, &state.ImapFolderMapping{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func folderIDs(entries []FolderEntry) []string {
	out := make([]string, 0, len(entries))
	for _, f := range entries {
		out = append(out, f.ServerID)
	}
	return out
}

func hasID(entries []FolderEntry, id string) bool {
	for _, f := range entries {
		if f.ServerID == id {
			return true
		}
	}
	return false
}

// Every calendar and address book becomes a folder, and the default ones keep
// the historical "personal" ID so devices already synced are not disrupted.
func TestFolderBuilderListsEveryCollection(t *testing.T) {
	db := openCollectionsTestDB(t)
	calRepo := repository.NewCalendarRepo(db)
	bookRepo := repository.NewAddressBookRepo(db)
	builder := NewFolderBuilder(nil, nil, calRepo, bookRepo)
	ctx := &Context{UserID: 1}

	if _, err := calRepo.EnsureDefault(1); err != nil {
		t.Fatal(err)
	}
	work := model.Calendar{UserID: 1, Name: "Work Projects", Color: "#ff0000"}
	if err := calRepo.Create(&work); err != nil {
		t.Fatal(err)
	}
	team := model.AddressBook{UserID: 1, DisplayName: "Team"}
	if err := bookRepo.Create(&team); err != nil {
		t.Fatal(err)
	}

	cals := builder.calendarFolders(ctx)
	if !hasID(cals, "vevent/personal") {
		t.Fatalf("default calendar lost its historical ID: %v", folderIDs(cals))
	}
	if !hasID(cals, "vevent/"+work.URI) {
		t.Fatalf("second calendar is not exposed: %v", folderIDs(cals))
	}

	books := builder.contactFolders(ctx)
	if !hasID(books, "vcard/personal") {
		t.Fatalf("default address book lost its historical ID: %v", folderIDs(books))
	}
	if !hasID(books, "vcard/"+team.URI) {
		t.Fatalf("second address book is not exposed: %v", folderIDs(books))
	}

	// The display name follows the collection, so a rename reaches the device.
	for _, f := range cals {
		if f.ServerID == "vevent/"+work.URI && f.DisplayName != "Work Projects" {
			t.Fatalf("display name = %q, want the calendar name", f.DisplayName)
		}
		if f.Type != folderTypeCalendar {
			t.Fatalf("calendar folder type = %d", f.Type)
		}
	}
}

// Both the alias and the stored URI must resolve, and to the same collection.
func TestResolveCollections(t *testing.T) {
	db := openCollectionsTestDB(t)
	calRepo := repository.NewCalendarRepo(db)
	bookRepo := repository.NewAddressBookRepo(db)

	def, err := calRepo.EnsureDefault(1)
	if err != nil {
		t.Fatal(err)
	}
	work := model.Calendar{UserID: 1, Name: "Work"}
	if err := calRepo.Create(&work); err != nil {
		t.Fatal(err)
	}

	got, ok := resolveCalendarCollection(calRepo, 1, "vevent/personal")
	if !ok || got.ID != def.ID {
		t.Fatalf("the personal alias must resolve to the default calendar: %+v ok=%v", got, ok)
	}
	got, ok = resolveCalendarCollection(calRepo, 1, "vevent/"+work.URI)
	if !ok || got.ID != work.ID {
		t.Fatalf("named calendar did not resolve: %+v ok=%v", got, ok)
	}
	if _, ok := resolveCalendarCollection(calRepo, 1, "vevent/nope"); ok {
		t.Fatal("an unknown collection must not resolve")
	}
	// Another user's collection must be invisible.
	if _, ok := resolveCalendarCollection(calRepo, 2, "vevent/"+work.URI); ok {
		t.Fatal("a collection of another user resolved")
	}

	defBook, err := bookRepo.EnsureDefault(1)
	if err != nil {
		t.Fatal(err)
	}
	book, ok := resolveAddressBookCollection(bookRepo, 1, "vcard/personal")
	if !ok || book.ID != defBook.ID {
		t.Fatalf("the personal alias must resolve to the default address book: %+v ok=%v", book, ok)
	}
}

// A device that already synced once must still learn about new folders.
func TestFolderSyncReportsHierarchyChanges(t *testing.T) {
	db := openCollectionsTestDB(t)
	store := state.NewStore(db)
	handler := &FolderSyncHandler{store: store}

	dev, err := store.EnsureDevice(1, "device-1", "iPhone")
	if err != nil {
		t.Fatal(err)
	}

	initial := []FolderEntry{
		{ServerID: "vevent/personal", ParentID: "0", DisplayName: "Calendar", Type: folderTypeCalendar},
		{ServerID: "vcard/personal", ParentID: "0", DisplayName: "Contacts", Type: folderTypeContacts},
	}

	// First sync: everything is an Add.
	changes := handler.diffHierarchy(dev, initial, true)
	if len(changes.Add) != 2 || len(changes.Delete) != 0 {
		t.Fatalf("initial sync should add everything: %+v", changes)
	}
	if err := handler.saveHierarchy(dev, initial); err != nil {
		t.Fatal(err)
	}

	// Nothing changed: the device must not be disturbed.
	if changes := handler.diffHierarchy(dev, initial, false); len(changes.Add)+
		len(changes.Update)+len(changes.Delete) != 0 {
		t.Fatalf("a quiet hierarchy reported changes: %+v", changes)
	}

	// A calendar created later must arrive as an Add.
	grown := append(initial, FolderEntry{
		ServerID: "vevent/work", ParentID: "0", DisplayName: "Work", Type: folderTypeCalendar,
	})
	changes = handler.diffHierarchy(dev, grown, false)
	if len(changes.Add) != 1 || changes.Add[0].ServerID != "vevent/work" {
		t.Fatalf("a new collection did not reach the device: %+v", changes)
	}
	if err := handler.saveHierarchy(dev, grown); err != nil {
		t.Fatal(err)
	}

	// A rename is an Update, not a remove-and-add: the device keeps its state.
	renamed := make([]FolderEntry, len(grown))
	copy(renamed, grown)
	renamed[2].DisplayName = "Work Projects"
	changes = handler.diffHierarchy(dev, renamed, false)
	if len(changes.Update) != 1 || changes.Update[0].DisplayName != "Work Projects" {
		t.Fatalf("a rename should produce an Update: %+v", changes)
	}
	if len(changes.Add) != 0 || len(changes.Delete) != 0 {
		t.Fatalf("a rename must not add or delete folders: %+v", changes)
	}
	if err := handler.saveHierarchy(dev, renamed); err != nil {
		t.Fatal(err)
	}

	// And a removed collection becomes a Delete.
	changes = handler.diffHierarchy(dev, initial, false)
	if len(changes.Delete) != 1 || changes.Delete[0].ServerID != "vevent/work" {
		t.Fatalf("a removed collection should produce a Delete: %+v", changes)
	}
}

// ── Client Add handling ───────────────────────────────────────────────────

// syncAddForContact builds the Add command a device sends when it creates a contact.
func syncAddForContact(t *testing.T, clientID string, ct easContactPayload) eas.SyncAdd {
	t.Helper()
	body, err := marshalApplicationDataBody(&ct)
	if err != nil {
		t.Fatal(err)
	}
	return eas.SyncAdd{
		ClientID:        clientID,
		ApplicationData: &wbxml.RawElement{Page: wbxml.PageAirSync, Bytes: body},
	}
}

// A contact with no e-mail address is valid and phones create them routinely.
// Dropping it left the client's Add without a ServerId, and devices retry an
// unanswered Add indefinitely.
func TestClientAddWithoutEmailIsAccepted(t *testing.T) {
	db := openCollectionsTestDB(t)
	engine := NewContactsSyncEngine(
		state.NewStore(db),
		repository.NewContactRepo(db),
		repository.NewAddressBookRepo(db),
	)
	book, err := repository.NewAddressBookRepo(db).EnsureDefault(1)
	if err != nil {
		t.Fatal(err)
	}

	cache := state.ItemSyncCache{Items: map[string]state.ItemSyncItem{}}
	cmds := &eas.SyncCommands{Add: []eas.SyncAdd{
		syncAddForContact(t, "client-1", easContactPayload{
			FirstName: "Ana", LastName: "Ribeiro", MobilePhoneNumber: "+5511900000000",
		}),
	}}

	responses, err := engine.applyClientCommands(&Context{UserID: 1}, cmds, book.ID, &cache)
	if err != nil {
		t.Fatal(err)
	}
	if responses == nil || len(responses.Add) != 1 {
		t.Fatalf("every client Add needs a response with a ServerId, got %+v", responses)
	}
	if responses.Add[0].ClientID != "client-1" || responses.Add[0].ServerID == "" {
		t.Fatalf("incomplete response: %+v", responses.Add[0])
	}

	stored, err := repository.NewContactRepo(db).ListByBook(1, book.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].Phone != "+5511900000000" {
		t.Fatalf("the contact was not stored: %+v", stored)
	}
}

// An entirely empty payload carries nothing to store and is still skipped.
func TestClientAddWithNothingUsableIsSkipped(t *testing.T) {
	db := openCollectionsTestDB(t)
	engine := NewContactsSyncEngine(
		state.NewStore(db),
		repository.NewContactRepo(db),
		repository.NewAddressBookRepo(db),
	)
	book, err := repository.NewAddressBookRepo(db).EnsureDefault(1)
	if err != nil {
		t.Fatal(err)
	}

	cache := state.ItemSyncCache{Items: map[string]state.ItemSyncItem{}}
	cmds := &eas.SyncCommands{Add: []eas.SyncAdd{
		syncAddForContact(t, "client-empty", easContactPayload{}),
	}}

	if _, err := engine.applyClientCommands(&Context{UserID: 1}, cmds, book.ID, &cache); err != nil {
		t.Fatal(err)
	}
	stored, err := repository.NewContactRepo(db).ListByBook(1, book.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 0 {
		t.Fatalf("an empty payload should not create a contact: %+v", stored)
	}
}
