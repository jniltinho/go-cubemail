package model

import "time"

// CalendarShare grants a user read (or read+write) access to another user's calendar.
type CalendarShare struct {
	ID           uint      `gorm:"primaryKey"`
	CalendarID   uint      `gorm:"uniqueIndex:idx_cal_share;not null"`
	OwnerID      uint      `gorm:"index;not null"`
	SharedWithID uint      `gorm:"uniqueIndex:idx_cal_share;not null"`
	CanWrite     bool      `gorm:"default:false"`
	CreatedAt    time.Time
}
