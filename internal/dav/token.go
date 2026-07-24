package dav

import (
	"strconv"
	"strings"
)

// tokenPrefix namespaces the opaque sync-token. RFC 6578 only requires the
// value to be opaque to the client; a URI form avoids collisions with tokens
// minted by other servers a client may have talked to before.
const tokenPrefix = "http://go-cubemail.local/ns/sync/"

// SyncToken renders a collection revision as the opaque DAV:sync-token value.
func SyncToken(rev uint64) string {
	return tokenPrefix + strconv.FormatUint(rev, 10)
}

// ParseSyncToken extracts the revision from a client-supplied DAV:sync-token.
// An empty token means "initial sync" and yields (0, true). A bare decimal is
// accepted so tokens minted by older builds keep working.
func ParseSyncToken(token string) (uint64, bool) {
	t := strings.TrimSpace(token)
	if t == "" {
		return 0, true
	}
	t = strings.TrimPrefix(t, tokenPrefix)
	rev, err := strconv.ParseUint(t, 10, 64)
	if err != nil {
		return 0, false
	}
	return rev, true
}

// CTag renders a collection revision as the calendarserver.org getctag value.
// Deriving it from the same counter as the sync-token keeps the two consistent:
// any change that advances one advances the other.
func CTag(rev uint64) string {
	return strconv.FormatUint(rev, 10)
}
