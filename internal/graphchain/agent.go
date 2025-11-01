package graphchain

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/service"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeTool       ErrorType = "tool_error"
	ErrorTypeMemory     ErrorType = "memory_error"
	ErrorTypeLLM        ErrorType = "llm_error"
	ErrorTypeValidation ErrorType = "validation_error"
)

// TimeoutConfig holds timeout configurations for different operations
type TimeoutConfig struct {
	QueryTimeout  time.Duration `json:"query_timeout"`
	ToolTimeout   time.Duration `json:"tool_timeout"`
	MemoryTimeout time.Duration `json:"memory_timeout"`
}

// ModelCapability represents the capability level of a model
type ModelCapability int

const (
	CapabilitySmall ModelCapability = iota
	CapabilityMedium
	CapabilityLarge
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
	config           *Config
	llm              llms.Model
	executor         ExecutorInterface
	tools            []tools.Tool
	database         *db.DB
	memory           *ConversationMemory
	capability       ModelCapability
	timeouts         TimeoutConfig
	queryCache       *QueryCache
	intentClassifier *IntentClassifier
}

// QueryResult represents the result of a query execution
type QueryResult struct {
	Success        bool          `json:"success"`
	Data           interface{}   `json:"data,omitempty"`
	Error          string        `json:"error,omitempty"`
	ErrorType      ErrorType     `json:"error_type,omitempty"`
	Explanation    string        `json:"explanation,omitempty"`
	ExecutionTime  time.Duration `json:"execution_time"`
	ToolsUsed      []string      `json:"tools_used,omitempty"`
	IntentDetected string        `json:"intent_detected,omitempty"`
}

// NewAgent creates a new GraphChain agent instance
func NewAgent(database *db.DB) *Agent {
	return &Agent{
		database:         database,
		queryCache:       NewQueryCache(100), // Cache last 100 queries
		intentClassifier: NewIntentClassifier(),
	}
}

// Initialize sets up the agent with configuration
func (a *Agent) Initialize(ctx context.Context, config *Config) error {
	a.config = config

	// Set up timeouts
	a.timeouts = TimeoutConfig{
		QueryTimeout:  config.GraphChain.LLM.Timeout,
		ToolTimeout:   time.Duration(float64(config.GraphChain.LLM.Timeout) * 0.7), // 70% of query timeout
		MemoryTimeout: time.Second * 5,
	}

	// Initialize LLM
	var err error
	a.llm, err = a.initializeLLM(config)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	// Determine model capability
	a.capability = a.determineModelCapability(config.GraphChain.LLM.Model)

	// Initialize memory if enabled
	if config.GraphChain.Agent.EnableMemory {
		a.memory = NewConversationMemory(config.GraphChain.Agent.MemorySize)
		a.memory.SetReturnMessages(false) // Use string format for LLM input
	}

	// Create database tools
	a.tools = a.createDatabaseTools()

	// Initialize agent executor based on capability
	a.executor, err = a.initializeExecutor()
	if err != nil {
		return fmt.Errorf("failed to initialize agent executor: %w", err)
	}

	return nil
}

// determineModelCapability determines the capability level of the model
func (a *Agent) determineModelCapability(modelName string) ModelCapability {
	modelName = strings.ToLower(modelName)

	// Small models
	smallPatterns := []string{
		"7b", "mini", "tiny", "phi", "small", "lite",
		"llama2-7b", "code-llama-7b", "mistral-7b",
	}

	// Large models
	largePatterns := []string{
		"70b", "65b", "175b", "gpt-4", "claude-3", "large",
		"llama2-70b", "code-llama-34b",
	}

	for _, pattern := range smallPatterns {
		if strings.Contains(modelName, pattern) {
			return CapabilitySmall
		}
	}

	for _, pattern := range largePatterns {
		if strings.Contains(modelName, pattern) {
			return CapabilityLarge
		}
	}

	return CapabilityMedium
}

