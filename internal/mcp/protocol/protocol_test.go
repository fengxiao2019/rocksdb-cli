package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test JSON-RPC Request/Response marshaling
func TestJSONRPCRequest_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		request JSONRPCRequest
		want    string
	}{
		{
			name: "initialize request with string id",
			request: JSONRPCRequest{
				JSONRPC: JSONRPCVersion,
				ID:      "1",
				Method:  MethodInitialize,
				Params: map[string]interface{}{
					"protocolVersion": MCPProtocolVersion,
				},
			},
			want: `{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		},
		{
			name: "request with numeric id",
			request: JSONRPCRequest{
				JSONRPC: JSONRPCVersion,
				ID:      42,
				Method:  MethodListTools,
			},
			want: `{"jsonrpc":"2.0","id":42,"method":"tools/list"}`,
		},
		{
			name: "notification without id",
			request: JSONRPCRequest{
				JSONRPC: JSONRPCVersion,
				Method:  MethodInitialized,
			},
			want: `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

func TestJSONRPCRequest_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    JSONRPCRequest
		wantErr bool
	}{
		{
			name: "valid initialize request",
			json: `{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
			want: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "1",
				Method:  "initialize",
				Params: map[string]interface{}{
					"protocolVersion": "2024-11-05",
				},
			},
		},
		{
			name: "request without params",
			json: `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			want: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      float64(1), // JSON numbers unmarshal as float64
				Method:  "tools/list",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got JSONRPCRequest
			err := json.Unmarshal([]byte(tt.json), &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.JSONRPC, got.JSONRPC)
			assert.Equal(t, tt.want.Method, got.Method)
		})
	}
}

