package repository

import (
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

type ContactRepo struct {
	db *gorm.DB
}

func NewContactRepo(db *gorm.DB) *ContactRepo {
	return &ContactRepo{db: db}
}

func (r *ContactRepo) List(userID uint) ([]model.Contact, error) {
	var contacts []model.Contact
	err := r.db.Where("user_id = ?", userID).Order("last_name, first_name").Find(&contacts).Error
	return contacts, err
}

func (r *ContactRepo) Search(userID uint, q string) ([]model.Contact, error) {
	var contacts []model.Contact
	like := "%" + q + "%"
	err := r.db.Where("user_id = ? AND (first_name LIKE ? OR last_name LIKE ? OR email LIKE ?)",
		userID, like, like, like).Find(&contacts).Error
	return contacts, err
}

func (r *ContactRepo) Get(userID, id uint) (*model.Contact, error) {
	var c model.Contact
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&c).Error
	return &c, err
}

func (r *ContactRepo) Create(c *model.Contact) error {
	return r.db.Create(c).Error
}

func (r *ContactRepo) Update(c *model.Contact) error {
	return r.db.Save(c).Error
}

func (r *ContactRepo) Delete(userID, id uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Contact{}).Error
}
