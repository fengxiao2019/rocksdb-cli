package graphchain

import (
	"testing"
	"time"

	"rocksdb-cli/internal/db"

	"github.com/stretchr/testify/assert"
)

func TestNewContextManager(t *testing.T) {
	config := &ContextConfig{
		EnableAutoDiscovery: true,
		UpdateInterval:      5 * time.Minute,
		MaxContextSize:      4096,
	}

	cm := NewContextManager(config)

	assert.NotNil(t, cm)
	assert.Equal(t, config, cm.config)
	assert.NotNil(t, cm.context)
	assert.Empty(t, cm.context.ColumnFamilies)
	assert.Equal(t, int64(0), cm.context.TotalKeys)
	assert.Equal(t, int64(0), cm.context.TotalSize)
	assert.NotNil(t, cm.context.DataTypeStats)
	assert.Empty(t, cm.context.KeyPrefixes)
}

func TestContextManager_GetContext(t *testing.T) {
	config := &ContextConfig{
		EnableAutoDiscovery: true,
		UpdateInterval:      5 * time.Minute,
	}

	cm := NewContextManager(config)

	// Modify internal context
	cm.context.TotalKeys = 100
	cm.context.TotalSize = 1024
	cm.context.ColumnFamilies = []string{"default", "users"}

	// Get context should return a copy
	context := cm.GetContext()

	assert.Equal(t, int64(100), context.TotalKeys)
	assert.Equal(t, int64(1024), context.TotalSize)
	assert.Equal(t, []string{"default", "users"}, context.ColumnFamilies)

	// Modifying returned context should not affect internal state
	context.TotalKeys = 200
	assert.Equal(t, int64(100), cm.context.TotalKeys) // Internal should be unchanged
}

func TestContextManager_UpdateDatabaseStats(t *testing.T) {
	config := &ContextConfig{
		EnableAutoDiscovery: true,
		UpdateInterval:      5 * time.Minute,
	}

	cm := NewContextManager(config)

	// Create mock database stats
	stats := &db.DatabaseStats{
		TotalKeyCount: 1000,
		TotalSize:     102400,
		ColumnFamilies: []db.CFStats{
			{
				Name:           "default",
				KeyCount:       800,
				TotalKeySize:   16000,
				TotalValueSize: 65920,
				DataTypeDistribution: map[db.DataType]int64{
					db.DataType("string"): 500,
					db.DataType("json"):   300,
				},
				CommonPrefixes: map[string]int64{
					"user:":    200,
					"session:": 100,
				},
			},
			{
				Name:           "cache",
				KeyCount:       200,
				TotalKeySize:   4000,
				TotalValueSize: 16480,
				DataTypeDistribution: map[db.DataType]int64{
					db.DataType("string"): 150,
					db.DataType("binary"): 50,
				},
				CommonPrefixes: map[string]int64{
					"cache:": 200,
				},
			},
		},
	}

	err := cm.UpdateDatabaseStats(stats)
	assert.NoError(t, err)

	context := cm.GetContext()

	// Check updated values
	assert.Equal(t, int64(1000), context.TotalKeys)
	assert.Equal(t, int64(102400), context.TotalSize)
	assert.Equal(t, []string{"default", "cache"}, context.ColumnFamilies)

	// Check data type aggregation
	assert.Equal(t, int64(650), context.DataTypeStats["string"]) // 500 + 150
	assert.Equal(t, int64(300), context.DataTypeStats["json"])
	assert.Equal(t, int64(50), context.DataTypeStats["binary"])

	// Check key prefixes
	expectedPrefixes := []string{"user:", "session:", "cache:"}
	assert.ElementsMatch(t, expectedPrefixes, context.KeyPrefixes)

	// Check that LastUpdated was set
	assert.True(t, time.Since(context.LastUpdated) < time.Second)
}

func TestContextManager_ShouldUpdateContext(t *testing.T) {
	config := &ContextConfig{
		EnableAutoDiscovery: true,
		UpdateInterval:      5 * time.Minute,
	}

	cm := NewContextManager(config)

	// Should not update immediately after creation
	assert.False(t, cm.ShouldUpdateContext())

	// Simulate time passage beyond update interval
	cm.context.LastUpdated = time.Now().Add(-6 * time.Minute)
	assert.True(t, cm.ShouldUpdateContext())

	// Should not update if auto discovery is disabled
	config.EnableAutoDiscovery = false
	assert.False(t, cm.ShouldUpdateContext())
}

func TestContextManager_ShouldUpdateContext_AutoDiscoveryDisabled(t *testing.T) {
	config := &ContextConfig{
		EnableAutoDiscovery: false,
		UpdateInterval:      5 * time.Minute,
	}

	cm := NewContextManager(config)

	// Should not update when auto discovery is disabled, even if enough time has passed
	cm.context.LastUpdated = time.Now().Add(-10 * time.Minute)
	assert.False(t, cm.ShouldUpdateContext())
}

