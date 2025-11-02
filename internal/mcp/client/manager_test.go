package client

import (
	"context"
	"testing"
	"time"

	"rocksdb-cli/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test client manager creation
func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{},
	}

	manager := NewManager(cfg)
	require.NotNil(t, manager)
	assert.Equal(t, 0, len(manager.ListClients()))
}

// Test loading clients from configuration
func TestManager_LoadClients(t *testing.T) {
	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"test-client": {
				Name:      "test-client",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      9999,
			},
			"disabled-client": {
				Name:      "disabled-client",
				Enabled:   false,
				Transport: "tcp",
				Host:      "localhost",
				Port:      9998,
			},
		},
	}

	manager := NewManager(cfg)

	t.Run("loads enabled clients", func(t *testing.T) {
		clients := manager.ListClients()
		assert.Equal(t, 1, len(clients))
		assert.Contains(t, clients, "test-client")
		assert.NotContains(t, clients, "disabled-client")
	})

	t.Run("gets client by name", func(t *testing.T) {
		client := manager.GetClient("test-client")
		require.NotNil(t, client)
		assert.Equal(t, "test-client", client.GetName())
	})

	t.Run("returns nil for non-existent client", func(t *testing.T) {
		client := manager.GetClient("non-existent")
		assert.Nil(t, client)
	})
}

// Test starting individual clients
func TestManager_StartClient(t *testing.T) {
	// Create mock TCP server for testing
	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"tcp-client": {
				Name:      "tcp-client",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port,
				Timeout:   5 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)
	ctx := context.Background()

	t.Run("start client successfully", func(t *testing.T) {
		err := manager.StartClient(ctx, "tcp-client")
		require.NoError(t, err)

		client := manager.GetClient("tcp-client")
		assert.True(t, client.IsConnected())
		assert.True(t, client.IsInitialized())
	})

	t.Run("start returns error for non-existent client", func(t *testing.T) {
		err := manager.StartClient(ctx, "non-existent")
		assert.Error(t, err)
	})

	t.Run("start returns error for already started client", func(t *testing.T) {
		err := manager.StartClient(ctx, "tcp-client")
		assert.Error(t, err)
	})

	// Cleanup
	manager.StopClient(ctx, "tcp-client")
}

// Test stopping individual clients
func TestManager_StopClient(t *testing.T) {
	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"tcp-client": {
				Name:      "tcp-client",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port,
				Timeout:   5 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)
	ctx := context.Background()

	// Start the client first
	err := manager.StartClient(ctx, "tcp-client")
	require.NoError(t, err)

	t.Run("stop client successfully", func(t *testing.T) {
		err := manager.StopClient(ctx, "tcp-client")
		require.NoError(t, err)

		client := manager.GetClient("tcp-client")
		assert.False(t, client.IsConnected())
	})

	t.Run("stop returns error for non-existent client", func(t *testing.T) {
		err := manager.StopClient(ctx, "non-existent")
		assert.Error(t, err)
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		err := manager.StopClient(ctx, "tcp-client")
		assert.NoError(t, err)
	})
}

// Test starting all clients
func TestManager_StartAll(t *testing.T) {
	server1, port1 := startMockMCPTCPServer(t)
	defer server1.Close()
	server2, port2 := startMockMCPTCPServer(t)
	defer server2.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"client-1": {
				Name:      "client-1",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port1,
				Timeout:   5 * time.Second,
			},
			"client-2": {
				Name:      "client-2",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port2,
				Timeout:   5 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)
	ctx := context.Background()

	t.Run("starts all clients", func(t *testing.T) {
		err := manager.StartAll(ctx)
		require.NoError(t, err)

		client1 := manager.GetClient("client-1")
		client2 := manager.GetClient("client-2")

		assert.True(t, client1.IsConnected())
		assert.True(t, client1.IsInitialized())
		assert.True(t, client2.IsConnected())
		assert.True(t, client2.IsInitialized())
	})

	// Cleanup
	manager.StopAll(ctx)
}

// Test stopping all clients
func TestManager_StopAll(t *testing.T) {
	server1, port1 := startMockMCPTCPServer(t)
	defer server1.Close()
	server2, port2 := startMockMCPTCPServer(t)
	defer server2.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"client-1": {
				Name:      "client-1",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port1,
				Timeout:   5 * time.Second,
			},
			"client-2": {
				Name:      "client-2",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port2,
				Timeout:   5 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)
	ctx := context.Background()

	// Start all clients first
	err := manager.StartAll(ctx)
	require.NoError(t, err)

	t.Run("stops all clients", func(t *testing.T) {
		err := manager.StopAll(ctx)
		require.NoError(t, err)

		client1 := manager.GetClient("client-1")
		client2 := manager.GetClient("client-2")

		assert.False(t, client1.IsConnected())
		assert.False(t, client2.IsConnected())
	})

	t.Run("stop all is idempotent", func(t *testing.T) {
		err := manager.StopAll(ctx)
		assert.NoError(t, err)
	})
}

