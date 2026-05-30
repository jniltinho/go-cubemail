package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/config"
	mailimap "go-cubemail/internal/imap"
)

// defaultSyncWindow is the maximum Add/Change/Delete items returned per Sync when WindowSize is unset.
const defaultSyncWindow = 100

// MailSyncEngine syncs IMAP mail folders via the EAS Sync command (mail/{guid} collections).
type MailSyncEngine struct {
	cfg   *config.Config // Application config (IMAP host, timeouts, debug).
	store *state.Store   // Folder GUID mapping and per-collection sync cache.
}

// NewMailSyncEngine creates a MailSyncEngine.
func NewMailSyncEngine(cfg *config.Config, store *state.Store) *MailSyncEngine {
	return &MailSyncEngine{cfg: cfg, store: store}
}

// SyncCollection processes one mail/* collection in a Sync request.
func (m *MailSyncEngine) SyncCollection(ctx *Context, dev *state.EasDevice, col eas.SyncCollection) (eas.SyncCollection, error) {
	guid, ok := parseMailCollectionID(col.CollectionID)
	if !ok {
		return eas.SyncCollection{
			SyncKey:      col.SyncKey,
			CollectionID: col.CollectionID,
			Status:       eas.SyncStatusProtocolError,
		}, nil
	}

	folderPath, err := m.store.FolderPathByGUID(ctx.UserID, guid)
	if err != nil {
		return eas.SyncCollection{
			SyncKey:      col.SyncKey,
			CollectionID: col.CollectionID,
			Status:       eas.SyncStatusObjectNotFound,
		}, nil
	}

	fst, err := m.store.GetFolderState(dev.ID, col.CollectionID)
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

	timeout := time.Duration(m.cfg.IMAP.TimeoutSec) * time.Second
	client, err := mailimap.Connect(m.cfg.IMAP.Host, m.cfg.IMAP.Port, m.cfg.IMAP.TLS, timeout, ctx.Username, ctx.Password, m.cfg.Server.Debug)
	if err != nil {
		return eas.SyncCollection{}, fmt.Errorf("imap connect: %w", err)
	}
	defer client.Close()

	if err := client.SelectMailbox(folderPath); err != nil {
		return eas.SyncCollection{
			SyncKey:      col.SyncKey,
			CollectionID: col.CollectionID,
			Status:       eas.SyncStatusObjectNotFound,
		}, nil
	}

	cache := state.LoadMailSyncCache(fst.SyncCache)
	resp := eas.SyncCollection{
		CollectionID: col.CollectionID,
		Class:        "Email",
		Status:       eas.SyncStatusSuccess,
	}

	if col.Commands != nil {
		if err := m.applyClientCommands(client, col.Commands, &cache); err != nil {
			return eas.SyncCollection{}, err
		}
	}

	window := int(col.WindowSize)
	if window <= 0 {
		window = defaultSyncWindow
	}

	if col.GetChanges != 0 || col.SyncKey == "0" {
		adds, changes, deletes, more, err := m.buildServerChanges(client, cache, window)
		if err != nil {
			return eas.SyncCollection{}, err
		}
		if len(adds)+len(changes)+len(deletes) > 0 {
			resp.Commands = &eas.SyncCommands{
				Add:    adds,
				Change: changes,
				Delete: deletes,
			}
		}
		if more {
			resp.MoreAvailable = 1
		}
	}

	newKey, err := m.store.NextCollectionSyncKey(dev.ID, col.CollectionID)
	if err != nil {
		return eas.SyncCollection{}, err
	}
	fst.SyncKey = newKey
	fst.SyncCache = state.EncodeMailSyncCache(cache)
	if err := m.store.SaveFolderState(fst); err != nil {
		return eas.SyncCollection{}, err
	}

	resp.SyncKey = newKey
	return resp, nil
}

// applyClientCommands applies client-originated Sync Change/Delete commands to IMAP and updates the cache.
func (m *MailSyncEngine) applyClientCommands(client *mailimap.Client, cmds *eas.SyncCommands, cache *state.MailSyncCache) error {
	for _, chg := range cmds.Change {
		uid, ok := parseServerUID(chg.ServerID)
		if !ok {
			continue
		}
		email, err := chg.Email()
		if err != nil {
			continue
		}
		if email.Read {
			if err := client.MarkSeen(uid); err != nil {
				return err
			}
		} else {
			if err := client.MarkUnseen(uid); err != nil {
				return err
			}
		}
		if item, ok := cache.Items[chg.ServerID]; ok {
			item.Seen = email.Read
			cache.Items[chg.ServerID] = item
		}
	}
	for _, del := range cmds.Delete {
		uid, ok := parseServerUID(del.ServerID)
		if !ok {
			continue
		}
		if err := client.DeleteMessage(uid); err != nil {
			return err
		}
		delete(cache.Items, del.ServerID)
	}
	return nil
}

