package transform

import (
	"encoding/json"
	"os"
	"testing"
)

// TestPythonExecutor_SimpleExpression tests simple Python expression execution
func TestPythonExecutor_SimpleExpression(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		context  map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "uppercase string",
			expr:     "value.upper()",
			context:  map[string]interface{}{"value": "hello"},
			expected: "HELLO",
			wantErr:  false,
		},
		{
			name:     "lowercase string",
			expr:     "value.lower()",
			context:  map[string]interface{}{"value": "WORLD"},
			expected: "world",
			wantErr:  false,
		},
		{
			name:     "string concatenation",
			expr:     "value + '_suffix'",
			context:  map[string]interface{}{"value": "prefix"},
			expected: "prefix_suffix",
			wantErr:  false,
		},
		{
			name:     "numeric operation",
			expr:     "str(int(value) * 2)",
			context:  map[string]interface{}{"value": "10"},
			expected: "20",
			wantErr:  false,
		},
	}

	executor := NewPythonExecutor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecuteExpression(tt.expr, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("ExecuteExpression() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPythonExecutor_JSONParsing tests JSON parsing and modification
func TestPythonExecutor_JSONParsing(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		value    string
		expected string
		wantErr  bool
	}{
		{
			name: "parse and modify JSON field",
			expr: `
import json
data = json.loads(value)
data['age'] = 30
json.dumps(data)
`,
			value:    `{"name":"Alice","age":25}`,
			expected: `{"name":"Alice","age":30}`,
			wantErr:  false,
		},
		{
			name: "add new JSON field",
			expr: `
import json
data = json.loads(value)
data['updated'] = True
json.dumps(data)
`,
			value:    `{"name":"Bob"}`,
			expected: `{"name":"Bob","updated":true}`,
			wantErr:  false,
		},
		{
			name: "filter JSON by field",
			expr: `
import json
data = json.loads(value)
data.get('age', 0) > 18
`,
			value:    `{"name":"Charlie","age":25}`,
			expected: "True",
			wantErr:  false,
		},
	}

	executor := NewPythonExecutor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := map[string]interface{}{"value": tt.value}
			result, err := executor.ExecuteExpression(tt.expr, context)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Compare JSON objects, not strings (field order doesn't matter)
				if isJSONEqual(result.(string), tt.expected) {
					return
				}
				t.Errorf("ExecuteExpression() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// isJSONEqual compares two JSON strings for equality, ignoring field order
func isJSONEqual(a, b string) bool {
	// Try to parse both as JSON
	var objA, objB interface{}
	if err := json.Unmarshal([]byte(a), &objA); err != nil {
		// Not JSON, compare as strings
		return a == b
	}
	if err := json.Unmarshal([]byte(b), &objB); err != nil {
		return a == b
	}
	// Compare the parsed objects
	aJSON, _ := json.Marshal(objA)
	bJSON, _ := json.Marshal(objB)
	return string(aJSON) == string(bJSON)
}

// TestPythonExecutor_FilterExpression tests filter expressions
func TestPythonExecutor_FilterExpression(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		key      string
		value    string
		expected bool
		wantErr  bool
	}{
		{
			name:     "filter by key prefix",
			expr:     "key.startswith('user:')",
			key:      "user:123",
			value:    "data",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "filter by value content",
			expr:     "'admin' in value",
			key:      "user:123",
			value:    "admin_user",
			expected: true,
			wantErr:  false,
		},
		{
			name: "filter by JSON field",
			expr: `
import json
data = json.loads(value)
data.get('active', False) == True
`,
			key:      "user:123",
			value:    `{"name":"Alice","active":true}`,
			expected: true,
			wantErr:  false,
		},
	}

	executor := NewPythonExecutor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := map[string]interface{}{
				"key":   tt.key,
				"value": tt.value,
			}
			result, err := executor.ExecuteExpression(tt.expr, context)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				boolResult := result == "True" || result == "true" || result == true
				if boolResult != tt.expected {
					t.Errorf("ExecuteExpression() = %v, want %v", boolResult, tt.expected)
				}
			}
		})
	}
}

// TestPythonExecutor_ErrorHandling tests error handling
func TestPythonExecutor_ErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		context map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "syntax error",
			expr:    "value.upper(",
			context: map[string]interface{}{"value": "test"},
			wantErr: true,
			errMsg:  "syntax",
		},
		{
			name:    "undefined variable",
			expr:    "undefined_var.upper()",
			context: map[string]interface{}{"value": "test"},
			wantErr: true,
			errMsg:  "undefined",
		},
		{
			name:    "type error",
			expr:    "value.upper()",
			context: map[string]interface{}{"value": 123},
			wantErr: true,
			errMsg:  "type",
		},
		{
			name:    "import error",
			expr:    "import nonexistent_module",
			context: map[string]interface{}{"value": "test"},
			wantErr: true,
			errMsg:  "import",
		},
	}

	executor := NewPythonExecutor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.ExecuteExpression(tt.expr, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteExpression() error = %v, wantErr %v", err, tt.wantErr)
			}
			// TODO: Check error message contains expected substring
		})
	}
}

