package commands

import (
	"strings"
	"time"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/config"
)

// PingHandler implements the MS-ASCMD Ping command with long-poll change detection.
//
// Monitors mail, calendar, and contact collections listed in the Ping request and
// polls until HeartbeatInterval expires or a change is detected.
type PingHandler struct {
	cfg      *config.Config
	store    *state.Store
	mail     *MailSyncEngine
	calendar *CalendarSyncEngine
	contacts *ContactsSyncEngine
}

// NewPingHandler creates a PingHandler wired to all sync engines.
func NewPingHandler(cfg *config.Config, store *state.Store, mail *MailSyncEngine, calendar *CalendarSyncEngine, contacts *ContactsSyncEngine) *PingHandler {
	return &PingHandler{cfg: cfg, store: store, mail: mail, calendar: calendar, contacts: contacts}
}

// Handle waits up to HeartbeatInterval seconds for changes in monitored folders.
//
// Returns status 2 with changed folder IDs when any collection differs from the sync cache,
// or status 1 when the heartbeat interval expires with no changes.
func (h *PingHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	var req eas.PingRequest
	if len(body) > 0 {
		_ = wbxml.Unmarshal(body, &req)
	}

	if len(req.Folders.Folder) == 0 {
		return wbxml.Marshal(eas.PingResponse{Status: 1})
	}
	if h.store == nil || h.mail == nil || h.cfg == nil {
		return wbxml.Marshal(eas.PingResponse{Status: eas.SyncStatusServerError})
	}

	heartbeat := req.HeartbeatInterval
	if heartbeat <= 0 {
		heartbeat = int32(h.cfg.ActiveSync.MaxPingIntervalSec)
	}
	if heartbeat <= 0 {
		heartbeat = 30
	}
	if h.cfg.ActiveSync.MaxPingIntervalSec > 0 && int32(h.cfg.ActiveSync.MaxPingIntervalSec) < heartbeat {
		heartbeat = int32(h.cfg.ActiveSync.MaxPingIntervalSec)
	}

	dev, err := h.store.EnsureDevice(ctx.UserID, ctx.DeviceID, ctx.DeviceType)
	if err != nil {
		return wbxml.Marshal(eas.PingResponse{Status: eas.SyncStatusServerError})
	}

	deadline := time.Now().Add(time.Duration(heartbeat) * time.Second)
	pollEvery := 2 * time.Second

	// Partition folder list by collection type once.
	var mailFolders, calFolders, contactFolders []eas.PingFolder
	for _, f := range req.Folders.Folder {
		switch {
		case strings.HasPrefix(f.ID, "mail/"):
			mailFolders = append(mailFolders, f)
		case strings.HasPrefix(f.ID, "vevent/"):
			calFolders = append(calFolders, f)
		case strings.HasPrefix(f.ID, "vcard/"):
			contactFolders = append(contactFolders, f)
		}
	}

	for {
		var changedIDs []string

		// Mail change detection.
		if len(mailFolders) > 0 {
			changed, ids, err := h.mail.PingChangedCollections(ctx, dev, mailFolders)
			if err == nil && changed {
				changedIDs = append(changedIDs, ids...)
			}
		}

		// Calendar change detection.
		if len(calFolders) > 0 && h.calendar != nil {
			ids, err := h.pingCalendarChanged(ctx, dev, calFolders)
			if err == nil {
				changedIDs = append(changedIDs, ids...)
			}
		}

		// Contacts change detection.
		if len(contactFolders) > 0 && h.contacts != nil {
			ids, err := h.pingContactsChanged(ctx, dev, contactFolders)
			if err == nil {
				changedIDs = append(changedIDs, ids...)
			}
		}

		if len(changedIDs) > 0 {
			return wbxml.Marshal(eas.PingResponse{
				Status:  2,
				Folders: eas.PingResponseFolders{Folder: changedIDs},
			})
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(pollEvery)
	}

	return wbxml.Marshal(eas.PingResponse{Status: 1})
}

// pingCalendarChanged reports vevent/* collections that have changed since the last sync.
func (h *PingHandler) pingCalendarChanged(ctx *Context, dev *state.EasDevice, folders []eas.PingFolder) ([]string, error) {
	var changedIDs []string
	for _, folder := range folders {
		fst, err := h.store.GetFolderState(dev.ID, folder.ID)
		if err != nil {
			continue
		}
		cache := state.LoadItemSyncCache(fst.SyncCache)
		cal, err := h.calendar.calRepo.EnsureDefault(ctx.UserID)
		if err != nil {
			continue
		}
		events, err := h.calendar.eventRepo.ListByCalendar(ctx.UserID, cal.ID)
		if err != nil {
			continue
		}
		if len(events) != len(cache.Items) {
			changedIDs = append(changedIDs, folder.ID)
			continue
		}
		for _, ev := range events {
			sid := serverIDForUint(ev.ID)
			item, ok := cache.Items[sid]
			if !ok || item.UpdatedAt != ev.UpdatedAt.Unix() {
				changedIDs = append(changedIDs, folder.ID)
				break
			}
		}
	}
	return changedIDs, nil
}

// pingContactsChanged reports vcard/* collections that have changed since the last sync.
func (h *PingHandler) pingContactsChanged(ctx *Context, dev *state.EasDevice, folders []eas.PingFolder) ([]string, error) {
	var changedIDs []string
	for _, folder := range folders {
		fst, err := h.store.GetFolderState(dev.ID, folder.ID)
		if err != nil {
			continue
		}
		cache := state.LoadItemSyncCache(fst.SyncCache)
		contacts, err := h.contacts.contactRepo.List(ctx.UserID)
		if err != nil {
			continue
		}
		if len(contacts) != len(cache.Items) {
			changedIDs = append(changedIDs, folder.ID)
			continue
		}
		for _, ct := range contacts {
			sid := serverIDForUint(ct.ID)
			item, ok := cache.Items[sid]
			if !ok || item.UpdatedAt != ct.UpdatedAt.Unix() {
				changedIDs = append(changedIDs, folder.ID)
				break
			}
		}
	}
	return changedIDs, nil
}
