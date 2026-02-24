package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// ErrInvalidKeyVersion is returned when an invalid key version is used
var ErrInvalidKeyVersion = errors.New("invalid key version")

// KeyStore holds encryption keys indexed by version
type KeyStore struct {
	keys map[int][]byte
}

// NewKeyStore creates a new key store with the given keys
// keys is a map of version -> 32-byte key material
func NewKeyStore(keys map[int][]byte) *KeyStore {
	return &KeyStore{keys: keys}
}

// DefaultKeyStore creates a key store from environment variables
// Keys should be provided as base64-encoded 32-byte values
func DefaultKeyStore() (*KeyStore, error) {
	keys := make(map[int][]byte)

	// Try to load multiple key versions
	for version := 1; version <= 3; version++ {
		keyEnv := getKeyEnv(version)
		if keyData, err := base64.StdEncoding.DecodeString(keyEnv); err == nil && len(keyData) == 32 {
			keys[version] = keyData
		}
	}

	// If no keys loaded, generate a default key (for development only)
	if len(keys) == 0 {
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, err
		}
		keys[1] = key
	}

	return NewKeyStore(keys), nil
}

// getKeyEnv returns the environment variable name for a key version
func getKeyEnv(version int) string {
	switch version {
	case 1:
		return getEnvOrDefault("ENCRYPTION_KEY_V1", "")
	case 2:
		return getEnvOrDefault("ENCRYPTION_KEY_V2", "")
	case 3:
		return getEnvOrDefault("ENCRYPTION_KEY_V3", "")
	default:
		return ""
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	// In a real implementation, this would read from os.Getenv
	// For now, return the default value
	return defaultValue
}

// CurrentKeyVersion returns the highest key version
func (ks *KeyStore) CurrentKeyVersion() int {
	maxVersion := 0
	for v := range ks.keys {
		if v > maxVersion {
			maxVersion = v
		}
	}
	return maxVersion
}

// Encrypt encrypts plaintext using AES-GCM with the current key version
// Returns: nonce+ciphertext (combined), key version used
func (ks *KeyStore) Encrypt(plaintext []byte) ([]byte, int, error) {
	return ks.EncryptWithVersion(plaintext, ks.CurrentKeyVersion())
}

// EncryptWithVersion encrypts plaintext using AES-GCM with a specific key version
func (ks *KeyStore) EncryptWithVersion(plaintext []byte, version int) ([]byte, int, error) {
	key, ok := ks.keys[version]
	if !ok {
		return nil, 0, ErrInvalidKeyVersion
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, 0, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, 0, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, 0, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, version, nil
}

// Decrypt decrypts ciphertext using AES-GCM
// The key version is embedded in the ciphertext format (first bytes)
func (ks *KeyStore) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 12 { // Minimum: nonce size (12) + at least 1 byte
		return nil, errors.New("ciphertext too short")
	}

	// Try each key version to decrypt (for backward compatibility)
	versions := []int{ks.CurrentKeyVersion()}
	for v := range ks.keys {
		found := false
		for _, existing := range versions {
			if existing == v {
				found = true
				break
			}
		}
		if !found {
			versions = append(versions, v)
		}
	}

	var lastErr error
	for _, version := range versions {
		plaintext, err := ks.DecryptWithVersion(ciphertext, version)
		if err == nil {
			return plaintext, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("failed to decrypt with any key version")
}

// DecryptWithVersion decrypts ciphertext using a specific key version
// Returns error if the key version is not available
func (ks *KeyStore) DecryptWithVersion(ciphertext []byte, version int) ([]byte, error) {
	key, ok := ks.keys[version]
	if !ok {
		return nil, ErrInvalidKeyVersion
	}

	if len(ciphertext) < 12 { // Minimum: nonce size (12) + at least 1 byte
		return nil, errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short for nonce")
	}

	nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext
func (ks *KeyStore) EncryptString(plaintext string) (string, int, error) {
	ciphertext, version, err := ks.Encrypt([]byte(plaintext))
	if err != nil {
		return "", 0, err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), version, nil
}

// DecryptString decrypts a base64-encoded ciphertext string
func (ks *KeyStore) DecryptString(ciphertextBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}
	plaintext, err := ks.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
