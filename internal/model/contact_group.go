package model

type ContactGroup struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"index;not null"`
	Name   string `gorm:"not null"`
}
