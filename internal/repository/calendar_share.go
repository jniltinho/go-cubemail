package repository

import (
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// CalendarShareRepo manages calendar share grants.
type CalendarShareRepo struct {
	db *gorm.DB
}

// NewCalendarShareRepo creates a CalendarShareRepo.
func NewCalendarShareRepo(db *gorm.DB) *CalendarShareRepo {
	return &CalendarShareRepo{db: db}
}

// ListSharedBy returns all shares granted by the given owner for their calendars.
func (r *CalendarShareRepo) ListSharedBy(ownerID uint) ([]model.CalendarShare, error) {
	var shares []model.CalendarShare
	err := r.db.Where("owner_id = ?", ownerID).Find(&shares).Error
	return shares, err
}

// ListSharedWith returns calendars shared with the given user.
func (r *CalendarShareRepo) ListSharedWith(userID uint) ([]model.CalendarShare, error) {
	var shares []model.CalendarShare
	err := r.db.Where("shared_with_id = ?", userID).Find(&shares).Error
	return shares, err
}

// Create creates a share grant.
func (r *CalendarShareRepo) Create(share *model.CalendarShare) error {
	return r.db.Create(share).Error
}

// Update persists can_write changes.
func (r *CalendarShareRepo) Update(share *model.CalendarShare) error {
	return r.db.Save(share).Error
}

// Delete removes a share by share ID, ensuring it belongs to ownerID.
func (r *CalendarShareRepo) Delete(ownerID, shareID uint) error {
	res := r.db.Where("id = ? AND owner_id = ?", shareID, ownerID).Delete(&model.CalendarShare{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetByCalendarAndUser returns the share for a specific calendar and grantee.
func (r *CalendarShareRepo) GetByCalendarAndUser(calendarID, sharedWithID uint) (*model.CalendarShare, error) {
	var share model.CalendarShare
	err := r.db.Where("calendar_id = ? AND shared_with_id = ?", calendarID, sharedWithID).First(&share).Error
	return &share, err
}
