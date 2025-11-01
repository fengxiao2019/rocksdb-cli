package handlers

import (
	"net/http"

	"rocksdb-cli/internal/service"

	"github.com/gin-gonic/gin"
)

// SearchHandler handles search-related API requests
type SearchHandler struct {
	searchService *service.SearchService
}

// NewSearchHandler creates a new SearchHandler
func NewSearchHandler(searchService *service.SearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

// Search handles POST /api/v1/cf/:cf/search
// @Summary Advanced search
// @Description Perform advanced search with regex and pagination support
// @Tags Search
// @Param cf path string true "Column Family"
// @Param body body service.SearchOptions true "Search options"
// @Success 200 {object} map[string]interface{} "success response with search results"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/cf/{cf}/search [post]
func (h *SearchHandler) Search(c *gin.Context) {
	cf := c.Param("cf")

	var opts service.SearchOptions
	if err := c.ShouldBindJSON(&opts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	// Validate that at least one pattern is provided
	if opts.KeyPattern == "" && opts.ValuePattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Missing search pattern",
			"message": "At least one of 'key_pattern' or 'value_pattern' must be provided",
		})
		return
	}

	// Set default limit if not provided
	if opts.Limit == 0 {
		opts.Limit = 50
	}

	result, err := h.searchService.Search(cf, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": "Search operation failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cf":          cf,
			"results":     result.Results,
			"count":       result.Count,
			"total":       result.Total,
			"has_more":    result.HasMore,
			"next_cursor": result.NextCursor,
			"query_time":  result.QueryTime,
		},
	})
}

// JSONQuery handles POST /api/v1/cf/:cf/jsonquery
// @Summary JSON field query
// @Description Query entries by JSON field value
// @Tags Search
// @Param cf path string true "Column Family"
// @Param body body map[string]string true "Field and value"
// @Success 200 {object} map[string]interface{} "success response with matching entries"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/cf/{cf}/jsonquery [post]
func (h *SearchHandler) JSONQuery(c *gin.Context) {
	cf := c.Param("cf")

	var req struct {
		Field string `json:"field" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"message": "Both 'field' and 'value' are required",
		})
		return
	}

	result, err := h.searchService.JSONQuery(cf, req.Field, req.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"message": "JSON query failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cf":      cf,
			"field":   result.Field,
			"value":   result.Value,
			"results": result.Data,
			"count":   result.Count,
		},
	})
}
