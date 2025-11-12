package graphchain

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

	// Test LLM defaults
	assert.Equal(t, "openai", config.GraphChain.LLM.Provider)
	assert.Equal(t, "gpt-4", config.GraphChain.LLM.Model)
	assert.Equal(t, 30*time.Second, config.GraphChain.LLM.Timeout)

	// Test Agent defaults
	assert.Equal(t, 10, config.GraphChain.Agent.MaxIterations)
	assert.Equal(t, 10*time.Second, config.GraphChain.Agent.ToolTimeout)
	assert.Equal(t, 100, config.GraphChain.Agent.MemorySize)

	// Test Security defaults
	assert.Equal(t, 10, config.GraphChain.Security.MaxQueryComplexity)
	expectedOps := []string{"get", "scan", "prefix", "jsonquery", "search", "stats"}
	assert.Equal(t, expectedOps, config.GraphChain.Security.AllowedOperations)

	// Test Context defaults
	assert.Equal(t, 5*time.Minute, config.GraphChain.Context.UpdateInterval)
	assert.Equal(t, 4096, config.GraphChain.Context.MaxContextSize)
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configYAML := `
graphchain:
  llm:
    provider: "ollama"
    model: "llama2"
    base_url: "http://localhost:11434"
    timeout: "45s"
  agent:
    max_iterations: 15
    tool_timeout: "20s"
    enable_memory: true
    memory_size: 200
  security:
    enable_audit: true
    read_only_mode: false
    max_query_complexity: 20
    allowed_operations: ["get", "scan"]
  context:
    enable_auto_discovery: true
    update_interval: "10m"
    max_context_size: 8192
`

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	require.NoError(t, err)

	// Load config
	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	// Verify loaded values
	assert.Equal(t, "ollama", config.GraphChain.LLM.Provider)
	assert.Equal(t, "llama2", config.GraphChain.LLM.Model)
	assert.Equal(t, "http://localhost:11434", config.GraphChain.LLM.BaseURL)
	assert.Equal(t, 45*time.Second, config.GraphChain.LLM.Timeout)

	assert.Equal(t, 15, config.GraphChain.Agent.MaxIterations)
	assert.Equal(t, 20*time.Second, config.GraphChain.Agent.ToolTimeout)
	assert.True(t, config.GraphChain.Agent.EnableMemory)
	assert.Equal(t, 200, config.GraphChain.Agent.MemorySize)

	assert.True(t, config.GraphChain.Security.EnableAudit)
	assert.False(t, config.GraphChain.Security.ReadOnlyMode)
	assert.Equal(t, 20, config.GraphChain.Security.MaxQueryComplexity)
	assert.Equal(t, []string{"get", "scan"}, config.GraphChain.Security.AllowedOperations)

	assert.True(t, config.GraphChain.Context.EnableAutoDiscovery)
	assert.Equal(t, 10*time.Minute, config.GraphChain.Context.UpdateInterval)
	assert.Equal(t, 8192, config.GraphChain.Context.MaxContextSize)
}

func TestLoadConfigWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	originalVars := map[string]string{
		"GRAPHCHAIN_LLM_PROVIDER": os.Getenv("GRAPHCHAIN_LLM_PROVIDER"),
		"GRAPHCHAIN_LLM_MODEL":    os.Getenv("GRAPHCHAIN_LLM_MODEL"),
		"GRAPHCHAIN_API_KEY":      os.Getenv("GRAPHCHAIN_API_KEY"),
		"GRAPHCHAIN_BASE_URL":     os.Getenv("GRAPHCHAIN_BASE_URL"),
		"GRAPHCHAIN_READ_ONLY":    os.Getenv("GRAPHCHAIN_READ_ONLY"),
		"GRAPHCHAIN_ENABLE_AUDIT": os.Getenv("GRAPHCHAIN_ENABLE_AUDIT"),
	}

	// Clean up environment variables at the end
	t.Cleanup(func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	})

	// Set test environment variables
	os.Setenv("GRAPHCHAIN_LLM_PROVIDER", "anthropic")
	os.Setenv("GRAPHCHAIN_LLM_MODEL", "claude-3")
	os.Setenv("GRAPHCHAIN_API_KEY", "test-api-key")
	os.Setenv("GRAPHCHAIN_BASE_URL", "https://api.anthropic.com")
	os.Setenv("GRAPHCHAIN_READ_ONLY", "true")
	os.Setenv("GRAPHCHAIN_ENABLE_AUDIT", "false")

	config, err := LoadConfig("")
	require.NoError(t, err)

	// Verify environment overrides
	assert.Equal(t, "anthropic", config.GraphChain.LLM.Provider)
	assert.Equal(t, "claude-3", config.GraphChain.LLM.Model)
	assert.Equal(t, "test-api-key", config.GraphChain.LLM.APIKey)
	assert.Equal(t, "https://api.anthropic.com", config.GraphChain.LLM.BaseURL)
	assert.True(t, config.GraphChain.Security.ReadOnlyMode)
	assert.False(t, config.GraphChain.Security.EnableAudit)
}

