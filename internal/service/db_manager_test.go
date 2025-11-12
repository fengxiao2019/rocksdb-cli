package service

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rocksdb-cli/internal/db"
)

// Helper function to create a test RocksDB database
func createTestDB(t *testing.T) string {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	// Create a simple RocksDB database
	testDB, err := db.Open(dbPath)
	require.NoError(t, err, "Failed to create test database")

	// Put some test data
	err = testDB.PutCF("default", "test-key", "test-value")
	require.NoError(t, err, "Failed to put test data")

	// Close the database
	testDB.Close()

	return dbPath
}

func TestNewDBManager(t *testing.T) {
	manager := NewDBManager()
	assert.NotNil(t, manager)
	assert.False(t, manager.IsConnected())
}

func TestDBManager_Connect_Success(t *testing.T) {
	manager := NewDBManager()
	dbPath := createTestDB(t)

	info, err := manager.Connect(dbPath)
	require.NoError(t, err)
	assert.NotNil(t, info)

	assert.Equal(t, dbPath, info.Path)
	assert.True(t, info.ReadOnly, "Database should be in read-only mode")
	assert.True(t, info.Connected)
	assert.NotZero(t, info.ConnectedAt)
	assert.Greater(t, info.CFCount, 0, "Should have at least one column family")

	// Cleanup
	err = manager.Disconnect()
	assert.NoError(t, err)
}

func TestDBManager_Connect_InvalidPath(t *testing.T) {
	manager := NewDBManager()

	// Test non-existent path
	info, err := manager.Connect("/non/existent/path")
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestDBManager_Connect_NotARocksDB(t *testing.T) {
	manager := NewDBManager()
	tmpDir := t.TempDir()

	// Create an empty directory (not a RocksDB)
	emptyDir := filepath.Join(tmpDir, "empty")
	err := os.Mkdir(emptyDir, 0755)
	require.NoError(t, err)

	info, err := manager.Connect(emptyDir)
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "failed to open database")
}

func TestDBManager_Connect_SwitchDatabase(t *testing.T) {
	manager := NewDBManager()

	// Create two test databases
	dbPath1 := createTestDB(t)
	dbPath2 := createTestDB(t)

	// Connect to first database
	info1, err := manager.Connect(dbPath1)
	require.NoError(t, err)
	assert.Equal(t, dbPath1, info1.Path)

	// Connect to second database (should close first one)
	info2, err := manager.Connect(dbPath2)
	require.NoError(t, err)
	assert.Equal(t, dbPath2, info2.Path)

	// Verify current connection is to dbPath2
	currentInfo, err := manager.GetCurrentInfo()
	require.NoError(t, err)
	assert.Equal(t, dbPath2, currentInfo.Path)

	// Cleanup
	err = manager.Disconnect()
	assert.NoError(t, err)
}

func TestDBManager_GetCurrentDB(t *testing.T) {
	manager := NewDBManager()

	// Should fail when not connected
	db, err := manager.GetCurrentDB()
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "no database connected")

	// Connect to database
	dbPath := createTestDB(t)
	_, err = manager.Connect(dbPath)
	require.NoError(t, err)

	// Should succeed now
	db, err = manager.GetCurrentDB()
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Verify it's read-only
	assert.True(t, db.IsReadOnly())

	// Cleanup
	err = manager.Disconnect()
	assert.NoError(t, err)
}

func TestDBManager_GetCurrentInfo(t *testing.T) {
	manager := NewDBManager()

	// Should fail when not connected
	info, err := manager.GetCurrentInfo()
	assert.Error(t, err)
	assert.Nil(t, info)

	// Connect to database
	dbPath := createTestDB(t)
	_, err = manager.Connect(dbPath)
	require.NoError(t, err)

	// Should succeed now
	info, err = manager.GetCurrentInfo()
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, dbPath, info.Path)

	// Cleanup
	err = manager.Disconnect()
	assert.NoError(t, err)
}

func TestDBManager_IsConnected(t *testing.T) {
	manager := NewDBManager()

	// Initially not connected
	assert.False(t, manager.IsConnected())

	// Connect
	dbPath := createTestDB(t)
	_, err := manager.Connect(dbPath)
	require.NoError(t, err)

	// Should be connected now
	assert.True(t, manager.IsConnected())

	// Disconnect
	err = manager.Disconnect()
	require.NoError(t, err)

	// Should not be connected
	assert.False(t, manager.IsConnected())
}

func TestDBManager_Disconnect(t *testing.T) {
	manager := NewDBManager()

	// Should fail when not connected
	err := manager.Disconnect()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no database connected")

	// Connect to database
	dbPath := createTestDB(t)
	_, err = manager.Connect(dbPath)
	require.NoError(t, err)

	// Disconnect should succeed
	err = manager.Disconnect()
	assert.NoError(t, err)

	// Should not be connected anymore
	assert.False(t, manager.IsConnected())
}

