package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/jsonutil"
	"rocksdb-cli/internal/repl"
	"sort"
	"syscall"
	"time"
)

const helpText = `rocksdb-cli - Interactive RocksDB command-line tool with column family support

USAGE:
    rocksdb-cli --db <database_path> [OPTIONS]

DESCRIPTION:
    A powerful command-line interface for RocksDB databases that provides:
    - Interactive REPL with column family support
    - Direct command-line operations for scripting
    - Data export capabilities
    - Real-time monitoring with watch mode

OPTIONS:
    --db <path>                  Path to RocksDB database (required)
    --read-only                  Open database in read-only mode (safe for concurrent access)
    --help                       Show this help message

DATA QUERY OPTIONS:
    --last <cf>                  Get the last key-value pair from column family
    --prefix <cf>                Column family for prefix scan
    --prefix-key <prefix>        Key prefix to search for (use with --prefix)
    --search <cf>                Column family for fuzzy search
    --search-key <pattern>       Key pattern to search for (use with --search)
    --search-value <pattern>     Value pattern to search for (use with --search)
    --search-limit <N>           Limit search results (use with --search, default: 50)
    --search-regex               Use regex patterns instead of wildcards (use with --search)
    --search-case-sensitive      Case sensitive search (use with --search)
    --search-keys-only           Show only keys, not values (use with --search)
    --scan <cf>                  Column family to scan
    --start <key>                Start key for scan (use with --scan, * for beginning)
    --end <key>                  End key for scan (use with --scan, * for end)
    --limit <N>                  Limit number of scan results (use with --scan)
    --reverse                    Scan in reverse order (use with --scan)
    --keys-only                  Show only keys, not values (use with --scan)
    --pretty                     Pretty print JSON values (use with --last, --prefix)

UTILITY OPTIONS:
    --export-cf <cf>             Column family to export (use with --export-file)
    --export-file <file>         Output CSV file path for export
    --watch <cf>                 Watch for new entries in column family (real-time)
    --interval <duration>        Watch interval (default: 1s, e.g., 500ms, 2s, 1m)

EXAMPLES:
    # Interactive mode
    rocksdb-cli --db /path/to/db
    rocksdb-cli --db /path/to/db --read-only  # Safe for concurrent access

    # Basic data queries
    rocksdb-cli --db /path/to/db --last users
    rocksdb-cli --db /path/to/db --last users --pretty

    # Prefix scanning (find keys starting with pattern)
    rocksdb-cli --db /path/to/db --prefix users --prefix-key "user:"
    rocksdb-cli --db /path/to/db --prefix users --prefix-key "user:" --pretty
    rocksdb-cli --db /path/to/db --prefix logs --prefix-key "error:"

    # Fuzzy searching (find keys or values containing patterns)
    rocksdb-cli --db /path/to/db --search users --search-key "user"
    rocksdb-cli --db /path/to/db --search users --search-value "Alice"
    rocksdb-cli --db /path/to/db --search users --search-key "temp:*" --search-value "Alice"
    rocksdb-cli --db /path/to/db --search logs --search-value "Error" --search-limit 5
    rocksdb-cli --db /path/to/db --search users --search-key "user:[0-9]+" --search-regex --pretty

    # Range scanning with options
    rocksdb-cli --db /path/to/db --scan users
    rocksdb-cli --db /path/to/db --scan users --start "user:1000" --end "user:2000"
    rocksdb-cli --db /path/to/db --scan users --start "user:1000" --limit 10 --reverse
    rocksdb-cli --db /path/to/db --scan users --keys-only

    # Utility operations
    rocksdb-cli --db /path/to/db --export-cf users --export-file users.csv
    rocksdb-cli --db /path/to/db --watch logs --interval 500ms

INTERACTIVE COMMANDS:
    Once in interactive mode, you can use these commands:
    
    # Column family management
    usecf <cf>                   Switch current column family
    listcf                       List all column families
    createcf <cf>                Create new column family
    dropcf <cf>                  Drop column family

    # Data operations
    get [<cf>] <key> [--pretty]  Query by key (use --pretty for JSON formatting)
    put [<cf>] <key> <value>     Insert/Update key-value pair
    prefix [<cf>] <prefix> [--pretty]  Query by key prefix (supports --pretty)
    scan [<cf>] [start] [end] [options]  Scan range with options
    last [<cf>] [--pretty]       Get last key-value pair from CF

    # Utility operations
    export [<cf>] <file_path>    Export CF to CSV file
    help                         Show interactive help
    exit/quit                    Exit the CLI

    USAGE PATTERNS:
    • Commands without [<cf>] use current column family shown in prompt
    • Commands with [<cf>] can specify column family explicitly
    • Write operations (put, createcf, dropcf) are disabled in read-only mode

    PREFIX EXAMPLES:
    prefix user:                 # Find all keys starting with "user:" in current CF
    prefix users user:           # Find all keys starting with "user:" in "users" CF
    prefix user: --pretty        # Same as first, but with pretty JSON formatting

SCAN OPTIONS:
    --limit=N                    Limit number of results
    --reverse                    Scan in reverse order
    --values=no                  Show only keys, not values

For more information, visit: https://github.com/yourusername/rocksdb-cli
`

