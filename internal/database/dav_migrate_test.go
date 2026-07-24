package database

// Migration tests for the DAV schema.
//
// The interesting case is not a fresh install — it is an existing database with
// rows that predate the DAV columns. Those rows have no resource name, so the
// new unique indexes cannot be built until they are backfilled, and the old
// UNIQUE index on events.uid has to go before two users can hold the same
// iCalendar UID.

import (
	"testing"
	"time"

	"go-cubemail/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// The legacy* structs are the models as they were before the DAV work: no
// resource names, no ETags, no address books, and a globally UNIQUE event UID.
// Migrating a database that GORM itself created from these is the real upgrade
// path, so the test builds the old schema the same way production did.

type legacyCalendar struct {
	ID                uint   `gorm:"primaryKey"`
	UserID            uint   `gorm:"index;not null"`
	Name              string `gorm:"size:255;not null"`
	Color             string `gorm:"size:7;default:'#3788d8'"`
	IsDefault         bool   `gorm:"default:false"`
	IsActive          bool   `gorm:"default:true"`
	IncludeInFreeBusy bool   `gorm:"default:true"`
	SortOrder         int    `gorm:"default:0"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (legacyCalendar) TableName() string { return "calendars" }

type legacyEvent struct {
	ID          uint      `gorm:"primaryKey"`
	CalendarID  uint      `gorm:"index;not null"`
	UserID      uint      `gorm:"index;not null"`
	UID         string    `gorm:"size:255;uniqueIndex;not null"` // the constraint to drop
	Summary     string    `gorm:"size:1000;not null"`
	StartAt     time.Time `gorm:"index;not null"`
	EndAt       time.Time `gorm:"index;not null"`
	RRule       string    `gorm:"type:text"`
	ICalContent string    `gorm:"type:text"`
	IsTask      bool      `gorm:"default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (legacyEvent) TableName() string { return "events" }

type legacyContact struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"index;not null"`
	UID       string `gorm:"size:255;index"`
	FirstName string
	LastName  string
	Email     string `gorm:"not null"`
	Title     string
	Phone     string
	Company   string
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (legacyContact) TableName() string { return "contacts" }

// openLegacyDB builds a database in the pre-DAV shape with a few rows in it.
func openLegacyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &legacyCalendar{}, &legacyEvent{}, &legacyContact{}); err != nil {
		t.Fatalf("legacy schema: %v", err)
	}

	seed := []string{
		`INSERT INTO users (id, imap_user) VALUES (1, 'ana@example.com'), (2, 'bruno@example.com')`,
		`INSERT INTO calendars (id, user_id, name, is_default) VALUES
			(1, 1, 'Personal', true), (2, 1, 'Work Projects', false), (3, 2, 'Personal', true)`,
		// Two events in the same calendar: both start with an empty resource
		// name, which is exactly what breaks a naive unique-index migration.
		`INSERT INTO events (calendar_id, user_id, uid, summary, start_at, end_at, i_cal_content) VALUES
			(1, 1, 'ev-1@test', 'One', '2026-07-01 10:00:00', '2026-07-01 11:00:00',
			 'BEGIN:VCALENDAR' || char(13) || char(10) || 'END:VCALENDAR'),
			(1, 1, 'ev-2@test', 'Two', '2026-07-02 10:00:00', '2026-07-02 11:00:00',
			 'BEGIN:VCALENDAR' || char(13) || char(10) || 'END:VCALENDAR')`,
		`INSERT INTO contacts (user_id, uid, first_name, last_name, email) VALUES
			(1, 'c-1@test', 'Ana', 'Ribeiro', 'ana@example.com'),
			(1, 'c-2@test', 'Bruno', 'Costa', 'bruno@example.com'),
			(2, 'c-3@test', 'Carla', 'Dias', 'carla@example.com')`,
	}
	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, stmt)
		}
	}
	return db
}

// runMigration performs the same sequence as the migrate command.
func runMigration(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&model.AddressBook{}, &model.DAVChange{}); err != nil {
		t.Fatalf("dav tables: %v", err)
	}
	if err := PrepareDAVSchema(db); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Calendar{}, &model.Event{}, &model.Contact{},
		&model.AddressBook{}, &model.DAVChange{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	if err := FinishDAVMigration(db); err != nil {
		t.Fatalf("finish: %v", err)
	}
}

func TestMigrationBackfillsExistingRows(t *testing.T) {
	db := openLegacyDB(t)
	runMigration(t, db)

	// Every event gets a distinct resource name, which is what the unique
	// index over (calendar_id, resource_uri) needs.
	var events []model.Event
	if err := db.Order("id").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	seen := map[string]bool{}
	for _, ev := range events {
		if ev.ResourceURI == "" {
			t.Fatalf("event %d has no resource name", ev.ID)
		}
		if seen[ev.ResourceURI] {
			t.Fatalf("duplicate resource name %q", ev.ResourceURI)
		}
		seen[ev.ResourceURI] = true
		if ev.ETag == "" {
			t.Fatalf("event %d has no ETag despite having a stored blob", ev.ID)
		}
	}

	// Calendars get stable URIs; the default one keeps the well-known name.
	var cals []model.Calendar
	if err := db.Order("id").Find(&cals).Error; err != nil {
		t.Fatal(err)
	}
	if cals[0].URI != "default" {
		t.Fatalf("default calendar URI = %q, want default", cals[0].URI)
	}
	if cals[1].URI != "work-projects" {
		t.Fatalf("second calendar URI = %q, want work-projects", cals[1].URI)
	}
	for _, cal := range cals {
		if cal.SyncToken == 0 {
			t.Fatalf("calendar %d has sync token 0; clients could not tell it apart from an initial sync", cal.ID)
		}
	}

	// Every user with contacts gets a default address book, and each contact
	// is attached to it.
	var books []model.AddressBook
	if err := db.Find(&books).Error; err != nil {
		t.Fatal(err)
	}
	if len(books) != 2 {
		t.Fatalf("expected 1 address book per user with contacts, got %d", len(books))
	}
	var orphans int64
	db.Model(&model.Contact{}).Where("address_book_id IS NULL OR address_book_id = 0").Count(&orphans)
	if orphans != 0 {
		t.Fatalf("%d contacts left without an address book", orphans)
	}
	var contacts []model.Contact
	db.Find(&contacts)
	for _, ct := range contacts {
		if ct.ResourceURI == "" {
			t.Fatalf("contact %d has no resource name", ct.ID)
		}
	}
}

// The old schema forbade two users from holding the same iCalendar UID, so
// importing a shared public calendar failed for the second user.
func TestMigrationDropsGlobalUIDUniqueIndex(t *testing.T) {
	db := openLegacyDB(t)
	runMigration(t, db)

	shared := model.Event{
		UserID: 2, CalendarID: 3, UID: "ev-1@test", Summary: "Same UID, other user",
		ResourceURI: "other.ics",
	}
	if err := db.Create(&shared).Error; err != nil {
		t.Fatalf("two users must be allowed to hold the same UID: %v", err)
	}

	// A recurrence override shares the master's UID inside the same calendar.
	override := model.Event{
		UserID: 1, CalendarID: 1, UID: "ev-1@test", Summary: "Override",
		ResourceURI: "override.ics",
	}
	if err := db.Create(&override).Error; err != nil {
		t.Fatalf("a recurrence override must be storable alongside its master: %v", err)
	}
}

// Two objects may never claim the same URL inside one collection.
func TestMigrationCreatesResourceUniquenessIndexes(t *testing.T) {
	db := openLegacyDB(t)
	runMigration(t, db)

	m := db.Migrator()
	if !m.HasIndex(&model.Event{}, "idx_events_cal_uri") {
		t.Fatal("missing unique index on (calendar_id, resource_uri)")
	}
	if !m.HasIndex(&model.Contact{}, "idx_contacts_ab_uri") {
		t.Fatal("missing unique index on (address_book_id, resource_uri)")
	}

	first := model.Event{UserID: 1, CalendarID: 1, UID: "dup-a@test", Summary: "A", ResourceURI: "same.ics"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	second := model.Event{UserID: 1, CalendarID: 1, UID: "dup-b@test", Summary: "B", ResourceURI: "same.ics"}
	if err := db.Create(&second).Error; err == nil {
		t.Fatal("two events shared a resource name inside one calendar")
	}
	// The same name in a different calendar is fine.
	other := model.Event{UserID: 1, CalendarID: 2, UID: "dup-c@test", Summary: "C", ResourceURI: "same.ics"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("resource names are scoped per collection: %v", err)
	}
}

func TestMigrationIsIdempotent(t *testing.T) {
	db := openLegacyDB(t)
	runMigration(t, db)

	var before []model.Event
	db.Order("id").Find(&before)

	runMigration(t, db) // running migrate twice must be safe

	var after []model.Event
	db.Order("id").Find(&after)
	if len(before) != len(after) {
		t.Fatalf("row count changed on the second run: %d → %d", len(before), len(after))
	}
	for i := range before {
		if before[i].ResourceURI != after[i].ResourceURI {
			t.Fatalf("resource name was reassigned on re-run: %q → %q",
				before[i].ResourceURI, after[i].ResourceURI)
		}
	}
	var books int64
	db.Model(&model.AddressBook{}).Count(&books)
	if books != 2 {
		t.Fatalf("re-running the migration created duplicate address books: %d", books)
	}
}

func TestMigrationOnEmptyDatabase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	// A fresh install must not need the legacy tables to exist.
	runMigration(t, db)
}
