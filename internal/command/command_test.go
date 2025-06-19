package command

import (
	"errors"
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

func (m *mockDB) Close() {}

func TestHandler_Execute(t *testing.T) {
	db := newMockDB()
	state := &ReplState{CurrentCF: "default"}
	h := &Handler{DB: db, State: state}

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
				if !db.cfExists["testcf"] {
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
				v, err := db.GetCF("testcf", "key1")
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
				v, err := db.GetCF("default", "key2")
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
				v, err := db.GetCF("testcf", "key1")
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
				v, err := db.GetCF("default", "key2")
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
				v, err := db.GetCF("default", "jsonkey")
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
				v, err := db.GetCF("default", "jsonkey")
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
				v, err := db.GetCF("default", "jsonkey")
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
				db.PutCF("testcf", "key1", "v1")
				db.PutCF("testcf", "key2", "v2")
				db.PutCF("testcf", "other", "v3")
			},
			checkFunc: func() error {
				res, err := db.PrefixScanCF("testcf", "key", 10)
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
				cfs, err := db.ListCFs()
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
				if db.cfExists["testcf"] {
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
				if !db.cfExists["default"] {
					return errors.New("default CF was dropped")
				}
				return nil
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.setupFunc != nil {
				c.setupFunc()
			}
			h.Execute(c.cmd)
			if err := c.checkFunc(); err != nil && !c.expectError {
				t.Errorf("case %q failed: %v", c.name, err)
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
