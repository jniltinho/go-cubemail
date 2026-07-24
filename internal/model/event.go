package model

import "time"

// Event represents a scheduled calendar entry stored as a VEVENT.
// ICalContent holds the original RFC 5545 blob exactly as the client sent it;
// every other field is a denormalized index used for fast range queries and by
// the web UI. Never rebuild ICalContent from the index fields when answering a
// DAV client — that would drop VALARM, VTIMEZONE, X-* and recurrence overrides.
//
// UID is NOT unique: a recurrence override (RECURRENCE-ID) is stored as a
// separate row sharing the master's UID. DAV resource identity is
// (CalendarID, ResourceURI) instead.
type Event struct {
	ID             uint       `gorm:"primaryKey"`
	CalendarID     uint       `gorm:"index;not null;uniqueIndex:idx_events_cal_uri,priority:1"`
	UserID         uint       `gorm:"not null;index:idx_events_user_uid,priority:1"`
	UID            string     `gorm:"size:255;not null;index:idx_events_user_uid,priority:2"`
	Summary        string     `gorm:"size:1000;not null"`
	Description    string     `gorm:"type:text"`
	Location       string     `gorm:"size:255"`
	StartAt        time.Time  `gorm:"index;not null"`
	EndAt          time.Time  `gorm:"index;not null"`
	IsAllDay       bool       `gorm:"default:false"`
	IsTransparent  bool       `gorm:"default:false"`
	Status         string     `gorm:"size:20;default:'CONFIRMED'"`
	Priority       int        `gorm:"default:0"`
	Classification string     `gorm:"size:20;default:'PUBLIC'"`
	Categories     string     `gorm:"size:255"`
	OrganizerName  string     `gorm:"size:255"`
	OrganizerEmail string     `gorm:"size:255"`
	RRule          string     `gorm:"type:text"`
	RecurrenceID   *time.Time
	Sequence       int        `gorm:"default:0"`
	ICalContent    string     `gorm:"type:text"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	IsTask     bool `gorm:"default:false"` // true = VTODO task, false = VEVENT

	// ── CalDAV resource fields ────────────────────────────────────────────
	// ResourceURI is the file name the client chose on PUT ("a1b2c3.ics").
	// It is deliberately independent from UID: RFC 4791 lets a client name a
	// resource anything, and assuming "<uid>.ics" breaks Apple Calendar.
	ResourceURI  string `gorm:"size:255;uniqueIndex:idx_events_cal_uri,priority:2"`
	ETag         string `gorm:"column:etag;size:70"` // sha256 prefix of ICalContent, unquoted
	SyncRevision uint64 `gorm:"index"`               // collection revision of the last write

	Attendees []EventAttendee `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE"`
	Calendar  Calendar        `gorm:"foreignKey:CalendarID"`
}