// buildServerChanges compares IMAP state with the sync cache and builds Add/Change/Delete commands.
// Respects window size; sets more=true when additional changes remain after this batch.
func (m *MailSyncEngine) buildServerChanges(client *mailimap.Client, cache state.MailSyncCache, window int) ([]eas.SyncAdd, []eas.SyncChange, []eas.SyncDelete, bool, error) {
	uids, err := client.FetchAllUIDs()
	if err != nil {
		return nil, nil, nil, false, err
	}
	if len(uids) == 0 {
		cache.Items = map[string]state.MailSyncItem{}
		return nil, nil, nil, false, nil
	}

	envelopes, err := client.FetchEnvelopes(uids)
	if err != nil {
		return nil, nil, nil, false, err
	}

	current := make(map[string]state.MailSyncItem, len(envelopes))
	for _, env := range envelopes {
		id := serverIDForUID(env.UID)
		current[id] = state.MailSyncItem{
			UID:     uint32(env.UID),
			Seen:    env.Seen,
			Flagged: env.Flagged,
		}
	}

	type pending struct {
		kind     string
		serverID string
		item     state.MailSyncItem
	}
	var queue []pending

	for id, item := range current {
		prev, ok := cache.Items[id]
		if !ok {
			queue = append(queue, pending{kind: "add", serverID: id, item: item})
			continue
		}
		if prev.Seen != item.Seen || prev.Flagged != item.Flagged {
			queue = append(queue, pending{kind: "change", serverID: id, item: item})
		}
	}
	for id := range cache.Items {
		if _, ok := current[id]; !ok {
			queue = append(queue, pending{kind: "delete", serverID: id})
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
			add, err := m.buildSyncAdd(p.serverID, p.item, envelopes)
			if err != nil {
				return nil, nil, nil, false, err
			}
			adds = append(adds, add)
			cache.Items[p.serverID] = p.item
		case "change":
			chg, err := m.buildSyncChange(p.serverID, p.item, envelopes)
			if err != nil {
				return nil, nil, nil, false, err
			}
			changes = append(changes, chg)
			cache.Items[p.serverID] = p.item
		case "delete":
			deletes = append(deletes, eas.SyncDelete{ServerID: p.serverID})
			delete(cache.Items, p.serverID)
		}
		remaining--
	}

	more := len(queue) > len(adds)+len(changes)+len(deletes)
	return adds, changes, deletes, more, nil
}

// buildSyncAdd wraps an IMAP envelope as an EAS Sync Add command with Email ApplicationData.
func (m *MailSyncEngine) buildSyncAdd(serverID string, item state.MailSyncItem, envelopes []mailimap.Envelope) (eas.SyncAdd, error) {
	email, err := m.envelopeToEmail(serverID, item, envelopes)
	if err != nil {
		return eas.SyncAdd{}, err
	}
	body, err := marshalApplicationDataBody(&email)
	if err != nil {
		return eas.SyncAdd{}, err
	}
	return eas.SyncAdd{
		ServerID: serverID,
		ApplicationData: &wbxml.RawElement{
			Page:  wbxml.PageAirSync,
			Bytes: body,
		},
	}, nil
}

// buildSyncChange wraps an updated IMAP envelope as an EAS Sync Change command.
func (m *MailSyncEngine) buildSyncChange(serverID string, item state.MailSyncItem, envelopes []mailimap.Envelope) (eas.SyncChange, error) {
	email, err := m.envelopeToEmail(serverID, item, envelopes)
	if err != nil {
		return eas.SyncChange{}, err
	}
	body, err := marshalApplicationDataBody(&email)
	if err != nil {
		return eas.SyncChange{}, err
	}
	return eas.SyncChange{
		ServerID: serverID,
		ApplicationData: &wbxml.RawElement{
			Page:  wbxml.PageAirSync,
			Bytes: body,
		},
	}, nil
}

// envelopeToEmail maps an internal/imap.Envelope plus flag state to an eas.Email ApplicationData payload.
func (m *MailSyncEngine) envelopeToEmail(serverID string, item state.MailSyncItem, envelopes []mailimap.Envelope) (eas.Email, error) {
	uid, _ := parseServerUID(serverID)
	for _, env := range envelopes {
		if env.UID != uid {
			continue
		}
		from := env.FromEmail
		if from == "" {
			from = env.From
		}
		return eas.Email{
			DateReceived: easDateFromEnvelope(env.Date),
			Subject:      env.Subject,
			From:         from,
			To:           env.To,
			Read:         item.Seen,
			Importance:   eas.ImportanceNormal,
			MessageClass: "IPM.Note",
			ContentClass: "urn:content-classes:message",
		}, nil
	}
	return eas.Email{Read: item.Seen}, nil
}