// formatValue formats a value based on pretty flag using jsonutil
func formatValue(value string, pretty bool) string {
	if !pretty {
		return value
	}
	return jsonutil.PrettyPrintWithNestedExpansion(value)
}

// executeScan executes a scan operation with the given parameters
// This function avoids code duplication between interactive and non-interactive modes
func executeScan(rdb db.KeyValueDB, cf string, start, end *string, limit int, reverse, keysOnly bool) error {
	// Convert start and end to byte slices, handling * wildcard
	var startBytes, endBytes []byte
	if start != nil && *start != "*" && *start != "" {
		startBytes = []byte(*start)
	}
	if end != nil && *end != "*" && *end != "" {
		endBytes = []byte(*end)
	}

	// Set up scan options
	opts := db.ScanOptions{
		Values:  !keysOnly,
		Reverse: reverse,
	}
	if limit > 0 {
		opts.Limit = limit
	}

	// Execute scan
	result, err := rdb.ScanCF(cf, startBytes, endBytes, opts)
	if err != nil {
		return fmt.Errorf("scan failed: %v", err)
	}

	// Sort keys to ensure consistent output order (same logic as interactive mode)
	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}

	// Sort keys based on scan direction
	if reverse {
		sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })
	} else {
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	}

	// Output in sorted order
	for _, k := range keys {
		if keysOnly {
			fmt.Printf("%s\n", k)
		} else {
			v := result[k]
			fmt.Printf("%s: %s\n", k, v)
		}
	}

	return nil
}

// executePrefix executes a prefix scan operation with the given parameters
// This function avoids code duplication between interactive and non-interactive modes
func executePrefix(rdb db.KeyValueDB, cf, prefix string, pretty bool) error {
	// Execute prefix scan with default limit of 20 (same as interactive mode)
	result, err := rdb.PrefixScanCF(cf, prefix, 20)
	if err != nil {
		return fmt.Errorf("prefix scan failed: %v", err)
	}

	// Sort keys to ensure consistent output order (same logic as interactive mode)
	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}

	// Sort in lexicographical order
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	// Output in sorted order with optional pretty printing
	for _, k := range keys {
		v := result[k]
		formattedValue := formatValue(v, pretty)
		fmt.Printf("%s: %s\n", k, formattedValue)
	}

	return nil
}

// executeSearch executes a search operation with the given parameters
// This function avoids code duplication between interactive and non-interactive modes
func executeSearch(rdb db.KeyValueDB, cf, keyPattern, valuePattern string, useRegex, caseSensitive, keysOnly bool, limit int, pretty bool) error {
	// Validate that at least one pattern is provided
	if keyPattern == "" && valuePattern == "" {
		return fmt.Errorf("must specify at least --search-key or --search-value pattern")
	}

	// Set up search options
	opts := db.SearchOptions{
		KeyPattern:    keyPattern,
		ValuePattern:  valuePattern,
		UseRegex:      useRegex,
		CaseSensitive: caseSensitive,
		KeysOnly:      keysOnly,
		Limit:         limit,
	}

	// Execute search
	results, err := rdb.SearchCF(cf, opts)
	if err != nil {
		return fmt.Errorf("search failed: %v", err)
	}

	// Display results
	if len(results.Results) == 0 {
		fmt.Println("No matches found")
		fmt.Printf("Query took: %s\n", results.QueryTime)
		return nil
	}

	// Show header with result count and timing
	limitedText := ""
	if results.Limited {
		limitedText = " (limited)"
	}
	fmt.Printf("Found %d matches%s in %s\n\n", len(results.Results), limitedText, results.QueryTime)

	// Output results
	for i, result := range results.Results {
		// Format matched fields display
		var matchedFields []string
		for _, field := range result.MatchedFields {
			matchedFields = append(matchedFields, field)
		}
		matchedFieldsStr := ""
		if len(matchedFields) > 0 {
			matchedFieldsStr = " (matched: " + fmt.Sprintf("%v", matchedFields) + ")"
		}

		fmt.Printf("[%d] Key: %s%s\n", i+1, result.Key, matchedFieldsStr)

		if !keysOnly {
			formattedValue := formatValue(result.Value, pretty)
			fmt.Printf("    %s\n", formattedValue)
		}
		if i < len(results.Results)-1 {
			fmt.Println()
		}
	}

	fmt.Printf("\nQuery completed in %s\n", results.QueryTime)
	return nil
}