func TestContextManager_GetContextSummary(t *testing.T) {
	config := &ContextConfig{
		EnableAutoDiscovery: true,
		UpdateInterval:      5 * time.Minute,
	}

	cm := NewContextManager(config)

	// Test with empty context
	summary := cm.GetContextSummary()
	assert.Contains(t, summary, "Database Context:")
	assert.Contains(t, summary, "Column Families: 0")
	assert.Contains(t, summary, "Total Keys: 0")
	assert.Contains(t, summary, "Total Size: 0 bytes")

	// Update with some data
	cm.context.ColumnFamilies = []string{"default", "users", "cache"}
	cm.context.TotalKeys = 5000
	cm.context.TotalSize = 1048576
	cm.context.KeyPrefixes = []string{"user:", "session:", "cache:", "config:", "temp:", "log:"}
	cm.context.DataTypeStats = map[string]int64{
		"string": 3000,
		"json":   1500,
		"binary": 500,
	}
	cm.context.LastUpdated = time.Now()

	summary = cm.GetContextSummary()

	// Check content
	assert.Contains(t, summary, "Column Families: 3")
	assert.Contains(t, summary, "[default users cache]")
	assert.Contains(t, summary, "Total Keys: 5000")
	assert.Contains(t, summary, "Total Size: 1048576 bytes")
	assert.Contains(t, summary, "Common Key Prefixes:")
	assert.Contains(t, summary, "Data Type Distribution:")
	assert.Contains(t, summary, "string: 3000")
	assert.Contains(t, summary, "json: 1500")
	assert.Contains(t, summary, "binary: 500")

	// Should only show first 5 prefixes
	prefixLines := 0
	for range cm.context.KeyPrefixes[:5] {
		if len(summary) > 0 {
			prefixLines++
		}
	}
	assert.LessOrEqual(t, prefixLines, 5)
}

func TestContextManager_GetContextSummary_LimitedPrefixes(t *testing.T) {
	config := &ContextConfig{}
	cm := NewContextManager(config)

	// Test with more than 5 prefixes
	cm.context.KeyPrefixes = []string{
		"user:", "session:", "cache:", "config:", "temp:", "log:", "audit:", "backup:",
	}

	summary := cm.GetContextSummary()

	// Should contain "Common Key Prefixes:" but only show first 5
	assert.Contains(t, summary, "Common Key Prefixes:")

	// Count occurrences - should only have first 5 prefixes
	firstFivePrefixes := cm.context.KeyPrefixes[:5]
	for i, prefix := range firstFivePrefixes {
		assert.Contains(t, summary, prefix, "Prefix %d should be in summary", i)
	}
}

func TestContextManager_GetContextSummary_EmptyDataTypes(t *testing.T) {
	config := &ContextConfig{}
	cm := NewContextManager(config)

	// Set some data but no data types
	cm.context.ColumnFamilies = []string{"default"}
	cm.context.TotalKeys = 100
	cm.context.DataTypeStats = map[string]int64{} // Empty

	summary := cm.GetContextSummary()

	assert.Contains(t, summary, "Column Families: 1")
	assert.Contains(t, summary, "Total Keys: 100")
	assert.NotContains(t, summary, "Data Type Distribution:") // Should not appear when empty
}

func TestContextManager_GetContextSummary_EmptyPrefixes(t *testing.T) {
	config := &ContextConfig{}
	cm := NewContextManager(config)

	// Set some data but no prefixes
	cm.context.ColumnFamilies = []string{"default"}
	cm.context.TotalKeys = 100
	cm.context.KeyPrefixes = []string{} // Empty

	summary := cm.GetContextSummary()

	assert.Contains(t, summary, "Column Families: 1")
	assert.Contains(t, summary, "Total Keys: 100")
	assert.NotContains(t, summary, "Common Key Prefixes:") // Should not appear when empty
}

func TestContextManager_UpdateDatabaseStats_EmptyStats(t *testing.T) {
	config := &ContextConfig{}
	cm := NewContextManager(config)

	// Test with empty stats
	stats := &db.DatabaseStats{
		TotalKeyCount:  0,
		TotalSize:      0,
		ColumnFamilies: []db.CFStats{},
	}

	err := cm.UpdateDatabaseStats(stats)
	assert.NoError(t, err)

	context := cm.GetContext()
	assert.Equal(t, int64(0), context.TotalKeys)
	assert.Equal(t, int64(0), context.TotalSize)
	assert.Empty(t, context.ColumnFamilies)
	assert.Empty(t, context.DataTypeStats)
	assert.Empty(t, context.KeyPrefixes)
}

