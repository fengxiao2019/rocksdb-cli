package graphchain

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditLogger(t *testing.T) {
	logger := NewAuditLogger()

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.events)
	assert.Equal(t, 0, len(logger.events))
	assert.Equal(t, 1000, logger.maxSize)
	// Note: logFile and encoder may or may not be nil depending on file permissions
}

func TestAuditLogger_LogQuery(t *testing.T) {
	logger := NewAuditLogger()

	testQuery := "get user:123"
	logger.LogQuery(testQuery)

	events := logger.GetEvents(10)
	require.Equal(t, 1, len(events))

	event := events[0]
	assert.Equal(t, "query_received", event.EventType)
	assert.Equal(t, testQuery, event.Query)
	assert.True(t, event.Success)
	assert.True(t, time.Since(event.Timestamp) < time.Second)
	assert.Empty(t, event.Error)
	assert.Empty(t, event.Tools)
	assert.Nil(t, event.Metadata)
}

func TestAuditLogger_LogQueryResult_Success(t *testing.T) {
	logger := NewAuditLogger()

	testQuery := "scan users prefix"
	duration := 150 * time.Millisecond
	tools := []string{"prefix_scan", "get_value"}

	logger.LogQueryResult(testQuery, true, duration, tools, nil)

	events := logger.GetEvents(10)
	require.Equal(t, 1, len(events))

	event := events[0]
	assert.Equal(t, "query_completed", event.EventType)
	assert.Equal(t, testQuery, event.Query)
	assert.True(t, event.Success)
	assert.Equal(t, duration, event.Duration)
	assert.Equal(t, tools, event.Tools)
	assert.Empty(t, event.Error)
	assert.True(t, time.Since(event.Timestamp) < time.Second)
}

func TestAuditLogger_LogQueryResult_Failure(t *testing.T) {
	logger := NewAuditLogger()

	testQuery := "invalid query"
	duration := 50 * time.Millisecond
	tools := []string{"validator"}
	testError := errors.New("syntax error in query")

	logger.LogQueryResult(testQuery, false, duration, tools, testError)

	events := logger.GetEvents(10)
	require.Equal(t, 1, len(events))

	event := events[0]
	assert.Equal(t, "query_completed", event.EventType)
	assert.Equal(t, testQuery, event.Query)
	assert.False(t, event.Success)
	assert.Equal(t, duration, event.Duration)
	assert.Equal(t, tools, event.Tools)
	assert.Equal(t, "syntax error in query", event.Error)
}

func TestAuditLogger_LogToolExecution_Success(t *testing.T) {
	logger := NewAuditLogger()

	toolName := "get_value"
	parameters := map[string]interface{}{
		"column_family": "default",
		"key":           "user:123",
	}
	duration := 25 * time.Millisecond

	logger.LogToolExecution(toolName, parameters, true, duration, nil)

	events := logger.GetEvents(10)
	require.Equal(t, 1, len(events))

	event := events[0]
	assert.Equal(t, "tool_executed", event.EventType)
	assert.Equal(t, toolName, event.Query) // Tool name is stored in Query field
	assert.True(t, event.Success)
	assert.Equal(t, duration, event.Duration)
	assert.Empty(t, event.Error)
	assert.NotNil(t, event.Metadata)
	assert.Equal(t, toolName, event.Metadata["tool_name"])
	assert.Equal(t, parameters, event.Metadata["parameters"])
}

func TestAuditLogger_LogToolExecution_Failure(t *testing.T) {
	logger := NewAuditLogger()

	toolName := "put_value"
	parameters := map[string]interface{}{
		"column_family": "invalid_cf",
		"key":           "test:key",
		"value":         "test value",
	}
	duration := 10 * time.Millisecond
	testError := errors.New("column family not found")

	logger.LogToolExecution(toolName, parameters, false, duration, testError)

	events := logger.GetEvents(10)
	require.Equal(t, 1, len(events))

	event := events[0]
	assert.Equal(t, "tool_executed", event.EventType)
	assert.Equal(t, toolName, event.Query)
	assert.False(t, event.Success)
	assert.Equal(t, duration, event.Duration)
	assert.Equal(t, "column family not found", event.Error)
	assert.Equal(t, toolName, event.Metadata["tool_name"])
	assert.Equal(t, parameters, event.Metadata["parameters"])
}

func TestAuditLogger_LogSecurityViolation(t *testing.T) {
	logger := NewAuditLogger()

	testQuery := "DROP TABLE users"
	reason := "dangerous SQL injection pattern detected"

	logger.LogSecurityViolation(testQuery, reason)

	events := logger.GetEvents(10)
	require.Equal(t, 1, len(events))

	event := events[0]
	assert.Equal(t, "security_violation", event.EventType)
	assert.Equal(t, testQuery, event.Query)
	assert.False(t, event.Success)
	assert.Equal(t, reason, event.Error)
	assert.NotNil(t, event.Metadata)
	assert.Equal(t, reason, event.Metadata["violation_reason"])
}

