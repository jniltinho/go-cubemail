package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	calpkg "go-cubemail/internal/calendar"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
)

// CalendarSyncEngine syncs calendar events between SQL (EventRepo) and EAS clients.
//
// Supported collection IDs: vevent/* (currently maps to the user's default calendar via EnsureDefault).
// Server IDs are decimal event primary keys. Change detection uses event.UpdatedAt stored in ItemSyncCache.
type CalendarSyncEngine struct {
	store     *state.Store           // Sync keys and per-collection item cache.
	calRepo   *repository.CalendarRepo // Resolves default calendar for vevent/personal.
	eventRepo *repository.EventRepo  // CRUD for model.Event rows.
}

// NewCalendarSyncEngine creates a CalendarSyncEngine backed by the given store and repositories.
func NewCalendarSyncEngine(store *state.Store, calRepo *repository.CalendarRepo, eventRepo *repository.EventRepo) *CalendarSyncEngine {
	return &CalendarSyncEngine{store: store, calRepo: calRepo, eventRepo: eventRepo}
}

// SyncCollection processes one vevent/* collection in an MS-ASCMD Sync request.
//
// Flow: validate sync key → apply client Commands (Add/Change/Delete) → build server changes
// when GetChanges is set → bump sync key and persist ItemSyncCache.
// Returns SyncStatusInvalidSyncKey (3) when the client key is stale.
func (c *CalendarSyncEngine) SyncCollection(ctx *Context, dev *state.EasDevice, col eas.SyncCollection) (eas.SyncCollection, error) {
	if !parseVEventCollectionID(col.CollectionID) {
		return eas.SyncCollection{
			SyncKey:      col.SyncKey,
			CollectionID: col.CollectionID,
			Status:       eas.SyncStatusProtocolError,
		}, nil
	}

	cal, err := c.calRepo.EnsureDefault(ctx.UserID)
	if err != nil {
		return eas.SyncCollection{}, err
	}

	fst, err := c.store.GetFolderState(dev.ID, col.CollectionID)
	if err != nil {
		return eas.SyncCollection{}, err
	}
	if col.SyncKey != "0" && col.SyncKey != fst.SyncKey {
		return eas.SyncCollection{
			SyncKey:      fst.SyncKey,
			CollectionID: col.CollectionID,
			Status:       eas.SyncStatusInvalidSyncKey,
		}, nil
	}

	cache := state.LoadItemSyncCache(fst.SyncCache)
	resp := eas.SyncCollection{
		CollectionID: col.CollectionID,
		Class:        "Calendar",
		Status:       eas.SyncStatusSuccess,
	}

	var responses *eas.SyncCommands
	if col.Commands != nil {
		responses, err = c.applyClientCommands(ctx, col.Commands, cal.ID, &cache)
		if err != nil {
			return eas.SyncCollection{}, err
		}
	}

	window := syncWindow(col.WindowSize)
	if col.GetChanges != 0 || col.SyncKey == "0" {
		adds, changes, deletes, more, err := c.buildServerChanges(ctx.UserID, cal.ID, cache, window)
		if err != nil {
			return eas.SyncCollection{}, err
		}
		if len(adds)+len(changes)+len(deletes) > 0 {
			resp.Commands = &eas.SyncCommands{Add: adds, Change: changes, Delete: deletes}
		}
		if more {
			resp.MoreAvailable = 1
		}
	}
	if responses != nil && (len(responses.Add)+len(responses.Change)+len(responses.Delete) > 0) {
		resp.Responses = responses
	}

	newKey, err := c.store.NextCollectionSyncKey(dev.ID, col.CollectionID)
	if err != nil {
		return eas.SyncCollection{}, err
	}
	fst.SyncKey = newKey
	fst.SyncCache = state.EncodeItemSyncCache(cache)
	if err := c.store.SaveFolderState(fst); err != nil {
		return eas.SyncCollection{}, err
	}
	resp.SyncKey = newKey
	return resp, nil
}

// EstimateCount returns the number of events in the user's default calendar.
// Used by GetItemEstimate for vevent/* collection IDs.
func (c *CalendarSyncEngine) EstimateCount(ctx *Context, collectionID string) (int32, error) {
	if !parseVEventCollectionID(collectionID) {
		return 0, fmt.Errorf("invalid calendar collection")
	}
	cal, err := c.calRepo.EnsureDefault(ctx.UserID)
	if err != nil {
		return 0, err
	}
	events, err := c.eventRepo.ListByCalendar(ctx.UserID, cal.ID)
	if err != nil {
		return 0, err
	}
	return int32(len(events)), nil
}