// EstimateCount returns the IMAP message count for a mail collection.
func (m *MailSyncEngine) EstimateCount(ctx *Context, collectionID string) (int32, error) {
	guid, ok := parseMailCollectionID(collectionID)
	if !ok {
		return 0, fmt.Errorf("invalid mail collection")
	}
	folderPath, err := m.store.FolderPathByGUID(ctx.UserID, guid)
	if err != nil {
		return 0, err
	}
	timeout := time.Duration(m.cfg.IMAP.TimeoutSec) * time.Second
	client, err := mailimap.Connect(m.cfg.IMAP.Host, m.cfg.IMAP.Port, m.cfg.IMAP.TLS, timeout, ctx.Username, ctx.Password, m.cfg.Server.Debug)
	if err != nil {
		return 0, err
	}
	defer client.Close()
	n, err := client.MessageCount(folderPath)
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}

// PingChangedCollections reports whether monitored mail folders differ from the last sync cache.
//
// Compares IMAP UID count and per-message Seen/Flagged state against MailSyncCache stored in
// EasFolderState. Non-mail folders in the Ping request are skipped. Returns the collection IDs
// that changed.
func (m *MailSyncEngine) PingChangedCollections(ctx *Context, dev *state.EasDevice, folders []eas.PingFolder) (bool, []string, error) {
	if len(folders) == 0 {
		return false, nil, nil
	}

	timeout := time.Duration(m.cfg.IMAP.TimeoutSec) * time.Second
	client, err := mailimap.Connect(m.cfg.IMAP.Host, m.cfg.IMAP.Port, m.cfg.IMAP.TLS, timeout, ctx.Username, ctx.Password, m.cfg.Server.Debug)
	if err != nil {
		return false, nil, err
	}
	defer client.Close()

	var changedIDs []string
	for _, folder := range folders {
		collectionID := folder.ID
		if !strings.HasPrefix(collectionID, "mail/") {
			continue
		}
		guid, ok := parseMailCollectionID(collectionID)
		if !ok {
			continue
		}
		folderPath, err := m.store.FolderPathByGUID(ctx.UserID, guid)
		if err != nil {
			continue
		}
		fst, err := m.store.GetFolderState(dev.ID, collectionID)
		if err != nil {
			return false, nil, err
		}
		cache := state.LoadMailSyncCache(fst.SyncCache)

		if err := client.SelectMailbox(folderPath); err != nil {
			continue
		}
		uids, err := client.FetchAllUIDs()
		if err != nil {
			return false, nil, err
		}
		if len(uids) != len(cache.Items) {
			changedIDs = append(changedIDs, collectionID)
			continue
		}
		if len(uids) == 0 {
			continue
		}
		envelopes, err := client.FetchEnvelopes(uids)
		if err != nil {
			return false, nil, err
		}
		for _, env := range envelopes {
			id := serverIDForUID(env.UID)
			item, ok := cache.Items[id]
			if !ok || item.Seen != env.Seen || item.Flagged != env.Flagged {
				changedIDs = append(changedIDs, collectionID)
				break
			}
		}
	}
	return len(changedIDs) > 0, changedIDs, nil
}

// parseMailCollectionID extracts the folder GUID from a mail/{guid} collection ID.
func parseMailCollectionID(collectionID string) (string, bool) {
	const prefix = "mail/"
	if !strings.HasPrefix(collectionID, prefix) {
		return "", false
	}
	guid := strings.TrimPrefix(collectionID, prefix)
	if guid == "" {
		return "", false
	}
	return guid, true
}

// serverIDForUID converts an IMAP UID to the EAS ServerId string (decimal UID).
func serverIDForUID(uid goimap.UID) string {
	return strconv.FormatUint(uint64(uid), 10)
}

// parseServerUID parses an EAS ServerId back to an IMAP UID.
func parseServerUID(serverID string) (goimap.UID, bool) {
	n, err := strconv.ParseUint(serverID, 10, 32)
	if err != nil {
		return 0, false
	}
	return goimap.UID(n), true
}

// easDateFromEnvelope converts an envelope date string to MS-ASEMAIL DateReceived format (YYYYMMDDTHHMMSSZ).
func easDateFromEnvelope(dateStr string) string {
	if dateStr == "" {
		return time.Now().UTC().Format("20060102T150405Z")
	}
	layouts := []string{
		"02/01/2006 15:04",
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t.UTC().Format("20060102T150405Z")
		}
	}
	return time.Now().UTC().Format("20060102T150405Z")
}
