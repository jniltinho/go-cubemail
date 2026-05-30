package commands

import (
	"strings"
	"time"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	calpkg "go-cubemail/internal/calendar"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
)

// TasksSyncEngine syncs VTODO tasks between SQL (EventRepo is_task=true) and EAS clients.
//
// Collection IDs match vtodo/* (e.g. vtodo/personal).
// Change detection uses event.UpdatedAt stored in ItemSyncCache, same pattern as CalendarSyncEngine.
// MS-ASTASK fields are mapped to/from model.Event fields.
type TasksSyncEngine struct {
	store     *state.Store
	eventRepo *repository.EventRepo
}

// NewTasksSyncEngine creates a TasksSyncEngine.
func NewTasksSyncEngine(store *state.Store, eventRepo *repository.EventRepo) *TasksSyncEngine {
	return &TasksSyncEngine{store: store, eventRepo: eventRepo}
}

// SyncCollection processes one vtodo/* collection in a Sync request.
func (t *TasksSyncEngine) SyncCollection(ctx *Context, dev *state.EasDevice, col eas.SyncCollection) (eas.SyncCollection, error) {
	if !strings.HasPrefix(col.CollectionID, "vtodo/") {
		return eas.SyncCollection{
			SyncKey:      col.SyncKey,
			CollectionID: col.CollectionID,
			Status:       eas.SyncStatusProtocolError,
		}, nil
	}

	fst, err := t.store.GetFolderState(dev.ID, col.CollectionID)
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
		Class:        "Tasks",
		Status:       eas.SyncStatusSuccess,
	}

	var responses *eas.SyncCommands
	if col.Commands != nil {
		responses, err = t.applyClientCommands(ctx, col.Commands, &cache)
		if err != nil {
			return eas.SyncCollection{}, err
		}
	}

	window := syncWindow(col.WindowSize)
	if col.GetChanges != 0 || col.SyncKey == "0" {
		adds, changes, deletes, more, err := t.buildServerChanges(ctx.UserID, cache, window)
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
	if responses != nil && len(responses.Add) > 0 {
		resp.Responses = responses
	}

	newKey, err := t.store.NextCollectionSyncKey(dev.ID, col.CollectionID)
	if err != nil {
		return eas.SyncCollection{}, err
	}
	fst.SyncKey = newKey
	fst.SyncCache = state.EncodeItemSyncCache(cache)
	if err := t.store.SaveFolderState(fst); err != nil {
		return eas.SyncCollection{}, err
	}
	resp.SyncKey = newKey
	return resp, nil
}

// EstimateCount returns the number of tasks for the user.
func (t *TasksSyncEngine) EstimateCount(ctx *Context) (int32, error) {
	tasks, err := t.eventRepo.ListTasks(ctx.UserID)
	if err != nil {
		return 0, err
	}
	return int32(len(tasks)), nil
}

