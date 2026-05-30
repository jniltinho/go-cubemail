package model

import "time"

// Contact stores a single address-book entry belonging to a user.
// GroupID optionally links the contact to a ContactGroup.
type Contact struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"index;not null"`
	UID       string `gorm:"size:255;index"` // vCard UID for CardDAV round-trip
	FirstName string
	LastName  string
	Email     string `gorm:"not null"`
	Title     string
	Phone     string
	Company   string
	Notes     string
	GroupID   *uint
	CreatedAt time.Time
	UpdatedAt time.Time
}
