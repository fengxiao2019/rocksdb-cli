package tools

import (
	"context"
	"fmt"

	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/mcp/protocol"
)

// LocalAdapter provides adapters for rocksdb-cli tools to work with MCP
type LocalAdapter struct {
	registry *Registry
	db       *db.DB
}

// NewLocalAdapter creates a new local tool adapter
// The db parameter is optional and can be nil
func NewLocalAdapter(registry *Registry, database *db.DB) *LocalAdapter {
	return &LocalAdapter{
		registry: registry,
		db:       database,
	}
}

// RegisterAll registers all local tools with the registry
func (la *LocalAdapter) RegisterAll() error {
	// Register database tools
	if err := la.registerDatabaseTools(); err != nil {
		return fmt.Errorf("failed to register database tools: %w", err)
	}

	// Register GraphChain tools
	if err := la.registerGraphChainTools(); err != nil {
		return fmt.Errorf("failed to register GraphChain tools: %w", err)
	}

	// Register transform tools
	if err := la.registerTransformTools(); err != nil {
		return fmt.Errorf("failed to register transform tools: %w", err)
	}

	return nil
}

// registerDatabaseTools registers RocksDB operation tools
func (la *LocalAdapter) registerDatabaseTools() error {
	// Register 'get' tool
	getTool := protocol.Tool{
		Name:        "db-get",
		Description: "Get a value from RocksDB by key",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "The key to retrieve",
				},
				"column_family": map[string]interface{}{
					"type":        "string",
					"description": "Column family name (optional)",
				},
			},
			"required": []string{"key"},
		},
	}

	if err := la.registry.RegisterLocal(getTool, la.handleDBGet); err != nil {
		return err
	}

	// Register 'put' tool
	putTool := protocol.Tool{
		Name:        "db-put",
		Description: "Put a key-value pair into RocksDB",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "The key to store",
				},
				"value": map[string]interface{}{
					"type":        "string",
					"description": "The value to store",
				},
				"column_family": map[string]interface{}{
					"type":        "string",
					"description": "Column family name (optional)",
				},
			},
			"required": []string{"key", "value"},
		},
	}

	if err := la.registry.RegisterLocal(putTool, la.handleDBPut); err != nil {
		return err
	}

	// Register 'delete' tool
	deleteTool := protocol.Tool{
		Name:        "db-delete",
		Description: "Delete a key from RocksDB",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "The key to delete",
				},
				"column_family": map[string]interface{}{
					"type":        "string",
					"description": "Column family name (optional)",
				},
			},
			"required": []string{"key"},
		},
	}

	if err := la.registry.RegisterLocal(deleteTool, la.handleDBDelete); err != nil {
		return err
	}

	// Register 'list' tool
	listTool := protocol.Tool{
		Name:        "db-list",
		Description: "List keys in RocksDB with optional prefix",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"prefix": map[string]interface{}{
					"type":        "string",
					"description": "Key prefix to filter (optional)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of keys to return (default: 100)",
				},
				"column_family": map[string]interface{}{
					"type":        "string",
					"description": "Column family name (optional)",
				},
			},
		},
	}

	if err := la.registry.RegisterLocal(listTool, la.handleDBList); err != nil {
		return err
	}

	return nil
}

// registerGraphChainTools registers GraphChain tools
func (la *LocalAdapter) registerGraphChainTools() error {
	// Register 'graphchain-query' tool
	queryTool := protocol.Tool{
		Name:        "graphchain-query",
		Description: "Query GraphChain knowledge graph",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Natural language query",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 10)",
				},
			},
			"required": []string{"query"},
		},
	}

	if err := la.registry.RegisterLocal(queryTool, la.handleGraphChainQuery); err != nil {
		return err
	}

	return nil
}

// registerTransformTools registers data transformation tools
func (la *LocalAdapter) registerTransformTools() error {
	// Register 'transform-keys' tool
	transformTool := protocol.Tool{
		Name:        "transform-keys",
		Description: "Transform keys in RocksDB using patterns",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Source key pattern",
				},
				"replacement": map[string]interface{}{
					"type":        "string",
					"description": "Target key pattern",
				},
				"dry_run": map[string]interface{}{
					"type":        "boolean",
					"description": "Preview changes without applying (default: true)",
				},
			},
			"required": []string{"pattern", "replacement"},
		},
	}

	if err := la.registry.RegisterLocal(transformTool, la.handleTransform); err != nil {
		return err
	}

	return nil
}

