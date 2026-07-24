package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/config"
	"go-cubemail/internal/imap"
	"go-cubemail/internal/repository"
)

// FolderSyncHandler implements the MS-ASCMD FolderSync command.
type FolderSyncHandler struct {
	store   *state.Store
	folders *FolderBuilder
}

// Handle processes FolderSync and returns folder hierarchy changes.
func (h *FolderSyncHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	var req eas.FolderSyncRequest
	if len(body) > 0 {
		if err := wbxml.Unmarshal(body, &req); err != nil {
			return nil, err
		}
	}

	dev, err := h.store.EnsureDevice(ctx.UserID, ctx.DeviceID, ctx.DeviceType)
	if err != nil {
		return nil, err
	}
	ctx.Device = dev

	if req.SyncKey != "0" && req.SyncKey != "" && req.SyncKey != dev.FolderSyncKey {
		return wbxml.Marshal(eas.FolderSyncResponse{Status: 3})
	}

	list, err := h.folders.Build(ctx)
	if err != nil {
		return nil, err
	}

	initial := req.SyncKey == "0" || req.SyncKey == ""
	changes := h.diffHierarchy(dev, list, initial)
	count := int32(len(changes.Add) + len(changes.Update) + len(changes.Delete))
	changes.Count = count

	// A response that reports no changes must keep the current key; bumping it
	// would invalidate the client's state for nothing.
	if count == 0 && !initial {
		return wbxml.Marshal(eas.FolderSyncResponse{
			Status:  eas.StatusSuccess,
			SyncKey: dev.FolderSyncKey,
			Changes: eas.FolderChanges{Count: 0},
		})
	}

	if err := h.saveHierarchy(dev, list); err != nil {
		return nil, err
	}
	return wbxml.Marshal(eas.FolderSyncResponse{
		Status:  eas.StatusSuccess,
		SyncKey: h.store.NextFolderSyncKey(dev),
		Changes: changes,
	})
}

// hierarchyCollectionID is the synthetic collection under which the last folder
// list sent to a device is stored. Reusing the folder-state table keeps the
// snapshot next to the rest of the device's sync state without a new table.
const hierarchyCollectionID = "__hierarchy__"

// diffHierarchy compares the current folder list against the one the device was
// last given, so a calendar or address book created after the device was set up
// still reaches it. Without this, a device that has already synced once keeps
// its original folder list forever.
func (h *FolderSyncHandler) diffHierarchy(dev *state.EasDevice, list []FolderEntry, initial bool) eas.FolderChanges {
	var changes eas.FolderChanges

	previous := map[string]FolderEntry{}
	if !initial {
		previous = h.loadHierarchy(dev)
	}

	current := make(map[string]FolderEntry, len(list))
	for _, f := range list {
		current[f.ServerID] = f
		prev, known := previous[f.ServerID]
		switch {
		case !known:
			changes.Add = append(changes.Add, eas.FolderAdd{
				ServerID: f.ServerID, ParentID: f.ParentID,
				DisplayName: f.DisplayName, Type: f.Type,
			})
		case prev.DisplayName != f.DisplayName || prev.Type != f.Type || prev.ParentID != f.ParentID:
			changes.Update = append(changes.Update, eas.FolderUpdate{
				ServerID: f.ServerID, ParentID: f.ParentID,
				DisplayName: f.DisplayName, Type: f.Type,
			})
		}
	}
	for id := range previous {
		if _, ok := current[id]; !ok {
			changes.Delete = append(changes.Delete, eas.FolderDelete{ServerID: id})
		}
	}
	return changes
}

// loadHierarchy returns the folder list last sent to the device.
func (h *FolderSyncHandler) loadHierarchy(dev *state.EasDevice) map[string]FolderEntry {
	out := map[string]FolderEntry{}
	fst, err := h.store.GetFolderState(dev.ID, hierarchyCollectionID)
	if err != nil || fst.SyncCache == "" {
		return out
	}
	var entries []FolderEntry
	if err := json.Unmarshal([]byte(fst.SyncCache), &entries); err != nil {
		return out
	}
	for _, f := range entries {
		out[f.ServerID] = f
	}
	return out
}

// saveHierarchy records the folder list just sent to the device.
func (h *FolderSyncHandler) saveHierarchy(dev *state.EasDevice, list []FolderEntry) error {
	fst, err := h.store.GetFolderState(dev.ID, hierarchyCollectionID)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(list)
	if err != nil {
		return err
	}
	fst.SyncCache = string(encoded)
	return h.store.SaveFolderState(fst)
}

// FolderEntry describes one folder in the EAS hierarchy returned by FolderSync.
type FolderEntry struct {
	ServerID    string `json:"server_id"`    // EAS collection ID (e.g. mail/{guid}, vevent/personal).
	ParentID    string `json:"parent_id"`    // Parent folder ServerID ("0" for top-level).
	DisplayName string `json:"display_name"` // Human-readable folder name shown on the device.
	Type        int32  `json:"type"`         // MS-ASCMD folder type code (2=inbox, 8=calendar, etc.).
}