func main() {
	// Custom usage function
	flag.Usage = func() {
		fmt.Print(helpText)
	}

	dbPath := flag.String("db", "", "Path to RocksDB database")
	exportCF := flag.String("export-cf", "", "Column family to export")
	exportFile := flag.String("export-file", "", "Output CSV file path")
	lastCF := flag.String("last", "", "Get last key-value pair from column family")
	prettyFlag := flag.Bool("pretty", false, "Pretty print JSON values (use with --last)")
	watchCF := flag.String("watch", "", "Watch for new entries in column family (like ping -t)")
	watchInterval := flag.Duration("interval", 1*time.Second, "Watch interval (default: 1s)")
	// Scan command flags
	scanCF := flag.String("scan", "", "Column family to scan")
	scanStart := flag.String("start", "", "Start key for scan (use with --scan)")
	scanEnd := flag.String("end", "", "End key for scan (use with --scan)")
	scanLimit := flag.Int("limit", 0, "Limit number of scan results (use with --scan)")
	scanReverse := flag.Bool("reverse", false, "Scan in reverse order (use with --scan)")
	scanKeysOnly := flag.Bool("keys-only", false, "Show only keys, not values (use with --scan)")
	// Prefix command flags
	prefixCF := flag.String("prefix", "", "Column family for prefix scan")
	prefixKey := flag.String("prefix-key", "", "Key prefix to search for (use with --prefix)")
	// Search command flags
	searchCF := flag.String("search", "", "Column family for fuzzy search")
	searchKey := flag.String("search-key", "", "Key pattern to search for (use with --search)")
	searchValue := flag.String("search-value", "", "Value pattern to search for (use with --search)")
	searchLimit := flag.Int("search-limit", 50, "Limit search results (use with --search, default: 50)")
	searchRegex := flag.Bool("search-regex", false, "Use regex patterns instead of wildcards (use with --search)")
	searchCaseSensitive := flag.Bool("search-case-sensitive", false, "Case sensitive search (use with --search)")
	searchKeysOnly := flag.Bool("search-keys-only", false, "Show only keys, not values (use with --search)")
	helpFlag := flag.Bool("help", false, "Show help message")
	readOnlyFlag := flag.Bool("read-only", false, "Open database in read-only mode (safe for concurrent access)")
	flag.Parse()

	// Show help if requested
	if *helpFlag {
		fmt.Print(helpText)
		os.Exit(0)
	}

	if *dbPath == "" {
		fmt.Println("Error: --db parameter is required")
		fmt.Println("\nUse --help for detailed usage information")
		fmt.Println("Quick start: rocksdb-cli --db /path/to/database")
		os.Exit(1)
	}

	var rdb db.KeyValueDB
	var err error
	if *readOnlyFlag {
		rdb, err = db.OpenReadOnly(*dbPath)
	} else {
		rdb, err = db.Open(*dbPath)
	}
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Handle export functionality
	if *exportCF != "" && *exportFile != "" {
		err := rdb.ExportToCSV(*exportCF, *exportFile)
		if err != nil {
			fmt.Printf("Export failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully exported column family '%s' to '%s'\n", *exportCF, *exportFile)
		return
	}

	// Handle get last functionality
	if *lastCF != "" {
		key, value, err := rdb.GetLastCF(*lastCF)
		if err != nil {
			fmt.Printf("Get last failed: %v\n", err)
			os.Exit(1)
		}
		formattedValue := formatValue(value, *prettyFlag)
		fmt.Printf("Last entry in '%s': %s = %s\n", *lastCF, key, formattedValue)
		return
	}

	// Handle scan functionality
	if *scanCF != "" {
		err := executeScan(rdb, *scanCF, scanStart, scanEnd, *scanLimit, *scanReverse, *scanKeysOnly)
		if err != nil {
			fmt.Printf("Scan failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Handle prefix functionality
	if *prefixCF != "" {
		if *prefixKey == "" {
			fmt.Println("Error: --prefix-key parameter is required when using --prefix")
			fmt.Println("Usage: rocksdb-cli --db <path> --prefix <cf> --prefix-key <prefix> [--pretty]")
			os.Exit(1)
		}
		err := executePrefix(rdb, *prefixCF, *prefixKey, *prettyFlag)
		if err != nil {
			fmt.Printf("Prefix scan failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Check for prefix parameter errors
	if *prefixKey != "" {
		if *prefixCF == "" {
			fmt.Println("--prefix <cf> must be specified when using --prefix-key")
			fmt.Println("Usage: rocksdb-cli --db <path> --prefix <cf> --prefix-key <prefix> [--pretty]")
			os.Exit(1)
		}
	}

	// Check for search parameter errors
	if *searchKey != "" || *searchValue != "" || *searchLimit != 50 || *searchRegex || *searchCaseSensitive || *searchKeysOnly {
		if *searchCF == "" {
			fmt.Println("--search <cf> must be specified when using search options")
			fmt.Println("Usage: rocksdb-cli --db <path> --search <cf> [--search-key <pattern>] [--search-value <pattern>] [options]")
			os.Exit(1)
		}
	}

	// Handle search functionality
	if *searchCF != "" {
		err := executeSearch(rdb, *searchCF, *searchKey, *searchValue, *searchRegex, *searchCaseSensitive, *searchKeysOnly, *searchLimit, *prettyFlag)
		if err != nil {
			fmt.Printf("Search failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Handle watch functionality
	if *watchCF != "" {
		fmt.Printf("Watching column family '%s' for new entries (interval: %v)...\n", *watchCF, *watchInterval)
		fmt.Println("Press Ctrl+C to stop")

		// Set up signal handling for graceful shutdown
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		var lastKey, lastValue string

		// Get initial last entry
		key, value, err := rdb.GetLastCF(*watchCF)
		if err != nil {
			if err.Error() != "column family is empty" {
				fmt.Printf("Watch failed: %v\n", err)
				os.Exit(1)
			}
			// Column family is empty, start with empty values
			lastKey = ""
			lastValue = ""
		} else {
			lastKey = key
			lastValue = value
			fmt.Printf("[%s] Initial: %s = %s\n", time.Now().Format("15:04:05"), key, value)
		}

		ticker := time.NewTicker(*watchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-c:
				fmt.Println("\nStopping watch...")
				return
			case <-ticker.C:
				key, value, err := rdb.GetLastCF(*watchCF)
				if err != nil {
					if err.Error() == "column family is empty" {
						// Still empty, continue
						continue
					}
					fmt.Printf("Watch error: %v\n", err)
					continue
				}

				// Check if there's a new entry
				if key != lastKey || value != lastValue {
					fmt.Printf("[%s] New: %s = %s\n", time.Now().Format("15:04:05"), key, value)
					lastKey = key
					lastValue = value
				}
			}
		}
	}

	// If only one export parameter is provided, show usage
	if *exportCF != "" || *exportFile != "" {
		fmt.Println("Both --export-cf and --export-file must be specified for export")
		fmt.Println("Usage: rocksdb-cli --db <path> --export-cf <cf> --export-file <file.csv>")
		os.Exit(1)
	}

	// Check for scan parameter errors
	if *scanStart != "" || *scanEnd != "" || *scanLimit > 0 || *scanReverse || *scanKeysOnly {
		if *scanCF == "" {
			fmt.Println("--scan <cf> must be specified when using scan options")
			fmt.Println("Usage: rocksdb-cli --db <path> --scan <cf> [--start <key>] [--end <key>] [--limit <N>] [--reverse] [--keys-only]")
			os.Exit(1)
		}
	}

	// Start interactive REPL if no special parameters
	repl.Start(rdb)
}
