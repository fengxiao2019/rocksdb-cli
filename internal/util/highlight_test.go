package util

import (
	"strings"
	"testing"

	"github.com/fatih/color"
)

// Helper function to force enable colors for testing
func forceEnableColors() func() {
	originalColorEnabled := colorEnabled
	originalNoColor := color.NoColor
	colorEnabled = true
	color.NoColor = false

	// Return cleanup function
	return func() {
		colorEnabled = originalColorEnabled
		color.NoColor = originalNoColor
	}
}

func TestHighlightPattern(t *testing.T) {
	defer forceEnableColors()()

	tests := []struct {
		name          string
		pattern       string
		text          string
		useRegex      bool
		caseSensitive bool
		shouldMatch   bool
		description   string
	}{
		{
			name:          "Simple wildcard match",
			pattern:       "email",
			text:          `{"name":"Alice","email":"alice@example.com"}`,
			useRegex:      false,
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should highlight 'email' in JSON",
		},
		{
			name:          "Wildcard with asterisk",
			pattern:       "user*",
			text:          "user001",
			useRegex:      false,
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should match user with any suffix",
		},
		{
			name:          "Regex pattern - digits",
			pattern:       "[0-9]+",
			text:          "user123",
			useRegex:      true,
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should highlight digit sequences",
		},
		{
			name:          "Regex pattern - email",
			pattern:       "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
			text:          `{"email":"alice@example.com"}`,
			useRegex:      true,
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should highlight email addresses",
		},
		{
			name:          "Case insensitive match",
			pattern:       "alice",
			text:          "Alice",
			useRegex:      false,
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should match case-insensitively by default",
		},
		{
			name:          "Case sensitive no match",
			pattern:       "alice",
			text:          "Alice",
			useRegex:      false,
			caseSensitive: true,
			shouldMatch:   false,
			description:   "Should NOT match when case differs with caseSensitive=true",
		},
		{
			name:          "Multiple matches in text",
			pattern:       "test",
			text:          "test test test",
			useRegex:      false,
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should highlight all occurrences",
		},
		{
			name:          "Partial match in word",
			pattern:       "active",
			text:          `{"status":"active","inactive":false}`,
			useRegex:      false,
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should match 'active' both as standalone and within 'inactive'",
		},
		{
			name:          "Empty pattern",
			pattern:       "",
			text:          "some text",
			useRegex:      false,
			caseSensitive: false,
			shouldMatch:   false,
			description:   "Empty pattern should return text unchanged",
		},
		{
			name:          "Empty text",
			pattern:       "test",
			text:          "",
			useRegex:      false,
			caseSensitive: false,
			shouldMatch:   false,
			description:   "Empty text should return empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HighlightPattern(tt.pattern, tt.text, tt.useRegex, tt.caseSensitive)

			if tt.shouldMatch {
				// If colors are enabled, result should contain ANSI codes
				// If colors are disabled (CI), result should equal original text
				if IsColorEnabled() {
					if result == tt.text {
						t.Errorf("Expected highlighting for pattern '%s' in text '%s', but got unchanged text",
							tt.pattern, tt.text)
					}
					// Check if result contains ANSI escape sequences (indicating highlighting)
					if !strings.Contains(result, "\x1b[") {
						t.Errorf("Expected ANSI color codes in result, but got: %s", result)
					}
				}
			} else {
				// No match expected - should return original text
				if result != tt.text {
					t.Errorf("Expected unchanged text for pattern '%s', but got: %s", tt.pattern, result)
				}
			}
		})
	}
}

