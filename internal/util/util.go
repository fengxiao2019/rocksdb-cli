package util

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// KeyFormat represents the detected format of keys in a column family
type KeyFormat int

const (
	KeyFormatString   KeyFormat = iota
	KeyFormatUint64BE           // 8-byte big-endian uint64
	KeyFormatHex                // hex-encoded binary data
	KeyFormatMixed              // mixed formats detected
)

// formatKey attempts to format a key in a human-readable way
func FormatKey(key string) string {
	// Try to decode as a long integer (big-endian)
	if len(key) == 8 {
		var val uint64
		for i := 0; i < 8; i++ {
			val = (val << 8) | uint64(key[i])
		}
		return fmt.Sprintf("%d (0x%x)", val, val)
	}

	// Check if the key is printable ASCII
	isPrintable := true
	for i := 0; i < len(key); i++ {
		if key[i] < 32 || key[i] > 126 {
			isPrintable = false
			break
		}
	}

	if !isPrintable {
		// Show as hex if not printable
		var hexStr strings.Builder
		hexStr.WriteString("0x")
		for i := 0; i < len(key); i++ {
			hexStr.WriteString(fmt.Sprintf("%02x", key[i]))
		}
		return hexStr.String()
	}

	return key
}

// DetectKeyFormat analyzes a sample of keys to determine the most likely format
func DetectKeyFormat(sampleKeys []string) KeyFormat {
	if len(sampleKeys) == 0 {
		return KeyFormatString
	}

	uint64Count := 0
	hexCount := 0
	stringCount := 0

	for _, key := range sampleKeys {
		if len(key) == 8 {
			// Check if it's a valid uint64 (all bytes)
			uint64Count++
		} else if isHexString(key) {
			hexCount++
		} else if isPrintableString(key) {
			stringCount++
		}
	}

	total := len(sampleKeys)

	// If 80% or more are uint64, consider it uint64 format
	if float64(uint64Count)/float64(total) >= 0.8 {
		return KeyFormatUint64BE
	}

	// If 80% or more are hex strings, consider it hex format
	if float64(hexCount)/float64(total) >= 0.8 {
		return KeyFormatHex
	}

	// If we have mixed formats, return mixed
	if uint64Count > 0 || hexCount > 0 {
		return KeyFormatMixed
	}

	return KeyFormatString
}

// ConvertStringToKey converts a user input string to the appropriate binary key format
func ConvertStringToKey(input string, format KeyFormat) ([]byte, error) {
	switch format {
	case KeyFormatUint64BE:
		return convertStringToUint64Key(input)
	case KeyFormatHex:
		return convertStringToHexKey(input)
	case KeyFormatMixed:
		// Try multiple formats in order of preference
		if key, err := convertStringToUint64Key(input); err == nil {
			return key, nil
		}
		if key, err := convertStringToHexKey(input); err == nil {
			return key, nil
		}
		// Fall back to string
		return []byte(input), nil
	case KeyFormatString:
		fallthrough
	default:
		return []byte(input), nil
	}
}

// convertStringToUint64Key converts a string to 8-byte big-endian uint64
func convertStringToUint64Key(input string) ([]byte, error) {
	// Try to parse as decimal number
	if val, err := strconv.ParseUint(input, 10, 64); err == nil {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, val)
		return buf, nil
	}

	// Try to parse as hex number (with or without 0x prefix)
	cleanInput := strings.TrimPrefix(strings.ToLower(input), "0x")
	if val, err := strconv.ParseUint(cleanInput, 16, 64); err == nil {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, val)
		return buf, nil
	}

	return nil, fmt.Errorf("cannot convert '%s' to uint64", input)
}

// convertStringToHexKey converts a hex string to binary
func convertStringToHexKey(input string) ([]byte, error) {
	// Remove 0x prefix if present
	cleanInput := strings.TrimPrefix(strings.ToLower(input), "0x")

	// Ensure even length for hex decoding
	if len(cleanInput)%2 != 0 {
		cleanInput = "0" + cleanInput
	}

	// Convert hex string to bytes
	result := make([]byte, len(cleanInput)/2)
	for i := 0; i < len(cleanInput); i += 2 {
		if val, err := strconv.ParseUint(cleanInput[i:i+2], 16, 8); err == nil {
			result[i/2] = byte(val)
		} else {
			return nil, fmt.Errorf("invalid hex string: %s", input)
		}
	}

	return result, nil
}

// isHexString checks if a string looks like a hex-encoded value
func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Check if it starts with 0x and has valid hex characters
	if strings.HasPrefix(strings.ToLower(s), "0x") {
		s = s[2:]
	}

	if len(s) == 0 {
		return false
	}

	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}

	return true
}

// isPrintableString checks if a string contains only printable ASCII characters
func isPrintableString(s string) bool {
	for _, b := range []byte(s) {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}

// ConvertStringToKeyForScan converts string inputs for scan operations, handling prefixes and ranges
func ConvertStringToKeyForScan(input string, format KeyFormat, isPrefix bool) ([]byte, error) {
	if input == "" || input == "*" {
		return nil, nil // Empty input means no bound
	}

	switch format {
	case KeyFormatUint64BE:
		if isPrefix {
			// For prefix scans with uint64, we need to handle partial numbers
			return convertStringToUint64Prefix(input)
		}
		return convertStringToUint64Key(input)
	case KeyFormatHex:
		if isPrefix {
			return convertStringToHexPrefix(input)
		}
		return convertStringToHexKey(input)
	case KeyFormatMixed:
		// Try uint64 first, then hex, then string
		if key, err := ConvertStringToKeyForScan(input, KeyFormatUint64BE, isPrefix); err == nil {
			return key, nil
		}
		if key, err := ConvertStringToKeyForScan(input, KeyFormatHex, isPrefix); err == nil {
			return key, nil
		}
		return []byte(input), nil
	case KeyFormatString:
		fallthrough
	default:
		return []byte(input), nil
	}
}

// convertStringToUint64Prefix handles prefix matching for uint64 keys
func convertStringToUint64Prefix(input string) ([]byte, error) {
	// For uint64 prefix, we convert the number and use it as a starting point
	// This is a simplified approach - for true prefix matching on binary data,
	// we would need more sophisticated logic
	return convertStringToUint64Key(input)
}

// convertStringToHexPrefix handles prefix matching for hex keys
func convertStringToHexPrefix(input string) ([]byte, error) {
	// Remove 0x prefix if present
	cleanInput := strings.TrimPrefix(strings.ToLower(input), "0x")

	// For prefix matching, we don't need to pad to even length
	// Convert what we have and let RocksDB handle the prefix matching
	if len(cleanInput)%2 != 0 {
		// For odd-length prefixes, we can either pad with 0 or truncate
		// Padding with 0 gives us the "smallest" key starting with this prefix
		cleanInput = cleanInput + "0"
	}

	result := make([]byte, len(cleanInput)/2)
	for i := 0; i < len(cleanInput); i += 2 {
		if val, err := strconv.ParseUint(cleanInput[i:i+2], 16, 8); err == nil {
			result[i/2] = byte(val)
		} else {
			return nil, fmt.Errorf("invalid hex prefix: %s", input)
		}
	}

	return result, nil
}
