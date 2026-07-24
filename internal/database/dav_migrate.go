package database

// Schema preparation for the DAV synchronisation layer.
//
// The DAV columns carry unique indexes — (calendar_id, resource_uri) and
// (address_book_id, resource_uri) — which cannot be created while existing rows
// all hold the empty default. AutoMigrate adds columns and indexes in one pass,
// so the values have to exist first: PrepareDAVSchema adds the bare columns and
// fills them, and only then is AutoMigrate allowed to build the indexes.

import (
	"fmt"
	"log/slog"

	"go-cubemail/internal/dav"
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// PrepareDAVSchema adds and backfills the DAV columns on pre-existing tables.
// It is a no-op on a fresh database, where AutoMigrate creates everything at
// once, and is safe to run repeatedly.
func PrepareDAVSchema(db *gorm.DB) error {
	m := db.Migrator()

	// The old schema declared events.uid UNIQUE, which is wrong: a recurrence
	// override shares the master's UID, and two users may legitimately import
	// the same public calendar. Drop it before the new composite index lands.
	if m.HasTable(&model.Event{}) && m.HasIndex(&model.Event{}, "idx_events_uid") {
		if err := m.DropIndex(&model.Event{}, "idx_events_uid"); err != nil {
			slog.Warn("Could not drop legacy unique index on events.uid", "error", err)
		}
	}

	if err := addColumns(db, &model.Event{}, "ResourceURI", "ETag", "SyncRevision"); err != nil {
		return err
	}
	if err := addColumns(db, &model.Contact{},
		"AddressBookID", "ResourceURI", "VCardContent", "ETag", "SyncRevision"); err != nil {
		return err
	}
	if err := addColumns(db, &model.Calendar{},
		"URI", "Description", "TimeZone", "SyncToken", "PrunedRevision"); err != nil {
		return err
	}

	if err := backfillResourceURIs(db, "events", ".ics"); err != nil {
		return err
	}
	if err := backfillResourceURIs(db, "contacts", ".vcf"); err != nil {
		return err
	}
	return nil
}

// addColumns adds any of the named model fields that the table is missing.
func addColumns(db *gorm.DB, m any, fields ...string) error {
	migrator := db.Migrator()
	if !migrator.HasTable(m) {
		return nil // fresh install: AutoMigrate will create the whole table
	}
	for _, f := range fields {
		if migrator.HasColumn(m, f) {
			continue
		}
		if err := migrator.AddColumn(m, f); err != nil {
			return fmt.Errorf("add column %s: %w", f, err)
		}
	}
	return nil
}

// backfillResourceURIs assigns a DAV resource name to rows that predate it.
func backfillResourceURIs(db *gorm.DB, table, ext string) error {
	if !db.Migrator().HasTable(table) {
		return nil
	}
	var ids []uint
	if err := db.Table(table).
		Where("resource_uri IS NULL OR resource_uri = ''").
		Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := db.Table(table).Where("id = ?", id).
			Update("resource_uri", dav.NewResourceURI(ext)).Error; err != nil {
			return err
		}
	}
	if len(ids) > 0 {
		slog.Info("Assigned DAV resource names", "table", table, "rows", len(ids))
	}
	return nil
}

// FinishDAVMigration fills the derived DAV state that needs the full schema in
// place: default address books, the address book of every contact, stable
// calendar URIs, ETags and initial sync tokens.
func FinishDAVMigration(db *gorm.DB) error {
	// Repeat the resource-name backfill. AutoMigrate rebuilds SQLite tables to
	// alter a column, and that rebuild copies only the columns its DDL parser
	// recognised — a hand-edited schema can therefore arrive here with the
	// names blanked out. The pass is idempotent, so this only ever fills gaps.
	if err := backfillResourceURIs(db, "events", ".ics"); err != nil {
		return err
	}
	if err := backfillResourceURIs(db, "contacts", ".vcf"); err != nil {
		return err
	}
	if err := ensureDAVIndexes(db); err != nil {
		return err
	}
	if err := seedCollectionTokens(db); err != nil {
		return err
	}
	if err := assignCalendarURIs(db); err != nil {
		return err
	}
	if err := assignAddressBooks(db); err != nil {
		return err
	}
	return backfillETags(db)
}

