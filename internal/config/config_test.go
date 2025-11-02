package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "RocksDB CLI", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, "./data/rocksdb", config.Database.Path)
	assert.False(t, config.Database.ReadOnly)
	assert.NotNil(t, config.MCPServer)
	assert.NotNil(t, config.MCPClients)
	assert.NotNil(t, config.GraphChain)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "missing name",
			config: &Config{
				Version: "1.0.0",
				Database: DatabaseConfig{
					Path: "/tmp/test",
				},
			},
			wantErr: true,
			errMsg:  "application name is required",
		},
		{
			name: "missing version",
			config: &Config{
				Name: "Test",
				Database: DatabaseConfig{
					Path: "/tmp/test",
				},
			},
			wantErr: true,
			errMsg:  "application version is required",
		},
		{
			name: "missing database path",
			config: &Config{
				Name:    "Test",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "database path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMCPClientConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *MCPClientConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid stdio config",
			config: &MCPClientConfig{
				Name:      "test",
				Transport: "stdio",
				Command:   "npx",
				Args:      []string{"-y", "@test/server"},
				Timeout:   30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing command for stdio",
			config: &MCPClientConfig{
				Name:      "test",
				Transport: "stdio",
				Timeout:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "command is required for stdio transport",
		},
		{
			name: "valid tcp config",
			config: &MCPClientConfig{
				Name:      "test",
				Transport: "tcp",
				Host:      "localhost",
				Port:      8080,
				Timeout:   30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid port for tcp",
			config: &MCPClientConfig{
				Name:      "test",
				Transport: "tcp",
				Host:      "localhost",
				Port:      0,
				Timeout:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "valid port number is required",
		},
		{
			name: "valid unix socket config",
			config: &MCPClientConfig{
				Name:       "test",
				Transport:  "unix",
				SocketPath: "/tmp/test.sock",
				Timeout:    30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing socket path for unix",
			config: &MCPClientConfig{
				Name:      "test",
				Transport: "unix",
				Timeout:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "socket path is required",
		},
		{
			name: "unsupported transport",
			config: &MCPClientConfig{
				Name:      "test",
				Transport: "invalid",
				Timeout:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "unsupported transport type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMCPClientConfig_IsToolEnabled(t *testing.T) {
	tests := []struct {
		name       string
		config     *MCPClientConfig
		toolName   string
		wantEnabled bool
	}{
		{
			name: "no filters - all enabled",
			config: &MCPClientConfig{
				EnabledTools:  []string{},
				DisabledTools: []string{},
			},
			toolName:   "any_tool",
			wantEnabled: true,
		},
		{
			name: "explicitly enabled",
			config: &MCPClientConfig{
				EnabledTools:  []string{"tool1", "tool2"},
				DisabledTools: []string{},
			},
			toolName:   "tool1",
			wantEnabled: true,
		},
		{
			name: "not in enabled list",
			config: &MCPClientConfig{
				EnabledTools:  []string{"tool1", "tool2"},
				DisabledTools: []string{},
			},
			toolName:   "tool3",
			wantEnabled: false,
		},
		{
			name: "explicitly disabled",
			config: &MCPClientConfig{
				EnabledTools:  []string{},
				DisabledTools: []string{"tool1", "tool2"},
			},
			toolName:   "tool1",
			wantEnabled: false,
		},
		{
			name: "not in disabled list",
			config: &MCPClientConfig{
				EnabledTools:  []string{},
				DisabledTools: []string{"tool1", "tool2"},
			},
			toolName:   "tool3",
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enabled := tt.config.IsToolEnabled(tt.toolName)
			assert.Equal(t, tt.wantEnabled, enabled)
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
name: "Test App"
version: "1.0.0"

database:
  path: "/tmp/testdb"
  read_only: true

mcp_clients:
  filesystem:
    enabled: true
    transport: "stdio"
    command: "npx"
    args: ["-y", "@test/fs"]
    timeout: 30s
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config
	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, "Test App", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, "/tmp/testdb", config.Database.Path)
	assert.True(t, config.Database.ReadOnly)
	assert.Len(t, config.MCPClients, 1)

	fsClient, ok := config.MCPClients["filesystem"]
	require.True(t, ok)
	assert.True(t, fsClient.Enabled)
	assert.Equal(t, "stdio", fsClient.Transport)
	assert.Equal(t, "npx", fsClient.Command)
}

func TestSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	config := DefaultConfig()
	config.Name = "Test Saved Config"
	config.Database.Path = "/tmp/saved"

	// Add an MCP client
	config.MCPClients["test"] = &MCPClientConfig{
		Name:      "test",
		Enabled:   true,
		Transport: "stdio",
		Command:   "test-command",
		Args:      []string{"arg1", "arg2"},
		Timeout:   30 * time.Second,
		Retry: RetryConfig{
			MaxAttempts: 3,
			Backoff:     "exponential",
			InitialWait: 1 * time.Second,
			MaxWait:     30 * time.Second,
		},
	}

	// Save config
	err := SaveConfig(config, configPath)
	require.NoError(t, err)

	// Load it back
	loadedConfig, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, config.Name, loadedConfig.Name)
	assert.Equal(t, config.Database.Path, loadedConfig.Database.Path)
	assert.Len(t, loadedConfig.MCPClients, 1)
}

func TestConfig_GetEnabledMCPClients(t *testing.T) {
	config := DefaultConfig()

	config.MCPClients["enabled1"] = &MCPClientConfig{
		Name:      "enabled1",
		Enabled:   true,
		Transport: "stdio",
		Command:   "cmd1",
	}

	config.MCPClients["disabled"] = &MCPClientConfig{
		Name:      "disabled",
		Enabled:   false,
		Transport: "stdio",
		Command:   "cmd2",
	}

	config.MCPClients["enabled2"] = &MCPClientConfig{
		Name:      "enabled2",
		Enabled:   true,
		Transport: "stdio",
		Command:   "cmd3",
	}

	enabled := config.GetEnabledMCPClients()
	assert.Len(t, enabled, 2)
	assert.Contains(t, enabled, "enabled1")
	assert.Contains(t, enabled, "enabled2")
	assert.NotContains(t, enabled, "disabled")
}

func TestConfig_GetMCPClient(t *testing.T) {
	config := DefaultConfig()

	config.MCPClients["test"] = &MCPClientConfig{
		Name:      "test",
		Transport: "stdio",
		Command:   "test-cmd",
	}

	// Test existing client
	client, err := config.GetMCPClient("test")
	require.NoError(t, err)
	assert.Equal(t, "test", client.Name)

	// Test non-existing client
	_, err = config.GetMCPClient("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
