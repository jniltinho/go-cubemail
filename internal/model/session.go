package model

import "time"

// Session persists an active HTTP session to the database so it survives server restarts.
// The password is stored AES-GCM encrypted; the key lives only in the server configuration.
type Session struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)"`
	IMAPHost    string    `gorm:"type:varchar(255)"`
	IMAPPort    int
	Username    string    `gorm:"type:varchar(255)"`
	EncPassword string    `gorm:"type:text"`
	LastUsed    time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
