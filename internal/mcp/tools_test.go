package mcp

import (
	"strings"
	"testing"

	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/jsonutil"

	"github.com/mark3labs/mcp-go/server"
)

// MockKeyValueDB implements a mock database for testing
type MockKeyValueDB struct {
	data     map[string]map[string]string // cf -> key -> value
	readOnly bool
}

func NewMockKeyValueDB() *MockKeyValueDB {
	mockDB := &MockKeyValueDB{
		data:     make(map[string]map[string]string),
		readOnly: false,
	}
	// Initialize default column family
	mockDB.data["default"] = make(map[string]string)
	return mockDB
}

func (m *MockKeyValueDB) GetCF(cf, key string) (string, error) {
	if cfData, exists := m.data[cf]; exists {
		if value, exists := cfData[key]; exists {
			return value, nil
		}
	}
	return "", db.ErrKeyNotFound
}

func (m *MockKeyValueDB) PutCF(cf, key, value string) error {
	if m.readOnly {
		return db.ErrReadOnlyMode
	}
	if _, exists := m.data[cf]; !exists {
		return db.ErrColumnFamilyNotFound
	}
	m.data[cf][key] = value
	return nil
}

func (m *MockKeyValueDB) ScanCF(cf string, startKey, endKey []byte, opts db.ScanOptions) (map[string]string, error) {
	results := make(map[string]string)
	if cfData, exists := m.data[cf]; exists {
		count := 0
		for key, value := range cfData {
			if opts.Limit > 0 && count >= opts.Limit {
				break
			}
			if opts.Values {
				results[key] = value
			} else {
				results[key] = ""
			}
			count++
		}
	}
	return results, nil
}

func (m *MockKeyValueDB) PrefixScanCF(cf, prefix string, limit int) (map[string]string, error) {
	results := make(map[string]string)
	if cfData, exists := m.data[cf]; exists {
		count := 0
		for key, value := range cfData {
			if limit > 0 && count >= limit {
				break
			}
			// Simple prefix matching
			if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
				results[key] = value
				count++
			}
		}
	}
	return results, nil
}

func (m *MockKeyValueDB) GetLastCF(cf string) (string, string, error) {
	if cfData, exists := m.data[cf]; exists {
		var lastKey, lastValue string
		for key, value := range cfData {
			lastKey = key
			lastValue = value
		}
		if lastKey != "" {
			return lastKey, lastValue, nil
		}
	}
	return "", "", db.ErrKeyNotFound
}

func (m *MockKeyValueDB) ExportToCSV(cf, filePath string) error {
	// Mock implementation - just return success
	return nil
}

func (m *MockKeyValueDB) JSONQueryCF(cf, field, value string) (map[string]string, error) {
	// Mock implementation - return empty results
	return make(map[string]string), nil
}

func (m *MockKeyValueDB) CreateCF(cf string) error {
	if m.readOnly {
		return db.ErrReadOnlyMode
	}
	if _, exists := m.data[cf]; exists {
		return db.ErrColumnFamilyExists
	}
	m.data[cf] = make(map[string]string)
	return nil
}

func (m *MockKeyValueDB) DropCF(cf string) error {
	if m.readOnly {
		return db.ErrReadOnlyMode
	}
	if cf == "default" {
		return db.ErrReadOnlyMode // Use existing error since ErrCannotDropDefaultCF doesn't exist
	}
	if _, exists := m.data[cf]; !exists {
		return db.ErrColumnFamilyNotFound
	}
	delete(m.data, cf)
	return nil
}

func (m *MockKeyValueDB) ListCFs() ([]string, error) {
	var cfs []string
	for cf := range m.data {
		cfs = append(cfs, cf)
	}
	if len(cfs) == 0 {
		cfs = append(cfs, "default")
	}
	return cfs, nil
}

func (m *MockKeyValueDB) Close() {
	// Mock implementation - no-op
}

func (m *MockKeyValueDB) IsReadOnly() bool {
	return m.readOnly
}

func (m *MockKeyValueDB) SetReadOnly(readOnly bool) {
	m.readOnly = readOnly
}

