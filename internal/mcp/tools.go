package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/jsonutil"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolManager manages MCP tools for RocksDB operations
type ToolManager struct {
	db     db.KeyValueDB
	config *Config
}

// NewToolManager creates a new tool manager
func NewToolManager(database db.KeyValueDB, config *Config) *ToolManager {
	return &ToolManager{
		db:     database,
		config: config,
	}
}

// RegisterTools registers all available tools with the MCP server
func (tm *ToolManager) RegisterTools(s *server.MCPServer) error {
	// RocksDB Get Tool
	getRocksDBTool := mcp.NewTool("rocksdb_get",
		mcp.WithDescription("Get a value by key from RocksDB"),
		mcp.WithString("key",
			mcp.Required(),
			mcp.Description("The key to retrieve"),
		),
		mcp.WithString("column_family",
			mcp.Description("Column family name (defaults to 'default')"),
		),
		mcp.WithBoolean("pretty",
			mcp.Description("Pretty print JSON values"),
		),
	)
	s.AddTool(getRocksDBTool, tm.handleGetTool)

	// RocksDB Put Tool (only if not read-only)
	if !tm.config.ReadOnly {
		putRocksDBTool := mcp.NewTool("rocksdb_put",
			mcp.WithDescription("Put a key-value pair into RocksDB"),
			mcp.WithString("key",
				mcp.Required(),
				mcp.Description("The key to set"),
			),
			mcp.WithString("value",
				mcp.Required(),
				mcp.Description("The value to set"),
			),
			mcp.WithString("column_family",
				mcp.Description("Column family name (defaults to 'default')"),
			),
		)
		s.AddTool(putRocksDBTool, tm.handlePutTool)
	}

	// RocksDB Scan Tool
	scanRocksDBTool := mcp.NewTool("rocksdb_scan",
		mcp.WithDescription("Scan a range of keys from RocksDB"),
		mcp.WithString("column_family",
			mcp.Description("Column family name (defaults to 'default')"),
		),
		mcp.WithString("start_key",
			mcp.Description("Start key for range scan"),
		),
		mcp.WithString("end_key",
			mcp.Description("End key for range scan"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results"),
		),
		mcp.WithBoolean("reverse",
			mcp.Description("Scan in reverse order"),
		),
		mcp.WithBoolean("values_only",
			mcp.Description("Return only keys, not values"),
		),
	)
	s.AddTool(scanRocksDBTool, tm.handleScanTool)

	// RocksDB Prefix Scan Tool
	prefixScanTool := mcp.NewTool("rocksdb_prefix_scan",
		mcp.WithDescription("Scan keys with a specific prefix from RocksDB"),
		mcp.WithString("prefix",
			mcp.Required(),
			mcp.Description("Key prefix to search for"),
		),
		mcp.WithString("column_family",
			mcp.Description("Column family name (defaults to 'default')"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results"),
		),
	)
	s.AddTool(prefixScanTool, tm.handlePrefixScanTool)

	// List Column Families Tool
	listCFTool := mcp.NewTool("rocksdb_list_column_families",
		mcp.WithDescription("List all column families in the database"),
	)
	s.AddTool(listCFTool, tm.handleListCFTool)

	// Create Column Family Tool (only if not read-only)
	if !tm.config.ReadOnly {
		createCFTool := mcp.NewTool("rocksdb_create_column_family",
			mcp.WithDescription("Create a new column family"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the column family to create"),
			),
		)
		s.AddTool(createCFTool, tm.handleCreateCFTool)

		// Drop Column Family Tool
		dropCFTool := mcp.NewTool("rocksdb_drop_column_family",
			mcp.WithDescription("Drop an existing column family"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the column family to drop"),
			),
		)
		s.AddTool(dropCFTool, tm.handleDropCFTool)
	}

	// Export to CSV Tool
	exportCSVTool := mcp.NewTool("rocksdb_export_to_csv",
		mcp.WithDescription("Export column family data to CSV file"),
		mcp.WithString("column_family",
			mcp.Required(),
			mcp.Description("Column family name to export"),
		),
		mcp.WithString("output_file",
			mcp.Required(),
			mcp.Description("Output CSV file path"),
		),
	)
	s.AddTool(exportCSVTool, tm.handleExportCSVTool)

	// JSON Query Tool
	jsonQueryTool := mcp.NewTool("rocksdb_json_query",
		mcp.WithDescription("Query JSON values by field"),
		mcp.WithString("column_family",
			mcp.Description("Column family name (defaults to 'default')"),
		),
		mcp.WithString("field",
			mcp.Required(),
			mcp.Description("JSON field to query"),
		),
		mcp.WithString("value",
			mcp.Required(),
			mcp.Description("Value to search for"),
		),
		mcp.WithBoolean("pretty",
			mcp.Description("Pretty print JSON values"),
		),
	)
	s.AddTool(jsonQueryTool, tm.handleJSONQueryTool)

	// Get Last Tool
	getLastTool := mcp.NewTool("rocksdb_get_last",
		mcp.WithDescription("Get the last key-value pair from a column family"),
		mcp.WithString("column_family",
			mcp.Description("Column family name (defaults to 'default')"),
		),
		mcp.WithBoolean("pretty",
			mcp.Description("Pretty print JSON values"),
		),
	)
	s.AddTool(getLastTool, tm.handleGetLastTool)

	// RocksDB Search Tool
	searchTool := mcp.NewTool("rocksdb_search",
		mcp.WithDescription("Fuzzy search for keys and values in RocksDB"),
		mcp.WithString("column_family",
			mcp.Description("Column family name (defaults to 'default')"),
		),
		mcp.WithString("key_pattern",
			mcp.Description("Key pattern to search for"),
		),
		mcp.WithString("value_pattern",
			mcp.Description("Value pattern to search for"),
		),
		mcp.WithBoolean("use_regex",
			mcp.Description("Use regex patterns instead of wildcards"),
		),
		mcp.WithBoolean("case_sensitive",
			mcp.Description("Case sensitive search"),
		),
		mcp.WithBoolean("keys_only",
			mcp.Description("Return only keys, not values"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results (default: 50)"),
		),
		mcp.WithBoolean("pretty",
			mcp.Description("Pretty print JSON values"),
		),
	)
	s.AddTool(searchTool, tm.handleSearchTool)

	// RocksDB Statistics Tool
	statsTool := mcp.NewTool("rocksdb_get_stats",
		mcp.WithDescription("Get database or column family statistics"),
		mcp.WithString("column_family",
			mcp.Description("Column family name (omit for database-wide stats)"),
		),
		mcp.WithBoolean("detailed",
			mcp.Description("Show detailed statistics"),
		),
		mcp.WithBoolean("pretty",
			mcp.Description("Pretty print JSON format"),
		),
	)
	s.AddTool(statsTool, tm.handleStatsTool)

	return nil
}

// Tool handlers

func (tm *ToolManager) handleGetTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cf := request.GetString("column_family", "default")

	pretty := request.GetBool("pretty", false)

	value, err := tm.db.GetCF(cf, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get key '%s' from CF '%s': %v", key, cf, err)), nil
	}

	result := value
	if pretty {
		result = tm.formatJSONValue(value)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Key: %s\nValue: %s", key, result)), nil
}

func (tm *ToolManager) handlePutTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tm.config.ReadOnly {
		return mcp.NewToolResultError("Write operations are not allowed in read-only mode"), nil
	}

	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	value, err := request.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cf := request.GetString("column_family", "default")

	if err := tm.db.PutCF(cf, key, value); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to put key '%s' to CF '%s': %v", key, cf, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully stored key '%s' in column family '%s'", key, cf)), nil
}

