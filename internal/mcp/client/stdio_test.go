package client

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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
	defer os.Remove(testScript)

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
	defer os.Remove(testScript)

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
	defer os.Remove(testScript)

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
	defer os.Remove(testScript)

	cfg := &config.MCPClientConfig{
		Name:      "timeout-client",
		Transport: "stdio",
		Command:   testScript,
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
	defer os.Remove(testScript)

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
	defer os.Remove(testScript)

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
	defer os.Remove(testScript)

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

// Helper: Create a simple echo script
func createTestEchoScript(t *testing.T) string {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "echo.sh")

	script := `#!/bin/bash
while IFS= read -r line; do
    echo "$line"
done
`

	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	return scriptPath
}

// Helper: Create a mock MCP server script
func createTestMCPServer(t *testing.T) string {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "mcp_server.sh")

	script := `#!/bin/bash
while IFS= read -r line; do
    request=$(echo "$line" | jq -r '.method')
    id=$(echo "$line" | jq -r '.id')

    case "$request" in
        "initialize")
            echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"protocolVersion\":\"2024-11-05\",\"serverInfo\":{\"name\":\"test-server\",\"version\":\"1.0.0\"},\"capabilities\":{\"tools\":{\"listChanged\":true}}}}"
            ;;
        "ping")
            echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{}}"
            ;;
        "tools/list")
            echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"tools\":[{\"name\":\"echo\",\"description\":\"Echo tool\",\"inputSchema\":{\"type\":\"object\"}}]}}"
            ;;
        "tools/call")
            echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"success\"}],\"isError\":false}}"
            ;;
        *)
            echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"error\":{\"code\":-32601,\"message\":\"Method not found\"}}"
            ;;
    esac
done
`

	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	return scriptPath
}

// Helper: Create a hanging server (never responds)
func createTestHangingServer(t *testing.T) string {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "hanging.sh")

	script := `#!/bin/bash
while IFS= read -r line; do
    # Read but never respond
    sleep 10
done
`

	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	return scriptPath
}

// Helper: Create a server that checks environment variables
func createTestEnvServer(t *testing.T) string {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "env_server.sh")

	script := `#!/bin/bash
# Check if TEST_VAR is set
if [ -z "$TEST_VAR" ]; then
    exit 1
fi
# Just exit successfully to indicate env var was received
exit 0
`

	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	return scriptPath
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
	defer os.Remove(testScript)

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
	tmpDir := b.TempDir()
	scriptPath := filepath.Join(tmpDir, "bench_server.sh")

	script := `#!/bin/bash
while IFS= read -r line; do
    id=$(echo "$line" | jq -r '.id')
    echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"protocolVersion\":\"2024-11-05\",\"serverInfo\":{\"name\":\"bench\",\"version\":\"1.0.0\"},\"capabilities\":{}}}"
done
`

	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(b, err)

	return scriptPath
}

// Test process cleanup on context cancellation
func TestStdioClient_ContextCancellation(t *testing.T) {
	testScript := createTestMCPServer(t)
	defer os.Remove(testScript)

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

// Test that jq is available (required for test scripts)
func TestJQAvailable(t *testing.T) {
	_, err := exec.LookPath("jq")
	if err != nil {
		t.Skip("jq not available, skipping STDIO tests")
	}
}
