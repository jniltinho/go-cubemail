package repository

import (
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// AddressBookRepo provides CRUD operations for CardDAV address book collections.
type AddressBookRepo struct {
	db *gorm.DB
}

// NewAddressBookRepo creates an AddressBookRepo backed by the given connection.
func NewAddressBookRepo(db *gorm.DB) *AddressBookRepo {
	return &AddressBookRepo{db: db}
}

// List returns all address books for a user, default first.
func (r *AddressBookRepo) List(userID uint) ([]model.AddressBook, error) {
	var books []model.AddressBook
	err := r.db.Where("user_id = ?", userID).
		Order("is_default DESC, display_name").
		Find(&books).Error
	return books, err
}

// Get retrieves an address book by ID, scoped to the user.
func (r *AddressBookRepo) Get(userID, id uint) (*model.AddressBook, error) {
	var ab model.AddressBook
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&ab).Error
	return &ab, err
}

// GetByURI resolves an address book from its DAV path segment.
func (r *AddressBookRepo) GetByURI(userID uint, uri string) (*model.AddressBook, error) {
	var ab model.AddressBook
	err := r.db.Where("user_id = ? AND uri = ?", userID, uri).First(&ab).Error
	return &ab, err
}

// Create inserts a new address book, assigning a free URI when none is given.
func (r *AddressBookRepo) Create(ab *model.AddressBook) error {
	if ab.SyncToken == 0 {
		ab.SyncToken = 1
	}
	if ab.URI == "" {
		ab.URI = r.freeURI(ab.UserID, Slugify(ab.DisplayName, "addressbook"))
	}
	return r.db.Create(ab).Error
}

// Update persists changes to an address book.
func (r *AddressBookRepo) Update(ab *model.AddressBook) error {
	return r.db.Save(ab).Error
}

// Delete removes an address book, its contacts and its changelog entries.
func (r *AddressBookRepo) Delete(userID, id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND address_book_id = ?", userID, id).
			Delete(&model.Contact{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ? AND user_id = ?", id, userID).Delete(&model.AddressBook{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("collection_kind = ? AND collection_id = ?",
			model.CollectionAddressBook, id).Delete(&model.DAVChange{}).Error
	})
}

// EnsureDefault returns the user's default address book, provisioning it on
// first access the same way CalendarRepo.EnsureDefault does for calendars.
func (r *AddressBookRepo) EnsureDefault(userID uint) (*model.AddressBook, error) {
	var ab model.AddressBook
	err := r.db.Where("user_id = ? AND is_default = ?", userID, true).First(&ab).Error
	if err == nil {
		return &ab, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	ab = model.AddressBook{
		UserID:      userID,
		URI:         "default",
		DisplayName: "Contacts",
		IsDefault:   true,
		SyncToken:   1,
	}
	if err := r.db.Create(&ab).Error; err != nil {
		return nil, err
	}
	return &ab, nil
}

// freeURI returns base, or base-2, base-3… when the URI is already taken.
func (r *AddressBookRepo) freeURI(userID uint, base string) string {
	return freeCollectionURI(r.db, &model.AddressBook{}, userID, base)
}
