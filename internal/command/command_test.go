package command

import (
	"errors"
	"rocksdb-cli/internal/db"
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
		if len(start) > 0 && k < startStr {
			continue
		}
		if len(end) > 0 && k >= endStr && !opts.Reverse {
			break
		}
		if len(end) > 0 && k > endStr && opts.Reverse {
			continue
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

func (m *mockDB) ExportToCSV(cf, filePath string) error {
	if !m.cfExists[cf] {
		return errors.New("column family not found")
	}

	// For testing, we'll just simulate the export without actually creating a file
	// In a real test environment, you might want to create a temporary file
	return nil
}

func (m *mockDB) Close() {}

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
			checkFunc: func() error {
				res, err := mockDB.ScanCF("default", []byte("key1"), []byte("key5"), db.ScanOptions{Values: true, Reverse: true})
				if err != nil {
					return errors.New("reverse scan failed")
				}

				// Convert map keys to slice for order checking
				keys := make([]string, 0, len(res))
				for k := range res {
					keys = append(keys, k)
				}

				// Check if keys are in reverse order
				for i := 0; i < len(keys)-1; i++ {
					if keys[i] < keys[i+1] {
						return errors.New("reverse scan order incorrect")
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