// initializeExecutor initializes the agent executor based on model capability
func (a *Agent) initializeExecutor() (ExecutorInterface, error) {
	maxIterations := a.config.GraphChain.Agent.MaxIterations

	// Adjust iterations based on capability
	switch a.capability {
	case CapabilitySmall:
		maxIterations = min(maxIterations, 3) // Limit iterations for small models
	case CapabilityMedium:
		maxIterations = min(maxIterations, 5)
	}

	return agents.Initialize(
		a.llm,
		a.tools,
		agents.ZeroShotReactDescription,
		agents.WithMaxIterations(maxIterations),
		agents.WithReturnIntermediateSteps(),
	)
}

// createDatabaseTools creates all database tools with standard tools.Tool interface
func (a *Agent) createDatabaseTools() []tools.Tool {
	// Create services
	dbService := service.NewDatabaseService(a.database)
	scanService := service.NewScanService(a.database)
	searchService := service.NewSearchService(a.database)
	statsService := service.NewStatsService(a.database)

	return []tools.Tool{
		NewGetValueTool(dbService),
		NewPutValueTool(dbService),
		NewScanRangeTool(scanService),
		NewPrefixScanTool(scanService),
		NewListColumnFamiliesTool(dbService),
		NewGetLastTool(dbService),
		NewJSONQueryTool(searchService),
		NewGetStatsTool(statsService),
		NewSearchTool(searchService),
	}
}

// ProcessQuery processes a natural language query using the agent
func (a *Agent) ProcessQuery(ctx context.Context, query string) (*QueryResult, error) {
	startTime := time.Now()

	// Check cache first
	if cached := a.queryCache.Get(query); cached != nil {
		cached.ExecutionTime = time.Since(startTime)
		return cached, nil
	}

	// Classify intent
	intent := a.intentClassifier.ClassifyIntent(query)

	// Create a timeout context for the query
	timeoutCtx, cancel := context.WithTimeout(ctx, a.timeouts.QueryTimeout)
	defer cancel()

	// Try rule-based approach first for simple queries
	if simpleResult := a.tryRuleBasedProcessing(timeoutCtx, query, intent); simpleResult != nil {
		simpleResult.ExecutionTime = time.Since(startTime)
		simpleResult.IntentDetected = intent
		a.queryCache.Set(query, simpleResult)
		return simpleResult, nil
	}

	// Use LLM-based approach
	result, err := a.processWithLLM(timeoutCtx, query, intent, startTime)
	if err != nil {
		return result, err
	}

	// Cache successful results
	if result.Success {
		a.queryCache.Set(query, result)
	}

	return result, nil
}

// tryRuleBasedProcessing attempts to process simple queries using rules
func (a *Agent) tryRuleBasedProcessing(ctx context.Context, query string, intent string) *QueryResult {
	query = strings.TrimSpace(strings.ToLower(query))

	// Pattern matching for common operations
	patterns := map[string]func(string) *QueryResult{
		`^get\s+(.+)$`:                a.handleSimpleGet,
		`^show\s+all\s+keys?$`:        a.handleShowAllKeys,
		`^list\s+column\s+families?$`: a.handleListColumnFamilies,
		`^scan\s+prefix\s+(.+)$`:      a.handlePrefixScan,
		`^stats?$`:                    a.handleGetStats,
	}

	for pattern, handler := range patterns {
		if matched, _ := regexp.MatchString(pattern, query); matched {
			return handler(query)
		}
	}

	return nil
}

// Rule-based handlers
func (a *Agent) handleSimpleGet(query string) *QueryResult {
	re := regexp.MustCompile(`^get\s+(.+)$`)
	matches := re.FindStringSubmatch(query)
	if len(matches) > 1 {
		key := strings.TrimSpace(matches[1])
		if value, err := a.database.GetCF("default", key); err == nil {
			return &QueryResult{
				Success:     true,
				Data:        value,
				Explanation: fmt.Sprintf("Retrieved value for key: %s", key),
				ToolsUsed:   []string{"get_value"},
			}
		}
	}
	return nil
}

