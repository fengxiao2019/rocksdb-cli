package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"rocksdb-cli/internal/config"
	"rocksdb-cli/internal/db"
	mcpserver "rocksdb-cli/internal/mcp/server"

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
	var cfg *config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.LoadConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	} else {
		cfg = config.DefaultConfig()
	}

	// Override config with command line flags
	if *dbPath != "" {
		cfg.Database.Path = *dbPath
	}
	if *readOnly {
		cfg.Database.ReadOnly = true
	}
	if *transport != "stdio" && cfg.MCPServer != nil {
		cfg.MCPServer.Transport.Type = *transport
		cfg.MCPServer.Transport.Host = *host
		cfg.MCPServer.Transport.Port = *port
		cfg.MCPServer.Transport.SocketPath = *socketPath
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Ensure database path exists
	if cfg.Database.Path == "" {
		log.Fatal("Database path is required")
	}

	// Convert to server config for backward compatibility
	serverConfig := mcpserver.NewConfigFromUnified(cfg)

	// Create database directory if it doesn't exist
	dbDir := filepath.Dir(cfg.Database.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Open database
	var database db.KeyValueDB
	if cfg.Database.ReadOnly {
		database, err = db.OpenReadOnly(cfg.Database.Path)
	} else {
		database, err = db.Open(cfg.Database.Path)
	}
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	log.Printf("Opened RocksDB at: %s (read-only: %v)", cfg.Database.Path, cfg.Database.ReadOnly)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		cfg.Name,
		cfg.Version,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithResourceCapabilities(true, true),
	)

	// Create and register tool manager
	toolManager := mcpserver.NewToolManager(database, serverConfig)
	if err := toolManager.RegisterTools(mcpServer); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	// Create and register prompt manager
	promptManager := mcpserver.NewPromptManager(database, serverConfig)
	if err := promptManager.RegisterPrompts(mcpServer); err != nil {
		log.Fatalf("Failed to register prompts: %v", err)
	}

	// Create and register resource manager
	resourceManager := mcpserver.NewResourceManager(database, serverConfig)
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
	log.Printf("Starting MCP server with %s transport...", serverConfig.Transport.Type)

	switch serverConfig.Transport.Type {
	case "stdio":
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("STDIO server error: %v", err)
		}
	case "tcp", "websocket", "unix":
		// For now, fall back to stdio for other transport types
		log.Printf("Transport type %s not fully implemented, falling back to stdio", serverConfig.Transport.Type)
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("STDIO server error: %v", err)
		}
	default:
		log.Fatalf("Unknown transport type: %s", serverConfig.Transport.Type)
	}

	log.Println("MCP server shutdown complete")
}
