// Package graphchain provides memory management for GraphChain agents
package graphchain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// ConversationMemory implements schema.Memory interface for storing conversation history
type ConversationMemory struct {
	history        []ConversationTurn `json:"history"`
	maxSize        int                `json:"max_size"`
	returnMessages bool               `json:"return_messages"`
	mutex          sync.RWMutex       `json:"-"`
}

// ConversationTurn represents a single turn in the conversation
type ConversationTurn struct {
	UserQuery     string                 `json:"user_query"`
	AgentResponse string                 `json:"agent_response"`
	Timestamp     time.Time              `json:"timestamp"`
	Context       map[string]interface{} `json:"context,omitempty"`
	ExecutionTime time.Duration          `json:"execution_time,omitempty"`
}

// NewConversationMemory creates a new conversation memory instance
func NewConversationMemory(maxSize int) *ConversationMemory {
	if maxSize <= 0 {
		maxSize = 100 // default size
	}

	return &ConversationMemory{
		history:        make([]ConversationTurn, 0, maxSize),
		maxSize:        maxSize,
		returnMessages: true,
		mutex:          sync.RWMutex{},
	}
}

// MemoryVariables returns the memory variables that this memory class will add to chain inputs
func (cm *ConversationMemory) MemoryVariables() []string {
	return []string{"history", "chat_history"}
}

// LoadMemoryVariables returns key-value pairs given the text input to the chain
func (cm *ConversationMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	memoryVars := make(map[string]any)

	if cm.returnMessages {
		// Return as chat messages
		messages := cm.getChatMessages()
		memoryVars["chat_history"] = messages
	} else {
		// Return as formatted string
		historyStr := cm.getFormattedHistory()
		memoryVars["history"] = historyStr
	}

	return memoryVars, nil
}

// SaveContext saves the inputs and outputs of this chain to memory
func (cm *ConversationMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Extract user query and agent response
	userQuery := extractStringFromMap(inputs, "input", "user_input", "query")
	agentResponse := extractStringFromMap(outputs, "output", "text", "response")

	if userQuery == "" && agentResponse == "" {
		return nil // Nothing to save
	}

	// Create conversation turn
	turn := ConversationTurn{
		UserQuery:     userQuery,
		AgentResponse: agentResponse,
		Timestamp:     time.Now(),
		Context:       make(map[string]interface{}),
	}

	// Extract additional context
	if executionTime, ok := outputs["execution_time"].(time.Duration); ok {
		turn.ExecutionTime = executionTime
	}

	// Add turn to history
	cm.history = append(cm.history, turn)

	// Trim history if needed (sliding window)
	if len(cm.history) > cm.maxSize {
		cm.history = cm.history[len(cm.history)-cm.maxSize:]
	}

	return nil
}

// Clear removes all stored states
func (cm *ConversationMemory) Clear(ctx context.Context) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.history = cm.history[:0] // Clear slice but keep capacity
	return nil
}

// getChatMessages converts history to chat messages format
func (cm *ConversationMemory) getChatMessages() []llms.ChatMessage {
	messages := make([]llms.ChatMessage, 0, len(cm.history)*2)

	for _, turn := range cm.history {
		if turn.UserQuery != "" {
			messages = append(messages, llms.HumanChatMessage{Content: turn.UserQuery})
		}
		if turn.AgentResponse != "" {
			messages = append(messages, llms.AIChatMessage{Content: turn.AgentResponse})
		}
	}

	return messages
}

// getFormattedHistory returns history as formatted string
func (cm *ConversationMemory) getFormattedHistory() string {
	if len(cm.history) == 0 {
		return ""
	}

	var formatted string
	for i, turn := range cm.history {
		if i > 0 {
			formatted += "\n\n"
		}

		if turn.UserQuery != "" {
			formatted += fmt.Sprintf("Human: %s", turn.UserQuery)
		}
		if turn.AgentResponse != "" {
			if turn.UserQuery != "" {
				formatted += "\n"
			}
			formatted += fmt.Sprintf("Assistant: %s", turn.AgentResponse)
		}
	}

	return formatted
}

// GetHistory returns the conversation history (read-only)
func (cm *ConversationMemory) GetHistory() []ConversationTurn {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Return a copy to prevent external modification
	historyCopy := make([]ConversationTurn, len(cm.history))
	copy(historyCopy, cm.history)
	return historyCopy
}

// GetRecentHistory returns the last N conversation turns
func (cm *ConversationMemory) GetRecentHistory(n int) []ConversationTurn {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if n <= 0 || len(cm.history) == 0 {
		return []ConversationTurn{}
	}

	start := len(cm.history) - n
	if start < 0 {
		start = 0
	}

	recent := make([]ConversationTurn, len(cm.history[start:]))
	copy(recent, cm.history[start:])
	return recent
}

// GetStats returns memory statistics
func (cm *ConversationMemory) GetStats() MemoryStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var totalChars int
	var oldestTime, newestTime time.Time

	if len(cm.history) > 0 {
		oldestTime = cm.history[0].Timestamp
		newestTime = cm.history[len(cm.history)-1].Timestamp

		for _, turn := range cm.history {
			totalChars += len(turn.UserQuery) + len(turn.AgentResponse)
		}
	}

	return MemoryStats{
		TotalTurns:  len(cm.history),
		MaxSize:     cm.maxSize,
		TotalChars:  totalChars,
		OldestTime:  oldestTime,
		NewestTime:  newestTime,
		MemoryUsage: float64(len(cm.history)) / float64(cm.maxSize) * 100,
	}
}

// MemoryStats provides statistics about memory usage
type MemoryStats struct {
	TotalTurns  int       `json:"total_turns"`
	MaxSize     int       `json:"max_size"`
	TotalChars  int       `json:"total_chars"`
	OldestTime  time.Time `json:"oldest_time"`
	NewestTime  time.Time `json:"newest_time"`
	MemoryUsage float64   `json:"memory_usage"` // percentage
}

// SetReturnMessages configures whether to return messages or formatted string
func (cm *ConversationMemory) SetReturnMessages(returnMessages bool) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.returnMessages = returnMessages
}

// Helper function to extract string from map with multiple possible keys
func extractStringFromMap(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}

// AddCustomTurn allows manual addition of conversation turns (for testing or import)
func (cm *ConversationMemory) AddCustomTurn(userQuery, agentResponse string, timestamp time.Time) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	turn := ConversationTurn{
		UserQuery:     userQuery,
		AgentResponse: agentResponse,
		Timestamp:     timestamp,
		Context:       make(map[string]interface{}),
	}

	cm.history = append(cm.history, turn)

	// Trim if needed
	if len(cm.history) > cm.maxSize {
		cm.history = cm.history[len(cm.history)-cm.maxSize:]
	}
}
