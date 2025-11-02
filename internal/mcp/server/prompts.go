package server

import (
	"context"
	"fmt"
	"strings"

	"rocksdb-cli/internal/db"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// PromptManager manages MCP prompts for RocksDB operations
type PromptManager struct {
	db     db.KeyValueDB
	config *Config
}

// NewPromptManager creates a new prompt manager
func NewPromptManager(database db.KeyValueDB, config *Config) *PromptManager {
	return &PromptManager{
		db:     database,
		config: config,
	}
}

// RegisterPrompts registers all available prompts with the MCP server
func (pm *PromptManager) RegisterPrompts(s *server.MCPServer) error {
	// Data Analysis Prompt
	dataAnalysisPrompt := mcp.NewPrompt("rocksdb_data_analysis",
		mcp.WithPromptDescription("Generate analysis prompts for RocksDB data exploration"),
		mcp.WithArgument("column_family", mcp.ArgumentDescription("Column family to analyze")),
		mcp.WithArgument("analysis_type", mcp.ArgumentDescription("Type of analysis (overview, patterns, statistics)")),
	)
	s.AddPrompt(dataAnalysisPrompt, pm.handleDataAnalysisPrompt)

	// Query Generation Prompt
	queryGenerationPrompt := mcp.NewPrompt("rocksdb_query_generator",
		mcp.WithPromptDescription("Generate prompts for creating RocksDB queries"),
		mcp.WithArgument("operation", mcp.ArgumentDescription("Operation type (get, scan, prefix)")),
		mcp.WithArgument("column_family", mcp.ArgumentDescription("Target column family")),
		mcp.WithArgument("use_case", mcp.ArgumentDescription("Specific use case or goal")),
	)
	s.AddPrompt(queryGenerationPrompt, pm.handleQueryGenerationPrompt)

	// Troubleshooting Prompt
	troubleshootingPrompt := mcp.NewPrompt("rocksdb_troubleshooting",
		mcp.WithPromptDescription("Generate troubleshooting prompts for RocksDB issues"),
		mcp.WithArgument("issue_type", mcp.ArgumentDescription("Type of issue (performance, data, errors)")),
		mcp.WithArgument("symptoms", mcp.ArgumentDescription("Observed symptoms or behaviors")),
	)
	s.AddPrompt(troubleshootingPrompt, pm.handleTroubleshootingPrompt)

	// Schema Design Prompt
	schemaDesignPrompt := mcp.NewPrompt("rocksdb_schema_design",
		mcp.WithPromptDescription("Generate prompts for RocksDB schema and column family design"),
		mcp.WithArgument("data_type", mcp.ArgumentDescription("Type of data being stored")),
		mcp.WithArgument("access_patterns", mcp.ArgumentDescription("Expected access patterns")),
		mcp.WithArgument("requirements", mcp.ArgumentDescription("Performance and consistency requirements")),
	)
	s.AddPrompt(schemaDesignPrompt, pm.handleSchemaDesignPrompt)

	return nil
}

// Prompt handlers

func (pm *PromptManager) handleDataAnalysisPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	cf := pm.getStringArg(req.Params.Arguments, "column_family", "default")
	analysisType := pm.getStringArg(req.Params.Arguments, "analysis_type", "overview")

	// Get basic info about the column family
	columnFamilies, _ := pm.db.ListCFs()
	cfExists := false
	for _, existingCF := range columnFamilies {
		if existingCF == cf {
			cfExists = true
			break
		}
	}

	var prompt strings.Builder

	switch analysisType {
	case "overview":
		prompt.WriteString(fmt.Sprintf("Please analyze the RocksDB column family '%s'.\n\n", cf))

		if cfExists {
			prompt.WriteString("Available information:\n")
			prompt.WriteString(fmt.Sprintf("- Database path: %s\n", pm.config.DatabasePath))
			prompt.WriteString(fmt.Sprintf("- Read-only mode: %v\n", pm.db.IsReadOnly()))
			prompt.WriteString(fmt.Sprintf("- Column families: %v\n", columnFamilies))

			// Try to get sample data
			if sampleData, err := pm.db.ScanCF(cf, nil, nil, db.ScanOptions{Limit: 10, Values: true}); err == nil {
				prompt.WriteString(fmt.Sprintf("- Sample data (first 10 entries): %v\n", sampleData))
			}
		} else {
			prompt.WriteString(fmt.Sprintf("Note: Column family '%s' does not exist. Available column families: %v\n", cf, columnFamilies))
		}

		prompt.WriteString("\nPlease provide:\n")
		prompt.WriteString("1. Data structure analysis\n")
		prompt.WriteString("2. Key patterns and naming conventions\n")
		prompt.WriteString("3. Value formats and types\n")
		prompt.WriteString("4. Potential optimization opportunities\n")
		prompt.WriteString("5. Recommendations for usage\n")

	case "patterns":
		prompt.WriteString(fmt.Sprintf("Analyze data patterns in RocksDB column family '%s'.\n\n", cf))
		prompt.WriteString("Please examine:\n")
		prompt.WriteString("1. Key naming patterns and hierarchies\n")
		prompt.WriteString("2. Value size distributions\n")
		prompt.WriteString("3. Common prefixes and suffixes\n")
		prompt.WriteString("4. Data access patterns\n")
		prompt.WriteString("5. Potential hot spots or imbalances\n")

	case "statistics":
		prompt.WriteString(fmt.Sprintf("Generate statistical analysis for RocksDB column family '%s'.\n\n", cf))
		prompt.WriteString("Please provide:\n")
		prompt.WriteString("1. Record count estimates\n")
		prompt.WriteString("2. Key and value size statistics\n")
		prompt.WriteString("3. Distribution analysis\n")
		prompt.WriteString("4. Growth trends if observable\n")
		prompt.WriteString("5. Storage efficiency metrics\n")
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("RocksDB %s analysis for column family '%s'", analysisType, cf),
		Messages: []mcp.PromptMessage{
			{
				Role:    "user",
				Content: mcp.NewTextContent(prompt.String()),
			},
		},
	}, nil
}

