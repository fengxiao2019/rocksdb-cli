package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"rocksdb-cli/internal/db"
	"strings"
	"testing"
)

// mockDB implements db.KeyValueDB interface for testing
type mockDB struct {
	data     map[string]map[string]string // cf -> key -> value
	cfExists map[string]bool
}

func newMockDB() *mockDB {
	return &mockDB{
		data: map[string]map[string]string{
			"default": {
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			"users": {
				"admin:001":   "admin data",
				"guest:001":   "guest data",
				"user:1001":   "user 1001 data",
				"user:1002":   "user 1002 data",
				"user:1003":   "user 1003 data",
				"user:nested": "nested data",
			},
		},
		cfExists: map[string]bool{
			"default": true,
			"users":   true,
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
		// Apply range filters
		if len(start) > 0 && k < startStr {
			continue
		}
		if len(end) > 0 && k >= endStr {
			if opts.Reverse {
				continue
			} else {
				break
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
		return db.ErrColumnFamilyNotFound
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
		return "", "", errors.New("column family is empty")
	}

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
	return nil
}

func (m *mockDB) Close() {}

func (m *mockDB) IsReadOnly() bool {
	return false
}

func (m *mockDB) JSONQueryCF(cf, field, value string) (map[string]string, error) {
	if !m.cfExists[cf] {
		return nil, db.ErrColumnFamilyNotFound
	}
	// For testing purposes, return empty result
	// In a real implementation, this would parse JSON and match fields
	return make(map[string]string), nil
}

// Helper function to capture stdout during test execution
func captureOutput(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	done := make(chan struct{})
	var output string

	go func() {
		defer close(done)
		buf := new(bytes.Buffer)
		io.Copy(buf, r)
		output = buf.String()
	}()

	fn()

	w.Close()
	os.Stdout = old
	<-done

	return output
}

func TestExecuteScan(t *testing.T) {
	tests := []struct {
		name      string
		cf        string
		start     *string
		end       *string
		limit     int
		reverse   bool
		keysOnly  bool
		wantError bool
		wantKeys  []string
	}{
		{
			name:     "scan all entries",
			cf:       "users",
			start:    nil,
			end:      nil,
			limit:    0,
			reverse:  false,
			keysOnly: false,
			wantKeys: []string{"admin:001", "guest:001", "user:1001", "user:1002", "user:1003", "user:nested"},
		},
		{
			name:     "scan with start key",
			cf:       "users",
			start:    stringPtr("user:1001"),
			end:      nil,
			limit:    0,
			reverse:  false,
			keysOnly: false,
			wantKeys: []string{"user:1001", "user:1002", "user:1003", "user:nested"},
		},
		{
			name:     "scan with range",
			cf:       "users",
			start:    stringPtr("user:1001"),
			end:      stringPtr("user:1003"),
			limit:    0,
			reverse:  false,
			keysOnly: false,
			wantKeys: []string{"user:1001", "user:1002"},
		},
		{
			name:     "scan with limit",
			cf:       "users",
			start:    nil,
			end:      nil,
			limit:    3,
			reverse:  false,
			keysOnly: false,
			wantKeys: []string{"admin:001", "guest:001", "user:1001"},
		},
		{
			name:     "reverse scan",
			cf:       "users",
			start:    nil,
			end:      nil,
			limit:    3,
			reverse:  true,
			keysOnly: false,
			wantKeys: []string{"user:nested", "user:1003", "user:1002"},
		},
		{
			name:     "keys only scan",
			cf:       "users",
			start:    nil,
			end:      nil,
			limit:    2,
			reverse:  false,
			keysOnly: true,
			wantKeys: []string{"admin:001", "guest:001"},
		},
		{
			name:     "scan with wildcard start",
			cf:       "users",
			start:    stringPtr("*"),
			end:      nil,
			limit:    2,
			reverse:  false,
			keysOnly: false,
			wantKeys: []string{"admin:001", "guest:001"},
		},
		{
			name:     "scan with wildcard end",
			cf:       "users",
			start:    stringPtr("user:1001"),
			end:      stringPtr("*"),
			limit:    0,
			reverse:  false,
			keysOnly: false,
			wantKeys: []string{"user:1001", "user:1002", "user:1003", "user:nested"},
		},
		{
			name:      "scan non-existent cf",
			cf:        "nonexistent",
			start:     nil,
			end:       nil,
			limit:     0,
			reverse:   false,
			keysOnly:  false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := newMockDB()

			output := captureOutput(func() {
				err := executeScan(mockDB, tt.cf, tt.start, tt.end, tt.limit, tt.reverse, tt.keysOnly)
				if (err != nil) != tt.wantError {
					t.Errorf("executeScan() error = %v, wantError %v", err, tt.wantError)
					return
				}
			})

			if tt.wantError {
				return // Skip output validation for error cases
			}

			lines := strings.Split(strings.TrimSpace(output), "\n")
			if len(lines) == 1 && lines[0] == "" {
				lines = []string{} // Handle empty output
			}

			if len(lines) != len(tt.wantKeys) {
				t.Errorf("Expected %d lines, got %d", len(tt.wantKeys), len(lines))
				t.Errorf("Output: %q", output)
				return
			}

			for i, line := range lines {
				expectedKey := tt.wantKeys[i]
				if tt.keysOnly {
					if line != expectedKey {
						t.Errorf("Line %d: expected key %q, got %q", i, expectedKey, line)
					}
				} else {
					if !strings.HasPrefix(line, expectedKey+": ") {
						t.Errorf("Line %d: expected line to start with %q, got %q", i, expectedKey+": ", line)
					}
				}
			}
		})
	}
}

func TestExecuteScanEmptyColumnFamily(t *testing.T) {
	mockDB := newMockDB()
	// Create an empty column family
	mockDB.cfExists["empty"] = true
	mockDB.data["empty"] = make(map[string]string)

	output := captureOutput(func() {
		err := executeScan(mockDB, "empty", nil, nil, 0, false, false)
		if err != nil {
			t.Errorf("executeScan() on empty CF should not error, got: %v", err)
		}
	})

	if strings.TrimSpace(output) != "" {
		t.Errorf("Expected empty output for empty CF, got: %q", output)
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