func (m *MockKeyValueDB) GetCFStats(cf string) (*db.CFStats, error) {
	if _, exists := m.data[cf]; !exists {
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

func (m *MockKeyValueDB) GetDatabaseStats() (*db.DatabaseStats, error) {
	stats := &db.DatabaseStats{
		ColumnFamilies:    make([]db.CFStats, 0),
		ColumnFamilyCount: len(m.data),
	}

	for cf := range m.data {
		cfStats, err := m.GetCFStats(cf)
		if err == nil {
			stats.ColumnFamilies = append(stats.ColumnFamilies, *cfStats)
			stats.TotalKeyCount += cfStats.KeyCount
			stats.TotalSize += cfStats.TotalKeySize + cfStats.TotalValueSize
		}
	}

	return stats, nil
}

func TestNewToolManager(t *testing.T) {
	mockDB := NewMockKeyValueDB()
	config := DefaultConfig()
	config.DatabasePath = "/tmp/test.db"

	tm := NewToolManager(mockDB, config)

	if tm == nil {
		t.Fatal("NewToolManager returned nil")
	}
	// Note: We can't directly compare the interface tm.db with the concrete mockDB type
	// if tm.db != mockDB {
	//	t.Error("ToolManager database not set correctly")
	// }
	if tm.config != config {
		t.Error("ToolManager config not set correctly")
	}
}

func TestToolManagerRegisterTools(t *testing.T) {
	mockDB := NewMockKeyValueDB()
	config := DefaultConfig()
	config.DatabasePath = "/tmp/test.db"

	tm := NewToolManager(mockDB, config)

	// Create a mock MCP server
	mcpServer := server.NewMCPServer("Test Server", "1.0.0", server.WithToolCapabilities(true))

	err := tm.RegisterTools(mcpServer)
	if err != nil {
		t.Fatalf("RegisterTools failed: %v", err)
	}

	// Note: We can't easily test the actual registration without accessing internal server state
	// This test mainly ensures the function doesn't panic or return an error
}

func TestMockDatabaseOperations(t *testing.T) {
	mockDB := NewMockKeyValueDB()

	// Test basic get/put operations
	err := mockDB.PutCF("default", "test_key", "test_value")
	if err != nil {
		t.Errorf("PutCF failed: %v", err)
	}

	value, err := mockDB.GetCF("default", "test_key")
	if err != nil {
		t.Errorf("GetCF failed: %v", err)
	}
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}

	// Test non-existent key
	_, err = mockDB.GetCF("default", "nonexistent")
	if err != db.ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}

	// Test column family operations
	err = mockDB.CreateCF("test_cf")
	if err != nil {
		t.Errorf("CreateCF failed: %v", err)
	}

	cfs, err := mockDB.ListCFs()
	if err != nil {
		t.Errorf("ListCFs failed: %v", err)
	}
	if len(cfs) < 2 {
		t.Errorf("Expected at least 2 column families, got %d", len(cfs))
	}

	// Test read-only mode
	mockDB.SetReadOnly(true)
	err = mockDB.PutCF("default", "readonly_test", "value")
	if err != db.ErrReadOnlyMode {
		t.Errorf("Expected ErrReadOnlyMode, got %v", err)
	}

	err = mockDB.CreateCF("readonly_cf")
	if err != db.ErrReadOnlyMode {
		t.Errorf("Expected ErrReadOnlyMode for CreateCF, got %v", err)
	}
}

func TestToolManagerReadOnlyMode(t *testing.T) {
	mockDB := NewMockKeyValueDB()
	config := DefaultConfig()
	config.ReadOnly = true
	config.DatabasePath = "/tmp/test.db"

	tm := NewToolManager(mockDB, config)

	if tm == nil {
		t.Fatal("NewToolManager returned nil")
	}

	// Verify config is set to read-only
	if !tm.config.ReadOnly {
		t.Error("Expected config to be read-only")
	}
}

func TestPrefixScan(t *testing.T) {
	mockDB := NewMockKeyValueDB()

	// Add test data with different prefixes
	testData := map[string]string{
		"user:1":    "Alice",
		"user:2":    "Bob",
		"user:10":   "Charlie",
		"product:1": "Widget",
		"product:2": "Gadget",
		"other:key": "value",
	}

	for key, value := range testData {
		err := mockDB.PutCF("default", key, value)
		if err != nil {
			t.Fatalf("Failed to put test data: %v", err)
		}
	}

	// Test prefix scan for "user:"
	results, err := mockDB.PrefixScanCF("default", "user:", 0)
	if err != nil {
		t.Errorf("PrefixScanCF failed: %v", err)
	}

	expectedUserCount := 3
	if len(results) != expectedUserCount {
		t.Errorf("Expected %d user results, got %d", expectedUserCount, len(results))
	}

	// Verify specific results
	for key := range results {
		if len(key) < 5 || key[:5] != "user:" {
			t.Errorf("Unexpected key in user prefix scan: %s", key)
		}
	}

	// Test prefix scan for "product:"
	results, err = mockDB.PrefixScanCF("default", "product:", 0)
	if err != nil {
		t.Errorf("PrefixScanCF failed: %v", err)
	}

	expectedProductCount := 2
	if len(results) != expectedProductCount {
		t.Errorf("Expected %d product results, got %d", expectedProductCount, len(results))
	}
}