// FolderBuilder assembles mail, calendar, contacts, and task folders for FolderSync.
type FolderBuilder struct {
	cfg      *config.Config
	store    *state.Store
	calRepo  *repository.CalendarRepo
	bookRepo *repository.AddressBookRepo
}

// NewFolderBuilder creates a FolderBuilder.
func NewFolderBuilder(cfg *config.Config, store *state.Store,
	calRepo *repository.CalendarRepo, bookRepo *repository.AddressBookRepo) *FolderBuilder {
	return &FolderBuilder{cfg: cfg, store: store, calRepo: calRepo, bookRepo: bookRepo}
}

// MS-ASCMD folder type codes.
const (
	folderTypeTasks    = 7
	folderTypeCalendar = 8
	folderTypeContacts = 9
)

// Build returns the full folder list for the authenticated user: every IMAP
// mailbox, every calendar, every address book, plus the task collection.
//
// One DAV collection is one device folder, so a calendar created in Thunderbird
// shows up on the phone. The default collections keep their historical
// "personal" IDs — see collections.go.
func (b *FolderBuilder) Build(ctx *Context) ([]FolderEntry, error) {
	mailFolders, err := b.mailFolders(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]FolderEntry, 0, len(mailFolders)+4)
	out = append(out, mailFolders...)
	out = append(out, b.calendarFolders(ctx)...)
	out = append(out, FolderEntry{
		ServerID: prefixTasks + defaultCollection, ParentID: "0",
		DisplayName: "Tasks", Type: folderTypeTasks,
	})
	out = append(out, b.contactFolders(ctx)...)
	return out, nil
}

// calendarFolders maps every calendar of the user to a device folder.
func (b *FolderBuilder) calendarFolders(ctx *Context) []FolderEntry {
	if b.calRepo == nil {
		return nil
	}
	if _, err := b.calRepo.EnsureDefault(ctx.UserID); err != nil {
		return nil
	}
	if err := b.calRepo.BackfillURIs(ctx.UserID); err != nil {
		return nil
	}
	cals, err := b.calRepo.List(ctx.UserID)
	if err != nil {
		return nil
	}
	out := make([]FolderEntry, 0, len(cals))
	for _, cal := range cals {
		name := cal.Name
		if cal.IsDefault {
			name = "Calendar"
		}
		out = append(out, FolderEntry{
			ServerID: calendarCollectionID(cal), ParentID: "0",
			DisplayName: name, Type: folderTypeCalendar,
		})
	}
	return out
}

// contactFolders maps every address book of the user to a device folder.
func (b *FolderBuilder) contactFolders(ctx *Context) []FolderEntry {
	if b.bookRepo == nil {
		return nil
	}
	if _, err := b.bookRepo.EnsureDefault(ctx.UserID); err != nil {
		return nil
	}
	books, err := b.bookRepo.List(ctx.UserID)
	if err != nil {
		return nil
	}
	out := make([]FolderEntry, 0, len(books))
	for _, book := range books {
		name := book.DisplayName
		if book.IsDefault || name == "" {
			name = "Contacts"
		}
		out = append(out, FolderEntry{
			ServerID: addressBookCollectionID(book), ParentID: "0",
			DisplayName: name, Type: folderTypeContacts,
		})
	}
	return out
}

// mailFolders lists IMAP mailboxes and maps each to a mail/{guid} FolderEntry.
func (b *FolderBuilder) mailFolders(ctx *Context) ([]FolderEntry, error) {
	timeout := time.Duration(b.cfg.IMAP.TimeoutSec) * time.Second
	client, err := imap.Connect(b.cfg.IMAP.Host, b.cfg.IMAP.Port, b.cfg.IMAP.TLS, timeout, ctx.Username, ctx.Password, b.cfg.Server.Debug)
	if err != nil {
		return nil, fmt.Errorf("imap connect: %w", err)
	}
	defer client.Close()

	boxes, err := client.ListMailboxes()
	if err != nil {
		return nil, err
	}

	out := make([]FolderEntry, 0, len(boxes))
	for _, box := range boxes {
		if box.NoSelect {
			continue
		}
		guid, err := b.store.FolderGUID(ctx.UserID, box.Name)
		if err != nil {
			return nil, err
		}
		out = append(out, FolderEntry{
			ServerID:    "mail/" + guid,
			ParentID:    "0",
			DisplayName: displayName(box),
			Type:        mailFolderType(box),
		})
	}
	return out, nil
}

// displayName returns the best display label for an IMAP mailbox.
func displayName(box imap.MailboxInfo) string {
	if box.DisplayName != "" {
		return box.DisplayName
	}
	return box.Name
}

// mailFolderType maps IMAP mailbox metadata to an MS-ASCMD folder type integer.
func mailFolderType(box imap.MailboxInfo) int32 {
	switch box.IconType {
	case "inbox":
		return 2
	case "drafts":
		return 3
	case "trash":
		return 4
	case "sent":
		return 5
	case "junk":
		return 6
	default:
		return 1
	}
}
