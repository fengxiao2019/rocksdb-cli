package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Config holds the complete application configuration
type Config struct {
	// Application information
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`

	// Database configuration
	Database DatabaseConfig `yaml:"database" json:"database"`

	// MCP Server configuration (this tool as MCP server)
	MCPServer *MCPServerConfig `yaml:"mcp_server,omitempty" json:"mcp_server,omitempty"`

	// MCP Clients configuration (connecting to other MCP servers)
	MCPClients MCPClientsConfig `yaml:"mcp_clients,omitempty" json:"mcp_clients,omitempty"`

	// GraphChain AI configuration
	GraphChain *GraphChainConfig `yaml:"graphchain,omitempty" json:"graphchain,omitempty"`

	// Logging configuration
	LogLevel string `yaml:"log_level" json:"log_level"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path     string `yaml:"path" json:"path"`
	ReadOnly bool   `yaml:"read_only" json:"read_only"`
}

// MCPServerConfig holds MCP server configuration
// This is a placeholder - actual implementation is in internal/mcp/config.go
type MCPServerConfig struct {
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Transport   TransportConfig   `yaml:"transport" json:"transport"`
	Tools       ToolsConfig       `yaml:"tools,omitempty" json:"tools,omitempty"`
}

// TransportConfig holds transport configuration for MCP server
type TransportConfig struct {
	Type       string `yaml:"type" json:"type"`
	Host       string `yaml:"host,omitempty" json:"host,omitempty"`
	Port       int    `yaml:"port,omitempty" json:"port,omitempty"`
	SocketPath string `yaml:"socket_path,omitempty" json:"socket_path,omitempty"`
}

// ToolsConfig holds tools configuration for MCP server
type ToolsConfig struct {
	EnableAll []string `yaml:"enable_all,omitempty" json:"enable_all,omitempty"`
	Disabled  []string `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

// GraphChainConfig holds GraphChain AI configuration
type GraphChainConfig struct {
	Model          string `yaml:"model" json:"model"`
	EnableMCPTools bool   `yaml:"enable_mcp_tools" json:"enable_mcp_tools"`
	AutoConnect    bool   `yaml:"auto_connect" json:"auto_connect"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Name:    "RocksDB CLI",
		Version: "1.0.0",
		Database: DatabaseConfig{
			Path:     "./data/rocksdb",
			ReadOnly: false,
		},
		MCPServer: &MCPServerConfig{
			Enabled: false,
			Name:    "RocksDB MCP Server",
			Transport: TransportConfig{
				Type: "stdio",
			},
		},
		MCPClients: make(MCPClientsConfig),
		GraphChain: &GraphChainConfig{
			Model:          "gpt-4",
			EnableMCPTools: true,
			AutoConnect:    true,
		},
		LogLevel: "info",
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse based on file extension
	config := DefaultConfig()
	ext := filepath.Ext(configPath)

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	return config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, configPath string) error {
	// Validate configuration first
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal based on file extension
	ext := filepath.Ext(configPath)
	var data []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML config: %w", err)
		}
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	// Write to file
	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("application name is required")
	}

	if c.Version == "" {
		return fmt.Errorf("application version is required")
	}

	// Validate database configuration
	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}

	// Validate MCP server configuration
	if c.MCPServer != nil && c.MCPServer.Enabled {
		if c.MCPServer.Name == "" {
			return fmt.Errorf("MCP server name is required")
		}
	}

	// Validate MCP clients configuration
	for name, client := range c.MCPClients {
		if client.Name == "" {
			client.Name = name
		}
		if err := client.Validate(); err != nil {
			return fmt.Errorf("invalid MCP client '%s': %w", name, err)
		}
	}

	return nil
}

// GetEnabledMCPClients returns a list of enabled MCP clients
func (c *Config) GetEnabledMCPClients() MCPClientsConfig {
	enabled := make(MCPClientsConfig)
	for name, client := range c.MCPClients {
		if client.Enabled {
			enabled[name] = client
		}
	}
	return enabled
}

// GetMCPClient returns a specific MCP client configuration by name
func (c *Config) GetMCPClient(name string) (*MCPClientConfig, error) {
	client, ok := c.MCPClients[name]
	if !ok {
		return nil, fmt.Errorf("MCP client '%s' not found", name)
	}
	return client, nil
}
