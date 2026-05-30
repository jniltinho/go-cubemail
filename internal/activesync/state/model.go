package state

import "time"

// EasDevice stores per-user ActiveSync device state (folder sync key, policy key).
type EasDevice struct {
	ID            uint      `gorm:"primaryKey"`
	UserID        uint      `gorm:"uniqueIndex:idx_eas_user_device;not null"`
	DeviceID      string    `gorm:"uniqueIndex:idx_eas_user_device;size:128;not null"`
	DeviceType    string    `gorm:"size:64"`
	FolderSyncKey string    `gorm:"size:64;default:'0'"`
	PolicyKey     uint      `gorm:"default:1"`
	LastPingAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// EasFolderState stores collection sync keys for a device.
type EasFolderState struct {
	ID           uint   `gorm:"primaryKey"`
	EasDeviceID  uint   `gorm:"uniqueIndex:idx_eas_device_collection;not null"`
	CollectionID string `gorm:"uniqueIndex:idx_eas_device_collection;size:255;not null"`
	SyncKey      string `gorm:"size:64;default:'0'"`
// SyncCache holds JSON-encoded MailSyncCache (UID + flag state per synced message).
	SyncCache    string `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ImapFolderMapping maps IMAP mailbox paths to stable EAS folder IDs.
type ImapFolderMapping struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"index;not null"`
	FolderPath string `gorm:"size:255;not null"`
	FolderGUID string `gorm:"size:64;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