// Test client status tracking
func TestManager_ClientStatus(t *testing.T) {
	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"tcp-client": {
				Name:      "tcp-client",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port,
				Timeout:   5 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)
	ctx := context.Background()

	t.Run("reports disconnected status initially", func(t *testing.T) {
		status := manager.GetClientStatus("tcp-client")
		assert.Equal(t, "disconnected", status.State)
		assert.False(t, status.Connected)
		assert.False(t, status.Initialized)
	})

	t.Run("reports connected status after start", func(t *testing.T) {
		err := manager.StartClient(ctx, "tcp-client")
		require.NoError(t, err)

		status := manager.GetClientStatus("tcp-client")
		assert.Equal(t, "connected", status.State)
		assert.True(t, status.Connected)
		assert.True(t, status.Initialized)
		assert.NotNil(t, status.ServerInfo)
	})

	t.Run("reports all clients status", func(t *testing.T) {
		allStatus := manager.GetAllStatus()
		assert.Equal(t, 1, len(allStatus))
		assert.Contains(t, allStatus, "tcp-client")
	})

	// Cleanup
	manager.StopClient(ctx, "tcp-client")
}

// Test client health check
func TestManager_HealthCheck(t *testing.T) {
	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"tcp-client": {
				Name:      "tcp-client",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port,
				Timeout:   5 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)
	ctx := context.Background()

	t.Run("health check fails for disconnected client", func(t *testing.T) {
		healthy := manager.HealthCheck(ctx, "tcp-client")
		assert.False(t, healthy)
	})

	t.Run("health check succeeds for connected client", func(t *testing.T) {
		err := manager.StartClient(ctx, "tcp-client")
		require.NoError(t, err)

		healthy := manager.HealthCheck(ctx, "tcp-client")
		assert.True(t, healthy)
	})

	t.Run("health check all clients", func(t *testing.T) {
		results := manager.HealthCheckAll(ctx)
		assert.Equal(t, 1, len(results))
		assert.True(t, results["tcp-client"])
	})

	// Cleanup
	manager.StopClient(ctx, "tcp-client")
}

// Test context cancellation
func TestManager_ContextCancellation(t *testing.T) {
	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"tcp-client": {
				Name:      "tcp-client",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port,
				Timeout:   30 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	err := manager.StartClient(ctx, "tcp-client")
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Stop should still work with new context
	err = manager.StopClient(context.Background(), "tcp-client")
	assert.NoError(t, err)
}

// Test concurrent operations
func TestManager_ConcurrentOperations(t *testing.T) {
	server1, port1 := startMockMCPTCPServer(t)
	defer server1.Close()
	server2, port2 := startMockMCPTCPServer(t)
	defer server2.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"client-1": {
				Name:      "client-1",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port1,
				Timeout:   5 * time.Second,
			},
			"client-2": {
				Name:      "client-2",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port2,
				Timeout:   5 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)
	ctx := context.Background()

	t.Run("concurrent start operations", func(t *testing.T) {
		errCh := make(chan error, 2)

		go func() {
			errCh <- manager.StartClient(ctx, "client-1")
		}()
		go func() {
			errCh <- manager.StartClient(ctx, "client-2")
		}()

		// Collect errors
		for i := 0; i < 2; i++ {
			err := <-errCh
			assert.NoError(t, err)
		}

		// Verify both started
		assert.True(t, manager.GetClient("client-1").IsConnected())
		assert.True(t, manager.GetClient("client-2").IsConnected())
	})

	t.Run("concurrent stop operations", func(t *testing.T) {
		errCh := make(chan error, 2)

		go func() {
			errCh <- manager.StopClient(ctx, "client-1")
		}()
		go func() {
			errCh <- manager.StopClient(ctx, "client-2")
		}()

		// Collect errors
		for i := 0; i < 2; i++ {
			err := <-errCh
			assert.NoError(t, err)
		}

		// Verify both stopped
		assert.False(t, manager.GetClient("client-1").IsConnected())
		assert.False(t, manager.GetClient("client-2").IsConnected())
	})
}

// Test creating clients with different transports
func TestManager_DifferentTransports(t *testing.T) {
	mockServerPath := getMockServerPath()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"stdio-client": {
				Name:      "stdio-client",
				Enabled:   true,
				Transport: "stdio",
				Command:   mockServerPath,
				Timeout:   5 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)

	t.Run("creates STDIO client", func(t *testing.T) {
		client := manager.GetClient("stdio-client")
		require.NotNil(t, client)
		assert.Equal(t, "stdio-client", client.GetName())

		// Type assertion to verify it's a StdioClient
		_, ok := client.(*StdioClient)
		assert.True(t, ok, "Client should be of type *StdioClient")
	})
}

// Test graceful shutdown
func TestManager_GracefulShutdown(t *testing.T) {
	server1, port1 := startMockMCPTCPServer(t)
	defer server1.Close()
	server2, port2 := startMockMCPTCPServer(t)
	defer server2.Close()

	cfg := &config.Config{
		MCPClients: config.MCPClientsConfig{
			"client-1": {
				Name:      "client-1",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port1,
				Timeout:   5 * time.Second,
			},
			"client-2": {
				Name:      "client-2",
				Enabled:   true,
				Transport: "tcp",
				Host:      "localhost",
				Port:      port2,
				Timeout:   5 * time.Second,
			},
		},
	}

	manager := NewManager(cfg)
	ctx := context.Background()

	// Start all clients
	err := manager.StartAll(ctx)
	require.NoError(t, err)

	t.Run("shutdown stops all clients gracefully", func(t *testing.T) {
		err := manager.Shutdown(ctx)
		require.NoError(t, err)

		// Verify all clients are stopped
		for _, name := range manager.ListClients() {
			client := manager.GetClient(name)
			assert.False(t, client.IsConnected())
		}
	})

	t.Run("shutdown is idempotent", func(t *testing.T) {
		err := manager.Shutdown(ctx)
		assert.NoError(t, err)
	})
}
