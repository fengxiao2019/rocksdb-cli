package graphchain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rocksdb-cli/internal/db"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
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

// ExecutorInterface contains only the Call method for easy mocking
type ExecutorInterface interface {
	Call(ctx context.Context, inputs map[string]any, opts ...chains.ChainCallOption) (map[string]any, error)
}

// Agent implements the GraphChainAgent interface using langchaingo
type Agent struct {
	config         *Config
	llm            llms.Model
	executor       ExecutorInterface
	tools          []tools.Tool
	database       *db.DB
	memory         *ConversationMemory
	SmallModelMode bool // New field for small model mode
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
		a.memory.SetReturnMessages(false) // Use string format for LLM input
	}

	// Create database tools
	a.tools = a.createDatabaseTools()

	// Automatically determine small model mode (e.g., model name contains "mini", "tiny", "llama2-7b", "phi" etc.)
	a.SmallModelMode = config.GraphChain.Agent.SmallModelMode
	modelName := strings.ToLower(config.GraphChain.LLM.Model)
	if !a.SmallModelMode && (strings.Contains(modelName, "mini") || strings.Contains(modelName, "tiny") || strings.Contains(modelName, "7b") || strings.Contains(modelName, "phi") || strings.Contains(modelName, "small")) {
		a.SmallModelMode = true
	}

	// Initialize agent executor
	a.executor, err = agents.Initialize(
		a.llm,
		a.tools,
		agents.ZeroShotReactDescription,
		agents.WithMaxIterations(config.GraphChain.Agent.MaxIterations),
		agents.WithReturnIntermediateSteps(),
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

	inputs := map[string]any{
		"input": query,
	}

	// In small model mode, use a minimal prompt and token limit
	if a.SmallModelMode {
		tokenLimit := 512 // Can be adjusted based on model type
		if strings.Contains(strings.ToLower(a.config.GraphChain.LLM.Model), "7b") {
			tokenLimit = 2048
		}
		prompt := a.BuildSmallModelPrompt(query, tokenLimit, a.config.GraphChain.LLM.Model)
		inputs["input"] = prompt
	} else {
		// Original flow
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

	// Print reasoning trace if available
	if intermediateSteps, exists := result["intermediate_steps"]; exists {
		fmt.Println("\n=== Reasoning Trace ===")
		if steps, ok := intermediateSteps.([]interface{}); ok {
			for i, step := range steps {
				fmt.Printf("Step %d: %v\n", i+1, step)
			}
		} else {
			fmt.Printf("Intermediate Steps: %v\n", intermediateSteps)
		}
		fmt.Println("=== End Reasoning Trace ===")
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
	systemContext := fmt.Sprintf(`You are a RocksDB database assistant. Please use your available tools to answer the user's question.

Conversation history:
%s

Current user question: %s

Please use the appropriate tool to answer the current question based on the context of the conversation history, and provide a clear explanation. If the user's question involves previously discussed content, please consider the historical context.`, history, query)

	return systemContext
}

// buildEnhancedQuery enhances the user query with database context
func (a *Agent) buildEnhancedQuery(query string) string {
	// Build enhanced system context with few-shot examples and anti-tokenization instruction
	systemContext := `You are a RocksDB database assistant. Please use your available tools to answer the user's question.

[Tool selection examples]
User question: Show all keys in users
Should choose tool: scan_range, params start_key="", end_key="", column_family="users"

User question: Show all keys starting with user:
Should choose tool: prefix_scan, params prefix="user:"

User question: Get the value for key user:123
Should choose tool: get_value, params key="user:123"

[Important]
- Please understand the user's question as a whole. Do not split the input into individual words for separate processing.
- Only select the most appropriate tool and parameters based on the overall meaning of the question.

User question: ` + query + `

Please use the appropriate tool to answer this question and provide a clear explanation.`
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

// GetLangChainExecutor returns the underlying executor for advanced usage
func (a *Agent) GetLangChainExecutor() ExecutorInterface {
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

// BuildSmallModelPrompt builds a minimal prompt for small models, automatically trimming history to fit the token limit
func (a *Agent) BuildSmallModelPrompt(query string, tokenLimit int, model string) string {
	var historyStr string
	if a.memory != nil {
		historyStr = a.memory.GetHistoryByTokenLimit(tokenLimit/2, model) // Reserve half tokens for history
	}
	// If history still exceeds limit, use summary
	if EstimateTokenCount(historyStr, model) > tokenLimit/2 {
		historyStr = SummarizeHistory(historyStr, model)
	}
	prompt := "You are a RocksDB assistant. Please answer as concisely as possible.\n"
	if historyStr != "" {
		prompt += historyStr + "\n"
	}
	prompt += "User question: " + query
	return prompt
}
