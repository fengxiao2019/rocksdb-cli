package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"rocksdb-cli/internal/db"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ResourceManager manages MCP resources for RocksDB operations
type ResourceManager struct {
	db     db.KeyValueDB
	config *Config
}

// NewResourceManager creates a new resource manager
func NewResourceManager(database db.KeyValueDB, config *Config) *ResourceManager {
	return &ResourceManager{
		db:     database,
		config: config,
	}
}

// RegisterResources registers all available resources with the MCP server
func (rm *ResourceManager) RegisterResources(s *server.MCPServer) error {
	if !rm.config.EnableResources {
		return nil
	}

	// Column Families resource - lists all column families
	columnFamiliesResource := mcp.NewResource(
		"rocksdb://column-families",
		"Column Families",
		mcp.WithResourceDescription("List of all column families in the database"),
		mcp.WithMIMEType("application/json"),
	)
	s.AddResource(columnFamiliesResource, rm.handleColumnFamiliesResource)

	// Column Family Data resource - access data from a specific column family
	columnFamilyDataResource := mcp.NewResource(
		"rocksdb://column-family/{name}",
		"Column Family Data",
		mcp.WithResourceDescription("Data from a specific column family"),
		mcp.WithMIMEType("application/json"),
	)
	s.AddResource(columnFamilyDataResource, rm.handleColumnFamilyDataResource)

	// Database Stats resource
	databaseStatsResource := mcp.NewResource(
		"rocksdb://stats",
		"Database Statistics",
		mcp.WithResourceDescription("Database statistics and metadata"),
		mcp.WithMIMEType("application/json"),
	)
	s.AddResource(databaseStatsResource, rm.handleDatabaseStatsResource)

	// Key-Value Pair resource - access a specific key
	keyValueResource := mcp.NewResource(
		"rocksdb://data/{column_family}/{key}",
		"Key-Value Pair",
		mcp.WithResourceDescription("Access a specific key-value pair"),
		mcp.WithMIMEType("text/plain"),
	)
	s.AddResource(keyValueResource, rm.handleKeyValueResource)

	// Prefix Scan resource - scan keys with a prefix
	prefixScanResource := mcp.NewResource(
		"rocksdb://prefix/{column_family}/{prefix}",
		"Prefix Scan",
		mcp.WithResourceDescription("Scan keys with a specific prefix"),
		mcp.WithMIMEType("application/json"),
	)
	s.AddResource(prefixScanResource, rm.handlePrefixScanResource)

	return nil
}

// Resource handlers

func (rm *ResourceManager) handleColumnFamiliesResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	columnFamilies, err := rm.db.ListCFs()
	if err != nil {
		return nil, fmt.Errorf("failed to list column families: %w", err)
	}

	// Create JSON response
	response := map[string]interface{}{
		"column_families": columnFamilies,
		"count":           len(columnFamilies),
		"read_only":       rm.db.IsReadOnly(),
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal column families: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (rm *ResourceManager) handleColumnFamilyDataResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract column family name from URI
	cfName := rm.extractPathParam(req.Params.URI, "name")
	if cfName == "" {
		return nil, fmt.Errorf("column family name is required")
	}

	// Get sample data from the column family (first 100 keys)
	sampleData, err := rm.db.ScanCF(cfName, nil, nil, db.ScanOptions{
		Limit:   100,
		Reverse: false,
		Values:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan column family '%s': %w", cfName, err)
	}

	// Create JSON response
	response := map[string]interface{}{
		"column_family": cfName,
		"sample_data":   sampleData,
		"sample_size":   len(sampleData),
		"read_only":     rm.db.IsReadOnly(),
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal column family data: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (rm *ResourceManager) handleDatabaseStatsResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	columnFamilies, err := rm.db.ListCFs()
	if err != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", err)
	}

	// Create basic stats response
	response := map[string]interface{}{
		"database_path":   rm.config.DatabasePath,
		"read_only":       rm.db.IsReadOnly(),
		"column_families": columnFamilies,
		"cf_count":        len(columnFamilies),
		"server_name":     rm.config.Name,
		"server_version":  rm.config.Version,
		"transport_type":  rm.config.Transport.Type,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal database stats: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (rm *ResourceManager) handleKeyValueResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract parameters from URI
	cfName := rm.extractPathParam(req.Params.URI, "column_family")
	key := rm.extractPathParam(req.Params.URI, "key")

	if cfName == "" || key == "" {
		return nil, fmt.Errorf("both column_family and key are required")
	}

	// Decode URL-encoded key
	decodedKey, err := url.QueryUnescape(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}

	// Get the value
	value, err := rm.db.GetCF(cfName, decodedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get key '%s' from CF '%s': %w", decodedKey, cfName, err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "text/plain",
			Text:     value,
		},
	}, nil
}

func (rm *ResourceManager) handlePrefixScanResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract parameters from URI
	cfName := rm.extractPathParam(req.Params.URI, "column_family")
	prefix := rm.extractPathParam(req.Params.URI, "prefix")

	if cfName == "" || prefix == "" {
		return nil, fmt.Errorf("both column_family and prefix are required")
	}

	// Decode URL-encoded prefix
	decodedPrefix, err := url.QueryUnescape(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to decode prefix: %w", err)
	}

	// Parse query parameters for limit
	u, err := url.Parse(req.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}

	limit := 100 // Default limit
	if limitStr := u.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Perform prefix scan
	results, err := rm.db.PrefixScanCF(cfName, decodedPrefix, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to prefix scan CF '%s': %w", cfName, err)
	}

	// Create JSON response
	response := map[string]interface{}{
		"column_family": cfName,
		"prefix":        decodedPrefix,
		"results":       results,
		"count":         len(results),
		"limit":         limit,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prefix scan results: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// Helper method to extract path parameters from URI
func (rm *ResourceManager) extractPathParam(uri, paramName string) string {
	// Simple path parameter extraction for rocksdb://type/param1/param2 format
	parts := strings.Split(uri, "/")

	switch paramName {
	case "name":
		// For rocksdb://column-family/{name}
		if len(parts) >= 3 && parts[0] == "rocksdb:" && parts[2] == "column-family" {
			if len(parts) >= 4 {
				return parts[3]
			}
		}
	case "column_family":
		// For rocksdb://data/{column_family}/{key} or rocksdb://prefix/{column_family}/{prefix}
		if len(parts) >= 4 && parts[0] == "rocksdb:" {
			return parts[3]
		}
	case "key":
		// For rocksdb://data/{column_family}/{key}
		if len(parts) >= 5 && parts[0] == "rocksdb:" && parts[2] == "data" {
			return parts[4]
		}
	case "prefix":
		// For rocksdb://prefix/{column_family}/{prefix}
		if len(parts) >= 5 && parts[0] == "rocksdb:" && parts[2] == "prefix" {
			return parts[4]
		}
	}

	return ""
}
