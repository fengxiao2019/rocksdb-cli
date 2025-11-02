package config

import (
	"fmt"
	"time"
)

// MCPClientConfig holds configuration for a single MCP client connection
type MCPClientConfig struct {
	// Basic information
	Name    string `yaml:"name" json:"name"`
	Enabled bool   `yaml:"enabled" json:"enabled"`

	// Transport configuration
	Transport string `yaml:"transport" json:"transport"` // stdio, tcp, websocket, unix

	// For STDIO transport
	Command string   `yaml:"command,omitempty" json:"command,omitempty"`
	Args    []string `yaml:"args,omitempty" json:"args,omitempty"`

	// For TCP/WebSocket transport
	Host string `yaml:"host,omitempty" json:"host,omitempty"`
	Port int    `yaml:"port,omitempty" json:"port,omitempty"`
	Path string `yaml:"path,omitempty" json:"path,omitempty"` // For WebSocket

	// For Unix Socket transport
	SocketPath string `yaml:"socket_path,omitempty" json:"socket_path,omitempty"`

	// Connection settings
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// Environment variables
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`

	// Retry configuration
	Retry RetryConfig `yaml:"retry,omitempty" json:"retry,omitempty"`

	// Tool filtering
	EnabledTools  []string `yaml:"enabled_tools,omitempty" json:"enabled_tools,omitempty"`
	DisabledTools []string `yaml:"disabled_tools,omitempty" json:"disabled_tools,omitempty"`
}

// RetryConfig holds retry configuration for MCP client connections
type RetryConfig struct {
	MaxAttempts int           `yaml:"max_attempts" json:"max_attempts"`
	Backoff     string        `yaml:"backoff" json:"backoff"` // constant, linear, exponential
	InitialWait time.Duration `yaml:"initial_wait" json:"initial_wait"`
	MaxWait     time.Duration `yaml:"max_wait" json:"max_wait"`
}

// MCPClientsConfig holds configuration for all MCP clients
type MCPClientsConfig map[string]*MCPClientConfig

// DefaultMCPClientConfig returns default configuration for an MCP client
func DefaultMCPClientConfig(name string) *MCPClientConfig {
	return &MCPClientConfig{
		Name:      name,
		Enabled:   true,
		Transport: "stdio",
		Timeout:   30 * time.Second,
		Retry: RetryConfig{
			MaxAttempts: 3,
			Backoff:     "exponential",
			InitialWait: 1 * time.Second,
			MaxWait:     30 * time.Second,
		},
	}
}

// Validate validates the MCP client configuration
func (c *MCPClientConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("client name is required")
	}

	// Validate timeout
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	// Validate transport-specific configuration
	switch c.Transport {
	case "stdio":
		if c.Command == "" {
			return fmt.Errorf("command is required for stdio transport")
		}
	case "tcp":
		if c.Host == "" {
			c.Host = "localhost"
		}
		if c.Port <= 0 || c.Port > 65535 {
			return fmt.Errorf("valid port number is required for TCP transport")
		}
	case "websocket":
		if c.Host == "" {
			c.Host = "localhost"
		}
		if c.Port <= 0 || c.Port > 65535 {
			return fmt.Errorf("valid port number is required for WebSocket transport")
		}
		if c.Path == "" {
			c.Path = "/mcp"
		}
	case "unix":
		if c.SocketPath == "" {
			return fmt.Errorf("socket path is required for Unix socket transport")
		}
	default:
		return fmt.Errorf("unsupported transport type: %s", c.Transport)
	}

	// Validate retry configuration
	if c.Retry.MaxAttempts <= 0 {
		c.Retry.MaxAttempts = 3
	}

	if c.Retry.Backoff == "" {
		c.Retry.Backoff = "exponential"
	}

	switch c.Retry.Backoff {
	case "constant", "linear", "exponential":
		// Valid backoff strategies
	default:
		return fmt.Errorf("unsupported backoff strategy: %s", c.Retry.Backoff)
	}

	if c.Retry.InitialWait <= 0 {
		c.Retry.InitialWait = 1 * time.Second
	}

	if c.Retry.MaxWait <= 0 {
		c.Retry.MaxWait = 30 * time.Second
	}

	return nil
}

// IsToolEnabled checks if a specific tool is enabled for this client
func (c *MCPClientConfig) IsToolEnabled(toolName string) bool {
	// If no filters are specified, all tools are enabled
	if len(c.EnabledTools) == 0 && len(c.DisabledTools) == 0 {
		return true
	}

	// Check if explicitly disabled
	for _, disabled := range c.DisabledTools {
		if disabled == toolName {
			return false
		}
	}

	// If enabled tools list exists, check if tool is in it
	if len(c.EnabledTools) > 0 {
		for _, enabled := range c.EnabledTools {
			if enabled == toolName {
				return true
			}
		}
		return false
	}

	// Tool is not in disabled list and no enabled list exists
	return true
}
