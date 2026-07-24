package dav

import (
	"errors"
	"testing"
	"time"

	"go-cubemail/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openTestDB builds an isolated in-memory database with the DAV tables.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.Calendar{}, &model.AddressBook{}, &model.DAVChange{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// newCalendar inserts a calendar collection to record changes against.
func newCalendar(t *testing.T, db *gorm.DB) *model.Calendar {
	t.Helper()
	cal := model.Calendar{UserID: 1, Name: "Personal", URI: "default", SyncToken: 1}
	if err := db.Create(&cal).Error; err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	return &cal
}

func TestComputeETagIsStableAndContentSensitive(t *testing.T) {
	a := ComputeETag([]byte("BEGIN:VCALENDAR"))
	if a != ComputeETag([]byte("BEGIN:VCALENDAR")) {
		t.Fatal("ETag must be deterministic for identical content")
	}
	if a == ComputeETag([]byte("BEGIN:VCALENDAR ")) {
		t.Fatal("ETag must change when the content changes")
	}
	if len(a) != 32 {
		t.Fatalf("ETag length = %d, want 32 hex chars", len(a))
	}
}

func TestMatchETag(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		current string
		want    bool
	}{
		{name: "empty header never matches", header: "", current: "abc", want: false},
		{name: "wildcard always matches", header: "*", current: "abc", want: true},
		{name: "quoted exact match", header: `"abc"`, current: "abc", want: true},
		{name: "unquoted exact match", header: "abc", current: "abc", want: true},
		{name: "weak validator match", header: `W/"abc"`, current: "abc", want: true},
		{name: "mismatch", header: `"other"`, current: "abc", want: false},
		{name: "list containing the tag", header: `"x", "abc" , "y"`, current: "abc", want: true},
		{name: "list without the tag", header: `"x","y"`, current: "abc", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := MatchETag(tc.header, tc.current); got != tc.want {
				t.Fatalf("MatchETag(%q, %q) = %v, want %v", tc.header, tc.current, got, tc.want)
			}
		})
	}
}

func TestSyncTokenRoundTrip(t *testing.T) {
	for _, rev := range []uint64{0, 1, 42, 1 << 40} {
		got, ok := ParseSyncToken(SyncToken(rev))
		if !ok || got != rev {
			t.Fatalf("round trip of %d = (%d, %v)", rev, got, ok)
		}
	}
	// An empty token means "never synced".
	if rev, ok := ParseSyncToken(""); !ok || rev != 0 {
		t.Fatalf("empty token = (%d, %v), want (0, true)", rev, ok)
	}
	// A bare decimal is accepted for tokens minted by older builds.
	if rev, ok := ParseSyncToken("7"); !ok || rev != 7 {
		t.Fatalf("legacy token = (%d, %v)", rev, ok)
	}
	if _, ok := ParseSyncToken("not-a-token"); ok {
		t.Fatal("garbage token must be rejected so the client resyncs")
	}
}