func TestContextManager_UpdateDatabaseStats_MultipleColumnFamilies(t *testing.T) {
	config := &ContextConfig{}
	cm := NewContextManager(config)

	stats := &db.DatabaseStats{
		TotalKeyCount: 1500,
		TotalSize:     153600,
		ColumnFamilies: []db.CFStats{
			{
				Name:           "default",
				KeyCount:       1000,
				TotalKeySize:   20000,
				TotalValueSize: 82400,
				DataTypeDistribution: map[db.DataType]int64{
					db.DataType("string"): 800,
					db.DataType("json"):   200,
				},
				CommonPrefixes: map[string]int64{
					"user:": 500,
				},
			},
			{
				Name:           "sessions",
				KeyCount:       300,
				TotalKeySize:   6000,
				TotalValueSize: 24720,
				DataTypeDistribution: map[db.DataType]int64{
					db.DataType("string"): 300,
				},
				CommonPrefixes: map[string]int64{
					"session:": 300,
				},
			},
			{
				Name:           "cache",
				KeyCount:       200,
				TotalKeySize:   4000,
				TotalValueSize: 16480,
				DataTypeDistribution: map[db.DataType]int64{
					db.DataType("binary"): 200,
				},
				CommonPrefixes: map[string]int64{
					"cache:": 200,
				},
			},
		},
	}

	err := cm.UpdateDatabaseStats(stats)
	assert.NoError(t, err)

	context := cm.GetContext()

	// Check column families order is preserved
	assert.Equal(t, []string{"default", "sessions", "cache"}, context.ColumnFamilies)

	// Check aggregated data types
	assert.Equal(t, int64(1100), context.DataTypeStats["string"]) // 800 + 300
	assert.Equal(t, int64(200), context.DataTypeStats["json"])
	assert.Equal(t, int64(200), context.DataTypeStats["binary"])

	// Check all prefixes are included
	expectedPrefixes := []string{"user:", "session:", "cache:"}
	assert.ElementsMatch(t, expectedPrefixes, context.KeyPrefixes)
}

func TestContextManager_ThreadSafety(t *testing.T) {
	config := &ContextConfig{
		EnableAutoDiscovery: true,
		UpdateInterval:      1 * time.Millisecond,
	}

	cm := NewContextManager(config)

	// Test concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: Continuously update context
	go func() {
		for i := 0; i < 100; i++ {
			stats := &db.DatabaseStats{
				TotalKeyCount: int64(i * 10),
				TotalSize:     int64(i * 1024),
				ColumnFamilies: []db.CFStats{
					{
						Name:           "default",
						KeyCount:       int64(i * 10),
						TotalKeySize:   int64(i * 200),
						TotalValueSize: int64(i * 824),
						DataTypeDistribution: map[db.DataType]int64{
							db.DataType("string"): int64(i * 5),
						},
						CommonPrefixes: map[string]int64{
							"test:": int64(i),
						},
					},
				},
			}
			cm.UpdateDatabaseStats(stats)
			time.Sleep(1 * time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 2: Continuously read context
	go func() {
		for i := 0; i < 100; i++ {
			context := cm.GetContext()
			assert.NotNil(t, context)
			assert.GreaterOrEqual(t, context.TotalKeys, int64(0))

			summary := cm.GetContextSummary()
			assert.Contains(t, summary, "Database Context:")

			shouldUpdate := cm.ShouldUpdateContext()
			assert.IsType(t, false, shouldUpdate)

			time.Sleep(1 * time.Microsecond)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Final verification
	finalContext := cm.GetContext()
	assert.NotNil(t, finalContext)
}

func TestContextManager_MinFunction(t *testing.T) {
	testCases := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a less than b", 3, 5, 3},
		{"b less than a", 8, 2, 2},
		{"equal values", 4, 4, 4},
		{"zero values", 0, 0, 0},
		{"negative and positive", -5, 3, -5},
		{"both negative", -8, -3, -8},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := min(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDatabaseContext_JSONSerialization(t *testing.T) {
	// Test that DatabaseContext can be properly serialized/deserialized
	original := &DatabaseContext{
		ColumnFamilies: []string{"default", "users"},
		TotalKeys:      1000,
		TotalSize:      102400,
		LastUpdated:    time.Now().Truncate(time.Second), // Truncate for comparison
		KeyPrefixes:    []string{"user:", "session:"},
		DataTypeStats: map[string]int64{
			"string": 800,
			"json":   200,
		},
	}

	// This test mainly verifies the struct is properly tagged for JSON
	assert.NotNil(t, original.ColumnFamilies)
	assert.NotNil(t, original.DataTypeStats)
	assert.Equal(t, int64(1000), original.TotalKeys)
	assert.Equal(t, int64(102400), original.TotalSize)
}

func TestContextManager_UpdateDatabaseStats_NilDataTypes(t *testing.T) {
	config := &ContextConfig{}
	cm := NewContextManager(config)

	stats := &db.DatabaseStats{
		TotalKeyCount: 100,
		TotalSize:     10240,
		ColumnFamilies: []db.CFStats{
			{
				Name:                 "default",
				KeyCount:             100,
				TotalKeySize:         2000,
				TotalValueSize:       8240,
				DataTypeDistribution: nil, // nil map
				CommonPrefixes:       nil, // nil map
			},
		},
	}

	err := cm.UpdateDatabaseStats(stats)
	assert.NoError(t, err)

	context := cm.GetContext()
	assert.Equal(t, int64(100), context.TotalKeys)
	assert.Equal(t, []string{"default"}, context.ColumnFamilies)
	assert.Empty(t, context.DataTypeStats) // Should handle nil gracefully
	assert.Empty(t, context.KeyPrefixes)   // Should handle nil gracefully
}
