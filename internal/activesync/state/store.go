// Package state provides GORM-backed persistence for ActiveSync device sync state.
package state

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// Store reads and writes EasDevice and EasFolderState records.
type Store struct {
	db *gorm.DB
}

// NewStore creates a Store backed by the given database connection.
func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// EnsureDevice returns an existing device row or creates a new one.
func (s *Store) EnsureDevice(userID uint, deviceID, deviceType string) (*EasDevice, error) {
	var dev EasDevice
	err := s.db.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&dev).Error
	if err == nil {
		if deviceType != "" && dev.DeviceType != deviceType {
			dev.DeviceType = deviceType
			_ = s.db.Save(&dev).Error
		}
		return &dev, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	dev = EasDevice{
		UserID:        userID,
		DeviceID:      deviceID,
		DeviceType:    deviceType,
		FolderSyncKey: "0",
		PolicyKey:     1,
	}
	if err := s.db.Create(&dev).Error; err != nil {
		return nil, err
	}
	return &dev, nil
}

// SaveDevice persists changes to a device record.
func (s *Store) SaveDevice(dev *EasDevice) error {
	return s.db.Save(dev).Error
}

// NextFolderSyncKey increments and saves the device folder sync key.
func (s *Store) NextFolderSyncKey(dev *EasDevice) string {
	n, _ := strconv.ParseUint(dev.FolderSyncKey, 10, 64)
	n++
	dev.FolderSyncKey = strconv.FormatUint(n, 10)
	_ = s.db.Save(dev).Error
	return dev.FolderSyncKey
}

// GetCollectionSyncKey returns the sync key for a collection, defaulting to "0".
func (s *Store) GetCollectionSyncKey(deviceID uint, collectionID string) (string, error) {
	var st EasFolderState
	err := s.db.Where("eas_device_id = ? AND collection_id = ?", deviceID, collectionID).First(&st).Error
	if err == gorm.ErrRecordNotFound {
		return "0", nil
	}
	if err != nil {
		return "", err
	}
	return st.SyncKey, nil
}

// SetCollectionSyncKey upserts the sync key for a collection.
func (s *Store) SetCollectionSyncKey(deviceID uint, collectionID, syncKey string) error {
	var st EasFolderState
	err := s.db.Where("eas_device_id = ? AND collection_id = ?", deviceID, collectionID).First(&st).Error
	if err == gorm.ErrRecordNotFound {
		st = EasFolderState{
			EasDeviceID:  deviceID,
			CollectionID: collectionID,
			SyncKey:      syncKey,
		}
		return s.db.Create(&st).Error
	}
	if err != nil {
		return err
	}
	st.SyncKey = syncKey
	return s.db.Save(&st).Error
}

// syncKeyCASAttempts bounds the compare-and-swap retry loop.
//
// Each round at least one contender wins, so a caller needs about as many
// attempts as there are concurrent writers on the same collection. In practice
// that is one or two — a device waits for its Sync response before issuing the
// next request for a collection — but a client that retries after a timeout can
// briefly double up, and the bound is cheap.
const syncKeyCASAttempts = 50

// syncKeyRetryBackoff spaces out retries so contenders stop colliding in
// lockstep, which is what turns a busy collection into repeated CAS failures.
const syncKeyRetryBackoff = 200 * time.Microsecond

