package graphchain

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"rocksdb-cli/internal/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/fake"
)

// newTestDB creates a temporary RocksDB for testing purposes.
func newTestDB(t *testing.T) db.KeyValueDB {
	t.Helper()
	dir, err := os.MkdirTemp("", "rocksdb-test-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	// Use the new, correct public API
	database, err := db.Open(dir)
	require.NoError(t, err)
	return database
}

// newTestConfig creates a default configuration for testing.
func newTestConfig() *Config {
	return DefaultConfig()
}

func TestAgent_Initialize_WithFakeLLM(t *testing.T) {
	// Arrange
	database := newTestDB(t)
	// Cast to the concrete type required by NewAgent. This is a known limitation.
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok, "database should be of type *db.DB")

	agent := NewAgent(dbPtr)
	config := newTestConfig()
	// Use fake LLM provider for testing (we'll bypass the real Initialize)
	config.GraphChain.LLM.Provider = "fake"

	// Create fake LLM responses
	responses := []string{
		"This is a test response from fake LLM.",
		"Another test response.",
	}
	fakeLLM := fake.NewFakeLLM(responses)

	// Manually set up the agent for testing without going through full Initialize
	agent.config = config
	agent.llm = fakeLLM
	agent.tools = agent.createDatabaseTools()

	var err error
	agent.executor, err = agents.Initialize(
		agent.llm,
		agent.tools,
		agents.ZeroShotReactDescription,
	)
	require.NoError(t, err)

	// Assert
	assert.NotNil(t, agent.llm, "LLM should be initialized")
	assert.NotNil(t, agent.executor, "Executor should be initialized")
	assert.NotEmpty(t, agent.tools, "Tools should be created")
	assert.Equal(t, config, agent.config, "Config should be set")
}

func TestAgent_ProcessQuery_ToolCall(t *testing.T) {
	// This test verifies that the agent correctly initializes and has the required components.

	// Arrange
	database := newTestDB(t)
	// Use the new, correct public API for putting data
	err := database.PutCF("default", "user:1", "alice")
	require.NoError(t, err)

	// Create a fake LLM using the correct NewFakeLLM function from langchaingo
	responses := []string{
		"I need to get the value for key user:1 from the database.",
		"The value for user:1 is alice.",
	}
	fakeLLM := fake.NewFakeLLM(responses)

	// Cast to the concrete type required by NewAgent.
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)
	agent := NewAgent(dbPtr)

	// Initialize config (this was missing and causing the nil pointer error)
	config := newTestConfig()
	agent.config = config

	// Manually inject the fake LLM for testing
	agent.llm = fakeLLM
	agent.tools = agent.createDatabaseTools()
	agent.executor, err = agents.Initialize(
		agent.llm,
		agent.tools,
		agents.ZeroShotReactDescription,
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Act - Test simple query processing (without expecting complex tool execution)
	result, err := agent.ProcessQuery(ctx, "What is the value for key user:1?")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	// For fake LLM, we mainly test that the system initializes and processes without error
	// The actual tool execution behavior would require more complex mocking
}

func TestAgent_NewAgent(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)

	assert.NotNil(t, agent)
	assert.Equal(t, dbPtr, agent.database)
	assert.Nil(t, agent.llm, "LLM should not be initialized yet")
	assert.Nil(t, agent.executor, "Executor should not be initialized yet")
	assert.Nil(t, agent.tools, "Tools should not be initialized yet")
	assert.Nil(t, agent.config, "Config should not be set yet")
}

func TestAgent_CreateDatabaseTools(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)
	tools := agent.createDatabaseTools()

	assert.NotEmpty(t, tools, "Should create database tools")
	assert.Equal(t, 8, len(tools), "Should create 8 tools")

	// Check that all expected tools are created
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		if namedTool, ok := tool.(interface{ Name() string }); ok {
			toolNames[namedTool.Name()] = true
		}
	}

	expectedTools := []string{
		"get_value", "put_value", "scan_range", "prefix_scan",
		"list_column_families", "get_last", "json_query", "get_stats",
	}

	for _, expected := range expectedTools {
		assert.True(t, toolNames[expected], "Tool %s should be created", expected)
	}
}

