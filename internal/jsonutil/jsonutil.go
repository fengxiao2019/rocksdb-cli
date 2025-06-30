// Package jsonutil provides utility functions for JSON processing with nested expansion support.
package jsonutil

import (
	"encoding/json"
	"strings"
)

// PrettyPrintWithNestedExpansion formats JSON with recursive nested JSON string expansion.
// It will recursively expand any JSON strings found within the data structure while
// maintaining proper formatting and indentation.
func PrettyPrintWithNestedExpansion(value string) string {
	var jsonData interface{}
	if err := json.Unmarshal([]byte(value), &jsonData); err != nil {
		return value // If not valid JSON, return as is
	}

	// Recursively expand nested JSON strings
	expandedData := expandNestedJSON(jsonData)

	prettyJSON, err := json.MarshalIndent(expandedData, "", "  ")
	if err != nil {
		return value // If can't pretty print, return as is
	}

	return string(prettyJSON)
}

// expandNestedJSON recursively expands JSON strings within the data structure.
// It handles objects, arrays, and string values that contain valid JSON.
func expandNestedJSON(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		// Handle JSON objects
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = expandNestedJSON(val)
		}
		return result
	case []interface{}:
		// Handle JSON arrays
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = expandNestedJSON(val)
		}
		return result
	case string:
		// Try to parse string as JSON
		if isJSONString(v) {
			var nestedData interface{}
			if err := json.Unmarshal([]byte(v), &nestedData); err == nil {
				// Recursively expand the nested JSON
				return expandNestedJSON(nestedData)
			}
		}
		return v
	default:
		// Return as is for primitive types (numbers, booleans, null)
		return v
	}
}

// isJSONString checks if a string appears to be valid JSON by examining its structure.
// It performs a basic check for JSON object or array delimiters after trimming whitespace.
func isJSONString(s string) bool {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 {
		return false
	}

	// Check if string starts and ends with JSON delimiters
	return (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"))
}
