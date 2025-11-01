package handlers

import (
	"net/http"

	"rocksdb-cli/internal/service"

	"github.com/gin-gonic/gin"
)

// ScanHandler handles scan-related API requests
type ScanHandler struct {
	scanService *service.ScanService
}

// NewScanHandler creates a new ScanHandler
func NewScanHandler(scanService *service.ScanService) *ScanHandler {
	return &ScanHandler{scanService: scanService}
}

// Scan handles POST /api/v1/cf/:cf/scan
// @Summary Scan key-value pairs
// @Description Perform a range scan on a column family with pagination support
// @Tags Scan
// @Param cf path string true "Column Family"
// @Param body body service.ScanOptions false "Scan options"
// @Success 200 {object} map[string]interface{} "success response with scan results"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/cf/{cf}/scan [post]
func (h *ScanHandler) Scan(c *gin.Context) {
	cf := c.Param("cf")

	var opts service.ScanOptions
	// Set defaults
	opts.Limit = 50
	opts.KeysOnly = false
	opts.Reverse = false

	// Bind JSON if provided
	if err := c.ShouldBindJSON(&opts); err != nil {
		// If no body provided, use defaults
		opts = service.ScanOptions{
			Limit:    50,
			KeysOnly: false,
			Reverse:  false,
		}
	}

	result, err := h.scanService.Scan(cf, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": "Scan operation failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cf":          cf,
			"results":     result.Data,    // Keep for backward compatibility
			"results_v2":  result.ResultsV2, // New format with binary support
			"count":       result.Count,
			"has_more":    result.HasMore,
			"next_cursor": result.NextCursor,
		},
	})
}

// PrefixScan handles POST /api/v1/cf/:cf/prefix
// @Summary Prefix scan
// @Description Scan keys with a specific prefix
// @Tags Scan
// @Param cf path string true "Column Family"
// @Param body body map[string]interface{} true "Prefix and limit"
// @Success 200 {object} map[string]interface{} "success response with matching entries"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/cf/{cf}/prefix [post]
func (h *ScanHandler) PrefixScan(c *gin.Context) {
	cf := c.Param("cf")

	var req struct {
		Prefix string `json:"prefix" binding:"required"`
		Limit  int    `json:"limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"message": "Field 'prefix' is required",
		})
		return
	}

	// Default limit to 0 (no limit) if not specified
	if req.Limit == 0 {
		req.Limit = 100 // Set a reasonable default
	}

	result, err := h.scanService.PrefixScan(cf, req.Prefix, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": "Prefix scan failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cf":      cf,
			"prefix":  req.Prefix,
			"results": result.Data,
			"count":   result.Count,
		},
	})
}
