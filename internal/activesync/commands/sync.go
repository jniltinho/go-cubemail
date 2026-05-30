package commands

import (
	"strings"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
)

// SyncHandler implements the MS-ASCMD Sync command for all collection types.
type SyncHandler struct {
	store    *state.Store          // Device and collection sync key persistence.
	mail     *MailSyncEngine       // mail/* collections (IMAP).
	calendar *CalendarSyncEngine   // vevent/* collections (SQL events).
	contacts *ContactsSyncEngine   // vcard/* collections (SQL contacts).
	tasks    *TasksSyncEngine      // vtodo/* collections (SQL tasks).
}

// NewSyncHandler constructs a SyncHandler wired with mail, calendar, contacts, and tasks engines.
func NewSyncHandler(store *state.Store, mail *MailSyncEngine, calendar *CalendarSyncEngine, contacts *ContactsSyncEngine, tasks *TasksSyncEngine) *SyncHandler {
	return &SyncHandler{
		store:    store,
		mail:     mail,
		calendar: calendar,
		contacts: contacts,
		tasks:    tasks,
	}
}

// Handle processes Sync requests for mail, calendar, contacts, and tasks collections.
func (h *SyncHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	var req eas.SyncRequest
	if len(body) > 0 {
		if err := wbxml.Unmarshal(body, &req); err != nil {
			return nil, err
		}
	}

	dev, err := h.store.EnsureDevice(ctx.UserID, ctx.DeviceID, ctx.DeviceType)
	if err != nil {
		return nil, err
	}

	collections := make([]eas.SyncCollection, 0, len(req.Collections.Collection))
	for _, col := range req.Collections.Collection {
		out, err := h.syncCollection(ctx, dev, col)
		if err != nil {
			return nil, err
		}
		collections = append(collections, out)
	}

	resp := eas.SyncResponse{
		Status: eas.StatusSuccess,
		Collections: eas.SyncCollections{
			Collection: collections,
		},
	}
	return wbxml.Marshal(resp)
}

// syncCollection routes a single Sync collection to the appropriate sync engine.
func (h *SyncHandler) syncCollection(ctx *Context, dev *state.EasDevice, col eas.SyncCollection) (eas.SyncCollection, error) {
	switch {
	case strings.HasPrefix(col.CollectionID, "mail/"):
		return h.mail.SyncCollection(ctx, dev, col)
	case strings.HasPrefix(col.CollectionID, "vevent/"):
		return h.calendar.SyncCollection(ctx, dev, col)
	case strings.HasPrefix(col.CollectionID, "vcard/"):
		return h.contacts.SyncCollection(ctx, dev, col)
	case strings.HasPrefix(col.CollectionID, "vtodo/"):
		return h.tasks.SyncCollection(ctx, dev, col)
	default:
		return h.syncCollectionStub(dev, col)
	}
}

// syncCollectionStub validates sync keys for collections not yet implemented (e.g. vtodo/*).
func (h *SyncHandler) syncCollectionStub(dev *state.EasDevice, col eas.SyncCollection) (eas.SyncCollection, error) {
	curKey, _ := h.store.GetCollectionSyncKey(dev.ID, col.CollectionID)
	if col.SyncKey != "0" && col.SyncKey != curKey {
		return eas.SyncCollection{
			SyncKey:      curKey,
			CollectionID: col.CollectionID,
			Status:       eas.SyncStatusInvalidSyncKey,
		}, nil
	}
	newKey, err := h.store.NextCollectionSyncKey(dev.ID, col.CollectionID)
	if err != nil {
		return eas.SyncCollection{}, err
	}
	return eas.SyncCollection{
		SyncKey:      newKey,
		CollectionID: col.CollectionID,
		Status:       eas.SyncStatusSuccess,
	}, nil
}