// ensureDAVIndexes creates the resource-identity indexes if AutoMigrate could
// not: on SQLite a table rebuild can drop them, and they are what keeps two
// objects from claiming the same URL inside one collection.
func ensureDAVIndexes(db *gorm.DB) error {
	m := db.Migrator()
	for _, idx := range []struct {
		model any
		name  string
	}{
		{&model.Event{}, "idx_events_cal_uri"},
		{&model.Contact{}, "idx_contacts_ab_uri"},
	} {
		if !m.HasTable(idx.model) || m.HasIndex(idx.model, idx.name) {
			continue
		}
		if err := m.CreateIndex(idx.model, idx.name); err != nil {
			// A duplicate left over from a partially migrated database must not
			// stop the upgrade; log it and let the operator resolve it.
			slog.Warn("Could not create DAV uniqueness index",
				"index", idx.name, "error", err)
		}
	}
	return nil
}

// seedCollectionTokens makes sure every collection starts at revision 1 rather
// than 0, so a client that has synced can never present a token equal to the
// "initial sync" sentinel.
func seedCollectionTokens(db *gorm.DB) error {
	if err := db.Model(&model.Calendar{}).Where("sync_token IS NULL OR sync_token = 0").
		Update("sync_token", 1).Error; err != nil {
		return err
	}
	return db.Model(&model.AddressBook{}).Where("sync_token IS NULL OR sync_token = 0").
		Update("sync_token", 1).Error
}

// assignCalendarURIs gives calendars created before DAV collections a stable URI.
func assignCalendarURIs(db *gorm.DB) error {
	var cals []model.Calendar
	if err := db.Where("uri IS NULL OR uri = ''").Find(&cals).Error; err != nil {
		return err
	}
	used := make(map[string]bool)
	for _, cal := range cals {
		uri := "default"
		if !cal.IsDefault {
			uri = slugForCalendar(cal, used)
		}
		key := fmt.Sprintf("%d/%s", cal.UserID, uri)
		if used[key] {
			uri = fmt.Sprintf("%s-%d", uri, cal.ID)
		}
		used[fmt.Sprintf("%d/%s", cal.UserID, uri)] = true
		if err := db.Model(&model.Calendar{}).Where("id = ?", cal.ID).
			Update("uri", uri).Error; err != nil {
			return err
		}
	}
	return nil
}

// slugForCalendar derives a URL-safe segment from the calendar name, falling
// back to the row ID when the name yields nothing usable.
func slugForCalendar(cal model.Calendar, used map[string]bool) string {
	slug := ""
	for _, r := range cal.Name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			slug += string(r)
		case r >= 'A' && r <= 'Z':
			slug += string(r + 32)
		case r == ' ':
			slug += "-"
		}
	}
	if slug == "" {
		slug = fmt.Sprintf("calendar-%d", cal.ID)
	}
	return slug
}

// assignAddressBooks provisions a default address book per user holding
// contacts and points every orphan contact at it.
func assignAddressBooks(db *gorm.DB) error {
	var userIDs []uint
	if err := db.Model(&model.Contact{}).
		Where("address_book_id IS NULL OR address_book_id = 0").
		Distinct().Pluck("user_id", &userIDs).Error; err != nil {
		return err
	}
	for _, userID := range userIDs {
		var book model.AddressBook
		err := db.Where("user_id = ? AND is_default = ?", userID, true).First(&book).Error
		if err == gorm.ErrRecordNotFound {
			book = model.AddressBook{
				UserID: userID, URI: "default", DisplayName: "Contacts",
				IsDefault: true, SyncToken: 1,
			}
			if err := db.Create(&book).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if err := db.Model(&model.Contact{}).
			Where("user_id = ? AND (address_book_id IS NULL OR address_book_id = 0)", userID).
			Update("address_book_id", book.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

// backfillETags computes the missing entity tags from the stored blobs, so the
// first conditional request from a client does not have to recompute them.
func backfillETags(db *gorm.DB) error {
	var events []model.Event
	if err := db.Where("(etag IS NULL OR etag = '') AND i_cal_content != ''").
		Find(&events).Error; err != nil {
		return err
	}
	for _, ev := range events {
		if err := db.Model(&model.Event{}).Where("id = ?", ev.ID).
			Update("etag", dav.ComputeETag([]byte(ev.ICalContent))).Error; err != nil {
			return err
		}
	}

	var list []model.Contact
	if err := db.Where("etag IS NULL OR etag = ''").Find(&list).Error; err != nil {
		return err
	}
	for _, ct := range list {
		if ct.VCardContent == "" {
			continue
		}
		if err := db.Model(&model.Contact{}).Where("id = ?", ct.ID).
			Update("etag", dav.ComputeETag([]byte(ct.VCardContent))).Error; err != nil {
			return err
		}
	}
	return nil
}
