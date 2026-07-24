package commands

// Tests for Ping's change detection.
//
// Ping long-polls: one request loops until the heartbeat expires, so with a
// 900-second heartbeat and a 2-second poll interval the detection code runs
// hundreds of times per request per device. What it must never do is read the
// iCalendar / vCard payloads — those are megabytes per contact once photos are
// involved, and the comparison only needs a timestamp.

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── SQL capture ───────────────────────────────────────────────────────────

// sqlRecorder is a GORM logger that keeps every statement it sees, so a test
// can assert what the database was actually asked for.
type sqlRecorder struct {
	statements []string
}

func (r *sqlRecorder) LogMode(logger.LogLevel) logger.Interface { return r }
func (r *sqlRecorder) Info(context.Context, string, ...any)     {}
func (r *sqlRecorder) Warn(context.Context, string, ...any)     {}
func (r *sqlRecorder) Error(context.Context, string, ...any)    {}
func (r *sqlRecorder) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	r.statements = append(r.statements, sql)
}

func (r *sqlRecorder) mentions(needle string) bool {
	for _, s := range r.statements {
		if strings.Contains(strings.ToLower(s), strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func openPingTestDB(t *testing.T) (*gorm.DB, *sqlRecorder) {
	t.Helper()
	rec := &sqlRecorder{}
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: rec})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Contact{}, &model.AddressBook{},
		&model.Calendar{}, &model.Event{}, &model.EventAttendee{}, &model.DAVChange{},
	); err != nil {
		t.Fatal(err)
	}
	return db, rec
}

