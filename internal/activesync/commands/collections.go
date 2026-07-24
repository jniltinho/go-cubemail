package commands

// Mapping between EAS collection IDs and DAV collections.
//
// An EAS collection ID looks like "vevent/work" or "vcard/personal". The part
// after the slash is the DAV collection URI, so one calendar or address book is
// one device folder and the two protocols agree on what a collection is.
//
// "personal" is kept as an alias for the user's default collection. Earlier
// builds advertised exactly one calendar folder under that fixed ID, and every
// device already synced holds folder state keyed on it — resolving the alias
// keeps those devices working instead of forcing a full resync.

import (
	"strings"

	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
)

// Collection ID prefixes, one per item class.
const (
	prefixCalendar    = "vevent/"
	prefixContacts    = "vcard/"
	prefixTasks       = "vtodo/"
	defaultCollection = "personal"
)

// collectionURI extracts the DAV collection URI from an EAS collection ID.
func collectionURI(collectionID, prefix string) (string, bool) {
	if !strings.HasPrefix(collectionID, prefix) {
		return "", false
	}
	uri := strings.TrimPrefix(collectionID, prefix)
	if uri == "" || strings.ContainsAny(uri, "/\\") {
		return "", false
	}
	return uri, true
}

// calendarCollectionID renders the EAS collection ID of a calendar.
func calendarCollectionID(cal model.Calendar) string {
	if cal.IsDefault {
		return prefixCalendar + defaultCollection
	}
	return prefixCalendar + cal.URI
}

// addressBookCollectionID renders the EAS collection ID of an address book.
func addressBookCollectionID(book model.AddressBook) string {
	if book.IsDefault {
		return prefixContacts + defaultCollection
	}
	return prefixContacts + book.URI
}

// resolveCalendarCollection maps an EAS collection ID to a calendar.
func resolveCalendarCollection(repo *repository.CalendarRepo, userID uint, collectionID string) (*model.Calendar, bool) {
	uri, ok := collectionURI(collectionID, prefixCalendar)
	if !ok {
		return nil, false
	}
	if uri == defaultCollection {
		cal, err := repo.EnsureDefault(userID)
		if err != nil {
			return nil, false
		}
		return cal, true
	}
	cal, err := repo.GetByURI(userID, uri)
	if err != nil {
		return nil, false
	}
	return cal, true
}

// resolveAddressBookCollection maps an EAS collection ID to an address book.
func resolveAddressBookCollection(repo *repository.AddressBookRepo, userID uint, collectionID string) (*model.AddressBook, bool) {
	uri, ok := collectionURI(collectionID, prefixContacts)
	if !ok {
		return nil, false
	}
	if uri == defaultCollection {
		book, err := repo.EnsureDefault(userID)
		if err != nil {
			return nil, false
		}
		return book, true
	}
	book, err := repo.GetByURI(userID, uri)
	if err != nil {
		return nil, false
	}
	return book, true
}
