package protocol

// JSON-RPC 2.0 Protocol Types

// JSONRPCVersion is the JSON-RPC version
const JSONRPCVersion = "2.0"

// MCPProtocolVersion is the MCP protocol version
const MCPProtocolVersion = "2024-11-05"

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"` // Can be string, number, or null
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      interface{}  `json:"id"`
	Result  interface{}  `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON-RPC 2.0 Standard Error Codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// MCP-specific Types

// ClientInfo holds information about the client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo holds information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Implementation holds implementation details
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Capabilities defines what features are supported
type Capabilities struct {
	Tools      *ToolsCapability      `json:"tools,omitempty"`
	Prompts    *PromptsCapability    `json:"prompts,omitempty"`
	Resources  *ResourcesCapability  `json:"resources,omitempty"`
	Logging    *LoggingCapability    `json:"logging,omitempty"`
	Completion *CompletionCapability `json:"completion,omitempty"`
}

// ToolsCapability indicates tools support
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability indicates prompts support
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates resources support
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability indicates logging support
type LoggingCapability struct{}

// CompletionCapability indicates completion support
type CompletionCapability struct{}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolCallResult represents the result of calling a tool
type ToolCallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents content in a message
type Content struct {
	Type string `json:"type"` // text, image, resource
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`
}

// Prompt represents an MCP prompt
type Prompt struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Arguments   []PromptArgument  `json:"arguments,omitempty"`
}

// PromptArgument represents a prompt argument
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

// ResourceContents represents the contents of a resource
type ResourceContents struct {
	URI      string    `json:"uri"`
	MIMEType string    `json:"mimeType,omitempty"`
	Contents []Content `json:"contents"`
}

// LogLevel represents logging levels
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
)

// LoggingMessage represents a logging message
type LoggingMessage struct {
	Level  LogLevel `json:"level"`
	Logger string   `json:"logger,omitempty"`
	Data   interface{} `json:"data"`
}
