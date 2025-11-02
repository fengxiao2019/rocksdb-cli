package protocol

// MCP Protocol Message Types

// InitializeRequest represents an initialize request
type InitializeRequest struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    Capabilities   `json:"capabilities"`
	ClientInfo      ClientInfo     `json:"clientInfo"`
}

// InitializeResult represents an initialize response
type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    Capabilities   `json:"capabilities"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
	Instructions    string         `json:"instructions,omitempty"`
}

// ListToolsRequest represents a tools/list request
type ListToolsRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListToolsResult represents a tools/list response
type ListToolsResult struct{
	Tools      []Tool  `json:"tools"`
	NextCursor string  `json:"nextCursor,omitempty"`
}

// CallToolRequest represents a tools/call request
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ListPromptsRequest represents a prompts/list request
type ListPromptsRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListPromptsResult represents a prompts/list response
type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

// GetPromptRequest represents a prompts/get request
type GetPromptRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// GetPromptResult represents a prompts/get response
type GetPromptResult struct {
	Description string     `json:"description,omitempty"`
	Messages    []Message  `json:"messages"`
}

// Message represents a message in a prompt
type Message struct {
	Role    string    `json:"role"` // user, assistant, system
	Content []Content `json:"content"`
}

// ListResourcesRequest represents a resources/list request
type ListResourcesRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListResourcesResult represents a resources/list response
type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// ReadResourceRequest represents a resources/read request
type ReadResourceRequest struct {
	URI string `json:"uri"`
}

// ReadResourceResult represents a resources/read response
type ReadResourceResult struct {
	Contents []ResourceContents `json:"contents"`
}

// SubscribeRequest represents a resources/subscribe request
type SubscribeRequest struct {
	URI string `json:"uri"`
}

// UnsubscribeRequest represents a resources/unsubscribe request
type UnsubscribeRequest struct {
	URI string `json:"uri"`
}

// SetLevelRequest represents a logging/setLevel request
type SetLevelRequest struct {
	Level LogLevel `json:"level"`
}

// ListChangedNotification represents a list changed notification
type ListChangedNotification struct{}

// Progress represents progress information
type Progress struct {
	ProgressToken string  `json:"progressToken"`
	Progress      float64 `json:"progress"`
	Total         float64 `json:"total,omitempty"`
}

// CancelledNotification represents a cancelled notification
type CancelledNotification struct {
	RequestID interface{} `json:"requestId"`
	Reason    string      `json:"reason,omitempty"`
}

// MCP Method Names
const (
	MethodInitialize        = "initialize"
	MethodInitialized       = "notifications/initialized"
	MethodPing              = "ping"
	MethodListTools         = "tools/list"
	MethodCallTool          = "tools/call"
	MethodListPrompts       = "prompts/list"
	MethodGetPrompt         = "prompts/get"
	MethodListResources     = "resources/list"
	MethodReadResource      = "resources/read"
	MethodSubscribe         = "resources/subscribe"
	MethodUnsubscribe       = "resources/unsubscribe"
	MethodSetLoggingLevel   = "logging/setLevel"
	MethodResourcesListChanged = "notifications/resources/list_changed"
	MethodToolsListChanged  = "notifications/tools/list_changed"
	MethodPromptsListChanged = "notifications/prompts/list_changed"
)
