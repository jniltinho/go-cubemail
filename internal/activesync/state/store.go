// Package state provides GORM-backed persistence for ActiveSync device sync state.
package state

import (
	"encoding/json"
	"fmt"
	"strconv"

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

// NextCollectionSyncKey increments the per-collection sync key and returns the new value.
func (s *Store) NextCollectionSyncKey(deviceID uint, collectionID string) (string, error) {
	cur, err := s.GetCollectionSyncKey(deviceID, collectionID)
	if err != nil {
		return "", err
	}
	n, _ := strconv.ParseUint(cur, 10, 64)
	n++
	next := strconv.FormatUint(n, 10)
	if err := s.SetCollectionSyncKey(deviceID, collectionID, next); err != nil {
		return "", err
	}
	return next, nil
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
