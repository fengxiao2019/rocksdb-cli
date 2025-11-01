package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"rocksdb-cli/internal/api"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/graphchain"
	"rocksdb-cli/internal/jsonutil"
	"rocksdb-cli/internal/repl"
	"rocksdb-cli/internal/service"
	"rocksdb-cli/internal/transform"
	"rocksdb-cli/internal/util"
	"rocksdb-cli/internal/webui"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	dbPath     string
	readOnly   bool
	configPath string
	pretty     bool
)

// Root command
var rootCmd = &cobra.Command{
	Use:   "rocksdb-cli",
	Short: "Powerful CLI for RocksDB databases",
	Long: `RocksDB CLI - Command-line interface for RocksDB databases

FEATURES:
  * Web UI                 - Modern web interface (single binary)
  * Interactive REPL       - Real-time database exploration
  * Transform Data         - Batch data transformation with Python
  * AI Assistant           - Natural language queries (GraphChain)
  * Data Export            - Export to CSV and other formats
  * Advanced Search        - Fuzzy search, JSON queries, prefix scan
  * Real-time Monitor      - Watch for changes as they happen

QUICK START:
  # Web UI (easiest way to get started)
  rocksdb-cli web --db /path/to/database
  # Then open http://localhost:8080 in your browser

  # Interactive mode (recommended for exploration)
  rocksdb-cli repl --db /path/to/database

  # Direct commands (good for scripting)
  rocksdb-cli get --db mydb --cf users user:1001
  rocksdb-cli scan --db mydb --cf logs --limit=100

  # AI-powered queries
  rocksdb-cli ai --db mydb "show me all active users"

  # Data transformation
  rocksdb-cli transform --db mydb --cf users --expr="value.upper()" --dry-run

REQUIREMENTS:
  • Python 3 (required for transform command)
  • RocksDB database file path

TIP: Use --read-only flag to safely explore production databases`,
}

// REPL command - maintains existing interactive experience
var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Start interactive REPL mode",
	Long:  `Start an interactive Read-Eval-Print Loop for database operations`,
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		// Use existing REPL functionality
		repl.Start(rdb)
	},
}

// Get command
var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get value by key from column family",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf := getColumnFamily(cmd)
		key := args[0]

		// Use DatabaseService instead of direct DB access
		dbService := service.NewDatabaseService(rdb)
		value, err := dbService.GetValue(cf, key)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Key: %s\n", util.FormatKey(key))
		fmt.Printf("Value: %s\n", formatValue(value, pretty))
	},
}

// Put command
var putCmd = &cobra.Command{
	Use:   "put <key> <value>",
	Short: "Put key-value pair in column family",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf := getColumnFamily(cmd)
		key, value := args[0], args[1]

		// Use DatabaseService instead of direct DB access
		dbService := service.NewDatabaseService(rdb)
		err := dbService.PutValue(cf, key, value)
		if err != nil {
			if err == db.ErrReadOnlyMode {
				fmt.Println("Error: Database is in read-only mode")
			} else {
				fmt.Printf("Error: %v\n", err)
			}
			os.Exit(1)
		}

		fmt.Printf("Successfully put: %s = %s\n", util.FormatKey(key), value)
	},
}

// Last command
var lastCmd = &cobra.Command{
	Use:   "last",
	Short: "Get the last key-value pair from column family",
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf := getColumnFamily(cmd)

		// Use DatabaseService instead of direct DB access
		dbService := service.NewDatabaseService(rdb)
		key, value, err := dbService.GetLastEntry(cf)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Last entry in '%s': %s = %s\n", cf, util.FormatKey(key), formatValue(value, pretty))
	},
}

// Scan command
var scanCmd = &cobra.Command{
	Use:   "scan [start] [end]",
	Short: "Scan key-value pairs in range",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf := getColumnFamily(cmd)

		var start, end *string
		if len(args) > 0 && args[0] != "*" {
			start = &args[0]
		}
		if len(args) > 1 && args[1] != "*" {
			end = &args[1]
		}

		limit, _ := cmd.Flags().GetInt("limit")
		reverse, _ := cmd.Flags().GetBool("reverse")
		keysOnly, _ := cmd.Flags().GetBool("keys-only")

		err := executeScan(rdb, cf, start, end, limit, reverse, keysOnly)
		if err != nil {
			fmt.Printf("Scan failed: %v\n", err)
			os.Exit(1)
		}
	},
}

