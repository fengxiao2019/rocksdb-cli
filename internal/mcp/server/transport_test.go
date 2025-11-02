package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTCPTransport_BasicConnection tests basic TCP connection
func TestTCPTransport_BasicConnection(t *testing.T) {
	// Create a minimal MCP server for testing
	mcpServer := server.NewMCPServer(
		"test-server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	config := &Config{
		Transport: TransportConfig{
			Type: "tcp",
			Host: "localhost",
			Port: 0, // Use random port
		},
	}

	tm := NewTransportManager(config, mcpServer)

	// Start transport in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get a free port
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	config.Transport.Port = port

	errChan := make(chan error, 1)
	go func() {
		errChan <- tm.StartTransport(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Connect to the server
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	require.NoError(t, err)
	defer conn.Close()

	// Connection should be established
	assert.NotNil(t, conn)
}

// TestTCPTransport_MCPProtocol tests MCP protocol over TCP
func TestTCPTransport_MCPProtocol(t *testing.T) {
	mcpServer := server.NewMCPServer(
		"test-server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	config := &Config{
		Transport: TransportConfig{
			Type:    "tcp",
			Host:    "localhost",
			Port:    0,
			Timeout: 10 * time.Second,
		},
	}

	// Get a free port
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	config.Transport.Port = port
	tm := NewTransportManager(config, mcpServer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start server
	go func() {
		tm.StartTransport(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	// Connect and send MCP initialize request
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	require.NoError(t, err)
	defer conn.Close()

	// Send JSON-RPC initialize request
	initRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]string{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	// Write request
	writer := bufio.NewWriter(conn)
	encoder := json.NewEncoder(writer)
	err = encoder.Encode(initRequest)
	require.NoError(t, err)
	err = writer.Flush()
	require.NoError(t, err)

	// Read response
	reader := bufio.NewReader(conn)
	decoder := json.NewDecoder(reader)
	var response map[string]interface{}
	err = decoder.Decode(&response)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, "2.0", response["jsonrpc"])
	assert.Equal(t, float64(1), response["id"])
	assert.NotNil(t, response["result"])
}

// TestTCPTransport_MultipleConnections tests handling multiple concurrent connections
func TestTCPTransport_MultipleConnections(t *testing.T) {
	mcpServer := server.NewMCPServer(
		"test-server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	config := &Config{
		Transport: TransportConfig{
			Type: "tcp",
			Host: "localhost",
			Port: 0,
		},
	}

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	config.Transport.Port = port
	tm := NewTransportManager(config, mcpServer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		tm.StartTransport(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	// Create multiple connections
	numConnections := 5
	connections := make([]net.Conn, numConnections)

	for i := 0; i < numConnections; i++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		require.NoError(t, err)
		defer conn.Close()
		connections[i] = conn
	}

	// All connections should be established
	for i, conn := range connections {
		assert.NotNil(t, conn, "Connection %d should be established", i)
	}
}

// TestTCPTransport_ConnectionTimeout tests connection timeout handling
func TestTCPTransport_ConnectionTimeout(t *testing.T) {
	mcpServer := server.NewMCPServer(
		"test-server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	config := &Config{
		Transport: TransportConfig{
			Type:    "tcp",
			Host:    "localhost",
			Port:    0,
			Timeout: 500 * time.Millisecond,
		},
	}

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	config.Transport.Port = port
	tm := NewTransportManager(config, mcpServer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		tm.StartTransport(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	// Connect but don't send anything
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	require.NoError(t, err)
	defer conn.Close()

	// Read should timeout
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, err = conn.Read(buf)

	// Should get timeout or connection close
	assert.Error(t, err)
}

// TestTCPTransport_Shutdown tests graceful shutdown
func TestTCPTransport_Shutdown(t *testing.T) {
	mcpServer := server.NewMCPServer(
		"test-server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	config := &Config{
		Transport: TransportConfig{
			Type: "tcp",
			Host: "localhost",
			Port: 0,
		},
	}

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	config.Transport.Port = port
	tm := NewTransportManager(config, mcpServer)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		tm.StartTransport(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	// Connect
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	require.NoError(t, err)
	defer conn.Close()

	// Cancel context to trigger shutdown
	cancel()

	// Wait a bit for shutdown
	time.Sleep(100 * time.Millisecond)

	// New connections should fail after shutdown
	_, err = net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	assert.Error(t, err, "Should not be able to connect after shutdown")
}

// TestTCPTransport_InvalidPort tests error handling for invalid ports
func TestTCPTransport_InvalidPort(t *testing.T) {
	mcpServer := server.NewMCPServer(
		"test-server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Try to use port 1 (usually requires root)
	config := &Config{
		Transport: TransportConfig{
			Type: "tcp",
			Host: "localhost",
			Port: 1,
		},
	}

	tm := NewTransportManager(config, mcpServer)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := tm.StartTransport(ctx)
	assert.Error(t, err, "Should fail to bind to privileged port")
}