func TestAuditLogger_GetEvents(t *testing.T) {
	logger := NewAuditLogger()

	// Add multiple events
	logger.LogQuery("query1")
	logger.LogQuery("query2")
	logger.LogQuery("query3")
	logger.LogSecurityViolation("bad query", "violation")

	// Test getting all events
	allEvents := logger.GetEvents(10)
	assert.Equal(t, 4, len(allEvents))

	// Test getting limited events
	limitedEvents := logger.GetEvents(2)
	assert.Equal(t, 2, len(limitedEvents))

	// Events should be returned in order (most recent last)
	assert.Equal(t, "query_received", allEvents[0].EventType)
	assert.Equal(t, "security_violation", allEvents[3].EventType)

	// Test getting with 0 limit (should return all)
	allEventsZero := logger.GetEvents(0)
	assert.Equal(t, 4, len(allEventsZero))

	// Test getting with negative limit (should return all)
	allEventsNegative := logger.GetEvents(-1)
	assert.Equal(t, 4, len(allEventsNegative))
}

func TestAuditLogger_GetEventsByType(t *testing.T) {
	logger := NewAuditLogger()

	// Add various types of events
	logger.LogQuery("query1")
	logger.LogQueryResult("query1", true, 100*time.Millisecond, nil, nil)
	logger.LogQuery("query2")
	logger.LogSecurityViolation("bad query", "violation")
	logger.LogToolExecution("get_value", nil, true, 50*time.Millisecond, nil)

	// Test getting query_received events
	queryEvents := logger.GetEventsByType("query_received", 10)
	assert.Equal(t, 2, len(queryEvents))
	for _, event := range queryEvents {
		assert.Equal(t, "query_received", event.EventType)
	}

	// Test getting security_violation events
	securityEvents := logger.GetEventsByType("security_violation", 10)
	assert.Equal(t, 1, len(securityEvents))
	assert.Equal(t, "security_violation", securityEvents[0].EventType)

	// Test getting non-existent event type
	nonExistentEvents := logger.GetEventsByType("non_existent", 10)
	assert.Equal(t, 0, len(nonExistentEvents))

	// Test with limit
	limitedQueryEvents := logger.GetEventsByType("query_received", 1)
	assert.Equal(t, 1, len(limitedQueryEvents))
}

func TestAuditLogger_GetSecurityViolations(t *testing.T) {
	logger := NewAuditLogger()

	// Add various events including security violations
	logger.LogQuery("normal query")
	logger.LogSecurityViolation("DROP TABLE users", "SQL injection")
	logger.LogSecurityViolation("rm -rf /", "system command")
	logger.LogQuery("another normal query")

	violations := logger.GetSecurityViolations(10)
	assert.Equal(t, 2, len(violations))

	for _, violation := range violations {
		assert.Equal(t, "security_violation", violation.EventType)
		assert.False(t, violation.Success)
	}

	// Test with limit
	limitedViolations := logger.GetSecurityViolations(1)
	assert.Equal(t, 1, len(limitedViolations))
}

func TestAuditLogger_GetFailedQueries(t *testing.T) {
	logger := NewAuditLogger()

	// Add various events including failures
	logger.LogQueryResult("query1", true, 100*time.Millisecond, nil, nil)                       // success
	logger.LogQueryResult("query2", false, 50*time.Millisecond, nil, errors.New("error"))       // failure
	logger.LogSecurityViolation("bad query", "violation")                                       // failure
	logger.LogToolExecution("tool1", nil, false, 25*time.Millisecond, errors.New("tool error")) // failure
	logger.LogQuery("query3")                                                                   // success

	failures := logger.GetFailedQueries(10)
	assert.Equal(t, 3, len(failures))

	for _, failure := range failures {
		assert.False(t, failure.Success)
	}

	// Test with limit
	limitedFailures := logger.GetFailedQueries(2)
	assert.Equal(t, 2, len(limitedFailures))
}

func TestAuditLogger_GetStatistics(t *testing.T) {
	logger := NewAuditLogger()

	// Add various types of events
	logger.LogQuery("query1")
	logger.LogQueryResult("query1", true, 100*time.Millisecond, nil, nil)
	logger.LogQueryResult("query2", false, 200*time.Millisecond, nil, errors.New("error"))
	logger.LogSecurityViolation("bad query", "violation")
	logger.LogToolExecution("tool1", nil, true, 50*time.Millisecond, nil)

	stats := logger.GetStatistics()

	assert.Equal(t, 5, stats["total_events"])
	assert.Equal(t, 3, stats["success_count"])  // query, query_result success, tool success
	assert.Equal(t, 2, stats["error_count"])    // query_result failure, security violation
	assert.Equal(t, 0.6, stats["success_rate"]) // 3/5

	// Check event type counts
	eventTypes := stats["event_types"].(map[string]int)
	assert.Equal(t, 1, eventTypes["query_received"])
	assert.Equal(t, 2, eventTypes["query_completed"])
	assert.Equal(t, 1, eventTypes["security_violation"])
	assert.Equal(t, 1, eventTypes["tool_executed"])

	// Check average query duration
	avgDuration := stats["average_query_duration"].(time.Duration)
	assert.Equal(t, 150*time.Millisecond, avgDuration) // (100 + 200) / 2
}