func (tm *ToolManager) handleScanTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cf := request.GetString("column_family", "default")

	startKey := request.GetString("start_key", "")
	endKey := request.GetString("end_key", "")
	limitFloat := request.GetFloat("limit", 0.0)
	reverse := request.GetBool("reverse", false)
	valuesOnly := request.GetBool("values_only", false)

	limit := int(limitFloat)
	if limit <= 0 {
		limit = 100 // Default limit
	}

	opts := db.ScanOptions{
		Limit:   limit,
		Reverse: reverse,
		Values:  !valuesOnly,
	}

	results, err := tm.db.ScanCF(cf, []byte(startKey), []byte(endKey), opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to scan CF '%s': %v", cf, err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No results found"), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Scan results from column family '%s' (%d results):\n", cf, len(results)))

	for key, value := range results {
		if valuesOnly {
			output.WriteString(fmt.Sprintf("Key: %s\n", key))
		} else {
			output.WriteString(fmt.Sprintf("Key: %s | Value: %s\n", key, value))
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}

func (tm *ToolManager) handlePrefixScanTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prefix, err := request.RequireString("prefix")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cf := request.GetString("column_family", "default")

	limitFloat := request.GetFloat("limit", 0.0)
	limit := int(limitFloat)
	if limit <= 0 {
		limit = 100 // Default limit
	}

	results, err := tm.db.PrefixScanCF(cf, prefix, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to prefix scan CF '%s': %v", cf, err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No keys found with prefix '%s'", prefix)), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Prefix scan results for '%s' in column family '%s' (%d results):\n", prefix, cf, len(results)))

	for key, value := range results {
		output.WriteString(fmt.Sprintf("Key: %s | Value: %s\n", key, value))
	}

	return mcp.NewToolResultText(output.String()), nil
}

