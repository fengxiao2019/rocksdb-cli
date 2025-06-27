package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/repl"
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
    --last <cf>                  Get the last key-value pair from column family
    --pretty                     Pretty print JSON values (use with --last)
    --export-cf <cf>             Column family to export (use with --export-file)
    --export-file <file>         Output CSV file path for export
    --watch <cf>                 Watch for new entries in column family (real-time)
    --interval <duration>        Watch interval (default: 1s, e.g., 500ms, 2s, 1m)
    --help                       Show this help message

EXAMPLES:
    # Start interactive mode
    rocksdb-cli --db /path/to/db

    # Start in read-only mode (safe for concurrent access)
    rocksdb-cli --db /path/to/db --read-only

    # Get last entry from a column family in read-only mode
    rocksdb-cli --db /path/to/db --read-only --last users

    # Get last entry with pretty-printed JSON
    rocksdb-cli --db /path/to/db --last users --pretty

    # Export column family to CSV
    rocksdb-cli --db /path/to/db --export-cf users --export-file users.csv

    # Watch for new entries (real-time monitoring)
    rocksdb-cli --db /path/to/db --watch logs --interval 500ms

INTERACTIVE COMMANDS:
    Once in interactive mode, you can use these commands:
    
    usecf <cf>                   Switch current column family
    get [<cf>] <key> [--pretty]  Query by key (use --pretty for JSON formatting)
    put [<cf>] <key> <value>     Insert/Update key-value pair
    prefix [<cf>] <prefix>       Query by key prefix
    scan [<cf>] [start] [end]    Scan range with options
    last [<cf>] [--pretty]       Get last key-value pair from CF
    export [<cf>] <file_path>    Export CF to CSV file
    listcf                       List all column families
    createcf <cf>                Create new column family
    dropcf <cf>                  Drop column family
    help                         Show interactive help
    exit/quit                    Exit the CLI

    Note: Commands without [<cf>] use current column family shown in prompt
    Note: Write operations (put, createcf, dropcf) are disabled in read-only mode

SCAN OPTIONS:
    --limit=N                    Limit number of results
    --reverse                    Scan in reverse order
    --values=no                  Show only keys, not values

For more information, visit: https://github.com/yourusername/rocksdb-cli
`

// formatValue formats a value based on pretty flag (same as in command package)
func formatValue(value string, pretty bool) string {
	if !pretty {
		return value
	}

	var jsonData interface{}
	if err := json.Unmarshal([]byte(value), &jsonData); err != nil {
		return value // If not valid JSON, return as is
	}
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return value // If can't pretty print, return as is
	}
	return string(prettyJSON)
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

	// Start interactive REPL if no special parameters
	repl.Start(rdb)
}