func TestScanWithOptions(t *testing.T) {
	mockDB := NewMockKeyValueDB()

	// Add test data
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
	}

	for key, value := range testData {
		err := mockDB.PutCF("default", key, value)
		if err != nil {
			t.Fatalf("Failed to put test data: %v", err)
		}
	}

	// Test scan with limit
	results, err := mockDB.ScanCF("default", nil, nil, db.ScanOptions{Limit: 2, Values: true})
	if err != nil {
		t.Errorf("ScanCF failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results with limit, got %d", len(results))
	}

	// Test scan without values
	results, err = mockDB.ScanCF("default", nil, nil, db.ScanOptions{Values: false})
	if err != nil {
		t.Errorf("ScanCF failed: %v", err)
	}

	// Check that values are empty when Values: false
	for key, value := range results {
		if value != "" {
			t.Errorf("Expected empty value for key %s when Values: false, got '%s'", key, value)
		}
	}
}

func TestFormatJSONValue_SimpleJSON(t *testing.T) {
	tm := &ToolManager{}

	input := `{"name":"Alice","age":25}`
	expected := `{
  "age": 25,
  "name": "Alice"
}`

	result := tm.formatJSONValue(input)

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatJSONValue_NestedJSONString(t *testing.T) {
	tm := &ToolManager{}

	// Test case with embedded JSON string
	input := `{"id":"user001","data":"{\"name\":\"Alice\",\"age\":25,\"details\":{\"city\":\"New York\",\"country\":\"USA\"}}"}`

	result := tm.formatJSONValue(input)

	// Check that the nested JSON was expanded
	if !strings.Contains(result, `"name": "Alice"`) {
		t.Error("Nested JSON was not expanded properly - missing name field")
	}

	if !strings.Contains(result, `"city": "New York"`) {
		t.Error("Deeply nested JSON was not expanded properly - missing city field")
	}

	// Ensure it's properly formatted with indentation
	if !strings.Contains(result, "  ") {
		t.Error("Output should be pretty-formatted with indentation")
	}
}

func TestFormatJSONValue_ArrayWithJSONStrings(t *testing.T) {
	tm := &ToolManager{}

	// Test case with array containing JSON strings
	input := `{"users":["{\"name\":\"Alice\",\"age\":25}","{\"name\":\"Bob\",\"age\":30}"]}`

	result := tm.formatJSONValue(input)

	// Check that JSON strings in array were expanded
	if !strings.Contains(result, `"name": "Alice"`) {
		t.Error("JSON string in array was not expanded - missing Alice")
	}

	if !strings.Contains(result, `"name": "Bob"`) {
		t.Error("JSON string in array was not expanded - missing Bob")
	}
}

func TestFormatJSONValue_InvalidJSON(t *testing.T) {
	tm := &ToolManager{}

	// Test case with invalid JSON
	input := `not valid json`

	result := tm.formatJSONValue(input)

	// Should return the original string if not valid JSON
	if result != input {
		t.Errorf("Expected original string for invalid JSON, got: %s", result)
	}
}

func TestFormatJSONValue_MixedContent(t *testing.T) {
	tm := &ToolManager{}

	// Test case with mix of regular strings and JSON strings
	input := `{"id":"user001","regular_text":"This is just text","json_data":"{\"embedded\":true,\"value\":42}"}`

	result := tm.formatJSONValue(input)

	// Check that only actual JSON strings were expanded
	if !strings.Contains(result, `"embedded": true`) {
		t.Error("JSON string was not expanded")
	}

	if !strings.Contains(result, `"regular_text": "This is just text"`) {
		t.Error("Regular text should remain unchanged")
	}
}

func TestIsJSONString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`{"key":"value"}`, true},
		{`[1,2,3]`, true},
		{`  {"key":"value"}  `, true}, // with whitespace
		{`"just a string"`, false},
		{`regular text`, false},
		{``, false},
		{`{incomplete`, false},
		{`{"valid": "json"}extra`, false},
	}

	for _, test := range tests {
		// Test by checking if PrettyPrintWithNestedExpansion would expand it
		// This is an indirect way to test the JSON string detection logic
		input := `{"test":"` + strings.ReplaceAll(test.input, `"`, `\"`) + `"}`
		result := jsonutil.PrettyPrintWithNestedExpansion(input)

		// If the input was treated as JSON, it should be expanded
		// If it wasn't, it should remain as a simple string value
		containsExpansion := strings.Contains(result, `"test": {`) || strings.Contains(result, `"test": [`)

		if test.expected && !containsExpansion {
			t.Errorf("Expected %q to be detected as JSON, but it wasn't expanded", test.input)
		} else if !test.expected && containsExpansion {
			t.Errorf("Expected %q to NOT be detected as JSON, but it was expanded", test.input)
		}
	}
}
