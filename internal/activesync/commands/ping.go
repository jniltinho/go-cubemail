package commands

import (
	"strings"
	"time"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/config"
	"go-cubemail/internal/repository"
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
		cal, ok := resolveCalendarCollection(h.calendar.calRepo, ctx.UserID, folder.ID)
		if !ok {
			continue
		}
		if rev, err := h.calendar.calRepo.Revision(ctx.UserID, cal.ID); err == nil &&
			unchangedSince(cache, rev) {
			continue
		}
		stamps, err := h.calendar.eventRepo.ListEventStamps(ctx.UserID, cal.ID)
		if err != nil {
			continue
		}
		if stampsDiffer(cache, stamps) {
			changedIDs = append(changedIDs, folder.ID)
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
		book, ok := resolveAddressBookCollection(h.contacts.bookRepo, ctx.UserID, folder.ID)
		if !ok {
			continue
		}
		if rev, err := h.contacts.bookRepo.Revision(ctx.UserID, book.ID); err == nil &&
			unchangedSince(cache, rev) {
			continue
		}
		stamps, err := h.contacts.contactRepo.ListContactStamps(ctx.UserID, book.ID)
		if err != nil {
			continue
		}
		if stampsDiffer(cache, stamps) {
			changedIDs = append(changedIDs, folder.ID)
		}
	}
	return changedIDs, nil
}

// unchangedSince reports that the collection token has not moved since the last
// Sync, which proves nothing in the collection changed.
//
// This is the fast path of the long-poll: one integer read instead of a scan.
// A zero token means the collection predates the DAV revision counter (or the
// last Sync did not record one), so the caller must fall back to comparing rows.
func unchangedSince(cache state.ItemSyncCache, revision uint64) bool {
	return revision != 0 && cache.DavRevision == revision
}

// stampsDiffer compares the collection's current object versions against what
// the device last received. It is the authoritative check; the revision above
// only decides whether it is worth running.
func stampsDiffer(cache state.ItemSyncCache, stamps []repository.ObjectStamp) bool {
	if len(stamps) != len(cache.Items) {
		return true
	}
	for _, s := range stamps {
		item, ok := cache.Items[serverIDForUint(s.ID)]
		if !ok {
			return true
		}
		current := state.ItemSyncItem{UpdatedAt: s.UpdatedAt.Unix(), Revision: s.SyncRevision}
		if !item.SameVersion(current) {
			return true
		}
	}
	return false
}
