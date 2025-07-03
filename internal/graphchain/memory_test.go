package graphchain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestNewConversationMemory(t *testing.T) {
	tests := []struct {
		name     string
		maxSize  int
		expected int
	}{
		{
			name:     "default size for zero",
			maxSize:  0,
			expected: 100,
		},
		{
			name:     "negative size",
			maxSize:  -1,
			expected: 100,
		},
		{
			name:     "custom size",
			maxSize:  50,
			expected: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory := NewConversationMemory(tt.maxSize)
			assert.Equal(t, tt.expected, memory.maxSize)
			assert.True(t, memory.returnMessages)
			assert.Empty(t, memory.history)
		})
	}
}

func TestConversationMemory_MemoryVariables(t *testing.T) {
	memory := NewConversationMemory(10)
	vars := memory.MemoryVariables()

	expected := []string{"history", "chat_history"}
	assert.Equal(t, expected, vars)
}

func TestConversationMemory_SaveAndLoadContext(t *testing.T) {
	memory := NewConversationMemory(10)
	ctx := context.Background()

	// Test saving context
	inputs := map[string]any{
		"input": "列出所有列族",
	}
	outputs := map[string]any{
		"output":         "数据库中有：users, products, logs",
		"execution_time": 500 * time.Millisecond,
	}

	err := memory.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	// Verify history was saved
	history := memory.GetHistory()
	assert.Len(t, history, 1)
	assert.Equal(t, "列出所有列族", history[0].UserQuery)
	assert.Equal(t, "数据库中有：users, products, logs", history[0].AgentResponse)
	assert.Equal(t, 500*time.Millisecond, history[0].ExecutionTime)

	// Test loading memory variables
	loadInputs := map[string]any{
		"input": "users里面有什么？",
	}
	memoryVars, err := memory.LoadMemoryVariables(ctx, loadInputs)
	require.NoError(t, err)

	// Check returned variables
	assert.Contains(t, memoryVars, "chat_history")

	// Verify chat history format
	chatHistory, ok := memoryVars["chat_history"].([]llms.ChatMessage)
	require.True(t, ok)
	assert.Len(t, chatHistory, 2) // One human, one AI message

	humanMsg, ok := chatHistory[0].(llms.HumanChatMessage)
	require.True(t, ok)
	assert.Equal(t, "列出所有列族", humanMsg.Content)

	aiMsg, ok := chatHistory[1].(llms.AIChatMessage)
	require.True(t, ok)
	assert.Equal(t, "数据库中有：users, products, logs", aiMsg.Content)
}

func TestConversationMemory_StringFormat(t *testing.T) {
	memory := NewConversationMemory(10)
	memory.SetReturnMessages(false) // Use string format
	ctx := context.Background()

	// Add conversation turns
	inputs1 := map[string]any{"input": "问题1"}
	outputs1 := map[string]any{"output": "回答1"}
	err := memory.SaveContext(ctx, inputs1, outputs1)
	require.NoError(t, err)

	inputs2 := map[string]any{"input": "问题2"}
	outputs2 := map[string]any{"output": "回答2"}
	err = memory.SaveContext(ctx, inputs2, outputs2)
	require.NoError(t, err)

	// Load as string format
	memoryVars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	require.NoError(t, err)

	historyStr, ok := memoryVars["history"].(string)
	require.True(t, ok)

	expected := "Human: 问题1\nAssistant: 回答1\n\nHuman: 问题2\nAssistant: 回答2"
	assert.Equal(t, expected, historyStr)
}

func TestConversationMemory_SlidingWindow(t *testing.T) {
	memory := NewConversationMemory(2) // Small window for testing
	ctx := context.Background()

	// Add 3 conversation turns (should exceed capacity)
	for i := 1; i <= 3; i++ {
		inputs := map[string]any{"input": fmt.Sprintf("问题%d", i)}
		outputs := map[string]any{"output": fmt.Sprintf("回答%d", i)}
		err := memory.SaveContext(ctx, inputs, outputs)
		require.NoError(t, err)
	}

	// Should only keep the last 2 turns
	history := memory.GetHistory()
	assert.Len(t, history, 2)
	assert.Equal(t, "问题2", history[0].UserQuery)
	assert.Equal(t, "问题3", history[1].UserQuery)
}

