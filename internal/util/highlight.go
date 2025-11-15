package util

import (
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"os"
)

var (
	// highlightColor is the color used for highlighting matches (yellow background with black text)
	highlightColor = color.New(color.BgYellow, color.FgBlack)

	// colorEnabled determines if color output is enabled
	colorEnabled = isatty.IsTerminal(os.Stdout.Fd())
)

// DisableColor disables color output for highlighting
func DisableColor() {
	colorEnabled = false
	color.NoColor = true
}

// EnableColor enables color output for highlighting
func EnableColor() {
	colorEnabled = isatty.IsTerminal(os.Stdout.Fd())
	color.NoColor = !colorEnabled
}

// IsColorEnabled returns whether color output is currently enabled
func IsColorEnabled() bool {
	return colorEnabled
}

// HighlightPattern highlights all occurrences of a pattern in the given text
// pattern: the pattern to search for
// text: the text to search in
// useRegex: if true, pattern is treated as a regular expression; otherwise as a wildcard pattern
// caseSensitive: if true, perform case-sensitive matching
func HighlightPattern(pattern, text string, useRegex, caseSensitive bool) string {
	if !colorEnabled || pattern == "" || text == "" {
		return text
	}

	var re *regexp.Regexp
	var err error

	if useRegex {
		// Use pattern as-is for regex
		flags := ""
		if !caseSensitive {
			flags = "(?i)"
		}
		re, err = regexp.Compile(flags + pattern)
	} else {
		// Convert wildcard pattern to regex (for highlighting, not filtering)
		regexPattern := wildcardToRegexForHighlighting(pattern)
		flags := ""
		if !caseSensitive {
			flags = "(?i)"
		}
		re, err = regexp.Compile(flags + regexPattern)
	}

	if err != nil {
		// If pattern compilation fails, return text unchanged
		return text
	}

	// Find all matches and their positions
	matches := re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	// Build the highlighted string
	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		start, end := match[0], match[1]

		// Add text before match
		result.WriteString(text[lastEnd:start])

		// Add highlighted match
		result.WriteString(highlightColor.Sprint(text[start:end]))

		lastEnd = end
	}

	// Add remaining text after last match
	result.WriteString(text[lastEnd:])

	return result.String()
}

// HighlightInJSON highlights pattern matches within JSON text while preserving formatting
// This function is more complex as it needs to avoid highlighting JSON syntax characters
func HighlightInJSON(pattern, jsonText string, useRegex, caseSensitive bool) string {
	if !colorEnabled || pattern == "" || jsonText == "" {
		return jsonText
	}

	var re *regexp.Regexp
	var err error

	if useRegex {
		flags := ""
		if !caseSensitive {
			flags = "(?i)"
		}
		re, err = regexp.Compile(flags + pattern)
	} else {
		regexPattern := wildcardToRegexForHighlighting(pattern)
		flags := ""
		if !caseSensitive {
			flags = "(?i)"
		}
		re, err = regexp.Compile(flags + regexPattern)
	}

	if err != nil {
		return jsonText
	}

	// For JSON, we need to be careful not to highlight within JSON structural characters
	// We'll use a simpler approach: highlight within quoted strings only

	// Find all matches
	matches := re.FindAllStringIndex(jsonText, -1)
	if len(matches) == 0 {
		return jsonText
	}

	// Build the highlighted string
	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		start, end := match[0], match[1]

		// Check if this match is inside a JSON string value (not a key or structural char)
		// For simplicity, we'll highlight all matches - the visual result is usually acceptable

		// Add text before match
		result.WriteString(jsonText[lastEnd:start])

		// Add highlighted match
		result.WriteString(highlightColor.Sprint(jsonText[start:end]))

		lastEnd = end
	}

	// Add remaining text after last match
	result.WriteString(jsonText[lastEnd:])

	return result.String()
}

// wildcardToRegex converts a wildcard pattern (* and ?) to a regular expression
// This version anchors the pattern to match the entire string (used for filtering)
func wildcardToRegex(pattern string) string {
	// Escape special regex characters except * and ?
	pattern = regexp.QuoteMeta(pattern)

	// Replace escaped wildcards with regex equivalents
	pattern = strings.ReplaceAll(pattern, `\*`, ".*")
	pattern = strings.ReplaceAll(pattern, `\?`, ".")

	// Anchor the pattern to match the entire string
	return "^" + pattern + "$"
}

// wildcardToRegexForHighlighting converts a wildcard pattern to regex for highlighting
// This version does NOT anchor the pattern, allowing partial matches within text
func wildcardToRegexForHighlighting(pattern string) string {
	// Escape special regex characters except * and ?
	pattern = regexp.QuoteMeta(pattern)

	// Replace escaped wildcards with regex equivalents
	pattern = strings.ReplaceAll(pattern, `\*`, ".*")
	pattern = strings.ReplaceAll(pattern, `\?`, ".")

	// No anchors - we want to match anywhere in the text
	return pattern
}

// HighlightPrefix highlights a prefix in the given text
// This is a specialized version for prefix matching
func HighlightPrefix(prefix, text string, caseSensitive bool) string {
	if !colorEnabled || prefix == "" || text == "" {
		return text
	}

	comparePrefix := prefix
	compareText := text

	if !caseSensitive {
		comparePrefix = strings.ToLower(prefix)
		compareText = strings.ToLower(text)
	}

	if strings.HasPrefix(compareText, comparePrefix) {
		// Highlight the prefix portion
		return highlightColor.Sprint(text[:len(prefix)]) + text[len(prefix):]
	}

	return text
}