func (pm *PromptManager) handleQueryGenerationPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	operation := pm.getStringArg(req.Params.Arguments, "operation", "get")
	cf := pm.getStringArg(req.Params.Arguments, "column_family", "default")
	useCase := pm.getStringArg(req.Params.Arguments, "use_case", "general data access")

	var prompt strings.Builder
	prompt.WriteString(fmt.Sprintf("Generate RocksDB %s operations for the use case: %s\n\n", operation, useCase))
	prompt.WriteString(fmt.Sprintf("Target column family: %s\n", cf))

	// Get database context
	columnFamilies, _ := pm.db.ListCFs()
	prompt.WriteString(fmt.Sprintf("Available column families: %v\n", columnFamilies))
	prompt.WriteString(fmt.Sprintf("Database mode: %s\n\n", func() string {
		if pm.db.IsReadOnly() {
			return "read-only"
		}
		return "read-write"
	}()))

	switch operation {
	case "get":
		prompt.WriteString("Please provide:\n")
		prompt.WriteString("1. Specific key patterns to retrieve\n")
		prompt.WriteString("2. Error handling strategies\n")
		prompt.WriteString("3. Performance considerations\n")
		prompt.WriteString("4. Example command syntax\n")

	case "scan":
		prompt.WriteString("Please provide:\n")
		prompt.WriteString("1. Optimal range scan parameters\n")
		prompt.WriteString("2. Limit and pagination strategies\n")
		prompt.WriteString("3. Forward vs reverse scan recommendations\n")
		prompt.WriteString("4. Performance optimization tips\n")

	case "prefix":
		prompt.WriteString("Please provide:\n")
		prompt.WriteString("1. Effective prefix patterns\n")
		prompt.WriteString("2. Prefix length optimization\n")
		prompt.WriteString("3. Result limiting strategies\n")
		prompt.WriteString("4. Use case specific examples\n")

	default:
		prompt.WriteString("Please provide comprehensive query strategies for this operation.\n")
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("RocksDB %s query generation for %s", operation, useCase),
		Messages: []mcp.PromptMessage{
			{
				Role:    "user",
				Content: mcp.NewTextContent(prompt.String()),
			},
		},
	}, nil
}

