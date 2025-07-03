package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/mcp"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Command line flags
	var (
		configPath = flag.String("config", "", "Path to configuration file")
		dbPath     = flag.String("db", "", "Path to RocksDB database")
		readOnly   = flag.Bool("readonly", false, "Open database in read-only mode")
		transport  = flag.String("transport", "stdio", "Transport type (stdio, tcp, websocket, unix)")
		host       = flag.String("host", "localhost", "Host for TCP/WebSocket transport")
		port       = flag.Int("port", 8080, "Port for TCP/WebSocket transport")
		socketPath = flag.String("socket", "/tmp/rocksdb-mcp.sock", "Unix socket path")
	)
	flag.Parse()

	// Load or create default configuration
	var config *mcp.Config
	var err error

	if *configPath != "" {
		config, err = mcp.LoadConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	} else {
		config = mcp.DefaultConfig()
	}

	// Override config with command line flags
	if *dbPath != "" {
		config.DatabasePath = *dbPath
	}
	if *readOnly {
		config.ReadOnly = true
	}
	if *transport != "stdio" {
		config.Transport.Type = *transport
		config.Transport.Host = *host
		config.Transport.Port = *port
		config.Transport.SocketPath = *socketPath
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Ensure database path exists
	if config.DatabasePath == "" {
		log.Fatal("Database path is required")
	}

	// Create database directory if it doesn't exist
	dbDir := filepath.Dir(config.DatabasePath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Open database
	var database db.KeyValueDB
	if config.ReadOnly {
		database, err = db.OpenReadOnly(config.DatabasePath)
	} else {
		database, err = db.Open(config.DatabasePath)
	}
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	log.Printf("Opened RocksDB at: %s (read-only: %v)", config.DatabasePath, config.ReadOnly)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		config.Name,
		config.Version,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithResourceCapabilities(true, true),
	)

	// Create and register tool manager
	toolManager := mcp.NewToolManager(database, config)
	if err := toolManager.RegisterTools(mcpServer); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	// Create and register prompt manager
	promptManager := mcp.NewPromptManager(database, config)
	if err := promptManager.RegisterPrompts(mcpServer); err != nil {
		log.Fatalf("Failed to register prompts: %v", err)
	}

	// Create and register resource manager
	resourceManager := mcp.NewResourceManager(database, config)
	if err := resourceManager.RegisterResources(mcpServer); err != nil {
		log.Fatalf("Failed to register resources: %v", err)
	}

	// Create transport manager - commented out for now
	// transportManager := mcp.NewTransportManager(config, mcpServer)

	// Setup graceful shutdown - simplified for stdio
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		signal.Stop(sigChan)
		close(sigChan)
	}()

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, shutting down gracefully...")
		// For stdio mode, the server will handle shutdown internally
	}()

	// Start the server based on transport type
	log.Printf("Starting MCP server with %s transport...", config.Transport.Type)

	switch config.Transport.Type {
	case "stdio":
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("STDIO server error: %v", err)
		}
	case "tcp", "websocket", "unix":
		// For now, fall back to stdio for other transport types
		log.Printf("Transport type %s not fully implemented, falling back to stdio", config.Transport.Type)
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("STDIO server error: %v", err)
		}
	default:
		log.Fatalf("Unknown transport type: %s", config.Transport.Type)
	}

	log.Println("MCP server shutdown complete")
}
