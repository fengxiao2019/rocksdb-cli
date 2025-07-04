package util

import (
	"encoding/binary"
	"testing"
)

func TestDetectKeyFormat(t *testing.T) {
	tests := []struct {
		name       string
		sampleKeys []string
		expected   KeyFormat
	}{
		{
			name:       "empty sample",
			sampleKeys: []string{},
			expected:   KeyFormatString,
		},
		{
			name: "uint64 keys",
			sampleKeys: []string{
				string(uint64ToBytes(123456789)),
				string(uint64ToBytes(987654321)),
				string(uint64ToBytes(111222333)),
			},
			expected: KeyFormatUint64BE,
		},
		{
			name: "hex keys",
			sampleKeys: []string{
				"0xdeadbeef",
				"0x12345678",
				"0xabcdef01",
			},
			expected: KeyFormatHex,
		},
		{
			name: "string keys",
			sampleKeys: []string{
				"user:1001",
				"product:abc",
				"session:xyz",
			},
			expected: KeyFormatString,
		},
		{
			name: "mixed keys",
			sampleKeys: []string{
				string(uint64ToBytes(123456789)),
				"user:1001",
				"0xdeadbeef",
			},
			expected: KeyFormatMixed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectKeyFormat(tt.sampleKeys)
			if result != tt.expected {
				t.Errorf("DetectKeyFormat() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertStringToKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		format   KeyFormat
		expected []byte
		wantErr  bool
	}{
		{
			name:     "uint64 decimal",
			input:    "123456789",
			format:   KeyFormatUint64BE,
			expected: uint64ToBytes(123456789),
			wantErr:  false,
		},
		{
			name:     "uint64 hex with 0x",
			input:    "0x75bf1515",
			format:   KeyFormatUint64BE,
			expected: uint64ToBytes(0x75bf1515),
			wantErr:  false,
		},
		{
			name:     "hex string",
			input:    "deadbeef",
			format:   KeyFormatHex,
			expected: []byte{0xde, 0xad, 0xbe, 0xef},
			wantErr:  false,
		},
		{
			name:     "hex string with 0x",
			input:    "0xdeadbeef",
			format:   KeyFormatHex,
			expected: []byte{0xde, 0xad, 0xbe, 0xef},
			wantErr:  false,
		},
		{
			name:     "string format",
			input:    "user:1001",
			format:   KeyFormatString,
			expected: []byte("user:1001"),
			wantErr:  false,
		},
		{
			name:     "mixed format uint64",
			input:    "123456789",
			format:   KeyFormatMixed,
			expected: uint64ToBytes(123456789),
			wantErr:  false,
		},
		{
			name:     "mixed format hex",
			input:    "deadbeef",
			format:   KeyFormatMixed,
			expected: uint64ToBytes(0xdeadbeef),
			wantErr:  false,
		},
		{
			name:     "mixed format string fallback",
			input:    "not_a_number_or_hex",
			format:   KeyFormatMixed,
			expected: []byte("not_a_number_or_hex"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStringToKey(tt.input, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertStringToKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytesEqual(result, tt.expected) {
				t.Errorf("ConvertStringToKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertStringToKeyForScan(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		format   KeyFormat
		isPrefix bool
		expected []byte
		wantErr  bool
	}{
		{
			name:     "empty input",
			input:    "",
			format:   KeyFormatUint64BE,
			isPrefix: false,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "wildcard input",
			input:    "*",
			format:   KeyFormatUint64BE,
			isPrefix: false,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "uint64 scan",
			input:    "100",
			format:   KeyFormatUint64BE,
			isPrefix: false,
			expected: uint64ToBytes(100),
			wantErr:  false,
		},
		{
			name:     "hex prefix",
			input:    "dead",
			format:   KeyFormatHex,
			isPrefix: true,
			expected: []byte{0xde, 0xad},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStringToKeyForScan(tt.input, tt.format, tt.isPrefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertStringToKeyForScan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytesEqual(result, tt.expected) {
				t.Errorf("ConvertStringToKeyForScan() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"deadbeef", true},
		{"0xdeadbeef", true},
		{"123abc", true},
		{"xyz", false},
		{"", false},
		{"0x", false},
		{"deadbeeg", false}, // 'g' is not hex
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isHexString(tt.input)
			if result != tt.expected {
				t.Errorf("isHexString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsPrintableString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"hello world", true},
		{"user:1001", true},
		{"", true},
		{string([]byte{0x01, 0x02}), false}, // non-printable
		{"hello\x00world", false},           // null byte
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isPrintableString(tt.input)
			if result != tt.expected {
				t.Errorf("isPrintableString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "uint64 key",
			input:    string(uint64ToBytes(123456789)),
			expected: "123456789 (0x75bcd15)",
		},
		{
			name:     "printable string",
			input:    "user:1001",
			expected: "user:1001",
		},
		{
			name:     "binary data",
			input:    string([]byte{0xde, 0xad, 0xbe, 0xef}),
			expected: "0xdeadbeef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatKey(tt.input)
			if result != tt.expected {
				t.Errorf("FormatKey() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Helper functions for tests
func uint64ToBytes(val uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, val)
	return buf
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
