package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"rocksdb-cli/internal/mcp/protocol"
)

// ToolHandler is a function that executes a local tool
type ToolHandler func(ctx context.Context, arguments map[string]interface{}) (*protocol.ToolCallResult, error)

// Registry manages both local and remote tools
type Registry struct {
	mu            sync.RWMutex
	localTools    map[string]*LocalTool    // Namespaced name -> tool
	remoteTools   map[string][]protocol.Tool // Client name -> tools
}

// LocalTool represents a locally registered tool
type LocalTool struct {
	Tool    protocol.Tool
	Handler ToolHandler
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		localTools:  make(map[string]*LocalTool),
		remoteTools: make(map[string][]protocol.Tool),
	}
}

// RegisterLocal registers a local tool with its handler
func (r *Registry) RegisterLocal(tool protocol.Tool, handler ToolHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create namespaced name
	namespacedName := "local." + tool.Name

	// Check if already registered
	if _, exists := r.localTools[namespacedName]; exists {
		return fmt.Errorf("tool %s already registered", namespacedName)
	}

	// Store with namespaced name
	toolCopy := tool
	toolCopy.Name = namespacedName

	r.localTools[namespacedName] = &LocalTool{
		Tool:    toolCopy,
		Handler: handler,
	}

	return nil
}

// RegisterRemote registers tools from a remote MCP client
func (r *Registry) RegisterRemote(clientName string, tools []protocol.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create namespaced tools
	namespacedTools := make([]protocol.Tool, len(tools))
	for i, tool := range tools {
		toolCopy := tool
		toolCopy.Name = clientName + "." + tool.Name
		namespacedTools[i] = toolCopy
	}

	// Replace existing tools for this client
	r.remoteTools[clientName] = namespacedTools

	return nil
}

// UnregisterRemote removes all tools from a remote client
func (r *Registry) UnregisterRemote(clientName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.remoteTools[clientName]; !exists {
		return fmt.Errorf("client %s not found", clientName)
	}

	delete(r.remoteTools, clientName)
	return nil
}

// ListTools returns all tools, optionally filtered by namespace
func (r *Registry) ListTools(namespace string) []protocol.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []protocol.Tool

	// Add local tools
	if namespace == "" || namespace == "local" {
		for _, localTool := range r.localTools {
			tools = append(tools, localTool.Tool)
		}
	}

	// Add remote tools
	if namespace == "" {
		// Add all remote tools
		for _, clientTools := range r.remoteTools {
			tools = append(tools, clientTools...)
		}
	} else if namespace != "local" {
		// Add tools from specific client
		if clientTools, exists := r.remoteTools[namespace]; exists {
			tools = append(tools, clientTools...)
		}
	}

	return tools
}

// GetTool returns a tool by its namespaced name
func (r *Registry) GetTool(namespacedName string) *protocol.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check local tools
	if localTool, exists := r.localTools[namespacedName]; exists {
		return &localTool.Tool
	}

	// Check remote tools
	parts := strings.SplitN(namespacedName, ".", 2)
	if len(parts) == 2 {
		clientName := parts[0]
		if clientTools, exists := r.remoteTools[clientName]; exists {
			for i := range clientTools {
				if clientTools[i].Name == namespacedName {
					return &clientTools[i]
				}
			}
		}
	}

	return nil
}

// Execute executes a tool by its namespaced name
func (r *Registry) Execute(ctx context.Context, namespacedName string, arguments map[string]interface{}) (*protocol.ToolCallResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if it's a local tool
	if localTool, exists := r.localTools[namespacedName]; exists {
		return localTool.Handler(ctx, arguments)
	}

	// For remote tools, execution is handled by the client manager
	// This method only handles local tools
	// Remote tool execution will be delegated through another mechanism

	return nil, fmt.Errorf("tool %s not found or not executable", namespacedName)
}

// Count returns the total number of registered tools
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := len(r.localTools)
	for _, clientTools := range r.remoteTools {
		count += len(clientTools)
	}

	return count
}

// ListNamespaces returns all available namespaces
func (r *Registry) ListNamespaces() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	namespaces := make([]string, 0)

	// Add "local" if we have local tools
	if len(r.localTools) > 0 {
		namespaces = append(namespaces, "local")
	}

	// Add client names
	for clientName := range r.remoteTools {
		namespaces = append(namespaces, clientName)
	}

	return namespaces
}
