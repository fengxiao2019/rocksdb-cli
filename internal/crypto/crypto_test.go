package crypto

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateKey(t *testing.T) {
	key := GenerateKey()
	
	// Verify it's a valid UUID
	if _, err := uuid.Parse(key); err != nil {
		t.Errorf("GenerateKey() should return valid UUID, got error: %v", err)
	}
	
	// Verify uniqueness by generating multiple keys
	key2 := GenerateKey()
	if key == key2 {
		t.Errorf("GenerateKey() should generate unique keys, got duplicate: %s", key)
	}
}

func TestDeriveKeyFromUUID(t *testing.T) {
	tests := []struct {
		name    string
		uuidStr string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			uuidStr: "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "invalid UUID",
			uuidStr: "invalid-uuid",
			wantErr: true,
		},
		{
			name:    "empty UUID",
			uuidStr: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := deriveKeyFromUUID(tt.uuidStr)
			
			if tt.wantErr {
				if err == nil {
					t.Error("deriveKeyFromUUID() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("deriveKeyFromUUID() unexpected error: %v", err)
				return
			}
			
			// Verify key length is 32 bytes (256 bits)
			if len(key) != 32 {
				t.Errorf("deriveKeyFromUUID() key length = %d, want 32", len(key))
			}
			
			// Verify deterministic - same UUID should produce same key
			key2, _ := deriveKeyFromUUID(tt.uuidStr)
			if string(key) != string(key2) {
				t.Error("deriveKeyFromUUID() should be deterministic")
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	keyUUID := GenerateKey()
	plaintext := "GRAPHCHAIN_API_KEY=sk-1234567890abcdef"
	
	// Test encryption
	ciphertext, err := Encrypt(plaintext, keyUUID)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Verify ciphertext is not empty
	if ciphertext == "" {
		t.Error("Encrypt() returned empty ciphertext")
	}
	
	// Verify ciphertext is different from plaintext
	if ciphertext == plaintext {
		t.Error("Encrypt() ciphertext should differ from plaintext")
	}
	
	// Verify ciphertext has encrypted prefix
	if !strings.HasPrefix(ciphertext, EncryptedPrefix) {
		t.Errorf("Encrypt() ciphertext should have prefix %s", EncryptedPrefix)
	}
	
	// Test decryption
	decrypted, err := Decrypt(ciphertext, keyUUID)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	// Verify decrypted matches original
	if decrypted != plaintext {
		t.Errorf("Decrypt() = %v, want %v", decrypted, plaintext)
	}
}

func TestEncryptEmptyString(t *testing.T) {
	keyUUID := GenerateKey()
	plaintext := ""
	
	ciphertext, err := Encrypt(plaintext, keyUUID)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	decrypted, err := Decrypt(ciphertext, keyUUID)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	if decrypted != plaintext {
		t.Errorf("Decrypt() = %v, want %v", decrypted, plaintext)
	}
}

func TestEncryptLongString(t *testing.T) {
	keyUUID := GenerateKey()
	plaintext := strings.Repeat("a", 10000)
	
	ciphertext, err := Encrypt(plaintext, keyUUID)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	decrypted, err := Decrypt(ciphertext, keyUUID)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	if decrypted != plaintext {
		t.Errorf("Decrypt() length = %d, want %d", len(decrypted), len(plaintext))
	}
}

func TestEncryptWithInvalidKey(t *testing.T) {
	_, err := Encrypt("test", "invalid-uuid")
	if err == nil {
		t.Error("Encrypt() with invalid UUID should return error")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	keyUUID1 := GenerateKey()
	keyUUID2 := GenerateKey()
	plaintext := "secret-data"
	
	ciphertext, err := Encrypt(plaintext, keyUUID1)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	_, err = Decrypt(ciphertext, keyUUID2)
	if err == nil {
		t.Error("Decrypt() with wrong key should return error")
	}
}

func TestDecryptWithInvalidCiphertext(t *testing.T) {
	keyUUID := GenerateKey()
	
	tests := []struct {
		name       string
		ciphertext string
	}{
		{"invalid base64", EncryptedPrefix + "not-valid-base64!!!"},
		{"too short", EncryptedPrefix + "YWJj"},
		{"no prefix", "randomdata"},
		{"empty", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.ciphertext, keyUUID)
			if err == nil {
				t.Error("Decrypt() should return error for invalid ciphertext")
			}
		})
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"encrypted value", EncryptedPrefix + "somedata", true},
		{"plain text", "plain-text-value", false},
		{"empty string", "", false},
		{"partial prefix", "ENC:", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEncrypted(tt.value); got != tt.want {
				t.Errorf("IsEncrypted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecryptIfNeeded(t *testing.T) {
	keyUUID := GenerateKey()
	plaintext := "my-secret-key"
	
	// Test with encrypted value
	encrypted, err := Encrypt(plaintext, keyUUID)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	decrypted, err := DecryptIfNeeded(encrypted, keyUUID)
	if err != nil {
		t.Fatalf("DecryptIfNeeded() error = %v", err)
	}
	
	if decrypted != plaintext {
		t.Errorf("DecryptIfNeeded() = %v, want %v", decrypted, plaintext)
	}
	
	// Test with plain text value (should return as-is)
	result, err := DecryptIfNeeded(plaintext, keyUUID)
	if err != nil {
		t.Fatalf("DecryptIfNeeded() error = %v", err)
	}
	
	if result != plaintext {
		t.Errorf("DecryptIfNeeded() = %v, want %v", result, plaintext)
	}
}

func TestEncryptionUniqueness(t *testing.T) {
	keyUUID := GenerateKey()
	plaintext := "same-plaintext"
	
	// Encrypt same plaintext multiple times
	ciphertext1, err := Encrypt(plaintext, keyUUID)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	ciphertext2, err := Encrypt(plaintext, keyUUID)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Ciphertexts should be different due to random nonce
	if ciphertext1 == ciphertext2 {
		t.Error("Encrypt() should produce different ciphertexts for same plaintext (nonce should be random)")
	}
	
	// But both should decrypt to same plaintext
	decrypted1, _ := Decrypt(ciphertext1, keyUUID)
	decrypted2, _ := Decrypt(ciphertext2, keyUUID)
	
	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both ciphertexts should decrypt to original plaintext")
	}
}


