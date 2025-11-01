package handlers

import (
	"net/http"

	"rocksdb-cli/internal/service"

	"github.com/gin-gonic/gin"
)

// DatabaseHandler handles database-related API requests
type DatabaseHandler struct {
	dbService *service.DatabaseService
}

// NewDatabaseHandler creates a new DatabaseHandler
func NewDatabaseHandler(dbService *service.DatabaseService) *DatabaseHandler {
	return &DatabaseHandler{dbService: dbService}
}

// GetValue handles GET /api/v1/cf/:cf/get/:key
// @Summary Get value by key
// @Description Retrieve the value for a specific key in a column family
// @Tags Database
// @Param cf path string true "Column Family"
// @Param key path string true "Key"
// @Success 200 {object} map[string]interface{} "success response with key and value"
// @Failure 404 {object} map[string]interface{} "key not found"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/cf/{cf}/get/{key} [get]
func (h *DatabaseHandler) GetValue(c *gin.Context) {
	cf := c.Param("cf")
	key := c.Param("key")

	value, err := h.dbService.GetValue(cf, key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": "Key not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cf":    cf,
			"key":   key,
			"value": value,
		},
	})
}

// PutValue handles POST /api/v1/cf/:cf/put
// @Summary Put key-value pair
// @Description Write or update a key-value pair in a column family
// @Tags Database
// @Param cf path string true "Column Family"
// @Param body body map[string]string true "Key-Value pair"
// @Success 200 {object} map[string]interface{} "success response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 403 {object} map[string]interface{} "read-only mode"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/cf/{cf}/put [post]
func (h *DatabaseHandler) PutValue(c *gin.Context) {
	cf := c.Param("cf")

	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"message": "Both 'key' and 'value' fields are required",
		})
		return
	}

	err := h.dbService.PutValue(cf, req.Key, req.Value)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Failed to put value"

		// Check if it's a read-only error
		if err.Error() == "operation not allowed in read-only mode" {
			statusCode = http.StatusForbidden
			message = "Database is in read-only mode"
		}

		c.JSON(statusCode, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Value saved successfully",
		"data": gin.H{
			"cf":    cf,
			"key":   req.Key,
			"value": req.Value,
		},
	})
}

// DeleteValue handles DELETE /api/v1/cf/:cf/delete/:key
// @Summary Delete a key
// @Description Delete a key from a column family
// @Tags Database
// @Param cf path string true "Column Family"
// @Param key path string true "Key"
// @Success 200 {object} map[string]interface{} "success response"
// @Failure 403 {object} map[string]interface{} "read-only mode"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/cf/{cf}/delete/{key} [delete]
func (h *DatabaseHandler) DeleteValue(c *gin.Context) {
	cf := c.Param("cf")
	key := c.Param("key")

	err := h.dbService.DeleteValue(cf, key)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Failed to delete key"

		if err.Error() == "operation not allowed in read-only mode" {
			statusCode = http.StatusForbidden
			message = "Database is in read-only mode"
		}

		c.JSON(statusCode, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Key deleted successfully",
		"data": gin.H{
			"cf":  cf,
			"key": key,
		},
	})
}

// ListColumnFamilies handles GET /api/v1/cf
// @Summary List column families
// @Description Get a list of all column families in the database
// @Tags Database
// @Success 200 {object} map[string]interface{} "success response with column families list"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/cf [get]
func (h *DatabaseHandler) ListColumnFamilies(c *gin.Context) {
	cfs, err := h.dbService.ListColumnFamilies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": "Failed to list column families",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"column_families": cfs,
			"count":           len(cfs),
		},
	})
}

// GetLastEntry handles GET /api/v1/cf/:cf/last
// @Summary Get last entry
// @Description Get the last key-value pair from a column family
// @Tags Database
// @Param cf path string true "Column Family"
// @Success 200 {object} map[string]interface{} "success response"
// @Failure 404 {object} map[string]interface{} "no entries found"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/cf/{cf}/last [get]
func (h *DatabaseHandler) GetLastEntry(c *gin.Context) {
	cf := c.Param("cf")

	key, value, err := h.dbService.GetLastEntry(cf)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": "No entries found in column family",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cf":    cf,
			"key":   key,
			"value": value,
		},
	})
}
