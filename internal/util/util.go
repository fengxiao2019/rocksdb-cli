package util

import (
	"fmt"
	"strings"
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
