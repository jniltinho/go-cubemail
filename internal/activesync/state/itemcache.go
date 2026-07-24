package state

import "encoding/json"

// ItemSyncCache tracks synced SQL item versions for calendar and contacts collections.
//
// Stored JSON in EasFolderState.SyncCache. Keys are EAS ServerId strings (decimal DB IDs).
// Values hold the last-known UpdatedAt unix timestamp used to detect Add/Change/Delete.
type ItemSyncCache struct {
	Items map[string]ItemSyncItem `json:"items"`

	// DavRevision is the collection sync token observed at the last Sync.
	// Ping uses it as a fast path: when the token has not moved, nothing in the
	// collection can have changed and no row scan is needed. It is an
	// optimisation only — the Items comparison stays authoritative, so a stale
	// or absent value costs a query, never correctness.
	DavRevision uint64 `json:"dav_revision,omitempty"`
}

// ItemSyncItem holds the last-known version of one synced item.
//
// Revision is the DAV collection revision of the item's last write. It is
// strictly monotonic, so unlike UpdatedAt it separates two edits made within the
// same second. Items outside any DAV collection carry zero and fall back to the
// timestamp; so do caches written before this field existed.
type ItemSyncItem struct {
	UpdatedAt int64  `json:"updated_at"`
	Revision  uint64 `json:"revision,omitempty"`
}

// SameVersion reports whether two records describe the same version of an item.
// The revision wins when both sides have one; otherwise the timestamp decides.
func (i ItemSyncItem) SameVersion(other ItemSyncItem) bool {
	if i.Revision != 0 && other.Revision != 0 {
		return i.Revision == other.Revision
	}
	return i.UpdatedAt == other.UpdatedAt
}

// LoadItemSyncCache decodes SyncCache JSON from EasFolderState.
// Returns an empty cache when raw is blank or JSON is invalid.
func LoadItemSyncCache(raw string) ItemSyncCache {
	if raw == "" {
		return ItemSyncCache{Items: map[string]ItemSyncItem{}}
	}
	var c ItemSyncCache
	if err := json.Unmarshal([]byte(raw), &c); err != nil || c.Items == nil {
		return ItemSyncCache{Items: map[string]ItemSyncItem{}}
	}
	return c
}

// EncodeItemSyncCache serializes an ItemSyncCache to JSON for EasFolderState.SyncCache.
func EncodeItemSyncCache(c ItemSyncCache) string {
	if c.Items == nil {
		c.Items = map[string]ItemSyncItem{}
	}
	b, _ := json.Marshal(c)
	return string(b)
}