func TestDBManager_ValidatePath(t *testing.T) {
	manager := NewDBManager()

	tests := []struct {
		name    string
		setup   func() string
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid RocksDB path",
			setup: func() string {
				return createTestDB(t)
			},
			wantErr: false,
		},
		{
			name: "Non-existent path",
			setup: func() string {
				return "/non/existent/path"
			},
			wantErr: true,
			errMsg:  "does not exist",
		},
		{
			name: "File instead of directory",
			setup: func() string {
				tmpFile := filepath.Join(t.TempDir(), "file.txt")
				err := os.WriteFile(tmpFile, []byte("test"), 0644)
				require.NoError(t, err)
				return tmpFile
			},
			wantErr: true,
			errMsg:  "not a directory",
		},
		{
			name: "Empty directory",
			setup: func() string {
				emptyDir := filepath.Join(t.TempDir(), "empty")
				err := os.Mkdir(emptyDir, 0755)
				require.NoError(t, err)
				return emptyDir
			},
			wantErr: true,
			errMsg:  "does not appear to be a RocksDB database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			err := manager.ValidatePath(path)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDBManager_ListAvailableDatabases(t *testing.T) {
	manager := NewDBManager()
	tmpDir := t.TempDir()

	// Create mount points with databases
	mountPoint1 := filepath.Join(tmpDir, "mount1")
	mountPoint2 := filepath.Join(tmpDir, "mount2")
	err := os.MkdirAll(mountPoint1, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(mountPoint2, 0755)
	require.NoError(t, err)

	// Create databases in mount points
	db1Path := filepath.Join(mountPoint1, "db1")
	db2Path := filepath.Join(mountPoint2, "db2")

	// Create actual RocksDB databases
	db1, err := db.Open(db1Path)
	require.NoError(t, err)
	db1.Close()

	db2, err := db.Open(db2Path)
	require.NoError(t, err)
	db2.Close()

	// Also make mount point 1 itself a database
	db3, err := db.Open(mountPoint1 + "_db")
	require.NoError(t, err)
	db3.Close()

	// List databases
	mountPoints := []string{mountPoint1, mountPoint2, mountPoint1 + "_db"}
	databases, err := manager.ListAvailableDatabases(mountPoints)
	assert.NoError(t, err)
	assert.NotEmpty(t, databases)

	// Should find at least our created databases
	foundDB1 := false
	foundDB2 := false
	for _, dbPath := range databases {
		if dbPath == db1Path {
			foundDB1 = true
		}
		if dbPath == db2Path {
			foundDB2 = true
		}
	}
	assert.True(t, foundDB1, "Should find db1")
	assert.True(t, foundDB2, "Should find db2")
}

func TestDBManager_ListAvailableDatabases_NonExistentMount(t *testing.T) {
	manager := NewDBManager()

	// Use non-existent mount points
	mountPoints := []string{"/non/existent/mount"}
	databases, err := manager.ListAvailableDatabases(mountPoints)

	assert.NoError(t, err, "Should not error on non-existent mounts")
	assert.Empty(t, databases, "Should return empty list")
}

func TestDBManager_ConcurrentAccess(t *testing.T) {
	manager := NewDBManager()
	dbPath := createTestDB(t)

	// Connect to database
	_, err := manager.Connect(dbPath)
	require.NoError(t, err)

	// Test concurrent reads
	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Concurrent GetCurrentDB calls
			db, err := manager.GetCurrentDB()
			assert.NoError(t, err)
			assert.NotNil(t, db)

			// Concurrent GetCurrentInfo calls
			info, err := manager.GetCurrentInfo()
			assert.NoError(t, err)
			assert.NotNil(t, info)

			// Concurrent IsConnected calls
			connected := manager.IsConnected()
			assert.True(t, connected)
		}()
	}

	wg.Wait()

	// Cleanup
	err = manager.Disconnect()
	assert.NoError(t, err)
}

func TestDBManager_ReadOnlyEnforcement(t *testing.T) {
	manager := NewDBManager()
	dbPath := createTestDB(t)

	// Connect to database
	_, err := manager.Connect(dbPath)
	require.NoError(t, err)

	// Get database
	testDB, err := manager.GetCurrentDB()
	require.NoError(t, err)

	// Verify it's read-only
	assert.True(t, testDB.IsReadOnly())

	// Try to write (should fail in read-only mode)
	err = testDB.PutCF("default", "new-key", "new-value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read-only")

	// Cleanup
	err = manager.Disconnect()
	assert.NoError(t, err)
}

func TestDBManager_InfoImmutability(t *testing.T) {
	manager := NewDBManager()
	dbPath := createTestDB(t)

	// Connect
	_, err := manager.Connect(dbPath)
	require.NoError(t, err)

	// Get info
	info1, err := manager.GetCurrentInfo()
	require.NoError(t, err)

	// Get info again
	info2, err := manager.GetCurrentInfo()
	require.NoError(t, err)

	// Should be different pointers (copies)
	assert.NotSame(t, info1, info2, "Should return copies, not same pointer")

	// But same content
	assert.Equal(t, info1.Path, info2.Path)
	assert.Equal(t, info1.ReadOnly, info2.ReadOnly)

	// Cleanup
	err = manager.Disconnect()
	assert.NoError(t, err)
}

func TestDBManager_ConnectedAtTimestamp(t *testing.T) {
	manager := NewDBManager()
	dbPath := createTestDB(t)

	before := time.Now()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure timestamp difference

	info, err := manager.Connect(dbPath)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)
	after := time.Now()

	// ConnectedAt should be between before and after
	assert.True(t, info.ConnectedAt.After(before))
	assert.True(t, info.ConnectedAt.Before(after))

	// Cleanup
	err = manager.Disconnect()
	assert.NoError(t, err)
}
