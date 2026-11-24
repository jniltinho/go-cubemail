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
	// Theme is a registered theme id, or empty to follow the server default.
	// It must not carry a `default:` tag: the settings upsert is a Create, and
	// GORM replaces a zero value with the tagged default, so it could never be cleared.
	Theme string
}
