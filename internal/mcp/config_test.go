package mcp

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test basic fields
	if config.Name == "" {
		t.Error("Expected non-empty server name")
	}
	if config.Version == "" {
		t.Error("Expected non-empty server version")
	}

	// Test transport defaults
	if config.Transport.Type != "stdio" {
		t.Errorf("Expected default transport type 'stdio', got '%s'", config.Transport.Type)
	}
	if config.Transport.Host != "" {
		t.Errorf("Expected empty host for stdio transport, got '%s'", config.Transport.Host)
	}
	if config.Transport.Port != 0 {
		t.Errorf("Expected port 0 for stdio transport, got %d", config.Transport.Port)
	}

	// Test feature flags
	if !config.EnableAllTools {
		t.Error("Expected tools to be enabled by default")
	}
	if !config.EnableResources {
		t.Error("Expected resources to be enabled by default")
	}

	// Test timeouts
	if config.Transport.Timeout <= 0 {
		t.Error("Expected positive timeout value")
	}
	if config.SessionTimeout <= 0 {
		t.Error("Expected positive session timeout value")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: true, // DatabasePath is empty in default config
		},
		{
			name: "valid config with database path",
			config: func() *Config {
				c := DefaultConfig()
				c.DatabasePath = "/tmp/test.db"
				return c
			}(),
			wantErr: false,
		},
		{
			name: "empty server name",
			config: &Config{
				Name:         "",
				Version:      "1.0.0",
				DatabasePath: "/tmp/test.db",
				Transport:    TransportConfig{Type: "stdio"},
			},
			wantErr: true,
		},
		{
			name: "empty version",
			config: &Config{
				Name:         "Test Server",
				Version:      "",
				DatabasePath: "/tmp/test.db",
				Transport:    TransportConfig{Type: "stdio"},
			},
			wantErr: true,
		},
		{
			name: "empty database path",
			config: &Config{
				Name:         "Test Server",
				Version:      "1.0.0",
				DatabasePath: "",
				Transport:    TransportConfig{Type: "stdio"},
			},
			wantErr: true,
		},
		{
			name: "invalid transport type",
			config: &Config{
				Name:         "Test Server",
				Version:      "1.0.0",
				DatabasePath: "/tmp/test.db",
				Transport:    TransportConfig{Type: "invalid"},
			},
			wantErr: true,
		},
		{
			name: "invalid port for tcp",
			config: &Config{
				Name:         "Test Server",
				Version:      "1.0.0",
				DatabasePath: "/tmp/test.db",
				Transport:    TransportConfig{Type: "tcp", Port: -1},
			},
			wantErr: true,
		},
		{
			name: "valid tcp config",
			config: &Config{
				Name:         "Test Server",
				Version:      "1.0.0",
				DatabasePath: "/tmp/test.db",
				Transport:    TransportConfig{Type: "tcp", Port: 8080},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigYAML(t *testing.T) {
	// Create temporary YAML config file
	yamlContent := `name: "Test MCP Server"
version: "1.0.0"
database_path: "/tmp/test.db"
read_only: true
enable_all_tools: true
enable_resources: false
transport:
  type: "tcp"
  host: "0.0.0.0"
  port: 9090
  timeout: 60s
max_concurrent_sessions: 5
session_timeout: 10m`

	tmpDir, err := ioutil.TempDir("", "mcp-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := ioutil.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test loading
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify loaded values
	if config.Name != "Test MCP Server" {
		t.Errorf("Expected name 'Test MCP Server', got '%s'", config.Name)
	}
	if config.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", config.Version)
	}
	if config.DatabasePath != "/tmp/test.db" {
		t.Errorf("Expected database path '/tmp/test.db', got '%s'", config.DatabasePath)
	}
	if !config.ReadOnly {
		t.Error("Expected read_only to be true")
	}
	if !config.EnableAllTools {
		t.Error("Expected enable_all_tools to be true")
	}
	if config.EnableResources {
		t.Error("Expected enable_resources to be false")
	}
	if config.Transport.Type != "tcp" {
		t.Errorf("Expected transport type 'tcp', got '%s'", config.Transport.Type)
	}
	if config.Transport.Host != "0.0.0.0" {
		t.Errorf("Expected transport host '0.0.0.0', got '%s'", config.Transport.Host)
	}
	if config.Transport.Port != 9090 {
		t.Errorf("Expected transport port 9090, got %d", config.Transport.Port)
	}
	if config.Transport.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", config.Transport.Timeout)
	}
	if config.MaxConcurrentSessions != 5 {
		t.Errorf("Expected max concurrent sessions 5, got %d", config.MaxConcurrentSessions)
	}
	if config.SessionTimeout != 10*time.Minute {
		t.Errorf("Expected session timeout 10m, got %v", config.SessionTimeout)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     string
		expectError bool
	}{
		{
			name:        "non-existent file",
			filename:    "nonexistent.yaml",
			content:     "",
			expectError: true,
		},
		{
			name:        "invalid YAML",
			filename:    "invalid.yaml",
			content:     "invalid: yaml: content: [",
			expectError: true,
		},
		{
			name:        "unsupported format",
			filename:    "config.txt",
			content:     "some content",
			expectError: true,
		},
	}

	tmpDir, err := ioutil.TempDir("", "mcp-config-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.filename)

			if tt.content != "" {
				if err := ioutil.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
			}

			_, err := LoadConfig(configPath)
			if (err != nil) != tt.expectError {
				t.Errorf("LoadConfig() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{Field: "test_field", Message: "test error"}
	expected := "config error in field 'test_field': test error"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}
