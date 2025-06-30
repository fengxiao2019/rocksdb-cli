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
