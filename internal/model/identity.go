package model

// Identity represents a sender identity (display name, email, optional reply-to, signature)
// that a user can select when composing messages.
type Identity struct {
	ID          uint   `gorm:"primaryKey"`
	UserID      uint   `gorm:"index;not null"`
	DisplayName string
	Email       string `gorm:"not null"`
	ReplyTo     string
	Signature   string
	IsDefault   bool
}