func TestRecordAdvancesRevisionAndLogsChange(t *testing.T) {
	db := openTestDB(t)
	cal := newCalendar(t, db)
	store := NewStore(db)

	rev1, err := Record(db, model.CollectionCalendar, cal.ID, "a.ics", false)
	if err != nil {
		t.Fatal(err)
	}
	rev2, err := Record(db, model.CollectionCalendar, cal.ID, "b.ics", false)
	if err != nil {
		t.Fatal(err)
	}
	if rev2 <= rev1 {
		t.Fatalf("revisions must increase: %d then %d", rev1, rev2)
	}

	current, err := store.CurrentRevision(model.CollectionCalendar, cal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current != rev2 {
		t.Fatalf("current revision = %d, want %d", current, rev2)
	}
}

func TestChangesSinceCollapsesToLatestState(t *testing.T) {
	db := openTestDB(t)
	cal := newCalendar(t, db)
	store := NewStore(db)

	base, _ := store.CurrentRevision(model.CollectionCalendar, cal.ID)

	// Created, then modified twice, then deleted: the client only needs to know
	// the resource is gone.
	for i := 0; i < 3; i++ {
		if _, err := Record(db, model.CollectionCalendar, cal.ID, "a.ics", false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Record(db, model.CollectionCalendar, cal.ID, "a.ics", true); err != nil {
		t.Fatal(err)
	}
	if _, err := Record(db, model.CollectionCalendar, cal.ID, "b.ics", false); err != nil {
		t.Fatal(err)
	}

	changes, token, err := store.ChangesSince(model.CollectionCalendar, cal.ID, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 collapsed changes, got %d: %+v", len(changes), changes)
	}
	byURI := map[string]bool{}
	for _, ch := range changes {
		byURI[ch.URI] = ch.Deleted
	}
	if !byURI["a.ics"] {
		t.Fatal("a.ics ended deleted; the collapsed change must say so")
	}
	if byURI["b.ics"] {
		t.Fatal("b.ics still exists and must not be reported as deleted")
	}
	if token == 0 {
		t.Fatal("ChangesSince must return the new token")
	}
}

func TestChangesSinceRejectsTokensOutsideRetention(t *testing.T) {
	db := openTestDB(t)
	cal := newCalendar(t, db)
	store := NewStore(db)

	for i := 0; i < 5; i++ {
		if _, err := Record(db, model.CollectionCalendar, cal.ID, "a.ics", false); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&model.Calendar{}).Where("id = ?", cal.ID).
		Update("pruned_revision", 4).Error; err != nil {
		t.Fatal(err)
	}

	if _, _, err := store.ChangesSince(model.CollectionCalendar, cal.ID, 2); !errors.Is(err, ErrInvalidSyncToken) {
		t.Fatalf("a pruned token must be rejected, got %v", err)
	}
	// A token from the future can only mean the collection was reset.
	if _, _, err := store.ChangesSince(model.CollectionCalendar, cal.ID, 9999); !errors.Is(err, ErrInvalidSyncToken) {
		t.Fatalf("a token ahead of the collection must be rejected, got %v", err)
	}
	// A token still inside the window works.
	if _, _, err := store.ChangesSince(model.CollectionCalendar, cal.ID, 5); err != nil {
		t.Fatalf("token inside the retention window failed: %v", err)
	}
}

func TestCleanupKeepsTheHeadRevision(t *testing.T) {
	db := openTestDB(t)
	cal := newCalendar(t, db)
	store := NewStore(db)

	for i := 0; i < 4; i++ {
		if _, err := Record(db, model.CollectionCalendar, cal.ID, "a.ics", false); err != nil {
			t.Fatal(err)
		}
	}
	// Age every entry past the retention window.
	old := time.Now().Add(-48 * time.Hour)
	if err := db.Model(&model.DAVChange{}).Where("1 = 1").
		UpdateColumn("created_at", old).Error; err != nil {
		t.Fatal(err)
	}

	if err := store.Cleanup(time.Now().Add(-24 * time.Hour)); err != nil {
		t.Fatal(err)
	}

	var remaining int64
	db.Model(&model.DAVChange{}).Count(&remaining)
	if remaining == 0 {
		t.Fatal("cleanup must keep the head revision so an up-to-date client is not forced to resync")
	}

	head, _ := store.CurrentRevision(model.CollectionCalendar, cal.ID)
	if _, _, err := store.ChangesSince(model.CollectionCalendar, cal.ID, head); err != nil {
		t.Fatalf("a client sitting at the head must still be served: %v", err)
	}
	// A client holding a pruned token must be told to start over.
	if _, _, err := store.ChangesSince(model.CollectionCalendar, cal.ID, 1); !errors.Is(err, ErrInvalidSyncToken) {
		t.Fatalf("pruned token should be rejected, got %v", err)
	}
}

func TestRecordIfExistsToleratesMissingCollection(t *testing.T) {
	db := openTestDB(t)

	// ActiveSync stores VTODO tasks that belong to no calendar; those writes
	// must not fail just because there is nothing to synchronise.
	rev, err := RecordIfExists(db, model.CollectionCalendar, 0, "x.ics", false)
	if err != nil || rev != 0 {
		t.Fatalf("collection 0 = (%d, %v), want (0, nil)", rev, err)
	}
	rev, err = RecordIfExists(db, model.CollectionCalendar, 999, "x.ics", false)
	if err != nil || rev != 0 {
		t.Fatalf("unknown collection = (%d, %v), want (0, nil)", rev, err)
	}
}

func TestRecordRollsBackWithItsTransaction(t *testing.T) {
	db := openTestDB(t)
	cal := newCalendar(t, db)
	store := NewStore(db)
	before, _ := store.CurrentRevision(model.CollectionCalendar, cal.ID)

	wantErr := errors.New("write failed")
	err := db.Transaction(func(tx *gorm.DB) error {
		if _, err := Record(tx, model.CollectionCalendar, cal.ID, "a.ics", false); err != nil {
			return err
		}
		return wantErr // the object write fails after the changelog entry
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("transaction error = %v", err)
	}

	after, _ := store.CurrentRevision(model.CollectionCalendar, cal.ID)
	if after != before {
		t.Fatalf("revision leaked from a rolled-back transaction: %d → %d", before, after)
	}
	var count int64
	db.Model(&model.DAVChange{}).Count(&count)
	if count != 0 {
		t.Fatalf("changelog entry survived a rollback: %d rows", count)
	}
}
