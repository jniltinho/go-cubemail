package model

import "time"

// AddressBook is a CardDAV collection of contacts owned by a user.
//
// It mirrors Calendar on the CalDAV side: URI is the stable path segment used
// in DAV URLs (never derived from DisplayName, so renaming does not break
// client sync), and SyncToken is the monotonic revision counter fed by the
// DAVChange changelog.
type AddressBook struct {
	ID          uint   `gorm:"primaryKey"`
	UserID      uint   `gorm:"index:idx_ab_user_uri,priority:1;not null"`
	URI         string `gorm:"size:255;index:idx_ab_user_uri,priority:2;not null"`
	DisplayName string `gorm:"size:255"`
	Description string `gorm:"type:text"`
	IsDefault   bool   `gorm:"default:false"`

	// SyncToken is the current revision of the collection; PrunedRevision is the
	// oldest revision still present in the changelog. A client presenting a token
	// older than PrunedRevision must be told to restart with a full sync.
	SyncToken      uint64 `gorm:"not null;default:1"`
	PrunedRevision uint64 `gorm:"not null;default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
