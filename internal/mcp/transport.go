package mcp

import (
	"context"
	"fmt"
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
				fmt.Printf("Error accepting connection: %v\n", err)
				continue
			}

			// Handle connection in a goroutine
			go tm.handleTCPConnection(conn)
		}
	}
}

// handleTCPConnection handles a single TCP connection
func (tm *TransportManager) handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	// Set connection timeout
	if tm.config.Transport.Timeout > 0 {
		conn.SetDeadline(time.Now().Add(tm.config.Transport.Timeout))
	}

	// TODO: Implement TCP connection handling with MCP server
	// This would require implementing a custom transport for mcp-go
	fmt.Printf("Handling TCP connection from %s\n", conn.RemoteAddr())

	// For now, we'll just close the connection
	// In a full implementation, we would need to:
	// 1. Create a custom transport that uses the TCP connection
	// 2. Start the MCP server with this transport
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
