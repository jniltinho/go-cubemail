package model

import "time"

type Draft struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"index;not null"`
	Subject   string
	ToAddr    string
	CcAddr    string
	BccAddr   string
	Body      string `gorm:"type:text"`
	IsHTML    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
