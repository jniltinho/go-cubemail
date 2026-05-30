package commands

import (
	"fmt"
	"strings"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
)

// ContactsSyncEngine syncs address book entries between SQL (ContactRepo) and EAS clients.
//
// Supported collection IDs: vcard/* (all contacts for the authenticated user).
// Server IDs are decimal contact primary keys. Change detection uses contact.UpdatedAt in ItemSyncCache.
type ContactsSyncEngine struct {
	store       *state.Store             // Sync keys and per-collection item cache.
	contactRepo *repository.ContactRepo // CRUD for model.Contact rows.
}

// NewContactsSyncEngine creates a ContactsSyncEngine backed by the given store and repository.
func NewContactsSyncEngine(store *state.Store, contactRepo *repository.ContactRepo) *ContactsSyncEngine {
	return &ContactsSyncEngine{store: store, contactRepo: contactRepo}
}

// SyncCollection processes one vcard/* collection in an MS-ASCMD Sync request.
//
// Flow: validate sync key → apply client Commands → build server changes when GetChanges is set
// → bump sync key and persist ItemSyncCache.
func (c *ContactsSyncEngine) SyncCollection(ctx *Context, dev *state.EasDevice, col eas.SyncCollection) (eas.SyncCollection, error) {
	if !parseVCardCollectionID(col.CollectionID) {
		return eas.SyncCollection{
			SyncKey:      col.SyncKey,
			CollectionID: col.CollectionID,
			Status:       eas.SyncStatusProtocolError,
		}, nil
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
		Class:        "Contacts",
		Status:       eas.SyncStatusSuccess,
	}

	var responses *eas.SyncCommands
	if col.Commands != nil {
		responses, err = c.applyClientCommands(ctx, col.Commands, &cache)
		if err != nil {
			return eas.SyncCollection{}, err
		}
	}

	window := syncWindow(col.WindowSize)
	if col.GetChanges != 0 || col.SyncKey == "0" {
		adds, changes, deletes, more, err := c.buildServerChanges(ctx.UserID, cache, window)
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

// EstimateCount returns the total number of contacts for the authenticated user.
// Used by GetItemEstimate for vcard/* collection IDs.
func (c *ContactsSyncEngine) EstimateCount(ctx *Context, collectionID string) (int32, error) {
	if !parseVCardCollectionID(collectionID) {
		return 0, fmt.Errorf("invalid contacts collection")
	}
	contacts, err := c.contactRepo.List(ctx.UserID)
	if err != nil {
		return 0, err
	}
	return int32(len(contacts)), nil
}

// applyClientCommands applies client-originated Sync Add/Change/Delete commands to SQL.
// Skips Add entries without Email1Address. Returns ClientId→ServerId mappings for new contacts.
func (c *ContactsSyncEngine) applyClientCommands(ctx *Context, cmds *eas.SyncCommands, cache *state.ItemSyncCache) (*eas.SyncCommands, error) {
	var responses eas.SyncCommands

	for _, add := range cmds.Add {
		ct, err := add.Contact()
		if err != nil {
			continue
		}
		contact := easContactToModel(ct, ctx.UserID)
		if contact.Email == "" {
			continue
		}
		if err := c.contactRepo.Create(contact); err != nil {
			return nil, err
		}
		sid := serverIDForUint(contact.ID)
		cache.Items[sid] = state.ItemSyncItem{UpdatedAt: contact.UpdatedAt.Unix()}
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
		ct, err := chg.Contact()
		if err != nil {
			continue
		}
		contact, err := c.contactRepo.Get(ctx.UserID, id)
		if err != nil {
			continue
		}
		applyEasContactToModel(ct, contact)
		if err := c.contactRepo.Update(contact); err != nil {
			return nil, err
		}
		cache.Items[chg.ServerID] = state.ItemSyncItem{UpdatedAt: contact.UpdatedAt.Unix()}
	}

	for _, del := range cmds.Delete {
		id, ok := parseServerID(del.ServerID)
		if !ok {
			continue
		}
		if err := c.contactRepo.Delete(ctx.UserID, id); err != nil {
			continue
		}
		delete(cache.Items, del.ServerID)
	}

	if len(responses.Add) == 0 {
		return nil, nil
	}
	return &responses, nil
}

// buildServerChanges compares SQL contacts with ItemSyncCache and builds Add/Change/Delete commands.
func (c *ContactsSyncEngine) buildServerChanges(userID uint, cache state.ItemSyncCache, window int) ([]eas.SyncAdd, []eas.SyncChange, []eas.SyncDelete, bool, error) {
	contacts, err := c.contactRepo.List(userID)
	if err != nil {
		return nil, nil, nil, false, err
	}

	current := make(map[string]state.ItemSyncItem, len(contacts))
	contactByID := make(map[string]model.Contact, len(contacts))
	for _, ct := range contacts {
		sid := serverIDForUint(ct.ID)
		current[sid] = state.ItemSyncItem{UpdatedAt: ct.UpdatedAt.Unix()}
		contactByID[sid] = ct
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
			ct := contactByID[p.serverID]
			add, err := c.buildSyncAdd(ct)
			if err != nil {
				return nil, nil, nil, false, err
			}
			adds = append(adds, add)
			cache.Items[p.serverID] = current[p.serverID]
		case "change":
			ct := contactByID[p.serverID]
			chg, err := c.buildSyncChange(ct)
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

// buildSyncAdd wraps a model.Contact as an EAS Sync Add with Contacts ApplicationData.
func (c *ContactsSyncEngine) buildSyncAdd(contact model.Contact) (eas.SyncAdd, error) {
	ct := modelContactToEas(contact)
	body, err := marshalApplicationDataBody(&ct)
	if err != nil {
		return eas.SyncAdd{}, err
	}
	return eas.SyncAdd{
		ServerID: serverIDForUint(contact.ID),
		ApplicationData: &wbxml.RawElement{
			Page:  wbxml.PageAirSync,
			Bytes: body,
		},
	}, nil
}

// buildSyncChange wraps an updated model.Contact as an EAS Sync Change command.
func (c *ContactsSyncEngine) buildSyncChange(contact model.Contact) (eas.SyncChange, error) {
	ct := modelContactToEas(contact)
	body, err := marshalApplicationDataBody(&ct)
	if err != nil {
		return eas.SyncChange{}, err
	}
	return eas.SyncChange{
		ServerID: serverIDForUint(contact.ID),
		ApplicationData: &wbxml.RawElement{
			Page:  wbxml.PageAirSync,
			Bytes: body,
		},
	}, nil
}

// modelContactToEas maps a model.Contact to an eas.Contact ApplicationData payload.
func modelContactToEas(c model.Contact) eas.Contact {
	return eas.Contact{
		FirstName:         c.FirstName,
		LastName:          c.LastName,
		Email1Address:     c.Email,
		Title:             c.Title,
		CompanyName:       c.Company,
		MobilePhoneNumber: c.Phone,
	}
}

// easContactToModel converts a client eas.Contact into a new model.Contact for Create.
// Email falls back to Email2Address when Email1Address is empty.
func easContactToModel(ct *eas.Contact, userID uint) *model.Contact {
	email := ct.Email1Address
	if email == "" {
		email = ct.Email2Address
	}
	return &model.Contact{
		UserID:    userID,
		FirstName: ct.FirstName,
		LastName:  ct.LastName,
		Email:     email,
		Title:     ct.Title,
		Company:   ct.CompanyName,
		Phone:     firstNonEmpty(ct.MobilePhoneNumber, ct.BusinessPhoneNumber, ct.HomePhoneNumber),
	}
}

// applyEasContactToModel merges non-empty eas.Contact fields into an existing model.Contact.
func applyEasContactToModel(ct *eas.Contact, contact *model.Contact) {
	if ct.FirstName != "" {
		contact.FirstName = ct.FirstName
	}
	if ct.LastName != "" {
		contact.LastName = ct.LastName
	}
	if ct.Email1Address != "" {
		contact.Email = ct.Email1Address
	}
	if ct.Title != "" {
		contact.Title = ct.Title
	}
	if ct.CompanyName != "" {
		contact.Company = ct.CompanyName
	}
	if phone := firstNonEmpty(ct.MobilePhoneNumber, ct.BusinessPhoneNumber, ct.HomePhoneNumber); phone != "" {
		contact.Phone = phone
	}
}

// firstNonEmpty returns the first non-empty string from the given values.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// parseVCardCollectionID reports whether collectionID is a supported contacts collection (vcard/*).
func parseVCardCollectionID(collectionID string) bool {
	return strings.HasPrefix(collectionID, "vcard/")
}