func TestHighlightInJSON(t *testing.T) {
	defer forceEnableColors()()

	tests := []struct {
		name          string
		pattern       string
		jsonText      string
		useRegex      bool
		caseSensitive bool
		shouldMatch   bool
		description   string
	}{
		{
			name:          "Highlight field name in JSON",
			pattern:       "email",
			jsonText:      `{"email":"test@example.com"}`,
			useRegex:      false,
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should highlight 'email' field name",
		},
		{
			name:          "Highlight value in pretty JSON",
			pattern:       "active",
			jsonText:      "{\n  \"status\": \"active\"\n}",
			useRegex:      false,
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should highlight in formatted JSON",
		},
		{
			name:          "Regex pattern in JSON",
			pattern:       "@[a-zA-Z0-9.-]+",
			jsonText:      `{"email":"user@example.com"}`,
			useRegex:      true,
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should highlight email domain",
		},
		{
			name:          "Empty pattern",
			pattern:       "",
			jsonText:      `{"test":"value"}`,
			useRegex:      false,
			caseSensitive: false,
			shouldMatch:   false,
			description:   "Empty pattern should return unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HighlightInJSON(tt.pattern, tt.jsonText, tt.useRegex, tt.caseSensitive)

			if tt.shouldMatch && IsColorEnabled() {
				if result == tt.jsonText {
					t.Errorf("Expected highlighting in JSON, but got unchanged text")
				}
			} else if !tt.shouldMatch {
				if result != tt.jsonText {
					t.Errorf("Expected unchanged JSON, but got: %s", result)
				}
			}
		})
	}
}

func TestHighlightPrefix(t *testing.T) {
	defer forceEnableColors()()

	tests := []struct {
		name          string
		prefix        string
		text          string
		caseSensitive bool
		shouldMatch   bool
		description   string
	}{
		{
			name:          "Simple prefix match",
			prefix:        "user",
			text:          "user001",
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should highlight 'user' prefix",
		},
		{
			name:          "Case insensitive prefix",
			prefix:        "user",
			text:          "USER001",
			caseSensitive: false,
			shouldMatch:   true,
			description:   "Should match prefix case-insensitively",
		},
		{
			name:          "Case sensitive no match",
			prefix:        "user",
			text:          "USER001",
			caseSensitive: true,
			shouldMatch:   false,
			description:   "Should NOT match when case differs",
		},
		{
			name:          "No prefix match",
			prefix:        "admin",
			text:          "user001",
			caseSensitive: false,
			shouldMatch:   false,
			description:   "Should return unchanged when prefix doesn't match",
		},
		{
			name:          "Empty prefix",
			prefix:        "",
			text:          "user001",
			caseSensitive: false,
			shouldMatch:   false,
			description:   "Empty prefix should return unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HighlightPrefix(tt.prefix, tt.text, tt.caseSensitive)

			if tt.shouldMatch && IsColorEnabled() {
				if result == tt.text {
					t.Errorf("Expected prefix highlighting, but got unchanged text")
				}
				// Verify the prefix portion is highlighted
				if !strings.Contains(result, "\x1b[") {
					t.Errorf("Expected ANSI color codes in result")
				}
			} else if !tt.shouldMatch {
				if result != tt.text {
					t.Errorf("Expected unchanged text, but got: %s", result)
				}
			}
		})
	}
}

func TestWildcardToRegex(t *testing.T) {
	tests := []struct {
		pattern  string
		text     string
		expected bool
	}{
		{"user*", "user001", true},
		{"user*", "user", true},
		{"user*", "admin", false},
		{"user?", "user1", true},
		{"user?", "user12", false},
		{"*test*", "mytest", true},
		{"*test*", "test", true},
		{"*test*", "testing", true},
		{"test", "test", true},
		{"test", "testing", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_vs_"+tt.text, func(t *testing.T) {
			regex := wildcardToRegex(tt.pattern)
			// The regex should have anchors for exact matching
			if !strings.HasPrefix(regex, "^") || !strings.HasSuffix(regex, "$") {
				t.Errorf("wildcardToRegex should add anchors, got: %s", regex)
			}
		})
	}
}

