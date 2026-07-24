package repository

import (
	"time"

	calpkg "go-cubemail/internal/calendar"
	"go-cubemail/internal/dav"
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

// ListByCalendarRange returns the stored rows of a calendar that may overlap a
// time window, without expanding recurrences.
//
// CalDAV clients expand RRULEs themselves, so a calendar-query must return the
// master object once — expanding here would emit the same href N times in a
// single multistatus. Recurring masters starting before the window are always
// included because their occurrences can reach into it; the extra rows are a
// harmless superset, and the client applies the exact filter.
func (r *EventRepo) ListByCalendarRange(userID, calendarID uint, start, end time.Time) ([]model.Event, error) {
	var events []model.Event
	err := r.db.Preload("Attendees").
		Where("user_id = ? AND calendar_id = ?", userID, calendarID).
		Where("(r_rule != '' AND r_rule IS NOT NULL AND start_at < ?) OR (start_at < ? AND end_at > ?)",
			end, end, start).
		Order("start_at").
		Find(&events).Error
	return events, err
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

// GetByResourceURI retrieves an event by its DAV resource name within a
// calendar. This is the identity CalDAV clients address, and it is deliberately
// separate from the iCalendar UID.
func (r *EventRepo) GetByResourceURI(userID, calendarID uint, uri string) (*model.Event, error) {
	var event model.Event
	err := r.db.Preload("Attendees").
		Where("user_id = ? AND calendar_id = ? AND resource_uri = ?", userID, calendarID, uri).
		First(&event).Error
	return &event, err
}

// ListSince returns the events of a calendar whose last write is newer than the
// given revision, used to answer sync-collection REPORTs.
func (r *EventRepo) ListSince(userID, calendarID uint, since uint64) ([]model.Event, error) {
	var events []model.Event
	err := r.db.Preload("Attendees").
		Where("user_id = ? AND calendar_id = ? AND sync_revision > ?", userID, calendarID, since).
		Order("sync_revision").
		Find(&events).Error
	return events, err
}

// prepareDAVFields fills the resource identity of an event before it is written:
// a resource name, the iCalendar blob and the ETag derived from it.
func prepareDAVFields(event *model.Event) {
	if event.UID == "" {
		event.UID = calpkg.NewUID("")
	}
	if event.ResourceURI == "" {
		event.ResourceURI = dav.NewResourceURI(".ics")
	}
	if event.ICalContent == "" {
		event.ICalContent = calpkg.BuildICalContent(event, event.Attendees)
	}
	event.ETag = dav.ComputeETag([]byte(event.ICalContent))
}

// Create inserts a new event and its attendees in a transaction, recording the
// change so CalDAV clients see it on their next sync.
func (r *EventRepo) Create(event *model.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		attendees := event.Attendees
		prepareDAVFields(event)
		event.Attendees = nil
		rev, err := dav.RecordIfExists(tx, model.CollectionCalendar,
			event.CalendarID, event.ResourceURI, false)
		if err != nil {
			return err
		}
		event.SyncRevision = rev
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

// Update replaces an event and its attendees in a transaction, advancing the
// collection revision so the change reaches DAV clients.
func (r *EventRepo) Update(event *model.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		prepareDAVFields(event)
		rev, err := dav.RecordIfExists(tx, model.CollectionCalendar,
			event.CalendarID, event.ResourceURI, false)
		if err != nil {
			return err
		}
		if rev > 0 {
			event.SyncRevision = rev
		}
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

// Delete removes an event by ID scoped to the given user and records a
// tombstone in the changelog.
//
// The row is loaded first because the tombstone needs the resource name: a bulk
// DELETE would leave clients unable to learn that the object is gone, which is
// exactly the failure the changelog exists to prevent.
func (r *EventRepo) Delete(userID, id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var event model.Event
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&event).Error; err != nil {
			return err
		}
		if err := tx.Where("event_id = ?", event.ID).
			Delete(&model.EventAttendee{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Event{}, event.ID).Error; err != nil {
			return err
		}
		_, err := dav.RecordIfExists(tx, model.CollectionCalendar,
			event.CalendarID, event.ResourceURI, true)
		return err
	})
}
