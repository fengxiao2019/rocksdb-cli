package graphchain

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"rocksdb-cli/internal/db"
)

// ToolInput represents the input structure for tools
type ToolInput struct {
	Args map[string]interface{} `json:"args"`
}

// parseToolInput parses the input string into structured data
func parseToolInput(input string) (*ToolInput, error) {
	var toolInput ToolInput
	if err := json.Unmarshal([]byte(input), &toolInput); err != nil {
		// If JSON parsing fails, try to parse as simple key-value
		toolInput.Args = map[string]interface{}{
			"input": strings.TrimSpace(input),
		}
	}
	return &toolInput, nil
}

// getString extracts string value from args
func getString(args map[string]interface{}, key string) string {
	if val, ok := args[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getInt extracts int value from args
func getInt(args map[string]interface{}, key string) int {
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return 0
}

// getBool extracts bool value from args
func getBool(args map[string]interface{}, key string) bool {
	if val, ok := args[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// GetValueTool implements tools.Tool interface for getting values
type GetValueTool struct {
	db *db.DB
}

func NewGetValueTool(database *db.DB) *GetValueTool {
	return &GetValueTool{db: database}
}

func (t *GetValueTool) Call(ctx context.Context, input string) (string, error) {
	toolInput, err := parseToolInput(input)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	key := getString(toolInput.Args, "key")
	if key == "" {
		return "", fmt.Errorf("key parameter is required")
	}

	cf := getString(toolInput.Args, "column_family")
	if cf == "" {
		cf = "default"
	}

	value, err := t.db.GetCF(cf, key)
	if err != nil {
		return "", fmt.Errorf("failed to get value: %w", err)
	}

	result := map[string]interface{}{
		"key":           key,
		"value":         value,
		"column_family": cf,
	}

	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

func (t *GetValueTool) Name() string {
	return "get_value"
}

func (t *GetValueTool) Description() string {
	return `Get a value by key.
【English】
Get value by key.
Args:
  - key (string, required)
  - column_family (string, optional, default: "default")
Returns: JSON {key, value, column_family}`
}

// PutValueTool implements tools.Tool interface for putting values
type PutValueTool struct {
	db *db.DB
}

func NewPutValueTool(database *db.DB) *PutValueTool {
	return &PutValueTool{db: database}
}

func (t *PutValueTool) Call(ctx context.Context, input string) (string, error) {
	toolInput, err := parseToolInput(input)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	key := getString(toolInput.Args, "key")
	value := getString(toolInput.Args, "value")
	if key == "" || value == "" {
		return "", fmt.Errorf("both key and value parameters are required")
	}

	cf := getString(toolInput.Args, "column_family")
	if cf == "" {
		cf = "default"
	}

	err = t.db.PutCF(cf, key, value)
	if err != nil {
		return "", fmt.Errorf("failed to put value: %w", err)
	}

	result := map[string]interface{}{
		"success":       true,
		"key":           key,
		"value":         value,
		"column_family": cf,
	}

	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

func (t *PutValueTool) Name() string {
	return "put_value"
}

func (t *PutValueTool) Description() string {
	return `Put a key-value pair.
【English】
Put key-value pair.
Args:
  - key (string, required)
  - value (string, required)
  - column_family (string, optional, default: "default")
Returns: JSON {success, key, value, column_family}`
}

// ScanRangeTool implements tools.Tool interface for range scanning
type ScanRangeTool struct {
	db *db.DB
}

func NewScanRangeTool(database *db.DB) *ScanRangeTool {
	return &ScanRangeTool{db: database}
}

func (t *ScanRangeTool) Call(ctx context.Context, input string) (string, error) {
	toolInput, err := parseToolInput(input)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	startKey := getString(toolInput.Args, "start_key")
	endKey := getString(toolInput.Args, "end_key")
	cf := getString(toolInput.Args, "column_family")
	if cf == "" {
		cf = "default"
	}

	limit := getInt(toolInput.Args, "limit")
	if limit <= 0 {
		limit = 10
	}

	reverse := getBool(toolInput.Args, "reverse")

	opts := db.ScanOptions{
		Limit:   limit,
		Reverse: reverse,
		Values:  true,
	}

	results, err := t.db.ScanCF(cf, []byte(startKey), []byte(endKey), opts)
	if err != nil {
		return "", fmt.Errorf("failed to scan range: %w", err)
	}

	result := map[string]interface{}{
		"results":       results,
		"count":         len(results),
		"start_key":     startKey,
		"end_key":       endKey,
		"column_family": cf,
		"limit":         limit,
		"reverse":       reverse,
	}

	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

func (t *ScanRangeTool) Name() string {
	return "scan_range"
}

// 优化描述：强调 scan_range 可用于全量遍历，prefix_scan 仅用于特定前缀，便于 LLM 区分
func (t *ScanRangeTool) Description() string {
	return `Scan a range of keys.
【English】
Scan a range of keys, or all keys if start_key and end_key are empty.
Args:
  - start_key (string, required, use "" for beginning)
  - end_key (string, required, use "" for end)
  - column_family (string, optional, default: "default")
  - limit (int, optional, default: 10)
  - reverse (bool, optional, default: false)
Returns: JSON {results, count, start_key, end_key, column_family, limit, reverse}
Note: To get all keys in a column family, set start_key and end_key to "".

Example: To get all keys from 'users', set start_key="", end_key="", column_family="users".

【中文】
扫描一段 key 范围，若 start_key 和 end_key 为空则遍历所有 key。
示例：获取 users 下所有 key，start_key=""，end_key=""，column_family="users"。
`
}

// PrefixScanTool implements tools.Tool interface for prefix scanning
type PrefixScanTool struct {
	db *db.DB
}

func NewPrefixScanTool(database *db.DB) *PrefixScanTool {
	return &PrefixScanTool{db: database}
}

func (t *PrefixScanTool) Call(ctx context.Context, input string) (string, error) {
	toolInput, err := parseToolInput(input)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	prefix := getString(toolInput.Args, "prefix")
	if prefix == "" {
		return "", fmt.Errorf("prefix parameter is required")
	}

	cf := getString(toolInput.Args, "column_family")
	if cf == "" {
		cf = "default"
	}

	limit := getInt(toolInput.Args, "limit")
	if limit <= 0 {
		limit = 10
	}

	results, err := t.db.PrefixScanCF(cf, prefix, limit)
	if err != nil {
		return "", fmt.Errorf("failed to scan prefix: %w", err)
	}

	result := map[string]interface{}{
		"results":       results,
		"count":         len(results),
		"prefix":        prefix,
		"column_family": cf,
		"limit":         limit,
	}

	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

func (t *PrefixScanTool) Name() string {
	return "prefix_scan"
}

// 优化描述：prefix_scan 仅用于前缀批量查询，避免误用
func (t *PrefixScanTool) Description() string {
	return `Scan keys with a specific prefix.
【English】
Scan keys by prefix. Only returns keys that start with the given prefix.
Args:
  - prefix (string, required)
  - column_family (string, optional, default: "default")
  - limit (int, optional, default: 10)
Returns: JSON {results, count, prefix, column_family, limit}
Note: Use this only if you want keys starting with a specific prefix.

Example: To get all keys starting with "user:", set prefix="user:".

【中文】
按前缀批量扫描 key，仅返回以 prefix 开头的 key。
示例：获取所有以 user: 开头的 key，prefix="user:"。
`
}

// ListColumnFamiliesTool implements tools.Tool interface for listing column families
type ListColumnFamiliesTool struct {
	db *db.DB
}

func NewListColumnFamiliesTool(database *db.DB) *ListColumnFamiliesTool {
	return &ListColumnFamiliesTool{db: database}
}

func (t *ListColumnFamiliesTool) Call(ctx context.Context, input string) (string, error) {
	cfs, err := t.db.ListCFs()
	if err != nil {
		return "", fmt.Errorf("failed to list column families: %w", err)
	}

	result := map[string]interface{}{
		"column_families": cfs,
		"count":           len(cfs),
	}

	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

func (t *ListColumnFamiliesTool) Name() string {
	return "list_column_families"
}

func (t *ListColumnFamiliesTool) Description() string {
	return `List all column families.
Args: none
Returns: JSON {column_families, count}`
}

// GetLastTool implements tools.Tool interface for getting the last key-value pair
type GetLastTool struct {
	db *db.DB
}

func NewGetLastTool(database *db.DB) *GetLastTool {
	return &GetLastTool{db: database}
}

func (t *GetLastTool) Call(ctx context.Context, input string) (string, error) {
	toolInput, err := parseToolInput(input)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	cf := getString(toolInput.Args, "column_family")
	if cf == "" {
		cf = "default"
	}

	key, value, err := t.db.GetLastCF(cf)
	if err != nil {
		return "", fmt.Errorf("failed to get last entry: %w", err)
	}

	result := map[string]interface{}{
		"key":           key,
		"value":         value,
		"column_family": cf,
	}

	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

func (t *GetLastTool) Name() string {
	return "get_last"
}

func (t *GetLastTool) Description() string {
	return `Get the last key-value pair.
Get last key-value pair in a column family.
Args:
  - column_family (string, optional, default: "default")
Returns: JSON {key, value, column_family}`
}

// JSONQueryTool implements tools.Tool interface for JSON field queries
type JSONQueryTool struct {
	db *db.DB
}

func NewJSONQueryTool(database *db.DB) *JSONQueryTool {
	return &JSONQueryTool{db: database}
}

func (t *JSONQueryTool) Call(ctx context.Context, input string) (string, error) {
	toolInput, err := parseToolInput(input)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	field := getString(toolInput.Args, "field")
	value := getString(toolInput.Args, "value")
	if field == "" || value == "" {
		return "", fmt.Errorf("both field and value parameters are required")
	}

	cf := getString(toolInput.Args, "column_family")
	if cf == "" {
		cf = "default"
	}

	results, err := t.db.JSONQueryCF(cf, field, value)
	if err != nil {
		return "", fmt.Errorf("failed to query JSON: %w", err)
	}

	result := map[string]interface{}{
		"results":       results,
		"count":         len(results),
		"field":         field,
		"value":         value,
		"column_family": cf,
	}

	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

func (t *JSONQueryTool) Name() string {
	return "json_query"
}

func (t *JSONQueryTool) Description() string {
	return `Query JSON values by field.
Query JSON values by field.
Args:
  - field (string, required)
  - value (string, required)
  - column_family (string, optional, default: "default")
  - limit (int, optional, default: 10)
Returns: JSON {results, count, field, value, column_family}`
}

// GetStatsTool implements tools.Tool interface for getting database statistics
type GetStatsTool struct {
	db *db.DB
}

func NewGetStatsTool(database *db.DB) *GetStatsTool {
	return &GetStatsTool{db: database}
}

func (t *GetStatsTool) Call(ctx context.Context, input string) (string, error) {
	stats, err := t.db.GetDatabaseStats()
	if err != nil {
		return "", fmt.Errorf("failed to get stats: %w", err)
	}

	result := map[string]interface{}{
		"stats": stats,
	}

	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

func (t *GetStatsTool) Name() string {
	return "get_stats"
}

func (t *GetStatsTool) Description() string {
	return `Get RocksDB database statistics.
Get database statistics.
Args: none
Returns: JSON {stats}`
}

// SearchTool implements tools.Tool interface for complex search
type SearchTool struct {
	db *db.DB
}

func NewSearchTool(database *db.DB) *SearchTool {
	return &SearchTool{db: database}
}

func (t *SearchTool) Call(ctx context.Context, input string) (string, error) {
	toolInput, err := parseToolInput(input)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}
	args := toolInput.Args
	cf := getString(args, "column_family")
	if cf == "" {
		cf = "default"
	}
	keyPattern := getString(args, "key_pattern")
	valuePattern := getString(args, "value_pattern")
	useRegex := getBool(args, "regex")
	limit := getInt(args, "limit")
	if limit <= 0 {
		limit = 10
	}
	after := getString(args, "after")

	opts := db.SearchOptions{
		KeyPattern:    keyPattern,
		ValuePattern:  valuePattern,
		UseRegex:      useRegex,
		CaseSensitive: false,
		Limit:         limit,
		KeysOnly:      false,
		After:         after,
	}
	results, err := t.db.SearchCF(cf, opts)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}
	// 组装结果
	resultMap := make(map[string]string)
	for _, r := range results.Results {
		resultMap[r.Key] = r.Value
	}
	result := map[string]interface{}{
		"results":     resultMap,
		"count":       len(resultMap),
		"next_cursor": results.NextCursor,
		"has_more":    results.HasMore,
	}
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

func (t *SearchTool) Name() string {
	return "search"
}

func (t *SearchTool) Description() string {
	return `复杂搜索。
Search keys/values with pattern or regex.
Args:
  - key_pattern (string, optional)
  - value_pattern (string, optional)
  - column_family (string, optional, default: "default")
  - limit (int, optional, default: 10)
  - after (string, optional)
  - regex (bool, optional, default: false)
Returns: JSON {results, count, next_cursor, has_more}`
}

// CreateRocksDBTools creates all RocksDB tools for the agent
func CreateRocksDBTools(database *db.DB) []interface{} {
	// Note: Using interface{} here because tools.Tool interface might be different
	// We'll need to import the correct tools package
	return []interface{}{
		NewGetValueTool(database),
		NewPutValueTool(database),
		NewScanRangeTool(database),
		NewPrefixScanTool(database),
		NewListColumnFamiliesTool(database),
		NewGetLastTool(database),
		NewJSONQueryTool(database),
		NewGetStatsTool(database),
		NewSearchTool(database),
	}
}