func TestWildcardToRegexForHighlighting(t *testing.T) {
	tests := []struct {
		pattern     string
		description string
		expected    string
	}{
		{
			pattern:     "test",
			description: "Simple pattern without wildcards",
			expected:    "test",
		},
		{
			pattern:     "test*",
			description: "Pattern with asterisk wildcard",
			expected:    "test.*",
		},
		{
			pattern:     "test?",
			description: "Pattern with question mark wildcard",
			expected:    "test.",
		},
		{
			pattern:     "*test*",
			description: "Pattern with wildcards on both sides",
			expected:    ".*test.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := wildcardToRegexForHighlighting(tt.pattern)

			// Should NOT have anchors (this is the key difference)
			if strings.HasPrefix(result, "^") || strings.HasSuffix(result, "$") {
				t.Errorf("wildcardToRegexForHighlighting should NOT add anchors, got: %s", result)
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestEnableDisableColor(t *testing.T) {
	// Test color enable/disable
	originalState := IsColorEnabled()
	defer func() {
		// Restore original state
		if originalState {
			EnableColor()
		} else {
			DisableColor()
		}
	}()

	// Test disable
	DisableColor()
	if IsColorEnabled() {
		t.Error("Expected color to be disabled")
	}
	if !color.NoColor {
		t.Error("Expected color.NoColor to be true")
	}

	// Test that highlighting returns unchanged text when disabled
	text := "test text"
	result := HighlightPattern("test", text, false, false)
	if result != text {
		t.Error("Expected unchanged text when colors are disabled")
	}

	// Test enable
	EnableColor()
	// Note: EnableColor checks if stdout is a terminal
	// In test environment, it might still be disabled
}

func TestHighlightPatternEdgeCases(t *testing.T) {
	defer forceEnableColors()()

	tests := []struct {
		name        string
		pattern     string
		text        string
		useRegex    bool
		description string
	}{
		{
			name:        "Special regex characters in wildcard mode",
			pattern:     "test.value",
			text:        "test.value",
			useRegex:    false,
			description: "Dots should be escaped in wildcard mode",
		},
		{
			name:        "Invalid regex pattern",
			pattern:     "[invalid",
			text:        "some text",
			useRegex:    true,
			description: "Invalid regex should return unchanged text",
		},
		{
			name:        "Unicode text",
			pattern:     "测试",
			text:        "这是测试文本",
			useRegex:    false,
			description: "Should handle unicode characters",
		},
		{
			name:        "Very long text",
			pattern:     "needle",
			text:        strings.Repeat("haystack ", 1000) + "needle" + strings.Repeat(" haystack", 1000),
			useRegex:    false,
			description: "Should handle very long text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HighlightPattern(tt.pattern, tt.text, tt.useRegex, false)
			// Just ensure it doesn't panic and returns something
			if result == "" && tt.text != "" {
				t.Error("Result should not be empty for non-empty text")
			}
		})
	}
}

func TestHighlightPatternMultipleMatches(t *testing.T) {
	defer forceEnableColors()()

	pattern := "test"
	text := "test1 test2 test3"
	result := HighlightPattern(pattern, text, false, false)

	// Count number of ANSI escape sequences (should be at least 3 for 3 matches)
	escapeCount := strings.Count(result, "\x1b[")
	if escapeCount < 3 {
		t.Errorf("Expected at least 3 ANSI escape sequences for 3 matches, got %d", escapeCount)
	}
}

func TestHighlightInJSONPreservesFormatting(t *testing.T) {
	defer forceEnableColors()()

	jsonText := `{
  "name": "Alice",
  "email": "alice@example.com"
}`

	result := HighlightInJSON("email", jsonText, false, false)

	// Ensure JSON structure is preserved (newlines should still be there)
	if strings.Count(result, "\n") != strings.Count(jsonText, "\n") {
		t.Error("JSON formatting should be preserved")
	}
}

// Benchmark tests
func BenchmarkHighlightPattern(b *testing.B) {
	defer forceEnableColors()()
	text := strings.Repeat("This is a test text with multiple test occurrences. ", 100)
	pattern := "test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HighlightPattern(pattern, text, false, false)
	}
}

func BenchmarkHighlightPatternRegex(b *testing.B) {
	defer forceEnableColors()()
	text := strings.Repeat("user001 user002 user003 ", 100)
	pattern := "user[0-9]+"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HighlightPattern(pattern, text, true, false)
	}
}

func BenchmarkHighlightInJSON(b *testing.B) {
	defer forceEnableColors()()
	jsonText := `{"name":"Alice","email":"alice@example.com","status":"active","age":30}`
	pattern := "email"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HighlightInJSON(pattern, jsonText, false, false)
	}
}