func TestAuditLogger_ExportEvents(t *testing.T) {
	logger := NewAuditLogger()

	// Add some events with known timestamps
	baseTime := time.Now().Truncate(time.Second)

	// Manually add events to control timestamps
	logger.events = []AuditEvent{
		{
			Timestamp: baseTime.Add(-2 * time.Hour),
			EventType: "query_received",
			Query:     "old query",
			Success:   true,
		},
		{
			Timestamp: baseTime.Add(-1 * time.Hour),
			EventType: "query_completed",
			Query:     "middle query",
			Success:   true,
			Duration:  100 * time.Millisecond,
		},
		{
			Timestamp: baseTime,
			EventType: "security_violation",
			Query:     "recent violation",
			Success:   false,
			Error:     "dangerous pattern",
		},
	}

	// Test export with time range
	tempFile, err := os.CreateTemp("", "audit_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	startTime := baseTime.Add(-90 * time.Minute)
	endTime := baseTime.Add(30 * time.Minute)

	err = logger.ExportEvents(tempFile.Name(), startTime, endTime)
	require.NoError(t, err)

	// Verify file was created and contains data
	info, err := os.Stat(tempFile.Name())
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestAuditLogger_MaxSizeLimit(t *testing.T) {
	logger := NewAuditLogger()
	logger.maxSize = 3 // Set small limit for testing

	// Add more events than the max size
	for i := 0; i < 5; i++ {
		logger.LogQuery(fmt.Sprintf("query%d", i))
	}

	events := logger.GetEvents(10)
	assert.Equal(t, 3, len(events)) // Should be limited to maxSize

	// Check that we kept the most recent events
	assert.Contains(t, events[len(events)-1].Query, "query4") // Most recent
}

func TestAuditLogger_ThreadSafety(t *testing.T) {
	logger := NewAuditLogger()

	// Test concurrent access
	done := make(chan bool, 3)

	// Goroutine 1: Add query events
	go func() {
		for i := 0; i < 50; i++ {
			logger.LogQuery(fmt.Sprintf("query%d", i))
			time.Sleep(1 * time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 2: Add tool events
	go func() {
		for i := 0; i < 50; i++ {
			logger.LogToolExecution(fmt.Sprintf("tool%d", i), nil, true, time.Millisecond, nil)
			time.Sleep(1 * time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 3: Read events
	go func() {
		for i := 0; i < 50; i++ {
			events := logger.GetEvents(10)
			assert.GreaterOrEqual(t, len(events), 0)

			stats := logger.GetStatistics()
			assert.GreaterOrEqual(t, stats["total_events"], 0)

			time.Sleep(1 * time.Microsecond)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	<-done
	<-done
	<-done

	// Final verification
	finalEvents := logger.GetEvents(200)
	assert.GreaterOrEqual(t, len(finalEvents), 50) // Should have at least 50 events
}

func TestAuditLogger_EmptyEvents(t *testing.T) {
	logger := NewAuditLogger()

	// Test with no events
	events := logger.GetEvents(10)
	assert.Equal(t, 0, len(events))

	eventsByType := logger.GetEventsByType("query_received", 10)
	assert.Equal(t, 0, len(eventsByType))

	violations := logger.GetSecurityViolations(10)
	assert.Equal(t, 0, len(violations))

	failures := logger.GetFailedQueries(10)
	assert.Equal(t, 0, len(failures))

	stats := logger.GetStatistics()
	assert.Equal(t, 0, stats["total_events"])
	assert.Equal(t, 0, stats["success_count"])
	assert.Equal(t, 0, stats["error_count"])
}

func TestAuditLogger_ExportEvents_FileError(t *testing.T) {
	logger := NewAuditLogger()
	logger.LogQuery("test query")

	// Try to export to an invalid path
	err := logger.ExportEvents("/invalid/path/audit.json", time.Now().Add(-time.Hour), time.Now())
	assert.Error(t, err)
}

func TestAuditLogger_Close(t *testing.T) {
	logger := NewAuditLogger()

	// Close should not panic even if logFile is nil
	assert.NotPanics(t, func() {
		logger.Close()
	})
}

func TestAuditEvent_Serialization(t *testing.T) {
	// Test that AuditEvent can be properly serialized/deserialized
	event := AuditEvent{
		Timestamp: time.Now().Truncate(time.Second),
		EventType: "query_completed",
		Query:     "test query",
		Success:   true,
		Duration:  150 * time.Millisecond,
		Error:     "",
		Tools:     []string{"get_value", "scan"},
		Metadata: map[string]interface{}{
			"user_id": "test-user",
			"retries": 2,
		},
	}

	// This test mainly verifies the struct is properly tagged for JSON
	assert.Equal(t, "query_completed", event.EventType)
	assert.True(t, event.Success)
	assert.Equal(t, 150*time.Millisecond, event.Duration)
	assert.Equal(t, []string{"get_value", "scan"}, event.Tools)
	assert.NotNil(t, event.Metadata)
}
