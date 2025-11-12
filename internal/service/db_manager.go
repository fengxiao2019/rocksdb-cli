package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"rocksdb-cli/internal/db"
)

// DatabaseInfo holds information about a database connection
type DatabaseInfo struct {
	Path         string    `json:"path"`
	ReadOnly     bool      `json:"readOnly"`
	Connected    bool      `json:"connected"`
	ConnectedAt  time.Time `json:"connectedAt,omitempty"`
	CFCount      int       `json:"cfCount"`
	ColumnFamilies []string `json:"columnFamilies"`
}

// DBManager manages database connections with thread-safe switching
type DBManager struct {
	currentDB   db.KeyValueDB
	currentInfo *DatabaseInfo
	mu          sync.RWMutex
}

// NewDBManager creates a new database manager
func NewDBManager() *DBManager {
	return &DBManager{}
}

// GetCurrentDB returns the current database connection (thread-safe read)
func (m *DBManager) GetCurrentDB() (db.KeyValueDB, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentDB == nil {
		return nil, fmt.Errorf("no database connected")
	}
	return m.currentDB, nil
}

// GetCurrentInfo returns current database information
func (m *DBManager) GetCurrentInfo() (*DatabaseInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentInfo == nil {
		return nil, fmt.Errorf("no database connected")
	}

	// Return a copy to prevent external modification
	info := *m.currentInfo
	return &info, nil
}

// IsConnected returns whether a database is currently connected
func (m *DBManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentDB != nil
}

// Connect connects to a database at the specified path (read-only mode enforced)
func (m *DBManager) Connect(dbPath string) (*DatabaseInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate path exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database path does not exist: %s", dbPath)
	}

	// Close existing database if connected
	if m.currentDB != nil {
		m.currentDB.Close()
		m.currentDB = nil
		m.currentInfo = nil
	}

	// Open new database in read-only mode
	newDB, err := db.OpenReadOnly(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Get column families
	columnFamilies, err := newDB.ListCFs()
	if err != nil {
		newDB.Close()
		return nil, fmt.Errorf("failed to list column families: %w", err)
	}

	// Update current connection
	m.currentDB = newDB
	m.currentInfo = &DatabaseInfo{
		Path:           dbPath,
		ReadOnly:       true,
		Connected:      true,
		ConnectedAt:    time.Now(),
		CFCount:        len(columnFamilies),
		ColumnFamilies: columnFamilies,
	}

	return m.currentInfo, nil
}

// Disconnect closes the current database connection
func (m *DBManager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentDB == nil {
		return fmt.Errorf("no database connected")
	}

	m.currentDB.Close()
	m.currentDB = nil
	m.currentInfo = nil

	return nil
}

// ValidatePath validates whether a path could be a valid RocksDB database
func (m *DBManager) ValidatePath(dbPath string) error {
	// Check if path exists
	info, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist")
	}
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}

	// Check for RocksDB markers (CURRENT, LOCK, etc.)
	currentFile := filepath.Join(dbPath, "CURRENT")
	if _, err := os.Stat(currentFile); os.IsNotExist(err) {
		return fmt.Errorf("path does not appear to be a RocksDB database (missing CURRENT file)")
	}

	return nil
}

// ListAvailableDatabases scans mount points for available databases
// mountPoints: list of directory paths to scan (e.g., /db1, /db2, /db3)
func (m *DBManager) ListAvailableDatabases(mountPoints []string) ([]string, error) {
	var databases []string
	seen := make(map[string]bool)

	for _, mountPoint := range mountPoints {
		// Check if mount point exists
		if _, err := os.Stat(mountPoint); os.IsNotExist(err) {
			continue // Skip non-existent mount points
		}

		// List directories in mount point
		entries, err := os.ReadDir(mountPoint)
		if err != nil {
			continue // Skip on error
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			dbPath := filepath.Join(mountPoint, entry.Name())

			// Skip if already seen (deduplication)
			if seen[dbPath] {
				continue
			}

			// Validate if it's a RocksDB database
			if err := m.ValidatePath(dbPath); err == nil {
				databases = append(databases, dbPath)
				seen[dbPath] = true
			}
		}

		// Also check if the mount point itself is a database
		if err := m.ValidatePath(mountPoint); err == nil {
			if !seen[mountPoint] {
				databases = append(databases, mountPoint)
				seen[mountPoint] = true
			}
		}
	}

	return databases, nil
}
