package service

import (
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/util"
)

// DatabaseService provides high-level database operations
// This service layer decouples business logic from CLI and API layers
type DatabaseService struct {
	db db.KeyValueDB
}

// NewDatabaseService creates a new DatabaseService instance
func NewDatabaseService(database db.KeyValueDB) *DatabaseService {
	return &DatabaseService{db: database}
}

// GetValue retrieves the value for a given key in a column family
func (s *DatabaseService) GetValue(cf, key string) (string, error) {
	return s.db.SmartGetCF(cf, key)
}

// PutValue writes or updates a key-value pair in a column family
func (s *DatabaseService) PutValue(cf, key, value string) error {
	if s.db.IsReadOnly() {
		return db.ErrReadOnlyMode
	}
	return s.db.PutCF(cf, key, value)
}

// DeleteValue deletes a key from a column family
func (s *DatabaseService) DeleteValue(cf, key string) error {
	if s.db.IsReadOnly() {
		return db.ErrReadOnlyMode
	}
	// Note: We need to add DeleteCF method to the KeyValueDB interface
	// For now, we can use PutCF with empty value as a workaround
	// This should be improved in the future
	return db.ErrReadOnlyMode // Placeholder - needs proper implementation
}

// ListColumnFamilies returns a list of all column families in the database
func (s *DatabaseService) ListColumnFamilies() ([]string, error) {
	return s.db.ListCFs()
}

// CreateColumnFamily creates a new column family
func (s *DatabaseService) CreateColumnFamily(name string) error {
	if s.db.IsReadOnly() {
		return db.ErrReadOnlyMode
	}
	return s.db.CreateCF(name)
}

// DropColumnFamily drops an existing column family
func (s *DatabaseService) DropColumnFamily(name string) error {
	if s.db.IsReadOnly() {
		return db.ErrReadOnlyMode
	}
	return s.db.DropCF(name)
}

// GetKeyFormatInfo returns the detected key format and example for a column family
func (s *DatabaseService) GetKeyFormatInfo(cf string) (util.KeyFormat, string) {
	return s.db.GetKeyFormatInfo(cf)
}

// IsReadOnly returns whether the database is in read-only mode
func (s *DatabaseService) IsReadOnly() bool {
	return s.db.IsReadOnly()
}

// GetLastEntry returns the last key-value pair from a column family
func (s *DatabaseService) GetLastEntry(cf string) (key string, value string, err error) {
	return s.db.GetLastCF(cf)
}
