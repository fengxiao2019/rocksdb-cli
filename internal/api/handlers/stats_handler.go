package handlers

import (
	"net/http"

	"rocksdb-cli/internal/service"

	"github.com/gin-gonic/gin"
)

// StatsHandler handles statistics-related API requests
type StatsHandler struct {
	statsService *service.StatsService
}

// NewStatsHandler creates a new StatsHandler
func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// GetDatabaseStats handles GET /api/v1/stats
// @Summary Get database statistics
// @Description Get overall database statistics including all column families
// @Tags Stats
// @Success 200 {object} map[string]interface{} "success response with database stats"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/stats [get]
func (h *StatsHandler) GetDatabaseStats(c *gin.Context) {
	stats, err := h.statsService.GetDatabaseStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": "Failed to get database statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetColumnFamilyStats handles GET /api/v1/cf/:cf/stats
// @Summary Get column family statistics
// @Description Get detailed statistics for a specific column family
// @Tags Stats
// @Param cf path string true "Column Family"
// @Success 200 {object} map[string]interface{} "success response with CF stats"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/cf/{cf}/stats [get]
func (h *StatsHandler) GetColumnFamilyStats(c *gin.Context) {
	cf := c.Param("cf")

	stats, err := h.statsService.GetColumnFamilyStats(cf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": "Failed to get column family statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cf":    cf,
			"stats": stats,
		},
	})
}
