package graphchain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rocksdb-cli/internal/db"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

// GraphChainAgent interface defines the contract for our database agent
type GraphChainAgent interface {
	// Initialize sets up the agent with configuration
	Initialize(ctx context.Context, config *Config) error

	// ProcessQuery processes a natural language query and returns results
	ProcessQuery(ctx context.Context, query string) (*QueryResult, error)

	// GetCapabilities returns the list of available capabilities
	GetCapabilities() []string

	// Close cleans up resources
	Close() error
}

// Agent implements the GraphChainAgent interface using langchaingo
type Agent struct {
	config   *Config
	llm      llms.Model
	executor *agents.Executor
	tools    []tools.Tool
	database *db.DB
	memory   *ConversationMemory
}

// QueryResult represents the result of a query execution
type QueryResult struct {
	Success       bool          `json:"success"`
	Data          interface{}   `json:"data,omitempty"`
	Error         string        `json:"error,omitempty"`
	Explanation   string        `json:"explanation,omitempty"`
	ExecutionTime time.Duration `json:"execution_time"`
}

// NewAgent creates a new GraphChain agent instance
func NewAgent(database *db.DB) *Agent {
	return &Agent{
		database: database,
	}
}

// Initialize sets up the agent with configuration
func (a *Agent) Initialize(ctx context.Context, config *Config) error {
	a.config = config

	// Initialize LLM
	var err error
	a.llm, err = a.initializeLLM(config)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	// Initialize memory if enabled
	if config.GraphChain.Agent.EnableMemory {
		a.memory = NewConversationMemory(config.GraphChain.Agent.MemorySize)
		a.memory.SetReturnMessages(true) // Use chat message format
	}

	// Create database tools
	a.tools = a.createDatabaseTools()

	// Initialize agent executor
	a.executor, err = agents.Initialize(
		a.llm,
		a.tools,
		agents.ZeroShotReactDescription,
		agents.WithMaxIterations(config.GraphChain.Agent.MaxIterations),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize agent executor: %w", err)
	}

	return nil
}

// createDatabaseTools creates all database tools with standard tools.Tool interface
func (a *Agent) createDatabaseTools() []tools.Tool {
	return []tools.Tool{
		NewGetValueTool(a.database),
		NewPutValueTool(a.database),
		NewScanRangeTool(a.database),
		NewPrefixScanTool(a.database),
		NewListColumnFamiliesTool(a.database),
		NewGetLastTool(a.database),
		NewJSONQueryTool(a.database),
		NewGetStatsTool(a.database),
	}
}

// ProcessQuery processes a natural language query using the agent
func (a *Agent) ProcessQuery(ctx context.Context, query string) (*QueryResult, error) {
	startTime := time.Now()

	// Create a timeout context for the query
	timeoutCtx, cancel := context.WithTimeout(ctx, a.config.GraphChain.LLM.Timeout)
	defer cancel()

	// Build inputs with memory context if enabled
	inputs := map[string]any{
		"input": query,
	}

	// Load memory context if memory is enabled
	if a.memory != nil {
		memoryVars, err := a.memory.LoadMemoryVariables(ctx, inputs)
		if err != nil {
			return &QueryResult{
				Success:       false,
				Error:         fmt.Sprintf("Failed to load memory: %v", err),
				ExecutionTime: time.Since(startTime),
			}, nil
		}

		// Add memory variables to inputs
		for key, value := range memoryVars {
			inputs[key] = value
		}

		// Enhance query with conversation history if available
		if history, ok := memoryVars["history"].(string); ok && history != "" {
			enhancedQuery := a.buildQueryWithHistory(query, history)
			inputs["input"] = enhancedQuery
		}
	} else {
		// Fallback to simple enhanced query
		enhancedQuery := a.buildEnhancedQuery(query)
		inputs["input"] = enhancedQuery
	}

	// Execute the query using agent executor
	result, err := a.executor.Call(timeoutCtx, inputs)
	executionTime := time.Since(startTime)

	if err != nil {
		// Check if it was a timeout error
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return &QueryResult{
				Success:       false,
				Error:         fmt.Sprintf("Query timed out after %v. Try a simpler query or increase timeout.", a.config.GraphChain.LLM.Timeout),
				ExecutionTime: executionTime,
			}, nil
		}
		return &QueryResult{
			Success:       false,
			Error:         fmt.Sprintf("Query execution failed: %v", err),
			ExecutionTime: executionTime,
		}, nil
	}

	// Extract output from result if it's a map
	var finalResult interface{} = result
	if output, exists := result["output"]; exists {
		finalResult = output
	}

	// Save conversation to memory if enabled
	if a.memory != nil {
		outputs := map[string]any{
			"output":         finalResult,
			"execution_time": executionTime,
		}
		if err := a.memory.SaveContext(ctx, inputs, outputs); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: Failed to save to memory: %v\n", err)
		}
	}

	return &QueryResult{
		Success:       true,
		Data:          finalResult,
		Explanation:   "Query executed successfully using database tools",
		ExecutionTime: executionTime,
	}, nil
}