func TestJSONRPCResponse_Marshal(t *testing.T) {
	tests := []struct {
		name     string
		response JSONRPCResponse
		want     string
	}{
		{
			name: "success response",
			response: JSONRPCResponse{
				JSONRPC: JSONRPCVersion,
				ID:      "1",
				Result: map[string]string{
					"status": "ok",
				},
			},
			want: `{"jsonrpc":"2.0","id":"1","result":{"status":"ok"}}`,
		},
		{
			name: "error response",
			response: JSONRPCResponse{
				JSONRPC: JSONRPCVersion,
				ID:      "2",
				Error: &JSONRPCError{
					Code:    MethodNotFound,
					Message: "Method not found",
				},
			},
			want: `{"jsonrpc":"2.0","id":"2","error":{"code":-32601,"message":"Method not found"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

// Test MCP-specific types
func TestInitializeRequest_Marshal(t *testing.T) {
	req := InitializeRequest{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities: Capabilities{
			Tools: &ToolsCapability{
				ListChanged: true,
			},
		},
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var got InitializeRequest
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, req.ProtocolVersion, got.ProtocolVersion)
	assert.Equal(t, req.ClientInfo.Name, got.ClientInfo.Name)
	assert.NotNil(t, got.Capabilities.Tools)
	assert.True(t, got.Capabilities.Tools.ListChanged)
}

func TestTool_Marshal(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]string{
					"type": "string",
				},
			},
			"required": []string{"param1"},
		},
	}

	data, err := json.Marshal(tool)
	require.NoError(t, err)

	var got Tool
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, tool.Name, got.Name)
	assert.Equal(t, tool.Description, got.Description)
	assert.NotNil(t, got.InputSchema)
}

func TestCallToolRequest_Marshal(t *testing.T) {
	req := CallToolRequest{
		Name: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
			"param2": 42,
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	want := `{"name":"test_tool","arguments":{"param1":"value1","param2":42}}`
	assert.JSONEq(t, want, string(data))

	var got CallToolRequest
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)
	assert.Equal(t, req.Name, got.Name)
	assert.Equal(t, "value1", got.Arguments["param1"])
}

func TestToolCallResult_Marshal(t *testing.T) {
	result := ToolCallResult{
		Content: []Content{
			{
				Type: "text",
				Text: "Result text",
			},
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var got ToolCallResult
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Len(t, got.Content, 1)
	assert.Equal(t, "text", got.Content[0].Type)
	assert.Equal(t, "Result text", got.Content[0].Text)
	assert.False(t, got.IsError)
}

// Test Error types
func TestMCPError_Error(t *testing.T) {
	tests := []struct {
		name     string
		mcpError *MCPError
		want     string
	}{
		{
			name: "error without data",
			mcpError: &MCPError{
				Code:    ErrorToolNotFound,
				Message: "Tool not found",
			},
			want: "MCP error -32020: Tool not found",
		},
		{
			name: "error with data",
			mcpError: &MCPError{
				Code:    ErrorToolExecutionFailed,
				Message: "Tool execution failed",
				Data: map[string]string{
					"tool": "test_tool",
				},
			},
			want: "MCP error -32021: Tool execution failed (data: map[tool:test_tool])",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mcpError.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMCPError_ToJSONRPCError(t *testing.T) {
	mcpErr := &MCPError{
		Code:    ErrorToolNotFound,
		Message: "Tool not found",
		Data:    map[string]string{"tool": "test"},
	}

	jsonRPCErr := mcpErr.ToJSONRPCError()

	assert.Equal(t, mcpErr.Code, jsonRPCErr.Code)
	assert.Equal(t, mcpErr.Message, jsonRPCErr.Message)
	assert.Equal(t, mcpErr.Data, jsonRPCErr.Data)
}

func TestNewConnectionError(t *testing.T) {
	err := NewConnectionError("Connection failed", map[string]string{
		"reason": "timeout",
	})

	assert.Equal(t, ErrorConnectionFailed, err.Code)
	assert.Equal(t, "Connection failed", err.Message)
	assert.NotNil(t, err.Data)
}

func TestNewConnectionTimeoutError(t *testing.T) {
	err := NewConnectionTimeoutError("Connection timed out")

	assert.Equal(t, ErrorConnectionTimeout, err.Code)
	assert.Equal(t, "Connection timed out", err.Message)
	assert.Nil(t, err.Data)
}

func TestNewProtocolMismatchError(t *testing.T) {
	err := NewProtocolMismatchError("2024-11-05", "2024-10-01")

	assert.Equal(t, ErrorProtocolMismatch, err.Code)
	assert.Contains(t, err.Message, "2024-11-05")
	assert.Contains(t, err.Message, "2024-10-01")
	assert.NotNil(t, err.Data)
}

func TestNewToolNotFoundError(t *testing.T) {
	err := NewToolNotFoundError("missing_tool")

	assert.Equal(t, ErrorToolNotFound, err.Code)
	assert.Contains(t, err.Message, "missing_tool")

	data, ok := err.Data.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "missing_tool", data["tool"])
}

func TestNewToolExecutionError(t *testing.T) {
	originalErr := assert.AnError
	err := NewToolExecutionError("test_tool", originalErr)

	assert.Equal(t, ErrorToolExecutionFailed, err.Code)
	assert.Contains(t, err.Message, "test_tool")

	data, ok := err.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test_tool", data["tool"])
	assert.Equal(t, originalErr.Error(), data["error"])
}

func TestNewResourceNotFoundError(t *testing.T) {
	err := NewResourceNotFoundError("file:///missing.txt")

	assert.Equal(t, ErrorResourceNotFound, err.Code)
	assert.Contains(t, err.Message, "file:///missing.txt")

	data, ok := err.Data.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "file:///missing.txt", data["uri"])
}

func TestNewPromptNotFoundError(t *testing.T) {
	err := NewPromptNotFoundError("missing_prompt")

	assert.Equal(t, ErrorPromptNotFound, err.Code)
	assert.Contains(t, err.Message, "missing_prompt")

	data, ok := err.Data.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "missing_prompt", data["prompt"])
}

func TestMCPError_IsRetryable(t *testing.T) {
	tests := []struct {
		name         string
		errorCode    int
		wantRetryable bool
	}{
		{
			name:         "connection timeout is retryable",
			errorCode:    ErrorConnectionTimeout,
			wantRetryable: true,
		},
		{
			name:         "connection closed is retryable",
			errorCode:    ErrorConnectionClosed,
			wantRetryable: true,
		},
		{
			name:         "tool timeout is retryable",
			errorCode:    ErrorToolTimeout,
			wantRetryable: true,
		},
		{
			name:         "resource timeout is retryable",
			errorCode:    ErrorResourceTimeout,
			wantRetryable: true,
		},
		{
			name:         "connection failed is not retryable",
			errorCode:    ErrorConnectionFailed,
			wantRetryable: false,
		},
		{
			name:         "tool not found is not retryable",
			errorCode:    ErrorToolNotFound,
			wantRetryable: false,
		},
		{
			name:         "protocol mismatch is not retryable",
			errorCode:    ErrorProtocolMismatch,
			wantRetryable: false,
		},
		{
			name:         "invalid params is not retryable",
			errorCode:    InvalidParams,
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &MCPError{
				Code:    tt.errorCode,
				Message: "test error",
			}
			assert.Equal(t, tt.wantRetryable, err.IsRetryable())
		})
	}
}

// Test Capabilities
func TestCapabilities_Marshal(t *testing.T) {
	caps := Capabilities{
		Tools: &ToolsCapability{
			ListChanged: true,
		},
		Prompts: &PromptsCapability{
			ListChanged: false,
		},
		Resources: &ResourcesCapability{
			Subscribe:   true,
			ListChanged: true,
		},
		Logging: &LoggingCapability{},
	}

	data, err := json.Marshal(caps)
	require.NoError(t, err)

	var got Capabilities
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.NotNil(t, got.Tools)
	assert.True(t, got.Tools.ListChanged)
	assert.NotNil(t, got.Prompts)
	assert.False(t, got.Prompts.ListChanged)
	assert.NotNil(t, got.Resources)
	assert.True(t, got.Resources.Subscribe)
}

// Test Content types
func TestContent_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		content Content
		want    string
	}{
		{
			name: "text content",
			content: Content{
				Type: "text",
				Text: "Hello, world!",
			},
			want: `{"type":"text","text":"Hello, world!"}`,
		},
		{
			name: "image content",
			content: Content{
				Type:     "image",
				Data:     "base64data",
				MIMEType: "image/png",
			},
			want: `{"type":"image","data":"base64data","mimeType":"image/png"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.content)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

// Test Message types
func TestMessage_Marshal(t *testing.T) {
	msg := Message{
		Role: "user",
		Content: []Content{
			{
				Type: "text",
				Text: "Test message",
			},
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var got Message
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, "user", got.Role)
	assert.Len(t, got.Content, 1)
	assert.Equal(t, "text", got.Content[0].Type)
	assert.Equal(t, "Test message", got.Content[0].Text)
}

// Test LogLevel
func TestLogLevel_Values(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  string
	}{
		{"debug level", LogLevelDebug, "debug"},
		{"info level", LogLevelInfo, "info"},
		{"warning level", LogLevelWarning, "warning"},
		{"error level", LogLevelError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.level))
		})
	}
}

