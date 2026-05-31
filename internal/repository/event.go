package repository

import (
	"time"

	calpkg "go-cubemail/internal/calendar"
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// EventRepo provides CRUD operations for the Event model.
type EventRepo struct {
	db *gorm.DB
}

// NewEventRepo creates an EventRepo backed by the given database connection.
func NewEventRepo(db *gorm.DB) *EventRepo {
	return &EventRepo{db: db}
}

// ListByRange returns events overlapping the given time range for a user.
// Non-recurring events are returned directly; recurring masters are expanded using their RRULE.
// Exception rows (RecurrenceID != nil) override their parent occurrence for the matching start time.
func (r *EventRepo) ListByRange(userID uint, start, end time.Time, calendarIDs []uint) ([]model.Event, error) {
	// Fetch non-recurring events that overlap the window.
	q := r.db.Preload("Attendees").Preload("Calendar").
		Where("user_id = ? AND is_task = ?", userID, false).
		Where("(r_rule = '' OR r_rule IS NULL) AND recurrence_id IS NULL").
		Where("start_at < ? AND end_at > ?", end, start)
	if len(calendarIDs) > 0 {
		q = q.Where("calendar_id IN ?", calendarIDs)
	}
	var plain []model.Event
	if err := q.Order("start_at").Find(&plain).Error; err != nil {
		return nil, err
	}

	// Fetch recurring master events (rrule non-empty, no recurrence_id).
	qr := r.db.Preload("Attendees").Preload("Calendar").
		Where("user_id = ? AND is_task = ?", userID, false).
		Where("r_rule != '' AND r_rule IS NOT NULL AND recurrence_id IS NULL").
		Where("start_at < ?", end)
	if len(calendarIDs) > 0 {
		qr = qr.Where("calendar_id IN ?", calendarIDs)
	}
	var masters []model.Event
	if err := qr.Find(&masters).Error; err != nil {
		return nil, err
	}

	// Build exception index: uid → map[startEpoch]exceptionEvent
	exceptions := make(map[string]map[int64]model.Event)
	for _, m := range masters {
		var exList []model.Event
		r.db.Preload("Attendees").Preload("Calendar").
			Where("user_id = ? AND uid = ? AND recurrence_id IS NOT NULL", userID, m.UID).
			Find(&exList)
		idx := make(map[int64]model.Event, len(exList))
		for _, ex := range exList {
			if ex.RecurrenceID != nil {
				idx[ex.RecurrenceID.Unix()] = ex
			}
		}
		exceptions[m.UID] = idx
	}

	// Expand each master into occurrences within the window.
	var expanded []model.Event
	for _, master := range masters {
		occs := calpkg.ExpandRecurring(&master, start, end)
		for _, occ := range occs {
			// If an exception exists for this occurrence start, use it instead.
			if ex, ok := exceptions[master.UID][occ.StartAt.Unix()]; ok {
				expanded = append(expanded, ex)
				continue
			}
			clone := master
			clone.StartAt = occ.StartAt
			clone.EndAt = occ.EndAt
			expanded = append(expanded, clone)
		}
	}

	all := append(plain, expanded...)
	// Sort by start_at ascending.
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].StartAt.Before(all[j-1].StartAt); j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
	return all, nil
}

// ListByCalendar returns all events in a calendar for export.
func (r *EventRepo) ListByCalendar(userID, calendarID uint) ([]model.Event, error) {
	var events []model.Event
	err := r.db.Preload("Attendees").
		Where("user_id = ? AND calendar_id = ?", userID, calendarID).
		Order("start_at").
		Find(&events).Error
	return events, err
}

// Get retrieves a single event with attendees, scoped to the given user.
func (r *EventRepo) Get(userID, id uint) (*model.Event, error) {
	var event model.Event
	err := r.db.Preload("Attendees").
		Preload("Calendar").
		Where("id = ? AND user_id = ?", id, userID).
		First(&event).Error
	return &event, err
}

// GetByUID retrieves an event by iCalendar UID for the given user.
func (r *EventRepo) GetByUID(userID uint, uid string) (*model.Event, error) {
	var event model.Event
	err := r.db.Preload("Attendees").
		Where("user_id = ? AND uid = ?", userID, uid).
		First(&event).Error
	return &event, err
}

// Create inserts a new event and its attendees in a transaction.
func (r *EventRepo) Create(event *model.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		attendees := event.Attendees
		event.Attendees = nil
		if err := tx.Create(event).Error; err != nil {
			return err
		}
		for i := range attendees {
			attendees[i].EventID = event.ID
			if attendees[i].PartStat == "" {
				attendees[i].PartStat = "NEEDS-ACTION"
			}
			if attendees[i].Role == "" {
				attendees[i].Role = "REQ-PARTICIPANT"
			}
			if err := tx.Create(&attendees[i]).Error; err != nil {
				return err
			}
		}
		event.Attendees = attendees
		return nil
	})
}

// Update replaces an event and its attendees in a transaction.
func (r *EventRepo) Update(event *model.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(event).Error; err != nil {
			return err
		}
		if err := tx.Where("event_id = ?", event.ID).Delete(&model.EventAttendee{}).Error; err != nil {
			return err
		}
		for i := range event.Attendees {
			event.Attendees[i].EventID = event.ID
			event.Attendees[i].ID = 0
			if event.Attendees[i].PartStat == "" {
				event.Attendees[i].PartStat = "NEEDS-ACTION"
			}
			if event.Attendees[i].Role == "" {
				event.Attendees[i].Role = "REQ-PARTICIPANT"
			}
			if err := tx.Create(&event.Attendees[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListTasks returns all VTODO tasks (is_task=true) for a user, ordered by end_at (due date).
func (r *EventRepo) ListTasks(userID uint) ([]model.Event, error) {
	var tasks []model.Event
	err := r.db.Preload("Attendees").
		Where("user_id = ? AND is_task = ?", userID, true).
		Order("end_at").
		Find(&tasks).Error
	return tasks, err
}

// Delete removes an event by ID scoped to the given user.
func (r *EventRepo) Delete(userID, id uint) error {
	res := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Event{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
