package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/util"
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

func (m *mockDB) ExportToCSV(cf, filePath, sep string) error {
	if !m.cfExists[cf] {
		return db.ErrColumnFamilyNotFound
	}
	return nil
}

func (m *mockDB) ExportSearchResultsToCSV(cf, filePath, sep string, opts db.SearchOptions) error {
	if !m.cfExists[cf] {
		return db.ErrColumnFamilyNotFound
	}
	// First perform search to validate inputs
	_, err := m.SearchCF(cf, opts)
	if err != nil {
		return err
	}
	// For testing, simulate the export without actually creating a file
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

func (m *mockDB) GetCFStats(cf string) (*db.CFStats, error) {
	if !m.cfExists[cf] {
		return nil, db.ErrColumnFamilyNotFound
	}

	stats := &db.CFStats{
		Name:                    cf,
		DataTypeDistribution:    make(map[db.DataType]int64),
		KeyLengthDistribution:   make(map[string]int64),
		ValueLengthDistribution: make(map[string]int64),
		CommonPrefixes:          make(map[string]int64),
		SampleKeys:              make([]string, 0),
	}

	cfData := m.data[cf]
	stats.KeyCount = int64(len(cfData))

	for key, value := range cfData {
		keyLen := int64(len(key))
		valueLen := int64(len(value))
		stats.TotalKeySize += keyLen
		stats.TotalValueSize += valueLen

		// Simple data type detection for mock
		if len(value) == 0 {
			stats.DataTypeDistribution[db.DataTypeEmpty]++
		} else if value[0] == '{' || value[0] == '[' {
			stats.DataTypeDistribution[db.DataTypeJSON]++
		} else {
			stats.DataTypeDistribution[db.DataTypeString]++
		}

		stats.SampleKeys = append(stats.SampleKeys, key)
	}

	if stats.KeyCount > 0 {
		stats.AverageKeySize = float64(stats.TotalKeySize) / float64(stats.KeyCount)
		stats.AverageValueSize = float64(stats.TotalValueSize) / float64(stats.KeyCount)
	}

	return stats, nil
}

func (m *mockDB) GetDatabaseStats() (*db.DatabaseStats, error) {
	cfs := []db.CFStats{}
	for cf := range m.cfExists {
		cfStats, err := m.GetCFStats(cf)
		if err == nil {
			cfs = append(cfs, *cfStats)
		}
	}

	stats := &db.DatabaseStats{
		ColumnFamilies:    cfs,
		ColumnFamilyCount: len(m.cfExists),
	}

	return stats, nil
}

func (m *mockDB) SearchCF(cf string, opts db.SearchOptions) (*db.SearchResults, error) {
	if !m.cfExists[cf] {
		return nil, db.ErrColumnFamilyNotFound
	}

	results := &db.SearchResults{
		Results:   make([]db.SearchResult, 0),
		Limited:   false,
		QueryTime: "0ms", // Mock timing
	}

	cfData, ok := m.data[cf]
	if !ok {
		return results, nil
	}

	// Simple substring matching for testing (not implementing full regex/wildcard logic)
	for key, value := range cfData {
		var keyMatches, valueMatches bool
		var matchedFields []string

		if opts.KeyPattern != "" {
			if opts.CaseSensitive {
				keyMatches = strings.Contains(key, opts.KeyPattern)
			} else {
				keyMatches = strings.Contains(strings.ToLower(key), strings.ToLower(opts.KeyPattern))
			}
			if keyMatches {
				matchedFields = append(matchedFields, "key")
			}
		} else {
			keyMatches = true
		}

		if opts.ValuePattern != "" {
			if opts.CaseSensitive {
				valueMatches = strings.Contains(value, opts.ValuePattern)
			} else {
				valueMatches = strings.Contains(strings.ToLower(value), strings.ToLower(opts.ValuePattern))
			}
			if valueMatches {
				matchedFields = append(matchedFields, "value")
			}
		} else {
			valueMatches = true
		}

		// Both patterns must match if both are specified
		if opts.KeyPattern != "" && opts.ValuePattern != "" {
			if keyMatches && valueMatches {
				result := db.SearchResult{
					Key:           key,
					Value:         value,
					MatchedFields: matchedFields,
				}
				results.Results = append(results.Results, result)
			}
		} else if keyMatches || valueMatches {
			result := db.SearchResult{
				Key:           key,
				Value:         value,
				MatchedFields: matchedFields,
			}
			results.Results = append(results.Results, result)
		}

		// Apply limit
		if opts.Limit > 0 && len(results.Results) >= opts.Limit {
			results.Limited = true
			break
		}
	}

	return results, nil
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

	expectedOutput := "No entries found in column family 'empty'\n"
	if strings.TrimSpace(output) != strings.TrimSpace(expectedOutput) {
		t.Errorf("Expected output %q for empty CF, got: %q", expectedOutput, output)
	}
}

func TestExecutePrefix(t *testing.T) {
	tests := []struct {
		name      string
		cf        string
		prefix    string
		pretty    bool
		wantError bool
		wantKeys  []string
	}{
		{
			name:     "basic prefix scan",
			cf:       "users",
			prefix:   "user:",
			pretty:   false,
			wantKeys: []string{"user:1001", "user:1002", "user:1003", "user:nested"},
		},
		{
			name:     "prefix scan with pretty",
			cf:       "users",
			prefix:   "user:",
			pretty:   true,
			wantKeys: []string{"user:1001", "user:1002", "user:1003", "user:nested"},
		},
		{
			name:     "prefix scan with admin prefix",
			cf:       "users",
			prefix:   "admin:",
			pretty:   false,
			wantKeys: []string{"admin:001"},
		},
		{
			name:     "prefix scan no matches",
			cf:       "users",
			prefix:   "nonexistent:",
			pretty:   false,
			wantKeys: []string{},
		},
		{
			name:      "prefix scan non-existent cf",
			cf:        "nonexistent",
			prefix:    "test:",
			pretty:    false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := newMockDB()

			output := captureOutput(func() {
				err := executePrefix(mockDB, tt.cf, tt.prefix, tt.pretty)
				if (err != nil) != tt.wantError {
					t.Errorf("executePrefix() error = %v, wantError %v", err, tt.wantError)
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
				if !strings.HasPrefix(line, expectedKey+": ") {
					t.Errorf("Line %d: expected line to start with %q, got %q", i, expectedKey+": ", line)
				}
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// Smart key conversion methods for the new interface
func (m *mockDB) SmartGetCF(cf, key string) (string, error) {
	// For testing, just delegate to regular GetCF
	return m.GetCF(cf, key)
}

func (m *mockDB) SmartPrefixScanCF(cf, prefix string, limit int) (map[string]string, error) {
	// For testing, just delegate to regular PrefixScanCF
	return m.PrefixScanCF(cf, prefix, limit)
}

func (m *mockDB) SmartScanCF(cf string, start, end string, opts db.ScanOptions) (map[string]string, error) {
	// For testing, convert strings to bytes and delegate to regular ScanCF
	var startBytes, endBytes []byte
	if start != "" && start != "*" {
		startBytes = []byte(start)
	}
	if end != "" && end != "*" {
		endBytes = []byte(end)
	}
	return m.ScanCF(cf, startBytes, endBytes, opts)
}

func (m *mockDB) GetKeyFormatInfo(cf string) (util.KeyFormat, string) {
	// For testing, just return string format
	return util.KeyFormatString, "Printable string keys"
}

func (m *mockDB) ScanCFPage(cf string, start, end []byte, opts db.ScanOptions) (db.ScanPageResult, error) {
	result, err := m.ScanCF(cf, start, end, opts)
	if err != nil {
		return db.ScanPageResult{}, err
	}
	return db.ScanPageResult{Results: result, NextCursor: "", HasMore: false}, nil
}

func (m *mockDB) SmartScanCFPage(cf string, start, end string, opts db.ScanOptions) (db.ScanPageResult, error) {
	result, err := m.SmartScanCF(cf, start, end, opts)
	if err != nil {
		return db.ScanPageResult{}, err
	}
	return db.ScanPageResult{Results: result, NextCursor: "", HasMore: false}, nil
}
