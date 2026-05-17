package model

import "time"

type Contact struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"index;not null"`
	FirstName string
	LastName  string
	Email     string `gorm:"not null"`
	Phone     string
	Company   string
	Notes     string
	GroupID   *uint
	CreatedAt time.Time
	UpdatedAt time.Time
}