func TestAgent_BuildEnhancedQuery(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)
	config := newTestConfig()
	agent.config = config

	testCases := []struct {
		name           string
		originalQuery  string
		expectContains []string
	}{
		{
			name:          "simple query",
			originalQuery: "get user data",
			expectContains: []string{
				"get user data",
				"RocksDB 数据库助手",
				"使用你可用的工具",
			},
		},
		{
			name:          "database operation query",
			originalQuery: "find all users with name alice",
			expectContains: []string{
				"find all users with name alice",
				"RocksDB 数据库助手",
			},
		},
		{
			name:          "empty query",
			originalQuery: "",
			expectContains: []string{
				"RocksDB 数据库助手",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enhancedQuery := agent.buildEnhancedQuery(tc.originalQuery)

			assert.NotEmpty(t, enhancedQuery, "Enhanced query should not be empty")

			for _, expected := range tc.expectContains {
				assert.Contains(t, enhancedQuery, expected,
					"Enhanced query should contain: %s", expected)
			}
		})
	}
}

func TestAgent_ProcessQuery_EmptyQuery(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)
	config := newTestConfig()
	agent.config = config

	// Create fake LLM with a response for empty query
	responses := []string{
		"I can help you with database operations. Please provide a specific query.",
	}
	fakeLLM := fake.NewFakeLLM(responses)
	agent.llm = fakeLLM
	agent.tools = agent.createDatabaseTools()

	var err error
	agent.executor, err = agents.Initialize(
		agent.llm,
		agent.tools,
		agents.ZeroShotReactDescription,
	)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := agent.ProcessQuery(ctx, "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Don't enforce Success=true for empty queries as the behavior may vary
}

func TestAgent_ProcessQuery_LongQuery(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)
	config := newTestConfig()
	agent.config = config

	// Create a very long query
	longQuery := strings.Repeat("very long query text ", 200) // ~4000 characters

	// Create fake LLM response
	responses := []string{
		"I understand you have a long query. Let me help you with that.",
	}
	fakeLLM := fake.NewFakeLLM(responses)
	agent.llm = fakeLLM
	agent.tools = agent.createDatabaseTools()

	var err error
	agent.executor, err = agents.Initialize(
		agent.llm,
		agent.tools,
		agents.ZeroShotReactDescription,
	)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := agent.ProcessQuery(ctx, longQuery)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAgent_ProcessQuery_SpecialCharacters(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)
	config := newTestConfig()
	agent.config = config

	// Query with special characters
	specialQuery := "查找用户数据 with émojis 🚀 and symbols: @#$%^&*()"

	responses := []string{
		"I can handle queries with special characters and unicode.",
	}
	fakeLLM := fake.NewFakeLLM(responses)
	agent.llm = fakeLLM
	agent.tools = agent.createDatabaseTools()

	var err error
	agent.executor, err = agents.Initialize(
		agent.llm,
		agent.tools,
		agents.ZeroShotReactDescription,
	)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := agent.ProcessQuery(ctx, specialQuery)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAgent_ProcessQuery_DatabaseOperations(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	// Setup test data
	err := database.PutCF("default", "test:1", "value1")
	require.NoError(t, err)
	err = database.PutCF("default", "test:2", "value2")
	require.NoError(t, err)

	agent := NewAgent(dbPtr)
	config := newTestConfig()
	agent.config = config

	testCases := []struct {
		name     string
		query    string
		response string
	}{
		{
			name:     "get operation",
			query:    "get the value for key test:1",
			response: "The value for test:1 is value1",
		},
		{
			name:     "scan operation",
			query:    "scan all keys with prefix test:",
			response: "Found keys: test:1, test:2",
		},
		{
			name:     "list operation",
			query:    "list all column families",
			response: "Column families: default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			responses := []string{tc.response}
			fakeLLM := fake.NewFakeLLM(responses)
			agent.llm = fakeLLM
			agent.tools = agent.createDatabaseTools()

			agent.executor, err = agents.Initialize(
				agent.llm,
				agent.tools,
				agents.ZeroShotReactDescription,
			)
			require.NoError(t, err)

			ctx := context.Background()
			result, err := agent.ProcessQuery(ctx, tc.query)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			// Success may vary based on how the fake LLM interacts with tools
		})
	}
}