// Prefix command
var prefixCmd = &cobra.Command{
	Use:   "prefix <prefix>",
	Short: "Search by key prefix",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf := getColumnFamily(cmd)
		prefix := args[0]

		err := executePrefix(rdb, cf, prefix, pretty)
		if err != nil {
			fmt.Printf("Prefix scan failed: %v\n", err)
			os.Exit(1)
		}
	},
}

// Search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Fuzzy search for keys and values",
	Long:  `Fuzzy search for keys and/or values with various options including .NET tick conversion`,
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf := getColumnFamily(cmd)

		keyPattern, _ := cmd.Flags().GetString("key")
		valuePattern, _ := cmd.Flags().GetString("value")
		useRegex, _ := cmd.Flags().GetBool("regex")
		caseSensitive, _ := cmd.Flags().GetBool("case-sensitive")
		keysOnly, _ := cmd.Flags().GetBool("keys-only")
		tick, _ := cmd.Flags().GetBool("tick")
		limit, _ := cmd.Flags().GetInt("limit")
		after, _ := cmd.Flags().GetString("after")
		exportFile, _ := cmd.Flags().GetString("export")
		exportSep, _ := cmd.Flags().GetString("export-sep")

		if keyPattern == "" && valuePattern == "" {
			fmt.Println("Error: Must specify at least --key or --value pattern")
			os.Exit(1)
		}

		err := executeSearch(rdb, cf, keyPattern, valuePattern, useRegex, caseSensitive, keysOnly, tick, limit, pretty, after, exportFile, exportSep)
		if err != nil {
			fmt.Printf("Search failed: %v\n", err)
			os.Exit(1)
		}
	},
}

// Export command
var exportCmd = &cobra.Command{
	Use:   "export <file>",
	Short: "Export column family to CSV file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf := getColumnFamily(cmd)
		filePath := args[0]

		sep, _ := cmd.Flags().GetString("sep")
		sep = parseSep(sep)

		err := rdb.ExportToCSV(cf, filePath, sep)
		if err != nil {
			fmt.Printf("Export failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Exported column family '%s' to %s (sep=%q)\n", cf, filePath, sep)
	},
}

// Watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for new entries in column family (real-time)",
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf := getColumnFamily(cmd)
		interval, _ := cmd.Flags().GetDuration("interval")

		fmt.Printf("Watching column family '%s' for new entries (interval: %v)...\n", cf, interval)
		fmt.Println("Press Ctrl+C to stop")

		// Set up signal handling for graceful shutdown
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(c)

		var lastKey, lastValue string

		// Get initial last entry
		key, value, err := rdb.GetLastCF(cf)
		if err != nil {
			if err.Error() != "column family is empty" {
				fmt.Printf("Watch failed: %v\n", err)
				os.Exit(1)
			}
			lastKey = ""
			lastValue = ""
		} else {
			lastKey = key
			lastValue = value
			fmt.Printf("[%s] Initial: %s = %s\n", time.Now().Format("15:04:05"), util.FormatKey(key), value)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-c:
				fmt.Println("\nStopping watch...")
				return
			case <-ticker.C:
				key, value, err := rdb.GetLastCF(cf)
				if err != nil {
					if err.Error() == "column family is empty" {
						continue
					}
					fmt.Printf("Watch error: %v\n", err)
					continue
				}

				if key != lastKey || value != lastValue {
					fmt.Printf("[%s] New: %s = %s\n", time.Now().Format("15:04:05"), util.FormatKey(key), value)
					lastKey = key
					lastValue = value
				}
			}
		}
	},
}

// Stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show database or column family statistics",
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf, _ := cmd.Flags().GetString("cf")

		// Use StatsService instead of direct DB access
		statsService := service.NewStatsService(rdb)

		if cf == "" {
			// Database-wide stats
			stats, err := statsService.GetDatabaseStats()
			if err != nil {
				fmt.Printf("Failed to get database stats: %v\n", err)
				os.Exit(1)
			}

			// Format sample keys and common prefixes
			for i := range stats.ColumnFamilies {
				for j, k := range stats.ColumnFamilies[i].SampleKeys {
					stats.ColumnFamilies[i].SampleKeys[j] = util.FormatKey(k)
				}
				newPrefixes := make(map[string]int64)
				for k, v := range stats.ColumnFamilies[i].CommonPrefixes {
					newPrefixes[util.FormatKey(k)] = v
				}
				stats.ColumnFamilies[i].CommonPrefixes = newPrefixes
			}

			if pretty {
				data, _ := json.MarshalIndent(stats, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Printf("Database Stats: %+v\n", stats)
			}
		} else {
			// Column family stats
			stats, err := statsService.GetColumnFamilyStats(cf)
			if err != nil {
				fmt.Printf("Failed to get stats for column family '%s': %v\n", cf, err)
				os.Exit(1)
			}

			// Format sample keys and common prefixes
			for j, k := range stats.SampleKeys {
				stats.SampleKeys[j] = util.FormatKey(k)
			}
			newPrefixes := make(map[string]int64)
			for k, v := range stats.CommonPrefixes {
				newPrefixes[util.FormatKey(k)] = v
			}
			stats.CommonPrefixes = newPrefixes

			if pretty {
				data, _ := json.MarshalIndent(stats, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Printf("Stats for column family '%s': %+v\n", cf, stats)
			}
		}
	},
}

// Keyformat command
var keyformatCmd = &cobra.Command{
	Use:   "keyformat",
	Short: "Show detected key format and conversion examples for column family",
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf := getColumnFamily(cmd)

		format, examples := rdb.GetKeyFormatInfo(cf)
		fmt.Printf("Column family '%s' key format: %v\n", cf, format)
		fmt.Printf("Examples: %s\n", examples)
	},
}

// JSON query command
var jsonqueryCmd = &cobra.Command{
	Use:   "jsonquery",
	Short: "Query by JSON field value",
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		cf := getColumnFamily(cmd)
		field, _ := cmd.Flags().GetString("field")
		value, _ := cmd.Flags().GetString("value")

		if field == "" || value == "" {
			fmt.Println("Error: --field and --value are required")
			os.Exit(1)
		}

		// Use SearchService instead of direct DB access
		searchService := service.NewSearchService(rdb)
		result, err := searchService.JSONQuery(cf, field, value)
		if err != nil {
			fmt.Printf("JSON query failed: %v\n", err)
			os.Exit(1)
		}

		if result.Count == 0 {
			fmt.Printf("No entries found in '%s' where field '%s' = '%s'\n", cf, field, value)
		} else {
			fmt.Printf("Found %d entries in '%s' where field '%s' = '%s':\n", result.Count, cf, field, value)
			for k, v := range result.Data {
				fmt.Printf("%s: %s\n", util.FormatKey(k), formatValue(v, pretty))
			}
		}
	},
}

// List column families command
var listcfCmd = &cobra.Command{
	Use:   "listcf",
	Short: "List all column families",
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		// Use DatabaseService instead of direct DB access
		dbService := service.NewDatabaseService(rdb)
		cfs, err := dbService.ListColumnFamilies()
		if err != nil {
			fmt.Printf("Error listing column families: %v\n", err)
			os.Exit(1)
		}

		sort.Strings(cfs)
		fmt.Printf("Column families (%d):\n", len(cfs))
		for i, cf := range cfs {
			fmt.Printf("  [%d] %s\n", i+1, cf)
		}
	},
}

