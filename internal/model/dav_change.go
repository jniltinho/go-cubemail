package model

import "time"

// Collection kinds addressed by the DAV synchronisation layer.
const (
	CollectionCalendar    = "calendar"
	CollectionAddressBook = "addressbook"
)

// DAVChange is one entry of the per-collection changelog required by RFC 6578
// (WebDAV Collection Synchronization). Every create, update and delete of a
// calendar object or address object appends a row here inside the same
// transaction that bumps the owning collection's sync token.
//
// Without this table the server cannot tell a client *what was deleted*, which
// forces clients into an endless full-sync loop.
type DAVChange struct {
	ID             uint64 `gorm:"primaryKey"`
	CollectionKind string `gorm:"size:16;not null;index:idx_dav_changes_coll,priority:1"`
	CollectionID   uint   `gorm:"not null;index:idx_dav_changes_coll,priority:2"`
	SyncRevision   uint64 `gorm:"not null;index:idx_dav_changes_coll,priority:3"`

	// URI is the resource name inside the collection ("a1b2c3.ics"), not the UID.
	URI     string `gorm:"size:255;not null"`
	Deleted bool   `gorm:"not null;default:false"`

	CreatedAt time.Time
}
