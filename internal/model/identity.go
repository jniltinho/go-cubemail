package model

type Identity struct {
	ID          uint   `gorm:"primaryKey"`
	UserID      uint   `gorm:"index;not null"`
	DisplayName string
	Email       string `gorm:"not null"`
	ReplyTo     string
	Signature   string
	IsDefault   bool
}
