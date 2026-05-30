package model

// UserSettings stores per-user UI preferences.
// The record is keyed by UserID and created on first access via FirstOrCreate.
type UserSettings struct {
	UserID         uint   `gorm:"primaryKey"`
	RowsPerPage    int    `gorm:"default:50"`
	Timezone       string `gorm:"default:'UTC'"`
	ComposeHTML    bool   `gorm:"default:true"`
	DateFormat     string
	SignaturePos   string `gorm:"default:'below'"` // "below" | "above"
	OOFEnabled     bool   `gorm:"default:false"`
	OOFMessage     string `gorm:"type:text"`
	OOFMessageHTML string `gorm:"type:text"`
}
