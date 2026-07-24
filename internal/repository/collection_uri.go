package repository

import (
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// Slugify converts a display name to a URL-safe DAV path segment, falling back
// to the given default when nothing usable remains.
//
// The result is only a starting point: collection URIs are stored, not derived
// on the fly, so renaming a collection never changes its DAV URLs.
func Slugify(name, fallback string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return fallback
	}
	if len(s) > 100 {
		s = s[:100]
	}
	return s
}

// freeCollectionURI returns base, or base-2, base-3… until no collection of the
// same kind owned by the user holds that URI.
func freeCollectionURI(db *gorm.DB, m any, userID uint, base string) string {
	candidate := base
	for i := 2; i < 1000; i++ {
		var count int64
		if err := db.Model(m).
			Where("user_id = ? AND uri = ?", userID, candidate).
			Count(&count).Error; err != nil || count == 0 {
			return candidate
		}
		candidate = base + "-" + strconv.Itoa(i)
	}
	return base + "-" + strconv.Itoa(int(userID))
}