// TestPythonExecutor_ValidateExpression tests expression validation
func TestPythonExecutor_ValidateExpression(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{
			name:    "valid expression",
			expr:    "value.upper()",
			wantErr: false,
		},
		{
			name:    "valid multiline expression",
			expr:    "import json\ndata = json.loads(value)\ndata['new'] = True",
			wantErr: false,
		},
		{
			name:    "invalid syntax",
			expr:    "value.upper(",
			wantErr: true,
		},
		{
			name:    "empty expression",
			expr:    "",
			wantErr: true,
		},
	}

	executor := NewPythonExecutor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateExpression(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExpression() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPythonExecutor_ScriptFile tests script file execution
func TestPythonExecutor_ScriptFile(t *testing.T) {
	executor := NewPythonExecutor()
	
	tests := []struct {
		name              string
		scriptContent     string
		key               string
		value             string
		expectedKey       string
		expectedValue     string
		expectSkip        bool // True if should_process returns False
		wantErr           bool
	}{
		{
			name: "basic transform_value",
			scriptContent: `
def transform_value(key, value):
    return value.upper()
`,
			key:           "test:1",
			value:         "hello world",
			expectedKey:   "test:1",
			expectedValue: "HELLO WORLD",
			expectSkip:    false,
			wantErr:       false,
		},
		{
			name: "json transformation",
			scriptContent: `
import json

def transform_value(key, value):
    data = json.loads(value)
    data['name'] = data['name'].upper()
    return json.dumps(data)
`,
			key:           "user:1",
			value:         `{"name":"alice","age":25}`,
			expectedKey:   "user:1",
			expectedValue: `{"name": "ALICE", "age": 25}`,
			expectSkip:    false,
			wantErr:       false,
		},
		{
			name: "filter passes - should_process returns True",
			scriptContent: `
def should_process(key, value):
    return 'alice' in value.lower()

def transform_value(key, value):
    return value.upper()
`,
			key:           "user:1",
			value:         "Alice Smith",
			expectedKey:   "user:1",
			expectedValue: "ALICE SMITH",
			expectSkip:    false,
			wantErr:       false,
		},
		{
			name: "filter fails - should_process returns False",
			scriptContent: `
def should_process(key, value):
    return 'bob' in value.lower()

def transform_value(key, value):
    return value.upper()
`,
			key:         "user:1",
			value:       "Alice Smith",
			expectSkip:  true,
			wantErr:     false,
		},
		{
			name: "filter with JSON",
			scriptContent: `
import json

def should_process(key, value):
    try:
        data = json.loads(value)
        return data.get('age', 0) > 30
    except:
        return False

def transform_value(key, value):
    data = json.loads(value)
    data['senior'] = True
    return json.dumps(data)
`,
			key:           "user:1",
			value:         `{"name":"Bob","age":35}`,
			expectedKey:   "user:1",
			expectedValue: `{"name": "Bob", "age": 35, "senior": true}`,
			expectSkip:    false,
			wantErr:       false,
		},
		{
			name: "only should_process, no transform",
			scriptContent: `
def should_process(key, value):
    return True
`,
			key:           "test:1",
			value:         "original",
			expectedKey:   "test:1",
			expectedValue: "original",
			expectSkip:    false,
			wantErr:       false,
		},
		{
			name: "error in transform_value",
			scriptContent: `
def transform_value(key, value):
    return undefined_variable
`,
			key:     "test:1",
			value:   "hello",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary script file
			tmpfile, err := createTempScript(tt.scriptContent)
			if err != nil {
				t.Fatalf("Failed to create temp script: %v", err)
			}
			defer removeTempScript(tmpfile)
			
			// Execute script
			resultKey, resultValue, err := executor.ExecuteScript(tmpfile, tt.key, tt.value)
			
			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				return // Test passed, error expected
			}
			
			// Check if entry should be skipped
			if tt.expectSkip {
				if resultValue != "" {
					t.Errorf("Expected skip (empty value), got value: %s", resultValue)
				}
				return
			}
			
			// Check results
			if resultKey != tt.expectedKey {
				t.Errorf("ExecuteScript() key = %v, want %v", resultKey, tt.expectedKey)
			}
			
			// For JSON comparison, normalize format
			if isJSONEqual(resultValue, tt.expectedValue) {
				// JSON comparison passed
			} else if resultValue != tt.expectedValue {
				t.Errorf("ExecuteScript() value = %v, want %v", resultValue, tt.expectedValue)
			}
		})
	}
}

// TestPythonExecutor_ScriptFileNotFound tests error when script file doesn't exist
func TestPythonExecutor_ScriptFileNotFound(t *testing.T) {
	executor := NewPythonExecutor()
	
	_, _, err := executor.ExecuteScript("/nonexistent/script.py", "key1", "value1")
	if err == nil {
		t.Error("Expected error for nonexistent script file, got nil")
	}
}

// Helper functions for script file tests
func createTempScript(content string) (string, error) {
	tmpfile, err := os.CreateTemp("", "test_script_*.py")
	if err != nil {
		return "", err
	}
	
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return "", err
	}
	
	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		return "", err
	}
	
	return tmpfile.Name(), nil
}

func removeTempScript(filename string) {
	os.Remove(filename)
}

// TestPythonExecutor_Timeout tests execution timeout
func TestPythonExecutor_Timeout(t *testing.T) {
	executor := NewPythonExecutor()
	
	// Infinite loop should timeout
	expr := "while True: pass"
	context := map[string]interface{}{"value": "test"}
	
	_, err := executor.ExecuteExpression(expr, context)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

// TestPythonExecutor_MemoryLimit tests memory usage limits
func TestPythonExecutor_MemoryLimit(t *testing.T) {
	// TODO: Test memory limits
	t.Skip("Memory limit tests to be implemented")
}