// applyClientCommands applies client-originated Sync Add/Change/Delete commands to SQL.
// Returns Responses with ClientId→ServerId mappings for successful Add operations.
func (c *CalendarSyncEngine) applyClientCommands(ctx *Context, cmds *eas.SyncCommands, calendarID uint, cache *state.ItemSyncCache) (*eas.SyncCommands, error) {
	var responses eas.SyncCommands

	for _, add := range cmds.Add {
		appt, err := add.Appointment()
		if err != nil {
			continue
		}
		event := appointmentToEvent(appt, ctx.UserID, calendarID)
		event.ICalContent = calpkg.BuildICalContent(event, event.Attendees)
		if err := c.eventRepo.Create(event); err != nil {
			return nil, err
		}
		sid := serverIDForUint(event.ID)
		cache.Items[sid] = state.ItemSyncItem{UpdatedAt: event.UpdatedAt.Unix()}
		responses.Add = append(responses.Add, eas.SyncAdd{
			ClientID: add.ClientID,
			ServerID: sid,
		})
	}

	for _, chg := range cmds.Change {
		id, ok := parseServerID(chg.ServerID)
		if !ok {
			continue
		}
		appt, err := chg.Appointment()
		if err != nil {
			continue
		}
		event, err := c.eventRepo.Get(ctx.UserID, id)
		if err != nil {
			continue
		}
		applyAppointmentToEvent(appt, event)
		event.ICalContent = calpkg.BuildICalContent(event, event.Attendees)
		if err := c.eventRepo.Update(event); err != nil {
			return nil, err
		}
		cache.Items[chg.ServerID] = state.ItemSyncItem{UpdatedAt: event.UpdatedAt.Unix()}
	}

	for _, del := range cmds.Delete {
		id, ok := parseServerID(del.ServerID)
		if !ok {
			continue
		}
		if err := c.eventRepo.Delete(ctx.UserID, id); err != nil {
			continue
		}
		delete(cache.Items, del.ServerID)
	}

	if len(responses.Add) == 0 {
		return nil, nil
	}
	return &responses, nil
}

// buildServerChanges compares SQL events with ItemSyncCache and builds Add/Change/Delete WBXML commands.
// Respects window size; sets more=true when additional pending changes remain.
func (c *CalendarSyncEngine) buildServerChanges(userID, calendarID uint, cache state.ItemSyncCache, window int) ([]eas.SyncAdd, []eas.SyncChange, []eas.SyncDelete, bool, error) {
	events, err := c.eventRepo.ListByCalendar(userID, calendarID)
	if err != nil {
		return nil, nil, nil, false, err
	}

	current := make(map[string]state.ItemSyncItem, len(events))
	eventByID := make(map[string]model.Event, len(events))
	for _, ev := range events {
		sid := serverIDForUint(ev.ID)
		current[sid] = state.ItemSyncItem{UpdatedAt: ev.UpdatedAt.Unix()}
		eventByID[sid] = ev
	}

	type pending struct {
		kind     string
		serverID string
	}
	var queue []pending

	for sid, item := range current {
		prev, ok := cache.Items[sid]
		if !ok {
			queue = append(queue, pending{kind: "add", serverID: sid})
			continue
		}
		if prev.UpdatedAt != item.UpdatedAt {
			queue = append(queue, pending{kind: "change", serverID: sid})
		}
	}
	for sid := range cache.Items {
		if _, ok := current[sid]; !ok {
			queue = append(queue, pending{kind: "delete", serverID: sid})
		}
	}

	var adds []eas.SyncAdd
	var changes []eas.SyncChange
	var deletes []eas.SyncDelete
	remaining := window

	for _, p := range queue {
		if remaining <= 0 {
			break
		}
		switch p.kind {
		case "add":
			ev := eventByID[p.serverID]
			add, err := c.buildSyncAdd(ev)
			if err != nil {
				return nil, nil, nil, false, err
			}
			adds = append(adds, add)
			cache.Items[p.serverID] = current[p.serverID]
		case "change":
			ev := eventByID[p.serverID]
			chg, err := c.buildSyncChange(ev)
			if err != nil {
				return nil, nil, nil, false, err
			}
			changes = append(changes, chg)
			cache.Items[p.serverID] = current[p.serverID]
		case "delete":
			deletes = append(deletes, eas.SyncDelete{ServerID: p.serverID})
			delete(cache.Items, p.serverID)
		}
		remaining--
	}

	more := len(queue) > len(adds)+len(changes)+len(deletes)
	return adds, changes, deletes, more, nil
}

// buildSyncAdd wraps a model.Event as an EAS Sync Add with Calendar ApplicationData.
func (c *CalendarSyncEngine) buildSyncAdd(event model.Event) (eas.SyncAdd, error) {
	appt := eventToAppointment(&event)
	body, err := marshalApplicationDataBody(&appt)
	if err != nil {
		return eas.SyncAdd{}, err
	}
	return eas.SyncAdd{
		ServerID: serverIDForUint(event.ID),
		ApplicationData: &wbxml.RawElement{
			Page:  wbxml.PageAirSync,
			Bytes: body,
		},
	}, nil
}

// buildSyncChange wraps an updated model.Event as an EAS Sync Change command.
func (c *CalendarSyncEngine) buildSyncChange(event model.Event) (eas.SyncChange, error) {
	appt := eventToAppointment(&event)
	body, err := marshalApplicationDataBody(&appt)
	if err != nil {
		return eas.SyncChange{}, err
	}
	return eas.SyncChange{
		ServerID: serverIDForUint(event.ID),
		ApplicationData: &wbxml.RawElement{
			Page:  wbxml.PageAirSync,
			Bytes: body,
		},
	}, nil
}

