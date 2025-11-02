package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// TransportManager manages different transport types for MCP communication
type TransportManager struct {
	config *Config
	server *server.MCPServer
}

// NewTransportManager creates a new transport manager
func NewTransportManager(config *Config, mcpServer *server.MCPServer) *TransportManager {
	return &TransportManager{
		config: config,
		server: mcpServer,
	}
}

// StartTransport starts the configured transport type
func (tm *TransportManager) StartTransport(ctx context.Context) error {
	switch tm.config.Transport.Type {
	case "stdio":
		return tm.startStdioTransport(ctx)
	case "tcp":
		return tm.startTCPTransport(ctx)
	case "websocket":
		return tm.startWebSocketTransport(ctx)
	case "unix":
		return tm.startUnixTransport(ctx)
	default:
		return fmt.Errorf("unsupported transport type: %s", tm.config.Transport.Type)
	}
}

// startStdioTransport starts stdio transport (standard input/output)
func (tm *TransportManager) startStdioTransport(ctx context.Context) error {
	// Start stdio server - this will block until the connection is closed
	if err := server.ServeStdio(tm.server); err != nil {
		return fmt.Errorf("stdio transport failed: %w", err)
	}
	return nil
}

// startTCPTransport starts TCP transport
func (tm *TransportManager) startTCPTransport(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", tm.config.Transport.Host, tm.config.Transport.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start TCP listener on %s: %w", addr, err)
	}
	defer listener.Close()

	fmt.Printf("MCP server listening on TCP %s\n", addr)

	// Create a channel for shutdown signal
	shutdown := make(chan struct{})
	go func() {
		<-ctx.Done()
		listener.Close() // Close listener to unblock Accept()
		close(shutdown)
	}()

	// Handle incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if we're shutting down
			select {
			case <-shutdown:
				return ctx.Err()
			default:
				// Only log error if not shutting down
				if ctx.Err() == nil {
					fmt.Printf("Error accepting connection: %v\n", err)
				}
				continue
			}
		}

		// Handle connection in a goroutine
		go tm.handleTCPConnection(conn)
	}
}

// handleTCPConnection handles a single TCP connection
func (tm *TransportManager) handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	// Set connection timeout (initial timeout for the connection)
	// We'll update it for each read/write operation
	if tm.config.Transport.Timeout > 0 {
		conn.SetDeadline(time.Now().Add(tm.config.Transport.Timeout))
	}

	fmt.Printf("Handling TCP connection from %s\n", conn.RemoteAddr())

	// Serve MCP protocol over this TCP connection
	// We use the connection as both reader and writer for JSON-RPC messages
	if err := tm.serveMCPConn(conn, conn); err != nil && err != io.EOF {
		fmt.Printf("Error serving TCP connection: %v\n", err)
	}
}

// serveMCPConn serves MCP protocol over a connection (implements JSON-RPC 2.0 over streams)
func (tm *TransportManager) serveMCPConn(reader io.Reader, writer io.Writer) error {
	// Create buffered reader and writer for JSON streaming
	bufReader := bufio.NewReader(reader)
	bufWriter := bufio.NewWriter(writer)

	// Create JSON decoder and encoder
	decoder := json.NewDecoder(bufReader)
	encoder := json.NewEncoder(bufWriter)

	// Process messages in a loop
	for {
		// Read JSON-RPC request
		var request map[string]interface{}
		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF {
				return nil // Normal connection close
			}
			return fmt.Errorf("failed to decode request: %w", err)
		}

		// Process the request through MCP server
		// Note: This is a simplified implementation
		// The actual mcp-go library handles this internally
		// For now, we'll create a basic response
		response := tm.processRequest(request)

		// Write response (skip if nil, which means it's a notification with no response)
		if response != nil {
			if err := encoder.Encode(response); err != nil {
				return fmt.Errorf("failed to encode response: %w", err)
			}

			// Flush the writer to ensure response is sent
			if err := bufWriter.Flush(); err != nil {
				return fmt.Errorf("failed to flush response: %w", err)
			}
		}
	}
}

// processRequest processes a JSON-RPC request and returns a response
func (tm *TransportManager) processRequest(request map[string]interface{}) map[string]interface{} {
	// Extract request fields
	id := request["id"]
	method, _ := request["method"].(string)

	// Handle different methods
	switch method {
	case "initialize":
		// Return initialize response
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]string{
					"name":    tm.config.Name,
					"version": tm.config.Version,
				},
			},
		}
	case "initialized":
		// No response needed for initialized notification
		return nil
	default:
		// Return error for unknown method
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
			},
		}
	}
}

// startWebSocketTransport starts WebSocket transport
func (tm *TransportManager) startWebSocketTransport(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", tm.config.Transport.Host, tm.config.Transport.Port)

	mux := http.NewServeMux()

	// Handle WebSocket connections
	mux.HandleFunc(tm.config.Transport.Path, tm.handleWebSocketConnection)

	// Handle health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"rocksdb-mcp-server"}`))
	})

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Printf("MCP server listening on WebSocket %s%s\n", addr, tm.config.Transport.Path)

	// Start server in a goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("WebSocket server error: %v\n", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return httpServer.Shutdown(shutdownCtx)
}

// handleWebSocketConnection handles WebSocket connections
func (tm *TransportManager) handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement WebSocket connection handling
	// This would require implementing WebSocket support for mcp-go
	fmt.Printf("WebSocket connection from %s\n", r.RemoteAddr)

	// For now, return a 501 Not Implemented
	http.Error(w, "WebSocket transport not yet implemented", http.StatusNotImplemented)
}

// startUnixTransport starts Unix socket transport
func (tm *TransportManager) startUnixTransport(ctx context.Context) error {
	socketPath := tm.config.Transport.SocketPath

	// Remove existing socket file if it exists
	if err := os.RemoveAll(socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket file: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to start Unix socket listener on %s: %w", socketPath, err)
	}
	defer listener.Close()
	defer os.RemoveAll(socketPath) // Clean up socket file

	fmt.Printf("MCP server listening on Unix socket %s\n", socketPath)

	// Handle incoming connections
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := listener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				fmt.Printf("Error accepting Unix socket connection: %v\n", err)
				continue
			}

			// Handle connection in a goroutine
			go tm.handleUnixConnection(conn)
		}
	}
}

// handleUnixConnection handles a single Unix socket connection
func (tm *TransportManager) handleUnixConnection(conn net.Conn) {
	defer conn.Close()

	// Set connection timeout
	if tm.config.Transport.Timeout > 0 {
		conn.SetDeadline(time.Now().Add(tm.config.Transport.Timeout))
	}

	// TODO: Implement Unix socket connection handling with MCP server
	fmt.Printf("Handling Unix socket connection\n")

	// For now, we'll just close the connection
	// In a full implementation, we would need to:
	// 1. Create a custom transport that uses the Unix socket connection
	// 2. Start the MCP server with this transport
}

// GetTransportInfo returns information about the current transport configuration
func (tm *TransportManager) GetTransportInfo() map[string]interface{} {
	info := map[string]interface{}{
		"type":    tm.config.Transport.Type,
		"timeout": tm.config.Transport.Timeout.String(),
	}

	switch tm.config.Transport.Type {
	case "tcp", "websocket":
		info["host"] = tm.config.Transport.Host
		info["port"] = tm.config.Transport.Port
		if tm.config.Transport.Type == "websocket" {
			info["path"] = tm.config.Transport.Path
		}
	case "unix":
		info["socket_path"] = tm.config.Transport.SocketPath
	}

	return info
}
