package handlers

import (
	"net/http"
	"time"

	"rocksdb-cli/internal/util"

	"github.com/gin-gonic/gin"
)

// TicksHandler handles ticks conversion API requests
type TicksHandler struct{}

// NewTicksHandler creates a new TicksHandler
func NewTicksHandler() *TicksHandler {
	return &TicksHandler{}
}

// DateTimeToTicksRequest represents the request for converting datetime to ticks
type DateTimeToTicksRequest struct {
	DateTime string `json:"datetime" binding:"required"` // ISO 8601 format
}

// DateTimeToTicksResponse represents the response with ticks value
type DateTimeToTicksResponse struct {
	DateTime string `json:"datetime"` // Normalized datetime in RFC3339Nano format
	Ticks    string `json:"ticks"`    // .NET ticks value as string to preserve precision
}

// TicksToDateTimeRequest represents the request for converting ticks to datetime
type TicksToDateTimeRequest struct {
	Ticks string `json:"ticks" binding:"required"` // Ticks as string to preserve precision
}

// TicksToDateTimeResponse represents the response with datetime
type TicksToDateTimeResponse struct {
	Ticks    string `json:"ticks"`    // .NET ticks value as string to preserve precision
	DateTime string `json:"datetime"` // ISO 8601 datetime in RFC3339Nano format
}

// ConvertDateTimeToTicks handles POST /api/v1/tools/ticks/from-datetime
// @Summary Convert DateTime to .NET Ticks
// @Description Converts an ISO 8601 datetime string to .NET ticks (100-nanosecond intervals since 0001-01-01)
// @Tags Tools
// @Accept json
// @Produce json
// @Param body body DateTimeToTicksRequest true "DateTime string in ISO 8601 format"
// @Success 200 {object} map[string]interface{} "success response with ticks value"
// @Failure 400 {object} map[string]interface{} "bad request - invalid datetime format"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/tools/ticks/from-datetime [post]
func (h *TicksHandler) ConvertDateTimeToTicks(c *gin.Context) {
	var req DateTimeToTicksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	// Parse the datetime string - try multiple formats
	var parsedTime time.Time
	var parseErr error

	// Try RFC3339Nano first (most precise)
	parsedTime, parseErr = time.Parse(time.RFC3339Nano, req.DateTime)
	if parseErr != nil {
		// Try RFC3339 (standard ISO 8601)
		parsedTime, parseErr = time.Parse(time.RFC3339, req.DateTime)
	}
	if parseErr != nil {
		// Try without timezone
		parsedTime, parseErr = time.Parse("2006-01-02T15:04:05", req.DateTime)
		if parseErr == nil {
			parsedTime = time.Date(parsedTime.Year(), parsedTime.Month(), parsedTime.Day(),
				parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second(),
				parsedTime.Nanosecond(), time.UTC)
		}
	}
	if parseErr != nil {
		// Try date only
		parsedTime, parseErr = time.Parse("2006-01-02", req.DateTime)
		if parseErr == nil {
			parsedTime = time.Date(parsedTime.Year(), parsedTime.Month(), parsedTime.Day(),
				0, 0, 0, 0, time.UTC)
		}
	}

	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid datetime format",
			"message": "Supported formats: RFC3339 (2006-01-02T15:04:05Z07:00), ISO 8601, or date only (2006-01-02)",
		})
		return
	}

	// Convert to ticks
	ticks := util.TimeToTicks(parsedTime)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": DateTimeToTicksResponse{
			DateTime: parsedTime.Format(time.RFC3339Nano),
			Ticks:    util.FormatTicks(ticks), // Convert to string
		},
	})
}

// ConvertTicksToDateTime handles POST /api/v1/tools/ticks/to-datetime
// @Summary Convert .NET Ticks to DateTime
// @Description Converts .NET ticks (100-nanosecond intervals since 0001-01-01) to ISO 8601 datetime
// @Tags Tools
// @Accept json
// @Produce json
// @Param body body TicksToDateTimeRequest true "Ticks value"
// @Success 200 {object} map[string]interface{} "success response with datetime"
// @Failure 400 {object} map[string]interface{} "bad request - invalid ticks value"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/tools/ticks/to-datetime [post]
func (h *TicksHandler) ConvertTicksToDateTime(c *gin.Context) {
	var req TicksToDateTimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	// Validate ticks range (should be positive and reasonable)
	ticks, err := util.ParseTicksString(req.Ticks)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid ticks value",
			"message": "Ticks must be a valid integer: " + err.Error(),
		})
		return
	}

	if ticks < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid ticks value",
			"message": "Ticks value must be positive",
		})
		return
	}

	// Convert ticks to time
	parsedTime := util.TicksToTime(ticks)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": TicksToDateTimeResponse{
			Ticks:    req.Ticks, // Keep as string
			DateTime: parsedTime.Format(time.RFC3339Nano),
		},
	})
}

