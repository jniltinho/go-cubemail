package state

import "encoding/json"

// ItemSyncCache tracks synced SQL item versions for calendar and contacts collections.
//
// Stored JSON in EasFolderState.SyncCache. Keys are EAS ServerId strings (decimal DB IDs).
// Values hold the last-known UpdatedAt unix timestamp used to detect Add/Change/Delete.
type ItemSyncCache struct {
	Items map[string]ItemSyncItem `json:"items"`
}

// ItemSyncItem holds the last-known UpdatedAt unix timestamp for one synced item.
type ItemSyncItem struct {
	UpdatedAt int64 `json:"updated_at"`
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