func (tm *ToolManager) handleListCFTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfs, err := tm.db.ListCFs()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list column families: %v", err)), nil
	}

	if len(cfs) == 0 {
		return mcp.NewToolResultText("No column families found"), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Column families (%d total):\n", len(cfs)))
	for _, cf := range cfs {
		output.WriteString(fmt.Sprintf("- %s\n", cf))
	}

	return mcp.NewToolResultText(output.String()), nil
}

func (tm *ToolManager) handleCreateCFTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tm.config.ReadOnly {
		return mcp.NewToolResultError("Write operations are not allowed in read-only mode"), nil
	}

	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := tm.db.CreateCF(name); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create column family '%s': %v", name, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully created column family '%s'", name)), nil
}

func (tm *ToolManager) handleDropCFTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tm.config.ReadOnly {
		return mcp.NewToolResultError("Write operations are not allowed in read-only mode"), nil
	}

	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := tm.db.DropCF(name); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to drop column family '%s': %v", name, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully dropped column family '%s'", name)), nil
}

func (tm *ToolManager) handleExportCSVTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cf, err := request.RequireString("column_family")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	outputFile, err := request.RequireString("output_file")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := tm.db.ExportToCSV(cf, outputFile, ","); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to export CF '%s' to '%s': %v", cf, outputFile, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully exported column family '%s' to '%s'", cf, outputFile)), nil
}

func (tm *ToolManager) handleJSONQueryTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cf := request.GetString("column_family", "default")

	field, err := request.RequireString("field")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	value, err := request.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pretty := request.GetBool("pretty", false)

	results, err := tm.db.JSONQueryCF(cf, field, value)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query JSON field '%s' in CF '%s': %v", field, cf, err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No results found for field '%s' = '%s'", field, value)), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("JSON query results for field '%s' = '%s' in column family '%s' (%d results):\n", field, value, cf, len(results)))

	for key, val := range results {
		resultValue := val
		if pretty {
			resultValue = tm.formatJSONValue(val)
		}
		output.WriteString(fmt.Sprintf("Key: %s | Value: %s\n", key, resultValue))
	}

	return mcp.NewToolResultText(output.String()), nil
}