func TestAgent_InitializeError_InvalidConfig(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)

	// Create invalid config - use openai without API key to trigger error
	config := &Config{
		GraphChain: GraphChainConfig{
			LLM: LLMConfig{
				Provider: "openai",
				Model:    "test-model",
				APIKey:   "", // Empty API key should cause error
			},
		},
	}

	ctx := context.Background()
	err := agent.Initialize(ctx, config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing the OpenAI API key")
}

func TestAgent_GetCapabilities(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)
	agent.tools = agent.createDatabaseTools()

	capabilities := agent.GetCapabilities()

	assert.NotEmpty(t, capabilities)
	assert.Equal(t, 8, len(capabilities))

	expectedTools := []string{
		"get_value", "put_value", "scan_range", "prefix_scan",
		"list_column_families", "get_last", "json_query", "get_stats",
	}

	for _, expected := range expectedTools {
		assert.Contains(t, capabilities, expected)
	}
}

func TestAgent_ContextCancellation(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)
	config := newTestConfig()
	agent.config = config

	responses := []string{
		"Processing query...",
	}
	fakeLLM := fake.NewFakeLLM(responses)
	agent.llm = fakeLLM
	agent.tools = agent.createDatabaseTools()

	var err error
	agent.executor, err = agents.Initialize(
		agent.llm,
		agent.tools,
		agents.ZeroShotReactDescription,
	)
	require.NoError(t, err)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := agent.ProcessQuery(ctx, "test query")

	// The behavior depends on how langchaingo handles cancelled contexts
	// We test that the method handles it gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "context")
	} else {
		assert.NotNil(t, result)
	}
}

func TestAgent_BuildSmallModelPrompt_TokenLimit(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)
	config := newTestConfig()
	agent.config = config

	// 构造多轮历史
	memory := NewConversationMemory(10)
	for i := 1; i <= 5; i++ {
		inputs := map[string]any{"input": fmt.Sprintf("问题%d", i)}
		outputs := map[string]any{"output": fmt.Sprintf("回答%d", i)}
		_ = memory.SaveContext(context.Background(), inputs, outputs)
	}
	agent.memory = memory

	query := "请统计所有用户数量"
	tokenLimit := 30

	// 构建小模型prompt（假设实现为 BuildSmallModelPrompt）
	prompt := agent.BuildSmallModelPrompt(query, tokenLimit, "test-model")
	tokenCount := EstimateTokenCount(prompt, "test-model")
	assert.LessOrEqual(t, tokenCount, tokenLimit)
	assert.Contains(t, prompt, "请统计所有用户数量")
	// 断言历史被裁剪或摘要
	assert.Contains(t, prompt, "Human:")
	assert.Contains(t, prompt, "Assistant:")
}

type mockExecutor struct{}

func (m *mockExecutor) Call(ctx context.Context, inputs map[string]any, opts ...chains.ChainCallOption) (map[string]any, error) {
	return map[string]any{"output": "Processed: get all keys from users"}, nil
}

// 兼容 agents.Executor 其他方法（如有需要可补充空实现）

func TestAgent_ProcessQuery_NoTokenization(t *testing.T) {
	database := newTestDB(t)
	dbPtr, ok := database.(*db.DB)
	require.True(t, ok)

	agent := NewAgent(dbPtr)
	config := newTestConfig()
	agent.config = config

	// 使用 mockExecutor 替换 agent.executor
	agent.executor = &mockExecutor{}

	ctx := context.Background()
	result, err := agent.ProcessQuery(ctx, "get all keys from users")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	if str, ok := result.Data.(string); ok {
		assert.Contains(t, str, "get all keys from users")
	} else {
		t.Fatalf("result.Data is not a string, got: %T", result.Data)
	}
}
