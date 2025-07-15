package graphchain

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the GraphChain agent configuration
type Config struct {
	GraphChain GraphChainConfig `yaml:"graphchain"`
}

// GraphChainConfig contains all GraphChain-related configuration
type GraphChainConfig struct {
	LLM      LLMConfig      `yaml:"llm"`
	Agent    AgentConfig    `yaml:"agent"`
	Security SecurityConfig `yaml:"security"`
	Context  ContextConfig  `yaml:"context"`
}

// LLMConfig contains LLM provider configuration
type LLMConfig struct {
	Provider string        `yaml:"provider"`
	Model    string        `yaml:"model"`
	APIKey   string        `yaml:"api_key"`
	BaseURL  string        `yaml:"base_url"`
	Timeout  time.Duration `yaml:"timeout"`
}

// AgentConfig contains agent-specific configuration
type AgentConfig struct {
	MaxIterations  int           `yaml:"max_iterations"`
	ToolTimeout    time.Duration `yaml:"tool_timeout"`
	EnableMemory   bool          `yaml:"enable_memory"`
	MemorySize     int           `yaml:"memory_size"`
	SmallModelMode bool          `yaml:"small_model_mode"` // 新增字段
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	EnableAudit        bool     `yaml:"enable_audit"`
	ReadOnlyMode       bool     `yaml:"read_only_mode"`
	MaxQueryComplexity int      `yaml:"max_query_complexity"`
	AllowedOperations  []string `yaml:"allowed_operations"`
}

// ContextConfig contains context management configuration
type ContextConfig struct {
	EnableAutoDiscovery bool          `yaml:"enable_auto_discovery"`
	UpdateInterval      time.Duration `yaml:"update_interval"`
	MaxContextSize      int           `yaml:"max_context_size"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}

	// Load from file if exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// Expand environment variables in config
		expandedData := os.ExpandEnv(string(data))

		if err := yaml.Unmarshal([]byte(expandedData), config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Set defaults if not configured
	setDefaults(config)

	// Override with environment variables
	overrideWithEnv(config)

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// DefaultConfig returns a default configuration with sensible defaults
func DefaultConfig() *Config {
	config := &Config{}
	setDefaults(config)
	return config
}

// setDefaults sets default values for configuration
func setDefaults(config *Config) {
	gc := &config.GraphChain

	// LLM defaults
	if gc.LLM.Provider == "" {
		gc.LLM.Provider = "openai"
	}
	if gc.LLM.Model == "" {
		gc.LLM.Model = "gpt-4"
	}
	if gc.LLM.Timeout == 0 {
		gc.LLM.Timeout = 30 * time.Second
	}

	// Agent defaults
	if gc.Agent.MaxIterations == 0 {
		gc.Agent.MaxIterations = 10
	}
	if gc.Agent.ToolTimeout == 0 {
		gc.Agent.ToolTimeout = 10 * time.Second
	}
	if gc.Agent.MemorySize == 0 {
		gc.Agent.MemorySize = 100
	}

	// Security defaults
	if gc.Security.MaxQueryComplexity == 0 {
		gc.Security.MaxQueryComplexity = 10
	}
	if len(gc.Security.AllowedOperations) == 0 {
		gc.Security.AllowedOperations = []string{"get", "scan", "prefix", "jsonquery", "search", "stats"}
	}

	// Context defaults
	if gc.Context.UpdateInterval == 0 {
		gc.Context.UpdateInterval = 5 * time.Minute
	}
	if gc.Context.MaxContextSize == 0 {
		gc.Context.MaxContextSize = 4096
	}
}

// overrideWithEnv overrides configuration with environment variables
func overrideWithEnv(config *Config) {
	gc := &config.GraphChain

	// LLM environment overrides
	if provider := os.Getenv("GRAPHCHAIN_LLM_PROVIDER"); provider != "" {
		gc.LLM.Provider = provider
	}
	if model := os.Getenv("GRAPHCHAIN_LLM_MODEL"); model != "" {
		gc.LLM.Model = model
	}
	if apiKey := os.Getenv("GRAPHCHAIN_API_KEY"); apiKey != "" {
		gc.LLM.APIKey = apiKey
	}
	if baseURL := os.Getenv("GRAPHCHAIN_BASE_URL"); baseURL != "" {
		gc.LLM.BaseURL = baseURL
	}

	// Security environment overrides
	if readOnly := os.Getenv("GRAPHCHAIN_READ_ONLY"); readOnly != "" {
		gc.Security.ReadOnlyMode = strings.ToLower(readOnly) == "true"
	}
	if enableAudit := os.Getenv("GRAPHCHAIN_ENABLE_AUDIT"); enableAudit != "" {
		gc.Security.EnableAudit = strings.ToLower(enableAudit) == "true"
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	gc := &config.GraphChain

	// Validate LLM configuration
	if gc.LLM.Provider == "" {
		return fmt.Errorf("LLM provider is required")
	}
	if gc.LLM.Model == "" {
		return fmt.Errorf("LLM model is required")
	}

	// Validate supported providers
	supportedProviders := []string{"openai", "googleai", "ollama", "anthropic", "local"}
	if !contains(supportedProviders, gc.LLM.Provider) {
		return fmt.Errorf("unsupported LLM provider: %s, supported: %v", gc.LLM.Provider, supportedProviders)
	}

	// Validate API key for cloud providers (ollama and local don't need API keys)
	requiresAPIKey := []string{"openai", "googleai", "anthropic"}
	if contains(requiresAPIKey, gc.LLM.Provider) && gc.LLM.APIKey == "" {
		return fmt.Errorf("API key is required for provider: %s", gc.LLM.Provider)
	}

	// Validate numeric constraints
	if gc.Agent.MaxIterations <= 0 {
		return fmt.Errorf("agent max_iterations must be positive")
	}
	if gc.Security.MaxQueryComplexity <= 0 {
		return fmt.Errorf("security max_query_complexity must be positive")
	}
	if gc.Context.MaxContextSize <= 0 {
		return fmt.Errorf("context max_context_size must be positive")
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
