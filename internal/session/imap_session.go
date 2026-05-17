package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

var globalDB *gorm.DB

type IMAPSession struct {
	IMAPHost    string
	IMAPPort    int
	Username    string
	EncPassword string
	LastUsed    time.Time
}

var (
	store   = map[string]*IMAPSession{}
	storeMu sync.RWMutex
)

// InitDB loads sessions from the database into memory.
func InitDB(db *gorm.DB) {
	globalDB = db
	var dbSessions []model.Session
	if err := db.Find(&dbSessions).Error; err != nil {
		slog.Error("Failed to load sessions from database", "error", err)
		return
	}
	
	storeMu.Lock()
	defer storeMu.Unlock()
	for _, ds := range dbSessions {
		store[ds.ID] = &IMAPSession{
			IMAPHost:    ds.IMAPHost,
			IMAPPort:    ds.IMAPPort,
			Username:    ds.Username,
			EncPassword: ds.EncPassword,
			LastUsed:    ds.LastUsed,
		}
	}
	slog.Info("Sessions loaded from database", "total", len(store))
}

// saveToDB inserts or updates a session in the database.
func saveToDB(id string, s *IMAPSession) {
	if globalDB == nil {
		return
	}
	dbSess := model.Session{
		ID:          id,
		IMAPHost:    s.IMAPHost,
		IMAPPort:    s.IMAPPort,
		Username:    s.Username,
		EncPassword: s.EncPassword,
		LastUsed:    s.LastUsed,
	}
	if err := globalDB.Save(&dbSess).Error; err != nil {
		slog.Error("Failed to save session to database", "error", err)
	}
}

// deleteFromDB removes a session from the database.
func deleteFromDB(id string) {
	if globalDB == nil {
		return
	}
	if err := globalDB.Where("id = ?", id).Delete(&model.Session{}).Error; err != nil {
		slog.Error("Failed to remove session from database", "error", err)
	}
}

// Set stores or updates a session by HTTP session ID.
func Set(sessionID string, s *IMAPSession) {
	storeMu.Lock()
	s.LastUsed = time.Now()
	store[sessionID] = s
	storeMu.Unlock()
	
	saveToDB(sessionID, s)
}

// Get returns the session by ID; nil if it doesn't exist.
func Get(sessionID string) *IMAPSession {
	storeMu.RLock()
	s := store[sessionID]
	storeMu.RUnlock()
	
	if s != nil {
		// Only update LastUsed in memory for now to avoid constant DB hits
		s.LastUsed = time.Now()
	}
	return s
}

// Delete removes the session.
func Delete(sessionID string) {
	storeMu.Lock()
	delete(store, sessionID)
	storeMu.Unlock()
	
	deleteFromDB(sessionID)
}

// EncryptPassword encrypts a plaintext password using AES-GCM.
func EncryptPassword(plaintext, key string) (string, error) {
	k := []byte(key)
	if len(k) > 32 {
		k = k[:32]
	}
	padded := make([]byte, 32)
	copy(padded, k)

	block, err := aes.NewCipher(padded)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// DecryptPassword decrypts the password stored in the session.
func DecryptPassword(ciphertext, key string) (string, error) {
	k := []byte(key)
	if len(k) > 32 {
		k = k[:32]
	}
	padded := make([]byte, 32)
	copy(padded, k)

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(padded)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// SetPassword encrypts and stores the password in the session.
func (s *IMAPSession) SetPassword(plain, key string) error {
	enc, err := EncryptPassword(plain, key)
	if err != nil {
		return err
	}
	s.EncPassword = enc
	return nil
}

// Password returns the decrypted password.
func (s *IMAPSession) Password(key string) (string, error) {
	return DecryptPassword(s.EncPassword, key)
}

// Cleanup removes idle sessions older than maxIdle.
func Cleanup(maxIdle time.Duration) {
	storeMu.Lock()
	defer storeMu.Unlock()
	cutoff := time.Now().Add(-maxIdle)
	var toDelete []string
	
	for id, s := range store {
		if s.LastUsed.Before(cutoff) {
			delete(store, id)
			toDelete = append(toDelete, id)
		}
	}
	
	if len(toDelete) > 0 && globalDB != nil {
		go func(ids []string) {
			globalDB.Where("id IN ?", ids).Delete(&model.Session{})
		}(toDelete)
	}
}