// Create column family command
var createcfCmd = &cobra.Command{
	Use:   "createcf <name>",
	Short: "Create new column family",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		if rdb.IsReadOnly() {
			fmt.Println("Error: Database is in read-only mode")
			os.Exit(1)
		}

		cfName := args[0]
		err := rdb.CreateCF(cfName)
		if err != nil {
			fmt.Printf("Failed to create column family '%s': %v\n", cfName, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully created column family '%s'\n", cfName)
	},
}

// Drop column family command
var dropcfCmd = &cobra.Command{
	Use:   "dropcf <name>",
	Short: "Drop column family",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		if rdb.IsReadOnly() {
			fmt.Println("Error: Database is in read-only mode")
			os.Exit(1)
		}

		cfName := args[0]
		err := rdb.DropCF(cfName)
		if err != nil {
			fmt.Printf("Failed to drop column family '%s': %v\n", cfName, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully dropped column family '%s'\n", cfName)
	},
}

// AI command - GraphChain agent
var aiCmd = &cobra.Command{
	Use:   "ai [query]",
	Short: "AI-powered database assistant (GraphChain)",
	Long: `Start AI-powered GraphChain assistant for natural language database queries.
If no query is provided, starts interactive mode.`,
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()

		if len(args) == 0 {
			// Interactive mode
			runGraphChainInteractive(rdb)
		} else {
			// Single query mode
			query := strings.Join(args, " ")
			runGraphChainQuery(rdb, query)
		}
	},
}

// Transform command - data transformation with Python
var transformCmd = &cobra.Command{
	Use:   "transform",
	Short: "Transform key-value data using Python expressions",
	Long: `Transform key-value data using Python expressions or scripts.

DESCRIPTION:
  Apply Python transformations to values in a column family. Supports:
  • Python expressions (inline)
  • Python script files (with custom functions)
  • Filtering (skip entries that don't match conditions)
  • Dry-run mode (preview changes safely)
  • Batch processing (handle large datasets efficiently)

QUICK START:
  # Preview transformation (safe, no changes)
  rocksdb-cli transform --db mydb --cf users --expr="value.upper()" --dry-run

  # Actually apply transformation
  rocksdb-cli transform --db mydb --cf users --expr="value.upper()"

EXPRESSION EXAMPLES:
  • Simple text:      --expr="value.upper()"
  • JSON field:       --expr="import json; d=json.loads(value); d['name']=d['name'].upper(); json.dumps(d)"
  • With filter:      --filter="'active' in value" --expr="value.upper()"
  • Key-based filter: --filter="key.startswith('user:')" --expr="value.upper()"

SCRIPT FILE EXAMPLES:
  # Use a Python script file
  rocksdb-cli transform --db mydb --cf users --script=scripts/transform/transform_uppercase_name.py

  # Script file format (scripts/transform/transform_uppercase_name.py):
  import json
  
  def should_process(key, value):
      # Return True to process, False to skip
      data = json.loads(value)
      return 'name' in data
  
  def transform_value(key, value):
      # Transform the value
      data = json.loads(value)
      data['name'] = data['name'].upper()
      return json.dumps(data)

SAFETY TIPS:
  WARNING: Always use --dry-run first to preview changes
  TIP: Start with --limit=10 to test on small dataset
  TIP: Check the statistics output before proceeding
  TIP: Consider backing up your database first

CONTEXT VARIABLES (available in expressions):
  • key    - The entry's key (string)
  • value  - The entry's value (string)`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open database
		rdb := openDatabase()
		defer rdb.Close()
		
		// Get flags
		cf, _ := cmd.Flags().GetString("cf")
		expr, _ := cmd.Flags().GetString("expr")
		keyExpr, _ := cmd.Flags().GetString("key-expr")
		valueExpr, _ := cmd.Flags().GetString("value-expr")
		filterExpr, _ := cmd.Flags().GetString("filter")
		scriptPath, _ := cmd.Flags().GetString("script")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		limit, _ := cmd.Flags().GetInt("limit")
		batchSize, _ := cmd.Flags().GetInt("batch-size")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		// Create transform options
		opts := transform.TransformOptions{
			Expression:       expr,
			KeyExpression:    keyExpr,
			ValueExpression:  valueExpr,
			FilterExpression: filterExpr,
			ScriptPath:       scriptPath,
			DryRun:           dryRun,
			Limit:            limit,
			BatchSize:        batchSize,
			Verbose:          verbose,
		}
		
		// Create transform processor
		processor := transform.NewTransformProcessor(rdb)
		
		// Execute transformation
		fmt.Printf("Transforming column family '%s'...\n", cf)
		if dryRun {
			fmt.Println("(DRY RUN - no changes will be made)")
		}
		fmt.Println()
		
		result, err := processor.Process(cf, opts)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		
		// Display results
		fmt.Printf("Transform completed in %v\n", result.Duration)
		fmt.Printf("Processed: %d entries\n", result.Processed)
		fmt.Printf("Modified:  %d entries\n", result.Modified)
		fmt.Printf("Skipped:   %d entries\n", result.Skipped)
		fmt.Printf("Errors:    %d\n", len(result.Errors))
		
		// Show dry-run preview
		if dryRun && len(result.DryRunData) > 0 {
			fmt.Printf("\nPreview (showing %d entries):\n", len(result.DryRunData))
			fmt.Println(strings.Repeat("=", 80))
			for i, entry := range result.DryRunData {
				if i >= 10 {
					fmt.Printf("\n... and %d more entries\n", len(result.DryRunData)-10)
					break
				}
				fmt.Printf("\n[%d] Key: %s\n", i+1, entry.OriginalKey)
				if entry.WillModify {
					fmt.Printf("    Original:    %s\n", entry.OriginalValue)
					fmt.Printf("    Transformed: %s\n", entry.TransformedValue)
					fmt.Printf("    Status: WILL MODIFY\n")
				} else if entry.Skipped {
					fmt.Printf("    Value: %s\n", entry.OriginalValue)
					fmt.Printf("    Status: SKIPPED (filtered out)\n")
				} else {
					fmt.Printf("    Value: %s\n", entry.OriginalValue)
					fmt.Printf("    Status: NO CHANGE\n")
				}
			}
		}
		
		// Show errors if any
		if len(result.Errors) > 0 {
			fmt.Printf("\nErrors encountered:\n")
			for i, err := range result.Errors {
				if i >= 5 {
					fmt.Printf("... and %d more errors\n", len(result.Errors)-5)
					break
				}
				fmt.Printf("  - Key: %s\n", err.Key)
				fmt.Printf("    Error: %s\n", err.Error)
			}
		}
	},
}

