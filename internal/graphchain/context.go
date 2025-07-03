package graphchain

import (
	"fmt"
	"sync"
	"time"

	"rocksdb-cli/internal/db"
)

// DatabaseContext represents the current database context
type DatabaseContext struct {
	ColumnFamilies []string         `json:"column_families"`
	TotalKeys      int64            `json:"total_keys"`
	TotalSize      int64            `json:"total_size"`
	LastUpdated    time.Time        `json:"last_updated"`
	KeyPrefixes    []string         `json:"key_prefixes"`
	DataTypeStats  map[string]int64 `json:"data_type_stats"`
}

// ContextManager manages database context information
type ContextManager struct {
	config  *ContextConfig
	context *DatabaseContext
	mutex   sync.RWMutex
}

// NewContextManager creates a new context manager
func NewContextManager(config *ContextConfig) *ContextManager {
	return &ContextManager{
		config: config,
		context: &DatabaseContext{
			ColumnFamilies: []string{},
			TotalKeys:      0,
			TotalSize:      0,
			LastUpdated:    time.Now(),
			KeyPrefixes:    []string{},
			DataTypeStats:  make(map[string]int64),
		},
	}
}

// GetContext returns the current database context
func (cm *ContextManager) GetContext() *DatabaseContext {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Return a copy to prevent concurrent modification
	contextCopy := *cm.context
	return &contextCopy
}

// UpdateDatabaseStats updates the context with new database statistics
func (cm *ContextManager) UpdateDatabaseStats(stats *db.DatabaseStats) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Update column families
	cm.context.ColumnFamilies = make([]string, 0, len(stats.ColumnFamilies))
	for _, cf := range stats.ColumnFamilies {
		cm.context.ColumnFamilies = append(cm.context.ColumnFamilies, cf.Name)
	}

	// Update totals
	cm.context.TotalKeys = stats.TotalKeyCount
	cm.context.TotalSize = stats.TotalSize
	cm.context.LastUpdated = time.Now()

	// Update data type statistics
	cm.context.DataTypeStats = make(map[string]int64)
	for _, cf := range stats.ColumnFamilies {
		for dataType, count := range cf.DataTypeDistribution {
			cm.context.DataTypeStats[string(dataType)] += count
		}
	}

	// Extract common key prefixes
	cm.context.KeyPrefixes = make([]string, 0)
	for _, cf := range stats.ColumnFamilies {
		for prefix := range cf.CommonPrefixes {
			cm.context.KeyPrefixes = append(cm.context.KeyPrefixes, prefix)
		}
	}

	return nil
}

// ShouldUpdateContext checks if context should be updated based on configuration
func (cm *ContextManager) ShouldUpdateContext() bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if !cm.config.EnableAutoDiscovery {
		return false
	}

	return time.Since(cm.context.LastUpdated) > cm.config.UpdateInterval
}

// GetContextSummary returns a human-readable summary of the database context
func (cm *ContextManager) GetContextSummary() string {
	context := cm.GetContext()

	summary := "Database Context:\n"
	summary += fmt.Sprintf("- Column Families: %d (%v)\n", len(context.ColumnFamilies), context.ColumnFamilies)
	summary += fmt.Sprintf("- Total Keys: %d\n", context.TotalKeys)
	summary += fmt.Sprintf("- Total Size: %d bytes\n", context.TotalSize)
	summary += fmt.Sprintf("- Last Updated: %s\n", context.LastUpdated.Format(time.RFC3339))

	if len(context.KeyPrefixes) > 0 {
		summary += fmt.Sprintf("- Common Key Prefixes: %v\n", context.KeyPrefixes[:min(5, len(context.KeyPrefixes))])
	}

	if len(context.DataTypeStats) > 0 {
		summary += "- Data Type Distribution:\n"
		for dataType, count := range context.DataTypeStats {
			summary += fmt.Sprintf("  - %s: %d\n", dataType, count)
		}
	}

	return summary
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
