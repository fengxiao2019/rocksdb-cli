package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"rocksdb-cli/internal/service"
)

// DBManagerHandler handles database management operations
type DBManagerHandler struct {
	manager    *service.DBManager
	mountPoints []string // Configured mount points from environment
}

// NewDBManagerHandler creates a new database manager handler
func NewDBManagerHandler(manager *service.DBManager) *DBManagerHandler {
	// Get mount points from environment variable
	// Format: /db1,/db2,/db3 or /data
	mountPointsEnv := os.Getenv("DB_MOUNT_POINTS")
	var mountPoints []string

	if mountPointsEnv != "" {
		mountPoints = strings.Split(mountPointsEnv, ",")
		// Trim whitespace
		for i := range mountPoints {
			mountPoints[i] = strings.TrimSpace(mountPoints[i])
		}
	} else {
		// Default mount point
		mountPoints = []string{"/data"}
	}

	return &DBManagerHandler{
		manager:    manager,
		mountPoints: mountPoints,
	}
}

// ConnectRequest represents a database connection request
type ConnectRequest struct {
	Path string `json:"path" binding:"required"`
}

// Connect handles database connection requests
// POST /api/v1/databases/connect
func (h *DBManagerHandler) Connect(c *gin.Context) {
	var req ConnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Validate path first
	if err := h.manager.ValidatePath(req.Path); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid database path: " + err.Error(),
		})
		return
	}

	// Connect to database
	info, err := h.manager.Connect(req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to connect to database: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Connected to database successfully",
		"data":    info,
	})
}

// Disconnect handles database disconnection requests
// POST /api/v1/databases/disconnect
func (h *DBManagerHandler) Disconnect(c *gin.Context) {
	if err := h.manager.Disconnect(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Disconnected from database successfully",
	})
}

// GetCurrent returns current database connection information
// GET /api/v1/databases/current
func (h *DBManagerHandler) GetCurrent(c *gin.Context) {
	info, err := h.manager.GetCurrentInfo()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     err.Error(),
			"connected": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
		"data":      info,
	})
}

// ListAvailable returns list of available databases from mount points
// GET /api/v1/databases/list
func (h *DBManagerHandler) ListAvailable(c *gin.Context) {
	databases, err := h.manager.ListAvailableDatabases(h.mountPoints)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list databases: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"databases":   databases,
			"mountPoints": h.mountPoints,
		},
	})
}

// ValidateRequest represents a path validation request
type ValidateRequest struct {
	Path string `json:"path" binding:"required"`
}

// Validate validates if a path is a valid RocksDB database
// POST /api/v1/databases/validate
func (h *DBManagerHandler) Validate(c *gin.Context) {
	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	if err := h.manager.ValidatePath(req.Path); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
	})
}

// GetStatus returns overall database connection status
// GET /api/v1/databases/status
func (h *DBManagerHandler) GetStatus(c *gin.Context) {
	connected := h.manager.IsConnected()

	status := gin.H{
		"connected": connected,
	}

	if connected {
		if info, err := h.manager.GetCurrentInfo(); err == nil {
			status["database"] = info
		}
	}

	c.JSON(http.StatusOK, status)
}