// NextCollectionSyncKey increments the per-collection sync key and returns the
// new value.
//
// The increment is a compare-and-swap rather than a read-modify-write: the
// UPDATE only applies while the key still holds the value that was read, so two
// concurrent Sync requests for the same collection — which devices do pipeline —
// cannot both be handed the same key. Handing out a duplicate key would make the
// server accept a stale request as current and silently drop a batch of changes.
//
// A plain transaction would not be enough on its own: without row locking, both
// readers would still see the same starting value.
func (s *Store) NextCollectionSyncKey(deviceID uint, collectionID string) (string, error) {
	for attempt := 0; attempt < syncKeyCASAttempts; attempt++ {
		var st EasFolderState
		err := s.db.Where("eas_device_id = ? AND collection_id = ?", deviceID, collectionID).
			First(&st).Error

		if err == gorm.ErrRecordNotFound {
			// First sync for this collection. A unique index on
			// (device, collection) makes a lost race here a duplicate-key
			// error, which the retry resolves by taking the update path.
			st = EasFolderState{
				EasDeviceID:  deviceID,
				CollectionID: collectionID,
				SyncKey:      "1",
			}
			if err := s.db.Create(&st).Error; err != nil {
				continue
			}
			return st.SyncKey, nil
		}
		if err != nil {
			return "", err
		}

		n, _ := strconv.ParseUint(st.SyncKey, 10, 64)
		next := strconv.FormatUint(n+1, 10)

		res := s.db.Model(&EasFolderState{}).
			Where("id = ? AND sync_key = ?", st.ID, st.SyncKey).
			Update("sync_key", next)
		if res.Error != nil {
			return "", res.Error
		}
		if res.RowsAffected == 1 {
			return next, nil
		}
		// Another request advanced the key first; re-read and try again.
		time.Sleep(syncKeyRetryBackoff)
	}
	return "", fmt.Errorf("sync key for collection %q is contended", collectionID)
}

// FolderGUID returns a stable EAS folder ID for an IMAP mailbox path.
func (s *Store) FolderGUID(userID uint, folderPath string) (string, error) {
	var m ImapFolderMapping
	err := s.db.Where("user_id = ? AND folder_path = ?", userID, folderPath).First(&m).Error
	if err == nil {
		return m.FolderGUID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return "", err
	}
	guid := fmt.Sprintf("%016x", hashFolder(userID, folderPath))
	m = ImapFolderMapping{
		UserID:     userID,
		FolderPath: folderPath,
		FolderGUID: guid,
	}
	if err := s.db.Create(&m).Error; err != nil {
		return "", err
	}
	return guid, nil
}

// GetFolderState returns the folder state row for a device collection.
func (s *Store) GetFolderState(deviceID uint, collectionID string) (*EasFolderState, error) {
	var st EasFolderState
	err := s.db.Where("eas_device_id = ? AND collection_id = ?", deviceID, collectionID).First(&st).Error
	if err == gorm.ErrRecordNotFound {
		return &EasFolderState{
			EasDeviceID:  deviceID,
			CollectionID: collectionID,
			SyncKey:      "0",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// SaveFolderState persists a folder state row (insert or update).
func (s *Store) SaveFolderState(st *EasFolderState) error {
	if st.ID == 0 {
		return s.db.Create(st).Error
	}
	return s.db.Save(st).Error
}

// FolderPathByGUID resolves a stable folder GUID back to an IMAP mailbox path.
func (s *Store) FolderPathByGUID(userID uint, guid string) (string, error) {
	var m ImapFolderMapping
	err := s.db.Where("user_id = ? AND folder_guid = ?", userID, guid).First(&m).Error
	if err != nil {
		return "", err
	}
	return m.FolderPath, nil
}

// MailSyncCache tracks synced mail UIDs and flags for change detection.
type MailSyncCache struct {
	Items map[string]MailSyncItem `json:"items"`
}

// MailSyncItem holds the last-known IMAP state for one synced message.
type MailSyncItem struct {
	UID     uint32 `json:"uid"`
	Seen    bool   `json:"seen"`
	Flagged bool   `json:"flagged"`
}

// LoadMailSyncCache decodes SyncCache JSON or returns an empty cache.
func LoadMailSyncCache(raw string) MailSyncCache {
	if raw == "" {
		return MailSyncCache{Items: map[string]MailSyncItem{}}
	}
	var c MailSyncCache
	if err := json.Unmarshal([]byte(raw), &c); err != nil || c.Items == nil {
		return MailSyncCache{Items: map[string]MailSyncItem{}}
	}
	return c
}

// EncodeMailSyncCache serializes a mail sync cache to JSON.
func EncodeMailSyncCache(c MailSyncCache) string {
	if c.Items == nil {
		c.Items = map[string]MailSyncItem{}
	}
	b, _ := json.Marshal(c)
	return string(b)
}

// hashFolder computes a stable FNV-1a hash for (userID, folderPath) used when creating folder GUIDs.
func hashFolder(userID uint, path string) uint64 {
	var h uint64 = 14695981039346656037
	for _, b := range []byte(fmt.Sprintf("%d:%s", userID, path)) {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}
