package model

import "time"

// CalendarSubscription represents a remote .ics calendar URL subscribed by a user.
// A background worker fetches the URL periodically and upserts events.
type CalendarSubscription struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"index;not null"`
	CalendarID   uint      `gorm:"index;not null"` // target calendar for imported events
	URL          string    `gorm:"size:2048;not null"`
	Name         string    `gorm:"size:255"`
	Color        string    `gorm:"size:7;default:'#999999'"`
	RefreshMins  int       `gorm:"default:60"` // how often to re-fetch (minutes)
	LastSyncedAt *time.Time
	LastError    string    `gorm:"size:512"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
