package commands

import (
	"fmt"

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
	store       *state.Store                // Sync keys and per-collection item cache.
	contactRepo *repository.ContactRepo     // CRUD for model.Contact rows.
	bookRepo    *repository.AddressBookRepo // Resolves vcard/* collection IDs.
}

// NewContactsSyncEngine creates a ContactsSyncEngine backed by the given store and repositories.
func NewContactsSyncEngine(store *state.Store, contactRepo *repository.ContactRepo, bookRepo *repository.AddressBookRepo) *ContactsSyncEngine {
	return &ContactsSyncEngine{store: store, contactRepo: contactRepo, bookRepo: bookRepo}
}

// SyncCollection processes one vcard/* collection in an MS-ASCMD Sync request.
//
// Flow: validate sync key → apply client Commands → build server changes when GetChanges is set
// → bump sync key and persist ItemSyncCache.
func (c *ContactsSyncEngine) SyncCollection(ctx *Context, dev *state.EasDevice, col eas.SyncCollection) (eas.SyncCollection, error) {
	book, ok := resolveAddressBookCollection(c.bookRepo, ctx.UserID, col.CollectionID)
	if !ok {
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
		responses, err = c.applyClientCommands(ctx, col.Commands, book.ID, &cache)
		if err != nil {
			return eas.SyncCollection{}, err
		}
	}

	window := syncWindow(col.WindowSize)
	if col.GetChanges != 0 || col.SyncKey == "0" {
		adds, changes, deletes, more, err := c.buildServerChanges(ctx.UserID, book.ID, cache, window)
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
	// Record the collection revision so Ping can skip the row scan while the
	// collection is quiet. Read after applying changes so a write that lands
	// mid-request is reported on the next Ping instead of being swallowed.
	if rev, err := c.bookRepo.Revision(ctx.UserID, book.ID); err == nil {
		cache.DavRevision = rev
	}
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
	book, ok := resolveAddressBookCollection(c.bookRepo, ctx.UserID, collectionID)
	if !ok {
		return 0, fmt.Errorf("invalid contacts collection")
	}
	contacts, err := c.contactRepo.ListByBook(ctx.UserID, book.ID)
	if err != nil {
		return 0, err
	}
	return int32(len(contacts)), nil
}

// applyClientCommands applies client-originated Sync Add/Change/Delete commands to SQL.
// Skips Add entries without Email1Address. Returns ClientId→ServerId mappings for new contacts.
func (c *ContactsSyncEngine) applyClientCommands(ctx *Context, cmds *eas.SyncCommands, bookID uint, cache *state.ItemSyncCache) (*eas.SyncCommands, error) {
	var responses eas.SyncCommands

	for _, add := range cmds.Add {
		ct, err := eas.UnmarshalApplicationData[easContactPayload](add.ApplicationData)
		if err != nil {
			continue
		}
		contact := easContactToModel(ct, ctx.UserID)
		contact.AddressBookID = bookID
		// A contact with a name but no e-mail address is perfectly valid, and
		// phones create them routinely. Dropping it here left the client's Add
		// without a ServerId, which most devices treat as a failure and retry
		// forever — the eas.SyncAdd response type has no Status field to report
		// the rejection with, so the only correct answer is to accept the item.
		if isEmptyContact(contact) {
			continue
		}
		if err := c.contactRepo.Create(contact); err != nil {
			return nil, err
		}
		sid := serverIDForUint(contact.ID)
		cache.Items[sid] = state.ItemSyncItem{
			UpdatedAt: contact.UpdatedAt.Unix(),
			Revision:  contact.SyncRevision,
		}
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
		ct, err := eas.UnmarshalApplicationData[easContactPayload](chg.ApplicationData)
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
		cache.Items[chg.ServerID] = state.ItemSyncItem{
			UpdatedAt: contact.UpdatedAt.Unix(),
			Revision:  contact.SyncRevision,
		}
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
func (c *ContactsSyncEngine) buildServerChanges(userID, bookID uint, cache state.ItemSyncCache, window int) ([]eas.SyncAdd, []eas.SyncChange, []eas.SyncDelete, bool, error) {
	contacts, err := c.contactRepo.ListByBook(userID, bookID)
	if err != nil {
		return nil, nil, nil, false, err
	}

	current := make(map[string]state.ItemSyncItem, len(contacts))
	contactByID := make(map[string]model.Contact, len(contacts))
	for _, ct := range contacts {
		sid := serverIDForUint(ct.ID)
		current[sid] = state.ItemSyncItem{
			UpdatedAt: ct.UpdatedAt.Unix(),
			Revision:  ct.SyncRevision,
		}
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
		if !prev.SameVersion(item) {
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

// easContactPayload is the ApplicationData of a contact.
//
// It mirrors eas.Contact and adds the AirSyncBase body element, which is where
// MS-ASCNTC carries a contact's notes — the library's Contact type models the
// scalar fields only, so notes could never reach the device without this.
type easContactPayload struct {
	XMLName             struct{}        `wbxml:"AirSync.ApplicationData"`
	FirstName           string          `wbxml:"Contacts.FirstName,omitempty"`
	LastName            string          `wbxml:"Contacts.LastName,omitempty"`
	MiddleName          string          `wbxml:"Contacts.MiddleName,omitempty"`
	Title               string          `wbxml:"Contacts.Title,omitempty"`
	CompanyName         string          `wbxml:"Contacts.CompanyName,omitempty"`
	JobTitle            string          `wbxml:"Contacts.JobTitle,omitempty"`
	Email1Address       string          `wbxml:"Contacts.Email1Address,omitempty"`
	Email2Address       string          `wbxml:"Contacts.Email2Address,omitempty"`
	Email3Address       string          `wbxml:"Contacts.Email3Address,omitempty"`
	HomePhoneNumber     string          `wbxml:"Contacts.HomePhoneNumber,omitempty"`
	MobilePhoneNumber   string          `wbxml:"Contacts.MobilePhoneNumber,omitempty"`
	BusinessPhoneNumber string          `wbxml:"Contacts.BusinessPhoneNumber,omitempty"`
	Body                *easAirSyncBody `wbxml:"AirSyncBase.Body,omitempty"`
}

// notes returns the contact's note text, if the payload carried one.
func (p *easContactPayload) notes() string {
	if p.Body == nil {
		return ""
	}
	return p.Body.Data
}

// jobTitle returns the occupation the client sent.
//
// MS-ASCNTC splits what this application stores as a single Title: JobTitle is
// the occupation, while Title is the honorific ("Dr."). Earlier builds wrote the
// occupation into Title, so that value is still accepted on the way in.
func (p *easContactPayload) jobTitle() string {
	return firstNonEmpty(p.JobTitle, p.Title)
}

// modelContactToEas maps a model.Contact to a contact ApplicationData payload.
func modelContactToEas(c model.Contact) easContactPayload {
	payload := easContactPayload{
		FirstName:         c.FirstName,
		LastName:          c.LastName,
		Email1Address:     c.Email,
		JobTitle:          c.Title,
		CompanyName:       c.Company,
		MobilePhoneNumber: c.Phone,
	}
	if c.Notes != "" {
		payload.Body = &easAirSyncBody{
			Type:              bodyTypePlain,
			Data:              c.Notes,
			EstimatedDataSize: int32(len(c.Notes)),
		}
	}
	return payload
}

// easContactToModel converts a client payload into a new model.Contact for Create.
// Email falls back to Email2Address when Email1Address is empty.
func easContactToModel(ct *easContactPayload, userID uint) *model.Contact {
	email := firstNonEmpty(ct.Email1Address, ct.Email2Address, ct.Email3Address)
	return &model.Contact{
		UserID:    userID,
		FirstName: ct.FirstName,
		LastName:  ct.LastName,
		Email:     email,
		Title:     ct.jobTitle(),
		Company:   ct.CompanyName,
		Phone:     firstNonEmpty(ct.MobilePhoneNumber, ct.BusinessPhoneNumber, ct.HomePhoneNumber),
		Notes:     ct.notes(),
	}
}

// applyEasContactToModel copies a client payload onto an existing model.Contact.
//
// Every mapped field is overwritten, including with an empty value. An MS-ASCMD
// Change carries the complete item, and the payload cannot distinguish "field
// omitted" from "field cleared" — both decode to the empty string. Merging only
// non-empty values, as this used to, meant a field cleared on the phone came
// back on the next sync and could never be deleted.
//
// Properties outside this flat model are unaffected: the stored vCard is patched
// rather than regenerated, so addresses, photos and extra numbers survive.
func applyEasContactToModel(ct *easContactPayload, contact *model.Contact) {
	contact.FirstName = ct.FirstName
	contact.LastName = ct.LastName
	contact.Email = firstNonEmpty(ct.Email1Address, ct.Email2Address, ct.Email3Address)
	contact.Title = ct.jobTitle()
	contact.Company = ct.CompanyName
	contact.Phone = firstNonEmpty(ct.MobilePhoneNumber, ct.BusinessPhoneNumber, ct.HomePhoneNumber)
	contact.Notes = ct.notes()
}

// isEmptyContact reports that a client payload carried nothing worth storing.
func isEmptyContact(c *model.Contact) bool {
	return c.FirstName == "" && c.LastName == "" && c.Email == "" &&
		c.Phone == "" && c.Company == ""
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