// Test Method constants
func TestMethodConstants(t *testing.T) {
	assert.Equal(t, "initialize", MethodInitialize)
	assert.Equal(t, "notifications/initialized", MethodInitialized)
	assert.Equal(t, "ping", MethodPing)
	assert.Equal(t, "tools/list", MethodListTools)
	assert.Equal(t, "tools/call", MethodCallTool)
	assert.Equal(t, "prompts/list", MethodListPrompts)
	assert.Equal(t, "prompts/get", MethodGetPrompt)
	assert.Equal(t, "resources/list", MethodListResources)
	assert.Equal(t, "resources/read", MethodReadResource)
	assert.Equal(t, "resources/subscribe", MethodSubscribe)
	assert.Equal(t, "resources/unsubscribe", MethodUnsubscribe)
	assert.Equal(t, "logging/setLevel", MethodSetLoggingLevel)
	assert.Equal(t, "notifications/resources/list_changed", MethodResourcesListChanged)
	assert.Equal(t, "notifications/tools/list_changed", MethodToolsListChanged)
	assert.Equal(t, "notifications/prompts/list_changed", MethodPromptsListChanged)
}

// Test Error code constants
func TestErrorCodeConstants(t *testing.T) {
	// JSON-RPC standard errors
	assert.Equal(t, -32700, ParseError)
	assert.Equal(t, -32600, InvalidRequest)
	assert.Equal(t, -32601, MethodNotFound)
	assert.Equal(t, -32602, InvalidParams)
	assert.Equal(t, -32603, InternalError)

	// MCP-specific errors
	assert.Equal(t, -32000, ErrorConnectionFailed)
	assert.Equal(t, -32001, ErrorConnectionTimeout)
	assert.Equal(t, -32002, ErrorConnectionClosed)
	assert.Equal(t, -32010, ErrorProtocolMismatch)
	assert.Equal(t, -32011, ErrorUnsupportedMethod)
	assert.Equal(t, -32012, ErrorInvalidResponse)
	assert.Equal(t, -32020, ErrorToolNotFound)
	assert.Equal(t, -32021, ErrorToolExecutionFailed)
	assert.Equal(t, -32022, ErrorToolTimeout)
	assert.Equal(t, -32023, ErrorInvalidToolInput)
	assert.Equal(t, -32030, ErrorResourceNotFound)
	assert.Equal(t, -32031, ErrorResourceAccessDenied)
	assert.Equal(t, -32032, ErrorResourceTimeout)
	assert.Equal(t, -32040, ErrorPromptNotFound)
	assert.Equal(t, -32041, ErrorInvalidPromptArgs)
}

// Test Protocol version
func TestProtocolVersion(t *testing.T) {
	assert.Equal(t, "2.0", JSONRPCVersion)
	assert.Equal(t, "2024-11-05", MCPProtocolVersion)
}
