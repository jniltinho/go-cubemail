// Package model defines the GORM database models for go-cubemail.
package model

import "time"

// User represents an authenticated IMAP account. ImapUser is the IMAP username
// and serves as the unique identifier linking identities, settings, and contacts.
type User struct {
	ID         uint      `gorm:"primaryKey"`
	ImapUser   string    `gorm:"uniqueIndex;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Settings   UserSettings
	Identities []Identity
}
