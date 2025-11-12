package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/service"
)

func init() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
}

// Helper function to create a test RocksDB database
func createTestDB(t *testing.T) string {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	testDB, err := db.Open(dbPath)
	require.NoError(t, err)

	err = testDB.PutCF("default", "test-key", "test-value")
	require.NoError(t, err)

	testDB.Close()

	return dbPath
}

func setupTestHandler(t *testing.T) (*DBManagerHandler, *service.DBManager) {
	manager := service.NewDBManager()
	handler := NewDBManagerHandler(manager)
	return handler, manager
}

func TestNewDBManagerHandler(t *testing.T) {
	manager := service.NewDBManager()
	handler := NewDBManagerHandler(manager)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.manager)
	assert.NotEmpty(t, handler.mountPoints)
}

func TestDBManagerHandler_Connect_Success(t *testing.T) {
	handler, _ := setupTestHandler(t)
	dbPath := createTestDB(t)

	// Create request
	reqBody := ConnectRequest{Path: dbPath}
	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// Create HTTP request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/databases/connect", bytes.NewReader(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Handle request
	handler.Connect(c)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Connected to database successfully", response["message"])
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.Equal(t, dbPath, data["path"])
	assert.True(t, data["readOnly"].(bool))
	assert.True(t, data["connected"].(bool))
}

func TestDBManagerHandler_Connect_InvalidPath(t *testing.T) {
	handler, _ := setupTestHandler(t)

	reqBody := ConnectRequest{Path: "/non/existent/path"}
	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/databases/connect", bytes.NewReader(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Connect(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Invalid database path")
}

func TestDBManagerHandler_Connect_InvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/databases/connect", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Connect(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Invalid request")
}

func TestDBManagerHandler_Disconnect_Success(t *testing.T) {
	handler, manager := setupTestHandler(t)
	dbPath := createTestDB(t)

	// Connect first
	_, err := manager.Connect(dbPath)
	require.NoError(t, err)

	// Disconnect via handler
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/databases/disconnect", nil)

	handler.Disconnect(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Disconnected from database successfully", response["message"])
}

func TestDBManagerHandler_Disconnect_NoConnection(t *testing.T) {
	handler, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/databases/disconnect", nil)

	handler.Disconnect(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "no database connected")
}

func TestDBManagerHandler_GetCurrent_Connected(t *testing.T) {
	handler, manager := setupTestHandler(t)
	dbPath := createTestDB(t)

	// Connect
	_, err := manager.Connect(dbPath)
	require.NoError(t, err)

	// Get current
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/databases/current", nil)

	handler.GetCurrent(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["connected"].(bool))
	assert.NotNil(t, response["data"])

	// Cleanup
	_ = manager.Disconnect()
}

func TestDBManagerHandler_GetCurrent_NotConnected(t *testing.T) {
	handler, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/databases/current", nil)

	handler.GetCurrent(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["connected"].(bool))
	assert.NotNil(t, response["error"])
}

func TestDBManagerHandler_ListAvailable(t *testing.T) {
	handler, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/databases/list", nil)

	handler.ListAvailable(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check that data exists
	require.Contains(t, response, "data")
	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok, "data should be a map")

	// Databases can be nil or an empty array when no databases are found
	assert.Contains(t, data, "databases")
	assert.Contains(t, data, "mountPoints")
}

func TestDBManagerHandler_ListAvailable_WithDatabases(t *testing.T) {
	// Create temporary mount point with databases
	tmpDir := t.TempDir()
	mountPoint := filepath.Join(tmpDir, "mount")
	err := os.MkdirAll(mountPoint, 0755)
	require.NoError(t, err)

	// Create a test database in mount point
	dbPath := filepath.Join(mountPoint, "testdb")
	testDB, err := db.Open(dbPath)
	require.NoError(t, err)
	testDB.Close()

	// Set environment variable for mount points
	originalEnv := os.Getenv("DB_MOUNT_POINTS")
	defer os.Setenv("DB_MOUNT_POINTS", originalEnv)
	os.Setenv("DB_MOUNT_POINTS", mountPoint)

	// Create handler (will read env var)
	manager := service.NewDBManager()
	handler := NewDBManagerHandler(manager)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/databases/list", nil)

	handler.ListAvailable(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	databases := data["databases"].([]interface{})

	// Should find our test database
	found := false
	for _, dbPathIface := range databases {
		if dbPathIface.(string) == dbPath {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find test database in list")
}

func TestDBManagerHandler_Validate_Valid(t *testing.T) {
	handler, _ := setupTestHandler(t)
	dbPath := createTestDB(t)

	reqBody := ValidateRequest{Path: dbPath}
	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/databases/validate", bytes.NewReader(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Validate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["valid"].(bool))
	assert.Nil(t, response["error"])
}

func TestDBManagerHandler_Validate_Invalid(t *testing.T) {
	handler, _ := setupTestHandler(t)

	reqBody := ValidateRequest{Path: "/non/existent/path"}
	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/databases/validate", bytes.NewReader(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Validate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["valid"].(bool))
	assert.NotNil(t, response["error"])
}

func TestDBManagerHandler_GetStatus_Connected(t *testing.T) {
	handler, manager := setupTestHandler(t)
	dbPath := createTestDB(t)

	// Connect
	_, err := manager.Connect(dbPath)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/databases/status", nil)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["connected"].(bool))
	assert.NotNil(t, response["database"])

	// Cleanup
	_ = manager.Disconnect()
}

func TestDBManagerHandler_GetStatus_NotConnected(t *testing.T) {
	handler, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/databases/status", nil)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["connected"].(bool))
	assert.Nil(t, response["database"])
}

func TestDBManagerHandler_MountPoints_FromEnv(t *testing.T) {
	// Test with custom mount points
	originalEnv := os.Getenv("DB_MOUNT_POINTS")
	defer os.Setenv("DB_MOUNT_POINTS", originalEnv)

	os.Setenv("DB_MOUNT_POINTS", "/mount1,/mount2,/mount3")

	manager := service.NewDBManager()
	handler := NewDBManagerHandler(manager)

	expected := []string{"/mount1", "/mount2", "/mount3"}
	assert.Equal(t, expected, handler.mountPoints)
}

func TestDBManagerHandler_MountPoints_Default(t *testing.T) {
	// Test with no environment variable (should use default)
	originalEnv := os.Getenv("DB_MOUNT_POINTS")
	defer os.Setenv("DB_MOUNT_POINTS", originalEnv)

	os.Unsetenv("DB_MOUNT_POINTS")

	manager := service.NewDBManager()
	handler := NewDBManagerHandler(manager)

	expected := []string{"/data"}
	assert.Equal(t, expected, handler.mountPoints)
}
