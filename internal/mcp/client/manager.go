package client

import (
	"context"
	"fmt"
	"sync"

	"rocksdb-cli/internal/config"
	"rocksdb-cli/internal/mcp/protocol"
)

// Manager manages multiple MCP clients
type Manager struct {
	mu      sync.RWMutex
	clients map[string]Client
	config  *config.Config
}

// ClientStatus represents the status of a client
type ClientStatus struct {
	Name        string
	State       string // "connected", "disconnected", "error"
	Connected   bool
	Initialized bool
	ServerInfo  *protocol.ServerInfo
	Error       string
}

// NewManager creates a new client manager
func NewManager(cfg *config.Config) *Manager {
	m := &Manager{
		clients: make(map[string]Client),
		config:  cfg,
	}

	// Load clients from configuration
	m.loadClients()

	return m
}

// loadClients loads all enabled clients from configuration
func (m *Manager) loadClients() {
	if m.config == nil || m.config.MCPClients == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for name, clientCfg := range m.config.MCPClients {
		// Skip disabled clients
		if !clientCfg.Enabled {
			continue
		}

		// Create client based on transport type
		var client Client
		switch clientCfg.Transport {
		case "stdio":
			client = NewStdioClient(name, clientCfg)
		case "tcp":
			client = NewTCPClient(name, clientCfg)
		default:
			// Skip unsupported transport types
			continue
		}

		m.clients[name] = client
	}
}

// ListClients returns a list of all client names
func (m *Manager) ListClients() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// GetClient returns a client by name
func (m *Manager) GetClient(name string) Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.clients[name]
}

// StartClient starts a specific client
func (m *Manager) StartClient(ctx context.Context, name string) error {
	m.mu.RLock()
	client, ok := m.clients[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("client %s not found", name)
	}

	// Check if already connected
	if client.IsConnected() {
		return fmt.Errorf("client %s is already connected", name)
	}

	// Connect to the client
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect client %s: %w", name, err)
	}

	// Initialize the client
	if _, err := client.Initialize(ctx); err != nil {
		// Disconnect on initialization failure
		client.Disconnect(ctx)
		return fmt.Errorf("failed to initialize client %s: %w", name, err)
	}

	return nil
}

// StopClient stops a specific client
func (m *Manager) StopClient(ctx context.Context, name string) error {
	m.mu.RLock()
	client, ok := m.clients[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("client %s not found", name)
	}

	// Disconnect is idempotent, so it's safe to call multiple times
	return client.Disconnect(ctx)
}

// StartAll starts all clients
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	clients := make(map[string]Client, len(m.clients))
	for name, client := range m.clients {
		clients[name] = client
	}
	m.mu.RUnlock()

	// Start all clients concurrently
	errCh := make(chan error, len(clients))
	var wg sync.WaitGroup

	for name := range clients {
		wg.Add(1)
		go func(clientName string) {
			defer wg.Done()
			if err := m.StartClient(ctx, clientName); err != nil {
				errCh <- fmt.Errorf("failed to start %s: %w", clientName, err)
			}
		}(name)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to start some clients: %v", errs)
	}

	return nil
}

// StopAll stops all clients
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	clients := make(map[string]Client, len(m.clients))
	for name, client := range m.clients {
		clients[name] = client
	}
	m.mu.RUnlock()

	// Stop all clients concurrently
	errCh := make(chan error, len(clients))
	var wg sync.WaitGroup

	for name := range clients {
		wg.Add(1)
		go func(clientName string) {
			defer wg.Done()
			if err := m.StopClient(ctx, clientName); err != nil {
				errCh <- fmt.Errorf("failed to stop %s: %w", clientName, err)
			}
		}(name)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to stop some clients: %v", errs)
	}

	return nil
}

// GetClientStatus returns the status of a specific client
func (m *Manager) GetClientStatus(name string) ClientStatus {
	m.mu.RLock()
	client, ok := m.clients[name]
	m.mu.RUnlock()

	status := ClientStatus{
		Name:  name,
		State: "unknown",
	}

	if !ok {
		status.State = "not_found"
		status.Error = "client not found"
		return status
	}

	status.Connected = client.IsConnected()
	status.Initialized = client.IsInitialized()

	if status.Connected {
		status.State = "connected"
		// Get server info if available
		if bc, ok := client.(*StdioClient); ok {
			status.ServerInfo = bc.GetServerInfo()
		} else if tc, ok := client.(*TCPClient); ok {
			status.ServerInfo = tc.GetServerInfo()
		}
	} else {
		status.State = "disconnected"
	}

	return status
}

// GetAllStatus returns the status of all clients
func (m *Manager) GetAllStatus() map[string]ClientStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]ClientStatus, len(m.clients))
	for name := range m.clients {
		// Temporarily unlock to call GetClientStatus
		m.mu.RUnlock()
		statuses[name] = m.GetClientStatus(name)
		m.mu.RLock()
	}

	return statuses
}

// HealthCheck performs a health check on a specific client
func (m *Manager) HealthCheck(ctx context.Context, name string) bool {
	m.mu.RLock()
	client, ok := m.clients[name]
	m.mu.RUnlock()

	if !ok || !client.IsConnected() {
		return false
	}

	// Try to ping the client
	if err := client.Ping(ctx); err != nil {
		return false
	}

	return true
}

// HealthCheckAll performs health checks on all clients
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]bool {
	m.mu.RLock()
	clients := make(map[string]Client, len(m.clients))
	for name, client := range m.clients {
		clients[name] = client
	}
	m.mu.RUnlock()

	results := make(map[string]bool, len(clients))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name := range clients {
		wg.Add(1)
		go func(clientName string) {
			defer wg.Done()
			healthy := m.HealthCheck(ctx, clientName)
			mu.Lock()
			results[clientName] = healthy
			mu.Unlock()
		}(name)
	}

	wg.Wait()
	return results
}

// Shutdown performs a graceful shutdown of all clients
func (m *Manager) Shutdown(ctx context.Context) error {
	return m.StopAll(ctx)
}