func (a *Agent) handleShowAllKeys(query string) *QueryResult {
	opts := db.ScanOptions{Values: true, Limit: 1000}
	if keys, err := a.database.ScanCF("default", nil, nil, opts); err == nil {
		return &QueryResult{
			Success:     true,
			Data:        keys,
			Explanation: "Retrieved all keys from database",
			ToolsUsed:   []string{"scan_range"},
		}
	}
	return nil
}

func (a *Agent) handleListColumnFamilies(query string) *QueryResult {
	if families, err := a.database.ListCFs(); err == nil {
		return &QueryResult{
			Success:     true,
			Data:        families,
			Explanation: "Listed all column families",
			ToolsUsed:   []string{"list_column_families"},
		}
	}
	return nil
}

func (a *Agent) handlePrefixScan(query string) *QueryResult {
	re := regexp.MustCompile(`^scan\s+prefix\s+(.+)$`)
	matches := re.FindStringSubmatch(query)
	if len(matches) > 1 {
		prefix := strings.TrimSpace(matches[1])
		if results, err := a.database.PrefixScanCF("default", prefix, 100); err == nil {
			return &QueryResult{
				Success:     true,
				Data:        results,
				Explanation: fmt.Sprintf("Scanned keys with prefix: %s", prefix),
				ToolsUsed:   []string{"prefix_scan"},
			}
		}
	}
	return nil
}

func (a *Agent) handleGetStats(query string) *QueryResult {
	if stats, err := a.database.GetDatabaseStats(); err == nil {
		return &QueryResult{
			Success:     true,
			Data:        stats,
			Explanation: "Retrieved database statistics",
			ToolsUsed:   []string{"get_stats"},
		}
	}
	return nil
}

// processWithLLM processes the query using the LLM with direct function calling
func (a *Agent) processWithLLM(ctx context.Context, query string, intent string, startTime time.Time) (*QueryResult, error) {
	// Use direct LLM function calling instead of agent executor for better temperature control
	// This is specifically needed for GPT-5 which requires temperature=1.0

	// Build messages with comprehensive system prompt
	systemPrompt := `You are a RocksDB database assistant. Use the available functions to answer user questions.

INSTRUCTIONS:
1. Read the user's question carefully
2. Call the appropriate function with correct parameters
3. Use the function result to answer the user's question
4. Respond in the same language as the question

PARAMETER EXTRACTION:
- Extract column_family from patterns like "users中", "in users", "from users" → column_family="users"
- Extract limit from patterns like "前N个", "first N", "N keys" → limit=N
- Extract key names directly from the question
- If no column_family specified, use "default"

CRITICAL RULES:
- The column_family parameter must match exactly what the user specifies
- Function names are self-explanatory: use the one that matches the user's intent
- Call only ONE function unless the first result requires clarification
- Always include all required parameters

Example pattern matching:
"获取users中最后一条记录" → column_family="users", use get_last_entry_in_column_family
"前10个key from products" → column_family="products", limit=10, use scan_keys_in_range with start_key="" and end_key=""
"列出所有column families" → use list_column_families`

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, query),
	}

	// Convert tools to function definitions
	functionDefs := a.convertToolsToFunctions()

	var toolsUsed []string
	var finalResponse string
	maxIterations := 5

	// Function calling loop
	for i := 0; i < maxIterations; i++ {
		// Call LLM with functions and temperature=1.0
		response, err := a.llm.GenerateContent(ctx, messages,
			llms.WithTemperature(1.0),
			llms.WithFunctions(functionDefs),
		)

		if err != nil {
			executionTime := time.Since(startTime)
			return a.handleExecutionError(err, executionTime), nil
		}

		// Check if LLM wants to call a function
		if len(response.Choices) == 0 {
			break
		}

		choice := response.Choices[0]

		// If no function call, we're done
		if choice.FuncCall == nil {
			finalResponse = choice.Content
			break
		}

		// Execute the function call
		funcName := choice.FuncCall.Name
		toolsUsed = append(toolsUsed, funcName)

		funcResult, err := a.executeToolByName(ctx, funcName, choice.FuncCall.Arguments)
		if err != nil {
			funcResult = fmt.Sprintf("Error: %v", err)
		}

		// Add function result to messages
		messages = append(messages,
			llms.MessageContent{
				Role: llms.ChatMessageTypeAI,
				Parts: []llms.ContentPart{
					llms.ToolCall{
						ID:   fmt.Sprintf("call_%d", i),
						Type: "function",
						FunctionCall: &llms.FunctionCall{
							Name:      funcName,
							Arguments: choice.FuncCall.Arguments,
						},
					},
				},
			},
			llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{
					llms.ToolCallResponse{
						ToolCallID: fmt.Sprintf("call_%d", i),
						Name:       funcName,
						Content:    funcResult,
					},
				},
			},
		)
	}

	// If we didn't get a final response, ask the LLM to summarize based on tool results
	if finalResponse == "" && len(toolsUsed) > 0 {
		response, err := a.llm.GenerateContent(ctx, messages, llms.WithTemperature(1.0))
		if err == nil && len(response.Choices) > 0 {
			finalResponse = response.Choices[0].Content
		} else {
			finalResponse = "Successfully executed tools but could not generate final response"
		}
	}

	executionTime := time.Since(startTime)

	return &QueryResult{
		Success:        true,
		Data:           finalResponse,
		Explanation:    finalResponse,
		ExecutionTime:  executionTime,
		ToolsUsed:      toolsUsed,
		IntentDetected: intent,
	}, nil
}

