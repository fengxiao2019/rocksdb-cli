package util

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
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

// ParseTimestamp attempts to parse a key as a timestamp and return formatted UTC time
// Supports various timestamp formats: Unix seconds, Unix milliseconds, Unix microseconds, Unix nanoseconds, .NET ticks
func ParseTimestamp(key string) string {
	// First, extract the numeric part if key is formatted like "123456789 (0x...)"
	// This handles binary keys that have been formatted by FormatKey()
	numStr := key
	if idx := strings.Index(key, " ("); idx != -1 {
		numStr = key[:idx]
	}

	// Try to parse as integer timestamp
	if ts, err := strconv.ParseInt(numStr, 10, 64); err == nil {
		var t time.Time

		// Determine timestamp format based on number of digits and value range
		switch {
		case ts >= 504911232000000000 && ts <= 3155378975999999999:
			// .NET DateTime.Ticks (100-nanosecond intervals since 0001-01-01 00:00:00)
			// Range: roughly 1986-01-01 to 9999-12-31
			// Convert to Unix timestamp: subtract .NET epoch offset and divide by 10,000,000
			const dotNetEpochTicks = 621355968000000000 // Ticks from 0001-01-01 to 1970-01-01
			unixTicks := ts - dotNetEpochTicks
			if unixTicks >= 0 {
				unixSeconds := unixTicks / 10000000
				unixNanos := (unixTicks % 10000000) * 100
				t = time.Unix(unixSeconds, unixNanos)
			} else {
				return "" // Date before Unix epoch
			}
		case ts > 1e15: // Unix Nanoseconds (16+ digits, but not in .NET range)
			t = time.Unix(0, ts)
		case ts > 1e12: // Microseconds (13-15 digits)
			t = time.Unix(0, ts*1000)
		case ts > 1e9: // Milliseconds (10-12 digits)
			t = time.Unix(0, ts*1e6)
		case ts > 1e6: // Seconds (7-9 digits, covers years ~1973-2033)
			t = time.Unix(ts, 0)
		default:
			return "" // Too small to be a reasonable timestamp
		}

		return t.UTC().Format("2006-01-02 15:04:05.000 UTC")
	}

	// Try to parse as float timestamp (seconds with fractional part)
	if ts, err := strconv.ParseFloat(numStr, 64); err == nil {
		if ts > 1e6 && ts < 1e12 { // Reasonable range for Unix timestamp in seconds
			t := time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e9))
			return t.UTC().Format("2006-01-02 15:04:05.000 UTC")
		}
	}

	return "" // Not a timestamp
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

// IsPrintable checks if data is printable UTF-8 string
// Returns true if data is valid UTF-8 and contains only printable characters
func IsPrintable(data []byte) bool {
	// Check if valid UTF-8
	if !utf8.Valid(data) {
		return false
	}

	// Check if all runes are printable
	s := string(data)
	for _, r := range s {
		// Allow printable ASCII, tab, newline, and other printable unicode
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
		// Reject control characters (except whitespace)
		if r == 127 || (r >= 0 && r < 32 && r != '\t' && r != '\n' && r != '\r') {
			return false
		}
	}
	return true
}

// EncodeValue encodes a value, using appropriate format based on data type
// Returns the encoded string and a flag indicating if it's binary
func EncodeValue(data []byte) (string, bool) {
	if IsPrintable(data) {
		return string(data), false
	}

	// Try to format as a structured key (e.g., uint64, etc.)
	formatted := FormatKey(string(data))
	// If FormatKey returns something different and more readable than raw hex,
	// it means it detected a pattern (like uint64)
	if formatted != string(data) {
		// Check if it's a simple hex representation
		if len(data) == 8 {
			// This is likely a uint64 key, FormatKey will show it as "123 (0x7b)"
			return formatted, false // Mark as non-binary since it's now human-readable
		}
	}

	// Use hex representation for truly binary data
	return ToHexString(data), true
}

// ToHexString converts binary data to hex string format
func ToHexString(data []byte) string {
	var result strings.Builder
	for i, b := range data {
		if i > 0 && i%16 == 0 {
			result.WriteString(" ")
		}
		result.WriteString(fmt.Sprintf("%02x", b))
	}
	return result.String()
}

// FromHexString converts hex string back to binary data
func FromHexString(hexStr string) ([]byte, error) {
	// Remove spaces
	hexStr = strings.ReplaceAll(hexStr, " ", "")

	if len(hexStr)%2 != 0 {
		return nil, fmt.Errorf("invalid hex string length")
	}

	result := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		val, err := strconv.ParseUint(hexStr[i:i+2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex string: %v", err)
		}
		result[i/2] = byte(val)
	}
	return result, nil
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
