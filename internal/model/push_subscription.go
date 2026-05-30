package model

import "time"

// PushSubscription stores a Web Push subscription endpoint for a user.
// Each browser/device has its own subscription; all are notified on new mail.
type PushSubscription struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index;not null"`
	Endpoint  string    `gorm:"type:text;not null"`
	P256DH    string    `gorm:"type:text;not null"` // client public key
	Auth      string    `gorm:"size:255;not null"`  // auth secret
	CreatedAt time.Time
}
