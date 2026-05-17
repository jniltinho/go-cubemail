package repository

import (
	"go-cubemail/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingsRepo struct {
	db *gorm.DB
}

func NewSettingsRepo(db *gorm.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Get(userID uint) (*model.UserSettings, error) {
	var s model.UserSettings
	err := r.db.FirstOrCreate(&s, model.UserSettings{UserID: userID}).Error
	return &s, err
}

func (r *SettingsRepo) Save(s *model.UserSettings) error {
	return r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(s).Error
}