// buildInputs builds the input map for the LLM based on model capability
func (a *Agent) buildInputs(ctx context.Context, query string, intent string) map[string]any {
	inputs := map[string]any{
		"input": query,
	}

	switch a.capability {
	case CapabilitySmall:
		inputs["input"] = a.buildSmallModelPrompt(query, intent)
	case CapabilityMedium:
		inputs["input"] = a.buildMediumModelPrompt(query, intent)
	default: // Large models
		inputs = a.buildLargeModelInputs(ctx, inputs, query, intent)
	}

	return inputs
}

// buildSmallModelPrompt builds a focused prompt for small models
func (a *Agent) buildSmallModelPrompt(query string, intent string) string {
	// Pre-select relevant tools based on intent
	relevantTools := a.getRelevantToolsForIntent(intent)
	toolDescriptions := a.getToolDescriptions(relevantTools)

	prompt := fmt.Sprintf(`You are a RocksDB assistant. Answer using ONE tool only.

Available tools:
%s

Examples:
- "get user:123" → use get_value with key="user:123"
- "show all keys" → use scan_range with start_key="", end_key=""
- "keys starting with user:" → use prefix_scan with prefix="user:"

Query: %s
Choose the best tool and provide exact parameters.`, toolDescriptions, query)

	// Add history if available and within token limit
	if a.memory != nil {
		history := a.memory.GetRecentHistory(3) // Last 3 turns
		historyStr := a.formatHistoryForPrompt(history)
		if historyStr != "" && len(historyStr) < 200 { // Keep it short
			prompt = fmt.Sprintf("Recent context: %s\n\n%s", historyStr, prompt)
		}
	}

	return prompt
}

// buildMediumModelPrompt builds a balanced prompt for medium models
func (a *Agent) buildMediumModelPrompt(query string, intent string) string {
	systemContext := fmt.Sprintf(`You are a RocksDB database assistant. Use your tools to answer the user's question.

Detected intent: %s

Tool selection guide:
- get_value: Retrieve specific key values
- scan_range: List keys in a range or all keys
- prefix_scan: Find keys starting with a prefix
- put_value: Store key-value pairs
- list_column_families: Show available column families
- json_query: Query JSON values
- get_stats: Show database statistics

User question: %s

Please select the appropriate tool and provide a clear explanation.`, intent, query)

	if a.memory != nil {
		if history := a.memory.GetRecentHistory(5); len(history) > 0 {
			historyStr := a.formatHistoryForPrompt(history)
			systemContext = fmt.Sprintf("Conversation history:\n%s\n\n%s", historyStr, systemContext)
		}
	}

	return systemContext
}

