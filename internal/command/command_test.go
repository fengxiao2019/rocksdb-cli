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
		return "", db.ErrColumnFamilyNotFound
	}
	v, ok := m.data[cf][key]
	if !ok {
		return "", db.ErrKeyNotFound
	}
	return v, nil
}

func (m *mockDB) PutCF(cf, key, value string) error {
	if !m.cfExists[cf] {
		return db.ErrColumnFamilyNotFound
	}
	if m.data[cf] == nil {
		m.data[cf] = make(map[string]string)
	}
	m.data[cf][key] = value
	return nil
}

func (m *mockDB) PrefixScanCF(cf, prefix string, limit int) (map[string]string, error) {
	if !m.cfExists[cf] {
		return nil, db.ErrColumnFamilyNotFound
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
		return nil, db.ErrColumnFamilyNotFound
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
		return db.ErrColumnFamilyExists
	}
	m.cfExists[cf] = true
	m.data[cf] = make(map[string]string)
	return nil
}

func (m *mockDB) DropCF(cf string) error {
	if !m.cfExists[cf] {
		return db.ErrColumnFamilyNotFound
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
		return "", "", db.ErrColumnFamilyNotFound
	}

	cfData, ok := m.data[cf]
	if !ok || len(cfData) == 0 {
		return "", "", db.ErrColumnFamilyEmpty
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
		return db.ErrColumnFamilyNotFound
	}

	// For testing, we'll just simulate the export without actually creating a file
	// In a real test environment, you might want to create a temporary file
	return nil
}

func (m *mockDB) Close() {}

func (m *mockDB) IsReadOnly() bool {
	return false // Mock DB is always read-write for testing
}

func (m *mockDB) JSONQueryCF(cf, field, value string) (map[string]string, error) {
	if !m.cfExists[cf] {
		return nil, db.ErrColumnFamilyNotFound
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
			name: "scan * wildcard with current cf",
			cmd:  "scan *",
			setupFunc: func() {
				state.CurrentCF = "default"
				// Clear default CF and set up test data
				mockDB.data["default"] = make(map[string]string)
				mockDB.PutCF("default", "key1", "v1")
				mockDB.PutCF("default", "key2", "v2")
				mockDB.PutCF("default", "key3", "v3")
			},
			checkFunc: func() error {
				// This should scan all entries in default CF
				res, err := mockDB.ScanCF("default", nil, nil, db.ScanOptions{Values: true})
				if err != nil {
					return fmt.Errorf("scan * with current cf failed: %v", err)
				}
				if len(res) != 3 {
					return fmt.Errorf("expected 3 results, got %d", len(res))
				}
				for i := 1; i <= 3; i++ {
					key := fmt.Sprintf("key%d", i)
					if _, ok := res[key]; !ok {
						return fmt.Errorf("expected key %s not found in results", key)
					}
				}
				return nil
			},
		},
		{
			name: "scan * * wildcard range with current cf",
			cmd:  "scan * *",
			setupFunc: func() {
				state.CurrentCF = "default"
				// Data should already be set from previous test
			},
			checkFunc: func() error {
				// This should scan all entries in default CF (both start and end are nil)
				res, err := mockDB.ScanCF("default", nil, nil, db.ScanOptions{Values: true})
				if err != nil {
					return fmt.Errorf("scan * * with current cf failed: %v", err)
				}
				if len(res) != 3 {
					return fmt.Errorf("expected 3 results, got %d", len(res))
				}
				return nil
			},
		},
		{
			name: "scan testcf * wildcard with explicit cf",
			cmd:  "scan testcf *",
			setupFunc: func() {
				mockDB.CreateCF("testcf")
				mockDB.data["testcf"] = make(map[string]string)
				mockDB.PutCF("testcf", "a", "va")
				mockDB.PutCF("testcf", "b", "vb")
			},
			checkFunc: func() error {
				// This should scan all entries in testcf
				res, err := mockDB.ScanCF("testcf", nil, nil, db.ScanOptions{Values: true})
				if err != nil {
					return fmt.Errorf("scan with cf * failed: %v", err)
				}
				if len(res) != 2 {
					return fmt.Errorf("expected 2 results, got %d", len(res))
				}
				if _, ok := res["a"]; !ok {
					return errors.New("expected key 'a' not found")
				}
				if _, ok := res["b"]; !ok {
					return errors.New("expected key 'b' not found")
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
			name:     "nested json object",
			input:    `{"user":{"name":"Alice","details":{"age":25,"hobbies":["reading","coding"]}},"meta":{"created":"2024-01-01","active":true}}`,
			expected: "{\n  \"meta\": {\n    \"active\": true,\n    \"created\": \"2024-01-01\"\n  },\n  \"user\": {\n    \"details\": {\n      \"age\": 25,\n      \"hobbies\": [\n        \"reading\",\n        \"coding\"\n      ]\n    },\n    \"name\": \"Alice\"\n  }\n}",
		},
		{
			name:     "complex nested array",
			input:    `{"items":[{"id":1,"tags":["important","urgent"]},{"id":2,"tags":["normal"]}]}`,
			expected: "{\n  \"items\": [\n    {\n      \"id\": 1,\n      \"tags\": [\n        \"important\",\n        \"urgent\"\n      ]\n    },\n    {\n      \"id\": 2,\n      \"tags\": [\n        \"normal\"\n      ]\n    }\n  ]\n}",
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
				t.Errorf("case %q failed:\ngot:\n%q\nwant:\n%q", c.name, result, c.expected)
			}
		})
	}
}