func (tm *ToolManager) handleGetLastTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cf := request.GetString("column_family", "default")

	pretty := request.GetBool("pretty", false)

	key, value, err := tm.db.GetLastCF(cf)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get last entry from CF '%s': %v", cf, err)), nil
	}

	result := value
	if pretty {
		result = tm.formatJSONValue(value)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Last entry in column family '%s':\nKey: %s\nValue: %s", cf, key, result)), nil
}

// handleSearchTool performs fuzzy search in RocksDB
func (tm *ToolManager) handleSearchTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cf := request.GetString("column_family", "default")
	keyPattern := request.GetString("key_pattern", "")
	valuePattern := request.GetString("value_pattern", "")
	useRegex := request.GetBool("use_regex", false)
	caseSensitive := request.GetBool("case_sensitive", false)
	keysOnly := request.GetBool("keys_only", false)
	pretty := request.GetBool("pretty", false)

	limitFloat := request.GetFloat("limit", 50.0)
	limit := int(limitFloat)
	if limit <= 0 {
		limit = 50
	}

	// Validate that at least one pattern is provided
	if keyPattern == "" && valuePattern == "" {
		return mcp.NewToolResultError("Must specify at least key_pattern or value_pattern"), nil
	}

	// Set up search options
	searchOpts := db.SearchOptions{
		KeyPattern:    keyPattern,
		ValuePattern:  valuePattern,
		UseRegex:      useRegex,
		CaseSensitive: caseSensitive,
		KeysOnly:      keysOnly,
		Limit:         limit,
	}

	// Execute search
	results, err := tm.db.SearchCF(cf, searchOpts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search CF '%s': %v", cf, err)), nil
	}

	if len(results.Results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No matches found in column family '%s'\nQuery took: %s", cf, results.QueryTime)), nil
	}

	// Format output
	var output strings.Builder
	limitedText := ""
	if results.Limited {
		limitedText = " (limited)"
	}

	output.WriteString(fmt.Sprintf("Search results in column family '%s' (%d matches%s, query took: %s):\n\n",
		cf, len(results.Results), limitedText, results.QueryTime))

	for i, result := range results.Results {
		// Format matched fields display
		matchedFieldsStr := ""
		if len(result.MatchedFields) > 0 {
			matchedFieldsStr = fmt.Sprintf(" (matched: %v)", result.MatchedFields)
		}

		output.WriteString(fmt.Sprintf("[%d] Key: %s%s\n", i+1, result.Key, matchedFieldsStr))

		if !keysOnly {
			resultValue := result.Value
			if pretty {
				resultValue = tm.formatJSONValue(result.Value)
			}
			output.WriteString(fmt.Sprintf("    Value: %s\n", resultValue))
		}
		if i < len(results.Results)-1 {
			output.WriteString("\n")
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}

// handleStatsTool gets database or column family statistics
func (tm *ToolManager) handleStatsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cf := request.GetString("column_family", "")
	detailed := request.GetBool("detailed", false)
	pretty := request.GetBool("pretty", false)

	if cf == "" {
		// Database-wide statistics
		stats, err := tm.db.GetDatabaseStats()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get database stats: %v", err)), nil
		}

		if pretty {
			// Return JSON format
			jsonBytes, err := json.Marshal(stats)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal stats to JSON: %v", err)), nil
			}
			return mcp.NewToolResultText(tm.formatJSONValue(string(jsonBytes))), nil
		}

		// Format human-readable output
		var output strings.Builder
		output.WriteString("=== Database Statistics ===\n")
		output.WriteString(fmt.Sprintf("Column Families: %d\n", stats.ColumnFamilyCount))
		output.WriteString(fmt.Sprintf("Total Keys: %s\n", tm.formatNumber(stats.TotalKeyCount)))
		output.WriteString(fmt.Sprintf("Total Size: %s\n", tm.formatBytes(stats.TotalSize)))

		if detailed && len(stats.ColumnFamilies) > 0 {
			output.WriteString("\nColumn Family Details:\n")
			for _, cfStats := range stats.ColumnFamilies {
				output.WriteString(fmt.Sprintf("- %s: %s keys, %s\n",
					cfStats.Name,
					tm.formatNumber(cfStats.KeyCount),
					tm.formatBytes(cfStats.TotalKeySize+cfStats.TotalValueSize)))
			}
		}

		return mcp.NewToolResultText(output.String()), nil
	} else {
		// Column family specific statistics
		stats, err := tm.db.GetCFStats(cf)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stats for CF '%s': %v", cf, err)), nil
		}

		if pretty {
			// Return JSON format
			jsonBytes, err := json.Marshal(stats)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal stats to JSON: %v", err)), nil
			}
			return mcp.NewToolResultText(tm.formatJSONValue(string(jsonBytes))), nil
		}

		// Format human-readable output
		var output strings.Builder
		output.WriteString(fmt.Sprintf("=== Column Family: %s ===\n", stats.Name))
		output.WriteString(fmt.Sprintf("Keys: %s\n", tm.formatNumber(stats.KeyCount)))
		output.WriteString(fmt.Sprintf("Total Key Size: %s\n", tm.formatBytes(stats.TotalKeySize)))
		output.WriteString(fmt.Sprintf("Total Value Size: %s\n", tm.formatBytes(stats.TotalValueSize)))

		if stats.KeyCount > 0 {
			output.WriteString(fmt.Sprintf("Average Key Size: %.1f bytes\n", stats.AverageKeySize))
			output.WriteString(fmt.Sprintf("Average Value Size: %.1f bytes\n", stats.AverageValueSize))
		}

		// Data type distribution
		if len(stats.DataTypeDistribution) > 0 {
			output.WriteString("\nData Type Distribution:\n")
			for dataType, count := range stats.DataTypeDistribution {
				percentage := float64(count) / float64(stats.KeyCount) * 100
				output.WriteString(fmt.Sprintf("  %-20s %s (%5.1f%%)\n",
					dataType, tm.formatNumber(count), percentage))
			}
		}

		// Additional details if requested
		if detailed {
			if len(stats.CommonPrefixes) > 0 {
				output.WriteString("\nCommon Key Prefixes:\n")
				for prefix, count := range stats.CommonPrefixes {
					output.WriteString(fmt.Sprintf("  %-20s %s\n", prefix, tm.formatNumber(count)))
				}
			}

			if len(stats.SampleKeys) > 0 {
				output.WriteString("\nSample Keys:\n")
				for _, key := range stats.SampleKeys {
					output.WriteString(fmt.Sprintf("  %s\n", key))
				}
			}
		}

		return mcp.NewToolResultText(output.String()), nil
	}
}

// formatJSONValue formats JSON values with recursive nested JSON expansion using jsonutil
func (tm *ToolManager) formatJSONValue(value string) string {
	return jsonutil.PrettyPrintWithNestedExpansion(value)
}

// formatNumber formats numbers with K/M/B suffixes for readability
func (tm *ToolManager) formatNumber(num int64) string {
	if num < 1000 {
		return fmt.Sprintf("%d", num)
	} else if num < 1000000 {
		return fmt.Sprintf("%.1fK", float64(num)/1000)
	} else if num < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(num)/1000000)
	} else {
		return fmt.Sprintf("%.1fB", float64(num)/1000000000)
	}
}

// formatBytes formats byte counts with appropriate units
func (tm *ToolManager) formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
	}
}

// IsToolEnabled checks if a specific tool is enabled based on configuration
func (tm *ToolManager) IsToolEnabled(toolName string) bool {
	if tm.config.EnableAllTools {
		// Check if it's in the disabled list
		for _, disabled := range tm.config.DisabledTools {
			if disabled == toolName {
				return false
			}
		}
		return true
	}

	// Check if it's in the enabled list
	for _, enabled := range tm.config.EnabledTools {
		if enabled == toolName {
			return true
		}
	}

	return false
}
