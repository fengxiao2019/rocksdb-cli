package client

import (
	"context"
	"fmt"
	"sync"

	"rocksdb-cli/internal/config"
	"rocksdb-cli/internal/mcp/protocol"
)

// Client represents an MCP client that can connect to and interact with MCP servers
type Client interface {
	// Connection lifecycle
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool

	// Protocol operations
	Initialize(ctx context.Context) (*protocol.InitializeResult, error)
	Ping(ctx context.Context) error

	// Tool operations
	ListTools(ctx context.Context, cursor string) (*protocol.ListToolsResult, error)
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*protocol.ToolCallResult, error)

	// Configuration
	GetConfig() *config.MCPClientConfig
	GetName() string
}

// BaseClient provides common functionality for MCP clients
type BaseClient struct {
	mu          sync.RWMutex
	name        string
	config      *config.MCPClientConfig
	connected   bool
	initialized bool
	serverInfo  *protocol.ServerInfo
}

// NewBaseClient creates a new base client with the given configuration
func NewBaseClient(name string, cfg *config.MCPClientConfig) *BaseClient {
	return &BaseClient{
		name:        name,
		config:      cfg,
		connected:   false,
		initialized: false,
	}
}

// IsConnected returns whether the client is currently connected
func (bc *BaseClient) IsConnected() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.connected
}

// GetConfig returns the client configuration
func (bc *BaseClient) GetConfig() *config.MCPClientConfig {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.config
}

// GetName returns the client name
func (bc *BaseClient) GetName() string {
	return bc.name
}

// SetConnected sets the connection state (for subclasses)
func (bc *BaseClient) SetConnected(connected bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.connected = connected
}

// SetInitialized sets the initialized state (for subclasses)
func (bc *BaseClient) SetInitialized(initialized bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.initialized = initialized
}

// IsInitialized returns whether the client has been initialized
func (bc *BaseClient) IsInitialized() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.initialized
}

// SetServerInfo stores the server information (for subclasses)
func (bc *BaseClient) SetServerInfo(info *protocol.ServerInfo) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.serverInfo = info
}

// GetServerInfo returns the server information
func (bc *BaseClient) GetServerInfo() *protocol.ServerInfo {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.serverInfo
}

// CheckConnected returns an error if not connected
func (bc *BaseClient) CheckConnected() error {
	if !bc.IsConnected() {
		return fmt.Errorf("client %s is not connected", bc.name)
	}
	return nil
}

// CheckInitialized returns an error if not initialized
func (bc *BaseClient) CheckInitialized() error {
	if !bc.IsInitialized() {
		return fmt.Errorf("client %s is not initialized", bc.name)
	}
	return nil
}

// Default implementations that return errors (to be overridden by actual implementations)

func (bc *BaseClient) Connect(ctx context.Context) error {
	return fmt.Errorf("Connect not implemented for base client")
}

func (bc *BaseClient) Disconnect(ctx context.Context) error {
	return fmt.Errorf("Disconnect not implemented for base client")
}

func (bc *BaseClient) Initialize(ctx context.Context) (*protocol.InitializeResult, error) {
	return nil, fmt.Errorf("Initialize not implemented for base client")
}

func (bc *BaseClient) Ping(ctx context.Context) error {
	return fmt.Errorf("Ping not implemented for base client")
}

func (bc *BaseClient) ListTools(ctx context.Context, cursor string) (*protocol.ListToolsResult, error) {
	return nil, fmt.Errorf("ListTools not implemented for base client")
}

func (bc *BaseClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*protocol.ToolCallResult, error) {
	return nil, fmt.Errorf("CallTool not implemented for base client")
}