// buildLargeModelInputs builds comprehensive inputs for large models
func (a *Agent) buildLargeModelInputs(ctx context.Context, inputs map[string]any, query string, intent string) map[string]any {
	if a.memory != nil {
		memoryCtx, cancel := context.WithTimeout(ctx, a.timeouts.MemoryTimeout)
		defer cancel()

		if memoryVars, err := a.memory.LoadMemoryVariables(memoryCtx, inputs); err == nil {
			for key, value := range memoryVars {
				inputs[key] = value
			}

			if history, ok := memoryVars["history"].(string); ok && history != "" {
				enhancedQuery := a.buildQueryWithHistory(query, history, intent)
				inputs["input"] = enhancedQuery
				return inputs
			}
		}
	}

	inputs["input"] = a.buildEnhancedQuery(query, intent)
	return inputs
}

// Helper methods for tool selection and processing
func (a *Agent) getRelevantToolsForIntent(intent string) []tools.Tool {
	intentToTools := map[string][]string{
		"get_value":  {"get_value"},
		"scan_keys":  {"scan_range", "prefix_scan"},
		"store_data": {"put_value"},
		"list_cf":    {"list_column_families"},
		"query_json": {"json_query"},
		"get_stats":  {"get_stats"},
	}

	if toolNames, exists := intentToTools[intent]; exists {
		var relevantTools []tools.Tool
		for _, tool := range a.tools {
			for _, name := range toolNames {
				if tool.Name() == name {
					relevantTools = append(relevantTools, tool)
					break
				}
			}
		}
		return relevantTools
	}

	return a.tools // Return all tools if intent not recognized
}

func (a *Agent) getToolDescriptions(tools []tools.Tool) string {
	var descriptions []string
	for _, tool := range tools {
		descriptions = append(descriptions, fmt.Sprintf("- %s: %s", tool.Name(), tool.Description()))
	}
	return strings.Join(descriptions, "\n")
}

func (a *Agent) extractToolsUsed(intermediateSteps interface{}) []string {
	var toolsUsed []string
	if steps, ok := intermediateSteps.([]interface{}); ok {
		for _, step := range steps {
			if stepStr := fmt.Sprintf("%v", step); stepStr != "" {
				// Extract tool name from step (this is a simplified extraction)
				for _, tool := range a.tools {
					if strings.Contains(stepStr, tool.Name()) {
						toolsUsed = append(toolsUsed, tool.Name())
						break
					}
				}
			}
		}
	}
	return toolsUsed
}

