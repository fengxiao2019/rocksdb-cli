package tools

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"rocksdb-cli/internal/config"
	"rocksdb-cli/internal/mcp/client"
	"rocksdb-cli/internal/mcp/protocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test creating a new remote proxy
func TestNewRemoteProxy(t *testing.T) {
	registry := NewRegistry()
	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{},
	}
	manager := client.NewManager(cfg)

	proxy := NewRemoteProxy(registry, manager)
	require.NotNil(t, proxy)
}

// Test syncing tools from remote clients
func TestRemoteProxy_SyncTools(t *testing.T) {
	registry := NewRegistry()

	// Create mock TCP server
	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"test-server": {
				Name:      "test-server",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port,
				Timeout:   5 * time.Second,
			},
		},
	}
	manager := client.NewManager(cfg)
	proxy := NewRemoteProxy(registry, manager)

	ctx := context.Background()

	t.Run("syncs tools from connected client", func(t *testing.T) {
		// Start the client
		err := manager.StartClient(ctx, "test-server")
		require.NoError(t, err)
		defer manager.StopClient(ctx, "test-server")

		// Sync tools
		err = proxy.SyncTools(ctx, "test-server")
		require.NoError(t, err)

		// Verify tools are registered
		tools := registry.ListTools("test-server")
		assert.Greater(t, len(tools), 0)
	})

	t.Run("returns error for non-existent client", func(t *testing.T) {
		err := proxy.SyncTools(ctx, "non-existent")
		assert.Error(t, err)
	})
}

// Test syncing all tools from all clients
func TestRemoteProxy_SyncAllTools(t *testing.T) {
	registry := NewRegistry()

	server1, port1 := startMockMCPTCPServer(t)
	defer server1.Close()
	server2, port2 := startMockMCPTCPServer(t)
	defer server2.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"server-1": {
				Name:      "server-1",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port1,
				Timeout:   5 * time.Second,
			},
			"server-2": {
				Name:      "server-2",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port2,
				Timeout:   5 * time.Second,
			},
		},
	}
	manager := client.NewManager(cfg)
	proxy := NewRemoteProxy(registry, manager)

	ctx := context.Background()

	// Start all clients
	err := manager.StartAll(ctx)
	require.NoError(t, err)
	defer manager.StopAll(ctx)

	t.Run("syncs tools from all clients", func(t *testing.T) {
		err := proxy.SyncAllTools(ctx)
		require.NoError(t, err)

		// Verify tools from both servers
		server1Tools := registry.ListTools("server-1")
		server2Tools := registry.ListTools("server-2")

		assert.Greater(t, len(server1Tools), 0)
		assert.Greater(t, len(server2Tools), 0)
	})
}

// Test executing remote tools
func TestRemoteProxy_ExecuteRemoteTool(t *testing.T) {
	registry := NewRegistry()

	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"test-server": {
				Name:      "test-server",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port,
				Timeout:   5 * time.Second,
			},
		},
	}
	manager := client.NewManager(cfg)
	proxy := NewRemoteProxy(registry, manager)

	ctx := context.Background()

	// Start client and sync tools
	err := manager.StartClient(ctx, "test-server")
	require.NoError(t, err)
	defer manager.StopClient(ctx, "test-server")

	err = proxy.SyncTools(ctx, "test-server")
	require.NoError(t, err)

	t.Run("executes remote tool successfully", func(t *testing.T) {
		result, err := proxy.Execute(ctx, "test-server.echo", map[string]interface{}{
			"message": "hello",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Greater(t, len(result.Content), 0)
	})

	t.Run("returns error for invalid namespaced name", func(t *testing.T) {
		_, err := proxy.Execute(ctx, "invalid-name", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid namespaced name")
	})

	t.Run("returns error for disconnected client", func(t *testing.T) {
		// Stop the client
		manager.StopClient(ctx, "test-server")

		_, err := proxy.Execute(ctx, "test-server.echo", nil)
		assert.Error(t, err)
	})
}

// Test enabling/disabling auto-sync
func TestRemoteProxy_AutoSync(t *testing.T) {
	registry := NewRegistry()
	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{},
	}
	manager := client.NewManager(cfg)
	proxy := NewRemoteProxy(registry, manager)

	t.Run("can enable auto-sync", func(t *testing.T) {
		proxy.EnableAutoSync()
		// Auto-sync is enabled (implementation placeholder)
	})

	t.Run("can disable auto-sync", func(t *testing.T) {
		proxy.DisableAutoSync()
		// Auto-sync is disabled (implementation placeholder)
	})
}

// Test tool refresh
func TestRemoteProxy_RefreshTools(t *testing.T) {
	registry := NewRegistry()

	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"test-server": {
				Name:      "test-server",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port,
				Timeout:   5 * time.Second,
			},
		},
	}
	manager := client.NewManager(cfg)
	proxy := NewRemoteProxy(registry, manager)

	ctx := context.Background()

	// Start client and sync tools
	err := manager.StartClient(ctx, "test-server")
	require.NoError(t, err)
	defer manager.StopClient(ctx, "test-server")

	err = proxy.SyncTools(ctx, "test-server")
	require.NoError(t, err)

	t.Run("refreshes tools from client", func(t *testing.T) {
		// Get initial count
		initialTools := registry.ListTools("test-server")
		initialCount := len(initialTools)

		// Refresh tools
		err := proxy.RefreshTools(ctx, "test-server")
		require.NoError(t, err)

		// Verify tools are still available
		refreshedTools := registry.ListTools("test-server")
		assert.Equal(t, initialCount, len(refreshedTools))
	})
}

