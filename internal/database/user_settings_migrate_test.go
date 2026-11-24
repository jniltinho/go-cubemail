package database

import (
	"path/filepath"
	"testing"

	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
)

// legacyUserSettings is model.UserSettings as it was before the theme column.
type legacyUserSettings struct {
	UserID         uint   `gorm:"primaryKey"`
	RowsPerPage    int    `gorm:"default:50"`
	Timezone       string `gorm:"default:'UTC'"`
	ComposeHTML    bool   `gorm:"default:true"`
	DateFormat     string
	SignaturePos   string `gorm:"default:'below'"`
	OOFEnabled     bool   `gorm:"default:false"`
	OOFMessage     string `gorm:"type:text"`
	OOFMessageHTML string `gorm:"type:text"`
}

func (legacyUserSettings) TableName() string { return "user_settings" }

// AutoMigrate must add the theme column to an existing SQLite table, and rows
// written before the column existed must read as "follow the server default".
func TestAutoMigrateAddsThemeToExistingUserSettings(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.Driver = "sqlite"
	cfg.Database.DSN = filepath.Join(t.TempDir(), "app.db")
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}

	if err := db.AutoMigrate(&legacyUserSettings{}); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if err := db.Create(&legacyUserSettings{UserID: 7, Timezone: "Europe/Lisbon"}).Error; err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	if db.Migrator().HasColumn(&model.UserSettings{}, "theme") {
		t.Fatal("legacy table already has a theme column")
	}

	if err := db.AutoMigrate(&model.UserSettings{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !db.Migrator().HasColumn(&model.UserSettings{}, "theme") {
		t.Fatal("AutoMigrate did not add the theme column")
	}

	var s model.UserSettings
	if err := db.First(&s, "user_id = ?", 7).Error; err != nil {
		t.Fatalf("read migrated row: %v", err)
	}
	if s.Theme != "" || s.Timezone != "Europe/Lisbon" {
		t.Errorf("migrated row = theme %q, timezone %q; want empty theme, Europe/Lisbon", s.Theme, s.Timezone)
	}
}
