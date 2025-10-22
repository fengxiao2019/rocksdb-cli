// Package jsonutil provides utility functions for JSON processing with nested expansion support.
package jsonutil

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/oliveagle/jsonpath"
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

// QueryJSONPath queries a JSON string using a JSONPath expression and returns the result as a JSON string.
// It supports standard JSONPath syntax including:
//   - Simple field access: $.name
//   - Nested field access: $.user.name
//   - Array indexing: $.items[0]
//   - Wildcard selection: $.items[*]
//
// Returns an error if the JSON is invalid, the JSONPath expression is invalid,
// or the path doesn't match any data.
func QueryJSONPath(jsonData string, jsonPathExpr string) (string, error) {
	// Validate JSON first
	if !IsValidJSON(jsonData) {
		return "", errors.New("invalid JSON data")
	}

	// Parse JSON data
	var data interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return "", errors.New("failed to parse JSON: " + err.Error())
	}

	// Compile and execute JSONPath
	result, err := jsonpath.JsonPathLookup(data, jsonPathExpr)
	if err != nil {
		return "", errors.New("JSONPath query failed: " + err.Error())
	}

	// Convert result back to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", errors.New("failed to marshal result: " + err.Error())
	}

	return string(resultJSON), nil
}

// IsValidJSON checks if a string is valid JSON.
// Returns true if the string can be parsed as valid JSON, false otherwise.
func IsValidJSON(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}

	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