func (a *Agent) printReasoningTrace(intermediateSteps interface{}) {
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

func (a *Agent) handleExecutionError(err error, executionTime time.Duration) *QueryResult {
	if err != nil && strings.Contains(err.Error(), "context deadline exceeded") {
		return &QueryResult{
			Success:       false,
			Error:         fmt.Sprintf("Query timed out after %v. Try a simpler query or increase timeout.", a.timeouts.QueryTimeout),
			ErrorType:     ErrorTypeTimeout,
			ExecutionTime: executionTime,
		}
	}

	// Classify other error types
	errorType := ErrorTypeLLM
	if err != nil && strings.Contains(err.Error(), "tool") {
		errorType = ErrorTypeTool
	}

	return &QueryResult{
		Success:       false,
		Error:         fmt.Sprintf("Query execution failed: %v", err),
		ErrorType:     errorType,
		ExecutionTime: executionTime,
	}
}

func (a *Agent) saveToMemory(ctx context.Context, inputs, outputs map[string]any) {
	memoryCtx, cancel := context.WithTimeout(ctx, a.timeouts.MemoryTimeout)
	defer cancel()

	if err := a.memory.SaveContext(memoryCtx, inputs, outputs); err != nil {
		fmt.Printf("Warning: Failed to save to memory: %v\n", err)
	}
}

// buildQueryWithHistory enhances the user query with conversation history
func (a *Agent) buildQueryWithHistory(query, history, intent string) string {
	return fmt.Sprintf(`You are a RocksDB database assistant. Use your available tools to answer the user's question.

Detected intent: %s

Conversation history:
%s

Current user question: %s

Please use the appropriate tool to answer the current question based on the context of the conversation history, and provide a clear explanation.`, intent, history, query)
}

// buildEnhancedQuery enhances the user query with database context
func (a *Agent) buildEnhancedQuery(query, intent string) string {
	return fmt.Sprintf(`You are a RocksDB database assistant. Use your available tools to answer the user's question.

Detected intent: %s

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

User question: %s

Please use the appropriate tool to answer this question and provide a clear explanation.`, intent, query)
}

func (a *Agent) formatHistoryForPrompt(history []ConversationTurn) string {
	var formatted []string
	for _, turn := range history {
		formatted = append(formatted, fmt.Sprintf("User: %s\nAssistant: %s", turn.UserQuery, turn.AgentResponse))
	}
	return strings.Join(formatted, "\n")
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
		// Configure OpenAI client with optional BaseURL for OpenAI-compatible endpoints
		openaiOptions := []openai.Option{
			openai.WithModel(llmConfig.Model),
			openai.WithToken(llmConfig.APIKey),
		}
		if llmConfig.BaseURL != "" {
			openaiOptions = append(openaiOptions, openai.WithBaseURL(llmConfig.BaseURL))
		}
		return openai.New(openaiOptions...)
	case "azureopenai":
		// Configure Azure OpenAI client
		// Azure OpenAI endpoint format: https://{resource-name}.cognitiveservices.azure.com/
		// The SDK will handle the full path construction
		endpoint := strings.TrimSuffix(llmConfig.AzureEndpoint, "/")

		// Debug logging for Azure OpenAI configuration
		fmt.Printf("🔧 Azure OpenAI Configuration:\n")
		fmt.Printf("   Endpoint: %s\n", llmConfig.AzureEndpoint)
		fmt.Printf("   Deployment: %s\n", llmConfig.AzureDeployment)
		fmt.Printf("   API Version: %s\n", llmConfig.AzureAPIVersion)
		fmt.Printf("   API Key: %s...%s\n", llmConfig.APIKey[:min(10, len(llmConfig.APIKey))], llmConfig.APIKey[max(0, len(llmConfig.APIKey)-4):])

		openaiOptions := []openai.Option{
			openai.WithModel(llmConfig.AzureDeployment), // Use deployment name as model
			openai.WithToken(llmConfig.APIKey),
			openai.WithBaseURL(endpoint),
			openai.WithAPIType(openai.APITypeAzure),
			openai.WithAPIVersion(llmConfig.AzureAPIVersion),
			// Note: Temperature is not set here to use default (1.0) as required by GPT-5
		}
		return openai.New(openaiOptions...)
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
		if llmConfig.BaseURL != "" {
			options = append(options, ollama.WithServerURL(llmConfig.BaseURL))
		}
		return ollama.New(options...)
	default:
		return openai.New(
			openai.WithModel(llmConfig.Model),
			openai.WithToken(llmConfig.APIKey),
		)
	}
}

// Additional getter methods
func (a *Agent) GetLangChainExecutor() ExecutorInterface {
	return a.executor
}

func (a *Agent) GetLLM() llms.Model {
	return a.llm
}

func (a *Agent) GetTools() []tools.Tool {
	return a.tools
}

func (a *Agent) GetMemory() *ConversationMemory {
	return a.memory
}

func (a *Agent) ClearMemory(ctx context.Context) error {
	if a.memory == nil {
		return fmt.Errorf("memory is not enabled")
	}
	return a.memory.Clear(ctx)
}

func (a *Agent) GetMemoryStats() *MemoryStats {
	if a.memory == nil {
		return nil
	}
	stats := a.memory.GetStats()
	return &stats
}

func (a *Agent) IsMemoryEnabled() bool {
	return a.memory != nil
}

func (a *Agent) GetConversationHistory(n int) []ConversationTurn {
	if a.memory == nil {
		return []ConversationTurn{}
	}
	return a.memory.GetRecentHistory(n)
}

func (a *Agent) GetModelCapability() ModelCapability {
	return a.capability
}