// Web command - starts web server with embedded UI
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start web UI server (all-in-one binary)",
	Long: `Start web server with embedded Web UI for database management.

DESCRIPTION:
  Launch a web server with full API and modern Web UI interface.
  Everything is bundled in a single binary - no external dependencies.

FEATURES:
  - Browse and search data with intuitive UI
  - Advanced search with regex support
  - Export data to CSV/JSON formats
  - Real-time data viewing with pagination
  - Binary data support with hex encoding
  - JSON tree visualization

QUICK START:
  # Start web UI on default port 8080
  rocksdb-cli web --db /path/to/database

  # Use custom port
  rocksdb-cli web --db mydb --port 3000

  # Read-only mode (recommended for production)
  rocksdb-cli web --db mydb --read-only

  Then open http://localhost:8080 in your browser

ENDPOINTS:
  /                - Web UI (React application)
  /api/v1/health   - Health check
  /api/v1/cf       - List column families
  /api/v1/stats    - Database statistics
  And more...`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open database
		rdb := openDatabase()
		defer rdb.Close()

		port, _ := cmd.Flags().GetString("port")
		enableAI, _ := cmd.Flags().GetBool("enable-ai")

		// Get embedded static files
		staticFS, err := webui.GetDistFS()
		if err != nil {
			log.Fatalf("Failed to load embedded Web UI: %v", err)
		}

		// Initialize AI agent if enabled
		var aiAgent interface{}
		if enableAI {
			// Type assert to *db.DB
			dbInstance, ok := rdb.(*db.DB)
			if !ok {
				log.Fatalf("AI requires a real DB instance, got %T", rdb)
			}

			agent := graphchain.NewAgent(dbInstance)
			cfg, err := graphchain.LoadConfig(configPath)
			if err != nil {
				// Use default config if file doesn't exist
				cfg = graphchain.DefaultConfig()
				fmt.Printf("⚠️  Using default GraphChain config (AI may not work without proper LLM setup)\n")
			}

			if err := agent.Initialize(context.Background(), cfg); err != nil {
				log.Fatalf("Failed to initialize AI agent: %v", err)
			}
			aiAgent = agent
			fmt.Printf("AI Assistant enabled (GraphChain)\n")
		}

		// Setup router with embedded UI and optional AI
		router := api.SetupRouterWithUI(rdb, staticFS, aiAgent)

		addr := ":" + port
		fmt.Printf("\nRocksDB Web UI Server starting...\n")
		fmt.Printf("   Database: %s\n", dbPath)
		fmt.Printf("   Read-only: %v\n", readOnly)
		fmt.Printf("   AI Enabled: %v\n", enableAI)
		fmt.Printf("   URL: http://localhost%s\n", addr)
		fmt.Printf("\nOpen http://localhost%s in your browser\n\n", addr)

		// Start server
		if err := router.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	},
}