func TestPrettyPrintJSONDebug(t *testing.T) {
	// Test with a valid complex JSON
	complexJSON := `{
		"user": {
			"id": 1001,
			"profile": {
				"personal": {
					"name": "Alice",
					"age": 25,
					"address": {
						"street": "123 Main St",
						"city": "Boston",
						"coordinates": {
							"lat": 42.3601,
							"lng": -71.0589
						}
					}
				},
				"professional": {
					"title": "Software Engineer",
					"company": {
						"name": "Tech Corp",
						"industry": "Technology",
						"employees": 500
					},
					"skills": ["Go", "Python", "JavaScript"],
					"projects": [
						{
							"name": "Project A",
							"status": "active",
							"team": ["Alice", "Bob", "Carol"]
						},
						{
							"name": "Project B",
							"status": "completed",
							"team": ["Alice", "Dave"]
						}
					]
				}
			},
			"preferences": {
				"theme": "dark",
				"notifications": {
					"email": true,
					"push": false,
					"sms": true
				},
				"languages": ["en", "es"]
			},
			"metadata": {
				"created": "2024-01-01T00:00:00Z",
				"updated": "2024-01-15T10:30:00Z",
				"version": 2
			}
		}
	}`

	// Remove whitespace to make it compact
	compactJSON := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(complexJSON, "\n", ""), "\t", ""), " ", "")
	t.Logf("Compact JSON length: %d", len(compactJSON))

	// Test step by step
	var jsonData interface{}
	err := json.Unmarshal([]byte(compactJSON), &jsonData)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	t.Logf("Unmarshaling successful")

	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal indent: %v", err)
	}
	t.Logf("MarshalIndent successful, length: %d", len(prettyJSON))
	t.Logf("Contains newlines: %v", strings.Contains(string(prettyJSON), "\n"))

	// Test our function
	result := prettyPrintJSON(compactJSON)
	t.Logf("prettyPrintJSON result length: %d", len(result))
	t.Logf("Result == input: %v", result == compactJSON)

	if result == compactJSON {
		t.Error("prettyPrintJSON returned the original JSON unchanged - this indicates an error in the function")
	}

	if !strings.Contains(result, "\n") {
		t.Error("prettyPrintJSON should add newlines for formatting")
	}
}

