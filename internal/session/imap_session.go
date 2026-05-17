package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"
)

// IMAPSession armazena credenciais e metadados da sessão autenticada.
type IMAPSession struct {
	IMAPHost    string
	IMAPPort    int
	Username    string
	encPassword string
	LastUsed    time.Time
}

var (
	store   = map[string]*IMAPSession{}
	storeMu sync.RWMutex
)

// Set guarda ou atualiza uma sessão pelo ID de sessão HTTP.
func Set(sessionID string, s *IMAPSession) {
	storeMu.Lock()
	defer storeMu.Unlock()
	s.LastUsed = time.Now()
	store[sessionID] = s
}

// Get retorna a sessão pelo ID; nil se não existir.
func Get(sessionID string) *IMAPSession {
	storeMu.RLock()
	defer storeMu.RUnlock()
	s := store[sessionID]
	if s != nil {
		s.LastUsed = time.Now()
	}
	return s
}

// Delete remove a sessão.
func Delete(sessionID string) {
	storeMu.Lock()
	defer storeMu.Unlock()
	delete(store, sessionID)
}

// EncryptPassword cifra a senha em texto plano com AES-GCM.
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

// DecryptPassword decifra a senha armazenada na sessão.
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
		return "", fmt.Errorf("ciphertext muito curto")
	}
	nonce, ct := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// SetPassword cifra e armazena a senha na sessão.
func (s *IMAPSession) SetPassword(plain, key string) error {
	enc, err := EncryptPassword(plain, key)
	if err != nil {
		return err
	}
	s.encPassword = enc
	return nil
}

// Password retorna a senha decifrada.
func (s *IMAPSession) Password(key string) (string, error) {
	return DecryptPassword(s.encPassword, key)
}

// Cleanup remove sessões ociosas há mais de maxIdle.
func Cleanup(maxIdle time.Duration) {
	storeMu.Lock()
	defer storeMu.Unlock()
	cutoff := time.Now().Add(-maxIdle)
	for id, s := range store {
		if s.LastUsed.Before(cutoff) {
			delete(store, id)
		}
	}
}