// Helper functions
func openDatabase() db.KeyValueDB {
	var rdb db.KeyValueDB
	var err error

	if readOnly {
		rdb, err = db.OpenReadOnly(dbPath)
	} else {
		rdb, err = db.Open(dbPath)
	}

	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}

	return rdb
}

func getColumnFamily(cmd *cobra.Command) string {
	cf, _ := cmd.Flags().GetString("cf")
	if cf == "" {
		return "default"
	}
	return cf
}

// formatValue formats a value based on pretty flag using jsonutil
func formatValue(value string, pretty bool) string {
	if !pretty {
		return value
	}
	return jsonutil.PrettyPrintWithNestedExpansion(value)
}

// executeScan executes a scan operation with smart key conversion
func executeScan(rdb db.KeyValueDB, cf string, start, end *string, limit int, reverse, keysOnly bool) error {
	// Convert pointers to strings for smart scan
	var startStr, endStr string
	if start != nil {
		startStr = *start
	}
	if end != nil {
		endStr = *end
	}

	// Use ScanService instead of direct DB access
	scanService := service.NewScanService(rdb)
	opts := service.ScanOptions{
		StartKey: startStr,
		EndKey:   endStr,
		Limit:    limit,
		Reverse:  reverse,
		KeysOnly: keysOnly,
	}

	result, err := scanService.Scan(cf, opts)
	if err != nil {
		return err
	}

	if result.Count == 0 {
		fmt.Printf("No entries found in column family '%s'\n", cf)
		return nil
	}

	fmt.Printf("Found %d entries in column family '%s':\n", result.Count, cf)
	i := 1
	for k, v := range result.Data {
		fmt.Printf("[%d] Key: %s\n", i, util.FormatKey(k))
		if !keysOnly {
			fmt.Printf("    Value: %s\n", formatValue(v, pretty))
		}
		fmt.Println()
		i++
	}

	return nil
}

// executePrefix executes a prefix scan operation
func executePrefix(rdb db.KeyValueDB, cf, prefix string, pretty bool) error {
	// Use ScanService instead of direct DB access
	scanService := service.NewScanService(rdb)
	result, err := scanService.PrefixScan(cf, prefix, 0) // 0 means no limit
	if err != nil {
		return err
	}

	if result.Count == 0 {
		fmt.Printf("No entries found with prefix '%s' in column family '%s'\n", prefix, cf)
		return nil
	}

	fmt.Printf("Found %d entries with prefix '%s' in column family '%s':\n", result.Count, prefix, cf)
	i := 1
	for k, v := range result.Data {
		fmt.Printf("[%d] Key: %s\n", i, util.FormatKey(k))
		fmt.Printf("    Value: %s\n", formatValue(v, pretty))
		fmt.Println()
		i++
	}

	return nil
}

