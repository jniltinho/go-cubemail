package dav

import (
	"errors"
	"fmt"
	"time"

	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// ErrInvalidSyncToken is returned when a client presents a sync-token older
// than the retained changelog window. Callers must answer the REPORT with
// 403 Forbidden and a DAV:valid-sync-token error element, which tells the
// client to discard its state and run a full sync.
var ErrInvalidSyncToken = errors.New("dav: sync-token predates the retained changelog")

// ErrUnknownCollectionKind guards against typos in the kind discriminator.
var ErrUnknownCollectionKind = errors.New("dav: unknown collection kind")

// Change is one entry returned by ChangesSince.
type Change struct {
	URI     string
	Deleted bool
}

// Store reads and writes the DAV synchronisation state.
type Store struct{ db *gorm.DB }

// NewStore creates a Store backed by the given database connection.
func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

// DB exposes the underlying connection so callers can start a transaction that
// spans both the object write and the changelog append.
func (s *Store) DB() *gorm.DB { return s.db }

// collectionModel maps a kind discriminator to the GORM model holding its
// SyncToken column.
func collectionModel(kind string) (any, error) {
	switch kind {
	case model.CollectionCalendar:
		return &model.Calendar{}, nil
	case model.CollectionAddressBook:
		return &model.AddressBook{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownCollectionKind, kind)
	}
}

// Record bumps the collection revision and appends a changelog entry, and must
// run inside the same transaction as the object write it describes. Passing a
// deleted flag records a tombstone so the next sync-collection REPORT can tell
// the client the resource is gone.
//
// It returns the new revision, which the caller stores on the object row as its
// SyncRevision.
func Record(tx *gorm.DB, kind string, collID uint, uri string, deleted bool) (uint64, error) {
	rev, err := bumpRevision(tx, kind, collID)
	if err != nil {
		return 0, err
	}
	change := model.DAVChange{
		CollectionKind: kind,
		CollectionID:   collID,
		SyncRevision:   rev,
		URI:            uri,
		Deleted:        deleted,
	}
	if err := tx.Create(&change).Error; err != nil {
		return 0, err
	}
	return rev, nil
}

// RecordIfExists behaves like Record but treats a missing collection as a
// no-op instead of an error, returning revision 0.
//
// Not every writer owns a DAV collection: ActiveSync stores VTODO tasks that
// belong to no calendar, and callers may write rows before the collection is
// provisioned. Those objects simply are not exposed over DAV, and failing their
// write because there is nothing to synchronise would be wrong.
func RecordIfExists(tx *gorm.DB, kind string, collID uint, uri string, deleted bool) (uint64, error) {
	if collID == 0 || uri == "" {
		return 0, nil
	}
	rev, err := Record(tx, kind, collID, uri, deleted)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return rev, err
}

// bumpRevision atomically increments the collection's sync token and returns
// the new value. The increment is done with a SQL expression rather than a
// read-modify-write so concurrent writers cannot hand out the same revision.
func bumpRevision(tx *gorm.DB, kind string, collID uint) (uint64, error) {
	m, err := collectionModel(kind)
	if err != nil {
		return 0, err
	}
	res := tx.Model(m).Where("id = ?", collID).
		UpdateColumn("sync_token", gorm.Expr("sync_token + 1"))
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return currentRevision(tx, kind, collID)
}

func currentRevision(tx *gorm.DB, kind string, collID uint) (uint64, error) {
	m, err := collectionModel(kind)
	if err != nil {
		return 0, err
	}
	var rev uint64
	err = tx.Model(m).Where("id = ?", collID).Select("sync_token").Scan(&rev).Error
	return rev, err
}

// CurrentRevision returns the collection's current sync revision.
func (s *Store) CurrentRevision(kind string, collID uint) (uint64, error) {
	return currentRevision(s.db, kind, collID)
}

// ChangesSince returns the resources created, modified or deleted after the
// given revision, collapsed so each URI appears at most once with its latest
// state. It also returns the revision the client should store as its new token.
//
// A since value below the collection's PrunedRevision means the changelog no
// longer covers the gap; ErrInvalidSyncToken is returned so the caller can ask
// the client for a full resync.
func (s *Store) ChangesSince(kind string, collID uint, since uint64) ([]Change, uint64, error) {
	current, err := s.CurrentRevision(kind, collID)
	if err != nil {
		return nil, 0, err
	}
	if since > 0 {
		pruned, err := s.prunedRevision(kind, collID)
		if err != nil {
			return nil, 0, err
		}
		if since < pruned {
			return nil, 0, ErrInvalidSyncToken
		}
		if since > current {
			// A token from the future can only mean the collection was reset.
			return nil, 0, ErrInvalidSyncToken
		}
	}

	var rows []model.DAVChange
	err = s.db.Where("collection_kind = ? AND collection_id = ? AND sync_revision > ?",
		kind, collID, since).
		Order("sync_revision").
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	// Collapse per URI: only the most recent state of a resource matters.
	// Iterating in ascending revision order means later rows overwrite earlier
	// ones while `order` preserves first-seen ordering for a stable response.
	latest := make(map[string]bool, len(rows))
	var order []string
	for _, r := range rows {
		if _, seen := latest[r.URI]; !seen {
			order = append(order, r.URI)
		}
		latest[r.URI] = r.Deleted
	}
	out := make([]Change, 0, len(order))
	for _, uri := range order {
		out = append(out, Change{URI: uri, Deleted: latest[uri]})
	}
	return out, current, nil
}

func (s *Store) prunedRevision(kind string, collID uint) (uint64, error) {
	m, err := collectionModel(kind)
	if err != nil {
		return 0, err
	}
	var pruned uint64
	err = s.db.Model(m).Where("id = ?", collID).Select("pruned_revision").Scan(&pruned).Error
	return pruned, err
}

// Cleanup drops changelog entries older than the cutoff, always keeping the
// most recent revision of every collection so a client sitting exactly at the
// head is never told to resync. Each affected collection's PrunedRevision is
// raised to the highest revision that was discarded.
func (s *Store) Cleanup(cutoff time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		type group struct {
			CollectionKind string
			CollectionID   uint
			MaxRev         uint64
		}
		var groups []group
		if err := tx.Model(&model.DAVChange{}).
			Select("collection_kind, collection_id, MAX(sync_revision) AS max_rev").
			Where("created_at < ?", cutoff).
			Group("collection_kind, collection_id").
			Scan(&groups).Error; err != nil {
			return err
		}
		for _, g := range groups {
			head, err := currentRevision(tx, g.CollectionKind, g.CollectionID)
			if err != nil {
				return err
			}
			if head == 0 {
				continue // collection was removed or never initialised
			}
			// Never discard the head revision: a client sitting exactly at it
			// must still be served a delta instead of being told to resync.
			upTo := g.MaxRev
			if upTo >= head {
				upTo = head - 1
			}
			if upTo == 0 {
				continue
			}
			if err := tx.Where("collection_kind = ? AND collection_id = ? AND sync_revision <= ?",
				g.CollectionKind, g.CollectionID, upTo).
				Delete(&model.DAVChange{}).Error; err != nil {
				return err
			}
			m, err := collectionModel(g.CollectionKind)
			if err != nil {
				return err
			}
			if err := tx.Model(m).Where("id = ?", g.CollectionID).
				UpdateColumn("pruned_revision", upTo).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
