package model

type UserSettings struct {
	UserID      uint   `gorm:"primaryKey"`
	RowsPerPage int    `gorm:"default:50"`
	Timezone    string `gorm:"default:'UTC'"`
	ComposeHTML bool   `gorm:"default:true"`
	DateFormat  string
	SignaturePos string `gorm:"default:'below'"` // below | above
}
