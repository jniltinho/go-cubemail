package model

import "time"

// Event represents a scheduled calendar entry stored as a VEVENT.
// ICalContent holds the canonical RFC 5545 representation; denormalized
// fields enable fast range queries without parsing ICS on every request.
type Event struct {
	ID             uint       `gorm:"primaryKey"`
	CalendarID     uint       `gorm:"index;not null"`
	UserID         uint       `gorm:"index;not null"`
	UID            string     `gorm:"size:255;uniqueIndex;not null"`
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

	Attendees []EventAttendee `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE"`
	Calendar  Calendar        `gorm:"foreignKey:CalendarID"`
}
