package model

import "time"

// Calendar represents a user-owned calendar container for events.
// Each user may have multiple calendars; one calendar is marked as the default
// ("Personal") and cannot be deleted through the API.
type Calendar struct {
	ID                uint      `gorm:"primaryKey"`
	UserID            uint      `gorm:"index;not null"`
	Name              string    `gorm:"size:255;not null"`
	Color             string    `gorm:"size:7;default:'#3788d8'"`
	IsDefault         bool      `gorm:"default:false"`
	IsActive          bool      `gorm:"default:true"`
	IncludeInFreeBusy bool      `gorm:"default:true"`
	SortOrder         int       `gorm:"default:0"`
	CreatedAt         time.Time
	UpdatedAt         time.Time

	// ── CalDAV collection fields ──────────────────────────────────────────
	// URI is the stable path segment used in DAV URLs
	// (/dav/{user}/calendars/{uri}/). It is assigned once at creation and never
	// derived from Name, so renaming a calendar neither invalidates every client
	// URL nor collides when two calendars slugify to the same name.
	URI         string `gorm:"size:255;index"`
	Description string `gorm:"type:text"`
	TimeZone    string `gorm:"type:text"` // serialised VTIMEZONE (CALDAV:calendar-timezone)

	// SyncToken is the collection's current revision; PrunedRevision is the
	// oldest revision still present in the DAVChange changelog. A client
	// presenting a token below PrunedRevision must restart with a full sync.
	SyncToken      uint64 `gorm:"not null;default:1"`
	PrunedRevision uint64 `gorm:"not null;default:0"`
}
