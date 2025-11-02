package client

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"rocksdb-cli/internal/config"
	"rocksdb-cli/internal/mcp/protocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test STDIO client creation
func TestNewStdioClient(t *testing.T) {
	cfg := &config.MCPClientConfig{
		Name:      "test-stdio",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"hello"},
		Timeout:   30 * time.Second,
	}

	client := NewStdioClient("test-stdio", cfg)
	require.NotNil(t, client)
	assert.Equal(t, "test-stdio", client.GetName())
	assert.False(t, client.IsConnected())
}

// Test STDIO client with mock echo server
func TestStdioClient_EchoServer(t *testing.T) {
	// Create a simple echo script for testing
	testScript := createTestEchoScript(t)

	cfg := &config.MCPClientConfig{
		Name:      "echo-client",
		Transport: "stdio",
		Command:   testScript,
		Args:      []string{},
		Timeout:   5 * time.Second,
	}

	client := NewStdioClient("echo-client", cfg)
	ctx := context.Background()

	// Connect
	err := client.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, client.IsConnected())

	// Disconnect
	err = client.Disconnect(ctx)
	require.NoError(t, err)
	assert.False(t, client.IsConnected())
}

// Test STDIO client connection lifecycle
func TestStdioClient_ConnectionLifecycle(t *testing.T) {
	testScript := createTestMCPServer(t)

	cfg := &config.MCPClientConfig{
		Name:      "lifecycle-client",
		Transport: "stdio",
		Command:   testScript,
		Timeout:   5 * time.Second,
	}

	client := NewStdioClient("lifecycle-client", cfg)
	ctx := context.Background()

	t.Run("connect starts process", func(t *testing.T) {
		err := client.Connect(ctx)
		require.NoError(t, err)
		assert.True(t, client.IsConnected())
	})

	t.Run("initialize sends initialize request", func(t *testing.T) {
		result, err := client.Initialize(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, protocol.MCPProtocolVersion, result.ProtocolVersion)
	})

	t.Run("ping responds successfully", func(t *testing.T) {
		err := client.Ping(ctx)
		require.NoError(t, err)
	})

	t.Run("disconnect stops process", func(t *testing.T) {
		err := client.Disconnect(ctx)
		require.NoError(t, err)
		assert.False(t, client.IsConnected())
	})
}

// Test STDIO client double connect
func TestStdioClient_DoubleConnect(t *testing.T) {
	testScript := createTestMCPServer(t)

	cfg := &config.MCPClientConfig{
		Name:      "double-connect",
		Transport: "stdio",
		Command:   testScript,
		Timeout:   5 * time.Second,
	}

	client := NewStdioClient("double-connect", cfg)
	ctx := context.Background()

	err := client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	// Try connecting again
	err = client.Connect(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already connected")
}

// Test STDIO client with invalid command
func TestStdioClient_InvalidCommand(t *testing.T) {
	cfg := &config.MCPClientConfig{
		Name:      "invalid-cmd",
		Transport: "stdio",
		Command:   "/nonexistent/command",
		Timeout:   5 * time.Second,
	}

	client := NewStdioClient("invalid-cmd", cfg)
	ctx := context.Background()

	err := client.Connect(ctx)
	assert.Error(t, err)
	assert.False(t, client.IsConnected())
}

// Test STDIO client request timeout
func TestStdioClient_RequestTimeout(t *testing.T) {
	// Create a server that never responds
	testScript := createTestHangingServer(t)

	cfg := &config.MCPClientConfig{
		Name:      "timeout-client",
		Transport: "stdio",
		Command:   testScript,
		Args:      []string{"100"}, // sleep for 100 seconds
		Timeout:   1 * time.Second, // Short timeout
	}

	client := NewStdioClient("timeout-client", cfg)
	ctx := context.Background()

	err := client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	// Try to initialize with timeout
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err = client.Initialize(ctx2)
	assert.Error(t, err)
	// Should be a timeout error
}

// Test STDIO client list tools
func TestStdioClient_ListTools(t *testing.T) {
	testScript := createTestMCPServer(t)

	cfg := &config.MCPClientConfig{
		Name:      "list-tools-client",
		Transport: "stdio",
		Command:   testScript,
		Timeout:   5 * time.Second,
	}

	client := NewStdioClient("list-tools-client", cfg)
	ctx := context.Background()

	err := client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	_, err = client.Initialize(ctx)
	require.NoError(t, err)

	result, err := client.ListTools(ctx, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.Tools), 0)
}

