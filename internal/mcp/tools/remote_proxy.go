package tools

import (
	"context"
	"fmt"
	"sync"

	"rocksdb-cli/internal/mcp/client"
	"rocksdb-cli/internal/mcp/protocol"
)

// RemoteProxy proxies tool executions to remote MCP clients
type RemoteProxy struct {
	registry  *Registry
	manager   *client.Manager
	mu        sync.RWMutex
	autoSync  bool
}

// NewRemoteProxy creates a new remote tool proxy
func NewRemoteProxy(registry *Registry, manager *client.Manager) *RemoteProxy {
	return &RemoteProxy{
		registry: registry,
		manager:  manager,
		autoSync: false,
	}
}

// EnableAutoSync enables automatic tool synchronization when clients connect
func (rp *RemoteProxy) EnableAutoSync() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.autoSync = true
}

// DisableAutoSync disables automatic tool synchronization
func (rp *RemoteProxy) DisableAutoSync() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.autoSync = false
}

// SyncTools synchronizes tools from a specific remote client
func (rp *RemoteProxy) SyncTools(ctx context.Context, clientName string) error {
	// Get the client
	mcpClient := rp.manager.GetClient(clientName)
	if mcpClient == nil {
		return fmt.Errorf("client %s not found", clientName)
	}

	// Check if client is connected and initialized
	if !mcpClient.IsConnected() || !mcpClient.IsInitialized() {
		return fmt.Errorf("client %s is not connected or initialized", clientName)
	}

	// List tools from the client
	toolsResult, err := mcpClient.ListTools(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to list tools from %s: %w", clientName, err)
	}

	// Register tools in the registry
	if err := rp.registry.RegisterRemote(clientName, toolsResult.Tools); err != nil {
		return fmt.Errorf("failed to register tools from %s: %w", clientName, err)
	}

	return nil
}

// SyncAllTools synchronizes tools from all connected clients
func (rp *RemoteProxy) SyncAllTools(ctx context.Context) error {
	clients := rp.manager.ListClients()

	var errs []error
	for _, clientName := range clients {
		if err := rp.SyncTools(ctx, clientName); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", clientName, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to sync some clients: %v", errs)
	}

	return nil
}

// RefreshTools refreshes tools from a specific client
func (rp *RemoteProxy) RefreshTools(ctx context.Context, clientName string) error {
	// Refreshing is the same as syncing - it will update the registry
	return rp.SyncTools(ctx, clientName)
}

// Execute executes a remote tool
func (rp *RemoteProxy) Execute(ctx context.Context, namespacedName string, arguments map[string]interface{}) (*protocol.ToolCallResult, error) {
	// Parse the namespaced name (format: "clientName.toolName")
	clientName, toolName, err := parseNamespacedName(namespacedName)
	if err != nil {
		return nil, err
	}

	// Get the client
	mcpClient := rp.manager.GetClient(clientName)
	if mcpClient == nil {
		return nil, fmt.Errorf("client %s not found", clientName)
	}

	// Check if client is connected
	if !mcpClient.IsConnected() {
		return nil, fmt.Errorf("client %s is not connected", clientName)
	}

	// Execute the tool on the remote client
	result, err := mcpClient.CallTool(ctx, toolName, arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tool %s on %s: %w", toolName, clientName, err)
	}

	return result, nil
}

// Helper function to parse namespaced name
func parseNamespacedName(namespacedName string) (clientName, toolName string, err error) {
	// Find the first dot
	dotIndex := -1
	for i, c := range namespacedName {
		if c == '.' {
			dotIndex = i
			break
		}
	}

	if dotIndex == -1 {
		return "", "", fmt.Errorf("invalid namespaced name: %s (expected format: client.tool)", namespacedName)
	}

	clientName = namespacedName[:dotIndex]
	toolName = namespacedName[dotIndex+1:]

	if clientName == "" || toolName == "" {
		return "", "", fmt.Errorf("invalid namespaced name: %s", namespacedName)
	}

	return clientName, toolName, nil
}