// applyClientCommands handles client Add/Change/Delete for vtodo collections.
func (t *TasksSyncEngine) applyClientCommands(ctx *Context, cmds *eas.SyncCommands, cache *state.ItemSyncCache) (*eas.SyncCommands, error) {
	var responses eas.SyncCommands

	for _, add := range cmds.Add {
		task, err := add.Task()
		if err != nil {
			continue
		}
		event := easTaskToEvent(task, ctx.UserID)
		event.ICalContent = calpkg.BuildICalContent(event, nil)
		if err := t.eventRepo.Create(event); err != nil {
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
		task, err := chg.Task()
		if err != nil {
			continue
		}
		event, err := t.eventRepo.Get(ctx.UserID, id)
		if err != nil {
			continue
		}
		applyEASTaskToEvent(task, event)
		event.ICalContent = calpkg.BuildICalContent(event, nil)
		if err := t.eventRepo.Update(event); err != nil {
			return nil, err
		}
		cache.Items[chg.ServerID] = state.ItemSyncItem{UpdatedAt: event.UpdatedAt.Unix()}
	}

	for _, del := range cmds.Delete {
		id, ok := parseServerID(del.ServerID)
		if !ok {
			continue
		}
		_ = t.eventRepo.Delete(ctx.UserID, id)
		delete(cache.Items, del.ServerID)
	}

	if len(responses.Add) == 0 {
		return nil, nil
	}
	return &responses, nil
}

// buildServerChanges compares SQL tasks with the ItemSyncCache and emits Add/Change/Delete commands.
func (t *TasksSyncEngine) buildServerChanges(userID uint, cache state.ItemSyncCache, window int) ([]eas.SyncAdd, []eas.SyncChange, []eas.SyncDelete, bool, error) {
	tasks, err := t.eventRepo.ListTasks(userID)
	if err != nil {
		return nil, nil, nil, false, err
	}

	current := make(map[string]state.ItemSyncItem, len(tasks))
	taskByID := make(map[string]model.Event, len(tasks))
	for _, ev := range tasks {
		sid := serverIDForUint(ev.ID)
		current[sid] = state.ItemSyncItem{UpdatedAt: ev.UpdatedAt.Unix()}
		taskByID[sid] = ev
	}

	type pending struct {
		kind     string
		serverID string
	}
	var queue []pending
	for sid := range current {
		if _, ok := cache.Items[sid]; !ok {
			queue = append(queue, pending{"add", sid})
		} else if cache.Items[sid].UpdatedAt != current[sid].UpdatedAt {
			queue = append(queue, pending{"change", sid})
		}
	}
	for sid := range cache.Items {
		if _, ok := current[sid]; !ok {
			queue = append(queue, pending{"delete", sid})
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
			ev := taskByID[p.serverID]
			task := eventToEASTask(&ev)
			body, err := marshalApplicationDataBody(&task)
			if err != nil {
				return nil, nil, nil, false, err
			}
			adds = append(adds, eas.SyncAdd{
				ServerID: p.serverID,
				ApplicationData: &wbxml.RawElement{Page: wbxml.PageAirSync, Bytes: body},
			})
			cache.Items[p.serverID] = current[p.serverID]
		case "change":
			ev := taskByID[p.serverID]
			task := eventToEASTask(&ev)
			body, err := marshalApplicationDataBody(&task)
			if err != nil {
				return nil, nil, nil, false, err
			}
			changes = append(changes, eas.SyncChange{
				ServerID: p.serverID,
				ApplicationData: &wbxml.RawElement{Page: wbxml.PageAirSync, Bytes: body},
			})
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

// eventToEASTask maps a model.Event task to an eas.Task ApplicationData payload.
func eventToEASTask(event *model.Event) eas.Task {
	task := eas.Task{
		Subject:     event.Summary,
		Importance:  easImportanceFromPriority(event.Priority),
		Sensitivity: 0,
	}
	if !event.EndAt.IsZero() {
		due := easTime(event.EndAt, false)
		task.DueDate = due
		task.UtcDueDate = due
	}
	if strings.EqualFold(event.Status, "COMPLETED") {
		task.Complete = 1
		task.DateCompleted = easTime(time.Now(), false)
	}
	return task
}

// easTaskToEvent converts a client eas.Task into a new model.Event for Create.
func easTaskToEvent(task *eas.Task, userID uint) *model.Event {
	var dueAt time.Time
	if task.DueDate != "" {
		dueAt, _ = parseEasTime(task.DueDate)
	}
	if dueAt.IsZero() {
		dueAt = time.Now().Add(24 * time.Hour)
	}
	status := "CONFIRMED"
	if task.Complete != 0 {
		status = "COMPLETED"
	}
	return &model.Event{
		UserID:  userID,
		UID:     calpkg.NewUID(""),
		Summary: task.Subject,
		EndAt:   dueAt,
		StartAt: dueAt,
		Status:  status,
		IsTask:  true,
	}
}

// applyEASTaskToEvent merges EAS Task fields into an existing model.Event task.
func applyEASTaskToEvent(task *eas.Task, event *model.Event) {
	if task.Subject != "" {
		event.Summary = task.Subject
	}
	if task.DueDate != "" {
		if t, ok := parseEasTime(task.DueDate); ok {
			event.EndAt = t
			event.StartAt = t
		}
	}
	if task.Complete != 0 {
		event.Status = "COMPLETED"
	} else {
		event.Status = "CONFIRMED"
	}
}

// easImportanceFromPriority converts an event priority to an EAS importance value.
func easImportanceFromPriority(priority int) int32 {
	switch {
	case priority >= 7:
		return 0 // low
	case priority >= 4:
		return 1 // normal
	default:
		return 2 // high
	}
}

// parseVTodoCollectionID reports whether collectionID is a supported task collection (vtodo/*).
func parseVTodoCollectionID(collectionID string) bool {
	return strings.HasPrefix(collectionID, "vtodo/")
}