// Test concurrent tool execution
func TestRemoteProxy_ConcurrentExecution(t *testing.T) {
	registry := NewRegistry()

	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"test-server": {
				Name:      "test-server",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port,
				Timeout:   10 * time.Second,
			},
		},
	}
	manager := client.NewManager(cfg)
	proxy := NewRemoteProxy(registry, manager)

	ctx := context.Background()

	// Start client and sync tools
	err := manager.StartClient(ctx, "test-server")
	require.NoError(t, err)
	defer manager.StopClient(ctx, "test-server")

	err = proxy.SyncTools(ctx, "test-server")
	require.NoError(t, err)

	t.Run("executes tools concurrently", func(t *testing.T) {
		const numConcurrent = 5
		errCh := make(chan error, numConcurrent)

		for i := 0; i < numConcurrent; i++ {
			go func(id int) {
				_, err := proxy.Execute(ctx, "test-server.echo", map[string]interface{}{
					"id": id,
				})
				errCh <- err
			}(i)
		}

		// Collect results
		for i := 0; i < numConcurrent; i++ {
			err := <-errCh
			assert.NoError(t, err)
		}
	})
}

// Helper function to start mock MCP TCP server
// Duplicated from client tests for convenience
func startMockMCPTCPServer(t testing.TB) (net.Listener, int) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleMCPConnection(conn)
		}
	}()

	return listener, port
}

// Helper: Handle MCP connection
func handleMCPConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var req protocol.JSONRPCRequest
		if err := decoder.Decode(&req); err != nil {
			return
		}

		var resp protocol.JSONRPCResponse
		resp.JSONRPC = protocol.JSONRPCVersion
		resp.ID = req.ID

		switch req.Method {
		case protocol.MethodInitialize:
			resp.Result = protocol.InitializeResult{
				ProtocolVersion: protocol.MCPProtocolVersion,
				ServerInfo: protocol.ServerInfo{
					Name:    "test-tcp-server",
					Version: "1.0.0",
				},
				Capabilities: protocol.Capabilities{
					Tools: &protocol.ToolsCapability{
						ListChanged: true,
					},
				},
			}

		case protocol.MethodPing:
			resp.Result = map[string]interface{}{}

		case protocol.MethodListTools:
			resp.Result = protocol.ListToolsResult{
				Tools: []protocol.Tool{
					{
						Name:        "echo",
						Description: "Echo tool",
						InputSchema: map[string]interface{}{
							"type": "object",
						},
					},
				},
			}

		case protocol.MethodCallTool:
			resp.Result = protocol.ToolCallResult{
				Content: []protocol.Content{
					{
						Type: "text",
						Text: "success",
					},
				},
				IsError: false,
			}

		default:
			resp.Error = &protocol.JSONRPCError{
				Code:    protocol.MethodNotFound,
				Message: "Method not found",
			}
		}

		if err := encoder.Encode(resp); err != nil {
			return
		}
	}
}
