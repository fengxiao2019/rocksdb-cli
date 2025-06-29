package mcp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

// Config holds the configuration for the MCP server
type Config struct {
	// Server information
	Name        string `yaml:"name" json:"name"`
	Version     string `yaml:"version" json:"version"`
	Description string `yaml:"description" json:"description"`

	// Database configuration
	DatabasePath string `yaml:"database_path" json:"database_path"`
	ReadOnly     bool   `yaml:"read_only" json:"read_only"`

	// Transport configuration
	Transport TransportConfig `yaml:"transport" json:"transport"`

	// Server settings
	MaxConcurrentSessions int           `yaml:"max_concurrent_sessions" json:"max_concurrent_sessions"`
	SessionTimeout        time.Duration `yaml:"session_timeout" json:"session_timeout"`

	// Tool configuration
	EnableAllTools bool     `yaml:"enable_all_tools" json:"enable_all_tools"`
	EnabledTools   []string `yaml:"enabled_tools" json:"enabled_tools"`
	DisabledTools  []string `yaml:"disabled_tools" json:"disabled_tools"`

	// Resource configuration
	EnableResources bool `yaml:"enable_resources" json:"enable_resources"`

	// Logging configuration
	LogLevel string `yaml:"log_level" json:"log_level"`
}

// TransportConfig defines the transport layer configuration
type TransportConfig struct {
	Type string `yaml:"type" json:"type"` // stdio, tcp, websocket, unix

	// For TCP transport
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`

	// For WebSocket transport
	Path string `yaml:"path" json:"path"`

	// For Unix Socket transport
	SocketPath string `yaml:"socket_path" json:"socket_path"`

	// Common transport settings
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// DefaultConfig returns the default configuration for the MCP server
func DefaultConfig() *Config {
	return &Config{
		Name:        "RocksDB MCP Server",
		Version:     "1.0.0",
		Description: "MCP server for RocksDB database operations with column family support",

		DatabasePath: "",
		ReadOnly:     false,

		Transport: TransportConfig{
			Type:    "stdio",
			Timeout: 30 * time.Second,
		},

		MaxConcurrentSessions: 10,
		SessionTimeout:        5 * time.Minute,

		EnableAllTools:  true,
		EnabledTools:    []string{},
		DisabledTools:   []string{},
		EnableResources: true,

		LogLevel: "info",
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Name == "" {
		return &ConfigError{Field: "name", Message: "server name is required"}
	}

	if c.Version == "" {
		return &ConfigError{Field: "version", Message: "server version is required"}
	}

	if c.DatabasePath == "" {
		return &ConfigError{Field: "database_path", Message: "database path is required"}
	}

	if c.MaxConcurrentSessions <= 0 {
		c.MaxConcurrentSessions = 10
	}

	if c.SessionTimeout <= 0 {
		c.SessionTimeout = 5 * time.Minute
	}

	// Validate transport configuration
	switch c.Transport.Type {
	case "stdio":
		// No additional validation needed for stdio
	case "tcp":
		if c.Transport.Host == "" {
			c.Transport.Host = "localhost"
		}
		if c.Transport.Port <= 0 || c.Transport.Port > 65535 {
			return &ConfigError{Field: "transport.port", Message: "valid port number is required for TCP transport"}
		}
	case "websocket":
		if c.Transport.Path == "" {
			c.Transport.Path = "/mcp"
		}
		if c.Transport.Port <= 0 || c.Transport.Port > 65535 {
			return &ConfigError{Field: "transport.port", Message: "valid port number is required for WebSocket transport"}
		}
	case "unix":
		if c.Transport.SocketPath == "" {
			return &ConfigError{Field: "transport.socket_path", Message: "socket path is required for Unix transport"}
		}
	default:
		return &ConfigError{Field: "transport.type", Message: "unsupported transport type: " + c.Transport.Type}
	}

	if c.Transport.Timeout <= 0 {
		c.Transport.Timeout = 30 * time.Second
	}

	return nil
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error in field '" + e.Field + "': " + e.Message
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
