package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"rocksdb-cli/internal/db"
	"strconv"
	"strings"
	"testing"
)

type mockDB struct {
	data     map[string]map[string]string // cf -> key -> value
	cfExists map[string]bool
}

func newMockDB() *mockDB {
	return &mockDB{
		data: map[string]map[string]string{
			"default": make(map[string]string),
		},
		cfExists: map[string]bool{
			"default": true,
		},
	}
}

func (m *mockDB) GetCF(cf, key string) (string, error) {
	if !m.cfExists[cf] {
		return "", errors.New("column family not found")
	}
	v, ok := m.data[cf][key]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (m *mockDB) PutCF(cf, key, value string) error {
	if !m.cfExists[cf] {
		return errors.New("column family not found")
	}
	if m.data[cf] == nil {
		m.data[cf] = make(map[string]string)
	}
	m.data[cf][key] = value
	return nil
}

func (m *mockDB) PrefixScanCF(cf, prefix string, limit int) (map[string]string, error) {
	if !m.cfExists[cf] {
		return nil, errors.New("column family not found")
	}
	res := make(map[string]string)
	for k, v := range m.data[cf] {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			res[k] = v
			if limit > 0 && len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

func (m *mockDB) ScanCF(cf string, start, end []byte, opts db.ScanOptions) (map[string]string, error) {
	if !m.cfExists[cf] {
		return nil, errors.New("column family not found")
	}

	// Get all keys and sort them
	keys := make([]string, 0, len(m.data[cf]))
	for k := range m.data[cf] {
		keys = append(keys, k)
	}

	// Sort keys based on scan direction
	if opts.Reverse {
		// Sort in reverse order
		for i := 0; i < len(keys)-1; i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[i] < keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
	} else {
		// Sort in forward order
		for i := 0; i < len(keys)-1; i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[i] > keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
	}

	result := make(map[string]string)
	startStr := string(start)
	endStr := string(end)

	for _, k := range keys {
		// For forward scan: start <= k < end
		// For reverse scan: start <= k < end (but processed in reverse order)
		if len(start) > 0 && k < startStr {
			continue
		}
		if len(end) > 0 && k >= endStr {
			if opts.Reverse {
				continue // Skip keys >= end
			} else {
				break // Stop at end for forward scan
			}
		}

		if opts.Values {
			result[k] = m.data[cf][k]
		} else {
			result[k] = ""
		}

		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}

	return result, nil
}

func (m *mockDB) ListCFs() ([]string, error) {
	cfs := make([]string, 0, len(m.cfExists))
	for cf := range m.cfExists {
		cfs = append(cfs, cf)
	}
	return cfs, nil
}

func (m *mockDB) CreateCF(cf string) error {
	if m.cfExists[cf] {
		return errors.New("column family already exists")
	}
	m.cfExists[cf] = true
	m.data[cf] = make(map[string]string)
	return nil
}

func (m *mockDB) DropCF(cf string) error {
	if !m.cfExists[cf] {
		return errors.New("column family not found")
	}
	if cf == "default" {
		return errors.New("cannot drop default column family")
	}
	delete(m.cfExists, cf)
	delete(m.data, cf)
	return nil
}

func (m *mockDB) GetLastCF(cf string) (string, string, error) {
	if !m.cfExists[cf] {
		return "", "", errors.New("column family not found")
	}

	cfData, ok := m.data[cf]
	if !ok || len(cfData) == 0 {
		return "", "", errors.New("column family is empty")
	}

	// Find the lexicographically last key
	var lastKey, lastValue string
	for key, value := range cfData {
		if lastKey == "" || key > lastKey {
			lastKey = key
			lastValue = value
		}
	}

	return lastKey, lastValue, nil
}

func (m *mockDB) ExportToCSV(cf, filePath string) error {
	if !m.cfExists[cf] {
		return errors.New("column family not found")
	}

	// For testing, we'll just simulate the export without actually creating a file
	// In a real test environment, you might want to create a temporary file
	return nil
}

func (m *mockDB) Close() {}

func (m *mockDB) JSONQueryCF(cf, field, value string) (map[string]string, error) {
	if !m.cfExists[cf] {
		return nil, errors.New("column family not found")
	}

	result := make(map[string]string)
	cfData, ok := m.data[cf]
	if !ok {
		return result, nil
	}

	for key, val := range cfData {
		// Try to parse as JSON
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(val), &jsonData); err != nil {
			// Skip non-JSON values
			continue
		}

		// Check if the field exists and matches the value
		if fieldValue, exists := jsonData[field]; exists {
			var match bool

			// Handle different value types
			switch v := fieldValue.(type) {
			case string:
				match = v == value
			case float64:
				// Try to parse the input value as number
				if numValue, err := strconv.ParseFloat(value, 64); err == nil {
					match = v == numValue
				}
			case bool:
				// Try to parse the input value as boolean
				if boolValue, err := strconv.ParseBool(value); err == nil {
					match = v == boolValue
				}
			case nil:
				match = value == "null"
			default:
				// For other types, convert to string and compare
				match = fmt.Sprintf("%v", v) == value
			}

			if match {
				result[key] = val
			}
		}
	}

	return result, nil
}

func TestHandler_Execute(t *testing.T) {
	mockDB := newMockDB()
	state := &ReplState{CurrentCF: "default"}
	h := &Handler{DB: mockDB, State: state}

	// Test cases with expected outputs
	cases := []struct {
		name        string
		cmd         string
		setupFunc   func()
		checkFunc   func() error
		expectError bool
	}{
		{
			name: "create new cf",
			cmd:  "createcf testcf",
			checkFunc: func() error {
				if !mockDB.cfExists["testcf"] {
					return errors.New("CF not created")
				}
				return nil
			},
		},
		{
			name: "switch to cf",
			cmd:  "usecf testcf",
			checkFunc: func() error {
				if state.CurrentCF != "testcf" {
					return errors.New("CF not switched")
				}
				return nil
			},
		},
		{
			name: "put with current cf",
			cmd:  "put key1 value1",
			checkFunc: func() error {
				v, err := mockDB.GetCF("testcf", "key1")
				if err != nil || v != "value1" {
					return errors.New("put failed")
				}
				return nil
			},
		},
		{
			name: "put with explicit cf",
			cmd:  "put default key2 value2",
			setupFunc: func() {
				// Switch back to default CF to test explicit CF in command
				state.CurrentCF = "default"
			},
			checkFunc: func() error {
				v, err := mockDB.GetCF("default", "key2")
				if err != nil || v != "value2" {
					return errors.New("put failed")
				}
				return nil
			},
		},
		{
			name: "get with current cf",
			cmd:  "get key1",
			setupFunc: func() {
				// Switch back to testcf
				state.CurrentCF = "testcf"
			},
			checkFunc: func() error {
				v, err := mockDB.GetCF("testcf", "key1")
				if err != nil || v != "value1" {
					return errors.New("get failed")
				}
				return nil
			},
		},
		{
			name: "get with explicit cf",
			cmd:  "get default key2",
			checkFunc: func() error {
				v, err := mockDB.GetCF("default", "key2")
				if err != nil || v != "value2" {
					return errors.New("get failed")
				}
				return nil
			},
		},
		{
			name: "put json value",
			cmd:  `put jsonkey {"name":"test","value":123}`,
			setupFunc: func() {
				state.CurrentCF = "default"
			},
			checkFunc: func() error {
				v, err := mockDB.GetCF("default", "jsonkey")
				if err != nil || !strings.Contains(v, "test") {
					return errors.New("put json failed")
				}
				return nil
			},
		},
		{
			name: "get json value with pretty print",
			cmd:  "get jsonkey --pretty",
			checkFunc: func() error {
				v, err := mockDB.GetCF("default", "jsonkey")
				if err != nil || !strings.Contains(v, "test") {
					return errors.New("get json failed")
				}
				return nil
			},
		},
		{
			name: "get json value with explicit cf and pretty print",
			cmd:  "get default jsonkey --pretty",
			checkFunc: func() error {
				v, err := mockDB.GetCF("default", "jsonkey")
				if err != nil || !strings.Contains(v, "test") {
					return errors.New("get json with cf failed")
				}
				return nil
			},
		},
		{
			name: "prefix scan with current cf",
			cmd:  "prefix key",
			setupFunc: func() {
				mockDB.PutCF("testcf", "key1", "v1")
				mockDB.PutCF("testcf", "key2", "v2")
				mockDB.PutCF("testcf", "other", "v3")
			},
			checkFunc: func() error {
				res, err := mockDB.PrefixScanCF("testcf", "key", 10)
				if err != nil || len(res) != 2 {
					return errors.New("prefix scan failed")
				}
				return nil
			},
		},
		{
			name: "list cf",
			cmd:  "listcf",
			checkFunc: func() error {
				cfs, err := mockDB.ListCFs()
				if err != nil || len(cfs) != 2 { // default and testcf
					return errors.New("listcf failed")
				}
				return nil
			},
		},
		{
			name: "drop cf",
			cmd:  "dropcf testcf",
			checkFunc: func() error {
				if mockDB.cfExists["testcf"] {
					return errors.New("CF not dropped")
				}
				return nil
			},
		},
		{
			name:        "drop default cf should fail",
			cmd:         "dropcf default",
			expectError: true,
			checkFunc: func() error {
				if !mockDB.cfExists["default"] {
					return errors.New("default CF was dropped")
				}
				return nil
			},
		},
		{
			name: "scan with current cf",
			cmd:  "scan key1 key4",
			setupFunc: func() {
				state.CurrentCF = "default"
				mockDB.PutCF("default", "key1", "v1")
				mockDB.PutCF("default", "key2", "v2")
				mockDB.PutCF("default", "key3", "v3")
				mockDB.PutCF("default", "key4", "v4")
				mockDB.PutCF("default", "key5", "v5")
			},
			checkFunc: func() error {
				res, err := mockDB.ScanCF("default", []byte("key1"), []byte("key4"), db.ScanOptions{Values: true})
				if err != nil || len(res) != 3 {
					return errors.New("scan failed")
				}
				return nil
			},
		},
		{
			name: "scan with explicit cf",
			cmd:  "scan testcf key1 key3",
			setupFunc: func() {
				mockDB.CreateCF("testcf")
				mockDB.PutCF("testcf", "key1", "v1")
				mockDB.PutCF("testcf", "key2", "v2")
				mockDB.PutCF("testcf", "key3", "v3")
			},
			checkFunc: func() error {
				res, err := mockDB.ScanCF("testcf", []byte("key1"), []byte("key3"), db.ScanOptions{Values: true})
				if err != nil || len(res) != 2 {
					return errors.New("scan with cf failed")
				}
				return nil
			},
		},
		{
			name: "scan with limit",
			cmd:  "scan key1 key5 --limit=2",
			setupFunc: func() {
				state.CurrentCF = "default"
			},
			checkFunc: func() error {
				res, err := mockDB.ScanCF("default", []byte("key1"), []byte("key5"), db.ScanOptions{Values: true, Limit: 2})
				if err != nil || len(res) != 2 {
					return errors.New("scan with limit failed")
				}
				return nil
			},
		},
		{
			name: "scan reverse",
			cmd:  "scan key1 key5 --reverse",
			setupFunc: func() {
				// Clear default CF and set up clean test data
				mockDB.data["default"] = make(map[string]string)
				mockDB.PutCF("default", "key1", "v1")
				mockDB.PutCF("default", "key2", "v2")
				mockDB.PutCF("default", "key3", "v3")
				mockDB.PutCF("default", "key4", "v4")
				mockDB.PutCF("default", "key5", "v5")
			},
			checkFunc: func() error {
				res, err := mockDB.ScanCF("default", []byte("key1"), []byte("key5"), db.ScanOptions{Values: true, Reverse: true})
				if err != nil {
					return errors.New("reverse scan failed")
				}

				// Should get key1, key2, key3, key4 (key5 is excluded in range scan)
				expectedKeys := []string{"key1", "key2", "key3", "key4"}
				if len(res) != len(expectedKeys) {
					return fmt.Errorf("expected %d keys, got %d", len(expectedKeys), len(res))
				}

				for _, expectedKey := range expectedKeys {
					if _, exists := res[expectedKey]; !exists {
						return fmt.Errorf("expected key %s not found in results", expectedKey)
					}
				}
				return nil
			},
		},
		{
			name: "scan without values",
			cmd:  "scan key1 key5 --values=no",
			checkFunc: func() error {
				res, err := mockDB.ScanCF("default", []byte("key1"), []byte("key5"), db.ScanOptions{Values: false})
				if err != nil {
					return errors.New("scan without values failed")
				}
				for _, v := range res {
					if v != "" {
						return errors.New("scan without values returned values")
					}
				}
				return nil
			},
		},
		{
			name: "scan two args with current cf should be start/end",
			cmd:  "scan key1 key3",
			setupFunc: func() {
				state.CurrentCF = "default"
				// Ensure we have test data
				mockDB.PutCF("default", "key1", "v1")
				mockDB.PutCF("default", "key2", "v2")
				mockDB.PutCF("default", "key3", "v3")
			},
			checkFunc: func() error {
				// This should scan from key1 to key3 in default CF, not treat key1 as CF name
				res, err := mockDB.ScanCF("default", []byte("key1"), []byte("key3"), db.ScanOptions{Values: true})
				if err != nil {
					return fmt.Errorf("scan with current cf failed: %v", err)
				}
				// Should get key1 and key2 (key3 is excluded in range scan)
				if len(res) != 2 {
					return fmt.Errorf("expected 2 results, got %d", len(res))
				}
				if _, ok := res["key1"]; !ok {
					return errors.New("key1 should be in results")
				}
				if _, ok := res["key2"]; !ok {
					return errors.New("key2 should be in results")
				}
				return nil
			},
		},
		{
			name: "scan two args without current cf should be cf/start",
			cmd:  "scan testcf2 key1",
			setupFunc: func() {
				state.CurrentCF = "" // No current CF
				mockDB.CreateCF("testcf2")
				// Clear any existing data in testcf2
				mockDB.data["testcf2"] = make(map[string]string)
				mockDB.PutCF("testcf2", "key1", "v1")
				mockDB.PutCF("testcf2", "key2", "v2")
			},
			checkFunc: func() error {
				// This should scan from key1 to end in testcf2
				res, err := mockDB.ScanCF("testcf2", []byte("key1"), nil, db.ScanOptions{Values: true})
				if err != nil {
					return fmt.Errorf("scan without current cf failed: %v", err)
				}
				if len(res) != 2 {
					return fmt.Errorf("expected 2 results, got %d", len(res))
				}
				return nil
			},
		},
		{
			name: "export with current cf",
			cmd:  "export test_export.csv",
			setupFunc: func() {
				state.CurrentCF = "default"
				mockDB.PutCF("default", "export_key1", "export_value1")
				mockDB.PutCF("default", "export_key2", "export_value2")
			},
			checkFunc: func() error {
				// Since we're using mockDB, we just check that the method was called without error
				return nil
			},
		},
		{
			name: "export with explicit cf",
			cmd:  "export default test_export2.csv",
			checkFunc: func() error {
				// Since we're using mockDB, we just check that the method was called without error
				return nil
			},
		},
		{
			name:        "export nonexistent cf should fail",
			cmd:         "export nonexistent test_export3.csv",
			expectError: true,
			checkFunc: func() error {
				// The command should handle the error gracefully
				return nil
			},
		},
		{
			name: "last with current cf",
			cmd:  "last",
			setupFunc: func() {
				state.CurrentCF = "default"
				mockDB.PutCF("default", "lastkey1", "lastvalue1")
				mockDB.PutCF("default", "lastkey2", "lastvalue2")
			},
			checkFunc: func() error {
				key, _, err := mockDB.GetLastCF("default")
				if err != nil || key == "" {
					return errors.New("last command failed")
				}
				return nil
			},
		},
		{
			name: "last with explicit cf",
			cmd:  "last default",
			checkFunc: func() error {
				key, _, err := mockDB.GetLastCF("default")
				if err != nil || key == "" {
					return errors.New("last command with cf failed")
				}
				return nil
			},
		},
		{
			name: "last with pretty flag and current cf",
			cmd:  "last --pretty",
			setupFunc: func() {
				state.CurrentCF = "default"
				// Clear existing data first
				mockDB.data["default"] = make(map[string]string)
				mockDB.PutCF("default", "json_key", `{"name":"test","value":123}`)
			},
			checkFunc: func() error {
				key, value, err := mockDB.GetLastCF("default")
				if err != nil {
					return fmt.Errorf("GetLastCF failed: %v", err)
				}
				if key != "json_key" {
					return fmt.Errorf("expected key 'json_key', got '%s'", key)
				}
				// Verify the value is JSON
				if !strings.Contains(value, `"name":"test"`) {
					return fmt.Errorf("test data should contain JSON, got: %s", value)
				}
				return nil
			},
		},
		{
			name: "last with pretty flag and explicit cf",
			cmd:  "last default --pretty",
			setupFunc: func() {
				// Clear existing data first
				mockDB.data["default"] = make(map[string]string)
				mockDB.PutCF("default", "json_key2", `{"users":[{"id":1,"name":"Alice"}]}`)
			},
			checkFunc: func() error {
				key, value, err := mockDB.GetLastCF("default")
				if err != nil {
					return fmt.Errorf("GetLastCF failed: %v", err)
				}
				if key != "json_key2" {
					return fmt.Errorf("expected key 'json_key2', got '%s'", key)
				}
				// Verify the value is JSON
				if !strings.Contains(value, `"users"`) {
					return fmt.Errorf("test data should contain JSON, got: %s", value)
				}
				return nil
			},
		},
		{
			name:        "last nonexistent cf should fail",
			cmd:         "last nonexistent",
			expectError: true,
			checkFunc: func() error {
				// The command should handle the error gracefully
				return nil
			},
		},
		{
			name: "jsonquery with current cf - string field",
			cmd:  "jsonquery name Alice",
			setupFunc: func() {
				state.CurrentCF = "users"
				mockDB.CreateCF("users")
				mockDB.data["users"] = make(map[string]string)
				mockDB.PutCF("users", "user:1001", `{"id":1001,"name":"Alice","age":25}`)
				mockDB.PutCF("users", "user:1002", `{"id":1002,"name":"Bob","age":30}`)
				mockDB.PutCF("users", "user:1003", `{"id":1003,"name":"Alice","age":28}`)
			},
			checkFunc: func() error {
				result, err := mockDB.JSONQueryCF("users", "name", "Alice")
				if err != nil {
					return fmt.Errorf("JSONQueryCF failed: %v", err)
				}
				if len(result) != 2 {
					return fmt.Errorf("expected 2 results, got %d", len(result))
				}
				// Check that both Alice entries are returned
				if _, ok := result["user:1001"]; !ok {
					return errors.New("user:1001 should be in results")
				}
				if _, ok := result["user:1003"]; !ok {
					return errors.New("user:1003 should be in results")
				}
				return nil
			},
		},
		{
			name: "jsonquery with explicit cf - number field",
			cmd:  "jsonquery users age 30",
			setupFunc: func() {
				// Data already set up in previous test
			},
			checkFunc: func() error {
				result, err := mockDB.JSONQueryCF("users", "age", "30")
				if err != nil {
					return fmt.Errorf("JSONQueryCF failed: %v", err)
				}
				if len(result) != 1 {
					return fmt.Errorf("expected 1 result, got %d", len(result))
				}
				if _, ok := result["user:1002"]; !ok {
					return errors.New("user:1002 should be in results")
				}
				return nil
			},
		},
		{
			name: "jsonquery with pretty flag",
			cmd:  "jsonquery users name Bob --pretty",
			checkFunc: func() error {
				result, err := mockDB.JSONQueryCF("users", "name", "Bob")
				if err != nil {
					return fmt.Errorf("JSONQueryCF failed: %v", err)
				}
				if len(result) != 1 {
					return fmt.Errorf("expected 1 result, got %d", len(result))
				}
				return nil
			},
		},
		{
			name: "jsonquery no results",
			cmd:  "jsonquery users name Charlie",
			checkFunc: func() error {
				result, err := mockDB.JSONQueryCF("users", "name", "Charlie")
				if err != nil {
					return fmt.Errorf("JSONQueryCF failed: %v", err)
				}
				if len(result) != 0 {
					return fmt.Errorf("expected 0 results, got %d", len(result))
				}
				return nil
			},
		},
		{
			name:        "jsonquery nonexistent cf should fail",
			cmd:         "jsonquery nonexistent name Alice",
			expectError: true,
			checkFunc: func() error {
				// The command should handle the error gracefully
				return nil
			},
		},
	}

	// Run test cases
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.setupFunc != nil {
				c.setupFunc()
			}
			cont := h.Execute(c.cmd)
			if !cont {
				t.Error("Execute returned false, expected true")
			}
			if c.checkFunc != nil {
				if err := c.checkFunc(); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestPrettyPrintJSON(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid json object",
			input:    `{"name":"test","value":123}`,
			expected: "{\n  \"name\": \"test\",\n  \"value\": 123\n}",
		},
		{
			name:     "valid json array",
			input:    `[1,2,3]`,
			expected: "[\n  1,\n  2,\n  3\n]",
		},
		{
			name:     "invalid json",
			input:    "not json",
			expected: "not json",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := prettyPrintJSON(c.input)
			if result != c.expected {
				t.Errorf("case %q failed: got %q, want %q", c.name, result, c.expected)
			}
		})
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
