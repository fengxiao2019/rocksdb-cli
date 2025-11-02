package client

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"rocksdb-cli/internal/config"
	"rocksdb-cli/internal/mcp/protocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test TCP client creation
func TestNewTCPClient(t *testing.T) {
	cfg := &config.MCPClientConfig{
		Name:      "test-tcp",
		Transport: "tcp",
		Host:      "localhost",
		Port:      9999,
		Timeout:   30 * time.Second,
	}

	client := NewTCPClient("test-tcp", cfg)
	require.NotNil(t, client)
	assert.Equal(t, "test-tcp", client.GetName())
	assert.False(t, client.IsConnected())
}

// Test TCP client connection
func TestTCPClient_Connect(t *testing.T) {
	// Start a test TCP server
	server, port := startTestTCPServer(t)
	defer server.Close()

	cfg := &config.MCPClientConfig{
		Name:      "tcp-connect",
		Transport: "tcp",
		Host:      "localhost",
		Port:      port,
		Timeout:   5 * time.Second,
	}

	client := NewTCPClient("tcp-connect", cfg)
	ctx := context.Background()

	err := client.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, client.IsConnected())

	err = client.Disconnect(ctx)
	require.NoError(t, err)
	assert.False(t, client.IsConnected())
}

// Test TCP client double connect
func TestTCPClient_DoubleConnect(t *testing.T) {
	server, port := startTestTCPServer(t)
	defer server.Close()

	cfg := &config.MCPClientConfig{
		Name:      "tcp-double",
		Transport: "tcp",
		Host:      "localhost",
		Port:      port,
		Timeout:   5 * time.Second,
	}

	client := NewTCPClient("tcp-double", cfg)
	ctx := context.Background()

	err := client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	// Try connecting again
	err = client.Connect(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already connected")
}

// Test TCP client connection failure
func TestTCPClient_ConnectionFailure(t *testing.T) {
	cfg := &config.MCPClientConfig{
		Name:      "tcp-fail",
		Transport: "tcp",
		Host:      "localhost",
		Port:      1, // Port 1 should not be accessible
		Timeout:   2 * time.Second,
	}

	client := NewTCPClient("tcp-fail", cfg)
	ctx := context.Background()

	err := client.Connect(ctx)
	assert.Error(t, err)
	assert.False(t, client.IsConnected())
}

// Test TCP client full lifecycle
func TestTCPClient_FullLifecycle(t *testing.T) {
	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.MCPClientConfig{
		Name:      "tcp-lifecycle",
		Transport: "tcp",
		Host:      "localhost",
		Port:      port,
		Timeout:   5 * time.Second,
	}

	client := NewTCPClient("tcp-lifecycle", cfg)
	ctx := context.Background()

	// Connect
	err := client.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, client.IsConnected())

	// Initialize
	result, err := client.Initialize(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, protocol.MCPProtocolVersion, result.ProtocolVersion)

	// Ping
	err = client.Ping(ctx)
	require.NoError(t, err)

	// List tools
	tools, err := client.ListTools(ctx, "")
	require.NoError(t, err)
	assert.NotNil(t, tools)

	// Call tool
	toolResult, err := client.CallTool(ctx, "echo", map[string]interface{}{
		"message": "hello",
	})
	require.NoError(t, err)
	assert.NotNil(t, toolResult)
	assert.False(t, toolResult.IsError)

	// Disconnect
	err = client.Disconnect(ctx)
	require.NoError(t, err)
	assert.False(t, client.IsConnected())
}

// Test TCP client request timeout
func TestTCPClient_RequestTimeout(t *testing.T) {
	// Start a server that never responds
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Accept connections but don't respond
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Just hold the connection, never respond
			defer conn.Close()
		}
	}()

	cfg := &config.MCPClientConfig{
		Name:      "tcp-timeout",
		Transport: "tcp",
		Host:      "localhost",
		Port:      port,
		Timeout:   1 * time.Second,
	}

	client := NewTCPClient("tcp-timeout", cfg)
	ctx := context.Background()

	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	// Try to initialize with timeout
	_, err = client.Initialize(ctx)
	assert.Error(t, err)
}

// Test TCP client connection loss
func TestTCPClient_ConnectionLoss(t *testing.T) {
	// Create a custom server that we can close connections on
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port

	// Track connections so we can close them
	var activeConn net.Conn
	var connMu sync.Mutex

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			connMu.Lock()
			activeConn = conn
			connMu.Unlock()
			go handleMCPConnection(conn)
		}
	}()
	defer listener.Close()

	cfg := &config.MCPClientConfig{
		Name:      "tcp-loss",
		Transport: "tcp",
		Host:      "localhost",
		Port:      port,
		Timeout:   1 * time.Second,
	}

	client := NewTCPClient("tcp-loss", cfg)
	ctx := context.Background()

	err = client.Connect(ctx)
	require.NoError(t, err)

	// Initialize to ensure connection is established
	_, err = client.Initialize(ctx)
	require.NoError(t, err)

	// Close the active connection from server side
	connMu.Lock()
	if activeConn != nil {
		activeConn.Close()
	}
	connMu.Unlock()

	// Wait for read goroutine to detect the closure
	time.Sleep(100 * time.Millisecond)

	// Try to send request - should fail
	err = client.Ping(ctx)
	assert.Error(t, err)
}

// Test TCP client concurrent requests
func TestTCPClient_ConcurrentRequests(t *testing.T) {
	server, port := startMockMCPTCPServer(t)
	defer server.Close()

	cfg := &config.MCPClientConfig{
		Name:      "tcp-concurrent",
		Transport: "tcp",
		Host:      "localhost",
		Port:      port,
		Timeout:   10 * time.Second,
	}

	client := NewTCPClient("tcp-concurrent", cfg)
	ctx := context.Background()

	err := client.Connect(ctx)
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	_, err = client.Initialize(ctx)
	require.NoError(t, err)

	// Send multiple concurrent requests
	const numRequests = 5
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			_, err := client.CallTool(ctx, "echo", map[string]interface{}{
				"id": id,
			})
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

// Helper: Start a simple TCP echo server
func startTestTCPServer(t *testing.T) (net.Listener, int) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				// Just keep connection alive
				buf := make([]byte, 1024)
				for {
					_, err := c.Read(buf)
					if err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	return listener, port
}

// Helper: Start a mock MCP TCP server
func startMockMCPTCPServer(t *testing.T) (net.Listener, int) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

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

// Helper: Handle MCP connection (reuse STDIO mock server logic)
func handleMCPConnection(conn net.Conn) {
	defer conn.Close()

	// Use the same mock server implementation as STDIO
	// For simplicity, we'll create a simple JSON-RPC handler here
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

// Benchmark TCP client operations
func BenchmarkTCPClient_Initialize(b *testing.B) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		b.Fatal(err)
	}
	defer listener.Close()

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

	cfg := &config.MCPClientConfig{
		Name:      "bench-tcp",
		Transport: "tcp",
		Host:      "localhost",
		Port:      port,
		Timeout:   30 * time.Second,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := NewTCPClient("bench-tcp", cfg)
		_ = client.Connect(ctx)
		_, _ = client.Initialize(ctx)
		_ = client.Disconnect(ctx)
	}
}
