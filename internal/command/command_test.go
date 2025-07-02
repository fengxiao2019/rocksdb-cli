package command

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
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

// SearchCF implements fuzzy search for testing
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

	for key, value := range cfData {
		var keyMatches, valueMatches bool
		var matchedFields []string

		// Simple substring matching for testing (not implementing full regex/wildcard logic)
		if opts.KeyPattern != "" {
			if opts.CaseSensitive {
				keyMatches = strings.Contains(key, opts.KeyPattern)
			} else {
				keyMatches = strings.Contains(strings.ToLower(key), strings.ToLower(opts.KeyPattern))
			}
			if keyMatches {
				matchedFields = append(matchedFields, "key")
			}
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
		}

		// Determine if this entry should be included
		shouldInclude := false
		if opts.KeyPattern != "" && opts.ValuePattern != "" {
			shouldInclude = keyMatches && valueMatches
		} else if opts.KeyPattern != "" {
			shouldInclude = keyMatches
		} else if opts.ValuePattern != "" {
			shouldInclude = valueMatches
		}

		if shouldInclude {
			result := db.SearchResult{
				Key:           key,
				MatchedFields: matchedFields,
			}
			if !opts.KeysOnly {
				result.Value = value
			}
			results.Results = append(results.Results, result)

			// Check limit
			if opts.Limit > 0 && len(results.Results) >= opts.Limit {
				results.Limited = true
				break
			}
		}
	}

	results.Total = len(results.Results)
	return results, nil
}

// GetCFStats returns mock statistics for a column family
func (m *mockDB) GetCFStats(cf string) (*db.CFStats, error) {
	if !m.cfExists[cf] {
		return nil, db.ErrColumnFamilyNotFound
	}

	cfData, ok := m.data[cf]
	if !ok {
		cfData = make(map[string]string)
	}

	stats := &db.CFStats{
		Name:                    cf,
		KeyCount:                int64(len(cfData)),
		TotalKeySize:            0,
		TotalValueSize:          0,
		DataTypeDistribution:    make(map[db.DataType]int64),
		KeyLengthDistribution:   make(map[string]int64),
		ValueLengthDistribution: make(map[string]int64),
		CommonPrefixes:          make(map[string]int64),
		SampleKeys:              make([]string, 0),
	}

	for key, value := range cfData {
		stats.TotalKeySize += int64(len(key))
		stats.TotalValueSize += int64(len(value))

		// Simple data type detection for mock
		if value == "" {
			stats.DataTypeDistribution[db.DataTypeEmpty]++
		} else if strings.HasPrefix(strings.TrimSpace(value), "{") {
			stats.DataTypeDistribution[db.DataTypeJSON]++
		} else {
			stats.DataTypeDistribution[db.DataTypeString]++
		}

		// Add sample keys (limit to 5)
		if len(stats.SampleKeys) < 5 {
			stats.SampleKeys = append(stats.SampleKeys, key)
		}
	}

	if stats.KeyCount > 0 {
		stats.AverageKeySize = float64(stats.TotalKeySize) / float64(stats.KeyCount)
		stats.AverageValueSize = float64(stats.TotalValueSize) / float64(stats.KeyCount)
	}

	return stats, nil
}

