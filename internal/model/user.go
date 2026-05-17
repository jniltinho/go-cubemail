package model

import "time"

type User struct {
	ID         uint      `gorm:"primaryKey"`
	ImapUser   string    `gorm:"uniqueIndex;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Settings   UserSettings
	Identities []Identity
}
