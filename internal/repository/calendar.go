package repository

import (
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// CalendarRepo provides CRUD operations for the Calendar model.
type CalendarRepo struct {
	db *gorm.DB
}

// NewCalendarRepo creates a CalendarRepo backed by the given database connection.
func NewCalendarRepo(db *gorm.DB) *CalendarRepo {
	return &CalendarRepo{db: db}
}

// List returns all calendars for the given user ordered by sort order and name.
func (r *CalendarRepo) List(userID uint) ([]model.Calendar, error) {
	var calendars []model.Calendar
	err := r.db.Where("user_id = ?", userID).
		Order("sort_order, name").
		Find(&calendars).Error
	return calendars, err
}

// Get retrieves a single calendar by ID scoped to the given user.
func (r *CalendarRepo) Get(userID, id uint) (*model.Calendar, error) {
	var cal model.Calendar
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&cal).Error
	return &cal, err
}

// Create inserts a new calendar record, assigning the stable DAV URI that
// client bookmarks and sync state are keyed on.
func (r *CalendarRepo) Create(cal *model.Calendar) error {
	if cal.SyncToken == 0 {
		cal.SyncToken = 1
	}
	if cal.URI == "" {
		cal.URI = freeCollectionURI(r.db, &model.Calendar{}, cal.UserID, Slugify(cal.Name, "calendar"))
	}
	return r.db.Create(cal).Error
}

// GetByURI resolves a calendar from its DAV path segment.
func (r *CalendarRepo) GetByURI(userID uint, uri string) (*model.Calendar, error) {
	var cal model.Calendar
	err := r.db.Where("user_id = ? AND uri = ?", userID, uri).First(&cal).Error
	return &cal, err
}

// Update persists changes to an existing calendar.
func (r *CalendarRepo) Update(cal *model.Calendar) error {
	return r.db.Save(cal).Error
}

// Delete removes a calendar and all its events for the given user, together
// with the changelog rows that referenced it.
func (r *CalendarRepo) Delete(userID, id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("calendar_id = ? AND user_id = ?", id, userID).
			Delete(&model.Event{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Calendar{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		// The collection itself is gone, so its per-object changelog is dead
		// weight; clients notice the removal from the calendar-home-set listing.
		return tx.Where("collection_kind = ? AND collection_id = ?",
			model.CollectionCalendar, id).Delete(&model.DAVChange{}).Error
	})
}

// EnsureDefault creates a default "Personal" calendar when the user has none.
func (r *CalendarRepo) EnsureDefault(userID uint) (*model.Calendar, error) {
	var cal model.Calendar
	err := r.db.Where("user_id = ? AND is_default = ?", userID, true).First(&cal).Error
	if err == nil {
		return &cal, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	cal = model.Calendar{
		UserID:            userID,
		Name:              "Personal",
		Color:             "#3788d8",
		IsDefault:         true,
		IsActive:          true,
		IncludeInFreeBusy: true,
		URI:               "default",
		SyncToken:         1,
	}
	if err := r.Create(&cal); err != nil {
		return nil, err
	}
	return &cal, nil
}

// BackfillURIs assigns a stable DAV URI and initial sync token to calendars
// created before CalDAV collections were persisted. Safe to call repeatedly.
func (r *CalendarRepo) BackfillURIs(userID uint) error {
	var cals []model.Calendar
	if err := r.db.Where("user_id = ? AND (uri IS NULL OR uri = '')", userID).
		Find(&cals).Error; err != nil {
		return err
	}
	for i := range cals {
		uri := "default"
		if !cals[i].IsDefault {
			uri = freeCollectionURI(r.db, &model.Calendar{}, userID, Slugify(cals[i].Name, "calendar"))
		}
		updates := map[string]any{"uri": uri}
		if cals[i].SyncToken == 0 {
			updates["sync_token"] = 1
		}
		if err := r.db.Model(&model.Calendar{}).Where("id = ?", cals[i].ID).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

// SetActive updates the is_active flag for the given calendar IDs.
func (r *CalendarRepo) SetActive(userID uint, ids []uint, active bool) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&model.Calendar{}).
		Where("user_id = ? AND id IN ?", userID, ids).
		Update("is_active", active).Error
}

// ListActive returns active calendars for the given user.
func (r *CalendarRepo) ListActive(userID uint) ([]model.Calendar, error) {
	var calendars []model.Calendar
	err := r.db.Where("user_id = ? AND is_active = ?", userID, true).
		Order("sort_order, name").
		Find(&calendars).Error
	return calendars, err
}