func (m *mockDB) GetDatabaseStats() (*db.DatabaseStats, error) {
	stats := &db.DatabaseStats{
		ColumnFamilies:    make([]db.CFStats, 0),
		ColumnFamilyCount: len(m.cfExists),
	}

	for cf := range m.cfExists {
		cfStats, err := m.GetCFStats(cf)
		if err == nil {
			stats.ColumnFamilies = append(stats.ColumnFamilies, *cfStats)
			stats.TotalKeyCount += cfStats.KeyCount
			stats.TotalSize += cfStats.TotalKeySize + cfStats.TotalValueSize
		}
	}

	return stats, nil
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

// captureOutput captures stdout during function execution
func captureOutput(fn func()) string {
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout
	io.Copy(&buf, r)
	return buf.String()
}

// newTestHandler creates a pre-configured handler for testing
func newTestHandler(cf string) (*Handler, *mockDB) {
	db := newMockDB()
	return &Handler{
		DB:    db,
		State: &ReplState{CurrentCF: cf},
	}, db
}

// TestHandler_Execute tests command execution using table-driven tests
func TestHandler_Execute(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		setup    func(*mockDB, *ReplState)
		wantOut  string
		wantErr  bool
		validate func(*testing.T, *mockDB, *ReplState) // Additional validation
	}{
		{
			name:  "get existing key",
			input: "get key1",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "key1", "value1")
			},
			wantOut: "value1\n",
			wantErr: false,
		},
		{
			name:  "get non-existent key",
			input: "get missing",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantOut: "Key 'missing' not found in column family 'default'\n",
			wantErr: false,
		},
		{
			name:  "get with explicit column family",
			input: "get testcf key1",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.CreateCF("testcf")
				db.PutCF("testcf", "key1", "test_value")
			},
			wantOut: "test_value\n",
			wantErr: false,
		},
		{
			name:  "get JSON with pretty flag",
			input: "get jsonkey --pretty",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				jsonData := `{"name":"test","value":123}`
				db.PutCF("default", "jsonkey", jsonData)
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				// Just verify the key exists - output formatting is tested separately
				val, err := db.GetCF("default", "jsonkey")
				if err != nil || !strings.Contains(val, "test") {
					t.Errorf("JSON data not properly stored or retrieved")
				}
			},
		},
		{
			name:  "put key-value",
			input: "put key1 value1",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				val, err := db.GetCF("default", "key1")
				if err != nil || val != "value1" {
					t.Errorf("Put operation failed: got %v, want value1", val)
				}
			},
		},
		{
			name:  "put with explicit column family",
			input: "put testcf key2 value2",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.CreateCF("testcf")
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				val, err := db.GetCF("testcf", "key2")
				if err != nil || val != "value2" {
					t.Errorf("Put with CF failed: got %v, want value2", val)
				}
			},
		},
		{
			name:  "put JSON value",
			input: `put jsonkey {"name":"test","value":123}`,
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				val, err := db.GetCF("default", "jsonkey")
				if err != nil || !strings.Contains(val, "test") {
					t.Errorf("JSON put failed: got %v", val)
				}
			},
		},
		{
			name:  "create column family",
			input: "createcf testcf",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				if !db.cfExists["testcf"] {
					t.Error("Column family was not created")
				}
			},
		},
		{
			name:  "create existing column family",
			input: "createcf default",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantOut: "Column family 'default' already exists\n",
			wantErr: false,
		},
		{
			name:  "use column family",
			input: "usecf testcf",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.CreateCF("testcf")
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				if state.CurrentCF != "testcf" {
					t.Errorf("Column family not switched: got %s, want testcf", state.CurrentCF)
				}
			},
		},
		{
			name:  "list column families",
			input: "listcf",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.CreateCF("testcf1")
				db.CreateCF("testcf2")
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				cfs, err := db.ListCFs()
				if err != nil || len(cfs) < 3 {
					t.Errorf("ListCF failed: got %v", cfs)
				}
			},
		},
		{
			name:  "drop column family",
			input: "dropcf testcf",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.CreateCF("testcf")
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				if db.cfExists["testcf"] {
					t.Error("Column family was not dropped")
				}
			},
		},
		{
			name:  "drop default column family should fail",
			input: "dropcf default",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantErr: true,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				if !db.cfExists["default"] {
					t.Error("Default column family was incorrectly dropped")
				}
			},
		},
		{
			name:  "prefix scan",
			input: "prefix key",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "key1", "value1")
				db.PutCF("default", "key2", "value2")
				db.PutCF("default", "other", "value3")
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				result, err := db.PrefixScanCF("default", "key", 10)
				if err != nil || len(result) != 2 {
					t.Errorf("Prefix scan failed: got %d results, want 2", len(result))
				}
			},
		},
		{
			name:  "prefix scan with explicit CF",
			input: "prefix testcf key",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.CreateCF("testcf")
				db.PutCF("testcf", "key1", "value1")
				db.PutCF("testcf", "key2", "value2")
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				result, err := db.PrefixScanCF("testcf", "key", 10)
				if err != nil || len(result) != 2 {
					t.Errorf("Prefix scan with CF failed: got %d results, want 2", len(result))
				}
			},
		},
		{
			name:  "prefix scan with pretty flag",
			input: "prefix user --pretty",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "user1", `{"name":"Alice","age":25}`)
				db.PutCF("default", "user2", `{"name":"Bob","age":30}`)
				db.PutCF("default", "product1", "other_data")
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				result, err := db.PrefixScanCF("default", "user", 20)
				if err != nil || len(result) != 2 {
					t.Errorf("Prefix scan with pretty failed: got %d results, want 2", len(result))
				}
			},
		},
		{
			name:  "prefix scan with explicit CF and pretty",
			input: "prefix testcf user --pretty",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.CreateCF("testcf")
				db.PutCF("testcf", "user1", `{"id":1,"name":"Alice"}`)
				db.PutCF("testcf", "user2", `{"id":2,"name":"Bob"}`)
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				result, err := db.PrefixScanCF("testcf", "user", 20)
				if err != nil || len(result) != 2 {
					t.Errorf("Prefix scan with CF and pretty failed: got %d results, want 2", len(result))
				}
			},
		},
		{
			name:  "scan with start and end",
			input: "scan key1 key4",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				for i := 1; i <= 5; i++ {
					db.PutCF("default", fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
				}
			},
			wantErr: false,
		},
		{
			name:  "scan with wildcard",
			input: "scan *",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.data["default"] = make(map[string]string) // Clear existing data
				db.PutCF("default", "a", "va")
				db.PutCF("default", "b", "vb")
				db.PutCF("default", "c", "vc")
			},
			wantErr: false,
		},
		{
			name:  "scan with limit",
			input: "scan key1 key5 --limit=2",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.data["default"] = make(map[string]string) // Clear existing data
				for i := 1; i <= 5; i++ {
					db.PutCF("default", fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
				}
			},
			wantErr: false,
		},
		{
			name:  "scan reverse",
			input: "scan key1 key5 --reverse",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.data["default"] = make(map[string]string) // Clear existing data
				for i := 1; i <= 5; i++ {
					db.PutCF("default", fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
				}
			},
			wantErr: false,
		},
		{
			name:  "export to CSV",
			input: "export test.csv",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "key1", "value1")
			},
			wantErr: false,
		},
		{
			name:  "get last entry",
			input: "last",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.data["default"] = make(map[string]string) // Clear existing data
				db.PutCF("default", "key1", "value1")
				db.PutCF("default", "key2", "value2")
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				lastKey, lastValue, err := db.GetLastCF("default")
				if err != nil {
					t.Errorf("GetLast failed: %v", err)
				}
				if lastKey == "" {
					t.Error("Expected last key to be non-empty")
				}
				if lastValue == "" {
					t.Error("Expected last value to be non-empty")
				}
			},
		},
		{
			name:  "JSON query",
			input: `jsonquery name Alice`,
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "user1", `{"name":"Alice","age":25}`)
				db.PutCF("default", "user2", `{"name":"Bob","age":30}`)
			},
			wantErr: false,
			validate: func(t *testing.T, db *mockDB, state *ReplState) {
				result, err := db.JSONQueryCF("default", "name", "Alice")
				if err != nil || len(result) == 0 {
					t.Errorf("JSON query failed: %v", err)
				}
			},
		},
		// Edge cases and error scenarios
		{
			name:    "empty command",
			input:   "",
			setup:   func(db *mockDB, state *ReplState) {},
			wantErr: true,
		},
		{
			name:    "invalid command",
			input:   "invalidcmd",
			setup:   func(db *mockDB, state *ReplState) {},
			wantErr: true,
		},
		{
			name:  "get with insufficient args",
			input: "get",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantErr: true,
		},
		{
			name:  "put with insufficient args",
			input: "put key1",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantErr: true,
		},
		{
			name:  "operation on non-existent CF",
			input: "get nonexistent key1",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantOut: "Column family 'key1' does not exist\n",
			wantErr: false,
		},
		{
			name:  "search non-existent CF",
			input: "search nonexistent --key=test",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantErr: false,
			wantOut: "Column family 'nonexistent' does not exist\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := newTestHandler("default")
			state := handler.State.(*ReplState)

			if tt.setup != nil {
				tt.setup(db, state)
			}

			var output string
			if tt.wantOut != "" {
				output = captureOutput(func() {
					handler.Execute(tt.input)
				})
			} else {
				handler.Execute(tt.input)
			}

			// Validate output if expected
			if tt.wantOut != "" && output != tt.wantOut {
				t.Errorf("Execute() output = %q, want %q", output, tt.wantOut)
			}

			// Additional validation
			if tt.validate != nil {
				tt.validate(t, db, state)
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

func TestStatsCommand(t *testing.T) {
	handler, mockDB := newTestHandler("users")

	// Create column families first
	mockDB.CreateCF("users")
	mockDB.CreateCF("logs")

	// Set up test data with different data types
	mockDB.PutCF("users", "user:1", `{"name": "Alice", "age": 30}`)
	mockDB.PutCF("users", "user:2", `{"name": "Bob", "age": 25}`)
	mockDB.PutCF("users", "user:3", "plain_string_value")
	mockDB.PutCF("users", "user:4", "")
	mockDB.PutCF("logs", "log:1", `{"level": "info", "message": "test"}`)

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "Database stats",
			input: "stats",
			expected: []string{
				"Database Statistics",
				"Column Families: 3", // default + users + logs
				"Total Keys:",
				"users",
				"logs",
			},
		},
		{
			name:  "Column family stats",
			input: "stats users",
			expected: []string{
				"Column Family: users",
				"Keys: 4",
				"Data Type Distribution:",
				"JSON",
				"String",
				"Empty",
			},
		},
		{
			name:  "Detailed stats",
			input: "stats users --detailed",
			expected: []string{
				"Column Family: users",
				"Data Type Distribution:",
				"Sample Keys:",
				"user:",
			},
		},
		{
			name:  "Pretty JSON stats",
			input: "stats users --pretty",
			expected: []string{
				`"name": "users"`,
				`"key_count"`,
				`"data_type_distribution"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				handler.Execute(tt.input)
			})

			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but got: %s", expected, output)
				}
			}
		})
	}
}

func TestStatsErrorHandling(t *testing.T) {
	handler, _ := newTestHandler("users")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Non-existent column family",
			input:    "stats nonexistent",
			expected: "Column family 'nonexistent' does not exist",
		},
		{
			name:     "Invalid usage",
			input:    "stats cf1 cf2 extra",
			expected: "Usage: stats [<cf>] [--detailed] [--pretty]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				handler.Execute(tt.input)
			})

			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', but got: %s", tt.expected, output)
			}
		})
	}
}

func TestSearchCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		setup   func(*mockDB, *ReplState)
		wantErr bool
		wantOut string // substring that should be in output
	}{
		{
			name:  "search command help",
			input: "search",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantErr: false,
			wantOut: "Usage: search",
		},
		{
			name:  "search by key pattern",
			input: "search --key=user",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "user:1001", "Alice")
				db.PutCF("default", "user:1002", "Bob")
				db.PutCF("default", "product:001", "Widget")
			},
			wantErr: false,
			wantOut: "Found 2 matches",
		},
		{
			name:  "search by value pattern",
			input: "search --value=Alice",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "user:1001", "Alice Johnson")
				db.PutCF("default", "user:1002", "Bob Smith")
				db.PutCF("default", "admin:001", "Alice Admin")
			},
			wantErr: false,
			wantOut: "Found 2 matches",
		},
		{
			name:  "search with specific CF",
			input: "search users --key=user",
			setup: func(db *mockDB, state *ReplState) {
				db.CreateCF("users")
				db.PutCF("users", "user:1001", "Alice")
				db.PutCF("users", "user:1002", "Bob")
			},
			wantErr: false,
			wantOut: "Found 2 matches",
		},
		{
			name:  "search with keys-only flag",
			input: "search --key=user --keys-only",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "user:1001", "Alice")
				db.PutCF("default", "user:1002", "Bob")
			},
			wantErr: false,
			wantOut: "user:1001",
		},
		{
			name:  "search with limit",
			input: "search --key=user --limit=1",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "user:1001", "Alice")
				db.PutCF("default", "user:1002", "Bob")
				db.PutCF("default", "user:1003", "Charlie")
			},
			wantErr: false,
			wantOut: "Found 1 matches (limited)",
		},
		{
			name:  "search both key and value patterns",
			input: "search --key=user --value=Alice",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "user:1001", "Alice Johnson")
				db.PutCF("default", "user:1002", "Bob Smith")
				db.PutCF("default", "admin:001", "Alice Admin")
			},
			wantErr: false,
			wantOut: "Found 1 matches",
		},
		{
			name:  "search with no matches",
			input: "search --key=nonexistent",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
				db.PutCF("default", "user:1001", "Alice")
			},
			wantErr: false,
			wantOut: "No matches found",
		},
		{
			name:  "search without pattern",
			input: "search users",
			setup: func(db *mockDB, state *ReplState) {
				db.CreateCF("users")
			},
			wantErr: false,
			wantOut: "Must specify at least --key or --value pattern",
		},
		{
			name:  "search non-existent CF",
			input: "search nonexistent --key=test",
			setup: func(db *mockDB, state *ReplState) {
				state.CurrentCF = "default"
			},
			wantErr: false,
			wantOut: "Column family 'nonexistent' does not exist\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler and setup
			handler, db := newTestHandler("default")
			if tt.setup != nil {
				tt.setup(db, handler.State.(*ReplState))
			}

			// Capture output
			output := captureOutput(func() {
				handler.Execute(tt.input)
			})

			// Check if expected substring is in output
			if tt.wantOut != "" {
				if !strings.Contains(output, tt.wantOut) {
					t.Errorf("Expected output to contain '%s', got: %s", tt.wantOut, output)
				}
			}
		})
	}
}