// eventToAppointment maps a model.Event (and attendees) to an eas.Appointment ApplicationData payload.
// Maps iCalendar PartStat values to MS-ASCAL AttendeeStatus integers.
func eventToAppointment(event *model.Event) eas.Appointment {
	appt := eas.Appointment{
		UID:            event.UID,
		Subject:        event.Summary,
		Location:       event.Location,
		StartTime:      easTime(event.StartAt, event.IsAllDay),
		EndTime:        easTime(event.EndAt, event.IsAllDay),
		OrganizerEmail: event.OrganizerEmail,
		OrganizerName:  event.OrganizerName,
		DtStamp:        easTime(event.UpdatedAt, false),
		BusyStatus:     eas.BusyStatusBusy,
		MeetingStatus:  eas.MeetingStatusNonMeeting,
		Sensitivity:    eas.SensitivityNormal,
	}
	if event.IsAllDay {
		appt.AllDayEvent = 1
	}
	if len(event.Attendees) > 0 {
		atts := make([]eas.Attendee, 0, len(event.Attendees))
		for _, a := range event.Attendees {
			atts = append(atts, eas.Attendee{
				Email:          a.Email,
				Name:           a.Name,
				AttendeeStatus: partStatToEas(a.PartStat),
				AttendeeType:   1,
			})
		}
		appt.Attendees = &eas.Attendees{Attendee: atts}
	}
	return appt
}

// appointmentToEvent converts a client eas.Appointment into a new model.Event for Create.
// Generates a UID via calendar.NewUID when the appointment carries no UID.
func appointmentToEvent(appt *eas.Appointment, userID, calendarID uint) *model.Event {
	start, _ := parseEasTime(appt.StartTime)
	end, _ := parseEasTime(appt.EndTime)
	if end.IsZero() && !start.IsZero() {
		end = start.Add(time.Hour)
	}
	uid := appt.UID
	if uid == "" {
		uid = calpkg.NewUID("")
	}
	event := &model.Event{
		UserID:         userID,
		CalendarID:     calendarID,
		UID:            uid,
		Summary:        appt.Subject,
		Location:       appt.Location,
		StartAt:        start,
		EndAt:          end,
		IsAllDay:       appt.AllDayEvent != 0,
		OrganizerEmail: appt.OrganizerEmail,
		OrganizerName:  appt.OrganizerName,
		Status:         "CONFIRMED",
	}
	if appt.Attendees != nil {
		for _, a := range appt.Attendees.Attendee {
			if a.Email == "" {
				continue
			}
			event.Attendees = append(event.Attendees, model.EventAttendee{
				Name:     a.Name,
				Email:    a.Email,
				PartStat: easPartStatToICal(a.AttendeeStatus),
			})
		}
	}
	return event
}

// applyAppointmentToEvent merges non-empty fields from eas.Appointment into an existing model.Event.
func applyAppointmentToEvent(appt *eas.Appointment, event *model.Event) {
	if appt.Subject != "" {
		event.Summary = appt.Subject
	}
	if appt.Location != "" {
		event.Location = appt.Location
	}
	if start, ok := parseEasTime(appt.StartTime); ok {
		event.StartAt = start
	}
	if end, ok := parseEasTime(appt.EndTime); ok {
		event.EndAt = end
	}
	event.IsAllDay = appt.AllDayEvent != 0
	if appt.OrganizerEmail != "" {
		event.OrganizerEmail = appt.OrganizerEmail
	}
	if appt.OrganizerName != "" {
		event.OrganizerName = appt.OrganizerName
	}
}

// partStatToEas converts an iCalendar PARTSTAT string to MS-ASCAL AttendeeStatus (0=none, 2=tentative, 3=accepted, 4=declined).
func partStatToEas(partStat string) int32 {
	switch strings.ToUpper(partStat) {
	case "ACCEPTED":
		return 3
	case "DECLINED":
		return 4
	case "TENTATIVE":
		return 2
	default:
		return 0
	}
}

// easPartStatToICal converts MS-ASCAL AttendeeStatus back to an iCalendar PARTSTAT string.
func easPartStatToICal(status int32) string {
	switch status {
	case 3:
		return "ACCEPTED"
	case 4:
		return "DECLINED"
	case 2:
		return "TENTATIVE"
	default:
		return "NEEDS-ACTION"
	}
}

// parseVEventCollectionID reports whether collectionID is a supported calendar collection (vevent/*).
func parseVEventCollectionID(collectionID string) bool {
	return strings.HasPrefix(collectionID, "vevent/")
}

// syncWindow returns the Sync batch size, defaulting to defaultSyncWindow when windowSize is zero.
func syncWindow(windowSize int32) int {
	if windowSize <= 0 {
		return defaultSyncWindow
	}
	return int(windowSize)
}
