// Package repository provides database access objects for go-cubemail models.
// All repositories enforce user-ID scoping so queries never cross user boundaries.
package repository

import (
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// ContactRepo provides CRUD operations for the Contact model.
type ContactRepo struct {
	db *gorm.DB
}

// NewContactRepo creates a ContactRepo backed by the given database connection.
func NewContactRepo(db *gorm.DB) *ContactRepo {
	return &ContactRepo{db: db}
}

// List returns all contacts for the given user, ordered by first and last name.
func (r *ContactRepo) List(userID uint) ([]model.Contact, error) {
	var contacts []model.Contact
	err := r.db.Where("user_id = ?", userID).Order("first_name, last_name").Find(&contacts).Error
	return contacts, err
}

// Search returns contacts whose first name, last name, or email contain the query string.
func (r *ContactRepo) Search(userID uint, q string) ([]model.Contact, error) {
	var contacts []model.Contact
	like := "%" + q + "%"
	err := r.db.Where("user_id = ? AND (first_name LIKE ? OR last_name LIKE ? OR email LIKE ?)",
		userID, like, like, like).Find(&contacts).Error
	return contacts, err
}

// Get retrieves a single contact by ID, scoped to the given user.
func (r *ContactRepo) Get(userID, id uint) (*model.Contact, error) {
	var c model.Contact
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&c).Error
	return &c, err
}

// Create inserts a new contact record.
func (r *ContactRepo) Create(c *model.Contact) error {
	return r.db.Create(c).Error
}

// Update persists changes to an existing contact using a full SAVE.
func (r *ContactRepo) Update(c *model.Contact) error {
	return r.db.Save(c).Error
}

// Delete removes a contact by ID, scoped to the given user.
func (r *ContactRepo) Delete(userID, id uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Contact{}).Error
}
