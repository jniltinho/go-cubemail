package model

// ContactGroup organises contacts into named groups for a user.
// Placeholder: not yet used — planned for future contact group management support.
type ContactGroup struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"index;not null"`
	Name   string `gorm:"not null"`
}