// buildQueryWithHistory enhances the user query with conversation history
func (a *Agent) buildQueryWithHistory(query, history string) string {
	// Include conversation history in the system context
	systemContext := fmt.Sprintf(`你是一个 RocksDB 数据库助手。请根据用户的问题，使用你可用的工具来回答。

对话历史：
%s

当前用户问题：%s

请根据对话历史的上下文，使用合适的工具回答当前问题，并提供清晰的解释。如果用户的问题涉及之前讨论的内容，请考虑历史上下文。`, history, query)

	return systemContext
}

// buildEnhancedQuery enhances the user query with database context
func (a *Agent) buildEnhancedQuery(query string) string {
	// Build simple system context - let agents.Executor handle tool information
	systemContext := fmt.Sprintf(`你是一个 RocksDB 数据库助手。请根据用户的问题，使用你可用的工具来回答。

用户问题：%s

请使用合适的工具回答这个问题，并提供清晰的解释。`, query)

	return systemContext
}

// GetCapabilities returns the list of available capabilities
func (a *Agent) GetCapabilities() []string {
	capabilities := make([]string, len(a.tools))
	for i, tool := range a.tools {
		capabilities[i] = tool.Name()
	}
	return capabilities
}

// Close cleans up resources
func (a *Agent) Close() error {
	return nil
}

// initializeLLM initializes the appropriate LLM based on configuration
func (a *Agent) initializeLLM(config *Config) (llms.Model, error) {
	llmConfig := &config.GraphChain.LLM

	switch strings.ToLower(llmConfig.Provider) {
	case "openai":
		return openai.New(
			openai.WithModel(llmConfig.Model),
			openai.WithToken(llmConfig.APIKey),
		)
	case "googleai", "google":
		return googleai.New(
			context.Background(),
			googleai.WithAPIKey(llmConfig.APIKey),
			googleai.WithDefaultModel(llmConfig.Model),
		)
	case "ollama":
		options := []ollama.Option{
			ollama.WithModel(llmConfig.Model),
		}
		// Add server URL if specified
		if llmConfig.BaseURL != "" {
			options = append(options, ollama.WithServerURL(llmConfig.BaseURL))
		}
		return ollama.New(options...)
	default:
		// Default to OpenAI if provider is not recognized
		return openai.New(
			openai.WithModel(llmConfig.Model),
			openai.WithToken(llmConfig.APIKey),
		)
	}
}

// GetLangChainExecutor returns the underlying langchaingo executor for advanced usage
func (a *Agent) GetLangChainExecutor() *agents.Executor {
	return a.executor
}

// GetLLM returns the underlying LLM for direct usage
func (a *Agent) GetLLM() llms.Model {
	return a.llm
}

// GetTools returns the tools used by the agent
func (a *Agent) GetTools() []tools.Tool {
	return a.tools
}

// GetMemory returns the conversation memory (if enabled)
func (a *Agent) GetMemory() *ConversationMemory {
	return a.memory
}

// ClearMemory clears the conversation history
func (a *Agent) ClearMemory(ctx context.Context) error {
	if a.memory == nil {
		return fmt.Errorf("memory is not enabled")
	}
	return a.memory.Clear(ctx)
}

// GetMemoryStats returns memory usage statistics
func (a *Agent) GetMemoryStats() *MemoryStats {
	if a.memory == nil {
		return nil
	}
	stats := a.memory.GetStats()
	return &stats
}

// IsMemoryEnabled returns whether memory is enabled for this agent
func (a *Agent) IsMemoryEnabled() bool {
	return a.memory != nil
}

// GetConversationHistory returns the recent conversation history
func (a *Agent) GetConversationHistory(n int) []ConversationTurn {
	if a.memory == nil {
		return []ConversationTurn{}
	}
	return a.memory.GetRecentHistory(n)
}
