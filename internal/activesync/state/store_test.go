package state

// Tests for the device sync-state store.

import (
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&EasDevice{}, &EasFolderState{}, &ImapFolderMapping{}); err != nil {
		t.Fatal(err)
	}
	// One connection keeps SQLite from serialising writers into "database is
	// locked" errors, so the test exercises the compare-and-swap rather than
	// the driver's locking.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	return db
}

func TestNextCollectionSyncKeyIncrements(t *testing.T) {
	store := NewStore(openStoreTestDB(t))

	for want := 1; want <= 3; want++ {
		key, err := store.NextCollectionSyncKey(1, "vcard/personal")
		if err != nil {
			t.Fatal(err)
		}
		got, err := store.GetCollectionSyncKey(1, "vcard/personal")
		if err != nil {
			t.Fatal(err)
		}
		if key != got {
			t.Fatalf("returned key %q but stored %q", key, got)
		}
	}
	final, _ := store.GetCollectionSyncKey(1, "vcard/personal")
	if final != "3" {
		t.Fatalf("sync key = %q after 3 increments, want 3", final)
	}
}

// Collections must not share a counter.
func TestSyncKeysAreScopedPerCollection(t *testing.T) {
	store := NewStore(openStoreTestDB(t))

	if _, err := store.NextCollectionSyncKey(1, "vcard/personal"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NextCollectionSyncKey(1, "vcard/personal"); err != nil {
		t.Fatal(err)
	}
	other, err := store.NextCollectionSyncKey(1, "vevent/personal")
	if err != nil {
		t.Fatal(err)
	}
	if other != "1" {
		t.Fatalf("a second collection started at %q, want 1", other)
	}
	// And devices must not share one either.
	otherDevice, err := store.NextCollectionSyncKey(2, "vcard/personal")
	if err != nil {
		t.Fatal(err)
	}
	if otherDevice != "1" {
		t.Fatalf("a second device started at %q, want 1", otherDevice)
	}
}

// Two concurrent Sync requests for the same collection — which devices do
// pipeline — must never receive the same key. A duplicate would let the server
// accept a stale request as current and silently drop a batch of changes.
func TestNextCollectionSyncKeyNeverRepeatsUnderConcurrency(t *testing.T) {
	store := NewStore(openStoreTestDB(t))

	const workers = 8
	const perWorker = 15

	var mu sync.Mutex
	seen := map[string]int{}
	var failures []error

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < perWorker; j++ {
				key, err := store.NextCollectionSyncKey(1, "vcard/personal")
				mu.Lock()
				if err != nil {
					failures = append(failures, err)
				} else {
					seen[key]++
				}
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()

	if len(failures) > 0 {
		t.Fatalf("%d increments failed, first: %v", len(failures), failures[0])
	}
	for key, count := range seen {
		if count > 1 {
			t.Fatalf("sync key %q was handed out %d times", key, count)
		}
	}
	if len(seen) != workers*perWorker {
		t.Fatalf("got %d distinct keys, want %d", len(seen), workers*perWorker)
	}
}
