package protocol

import "fmt"

// MCPError represents an MCP-specific error
type MCPError struct {
	Code    int
	Message string
	Data    interface{}
}

// Error implements the error interface
func (e *MCPError) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("MCP error %d: %s (data: %v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// ToJSONRPCError converts MCPError to JSONRPCError
func (e *MCPError) ToJSONRPCError() *JSONRPCError {
	return &JSONRPCError{
		Code:    e.Code,
		Message: e.Message,
		Data:    e.Data,
	}
}

// MCP-specific Error Codes (starting from -32000 as per JSON-RPC spec)
const (
	// Client errors
	ErrorConnectionFailed    = -32000
	ErrorConnectionTimeout   = -32001
	ErrorConnectionClosed    = -32002

	// Protocol errors
	ErrorProtocolMismatch    = -32010
	ErrorUnsupportedMethod   = -32011
	ErrorInvalidResponse     = -32012

	// Tool errors
	ErrorToolNotFound        = -32020
	ErrorToolExecutionFailed = -32021
	ErrorToolTimeout         = -32022
	ErrorInvalidToolInput    = -32023

	// Resource errors
	ErrorResourceNotFound    = -32030
	ErrorResourceAccessDenied = -32031
	ErrorResourceTimeout     = -32032

	// Prompt errors
	ErrorPromptNotFound      = -32040
	ErrorInvalidPromptArgs   = -32041
)

// Predefined MCP Errors

// NewConnectionError creates a new connection error
func NewConnectionError(message string, data interface{}) *MCPError {
	return &MCPError{
		Code:    ErrorConnectionFailed,
		Message: message,
		Data:    data,
	}
}

// NewConnectionTimeoutError creates a new connection timeout error
func NewConnectionTimeoutError(message string) *MCPError {
	return &MCPError{
		Code:    ErrorConnectionTimeout,
		Message: message,
	}
}

// NewProtocolMismatchError creates a new protocol mismatch error
func NewProtocolMismatchError(expected, actual string) *MCPError {
	return &MCPError{
		Code:    ErrorProtocolMismatch,
		Message: fmt.Sprintf("protocol version mismatch: expected %s, got %s", expected, actual),
		Data: map[string]string{
			"expected": expected,
			"actual":   actual,
		},
	}
}

// NewToolNotFoundError creates a new tool not found error
func NewToolNotFoundError(toolName string) *MCPError {
	return &MCPError{
		Code:    ErrorToolNotFound,
		Message: fmt.Sprintf("tool not found: %s", toolName),
		Data:    map[string]string{"tool": toolName},
	}
}

// NewToolExecutionError creates a new tool execution error
func NewToolExecutionError(toolName string, err error) *MCPError {
	return &MCPError{
		Code:    ErrorToolExecutionFailed,
		Message: fmt.Sprintf("tool execution failed: %s", toolName),
		Data: map[string]interface{}{
			"tool":  toolName,
			"error": err.Error(),
		},
	}
}

// NewResourceNotFoundError creates a new resource not found error
func NewResourceNotFoundError(uri string) *MCPError {
	return &MCPError{
		Code:    ErrorResourceNotFound,
		Message: fmt.Sprintf("resource not found: %s", uri),
		Data:    map[string]string{"uri": uri},
	}
}

// NewPromptNotFoundError creates a new prompt not found error
func NewPromptNotFoundError(promptName string) *MCPError {
	return &MCPError{
		Code:    ErrorPromptNotFound,
		Message: fmt.Sprintf("prompt not found: %s", promptName),
		Data:    map[string]string{"prompt": promptName},
	}
}

// IsRetryable returns true if the error is retryable
func (e *MCPError) IsRetryable() bool {
	switch e.Code {
	case ErrorConnectionTimeout,
		ErrorConnectionClosed,
		ErrorToolTimeout,
		ErrorResourceTimeout:
		return true
	default:
		return false
	}
}