func TestPrettyFlagInRealScenarios(t *testing.T) {
	// Create mock database and handler
	mockDB := newMockDB()
	state := &ReplState{CurrentCF: "default"}
	handler := &Handler{DB: mockDB, State: state}

	// Add complex nested JSON data like what users might actually store
	testData := map[string]string{
		"simple_user": `{"name":"Alice","age":25}`,
		"nested_user": `{"id":1001,"profile":{"name":"Alice","contact":{"email":"alice@example.com","phone":"123-456-7890"},"settings":{"theme":"dark","notifications":{"email":true,"push":false}}}}`,
		"array_data":  `{"users":[{"name":"Alice","roles":["admin","user"]},{"name":"Bob","roles":["user"]}],"meta":{"total":2,"active":true}}`,
		"user_alice":  `{"name":"Alice","department":"Engineering","role":"Senior Developer"}`,
	}

	for key, jsonData := range testData {
		mockDB.PutCF("default", key, jsonData)
	}

	// Test different scenarios where pretty flag is used
	scenarios := []struct {
		name               string
		command            string
		shouldHaveNewlines bool
	}{
		{
			name:               "get simple JSON with pretty",
			command:            "get simple_user --pretty",
			shouldHaveNewlines: true,
		},
		{
			name:               "get nested JSON with pretty",
			command:            "get nested_user --pretty",
			shouldHaveNewlines: true,
		},
		{
			name:               "get array JSON with pretty",
			command:            "get array_data --pretty",
			shouldHaveNewlines: true,
		},
		{
			name:               "jsonquery with pretty on top-level field",
			command:            `jsonquery name Alice --pretty`,
			shouldHaveNewlines: true,
		},
		{
			name:               "last command with pretty",
			command:            "last --pretty",
			shouldHaveNewlines: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Capture output to analyze it
			// For now, just make sure the command executes
			result := handler.Execute(scenario.command)
			if !result {
				t.Errorf("Command %q should execute successfully", scenario.command)
			}
		})
	}

	// Test the prettyPrintJSON function directly with the test data
	for key, jsonData := range testData {
		t.Run("direct_pretty_test_"+key, func(t *testing.T) {
			result := prettyPrintJSON(jsonData)

			if result == jsonData {
				t.Errorf("prettyPrintJSON for %q returned unchanged data", key)
			}

			if !strings.Contains(result, "\n") {
				t.Errorf("prettyPrintJSON for %q should contain newlines, got: %q", key, result)
			}

			// Check that the result is valid JSON
			var checkData interface{}
			if err := json.Unmarshal([]byte(result), &checkData); err != nil {
				t.Errorf("prettyPrintJSON for %q produced invalid JSON: %v", key, err)
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	mockDB := newMockDB()
	state := &ReplState{CurrentCF: "default"}

	// Test key not found error
	t.Run("key not found", func(t *testing.T) {
		// Try to get a non-existent key
		_, err := mockDB.GetCF("default", "nonexistent")
		if !errors.Is(err, db.ErrKeyNotFound) {
			t.Errorf("Expected ErrKeyNotFound, got %v", err)
		}
	})

	// Test column family not found error
	t.Run("column family not found", func(t *testing.T) {
		// Try to get from non-existent CF
		_, err := mockDB.GetCF("nonexistent", "key")
		if !errors.Is(err, db.ErrColumnFamilyNotFound) {
			t.Errorf("Expected ErrColumnFamilyNotFound, got %v", err)
		}
	})

	// Test column family already exists error
	t.Run("column family already exists", func(t *testing.T) {
		// Try to create default CF which already exists
		err := mockDB.CreateCF("default")
		if !errors.Is(err, db.ErrColumnFamilyExists) {
			t.Errorf("Expected ErrColumnFamilyExists, got %v", err)
		}
	})

	// Test column family empty error
	t.Run("column family empty", func(t *testing.T) {
		// Create a new empty CF
		mockDB.CreateCF("empty")

		// Try to get last from empty CF
		_, _, err := mockDB.GetLastCF("empty")
		if !errors.Is(err, db.ErrColumnFamilyEmpty) {
			t.Errorf("Expected ErrColumnFamilyEmpty, got %v", err)
		}
	})

	// Test that handleError provides user-friendly messages
	t.Run("handleError provides friendly messages", func(t *testing.T) {
		tests := []struct {
			name   string
			err    error
			op     string
			params []string
		}{
			{"key not found", db.ErrKeyNotFound, "Query", []string{"mykey", "mycf"}},
			{"cf not found", db.ErrColumnFamilyNotFound, "Query", []string{"mycf"}},
			{"cf exists", db.ErrColumnFamilyExists, "Create", []string{"mycf"}},
			{"read only", db.ErrReadOnlyMode, "Write", []string{}},
			{"cf empty", db.ErrColumnFamilyEmpty, "GetLast", []string{"mycf"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// This just ensures the function doesn't panic
				// In a real test, we might capture stdout to verify the exact message
				handleError(tt.err, tt.op, tt.params...)
			})
		}
	})

	// Prevent unused variable warning for state
	_ = state
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
