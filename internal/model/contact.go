package model

import "time"

// Contact stores a single address-book entry belonging to a user.
// GroupID optionally links the contact to a ContactGroup.
//
// VCardContent holds the original vCard blob byte-for-byte as received from a
// CardDAV client; the flat fields below are a denormalized index for the web UI
// and ActiveSync. Edits from the web UI patch the blob in place (see
// internal/contacts.ApplyToVCard) instead of regenerating it, so properties the
// flat model cannot express — ADR, PHOTO, BDAY, extra EMAIL/TEL, X-* — survive
// a round trip.
type Contact struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"index;not null"`
	UID       string `gorm:"size:255;index"` // vCard UID for CardDAV round-trip
	FirstName string
	LastName  string
	Email     string `gorm:"not null"`
	Title     string
	Phone     string
	Company   string
	Notes     string
	GroupID   *uint
	CreatedAt time.Time
	UpdatedAt time.Time

	// ── CardDAV resource fields ───────────────────────────────────────────
	AddressBookID uint   `gorm:"index;uniqueIndex:idx_contacts_ab_uri,priority:1"`
	ResourceURI   string `gorm:"size:255;uniqueIndex:idx_contacts_ab_uri,priority:2"` // "a1b2c3.vcf"
	VCardContent  string `gorm:"column:vcard_content;type:text"`                      // raw vCard, never normalised
	ETag          string `gorm:"column:etag;size:70"`                                  // sha256 prefix, unquoted
	SyncRevision  uint64 `gorm:"index"`
}
