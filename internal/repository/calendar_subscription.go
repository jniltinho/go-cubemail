package repository

import (
	"time"

	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// CalendarSubscriptionRepo manages remote .ics subscriptions.
type CalendarSubscriptionRepo struct {
	db *gorm.DB
}

// NewCalendarSubscriptionRepo creates a CalendarSubscriptionRepo.
func NewCalendarSubscriptionRepo(db *gorm.DB) *CalendarSubscriptionRepo {
	return &CalendarSubscriptionRepo{db: db}
}

// List returns all subscriptions for a user.
func (r *CalendarSubscriptionRepo) List(userID uint) ([]model.CalendarSubscription, error) {
	var subs []model.CalendarSubscription
	err := r.db.Where("user_id = ?", userID).Order("created_at").Find(&subs).Error
	return subs, err
}

// Create inserts a new subscription.
func (r *CalendarSubscriptionRepo) Create(sub *model.CalendarSubscription) error {
	return r.db.Create(sub).Error
}

// Delete removes a subscription by ID scoped to the user.
func (r *CalendarSubscriptionRepo) Delete(userID, id uint) error {
	res := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.CalendarSubscription{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListDue returns all subscriptions whose next sync time has arrived.
func (r *CalendarSubscriptionRepo) ListDue() ([]model.CalendarSubscription, error) {
	var subs []model.CalendarSubscription
	err := r.db.Where(
		"last_synced_at IS NULL OR last_synced_at < ?",
		time.Now().Add(-time.Minute),
	).Find(&subs).Error
	return subs, err
}

// UpdateSyncResult persists the sync outcome (last_synced_at + optional error).
func (r *CalendarSubscriptionRepo) UpdateSyncResult(id uint, syncErr string) error {
	now := time.Now()
	return r.db.Model(&model.CalendarSubscription{}).Where("id = ?", id).Updates(map[string]any{
		"last_synced_at": &now,
		"last_error":     syncErr,
	}).Error
}
