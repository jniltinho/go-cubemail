// Package repository provides database access objects for go-cubemail models.
// All repositories enforce user-ID scoping so queries never cross user boundaries.
package repository

import (
	"go-cubemail/internal/contacts"
	"go-cubemail/internal/dav"
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// ContactRepo provides CRUD operations for the Contact model.
type ContactRepo struct {
	db    *gorm.DB
	books *AddressBookRepo
}

// NewContactRepo creates a ContactRepo backed by the given database connection.
func NewContactRepo(db *gorm.DB) *ContactRepo {
	return &ContactRepo{db: db, books: NewAddressBookRepo(db)}
}

// List returns all contacts for the given user, ordered by first and last name.
func (r *ContactRepo) List(userID uint) ([]model.Contact, error) {
	var list []model.Contact
	err := r.db.Where("user_id = ?", userID).Order("first_name, last_name").Find(&list).Error
	return list, err
}

// ListByBook returns the contacts of one address book.
func (r *ContactRepo) ListByBook(userID, bookID uint) ([]model.Contact, error) {
	var list []model.Contact
	err := r.db.Where("user_id = ? AND address_book_id = ?", userID, bookID).
		Order("first_name, last_name").Find(&list).Error
	return list, err
}

// ListSince returns the contacts of an address book written after the given
// revision, used to answer sync-collection REPORTs.
func (r *ContactRepo) ListSince(userID, bookID uint, since uint64) ([]model.Contact, error) {
	var list []model.Contact
	err := r.db.Where("user_id = ? AND address_book_id = ? AND sync_revision > ?",
		userID, bookID, since).
		Order("sync_revision").Find(&list).Error
	return list, err
}

// Search returns contacts whose first name, last name, or email contain the query string.
func (r *ContactRepo) Search(userID uint, q string) ([]model.Contact, error) {
	var list []model.Contact
	like := "%" + q + "%"
	err := r.db.Where("user_id = ? AND (first_name LIKE ? OR last_name LIKE ? OR email LIKE ?)",
		userID, like, like, like).Find(&list).Error
	return list, err
}

// Get retrieves a single contact by ID, scoped to the given user.
func (r *ContactRepo) Get(userID, id uint) (*model.Contact, error) {
	var c model.Contact
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&c).Error
	return &c, err
}

// GetByUID retrieves a contact by vCard UID, scoped to the given user.
func (r *ContactRepo) GetByUID(userID uint, uid string) (*model.Contact, error) {
	var c model.Contact
	err := r.db.Where("user_id = ? AND uid = ?", userID, uid).First(&c).Error
	return &c, err
}

// GetByResourceURI retrieves a contact by its DAV resource name within an
// address book — the identity CardDAV clients address.
func (r *ContactRepo) GetByResourceURI(userID, bookID uint, uri string) (*model.Contact, error) {
	var c model.Contact
	err := r.db.Where("user_id = ? AND address_book_id = ? AND resource_uri = ?",
		userID, bookID, uri).First(&c).Error
	return &c, err
}

// Create inserts a new contact, provisioning its address book and vCard blob
// when the caller did not supply them.
func (r *ContactRepo) Create(c *model.Contact) error {
	if err := r.prepare(c); err != nil {
		return err
	}
	if c.VCardContent == "" {
		c.VCardContent = contacts.Build(*c)
	}
	return r.save(c, true)
}

// Update persists changes to an existing contact.
//
// The stored vCard is patched rather than regenerated so properties the flat
// model cannot represent survive an edit made from the web UI.
func (r *ContactRepo) Update(c *model.Contact) error {
	if err := r.prepare(c); err != nil {
		return err
	}
	c.VCardContent = contacts.ApplyToVCard(c.VCardContent, *c)
	return r.save(c, false)
}

// SaveRaw stores a contact whose VCardContent came straight from a CardDAV
// client, keeping the blob byte-for-byte as received. The indexed columns must
// already have been filled by the caller from that same blob.
func (r *ContactRepo) SaveRaw(c *model.Contact) error {
	if err := r.prepare(c); err != nil {
		return err
	}
	return r.save(c, c.ID == 0)
}

// prepare fills the address book, UID and resource name of a contact.
func (r *ContactRepo) prepare(c *model.Contact) error {
	if c.AddressBookID == 0 {
		book, err := r.books.EnsureDefault(c.UserID)
		if err != nil {
			return err
		}
		c.AddressBookID = book.ID
	}
	if c.UID == "" {
		c.UID = dav.NewResourceURI("") + "@go-cubemail"
	}
	if c.ResourceURI == "" {
		c.ResourceURI = dav.NewResourceURI(".vcf")
	}
	return nil
}

// save writes the contact and its changelog entry in one transaction.
func (r *ContactRepo) save(c *model.Contact, create bool) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		c.ETag = dav.ComputeETag([]byte(c.VCardContent))
		rev, err := dav.RecordIfExists(tx, model.CollectionAddressBook,
			c.AddressBookID, c.ResourceURI, false)
		if err != nil {
			return err
		}
		if rev > 0 {
			c.SyncRevision = rev
		}
		if create {
			return tx.Create(c).Error
		}
		return tx.Save(c).Error
	})
}

// Delete removes a contact by ID, scoped to the given user, and records a
// tombstone so CardDAV clients learn about the removal.
func (r *ContactRepo) Delete(userID, id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var c model.Contact
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&c).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Contact{}, c.ID).Error; err != nil {
			return err
		}
		_, err := dav.RecordIfExists(tx, model.CollectionAddressBook,
			c.AddressBookID, c.ResourceURI, true)
		return err
	})
}
