package repository

import (
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

type IdentityRepo struct {
	db *gorm.DB
}

func NewIdentityRepo(db *gorm.DB) *IdentityRepo {
	return &IdentityRepo{db: db}
}

func (r *IdentityRepo) List(userID uint) ([]model.Identity, error) {
	var ids []model.Identity
	err := r.db.Where("user_id = ?", userID).Find(&ids).Error
	return ids, err
}

func (r *IdentityRepo) Default(userID uint) (*model.Identity, error) {
	var id model.Identity
	err := r.db.Where("user_id = ? AND is_default = ?", userID, true).First(&id).Error
	return &id, err
}

func (r *IdentityRepo) Create(id *model.Identity) error {
	return r.db.Create(id).Error
}

func (r *IdentityRepo) Update(id *model.Identity) error {
	return r.db.Save(id).Error
}

func (r *IdentityRepo) Delete(userID, id uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Identity{}).Error
}
