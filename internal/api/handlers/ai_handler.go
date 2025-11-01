package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"rocksdb-cli/internal/graphchain"

	"github.com/gin-gonic/gin"
)

// AIHandler handles AI-powered query requests
type AIHandler struct {
	agent *graphchain.Agent
}

// NewAIHandler creates a new AIHandler
func NewAIHandler(agent interface{}) *AIHandler {
	if a, ok := agent.(*graphchain.Agent); ok {
		return &AIHandler{
			agent: a,
		}
	}
	panic(fmt.Sprintf("invalid agent type: %T", agent))
}

// QueryRequest represents an AI query request
type QueryRequest struct {
	Query string `json:"query" binding:"required"`
}

// QueryResponse represents an AI query response
type QueryResponse struct {
	Success        bool        `json:"success"`
	Data           interface{} `json:"data,omitempty"`
	Error          string      `json:"error,omitempty"`
	ErrorType      string      `json:"error_type,omitempty"`
	Explanation    string      `json:"explanation,omitempty"`
	ExecutionTime  string      `json:"execution_time"`
	ToolsUsed      []string    `json:"tools_used,omitempty"`
	IntentDetected string      `json:"intent_detected,omitempty"`
}

// Query handles AI-powered natural language queries
// @Summary Process natural language query
// @Description Process a natural language query using GraphChain AI agent
// @Tags AI
// @Accept json
// @Produce json
// @Param query body QueryRequest true "Natural language query"
// @Success 200 {object} QueryResponse
// @Router /api/v1/ai/query [post]
func (h *AIHandler) Query(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// Process query
	result, err := h.agent.ProcessQuery(ctx, req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Query processing failed: " + err.Error(),
		})
		return
	}

	// Convert to response
	response := QueryResponse{
		Success:        result.Success,
		Data:           result.Data,
		Error:          result.Error,
		ErrorType:      string(result.ErrorType),
		Explanation:    result.Explanation,
		ExecutionTime:  result.ExecutionTime.String(),
		ToolsUsed:      result.ToolsUsed,
		IntentDetected: result.IntentDetected,
	}

	if result.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// GetCapabilities returns the AI agent's capabilities
// @Summary Get AI agent capabilities
// @Description Get list of available capabilities and tools
// @Tags AI
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ai/capabilities [get]
func (h *AIHandler) GetCapabilities(c *gin.Context) {
	capabilities := h.agent.GetCapabilities()

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"capabilities": capabilities,
		"count":        len(capabilities),
	})
}

// HealthCheck checks if AI agent is ready
// @Summary AI health check
// @Description Check if AI agent is initialized and ready
// @Tags AI
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ai/health [get]
func (h *AIHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "ready",
		"agent":   "GraphChain",
	})
}
