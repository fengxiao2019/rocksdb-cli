package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

const (
	// EncryptedPrefix is prepended to encrypted values to identify them
	EncryptedPrefix = "ENC:AES256:"
)

// GenerateKey generates a new random UUID to be used as encryption key
func GenerateKey() string {
	return uuid.New().String()
}

// deriveKeyFromUUID derives a 32-byte AES-256 key from UUID string
func deriveKeyFromUUID(uuidStr string) ([]byte, error) {
	// Parse UUID to validate format
	u, err := uuid.Parse(uuidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}
	
	// Use SHA-256 to derive a 32-byte key from UUID bytes
	// This ensures we always get exactly 32 bytes for AES-256
	hash := sha256.Sum256(u[:])
	return hash[:], nil
}

// Encrypt encrypts plaintext using AES-256-GCM with the provided UUID key
func Encrypt(plaintext, keyUUID string) (string, error) {
	// Derive encryption key from UUID
	key, err := deriveKeyFromUUID(keyUUID)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}
	
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode (Galois/Counter Mode provides authenticated encryption)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// Encode to base64 and add prefix
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return EncryptedPrefix + encoded, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the provided UUID key
func Decrypt(ciphertext, keyUUID string) (string, error) {
	// Check for encrypted prefix
	if !strings.HasPrefix(ciphertext, EncryptedPrefix) {
		return "", fmt.Errorf("invalid encrypted format: missing prefix")
	}
	
	// Remove prefix and decode from base64
	encoded := strings.TrimPrefix(ciphertext, EncryptedPrefix)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	
	// Derive decryption key from UUID
	key, err := deriveKeyFromUUID(keyUUID)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}
	
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Verify data length
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	
	// Extract nonce and encrypted data
	nonce, encryptedData := data[:nonceSize], data[nonceSize:]
	
	// Decrypt and verify authentication
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return string(plaintext), nil
}

// IsEncrypted checks if a value is encrypted (has the encrypted prefix)
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, EncryptedPrefix)
}

// DecryptIfNeeded decrypts the value if it's encrypted, otherwise returns it as-is
func DecryptIfNeeded(value, keyUUID string) (string, error) {
	if !IsEncrypted(value) {
		return value, nil
	}
	return Decrypt(value, keyUUID)
}

