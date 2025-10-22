package jsonutil

import (
	"testing"
)

func TestPrettyPrintWithNestedExpansion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple JSON object",
			input:    `{"name":"Alice","age":25}`,
			expected: "{\n  \"age\": 25,\n  \"name\": \"Alice\"\n}",
		},
		{
			name:     "nested JSON string",
			input:    `{"user_id":"123","profile":"{\"name\":\"Alice\",\"age\":30,\"city\":\"New York\"}"}`,
			expected: "{\n  \"profile\": {\n    \"age\": 30,\n    \"city\": \"New York\",\n    \"name\": \"Alice\"\n  },\n  \"user_id\": \"123\"\n}",
		},
		{
			name:     "deeply nested JSON",
			input:    `{"event_id":"evt_001","metadata":"{\"timestamp\":\"2024-01-15T10:30:00Z\",\"source\":\"api\",\"details\":{\"user_agent\":\"Mozilla/5.0\",\"ip\":\"192.168.1.1\"}}"}`,
			expected: "{\n  \"event_id\": \"evt_001\",\n  \"metadata\": {\n    \"details\": {\n      \"ip\": \"192.168.1.1\",\n      \"user_agent\": \"Mozilla/5.0\"\n    },\n    \"source\": \"api\",\n    \"timestamp\": \"2024-01-15T10:30:00Z\"\n  }\n}",
		},
		{
			name:     "array with JSON strings",
			input:    `{"users":["{\"name\":\"Alice\",\"age\":25}","{\"name\":\"Bob\",\"age\":30}"]}`,
			expected: "{\n  \"users\": [\n    {\n      \"age\": 25,\n      \"name\": \"Alice\"\n    },\n    {\n      \"age\": 30,\n      \"name\": \"Bob\"\n    }\n  ]\n}",
		},
		{
			name:     "mixed content",
			input:    `{"id":"order_123","description":"Customer order","order_data":"{\"items\":[{\"product\":\"laptop\",\"price\":999.99}],\"total\":999.99}","note":"Priority shipping"}`,
			expected: "{\n  \"description\": \"Customer order\",\n  \"id\": \"order_123\",\n  \"note\": \"Priority shipping\",\n  \"order_data\": {\n    \"items\": [\n      {\n        \"price\": 999.99,\n        \"product\": \"laptop\"\n      }\n    ],\n    \"total\": 999.99\n  }\n}",
		},
		{
			name:     "invalid JSON",
			input:    `not valid json`,
			expected: `not valid json`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "JSON array with nested objects",
			input:    `["{\"name\":\"Alice\"}","{\"name\":\"Bob\"}"]`,
			expected: "[\n  {\n    \"name\": \"Alice\"\n  },\n  {\n    \"name\": \"Bob\"\n  }\n]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrettyPrintWithNestedExpansion(tt.input)
			if result != tt.expected {
				t.Errorf("PrettyPrintWithNestedExpansion() failed for %q\nGot:\n%s\nExpected:\n%s", tt.name, result, tt.expected)
			}
		})
	}
}

func TestExpandNestedJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "simple map",
			input:    map[string]interface{}{"name": "Alice", "age": 25},
			expected: map[string]interface{}{"name": "Alice", "age": 25},
		},
		{
			name: "map with JSON string",
			input: map[string]interface{}{
				"user_id": "123",
				"profile": `{"name":"Alice","age":30}`,
			},
			expected: map[string]interface{}{
				"user_id": "123",
				"profile": map[string]interface{}{"name": "Alice", "age": float64(30)},
			},
		},
		{
			name:     "simple array",
			input:    []interface{}{"item1", "item2"},
			expected: []interface{}{"item1", "item2"},
		},
		{
			name: "array with JSON strings",
			input: []interface{}{
				`{"name":"Alice"}`,
				`{"name":"Bob"}`,
			},
			expected: []interface{}{
				map[string]interface{}{"name": "Alice"},
				map[string]interface{}{"name": "Bob"},
			},
		},
		{
			name:     "string - not JSON",
			input:    "regular string",
			expected: "regular string",
		},
		{
			name:     "string - JSON object",
			input:    `{"name":"Alice"}`,
			expected: map[string]interface{}{"name": "Alice"},
		},
		{
			name:     "primitive - number",
			input:    42,
			expected: 42,
		},
		{
			name:     "primitive - boolean",
			input:    true,
			expected: true,
		},
		{
			name:     "primitive - nil",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandNestedJSON(tt.input)
			if !deepEqual(result, tt.expected) {
				t.Errorf("expandNestedJSON() failed for %q\nGot: %+v\nExpected: %+v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsJSONString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid JSON object",
			input:    `{"key":"value"}`,
			expected: true,
		},
		{
			name:     "valid JSON array",
			input:    `[1,2,3]`,
			expected: true,
		},
		{
			name:     "JSON object with whitespace",
			input:    `  {"key":"value"}  `,
			expected: true,
		},
		{
			name:     "JSON array with whitespace",
			input:    `  [1,2,3]  `,
			expected: true,
		},
		{
			name:     "regular string",
			input:    `regular text`,
			expected: false,
		},
		{
			name:     "quoted string",
			input:    `"just a string"`,
			expected: false,
		},
		{
			name:     "empty string",
			input:    ``,
			expected: false,
		},
		{
			name:     "incomplete JSON object",
			input:    `{incomplete`,
			expected: false,
		},
		{
			name:     "incomplete JSON array",
			input:    `[incomplete`,
			expected: false,
		},
		{
			name:     "JSON with extra characters",
			input:    `{"valid": "json"}extra`,
			expected: false,
		},
		{
			name:     "whitespace only",
			input:    `   `,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isJSONString(tt.input)
			if result != tt.expected {
				t.Errorf("isJSONString(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkPrettyPrintWithNestedExpansion(b *testing.B) {
	input := `{"user_id":"123","profile":"{\"name\":\"Alice\",\"age\":30,\"city\":\"New York\"}","preferences":"{\"theme\":\"dark\",\"notifications\":true}"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PrettyPrintWithNestedExpansion(input)
	}
}

func BenchmarkExpandNestedJSON(b *testing.B) {
	data := map[string]interface{}{
		"user_id":     "123",
		"profile":     `{"name":"Alice","age":30,"city":"New York"}`,
		"preferences": `{"theme":"dark","notifications":true}`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = expandNestedJSON(data)
	}
}

func BenchmarkIsJSONString(b *testing.B) {
	testStrings := []string{
		`{"key":"value"}`,
		`[1,2,3]`,
		`regular text`,
		`"quoted string"`,
		``,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range testStrings {
			_ = isJSONString(s)
		}
	}
}

func TestQueryJSONPath(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		jsonPath    string
		expected    string
		expectError bool
	}{
		{
			name:        "simple field access",
			jsonData:    `{"name":"Alice","age":30}`,
			jsonPath:    "$.name",
			expected:    `"Alice"`,
			expectError: false,
		},
		{
			name:        "nested field access",
			jsonData:    `{"user":{"name":"Bob","age":25}}`,
			jsonPath:    "$.user.name",
			expected:    `"Bob"`,
			expectError: false,
		},
		{
			name:        "array index access",
			jsonData:    `{"items":[{"id":1,"name":"first"},{"id":2,"name":"second"}]}`,
			jsonPath:    "$.items[0].name",
			expected:    `"first"`,
			expectError: false,
		},
		{
			name:        "all array items",
			jsonData:    `{"items":[{"id":1,"name":"first"},{"id":2,"name":"second"}]}`,
			jsonPath:    "$.items[*].name",
			expected:    `["first","second"]`,
			expectError: false,
		},
		{
			name:        "root access",
			jsonData:    `{"name":"Alice","age":30}`,
			jsonPath:    "$",
			expected:    `{"age":30,"name":"Alice"}`,
			expectError: false,
		},
		{
			name:        "number value",
			jsonData:    `{"price":99.99}`,
			jsonPath:    "$.price",
			expected:    `99.99`,
			expectError: false,
		},
		{
			name:        "boolean value",
			jsonData:    `{"active":true}`,
			jsonPath:    "$.active",
			expected:    `true`,
			expectError: false,
		},
		{
			name:        "null value",
			jsonData:    `{"value":null}`,
			jsonPath:    "$.value",
			expected:    `null`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			jsonData:    `not valid json`,
			jsonPath:    "$.name",
			expected:    "",
			expectError: true,
		},
		{
			name:        "field not found",
			jsonData:    `{"name":"Alice"}`,
			jsonPath:    "$.nonexistent",
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid jsonpath expression",
			jsonData:    `{"name":"Alice"}`,
			jsonPath:    "$[invalid",
			expected:    "",
			expectError: true,
		},
		{
			name:        "deeply nested access",
			jsonData:    `{"level1":{"level2":{"level3":{"value":"deep"}}}}`,
			jsonPath:    "$.level1.level2.level3.value",
			expected:    `"deep"`,
			expectError: false,
		},
		{
			name:        "array of primitives",
			jsonData:    `{"numbers":[1,2,3,4,5]}`,
			jsonPath:    "$.numbers[2]",
			expected:    `3`,
			expectError: false,
		},
		{
			name:        "complex nested structure",
			jsonData:    `{"store":{"book":[{"title":"Book 1","price":10.99},{"title":"Book 2","price":15.50}]}}`,
			jsonPath:    "$.store.book[1].price",
			expected:    `15.5`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := QueryJSONPath(tt.jsonData, tt.jsonPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("QueryJSONPath() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("QueryJSONPath() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("QueryJSONPath() failed for %q\nGot: %s\nExpected: %s", tt.name, result, tt.expected)
				}
			}
		})
	}
}

func TestIsValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid JSON object",
			input:    `{"name":"Alice"}`,
			expected: true,
		},
		{
			name:     "valid JSON array",
			input:    `[1,2,3]`,
			expected: true,
		},
		{
			name:     "invalid JSON",
			input:    `not valid`,
			expected: false,
		},
		{
			name:     "empty string",
			input:    ``,
			expected: false,
		},
		{
			name:     "valid JSON string",
			input:    `"hello"`,
			expected: true,
		},
		{
			name:     "valid JSON number",
			input:    `42`,
			expected: true,
		},
		{
			name:     "valid JSON boolean",
			input:    `true`,
			expected: true,
		},
		{
			name:     "valid JSON null",
			input:    `null`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidJSON(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidJSON(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Benchmark tests for JSONPath
func BenchmarkQueryJSONPath(b *testing.B) {
	jsonData := `{"user":{"name":"Alice","age":30,"address":{"city":"New York","zip":"10001"}}}`
	jsonPath := "$.user.address.city"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = QueryJSONPath(jsonData, jsonPath)
	}
}

// Helper function for deep comparison since Go doesn't have built-in deep equality for interfaces
func deepEqual(a, b interface{}) bool {
	switch va := a.(type) {
	case map[string]interface{}:
		vb, ok := b.(map[string]interface{})
		if !ok || len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			if !deepEqual(v, vb[k]) {
				return false
			}
		}
		return true
	case []interface{}:
		vb, ok := b.([]interface{})
		if !ok || len(va) != len(vb) {
			return false
		}
		for i, v := range va {
			if !deepEqual(v, vb[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
