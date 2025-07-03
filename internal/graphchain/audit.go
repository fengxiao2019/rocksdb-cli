package graphchain

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// AuditEvent represents an audit event
type AuditEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	Query     string                 `json:"query"`
	Success   bool                   `json:"success"`
	Duration  time.Duration          `json:"duration"`
	Error     string                 `json:"error,omitempty"`
	Tools     []string               `json:"tools,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogger handles audit logging
type AuditLogger struct {
	mutex   sync.RWMutex
	events  []AuditEvent
	maxSize int
	logFile *os.File
	encoder *json.Encoder
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	logger := &AuditLogger{
		events:  make([]AuditEvent, 0),
		maxSize: 1000, // Keep last 1000 events in memory
	}

	// Try to open audit log file
	if logFile, err := os.OpenFile("graphchain_audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		logger.logFile = logFile
		logger.encoder = json.NewEncoder(logFile)
	} else {
		log.Printf("Warning: Failed to open audit log file: %v", err)
	}

	return logger
}

// LogQuery logs a query event
func (al *AuditLogger) LogQuery(query string) {
	al.logEvent(AuditEvent{
		Timestamp: time.Now(),
		EventType: "query_received",
		Query:     query,
		Success:   true,
	})
}

// LogQueryResult logs a query result event
func (al *AuditLogger) LogQueryResult(query string, success bool, duration time.Duration, tools []string, err error) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "query_completed",
		Query:     query,
		Success:   success,
		Duration:  duration,
		Tools:     tools,
	}

	if err != nil {
		event.Error = err.Error()
	}

	al.logEvent(event)
}

// LogToolExecution logs a tool execution event
func (al *AuditLogger) LogToolExecution(toolName string, parameters map[string]interface{}, success bool, duration time.Duration, err error) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "tool_executed",
		Query:     toolName,
		Success:   success,
		Duration:  duration,
		Metadata: map[string]interface{}{
			"tool_name":  toolName,
			"parameters": parameters,
		},
	}

	if err != nil {
		event.Error = err.Error()
	}

	al.logEvent(event)
}

// LogSecurityViolation logs a security violation event
func (al *AuditLogger) LogSecurityViolation(query string, reason string) {
	al.logEvent(AuditEvent{
		Timestamp: time.Now(),
		EventType: "security_violation",
		Query:     query,
		Success:   false,
		Error:     reason,
		Metadata: map[string]interface{}{
			"violation_reason": reason,
		},
	})
}

// logEvent logs an audit event
func (al *AuditLogger) logEvent(event AuditEvent) {
	al.mutex.Lock()
	defer al.mutex.Unlock()

	// Add to in-memory store
	al.events = append(al.events, event)

	// Maintain max size
	if len(al.events) > al.maxSize {
		al.events = al.events[len(al.events)-al.maxSize:]
	}

	// Write to log file if available
	if al.encoder != nil {
		if err := al.encoder.Encode(event); err != nil {
			log.Printf("Warning: Failed to write audit log: %v", err)
		}
	}

	// Also log to standard logger for important events
	if event.EventType == "security_violation" || !event.Success {
		log.Printf("AUDIT: %s - %s (success: %t)", event.EventType, event.Query, event.Success)
	}
}

// GetEvents returns recent audit events
func (al *AuditLogger) GetEvents(limit int) []AuditEvent {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	if limit <= 0 || limit > len(al.events) {
		limit = len(al.events)
	}

	// Return the last 'limit' events
	start := len(al.events) - limit
	if start < 0 {
		start = 0
	}

	events := make([]AuditEvent, limit)
	copy(events, al.events[start:])
	return events
}

// GetEventsByType returns events of a specific type
func (al *AuditLogger) GetEventsByType(eventType string, limit int) []AuditEvent {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	events := make([]AuditEvent, 0)
	count := 0

	// Search from most recent to oldest
	for i := len(al.events) - 1; i >= 0 && count < limit; i-- {
		if al.events[i].EventType == eventType {
			events = append(events, al.events[i])
			count++
		}
	}

	return events
}

// GetSecurityViolations returns recent security violations
func (al *AuditLogger) GetSecurityViolations(limit int) []AuditEvent {
	return al.GetEventsByType("security_violation", limit)
}

// GetFailedQueries returns recent failed queries
func (al *AuditLogger) GetFailedQueries(limit int) []AuditEvent {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	events := make([]AuditEvent, 0)
	count := 0

	// Search from most recent to oldest
	for i := len(al.events) - 1; i >= 0 && count < limit; i-- {
		if !al.events[i].Success {
			events = append(events, al.events[i])
			count++
		}
	}

	return events
}

// GetStatistics returns audit statistics
func (al *AuditLogger) GetStatistics() map[string]interface{} {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	stats := make(map[string]interface{})
	eventTypeCount := make(map[string]int)
	successCount := 0
	errorCount := 0
	totalDuration := time.Duration(0)
	queryCount := 0

	for _, event := range al.events {
		eventTypeCount[event.EventType]++

		if event.Success {
			successCount++
		} else {
			errorCount++
		}

		if event.EventType == "query_completed" {
			totalDuration += event.Duration
			queryCount++
		}
	}

	stats["total_events"] = len(al.events)
	stats["event_types"] = eventTypeCount
	stats["success_count"] = successCount
	stats["error_count"] = errorCount
	stats["success_rate"] = float64(successCount) / float64(len(al.events))

	if queryCount > 0 {
		stats["average_query_duration"] = totalDuration / time.Duration(queryCount)
	}

	return stats
}

// ExportEvents exports audit events to a file
func (al *AuditLogger) ExportEvents(filename string, startTime, endTime time.Time) error {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	// Filter events by time range
	var filteredEvents []AuditEvent
	for _, event := range al.events {
		if event.Timestamp.After(startTime) && event.Timestamp.Before(endTime) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	// Create export file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	// Write events as JSON lines
	encoder := json.NewEncoder(file)
	for _, event := range filteredEvents {
		if err := encoder.Encode(event); err != nil {
			return fmt.Errorf("failed to write event: %w", err)
		}
	}

	return nil
}

// Close closes the audit logger and any open files
func (al *AuditLogger) Close() {
	al.mutex.Lock()
	defer al.mutex.Unlock()

	if al.logFile != nil {
		al.logFile.Close()
		al.logFile = nil
		al.encoder = nil
	}
}
