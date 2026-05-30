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
}
