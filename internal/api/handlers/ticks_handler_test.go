package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTicksHandler_ConvertDateTimeToTicks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTicksHandler()

	tests := []struct {
		name           string
		request        DateTimeToTicksRequest
		expectedTicks  string // Changed to string
		expectError    bool
	}{
		{
			name:          "Unix Epoch",
			request:       DateTimeToTicksRequest{DateTime: "1970-01-01T00:00:00Z"},
			expectedTicks: "621355968000000000",
			expectError:   false,
		},
		{
			name:          "User example with nanosecond precision",
			request:       DateTimeToTicksRequest{DateTime: "2025-11-26T14:44:51.5691759Z"},
			expectedTicks: "638997650915691759",
			expectError:   false,
		},
		{
			name:          "2024-01-01",
			request:       DateTimeToTicksRequest{DateTime: "2024-01-01T00:00:00Z"},
			expectedTicks: "638396640000000000",
			expectError:   false,
		},
		{
			name:        "Invalid datetime format",
			request:     DateTimeToTicksRequest{DateTime: "invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			jsonBody, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/ticks/from-datetime", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Create context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handler.ConvertDateTimeToTicks(c)

			// Check response
			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)

				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.Equal(t, tt.expectedTicks, data["ticks"].(string)) // Compare as string
			}
		})
	}
}

func TestTicksHandler_ConvertTicksToDateTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTicksHandler()

	tests := []struct {
		name             string
		request          TicksToDateTimeRequest
		expectedDateTime string
		expectError      bool
	}{
		{
			name:             "Unix Epoch",
			request:          TicksToDateTimeRequest{Ticks: "621355968000000000"},
			expectedDateTime: "1970-01-01T00:00:00Z",
			expectError:      false,
		},
		{
			name:             "User example with full precision",
			request:          TicksToDateTimeRequest{Ticks: "638997650915691759"},
			expectedDateTime: "2025-11-26T14:44:51.5691759Z",
			expectError:      false,
		},
		{
			name:        "Negative ticks",
			request:     TicksToDateTimeRequest{Ticks: "-1"},
			expectError: true,
		},
		{
			name:        "Invalid ticks format",
			request:     TicksToDateTimeRequest{Ticks: "not-a-number"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			jsonBody, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tools/ticks/to-datetime", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Create context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handler.ConvertTicksToDateTime(c)

			// Check response
			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)

				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.Equal(t, tt.expectedDateTime, data["datetime"].(string))
			}
		})
	}
}