// Tool handlers

func (la *LocalAdapter) handleDBGet(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
	key, ok := args["key"].(string)
	if !ok {
		return errorResult("key must be a string"), nil
	}

	// Check if database is available
	if la.db == nil {
		return errorResult("database not initialized"), nil
	}

	// Get column family (default to "default")
	cf := "default"
	if cfArg, ok := args["column_family"].(string); ok && cfArg != "" {
		cf = cfArg
	}

	// Get value from database
	value, err := la.db.GetCF(cf, key)
	if err != nil {
		if err == db.ErrKeyNotFound {
			return successResult(fmt.Sprintf("Key '%s' not found in column family '%s'", key, cf)), nil
		}
		return errorResult(fmt.Sprintf("failed to get key: %v", err)), nil
	}

	return successResult(fmt.Sprintf("Value: %s", value)), nil
}

func (la *LocalAdapter) handleDBPut(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
	key, ok := args["key"].(string)
	if !ok {
		return errorResult("key must be a string"), nil
	}

	value, ok := args["value"].(string)
	if !ok {
		return errorResult("value must be a string"), nil
	}

	if la.db == nil {
		return errorResult("database not initialized"), nil
	}

	// Get column family (default to "default")
	cf := "default"
	if cfArg, ok := args["column_family"].(string); ok && cfArg != "" {
		cf = cfArg
	}

	// Put value into database
	if err := la.db.PutCF(cf, key, value); err != nil {
		return errorResult(fmt.Sprintf("failed to put key: %v", err)), nil
	}

	return successResult(fmt.Sprintf("Successfully stored key '%s' in column family '%s'", key, cf)), nil
}

func (la *LocalAdapter) handleDBDelete(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
	key, ok := args["key"].(string)
	if !ok {
		return errorResult("key must be a string"), nil
	}

	if la.db == nil {
		return errorResult("database not initialized"), nil
	}

	// Get column family (default to "default")
	cf := "default"
	if cfArg, ok := args["column_family"].(string); ok && cfArg != "" {
		cf = cfArg
	}

	// Note: Delete operation would be implemented here
	// For now, return a placeholder message
	return successResult(fmt.Sprintf("Delete operation for key '%s' in column family '%s'\n(Integration pending)", key, cf)), nil
}

func (la *LocalAdapter) handleDBList(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
	if la.db == nil {
		return errorResult("database not initialized"), nil
	}

	// List all column families available
	cfs, err := la.db.ListCFs()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list column families: %v", err)), nil
	}

	result := fmt.Sprintf("Column families in database:\n")
	for i, cfName := range cfs {
		result += fmt.Sprintf("%d. %s\n", i+1, cfName)
	}

	return successResult(result), nil
}

func (la *LocalAdapter) handleGraphChainQuery(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
	query, ok := args["query"].(string)
	if !ok {
		return errorResult("query must be a string"), nil
	}

	// Placeholder for GraphChain integration
	// This would integrate with the actual GraphChain implementation
	return successResult(fmt.Sprintf("GraphChain query: %s\n(Integration pending)", query)), nil
}

func (la *LocalAdapter) handleTransform(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return errorResult("pattern must be a string"), nil
	}

	replacement, ok := args["replacement"].(string)
	if !ok {
		return errorResult("replacement must be a string"), nil
	}

	dryRun := true
	if dr, ok := args["dry_run"].(bool); ok {
		dryRun = dr
	}

	// Placeholder for transform integration
	mode := "DRY RUN"
	if !dryRun {
		mode = "APPLIED"
	}

	return successResult(fmt.Sprintf("Transform %s: %s → %s\n(Integration pending)", mode, pattern, replacement)), nil
}

// Helper functions

func successResult(text string) *protocol.ToolCallResult {
	return &protocol.ToolCallResult{
		Content: []protocol.Content{
			{
				Type: "text",
				Text: text,
			},
		},
		IsError: false,
	}
}

func errorResult(text string) *protocol.ToolCallResult {
	return &protocol.ToolCallResult{
		Content: []protocol.Content{
			{
				Type: "text",
				Text: text,
			},
		},
		IsError: true,
	}
}