func (pm *PromptManager) handleTroubleshootingPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	issueType := pm.getStringArg(req.Params.Arguments, "issue_type", "general")
	symptoms := pm.getStringArg(req.Params.Arguments, "symptoms", "unspecified")

	var prompt strings.Builder
	prompt.WriteString(fmt.Sprintf("Troubleshoot RocksDB %s issues with symptoms: %s\n\n", issueType, symptoms))

	// Add database context
	columnFamilies, _ := pm.db.ListCFs()
	prompt.WriteString("Database context:\n")
	prompt.WriteString(fmt.Sprintf("- Database path: %s\n", pm.config.DatabasePath))
	prompt.WriteString(fmt.Sprintf("- Read-only mode: %v\n", pm.db.IsReadOnly()))
	prompt.WriteString(fmt.Sprintf("- Column families: %v\n\n", columnFamilies))

	switch issueType {
	case "performance":
		prompt.WriteString("Please analyze and provide solutions for:\n")
		prompt.WriteString("1. Query performance bottlenecks\n")
		prompt.WriteString("2. Memory usage optimization\n")
		prompt.WriteString("3. I/O efficiency improvements\n")
		prompt.WriteString("4. Column family design optimization\n")
		prompt.WriteString("5. Configuration tuning recommendations\n")

	case "data":
		prompt.WriteString("Please investigate and resolve:\n")
		prompt.WriteString("1. Data inconsistency issues\n")
		prompt.WriteString("2. Missing or corrupted records\n")
		prompt.WriteString("3. Unexpected data formats\n")
		prompt.WriteString("4. Column family access problems\n")
		prompt.WriteString("5. Data integrity verification\n")

	case "errors":
		prompt.WriteString("Please diagnose and fix:\n")
		prompt.WriteString("1. Connection and access errors\n")
		prompt.WriteString("2. Operation timeout issues\n")
		prompt.WriteString("3. Resource limit problems\n")
		prompt.WriteString("4. Configuration errors\n")
		prompt.WriteString("5. Environment setup issues\n")

	default:
		prompt.WriteString("Please provide comprehensive troubleshooting guidance:\n")
		prompt.WriteString("1. Systematic problem diagnosis\n")
		prompt.WriteString("2. Common issue identification\n")
		prompt.WriteString("3. Step-by-step resolution\n")
		prompt.WriteString("4. Prevention strategies\n")
		prompt.WriteString("5. Monitoring recommendations\n")
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("RocksDB %s troubleshooting for symptoms: %s", issueType, symptoms),
		Messages: []mcp.PromptMessage{
			{
				Role:    "user",
				Content: mcp.NewTextContent(prompt.String()),
			},
		},
	}, nil
}

func (pm *PromptManager) handleSchemaDesignPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	dataType := pm.getStringArg(req.Params.Arguments, "data_type", "general")
	accessPatterns := pm.getStringArg(req.Params.Arguments, "access_patterns", "mixed")
	requirements := pm.getStringArg(req.Params.Arguments, "requirements", "standard")

	var prompt strings.Builder
	prompt.WriteString("Design an optimal RocksDB schema for the following requirements:\n\n")
	prompt.WriteString(fmt.Sprintf("Data type: %s\n", dataType))
	prompt.WriteString(fmt.Sprintf("Access patterns: %s\n", accessPatterns))
	prompt.WriteString(fmt.Sprintf("Requirements: %s\n\n", requirements))

	// Add current database context
	columnFamilies, _ := pm.db.ListCFs()
	prompt.WriteString("Current database state:\n")
	prompt.WriteString(fmt.Sprintf("- Existing column families: %v\n", columnFamilies))
	prompt.WriteString(fmt.Sprintf("- Read-only mode: %v\n\n", pm.db.IsReadOnly()))

	prompt.WriteString("Please provide:\n")
	prompt.WriteString("1. Column family design recommendations\n")
	prompt.WriteString("2. Key naming conventions and patterns\n")
	prompt.WriteString("3. Value serialization strategies\n")
	prompt.WriteString("4. Indexing and secondary access methods\n")
	prompt.WriteString("5. Performance optimization considerations\n")
	prompt.WriteString("6. Scalability and maintenance guidelines\n")
	prompt.WriteString("7. Migration strategies from current schema\n")

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("RocksDB schema design for %s data with %s access patterns", dataType, accessPatterns),
		Messages: []mcp.PromptMessage{
			{
				Role:    "user",
				Content: mcp.NewTextContent(prompt.String()),
			},
		},
	}, nil
}

// Helper method to safely get string arguments
func (pm *PromptManager) getStringArg(args map[string]string, key, defaultValue string) string {
	if val, exists := args[key]; exists {
		return val
	}
	return defaultValue
}
