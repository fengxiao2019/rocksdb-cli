package service

import (
	"testing"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/util"
)

// MockDB is a mock implementation of KeyValueDB for testing
type MockDB struct {
	data       map[string]map[string]string // cf -> key -> value
	readOnly   bool
}

func NewMockDB() *MockDB {
	return &MockDB{
		data: make(map[string]map[string]string),
		readOnly: false,
	}
}

func (m *MockDB) SmartGetCF(cf, key string) (string, error) {
	if cfData, ok := m.data[cf]; ok {
		if val, ok := cfData[key]; ok {
			return val, nil
		}
	}
	return "", db.ErrKeyNotFound
}

func (m *MockDB) PutCF(cf, key, value string) error {
	if m.readOnly {
		return db.ErrReadOnlyMode
	}
	if m.data[cf] == nil {
		m.data[cf] = make(map[string]string)
	}
	m.data[cf][key] = value
	return nil
}

func (m *MockDB) ListCFs() ([]string, error) {
	cfs := make([]string, 0, len(m.data))
	for cf := range m.data {
		cfs = append(cfs, cf)
	}
	return cfs, nil
}

func (m *MockDB) CreateCF(name string) error {
	if m.readOnly {
		return db.ErrReadOnlyMode
	}
	if m.data[name] != nil {
		return db.ErrColumnFamilyExists
	}
	m.data[name] = make(map[string]string)
	return nil
}

func (m *MockDB) DropCF(name string) error {
	if m.readOnly {
		return db.ErrReadOnlyMode
	}
	if m.data[name] == nil {
		return db.ErrColumnFamilyNotFound
	}
	delete(m.data, name)
	return nil
}

func (m *MockDB) GetKeyFormatInfo(cf string) (util.KeyFormat, string) {
	return util.KeyFormatString, "example"
}

func (m *MockDB) IsReadOnly() bool {
	return m.readOnly
}

func (m *MockDB) GetLastCF(cf string) (string, string, error) {
	return "", "", nil
}

// Stub implementations for other interface methods
func (m *MockDB) GetCF(cf, key string) (string, error) { return "", nil }
func (m *MockDB) PrefixScanCF(cf, prefix string, limit int) (map[string]string, error) { return nil, nil }
func (m *MockDB) ScanCF(cf string, start, end []byte, opts db.ScanOptions) (map[string]string, error) { return nil, nil }
func (m *MockDB) ScanCFPage(cf string, start, end []byte, opts db.ScanOptions) (db.ScanPageResult, error) { return db.ScanPageResult{}, nil }
func (m *MockDB) ExportToCSV(cf, filePath, sep string) error { return nil }
func (m *MockDB) JSONQueryCF(cf, field, value string) (map[string]string, error) { return nil, nil }
func (m *MockDB) SearchCF(cf string, opts db.SearchOptions) (*db.SearchResults, error) { return nil, nil }
func (m *MockDB) ExportSearchResultsToCSV(cf, filePath, sep string, opts db.SearchOptions) error { return nil }
func (m *MockDB) GetCFStats(cf string) (*db.CFStats, error) { return nil, nil }
func (m *MockDB) GetDatabaseStats() (*db.DatabaseStats, error) { return nil, nil }
func (m *MockDB) Close() {}
func (m *MockDB) SmartPrefixScanCF(cf, prefix string, limit int) (map[string]string, error) { return nil, nil }
func (m *MockDB) SmartScanCF(cf string, start, end string, opts db.ScanOptions) (map[string]string, error) { return nil, nil }
func (m *MockDB) SmartScanCFPage(cf string, start, end string, opts db.ScanOptions) (db.ScanPageResult, error) { return db.ScanPageResult{}, nil }

func TestDatabaseService_GetValue(t *testing.T) {
	mockDB := NewMockDB()
	mockDB.data["users"] = map[string]string{
		"user:1001": `{"name":"Alice"}`,
	}

	service := NewDatabaseService(mockDB)

	value, err := service.GetValue("users", "user:1001")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := `{"name":"Alice"}`
	if value != expected {
		t.Errorf("Expected value %s, got %s", expected, value)
	}
}

func TestDatabaseService_PutValue(t *testing.T) {
	mockDB := NewMockDB()
	mockDB.data["users"] = make(map[string]string)

	service := NewDatabaseService(mockDB)

	err := service.PutValue("users", "user:1002", `{"name":"Bob"}`)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify value was set
	value, err := service.GetValue("users", "user:1002")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := `{"name":"Bob"}`
	if value != expected {
		t.Errorf("Expected value %s, got %s", expected, value)
	}
}

func TestDatabaseService_ListColumnFamilies(t *testing.T) {
	mockDB := NewMockDB()
	mockDB.data["users"] = make(map[string]string)
	mockDB.data["products"] = make(map[string]string)

	service := NewDatabaseService(mockDB)

	cfs, err := service.ListColumnFamilies()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(cfs) != 2 {
		t.Errorf("Expected 2 column families, got %d", len(cfs))
	}
}

func TestDatabaseService_ReadOnlyMode(t *testing.T) {
	mockDB := NewMockDB()
	mockDB.readOnly = true

	service := NewDatabaseService(mockDB)

	err := service.PutValue("users", "user:1001", "value")
	if err != db.ErrReadOnlyMode {
		t.Errorf("Expected ErrReadOnlyMode, got %v", err)
	}
}