// executeSearch executes a fuzzy search operation
func executeSearch(rdb db.KeyValueDB, cf, keyPattern, valuePattern string, useRegex, caseSensitive, keysOnly, tick bool, limit int, pretty bool, after, exportFile, exportSep string) error {
	// Use SearchService instead of direct DB access
	searchService := service.NewSearchService(rdb)

	opts := service.SearchOptions{
		KeyPattern:    keyPattern,
		ValuePattern:  valuePattern,
		UseRegex:      useRegex,
		CaseSensitive: caseSensitive,
		KeysOnly:      keysOnly,
		Limit:         limit,
		After:         after,
	}

	// Handle export (still use direct DB access for now as ExportService needs enhancement)
	if exportFile != "" {
		sep := parseSep(exportSep)
		dbOpts := db.SearchOptions{
			KeyPattern:    keyPattern,
			ValuePattern:  valuePattern,
			UseRegex:      useRegex,
			CaseSensitive: caseSensitive,
			KeysOnly:      keysOnly,
			Tick:          tick,
			Limit:         limit,
			After:         after,
		}
		err := rdb.ExportSearchResultsToCSV(cf, exportFile, sep, dbOpts)
		if err != nil {
			return err
		}
		fmt.Printf("Search results exported to %s\n", exportFile)
		return nil
	}

	// Execute search
	results, err := searchService.Search(cf, opts)
	if err != nil {
		return err
	}

	if results.Count == 0 {
		fmt.Printf("No matches found in column family '%s'\n", cf)
		return nil
	}

	fmt.Printf("Found %d matches in column family '%s' (query time: %s)\n\n", results.Count, cf, results.QueryTime)

	for i, result := range results.Results {
		fmt.Printf("[%d] Key: %s\n", i+1, result.Key)
		if !keysOnly {
			fmt.Printf("    Value: %s\n", formatValue(result.Value, pretty))
		}
		if i < len(results.Results)-1 {
			fmt.Println()
		}
	}

	if results.HasMore {
		fmt.Printf("\nMore results available. Use --after='%s' for next page\n", results.NextCursor)
	}

	return nil
}

// parseSep parses separator string, handling escape sequences
func parseSep(s string) string {
	switch s {
	case "\\t":
		return "\t"
	case "\\n":
		return "\n"
	case "\\r":
		return "\r"
	default:
		return s
	}
}

// runGraphChainInteractive starts GraphChain in interactive mode
func runGraphChainInteractive(database db.KeyValueDB) {
	fmt.Println("GraphChain AI Assistant - Interactive Mode")
	fmt.Println("Ask me questions about your database!")
	fmt.Println("Type 'exit' or press Ctrl+C to quit")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  - Show me all users in the database")
	fmt.Println("  - What's the last entry in the logs column family?")
	fmt.Println("  - Find keys that start with 'user:' and contain 'admin'")
	fmt.Println()

	// Setup signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nGoodbye!")
		os.Exit(0)
	}()

	for {
		fmt.Print("ai> ")

		var input string
		fmt.Scanln(&input)

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		if input != "" {
			runGraphChainQuery(database, input)
		}
	}
}

