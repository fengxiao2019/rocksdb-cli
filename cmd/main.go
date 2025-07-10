package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/graphchain"
	"rocksdb-cli/internal/jsonutil"
	"rocksdb-cli/internal/repl"
	"rocksdb-cli/internal/util"

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
	Short: "Interactive RocksDB command-line tool with column family support",
	Long: `RocksDB CLI is a powerful command-line interface for RocksDB databases that provides:
- Interactive REPL with column family support
- Direct command-line operations for scripting
- Data export capabilities
- Real-time monitoring with watch mode
- GraphChain Agent for natural language queries (AI-powered)

For interactive mode, use: rocksdb-cli repl --db /path/to/database`,
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

		value, err := rdb.GetCF(cf, key)
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

		if rdb.IsReadOnly() {
			fmt.Println("Error: Database is in read-only mode")
			os.Exit(1)
		}

		cf := getColumnFamily(cmd)
		key, value := args[0], args[1]

		err := rdb.PutCF(cf, key, value)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
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

		key, value, err := rdb.GetLastCF(cf)
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

		if cf == "" {
			// Database-wide stats
			stats, err := rdb.GetDatabaseStats()
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
			stats, err := rdb.GetCFStats(cf)
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

		result, err := rdb.JSONQueryCF(cf, field, value)
		if err != nil {
			fmt.Printf("JSON query failed: %v\n", err)
			os.Exit(1)
		}

		if len(result) == 0 {
			fmt.Printf("No entries found in '%s' where field '%s' = '%s'\n", cf, field, value)
		} else {
			fmt.Printf("Found %d entries in '%s' where field '%s' = '%s':\n", len(result), cf, field, value)
			for k, v := range result {
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

		cfs, err := rdb.ListCFs()
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

	// Set up scan options
	opts := db.ScanOptions{
		Values:  !keysOnly,
		Reverse: reverse,
	}
	if limit > 0 {
		opts.Limit = limit
	}

	// Use smart scan for automatic key conversion
	results, err := rdb.SmartScanCF(cf, startStr, endStr, opts)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Printf("No entries found in column family '%s'\n", cf)
		return nil
	}

	fmt.Printf("Found %d entries in column family '%s':\n", len(results), cf)
	i := 1
	for k, v := range results {
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
	results, err := rdb.SmartPrefixScanCF(cf, prefix, 0) // 0 means no limit
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Printf("No entries found with prefix '%s' in column family '%s'\n", prefix, cf)
		return nil
	}

	fmt.Printf("Found %d entries with prefix '%s' in column family '%s':\n", len(results), prefix, cf)
	i := 1
	for k, v := range results {
		fmt.Printf("[%d] Key: %s\n", i, util.FormatKey(k))
		fmt.Printf("    Value: %s\n", formatValue(v, pretty))
		fmt.Println()
		i++
	}

	return nil
}

// executeSearch executes a fuzzy search operation
func executeSearch(rdb db.KeyValueDB, cf, keyPattern, valuePattern string, useRegex, caseSensitive, keysOnly, tick bool, limit int, pretty bool, after, exportFile, exportSep string) error {
	opts := db.SearchOptions{
		KeyPattern:    keyPattern,
		ValuePattern:  valuePattern,
		UseRegex:      useRegex,
		CaseSensitive: caseSensitive,
		KeysOnly:      keysOnly,
		Tick:          tick,
		Limit:         limit,
		After:         after,
	}

	// Handle export
	if exportFile != "" {
		sep := parseSep(exportSep)
		err := rdb.ExportSearchResultsToCSV(cf, exportFile, sep, opts)
		if err != nil {
			return err
		}
		fmt.Printf("Search results exported to %s\n", exportFile)
		return nil
	}

	// Execute search
	results, err := rdb.SearchCF(cf, opts)
	if err != nil {
		return err
	}

	if len(results.Results) == 0 {
		fmt.Printf("No matches found in column family '%s'\n", cf)
		return nil
	}

	fmt.Printf("Found %d matches in column family '%s' (query time: %s)\n\n", len(results.Results), cf, results.QueryTime)

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
	fmt.Println("🤖 GraphChain AI Assistant - Interactive Mode")
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

	fmt.Printf("🔍 Processing: %s\n", query)
	fmt.Printf("⏳ Please wait (this may take up to 20 seconds)...\n")

	// Create a context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	result, err := agent.ProcessQuery(queryCtx, query)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else if result.Success {
		fmt.Printf("✅ Result:\n%v\n", result.Data)
		if result.Explanation != "" {
			fmt.Printf("💡 Explanation: %s\n", result.Explanation)
		}
		fmt.Printf("⏱️  Execution time: %v\n", result.ExecutionTime)
	} else {
		fmt.Printf("❌ Failed: %s\n", result.Error)
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

	// Mark required flags
	rootCmd.MarkPersistentFlagRequired("db")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
