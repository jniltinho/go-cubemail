package repository

import (
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// IdentityRepo provides CRUD operations for the Identity model.
// Note: not yet wired to any handler — defined for future identity management support.
type IdentityRepo struct {
	db *gorm.DB
}

// NewIdentityRepo creates an IdentityRepo backed by the given database connection.
func NewIdentityRepo(db *gorm.DB) *IdentityRepo {
	return &IdentityRepo{db: db}
}

// List returns all identities belonging to the given user.
func (r *IdentityRepo) List(userID uint) ([]model.Identity, error) {
	var ids []model.Identity
	err := r.db.Where("user_id = ?", userID).Find(&ids).Error
	return ids, err
}

// Default returns the identity marked as default for the given user.
func (r *IdentityRepo) Default(userID uint) (*model.Identity, error) {
	var id model.Identity
	err := r.db.Where("user_id = ? AND is_default = ?", userID, true).First(&id).Error
	return &id, err
}

// Create inserts a new identity record.
func (r *IdentityRepo) Create(id *model.Identity) error {
	return r.db.Create(id).Error
}

// Update persists changes to an existing identity.
func (r *IdentityRepo) Update(id *model.Identity) error {
	return r.db.Save(id).Error
}

// Delete removes an identity by ID, scoped to the given user.
func (r *IdentityRepo) Delete(userID, id uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Identity{}).Error
}