// The payload columns must never appear in a change-detection query.
func TestPingStampQueriesDoNotReadPayloads(t *testing.T) {
	db, rec := openPingTestDB(t)
	contactRepo := repository.NewContactRepo(db)
	eventRepo := repository.NewEventRepo(db)
	calRepo := repository.NewCalendarRepo(db)
	bookRepo := repository.NewAddressBookRepo(db)

	book, err := bookRepo.EnsureDefault(1)
	if err != nil {
		t.Fatal(err)
	}
	ct := model.Contact{UserID: 1, FirstName: "Ana", Email: "ana@example.com"}
	if err := contactRepo.Create(&ct); err != nil {
		t.Fatal(err)
	}
	cal, err := calRepo.EnsureDefault(1)
	if err != nil {
		t.Fatal(err)
	}
	ev := model.Event{UserID: 1, CalendarID: cal.ID, UID: "e1@test", Summary: "Standup"}
	if err := eventRepo.Create(&ev); err != nil {
		t.Fatal(err)
	}

	// Only what the polling loop runs is under scrutiny.
	rec.statements = nil

	if _, err := bookRepo.Revision(1, book.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := contactRepo.ListContactStamps(1, book.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := calRepo.Revision(1, cal.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := eventRepo.ListEventStamps(1, cal.ID); err != nil {
		t.Fatal(err)
	}

	for _, blob := range []string{"vcard_content", "i_cal_content"} {
		if rec.mentions(blob) {
			t.Errorf("change detection read the %s column:\n%s", blob,
				strings.Join(rec.statements, "\n"))
		}
	}
	// Attendees are eager-loaded by the full listing; the stamp query must not.
	if rec.mentions("event_attendees") {
		t.Errorf("change detection joined event_attendees:\n%s",
			strings.Join(rec.statements, "\n"))
	}
}

func TestStampQueriesReturnCurrentVersions(t *testing.T) {
	db, _ := openPingTestDB(t)
	contactRepo := repository.NewContactRepo(db)
	book, err := repository.NewAddressBookRepo(db).EnsureDefault(1)
	if err != nil {
		t.Fatal(err)
	}

	first := model.Contact{UserID: 1, FirstName: "Ana", Email: "ana@example.com"}
	second := model.Contact{UserID: 1, FirstName: "Bruno", Email: "bruno@example.com"}
	other := model.Contact{UserID: 2, FirstName: "Carla", Email: "carla@example.com"}
	for _, c := range []*model.Contact{&first, &second, &other} {
		if err := contactRepo.Create(c); err != nil {
			t.Fatal(err)
		}
	}

	stamps, err := contactRepo.ListContactStamps(1, book.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stamps) != 2 {
		t.Fatalf("stamps are not scoped to the collection: got %d, want 2", len(stamps))
	}
	for _, s := range stamps {
		if s.ID == 0 || s.UpdatedAt.IsZero() {
			t.Fatalf("incomplete stamp: %+v", s)
		}
		if s.SyncRevision == 0 {
			t.Fatalf("stamp is missing the sync revision: %+v", s)
		}
	}
}

// The collection revision must advance on every write, including deletions —
// that is what lets Ping skip the scan while the collection is quiet.
func TestCollectionRevisionAdvancesOnEveryWrite(t *testing.T) {
	db, _ := openPingTestDB(t)
	repo := repository.NewContactRepo(db)
	books := repository.NewAddressBookRepo(db)

	book, err := books.EnsureDefault(1)
	if err != nil {
		t.Fatal(err)
	}
	ct := model.Contact{UserID: 1, FirstName: "Ana", Email: "ana@example.com"}
	if err := repo.Create(&ct); err != nil {
		t.Fatal(err)
	}
	afterCreate, _ := books.Revision(1, book.ID)

	ct.Company = "Criarenet"
	if err := repo.Update(&ct); err != nil {
		t.Fatal(err)
	}
	afterUpdate, _ := books.Revision(1, book.ID)
	if afterUpdate <= afterCreate {
		t.Fatalf("revision did not advance on update: %d → %d", afterCreate, afterUpdate)
	}

	if err := repo.Delete(1, ct.ID); err != nil {
		t.Fatal(err)
	}
	afterDelete, _ := books.Revision(1, book.ID)
	if afterDelete <= afterUpdate {
		t.Fatalf("revision did not advance on delete: %d → %d", afterUpdate, afterDelete)
	}
}

// ── Detection semantics ───────────────────────────────────────────────────

func TestUnchangedSince(t *testing.T) {
	tests := []struct {
		name     string
		cached   uint64
		revision uint64
		want     bool
	}{
		{name: "same revision means nothing changed", cached: 7, revision: 7, want: true},
		{name: "advanced revision means something changed", cached: 7, revision: 8, want: false},
		{
			// Collections that predate the counter, or a Sync that never
			// recorded one, must fall through to the row comparison.
			name: "zero revision forces the slow path", cached: 7, revision: 0, want: false,
		},
		{name: "no recorded revision forces the slow path", cached: 0, revision: 7, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cache := state.ItemSyncCache{Items: map[string]state.ItemSyncItem{}, DavRevision: tc.cached}
			if got := unchangedSince(cache, tc.revision); got != tc.want {
				t.Fatalf("unchangedSince(%d, %d) = %v, want %v", tc.cached, tc.revision, got, tc.want)
			}
		})
	}
}

func TestStampsDiffer(t *testing.T) {
	at := func(sec int64) time.Time { return time.Unix(sec, 0) }
	cache := func(items map[string]int64) state.ItemSyncCache {
		c := state.ItemSyncCache{Items: map[string]state.ItemSyncItem{}}
		for k, v := range items {
			c.Items[k] = state.ItemSyncItem{UpdatedAt: v}
		}
		return c
	}

	tests := []struct {
		name   string
		cached map[string]int64
		stamps []repository.ObjectStamp
		want   bool
	}{
		{
			name:   "identical state",
			cached: map[string]int64{"1": 100, "2": 200},
			stamps: []repository.ObjectStamp{{ID: 1, UpdatedAt: at(100)}, {ID: 2, UpdatedAt: at(200)}},
			want:   false,
		},
		{
			name:   "an object was modified",
			cached: map[string]int64{"1": 100},
			stamps: []repository.ObjectStamp{{ID: 1, UpdatedAt: at(101)}},
			want:   true,
		},
		{
			name:   "an object was added",
			cached: map[string]int64{"1": 100},
			stamps: []repository.ObjectStamp{{ID: 1, UpdatedAt: at(100)}, {ID: 2, UpdatedAt: at(200)}},
			want:   true,
		},
		{
			name:   "an object was deleted",
			cached: map[string]int64{"1": 100, "2": 200},
			stamps: []repository.ObjectStamp{{ID: 1, UpdatedAt: at(100)}},
			want:   true,
		},
		{
			// Equal counts must not be mistaken for an equal set.
			name:   "one added and one deleted",
			cached: map[string]int64{"1": 100},
			stamps: []repository.ObjectStamp{{ID: 2, UpdatedAt: at(100)}},
			want:   true,
		},
		{
			name:   "empty on both sides",
			cached: map[string]int64{},
			stamps: nil,
			want:   false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stampsDiffer(cache(tc.cached), tc.stamps); got != tc.want {
				t.Fatalf("stampsDiffer = %v, want %v", got, tc.want)
			}
		})
	}
}

// Two writes inside the same second share an UpdatedAt, so a timestamp
// comparison declares the second one invisible. The collection revision is
// strictly monotonic and catches it.
func TestChangeWithinTheSameSecondIsDetected(t *testing.T) {
	sameSecond := time.Unix(1_700_000_000, 0)

	cache := state.ItemSyncCache{Items: map[string]state.ItemSyncItem{
		serverIDForUint(1): {UpdatedAt: sameSecond.Unix(), Revision: 4},
	}}
	stamps := []repository.ObjectStamp{{ID: 1, UpdatedAt: sameSecond, SyncRevision: 5}}

	if !stampsDiffer(cache, stamps) {
		t.Fatal("a second edit within the same second went undetected")
	}
	// Same revision means genuinely unchanged.
	stamps[0].SyncRevision = 4
	if stampsDiffer(cache, stamps) {
		t.Fatal("an unchanged item was reported as changed")
	}
}

func TestSameVersionFallsBackToTimestamp(t *testing.T) {
	tests := []struct {
		name   string
		cached state.ItemSyncItem
		now    state.ItemSyncItem
		want   bool
	}{
		{
			name:   "revisions decide when both are present",
			cached: state.ItemSyncItem{UpdatedAt: 100, Revision: 4},
			now:    state.ItemSyncItem{UpdatedAt: 100, Revision: 5},
			want:   false,
		},
		{
			name:   "equal revisions win over a differing timestamp",
			cached: state.ItemSyncItem{UpdatedAt: 100, Revision: 4},
			now:    state.ItemSyncItem{UpdatedAt: 101, Revision: 4},
			want:   true,
		},
		{
			// ActiveSync tasks belong to no DAV collection and never get one.
			name:   "no revision falls back to the timestamp",
			cached: state.ItemSyncItem{UpdatedAt: 100},
			now:    state.ItemSyncItem{UpdatedAt: 100},
			want:   true,
		},
		{
			name:   "a cache written before the field existed still compares",
			cached: state.ItemSyncItem{UpdatedAt: 100},
			now:    state.ItemSyncItem{UpdatedAt: 100, Revision: 9},
			want:   true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cached.SameVersion(tc.now); got != tc.want {
				t.Fatalf("SameVersion = %v, want %v", got, tc.want)
			}
		})
	}
}

// The revision travels through the JSON blob the folder state is stored in;
// older rows simply lack the field.
func TestItemSyncCacheCarriesRevision(t *testing.T) {
	c := state.ItemSyncCache{
		Items:       map[string]state.ItemSyncItem{"1": {UpdatedAt: 100}},
		DavRevision: 42,
	}
	back := state.LoadItemSyncCache(state.EncodeItemSyncCache(c))
	if back.DavRevision != 42 {
		t.Fatalf("revision did not round-trip: %d", back.DavRevision)
	}
	if back.Items["1"].UpdatedAt != 100 {
		t.Fatal("item cache did not round-trip")
	}

	legacy := state.LoadItemSyncCache(`{"items":{"1":{"updated_at":100}}}`)
	if legacy.DavRevision != 0 {
		t.Fatalf("a cache written before the field existed must decode to 0, got %d", legacy.DavRevision)
	}
}