// convertToolsToFunctions converts tools to LLM function definitions
func (a *Agent) convertToolsToFunctions() []llms.FunctionDefinition {
	functions := make([]llms.FunctionDefinition, 0, len(a.tools))

	for _, tool := range a.tools {
		// Create proper parameter schema based on tool name
		params := a.getToolParameterSchema(tool.Name())

		funcDef := llms.FunctionDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  params,
		}
		functions = append(functions, funcDef)
	}

	return functions
}

// getToolParameterSchema returns the parameter schema for a specific tool
func (a *Agent) getToolParameterSchema(toolName string) map[string]any {
	switch toolName {
	case "list_column_families":
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}

	case "get_last_entry_in_column_family":
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"column_family": map[string]any{
					"type":        "string",
					"description": "The name of the column family to get the last entry from",
				},
			},
			"required": []string{},
		}

	case "scan_keys_in_range":
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"column_family": map[string]any{
					"type":        "string",
					"description": "The name of the column family to scan",
				},
				"start_key": map[string]any{
					"type":        "string",
					"description": "The start key (empty string for beginning)",
				},
				"end_key": map[string]any{
					"type":        "string",
					"description": "The end key (empty string for end)",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results to return",
				},
			},
			"required": []string{},
		}

	case "scan_keys_with_prefix":
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"column_family": map[string]any{
					"type":        "string",
					"description": "The name of the column family to scan",
				},
				"prefix": map[string]any{
					"type":        "string",
					"description": "The key prefix to search for",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results to return",
				},
			},
			"required": []string{"prefix"},
		}

	case "get_value_by_key":
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"column_family": map[string]any{
					"type":        "string",
					"description": "The name of the column family",
				},
				"key": map[string]any{
					"type":        "string",
					"description": "The key to retrieve",
				},
			},
			"required": []string{"key"},
		}

	case "get_database_stats":
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"column_family": map[string]any{
					"type":        "string",
					"description": "The name of the column family (optional)",
				},
			},
			"required": []string{},
		}

	case "query_json_field":
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"column_family": map[string]any{
					"type":        "string",
					"description": "The name of the column family",
				},
				"field": map[string]any{
					"type":        "string",
					"description": "The JSON field name to query",
				},
				"value": map[string]any{
					"type":        "string",
					"description": "The value to search for",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results",
				},
			},
			"required": []string{"field", "value"},
		}

	case "search_keys_and_values":
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"column_family": map[string]any{
					"type":        "string",
					"description": "The name of the column family",
				},
				"key_pattern": map[string]any{
					"type":        "string",
					"description": "Pattern to search in keys",
				},
				"value_pattern": map[string]any{
					"type":        "string",
					"description": "Pattern to search in values",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results",
				},
			},
			"required": []string{},
		}

	default:
		// Fallback to generic input parameter
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type":        "string",
					"description": "The input for the tool",
				},
			},
			"required": []string{"input"},
		}
	}
}

// executeToolByName executes a tool by name with the given arguments
func (a *Agent) executeToolByName(ctx context.Context, toolName string, arguments string) (string, error) {
	// Find the tool
	var targetTool tools.Tool
	for _, tool := range a.tools {
		if tool.Name() == toolName {
			targetTool = tool
			break
		}
	}

	if targetTool == nil {
		return "", fmt.Errorf("tool not found: %s", toolName)
	}

	// Parse arguments from GPT
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Convert structured parameters to tool input format
	// Tools expect: {"args": {"column_family": "users", "limit": 10}}
	var toolInput string
	if input, ok := args["input"].(string); ok {
		// Legacy format: {"input": "some string"}
		toolInput = input
	} else {
		// New structured format: {"column_family": "users", "limit": 10}
		// Convert to tool expected format: {"args": {...}}
		toolInputMap := map[string]interface{}{
			"args": args,
		}
		toolInputBytes, err := json.Marshal(toolInputMap)
		if err != nil {
			return "", fmt.Errorf("failed to marshal tool input: %w", err)
		}
		toolInput = string(toolInputBytes)
	}

	// Call the tool
	result, err := targetTool.Call(ctx, toolInput)
	if err != nil {
		return "", err
	}

	return result, nil
}
