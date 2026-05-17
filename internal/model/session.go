package model

import "time"

// Session representa uma sessão de usuário persistida no banco de dados.
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