func TestLoadConfigWithEnvironmentVariableExpansion(t *testing.T) {
	// Set environment variable for expansion
	os.Setenv("TEST_API_KEY", "expanded-api-key")
	t.Cleanup(func() {
		os.Unsetenv("TEST_API_KEY")
	})

	// Create temporary config file with environment variable
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configYAML := `
graphchain:
  llm:
    provider: "openai"
    model: "gpt-4"
    api_key: "${TEST_API_KEY}"
`

	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	require.NoError(t, err)

	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, "expanded-api-key", config.GraphChain.LLM.APIKey)
}

func TestValidateConfig_ValidConfig(t *testing.T) {
	config := &Config{
		GraphChain: GraphChainConfig{
			LLM: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "test-key",
			},
			Agent: AgentConfig{
				MaxIterations: 5,
			},
			Security: SecurityConfig{
				MaxQueryComplexity: 5,
			},
			Context: ContextConfig{
				MaxContextSize: 1024,
			},
		},
	}

	err := validateConfig(config)
	assert.NoError(t, err)
}

func TestValidateConfig_InvalidProvider(t *testing.T) {
	config := &Config{
		GraphChain: GraphChainConfig{
			LLM: LLMConfig{
				Provider: "invalid-provider",
				Model:    "test-model",
			},
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported LLM provider")
}

func TestValidateConfig_MissingAPIKey(t *testing.T) {
	config := &Config{
		GraphChain: GraphChainConfig{
			LLM: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				// APIKey is missing
			},
			Agent: AgentConfig{
				MaxIterations: 5,
			},
			Security: SecurityConfig{
				MaxQueryComplexity: 5,
			},
			Context: ContextConfig{
				MaxContextSize: 1024,
			},
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required for provider: openai")
}

func TestValidateConfig_OllamaNoAPIKey(t *testing.T) {
	config := &Config{
		GraphChain: GraphChainConfig{
			LLM: LLMConfig{
				Provider: "ollama",
				Model:    "llama2",
				// No API key needed for Ollama
			},
			Agent: AgentConfig{
				MaxIterations: 5,
			},
			Security: SecurityConfig{
				MaxQueryComplexity: 5,
			},
			Context: ContextConfig{
				MaxContextSize: 1024,
			},
		},
	}

	err := validateConfig(config)
	assert.NoError(t, err)
}

func TestValidateConfig_InvalidConstraints(t *testing.T) {
	testCases := []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name: "negative max iterations",
			config: &Config{
				GraphChain: GraphChainConfig{
					LLM: LLMConfig{Provider: "ollama", Model: "test"},
					Agent: AgentConfig{
						MaxIterations: -1,
					},
					Security: SecurityConfig{MaxQueryComplexity: 5},
					Context:  ContextConfig{MaxContextSize: 1024},
				},
			},
			expectedErr: "agent max_iterations must be positive",
		},
		{
			name: "negative query complexity",
			config: &Config{
				GraphChain: GraphChainConfig{
					LLM: LLMConfig{Provider: "ollama", Model: "test"},
					Agent: AgentConfig{
						MaxIterations: 5,
					},
					Security: SecurityConfig{MaxQueryComplexity: -1},
					Context:  ContextConfig{MaxContextSize: 1024},
				},
			},
			expectedErr: "security max_query_complexity must be positive",
		},
		{
			name: "negative context size",
			config: &Config{
				GraphChain: GraphChainConfig{
					LLM: LLMConfig{Provider: "ollama", Model: "test"},
					Agent: AgentConfig{
						MaxIterations: 5,
					},
					Security: SecurityConfig{MaxQueryComplexity: 5},
					Context:  ContextConfig{MaxContextSize: -1},
				},
			},
			expectedErr: "context max_context_size must be positive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfig(tc.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	assert.True(t, contains(slice, "apple"))
	assert.True(t, contains(slice, "banana"))
	assert.True(t, contains(slice, "cherry"))
	assert.False(t, contains(slice, "grape"))
	assert.False(t, contains(slice, ""))
	assert.False(t, contains([]string{}, "apple"))
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	config, err := LoadConfig("/nonexistent/path/config.yaml")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-config.yaml")

	invalidYAML := `
graphchain:
  llm:
    provider: "openai
    # Missing closing quote - invalid YAML
`

	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	config, err := LoadConfig(configPath)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLegacyEnvironmentVariableFallback(t *testing.T) {
	testCases := []struct {
		name             string
		provider         string
		graphchainAPIKey string
		legacyAPIKey     string
		legacyEnvVar     string
		expectedAPIKey   string
	}{
		{
			name:             "GRAPHCHAIN_API_KEY takes priority for OpenAI",
			provider:         "openai",
			graphchainAPIKey: "new-openai-key",
			legacyAPIKey:     "legacy-openai-key",
			legacyEnvVar:     "OPENAI_API_KEY",
			expectedAPIKey:   "new-openai-key",
		},
		{
			name:             "Fallback to OPENAI_API_KEY when GRAPHCHAIN_API_KEY is empty",
			provider:         "openai",
			graphchainAPIKey: "",
			legacyAPIKey:     "legacy-openai-key",
			legacyEnvVar:     "OPENAI_API_KEY",
			expectedAPIKey:   "legacy-openai-key",
		},
		{
			name:             "GRAPHCHAIN_API_KEY takes priority for Azure OpenAI",
			provider:         "azureopenai",
			graphchainAPIKey: "new-azure-key",
			legacyAPIKey:     "legacy-azure-key",
			legacyEnvVar:     "AZURE_OPENAI_API_KEY",
			expectedAPIKey:   "new-azure-key",
		},
		{
			name:             "Fallback to AZURE_OPENAI_API_KEY when GRAPHCHAIN_API_KEY is empty",
			provider:         "azureopenai",
			graphchainAPIKey: "",
			legacyAPIKey:     "legacy-azure-key",
			legacyEnvVar:     "AZURE_OPENAI_API_KEY",
			expectedAPIKey:   "legacy-azure-key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save original environment
			originalGraphchain := os.Getenv("GRAPHCHAIN_API_KEY")
			originalLegacy := os.Getenv(tc.legacyEnvVar)
			originalProvider := os.Getenv("GRAPHCHAIN_LLM_PROVIDER")
			originalAzureEndpoint := os.Getenv("GRAPHCHAIN_AZURE_ENDPOINT")
			originalAzureDeployment := os.Getenv("GRAPHCHAIN_AZURE_DEPLOYMENT")

			t.Cleanup(func() {
				if originalGraphchain == "" {
					os.Unsetenv("GRAPHCHAIN_API_KEY")
				} else {
					os.Setenv("GRAPHCHAIN_API_KEY", originalGraphchain)
				}
				if originalLegacy == "" {
					os.Unsetenv(tc.legacyEnvVar)
				} else {
					os.Setenv(tc.legacyEnvVar, originalLegacy)
				}
				if originalProvider == "" {
					os.Unsetenv("GRAPHCHAIN_LLM_PROVIDER")
				} else {
					os.Setenv("GRAPHCHAIN_LLM_PROVIDER", originalProvider)
				}
				if originalAzureEndpoint == "" {
					os.Unsetenv("GRAPHCHAIN_AZURE_ENDPOINT")
				} else {
					os.Setenv("GRAPHCHAIN_AZURE_ENDPOINT", originalAzureEndpoint)
				}
				if originalAzureDeployment == "" {
					os.Unsetenv("GRAPHCHAIN_AZURE_DEPLOYMENT")
				} else {
					os.Setenv("GRAPHCHAIN_AZURE_DEPLOYMENT", originalAzureDeployment)
				}
			})

			// Set up test environment
			os.Setenv("GRAPHCHAIN_LLM_PROVIDER", tc.provider)
			if tc.graphchainAPIKey != "" {
				os.Setenv("GRAPHCHAIN_API_KEY", tc.graphchainAPIKey)
			} else {
				os.Unsetenv("GRAPHCHAIN_API_KEY")
			}
			os.Setenv(tc.legacyEnvVar, tc.legacyAPIKey)

			// For Azure OpenAI, set required fields
			if tc.provider == "azureopenai" {
				os.Setenv("GRAPHCHAIN_AZURE_ENDPOINT", "https://test.openai.azure.com")
				os.Setenv("GRAPHCHAIN_AZURE_DEPLOYMENT", "test-deployment")
			}

			// Load config
			config, err := LoadConfig("")
			require.NoError(t, err)

			// Verify API key
			assert.Equal(t, tc.expectedAPIKey, config.GraphChain.LLM.APIKey)
		})
	}
}

func TestLegacyEnvironmentVariableFallback_NoFallbackForOtherProviders(t *testing.T) {
	// Save original environment
	originalProvider := os.Getenv("GRAPHCHAIN_LLM_PROVIDER")
	originalGraphchain := os.Getenv("GRAPHCHAIN_API_KEY")
	originalOpenAI := os.Getenv("OPENAI_API_KEY")

	t.Cleanup(func() {
		if originalProvider == "" {
			os.Unsetenv("GRAPHCHAIN_LLM_PROVIDER")
		} else {
			os.Setenv("GRAPHCHAIN_LLM_PROVIDER", originalProvider)
		}
		if originalGraphchain == "" {
			os.Unsetenv("GRAPHCHAIN_API_KEY")
		} else {
			os.Setenv("GRAPHCHAIN_API_KEY", originalGraphchain)
		}
		if originalOpenAI == "" {
			os.Unsetenv("OPENAI_API_KEY")
		} else {
			os.Setenv("OPENAI_API_KEY", originalOpenAI)
		}
	})

	// Set up environment for Anthropic (should not use OPENAI_API_KEY fallback)
	os.Setenv("GRAPHCHAIN_LLM_PROVIDER", "anthropic")
	os.Unsetenv("GRAPHCHAIN_API_KEY")
	os.Setenv("OPENAI_API_KEY", "should-not-be-used")
	os.Setenv("GRAPHCHAIN_LLM_MODEL", "claude-3")

	// Load config - should fail validation because no API key for anthropic
	_, err := LoadConfig("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required for provider: anthropic")
}