func TestConversationMemory_Clear(t *testing.T) {
	memory := NewConversationMemory(10)
	ctx := context.Background()

	// Add some history
	inputs := map[string]any{"input": "测试问题"}
	outputs := map[string]any{"output": "测试回答"}
	err := memory.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	// Verify history exists
	assert.Len(t, memory.GetHistory(), 1)

	// Clear memory
	err = memory.Clear(ctx)
	require.NoError(t, err)

	// Verify history is cleared
	assert.Empty(t, memory.GetHistory())
}

func TestConversationMemory_GetRecentHistory(t *testing.T) {
	memory := NewConversationMemory(10)
	ctx := context.Background()

	// Add 5 conversation turns
	for i := 1; i <= 5; i++ {
		inputs := map[string]any{"input": fmt.Sprintf("问题%d", i)}
		outputs := map[string]any{"output": fmt.Sprintf("回答%d", i)}
		err := memory.SaveContext(ctx, inputs, outputs)
		require.NoError(t, err)
	}

	// Test getting recent history
	recent := memory.GetRecentHistory(3)
	assert.Len(t, recent, 3)
	assert.Equal(t, "问题3", recent[0].UserQuery)
	assert.Equal(t, "问题4", recent[1].UserQuery)
	assert.Equal(t, "问题5", recent[2].UserQuery)

	// Test edge cases
	assert.Empty(t, memory.GetRecentHistory(0))
	assert.Empty(t, memory.GetRecentHistory(-1))

	// Test requesting more than available
	all := memory.GetRecentHistory(10)
	assert.Len(t, all, 5)
}

func TestConversationMemory_GetStats(t *testing.T) {
	memory := NewConversationMemory(10)
	ctx := context.Background()

	// Test empty memory stats
	stats := memory.GetStats()
	assert.Equal(t, 0, stats.TotalTurns)
	assert.Equal(t, 10, stats.MaxSize)
	assert.Equal(t, 0, stats.TotalChars)
	assert.Equal(t, 0.0, stats.MemoryUsage)

	// Add some conversation turns
	inputs := map[string]any{"input": "hello"}
	outputs := map[string]any{"output": "world"}
	err := memory.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	stats = memory.GetStats()
	assert.Equal(t, 1, stats.TotalTurns)
	assert.Equal(t, 10, stats.MaxSize)
	assert.Equal(t, 10, stats.TotalChars)    // "hello" + "world" = 10 chars
	assert.Equal(t, 10.0, stats.MemoryUsage) // 1/10 * 100 = 10%
}

func TestConversationMemory_AddCustomTurn(t *testing.T) {
	memory := NewConversationMemory(10)
	timestamp := time.Now()

	// Add custom turn
	memory.AddCustomTurn("自定义问题", "自定义回答", timestamp)

	history := memory.GetHistory()
	assert.Len(t, history, 1)
	assert.Equal(t, "自定义问题", history[0].UserQuery)
	assert.Equal(t, "自定义回答", history[0].AgentResponse)
	assert.Equal(t, timestamp, history[0].Timestamp)
}

func TestExtractStringFromMap(t *testing.T) {
	testMap := map[string]any{
		"input":  "test_input",
		"query":  "test_query",
		"number": 123,
		"empty":  "",
	}

	// Test successful extraction
	result := extractStringFromMap(testMap, "input", "missing")
	assert.Equal(t, "test_input", result)

	// Test fallback to second key
	result = extractStringFromMap(testMap, "missing", "query")
	assert.Equal(t, "test_query", result)

	// Test non-string value
	result = extractStringFromMap(testMap, "number")
	assert.Equal(t, "", result)

	// Test empty string
	result = extractStringFromMap(testMap, "empty")
	assert.Equal(t, "", result)

	// Test missing keys
	result = extractStringFromMap(testMap, "missing1", "missing2")
	assert.Equal(t, "", result)
}
