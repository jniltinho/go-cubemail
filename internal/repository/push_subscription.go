package repository

import (
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// PushSubscriptionRepo manages Web Push subscriptions.
type PushSubscriptionRepo struct {
	db *gorm.DB
}

// NewPushSubscriptionRepo creates a PushSubscriptionRepo.
func NewPushSubscriptionRepo(db *gorm.DB) *PushSubscriptionRepo {
	return &PushSubscriptionRepo{db: db}
}

// Upsert creates or replaces a subscription for a given endpoint.
// Each browser generates a unique endpoint, so we use it as a natural key.
func (r *PushSubscriptionRepo) Upsert(sub *model.PushSubscription) error {
	var existing model.PushSubscription
	err := r.db.Where("user_id = ? AND endpoint = ?", sub.UserID, sub.Endpoint).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(sub).Error
	}
	if err != nil {
		return err
	}
	existing.P256DH = sub.P256DH
	existing.Auth = sub.Auth
	return r.db.Save(&existing).Error
}

// Delete removes a subscription by endpoint for a user.
func (r *PushSubscriptionRepo) Delete(userID uint, endpoint string) error {
	return r.db.Where("user_id = ? AND endpoint = ?", userID, endpoint).
		Delete(&model.PushSubscription{}).Error
}

// ListForUser returns all active push subscriptions for a user.
func (r *PushSubscriptionRepo) ListForUser(userID uint) ([]model.PushSubscription, error) {
	var subs []model.PushSubscription
	err := r.db.Where("user_id = ?", userID).Find(&subs).Error
	return subs, err
}
