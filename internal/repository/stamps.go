package repository

// Cheap change-detection queries.
//
// ActiveSync's Ping command long-polls: a single request loops until the
// heartbeat expires, which for a 900-second heartbeat means hundreds of
// iterations. Loading whole rows there — with the iCalendar and vCard blobs and
// eager-loaded attendees — reads hundreds of megabytes per device per Ping just
// to compare timestamps. These queries return only what change detection needs.

import (
	"time"

	"go-cubemail/internal/model"
)

// ObjectStamp is the identity and version of one object, without its payload.
//
// SyncRevision is the collection revision of the last write. It is strictly
// monotonic, so it distinguishes two edits made inside the same second, which
// UpdatedAt cannot. It is zero for rows that belong to no DAV collection —
// ActiveSync tasks, for one — hence UpdatedAt is kept as the fallback.
type ObjectStamp struct {
	ID           uint
	UpdatedAt    time.Time
	SyncRevision uint64
}

// ListContactStamps returns the version of every contact in an address book.
func (r *ContactRepo) ListContactStamps(userID, bookID uint) ([]ObjectStamp, error) {
	var stamps []ObjectStamp
	err := r.db.Model(&model.Contact{}).
		Select("id, updated_at, sync_revision").
		Where("user_id = ? AND address_book_id = ?", userID, bookID).
		Scan(&stamps).Error
	return stamps, err
}

// ListEventStamps returns the version of every event in a calendar.
func (r *EventRepo) ListEventStamps(userID, calendarID uint) ([]ObjectStamp, error) {
	var stamps []ObjectStamp
	err := r.db.Model(&model.Event{}).
		Select("id, updated_at, sync_revision").
		Where("user_id = ? AND calendar_id = ?", userID, calendarID).
		Scan(&stamps).Error
	return stamps, err
}

// Revision returns the current sync token of one address book.
func (r *AddressBookRepo) Revision(userID, bookID uint) (uint64, error) {
	var rev uint64
	err := r.db.Model(&model.AddressBook{}).
		Select("COALESCE(sync_token, 0)").
		Where("id = ? AND user_id = ?", bookID, userID).
		Scan(&rev).Error
	return rev, err
}

// Revision returns the current sync token of one calendar.
func (r *CalendarRepo) Revision(userID, calendarID uint) (uint64, error) {
	var rev uint64
	err := r.db.Model(&model.Calendar{}).
		Select("COALESCE(sync_token, 0)").
		Where("id = ? AND user_id = ?", calendarID, userID).
		Scan(&rev).Error
	return rev, err
}