// runGraphChainQuery executes a single GraphChain query
func runGraphChainQuery(database db.KeyValueDB, query string) {
	// Load configuration
	config, err := graphchain.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Warning: Failed to load config, using defaults: %v\n", err)
		config = graphchain.DefaultConfig()
	}

	// Cast to *db.DB (required for GraphChain Agent)
	dbPtr, ok := database.(*db.DB)
	if !ok {
		fmt.Printf("Error: GraphChain Agent requires a writable database connection\n")
		return
	}

	// Create and initialize agent
	agent := graphchain.NewAgent(dbPtr)
	ctx := context.Background()

	err = agent.Initialize(ctx, config)
	if err != nil {
		fmt.Printf("Failed to initialize GraphChain Agent: %v\n", err)
		fmt.Println("\nTroubleshooting:")
		fmt.Println("1. Make sure Ollama is running: ollama serve")
		fmt.Println("2. Check if your model is available: ollama list")
		fmt.Printf("3. Update config file (%s) with correct model name\n", configPath)
		return
	}
	defer agent.Close()

	fmt.Printf("Processing: %s\n", query)
	fmt.Printf("Please wait (this may take up to 20 seconds)...\n")

	// Create a context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	result, err := agent.ProcessQuery(queryCtx, query)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else if result.Success {
		fmt.Printf("Result:\n%v\n", result.Data)
		if result.Explanation != "" {
			fmt.Printf("Explanation: %s\n", result.Explanation)
		}
		fmt.Printf("Execution time: %v\n", result.ExecutionTime)
	} else {
		fmt.Printf("Failed: %s\n", result.Error)
	}
	fmt.Println()
}

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to RocksDB database (required)")
	rootCmd.PersistentFlags().BoolVar(&readOnly, "read-only", false, "Open database in read-only mode")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config/graphchain.yaml", "Path to GraphChain configuration file")
	rootCmd.PersistentFlags().BoolVar(&pretty, "pretty", false, "Pretty print JSON values")

	// Column family flag for commands that need it
	getCmd.Flags().StringP("cf", "c", "default", "Column family")
	putCmd.Flags().StringP("cf", "c", "default", "Column family")
	lastCmd.Flags().StringP("cf", "c", "default", "Column family")
	scanCmd.Flags().StringP("cf", "c", "default", "Column family")
	prefixCmd.Flags().StringP("cf", "c", "default", "Column family")
	searchCmd.Flags().StringP("cf", "c", "default", "Column family")
	exportCmd.Flags().StringP("cf", "c", "default", "Column family")
	watchCmd.Flags().StringP("cf", "c", "default", "Column family")
	keyformatCmd.Flags().StringP("cf", "c", "default", "Column family")
	jsonqueryCmd.Flags().StringP("cf", "c", "default", "Column family")

	// Scan command specific flags
	scanCmd.Flags().Int("limit", 0, "Limit number of results")
	scanCmd.Flags().Bool("reverse", false, "Scan in reverse order")
	scanCmd.Flags().Bool("keys-only", false, "Show only keys, not values")

	// Search command specific flags
	searchCmd.Flags().String("key", "", "Key pattern to search for")
	searchCmd.Flags().String("value", "", "Value pattern to search for")
	searchCmd.Flags().Bool("regex", false, "Use regex patterns instead of wildcards")
	searchCmd.Flags().Bool("case-sensitive", false, "Case sensitive search")
	searchCmd.Flags().Bool("keys-only", false, "Show only keys, not values")
	searchCmd.Flags().Bool("tick", false, "Treat keys as .NET tick times and convert to UTC string format")
	searchCmd.Flags().Int("limit", 50, "Limit search results")
	searchCmd.Flags().String("after", "", "Start search after this key (pagination)")
	searchCmd.Flags().String("export", "", "Export results to CSV file")
	searchCmd.Flags().String("export-sep", ",", "CSV separator for export")

	// Export command specific flags
	exportCmd.Flags().String("sep", ",", "CSV separator")

	// Watch command specific flags
	watchCmd.Flags().Duration("interval", 1*time.Second, "Watch interval")

	// Stats command specific flags
	statsCmd.Flags().String("cf", "", "Column family for stats (omit for database-wide stats)")

	// JSON query command specific flags
	jsonqueryCmd.Flags().String("field", "", "Field name for JSON query")
	jsonqueryCmd.Flags().String("value", "", "Field value for JSON query")

	// Transform command specific flags
	transformCmd.Flags().StringP("cf", "c", "default", "Column family to transform")
	transformCmd.Flags().String("expr", "", "Python expression (e.g., \"value.upper()\")")
	transformCmd.Flags().String("key-expr", "", "Transform keys with Python expression")
	transformCmd.Flags().String("value-expr", "", "Transform values with Python expression (alternative to --expr)")
	transformCmd.Flags().String("filter", "", "Filter entries with Python boolean (e.g., \"'active' in value\")")
	transformCmd.Flags().String("script", "", "Python script file (must define transform_value() and optionally should_process())")
	transformCmd.Flags().Bool("dry-run", false, "Preview mode - show changes without applying them (RECOMMENDED)")
	transformCmd.Flags().Int("limit", 0, "Process only N entries (0 = all, use small number for testing)")
	transformCmd.Flags().Int("batch-size", 1000, "Internal batch size for processing")
	transformCmd.Flags().Bool("verbose", false, "Show detailed progress information")

	// Web command specific flags
	webCmd.Flags().String("port", "8080", "Port to listen on")
	webCmd.Flags().Bool("enable-ai", false, "Enable AI assistant powered by GraphChain")

	// Add all commands to root
	rootCmd.AddCommand(replCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(putCmd)
	rootCmd.AddCommand(lastCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(prefixCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(keyformatCmd)
	rootCmd.AddCommand(jsonqueryCmd)
	rootCmd.AddCommand(listcfCmd)
	rootCmd.AddCommand(createcfCmd)
	rootCmd.AddCommand(dropcfCmd)
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(transformCmd)
	rootCmd.AddCommand(webCmd)

	// Mark required flags
	rootCmd.MarkPersistentFlagRequired("db")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