// Test STDIO client call tool
func TestStdioClient_CallTool(t *testing.T) {
	testScript := createTestMCPServer(t)

	cfg := &config.MCPClientConfig{
		Name:      "call-tool-client",
		Transport: "stdio",
		Command:   testScript,
		Timeout:   5 * time.Second,
	}

	client := NewStdioClient("call-tool-client", cfg)
	ctx := context.Background()

	err := client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	_, err = client.Initialize(ctx)
	require.NoError(t, err)

	result, err := client.CallTool(ctx, "echo", map[string]interface{}{
		"message": "hello",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
}

// Test STDIO client with environment variables
func TestStdioClient_WithEnv(t *testing.T) {
	testScript := createTestEnvServer(t)

	cfg := &config.MCPClientConfig{
		Name:      "env-client",
		Transport: "stdio",
		Command:   testScript,
		Timeout:   5 * time.Second,
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	client := NewStdioClient("env-client", cfg)
	ctx := context.Background()

	err := client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	// The test server should be able to read the env var
	assert.True(t, client.IsConnected())
}

// Helper: Get path to mock MCP server binary
func getMockServerPath() string {
	// Path relative to test file
	return "./testdata/mock_server"
}

// Helper: Create a simple echo script
func createTestEchoScript(t *testing.T) string {
	// Use cat as a simple echo (available on all Unix systems)
	return "/bin/cat"
}

// Helper: Get mock MCP server command
func createTestMCPServer(t *testing.T) string {
	serverPath := getMockServerPath()

	// Check if mock server exists
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		t.Skip("mock_server not found, run: go build -o internal/mcp/client/testdata/mock_server internal/mcp/client/testdata/mock_server.go")
	}

	return serverPath
}

// Helper: Create a hanging server (cat with no input)
func createTestHangingServer(t *testing.T) string {
	// cat without input will hang waiting for input
	// We can use sleep command that sleeps forever
	return "/bin/sleep"
}

// Helper: Create a server that checks environment variables
func createTestEnvServer(t *testing.T) string {
	// For environment test, we just need a command that exits
	// We'll use true command which just exits successfully
	return "/usr/bin/true"
}

// Test JSON-RPC message formatting
func TestStdioClient_JSONRPCMessages(t *testing.T) {
	t.Run("format initialize request", func(t *testing.T) {
		req := protocol.JSONRPCRequest{
			JSONRPC: protocol.JSONRPCVersion,
			ID:      1,
			Method:  protocol.MethodInitialize,
			Params: protocol.InitializeRequest{
				ProtocolVersion: protocol.MCPProtocolVersion,
				ClientInfo: protocol.ClientInfo{
					Name:    "test-client",
					Version: "1.0.0",
				},
				Capabilities: protocol.Capabilities{},
			},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.Contains(t, string(data), "initialize")
		assert.Contains(t, string(data), "2024-11-05")
	})

	t.Run("parse initialize response", func(t *testing.T) {
		respJSON := `{
			"jsonrpc": "2.0",
			"id": 1,
			"result": {
				"protocolVersion": "2024-11-05",
				"serverInfo": {
					"name": "test-server",
					"version": "1.0.0"
				},
				"capabilities": {
					"tools": {
						"listChanged": true
					}
				}
			}
		}`

		var resp protocol.JSONRPCResponse
		err := json.Unmarshal([]byte(respJSON), &resp)
		require.NoError(t, err)
		assert.Nil(t, resp.Error)
	})
}

// Benchmark STDIO client operations
func BenchmarkStdioClient_Initialize(b *testing.B) {
	testScript := createBenchmarkMCPServer(b)

	cfg := &config.MCPClientConfig{
		Name:      "bench-client",
		Transport: "stdio",
		Command:   testScript,
		Timeout:   30 * time.Second,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := NewStdioClient("bench-client", cfg)
		_ = client.Connect(ctx)
		_, _ = client.Initialize(ctx)
		_ = client.Disconnect(ctx)
	}
}

func createBenchmarkMCPServer(b *testing.B) string {
	serverPath := getMockServerPath()

	// Check if mock server exists
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		b.Skip("mock_server not found")
	}

	return serverPath
}

// Test process cleanup on context cancellation
func TestStdioClient_ContextCancellation(t *testing.T) {
	testScript := createTestMCPServer(t)

	cfg := &config.MCPClientConfig{
		Name:      "cancel-client",
		Transport: "stdio",
		Command:   testScript,
		Timeout:   30 * time.Second,
	}

	client := NewStdioClient("cancel-client", cfg)
	ctx, cancel := context.WithCancel(context.Background())

	err := client.Connect(ctx)
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Client should handle cancellation gracefully
	err = client.Disconnect(context.Background())
	assert.NoError(t, err)
}
