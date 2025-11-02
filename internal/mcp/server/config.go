package server

import (
	"time"

	"rocksdb-cli/internal/config"
)

// Config holds the configuration for the MCP server (legacy compatibility)
// This is kept for backward compatibility with existing server manager code
type Config struct {
	// Server information
	Name        string
	Version     string
	Description string

	// Database configuration
	DatabasePath string
	ReadOnly     bool

	// Transport configuration
	Transport TransportConfig

	// Server settings
	MaxConcurrentSessions int
	SessionTimeout        time.Duration

	// Tool configuration
	EnableAllTools bool
	EnabledTools   []string
	DisabledTools  []string

	// Resource configuration
	EnableResources bool

	// Logging configuration
	LogLevel string
}

// TransportConfig defines the transport layer configuration
type TransportConfig struct {
	Type string // stdio, tcp, websocket, unix

	// For TCP transport
	Host string
	Port int

	// For WebSocket transport
	Path string

	// For Unix Socket transport
	SocketPath string

	// Common transport settings
	Timeout time.Duration
}

// NewConfigFromUnified creates a server Config from the new unified config
func NewConfigFromUnified(cfg *config.Config) *Config {
	serverConfig := &Config{
		Name:                  cfg.Name,
		Version:               cfg.Version,
		Description:           "MCP server for RocksDB database operations",
		DatabasePath:          cfg.Database.Path,
		ReadOnly:              cfg.Database.ReadOnly,
		MaxConcurrentSessions: 10,
		SessionTimeout:        5 * time.Minute,
		EnableAllTools:        true,
		EnabledTools:          []string{},
		DisabledTools:         []string{},
		EnableResources:       true,
		LogLevel:              cfg.LogLevel,
	}

	// Convert MCP server transport config if enabled
	if cfg.MCPServer != nil && cfg.MCPServer.Enabled {
		serverConfig.Transport = TransportConfig{
			Type:       cfg.MCPServer.Transport.Type,
			Host:       cfg.MCPServer.Transport.Host,
			Port:       cfg.MCPServer.Transport.Port,
			SocketPath: cfg.MCPServer.Transport.SocketPath,
			Timeout:    30 * time.Second,
		}
	} else {
		// Default to stdio
		serverConfig.Transport = TransportConfig{
			Type:    "stdio",
			Timeout: 30 * time.Second,
		}
	}

	return serverConfig
}

// DefaultConfig returns a default server configuration for testing
func DefaultConfig() *Config {
	return &Config{
		Name:                  "RocksDB MCP Server",
		Version:               "1.0.0",
		Description:           "MCP server for RocksDB database operations",
		DatabasePath:          "./data/rocksdb",
		ReadOnly:              false,
		Transport: TransportConfig{
			Type:    "stdio",
			Timeout: 30 * time.Second,
		},
		MaxConcurrentSessions: 10,
		SessionTimeout:        5 * time.Minute,
		EnableAllTools:        true,
		EnabledTools:          []string{},
		DisabledTools:         []string{},
		EnableResources:       true,
		LogLevel:              "info",
	}
}

// IsToolEnabled checks if a tool is enabled based on configuration
func (c *Config) IsToolEnabled(toolName string) bool {
	// If all tools are enabled and tool is not in disabled list
	if c.EnableAllTools {
		for _, disabled := range c.DisabledTools {
			if disabled == toolName {
				return false
			}
		}
		return true
	}

	// If not all tools are enabled, check if it's in enabled list
	for _, enabled := range c.EnabledTools {
		if enabled == toolName {
			return true
		}
	}

	return false
}
