package repository

import (
	"go-cubemail/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SettingsRepo provides read and write access to per-user settings.
// Note: not yet wired to any handler — defined for future settings management support.
type SettingsRepo struct {
	db *gorm.DB
}

// NewSettingsRepo creates a SettingsRepo backed by the given database connection.
func NewSettingsRepo(db *gorm.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

// Get returns the settings for the given user, creating a default record if none exists.
func (r *SettingsRepo) Get(userID uint) (*model.UserSettings, error) {
	var s model.UserSettings
	err := r.db.FirstOrCreate(&s, model.UserSettings{UserID: userID}).Error
	return &s, err
}

// Save upserts the settings record using an ON CONFLICT UPDATE ALL strategy.
func (r *SettingsRepo) Save(s *model.UserSettings) error {
	return r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(s).Error
}
