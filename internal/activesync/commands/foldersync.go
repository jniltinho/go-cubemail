package commands

import (
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
	calRepo *repository.CalendarRepo
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

	if req.SyncKey == "0" || req.SyncKey == "" {
		adds := make([]eas.FolderAdd, 0, len(list))
		for _, f := range list {
			adds = append(adds, eas.FolderAdd{
				ServerID:    f.ServerID,
				ParentID:    f.ParentID,
				DisplayName: f.DisplayName,
				Type:        f.Type,
			})
		}
		newKey := h.store.NextFolderSyncKey(dev)
		return wbxml.Marshal(eas.FolderSyncResponse{
			Status:  eas.StatusSuccess,
			SyncKey: newKey,
			Changes: eas.FolderChanges{
				Count: int32(len(adds)),
				Add:   adds,
			},
		})
	}

	return wbxml.Marshal(eas.FolderSyncResponse{
		Status:  eas.StatusSuccess,
		SyncKey: dev.FolderSyncKey,
		Changes: eas.FolderChanges{Count: 0},
	})
}

// FolderEntry describes one folder in the EAS hierarchy returned by FolderSync.
type FolderEntry struct {
	ServerID    string // EAS collection ID (e.g. mail/{guid}, vevent/personal).
	ParentID    string // Parent folder ServerID ("0" for top-level).
	DisplayName string // Human-readable folder name shown on the device.
	Type        int32  // MS-ASCMD folder type code (2=inbox, 8=calendar, etc.).
}

// FolderBuilder assembles mail, calendar, contacts, and task folders for FolderSync.
type FolderBuilder struct {
	cfg   *config.Config
	store *state.Store
}

// NewFolderBuilder creates a FolderBuilder.
func NewFolderBuilder(cfg *config.Config, store *state.Store) *FolderBuilder {
	return &FolderBuilder{cfg: cfg, store: store}
}

// Build returns the full folder list for the authenticated user.
func (b *FolderBuilder) Build(ctx *Context) ([]FolderEntry, error) {
	mailFolders, err := b.mailFolders(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]FolderEntry, 0, len(mailFolders)+3)
	out = append(out, mailFolders...)
	out = append(out,
		FolderEntry{ServerID: "vevent/personal", ParentID: "0", DisplayName: "Calendar", Type: 8},
		FolderEntry{ServerID: "vtodo/personal", ParentID: "0", DisplayName: "Tasks", Type: 7},
		FolderEntry{ServerID: "vcard/personal", ParentID: "0", DisplayName: "Contacts", Type: 9},
	)
	return out, nil
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
