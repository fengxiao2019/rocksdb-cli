package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/graphchain"
	"rocksdb-cli/internal/jsonutil"
	"rocksdb-cli/internal/util"
	"sort"
	"strconv"
	"strings"
	"time"
)

// parseSep parses separator string and handles common escape sequences
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

type ReplState struct {
	CurrentCF string
}

type Handler struct {
	DB              db.KeyValueDB
	State           interface{}       // *ReplState, used to manage the active column family
	GraphChainAgent *graphchain.Agent // GraphChain agent for natural language queries
}

// prettyPrintJSON formats JSON with recursive nested expansion using jsonutil
func prettyPrintJSON(val string) string {
	return jsonutil.PrettyPrintWithNestedExpansion(val)
}

// handleError provides user-friendly error messages for specific error types
func handleError(err error, operation string, params ...string) {
	if errors.Is(err, db.ErrKeyNotFound) {
		if len(params) >= 2 {
			fmt.Printf("Key '%s' not found in column family '%s'\n", params[0], params[1])
		} else {
			fmt.Println("Key not found")
		}
	} else if errors.Is(err, db.ErrColumnFamilyNotFound) {
		if len(params) >= 1 {
			fmt.Printf("Column family '%s' does not exist\n", params[0])
		} else {
			fmt.Println("Column family not found")
		}
	} else if errors.Is(err, db.ErrColumnFamilyExists) {
		if len(params) >= 1 {
			fmt.Printf("Column family '%s' already exists\n", params[0])
		} else {
			fmt.Println("Column family already exists")
		}
	} else if errors.Is(err, db.ErrReadOnlyMode) {
		fmt.Println("Operation not allowed in read-only mode")
	} else if errors.Is(err, db.ErrColumnFamilyEmpty) {
		if len(params) >= 1 {
			fmt.Printf("Column family '%s' is empty\n", params[0])
		} else {
			fmt.Println("Column family is empty")
		}
	} else if errors.Is(err, db.ErrDatabaseClosed) {
		fmt.Println("Database is closed")
	} else {
		// Generic error message for unknown errors
		fmt.Printf("%s failed: %v\n", operation, err)
	}
}

// parseTimestamp attempts to parse a key as a timestamp and return formatted UTC time
// Supports various timestamp formats: Unix seconds, Unix milliseconds, Unix microseconds, Unix nanoseconds
func parseTimestamp(key string) string {
	// Try to parse as integer timestamp
	if ts, err := strconv.ParseInt(key, 10, 64); err == nil {
		var t time.Time

		// Determine timestamp format based on number of digits
		switch {
		case ts > 1e15: // Nanoseconds (16+ digits)
			t = time.Unix(0, ts)
		case ts > 1e12: // Microseconds (13-15 digits)
			t = time.Unix(0, ts*1000)
		case ts > 1e9: // Milliseconds (10-12 digits)
			t = time.Unix(0, ts*1e6)
		case ts > 1e6: // Seconds (7-9 digits, covers years ~1973-2033)
			t = time.Unix(ts, 0)
		default:
			return "" // Too small to be a reasonable timestamp
		}

		return t.UTC().Format("2006-01-02 15:04:05 UTC")
	}

	// Try to parse as float timestamp (seconds with fractional part)
	if ts, err := strconv.ParseFloat(key, 64); err == nil {
		if ts > 1e6 && ts < 1e12 { // Reasonable range for Unix timestamp in seconds
			t := time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e9))
			return t.UTC().Format("2006-01-02 15:04:05 UTC")
		}
	}

	return "" // Not a timestamp
}

func parseFlags(args []string) (map[string]string, []string) {
	flags := make(map[string]string)
	var nonFlags []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg[2:], "=", 2)
			if len(parts) == 2 {
				flags[parts[0]] = parts[1]
			} else {
				flags[parts[0]] = "true"
			}
		} else {
			nonFlags = append(nonFlags, arg)
		}
	}

	return flags, nonFlags
}

// formatValue formats a value based on pretty flag
func formatValue(value string, pretty bool) string {
	if pretty {
		return prettyPrintJSON(value)
	}
	return value
}

// parseCFAndArgs extracts column family and remaining arguments from command parts
// Supports patterns like: cmd [<cf>] <args...> [--flags]
func parseCFAndArgs(parts []string, currentCF string) (cf string, args []string, flags map[string]string) {
	if len(parts) <= 1 {
		return currentCF, []string{}, make(map[string]string)
	}

	// Parse flags first
	flags, nonFlags := parseFlags(parts[1:])

	// If no non-flag arguments, use current CF
	if len(nonFlags) == 0 {
		return currentCF, []string{}, flags
	}

	// If first arg looks like a CF name and we have more args, treat it as CF
	// Otherwise, treat first arg as a command parameter and use current CF
	if len(nonFlags) >= 2 {
		// Multiple args: first could be CF
		return nonFlags[0], nonFlags[1:], flags
	} else {
		// Single arg: use current CF, arg is the parameter
		return currentCF, nonFlags, flags
	}
}

func (h *Handler) Execute(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return true
	}
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "usecf":
		if len(parts) != 2 {
			fmt.Println("Usage: usecf <cf>")
			return true
		}
		if h.State != nil {
			if s, ok := h.State.(*ReplState); ok {
				s.CurrentCF = parts[1]
				fmt.Printf("Switched to column family: %s\n", parts[1])
			}
		}
		return true
	case "scan":
		flags, args := parseFlags(parts[1:])
		cf := ""
		var startStr, endStr string
		useSmart := flags["smart"] != "false" // Default to true, can disable with --smart=false

		// Get current CF if available
		currentCF := ""
		if s, ok := h.State.(*ReplState); ok && s != nil {
			currentCF = s.CurrentCF
		}

		// Parse column family and range
		switch len(args) {
		case 0: // scan (no args)
			fmt.Println("Usage: scan [<cf>] [start] [end] [--limit=N] [--reverse] [--values=no] [--timestamp] [--smart=true|false]")
			fmt.Println("  Use * as wildcard to scan all entries (e.g., scan * or scan * *)")
			fmt.Println("  --smart=false disables automatic key format conversion")
			return true
		case 1: // scan <start> (using current CF) or scan * (scan all)
			if currentCF == "" {
				fmt.Println("No current column family set")
				return true
			}
			cf = currentCF
			// Handle * wildcard to scan all entries
			if args[0] == "*" {
				// Leave start and end empty to scan all entries
				startStr = ""
				endStr = ""
			} else {
				startStr = args[0]
			}
		case 2:
			// Check if first arg is likely a CF name by checking if it exists
			// If we can't determine, prefer using current CF with start/end pattern
			if currentCF != "" {
				// scan <start> <end> (using current CF)
				cf = currentCF
				// Handle * wildcards
				if args[0] == "*" {
					startStr = ""
				} else {
					startStr = args[0]
				}
				if args[1] == "*" {
					endStr = ""
				} else {
					endStr = args[1]
				}
			} else {
				// scan <cf> <start> (no current CF set)
				cf = args[0]
				if args[1] == "*" {
					startStr = ""
				} else {
					startStr = args[1]
				}
			}
		case 3: // scan <cf> <start> <end>
			cf = args[0]
			// Handle * wildcards
			if args[1] == "*" {
				startStr = ""
			} else {
				startStr = args[1]
			}
			if args[2] == "*" {
				endStr = ""
			} else {
				endStr = args[2]
			}
		default:
			fmt.Println("Usage: scan [<cf>] [start] [end] [--limit=N] [--reverse] [--values=no] [--timestamp] [--smart=true|false]")
			fmt.Println("  Use * as wildcard to scan all entries (e.g., scan * or scan * *)")
			fmt.Println("  --smart=false disables automatic key format conversion")
			return true
		}

		// Parse options
		opts := db.ScanOptions{
			Values: true, // Default to showing values
		}

		userSetLimit := false
		if limit, ok := flags["limit"]; ok {
			if n, err := strconv.Atoi(limit); err == nil && n > 0 {
				opts.Limit = n
				userSetLimit = true
			} else if n == 0 {
				opts.Limit = 0 // explicit unlimited
				userSetLimit = true
			} else {
				fmt.Println("Invalid limit value")
				return true
			}
		}

		if !userSetLimit {
			if _, hasAfter := flags["after"]; hasAfter || !hasAfter {
				opts.Limit = 100
			}
		}

		if _, ok := flags["reverse"]; ok {
			opts.Reverse = true
		}

		if val, ok := flags["values"]; ok && val == "no" {
			opts.Values = false
		}

		// Check for timestamp flag
		showTimestamp := flags["timestamp"] == "true"

		var result map[string]string
		var err error

		if useSmart {
			result, err = h.DB.SmartScanCF(cf, startStr, endStr, opts)
		} else {
			// Convert strings to bytes for regular scan
			var start, end []byte
			if startStr != "" {
				start = []byte(startStr)
			}
			if endStr != "" {
				end = []byte(endStr)
			}
			result, err = h.DB.ScanCF(cf, start, end, opts)
		}

		if err != nil {
			handleError(err, "Scan", cf)
		} else {
			// Sort keys to ensure consistent output order
			keys := make([]string, 0, len(result))
			for k := range result {
				keys = append(keys, k)
			}

			// Sort keys based on scan direction
			if opts.Reverse {
				// Sort in reverse lexicographical order
				for i := 0; i < len(keys)-1; i++ {
					for j := i + 1; j < len(keys); j++ {
						if keys[i] < keys[j] {
							keys[i], keys[j] = keys[j], keys[i]
						}
					}
				}
			} else {
				// Sort in forward lexicographical order
				for i := 0; i < len(keys)-1; i++ {
					for j := i + 1; j < len(keys); j++ {
						if keys[i] > keys[j] {
							keys[i], keys[j] = keys[j], keys[i]
						}
					}
				}
			}

			// Output in sorted order
			for _, k := range keys {
				v := result[k]
				if showTimestamp {
					if timestamp := parseTimestamp(k); timestamp != "" {
						if opts.Values {
							fmt.Printf("%s (%s): %s\n", util.FormatKey(k), timestamp, v)
						} else {
							fmt.Printf("%s (%s)\n", util.FormatKey(k), timestamp)
						}
					} else {
						if opts.Values {
							fmt.Printf("%s: %s\n", util.FormatKey(k), v)
						} else {
							fmt.Printf("%s\n", util.FormatKey(k))
						}
					}
				} else {
					if opts.Values {
						fmt.Printf("%s: %s\n", util.FormatKey(k), v)
					} else {
						fmt.Printf("%s\n", util.FormatKey(k))
					}
				}
			}
		}
	case "get":
		// Parse flags and arguments
		flags, args := parseFlags(parts[1:])
		pretty := flags["pretty"] == "true"
		useSmart := flags["smart"] != "false" // Default to true, can disable with --smart=false

		// Get current CF if available
		currentCF := ""
		if s, ok := h.State.(*ReplState); ok && s != nil {
			currentCF = s.CurrentCF
		}

		var cf, key string

		switch len(args) {
		case 1: // get <key> (using current CF)
			if currentCF == "" {
				fmt.Println("No current column family set")
				return true
			}
			cf = currentCF
			key = args[0]
		case 2: // get <cf> <key>
			cf = args[0]
			key = args[1]
		default:
			fmt.Println("Usage: get [<cf>] <key> [--pretty] [--smart=true|false]")
			fmt.Println("  Query by key with automatic binary key conversion")
			fmt.Println("  --smart=false disables automatic key format conversion")
			return true
		}

		var val string
		var err error

		if useSmart {
			val, err = h.DB.SmartGetCF(cf, key)
		} else {
			val, err = h.DB.GetCF(cf, key)
		}

		if err != nil {
			handleError(err, "Query", key, cf)
		} else {
			if pretty {
				fmt.Printf("%s\n", prettyPrintJSON(val))
			} else {
				fmt.Printf("%s\n", val)
			}
		}
	case "put":
		cf, key, value := "", "", ""
		if len(parts) == 3 { // put <key> <value>
			if s, ok := h.State.(*ReplState); ok && s != nil {
				cf = s.CurrentCF
				key = parts[1]
				value = parts[2]
			} else {
				fmt.Println("No current column family set")
				return true
			}
		} else if len(parts) == 4 { // put <cf> <key> <value>
			cf = parts[1]
			key = parts[2]
			value = parts[3]
		} else {
			fmt.Println("Usage: put [<cf>] <key> <value>")
			return true
		}
		err := h.DB.PutCF(cf, key, value)
		if err != nil {
			handleError(err, "Write", cf)
		} else {
			fmt.Println("OK")
		}
	case "prefix":
		// Get current CF if available
		currentCF := ""
		if s, ok := h.State.(*ReplState); ok && s != nil {
			currentCF = s.CurrentCF
		}

		// Parse flags and arguments
		flags, args := parseFlags(parts[1:])
		pretty := flags["pretty"] == "true"
		useSmart := flags["smart"] != "false" // Default to true, can disable with --smart=false

		var cf, prefix string

		switch len(args) {
		case 1: // prefix <prefix> (using current CF)
			if currentCF == "" {
				fmt.Println("No current column family set")
				return true
			}
			cf = currentCF
			prefix = args[0]
		case 2: // prefix <cf> <prefix>
			cf = args[0]
			prefix = args[1]
		default:
			fmt.Println("Usage: prefix [<cf>] <prefix> [--pretty] [--smart=true|false]")
			fmt.Println("  Query by key prefix with automatic binary key conversion")
			fmt.Println("  --smart=false disables automatic key format conversion")
			return true
		}

		var result map[string]string
		var err error

		if useSmart {
			result, err = h.DB.SmartPrefixScanCF(cf, prefix, 20)
		} else {
			result, err = h.DB.PrefixScanCF(cf, prefix, 20)
		}

		if err != nil {
			handleError(err, "Prefix scan", cf)
		} else {
			// Sort keys to ensure consistent output order
			keys := make([]string, 0, len(result))
			for k := range result {
				keys = append(keys, k)
			}

			// Sort in lexicographical order
			for i := 0; i < len(keys)-1; i++ {
				for j := i + 1; j < len(keys); j++ {
					if keys[i] > keys[j] {
						keys[i], keys[j] = keys[j], keys[i]
					}
				}
			}

			// Output in sorted order with optional pretty printing
			for _, k := range keys {
				v := result[k]
				formattedValue := formatValue(v, pretty)
				fmt.Printf("%s: %s\n", util.FormatKey(k), formattedValue)
			}

			// In the prefix command handler, after detecting key format:
			keyFormat, _ := h.DB.GetKeyFormatInfo(cf)
			if keyFormat == util.KeyFormatUint64BE || keyFormat == util.KeyFormatHex || keyFormat == util.KeyFormatMixed {
				fmt.Println("[Notice] This column family uses binary/uint64 keys. Prefix only matches byte prefixes, not numeric string prefixes. For example, use a hex prefix like 'prefix 0x00' to match all keys starting with 0x00, or use a full uint64 value.")
			}
		}
	case "listcf":
		cfs, err := h.DB.ListCFs()
		if err != nil {
			handleError(err, "List column families")
		} else {
			fmt.Println("Column Families:")
			for _, cf := range cfs {
				fmt.Println(cf)
			}
		}
	case "createcf":
		if len(parts) != 2 {
			fmt.Println("Usage: createcf <cf>")
			return true
		}
		err := h.DB.CreateCF(parts[1])
		if err != nil {
			handleError(err, "Create column family", parts[1])
		} else {
			fmt.Println("OK")
		}
	case "dropcf":
		if len(parts) != 2 {
			fmt.Println("Usage: dropcf <cf>")
			return true
		}
		err := h.DB.DropCF(parts[1])
		if err != nil {
			handleError(err, "Drop column family", parts[1])
		} else {
			fmt.Println("OK")
		}
	case "export":
		cf, filePath := "", ""
		if len(parts) == 2 && h.State != nil {
			// export <file_path> - use current CF
			if s, ok := h.State.(*ReplState); ok && s != nil {
				cf = s.CurrentCF
				filePath = parts[1]
			} else {
				fmt.Println("No current column family set")
				return true
			}
		} else if len(parts) == 3 {
			// export <cf> <file_path>
			cf = parts[1]
			filePath = parts[2]
		} else {
			fmt.Println("Usage: export [<cf>] <file_path> [--sep=<sep>]")
			fmt.Println("  Export column family data to CSV file")
			fmt.Println("  --sep=<sep>   CSV separator (default: ,). Supports \\t for tab, ; for semicolon, etc.")
			fmt.Println("  Example: export users users.csv --sep=\";\"")
			fmt.Println("           export logs logs.tsv --sep=\"\\t\"")
			return true
		}

		err := h.DB.ExportToCSV(cf, filePath, ",")
		if err != nil {
			handleError(err, "Export", cf)
		} else {
			fmt.Printf("Successfully exported column family '%s' to '%s'\n", cf, filePath)
		}
	case "last":
		var cf string
		var pretty bool

		// Get current CF if available
		currentCF := ""
		if s, ok := h.State.(*ReplState); ok && s != nil {
			currentCF = s.CurrentCF
		}

		// Parse command arguments
		if len(parts) == 1 {
			// last - use current CF
			if currentCF == "" {
				fmt.Println("No current column family set")
				return true
			}
			cf = currentCF
		} else {
			// Parse flags and arguments
			flags, nonFlags := parseFlags(parts[1:])
			pretty = flags["pretty"] == "true"

			if len(nonFlags) == 0 {
				// last --pretty
				if currentCF == "" {
					fmt.Println("No current column family set")
					return true
				}
				cf = currentCF
			} else if len(nonFlags) == 1 {
				// last <cf> or last <cf> --pretty
				cf = nonFlags[0]
			} else {
				fmt.Println("Usage: last [<cf>] [--pretty]")
				fmt.Println("  Get the last key-value pair from column family")
				return true
			}
		}

		key, value, err := h.DB.GetLastCF(cf)
		if err != nil {
			handleError(err, "Get last", cf)
		} else {
			formattedValue := formatValue(value, pretty)
			fmt.Printf("Last entry in '%s': %s = %s\n", cf, util.FormatKey(key), formattedValue)
		}
	case "jpath", "jsonpath":
		// Get current CF if available
		currentCF := ""
		if s, ok := h.State.(*ReplState); ok && s != nil {
			currentCF = s.CurrentCF
		}

		// Parse flags and arguments
		flags, args := parseFlags(parts[1:])
		pretty := flags["pretty"] == "true"
		useSmart := flags["smart"] != "false" // Default to true

		var cf, key, jsonPathExpr string

		switch len(args) {
		case 2: // jpath <key> <jsonpath> (using current CF)
			if currentCF == "" {
				fmt.Println("No current column family set")
				return true
			}
			cf = currentCF
			key = args[0]
			jsonPathExpr = args[1]
		case 3: // jpath <cf> <key> <jsonpath>
			cf = args[0]
			key = args[1]
			jsonPathExpr = args[2]
		default:
			fmt.Println("Usage: jpath [<cf>] <key> <jsonpath> [--pretty] [--smart=true|false]")
			fmt.Println("  Query JSON value using JSONPath expression")
			fmt.Println("  Examples:")
			fmt.Println("    jpath user:123 \"$.name\"")
			fmt.Println("    jpath users user:123 \"$.profile.email\"")
			fmt.Println("    jpath orders order:456 \"$.items[0].price\" --pretty")
			fmt.Println("    jpath products prod:789 \"$.tags[*]\"")
			fmt.Println("")
			fmt.Println("  Common JSONPath expressions:")
			fmt.Println("    $.field         - Get field value")
			fmt.Println("    $.user.name     - Get nested field")
			fmt.Println("    $.items[0]      - Get first array item")
			fmt.Println("    $.items[*]      - Get all array items")
			fmt.Println("    $               - Get entire JSON")
			return true
		}

		// Get the value from the database
		var value string
		var err error

		if useSmart {
			value, err = h.DB.SmartGetCF(cf, key)
		} else {
			value, err = h.DB.GetCF(cf, key)
		}

		if err != nil {
			handleError(err, "Query", key, cf)
			return true
		}

		// Check if value is valid JSON
		if !jsonutil.IsValidJSON(value) {
			fmt.Printf("Error: Value for key '%s' is not valid JSON\n", key)
			fmt.Printf("Value: %s\n", value)
			return true
		}

		// Query using JSONPath
		result, err := jsonutil.QueryJSONPath(value, jsonPathExpr)
		if err != nil {
			fmt.Printf("JSONPath query error: %v\n", err)
			return true
		}

		// Format and display result
		if pretty {
			formattedResult := jsonutil.PrettyPrintWithNestedExpansion(result)
			fmt.Printf("%s\n", formattedResult)
		} else {
			fmt.Printf("%s\n", result)
		}
	case "jsonquery":
		// Get current CF if available
		currentCF := ""
		if s, ok := h.State.(*ReplState); ok && s != nil {
			currentCF = s.CurrentCF
		}

		// Parse flags and arguments
		flags, args := parseFlags(parts[1:])
		pretty := flags["pretty"] == "true"

		var cf, field, value string

		switch len(args) {
		case 2: // jsonquery <field> <value> (using current CF)
			if currentCF == "" {
				fmt.Println("No current column family set")
				return true
			}
			cf = currentCF
			field = args[0]
			value = args[1]
		case 3: // jsonquery <cf> <field> <value>
			cf = args[0]
			field = args[1]
			value = args[2]
		default:
			fmt.Println("Usage: jsonquery [<cf>] <field> <value> [--pretty]")
			fmt.Println("  Query entries by JSON field value")
			fmt.Println("  Examples:")
			fmt.Println("    jsonquery name \"Alice\"")
			fmt.Println("    jsonquery users name \"Alice\"")
			fmt.Println("    jsonquery products category \"fruit\" --pretty")
			return true
		}

		result, err := h.DB.JSONQueryCF(cf, field, value)
		if err != nil {
			handleError(err, "JSON query", cf)
		} else {
			if len(result) == 0 {
				fmt.Printf("No entries found in '%s' where field '%s' = '%s'\n", cf, field, value)
			} else {
				fmt.Printf("Found %d entries in '%s' where field '%s' = '%s':\n", len(result), cf, field, value)
				for k, v := range result {
					formattedValue := formatValue(v, pretty)
					fmt.Printf("%s: %s\n", util.FormatKey(k), formattedValue)
				}
			}
		}
	case "stats":
		// Parse flags and arguments
		flags, args := parseFlags(parts[1:])
		detailed := flags["detailed"] == "true" || flags["d"] == "true"
		pretty := flags["pretty"] == "true"

		var cf string

		switch len(args) {
		case 0: // stats (show database stats)
			stats, err := h.DB.GetDatabaseStats()
			if err != nil {
				handleError(err, "Get database statistics")
			} else {
				h.formatDatabaseStats(stats, detailed, pretty)
			}
		case 1: // stats <cf> (show specific CF stats)
			cf = args[0]
			stats, err := h.DB.GetCFStats(cf)
			if err != nil {
				handleError(err, "Get column family statistics", cf)
			} else {
				h.formatCFStats(stats, detailed, pretty)
			}
		default:
			fmt.Println("Usage: stats [<cf>] [--detailed] [--pretty]")
			fmt.Println("  Show database or column family statistics")
			fmt.Println("  Examples:")
			fmt.Println("    stats                    # Database overview")
			fmt.Println("    stats users              # Detailed stats for 'users' CF")
			fmt.Println("    stats --detailed         # Detailed database stats")
			fmt.Println("    stats users --detailed   # Detailed stats for 'users' CF")
			fmt.Println("    stats --pretty           # Pretty JSON output")
			return true
		}
	case "search":
		// Get current CF if available
		currentCF := ""
		if s, ok := h.State.(*ReplState); ok && s != nil {
			currentCF = s.CurrentCF
		}

		// Parse flags and arguments
		flags, args := parseFlags(parts[1:])

		var cf string
		var keyPattern, valuePattern string

		// Extract patterns from flags first
		keyPattern = flags["key"]
		valuePattern = flags["value"]

		// Parse command syntax variations
		switch len(args) {
		case 0:
			// Either help request or search with only flags
			if keyPattern == "" && valuePattern == "" {
				// Show help if no patterns provided
				fmt.Println("Usage: search [<cf>] [options]")
				fmt.Println("  Fuzzy search for keys and/or values in column family")
				fmt.Println("")
				fmt.Println("Options:")
				fmt.Println("  --key=<pattern>       Search in key names")
				fmt.Println("  --value=<pattern>     Search in values")
				fmt.Println("  --regex               Use regex patterns (default: wildcard)")
				fmt.Println("  --case-sensitive      Case sensitive search (default: false)")
				fmt.Println("  --limit=N             Limit results (default: 50)")
				fmt.Println("  --after=<key>         Start search after this key (for cursor-based pagination)")
				fmt.Println("  --keys-only           Show only keys, not values")
				fmt.Println("  --tick                Treat keys as .NET tick times and convert to UTC string format")
				fmt.Println("  --pretty              Pretty format JSON values")
				fmt.Println("  --export=<file>       Export results to CSV file")
				fmt.Println("  --export-sep=<sep>    CSV separator (default: ,)")
				fmt.Println("")
				fmt.Println("Pattern Syntax:")
				fmt.Println("  Wildcard: * (any chars), ? (single char)")
				fmt.Println("  Regex: full regex support with --regex flag")
				fmt.Println("")
				fmt.Println("Examples:")
				fmt.Println("  search --key=user:*               # Keys starting with 'user:'")
				fmt.Println("  search users --value=Alice        # Values containing 'Alice' in 'users' CF")
				fmt.Println("  search --key=*product* --value=*widget*  # Both key and value patterns")
				fmt.Println("  search --key=user:[0-9]+ --regex  # Regex: keys matching 'user:' + digits")
				fmt.Println("  search --value=error --limit=10    # First 10 entries with 'error' in value")
				fmt.Println("  search --key=user:* --limit=100    # First 100 keys starting with 'user:'")
				fmt.Println("  search --key=user:* --limit=100 --after=user:1100   # Next page after 'user:1100'")
				fmt.Println("  search --key=* --tick                               # Show all keys as converted .NET tick times")
				fmt.Println("  search --key=admin --export=admins.csv              # Export admin users to CSV")
				fmt.Println("  search --value=error --export=errors.csv --export-sep=\";\"  # Export with semicolon separator")
				return true
			} else {
				// Use current CF when only flags are provided
				if currentCF == "" {
					fmt.Println("No current column family set")
					return true
				}
				cf = currentCF
			}
		case 1:
			// search <cf> or search with --key/--value flags only
			if !strings.HasPrefix(args[0], "--") {
				cf = args[0]
			} else {
				if currentCF == "" {
					fmt.Println("No current column family set")
					return true
				}
				cf = currentCF
			}
		default:
			// search <cf> with additional arguments
			cf = args[0]
		}

		// Validate that at least one pattern is provided
		if keyPattern == "" && valuePattern == "" {
			fmt.Println("Error: Must specify at least --key or --value pattern")
			return true
		}

		// If no CF specified, use current CF
		if cf == "" {
			if currentCF == "" {
				fmt.Println("No current column family set")
				return true
			}
			cf = currentCF
		}

		// Build search options
		opts := db.SearchOptions{
			KeyPattern:    keyPattern,
			ValuePattern:  valuePattern,
			UseRegex:      flags["regex"] == "true",
			CaseSensitive: flags["case-sensitive"] == "true",
			KeysOnly:      flags["keys-only"] == "true",
			Tick:          flags["tick"] == "true",
			Limit:         50, // Default limit
		}

		// Parse limit
		if limitStr, ok := flags["limit"]; ok {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
				opts.Limit = limit
			} else {
				fmt.Println("Invalid limit value")
				return true
			}
		}

		// Check for export option
		if exportFile, ok := flags["export"]; ok && exportFile != "" {
			// Parse export separator
			exportSep := flags["export-sep"]
			if exportSep == "" {
				exportSep = ","
			}

			// Convert separator string (handle escaped characters)
			sep := parseSep(exportSep)

			// Export search results
			err := h.DB.ExportSearchResultsToCSV(cf, exportFile, sep, opts)
			if err != nil {
				handleError(err, "Export search results", cf, exportFile)
			} else {
				fmt.Printf("Search results exported to %s (sep=%q)\n", exportFile, sep)
			}
			return true
		}

		// Execute search
		results, err := h.DB.SearchCF(cf, opts)
		if err != nil {
			handleError(err, "Search", cf)
		} else {
			h.formatSearchResults(results, flags["pretty"] == "true")
		}
	case "keyformat":
		// Parse arguments
		_, args := parseFlags(parts[1:])

		// Get current CF if available
		currentCF := ""
		if s, ok := h.State.(*ReplState); ok && s != nil {
			currentCF = s.CurrentCF
		}

		var cf string

		switch len(args) {
		case 0: // keyformat (show current CF format)
			if currentCF == "" {
				fmt.Println("No current column family set")
				return true
			}
			cf = currentCF
		case 1: // keyformat <cf>
			cf = args[0]
		default:
			fmt.Println("Usage: keyformat [<cf>]")
			fmt.Println("  Show the detected key format for a column family")
			fmt.Println("  Provides examples of how to query binary keys with string inputs")
			return true
		}

		// Get key format information
		keyFormat, description := h.DB.GetKeyFormatInfo(cf)

		fmt.Printf("Column Family: %s\n", cf)
		fmt.Printf("Detected Key Format: %s\n", description)
		fmt.Println()

		// Show examples based on format
		switch keyFormat {
		case util.KeyFormatUint64BE:
			fmt.Println("Examples for querying with string inputs:")
			fmt.Println("  get binary_keys \"123456789\"     # Query uint64 key as decimal")
			fmt.Println("  get binary_keys \"0x75bf1515\"   # Query uint64 key as hex")
			fmt.Println("  prefix binary_keys \"123456\"    # Find keys starting with number")
			fmt.Println("  scan binary_keys \"100\" \"200\"  # Scan range of uint64 keys")
			fmt.Println()
			fmt.Println("Note: All string inputs are automatically converted to 8-byte big-endian format")

		case util.KeyFormatHex:
			fmt.Println("Examples for querying with string inputs:")
			fmt.Println("  get hex_keys \"deadbeef\"         # Query hex key")
			fmt.Println("  get hex_keys \"0xdeadbeef\"       # Query hex key with 0x prefix")
			fmt.Println("  prefix hex_keys \"dead\"          # Find keys starting with hex pattern")
			fmt.Println("  scan hex_keys \"aa\" \"ff\"        # Scan range of hex keys")
			fmt.Println()
			fmt.Println("Note: String inputs are converted to binary from hex representation")

		case util.KeyFormatMixed:
			fmt.Println("Mixed key format detected. Examples:")
			fmt.Println("  get mixed_keys \"123456789\"      # Try as uint64 first")
			fmt.Println("  get mixed_keys \"deadbeef\"       # Try as hex if uint64 fails")
			fmt.Println("  get mixed_keys \"string_key\"     # Fall back to string key")
			fmt.Println()
			fmt.Println("Note: Smart conversion tries multiple formats automatically")

		case util.KeyFormatString:
			fmt.Println("String key format detected - no conversion needed.")
			fmt.Println("Examples:")
			fmt.Println("  get string_keys \"my_key\"        # Direct string key lookup")
			fmt.Println("  prefix string_keys \"user:\"      # String prefix matching")
			fmt.Println()
			fmt.Println("Note: You can disable smart conversion with --smart=false")
		}

		fmt.Println()
		fmt.Println("Smart conversion is enabled by default for get, prefix, and scan commands.")
		fmt.Println("Use --smart=false to disable automatic key format conversion.")

		// Show some actual sample keys if available
		if stats, err := h.DB.GetCFStats(cf); err == nil && len(stats.SampleKeys) > 0 {
			fmt.Println()
			fmt.Printf("Sample keys from '%s':\n", cf)
			for i, key := range stats.SampleKeys {
				if i >= 5 { // Limit to first 5 keys
					fmt.Printf("  ... and %d more\n", len(stats.SampleKeys)-5)
					break
				}
				fmt.Printf("  %s\n", util.FormatKey(key))
			}
		}
	case "help":
		fmt.Println("Available commands:")
		fmt.Println("  usecf <cf>                    - Switch current column family")
		fmt.Println("  get [<cf>] <key> [--pretty] [--smart=true|false]   - Query by key with smart binary conversion")
		fmt.Println("  put [<cf>] <key> <value>      - Insert/Update key-value pair")
		fmt.Println("  prefix [<cf>] <prefix> [--pretty] [--smart=true|false] - Query by key prefix with smart conversion")
		fmt.Println("  scan [<cf>] [start] [end]     - Scan range with options and smart conversion")
		fmt.Println("    Options: --limit=N --reverse --values=no --timestamp --smart=true|false")
		fmt.Println("    Use * as wildcard to scan all entries (e.g., scan * or scan * *)")
		fmt.Println("  last [<cf>] [--pretty]        - Get last key-value pair from CF")
		fmt.Println("  export [<cf>] <file_path>     - Export CF to CSV file")
		fmt.Println("  jpath [<cf>] <key> <jsonpath> [--pretty] - Query JSON value using JSONPath")
		fmt.Println("  jsonquery [<cf>] <field> <value> [--pretty] - Query entries by JSON field value")
		fmt.Println("  stats [<cf>] [--detailed] [--pretty] - Show database/column family statistics")
		fmt.Println("  keyformat [<cf>]              - Show detected key format and conversion examples")
		fmt.Println("  listcf                        - List all column families")
		fmt.Println("  createcf <cf>                 - Create new column family")
		fmt.Println("  dropcf <cf>                   - Drop column family")
		fmt.Println("  search [<cf>] [options]        - Fuzzy search for keys and/or values")
		fmt.Println("  help                          - Show this help message")
		fmt.Println("  exit/quit                     - Exit the CLI")
		fmt.Println("")
		fmt.Println("🔄 Smart Key Conversion:")
		fmt.Println("  The CLI automatically detects binary key formats (uint64, hex) in column families")
		fmt.Println("  When enabled (default), you can query binary keys using string inputs:")
		fmt.Println("    get binary_keys \"123456789\"   # Query uint64 key with decimal string")
		fmt.Println("    get hex_keys \"deadbeef\"       # Query hex key with hex string")
		fmt.Println("  Use 'keyformat [<cf>]' to see detected format and examples")
		fmt.Println("  Add --smart=false to any command to disable conversion")
		fmt.Println("")
		fmt.Println("🤖 AI-Powered Features:")
		fmt.Println("  For natural language queries, restart with:")
		fmt.Println("  rocksdb-cli --db <path> --graphchain")
		fmt.Println("  Examples: 'Show me all users', 'Find products with category electronics'")
		fmt.Println("")
		fmt.Println("🔧 MCP Server Integration:")
		fmt.Println("  Start MCP server for Claude Desktop integration:")
		fmt.Println("  ./cmd/mcp-server/rocksdb-mcp-server --db <path>")
		fmt.Println("  Configure in Claude Desktop's claude_desktop_config.json")
		fmt.Println("")
		fmt.Println("Notes:")
		fmt.Println("  - Commands without [<cf>] use current column family")

		// Check if we're in read-only mode and show appropriate message
		if h.DB.IsReadOnly() {
			fmt.Println("  - Database is in READ-ONLY mode")
			fmt.Println("  - Write operations (put, createcf, dropcf) are disabled")
		} else {
			fmt.Println("  - Current column family is shown in prompt: rocksdb[current_cf]>")
		}

		fmt.Println("  - jpath queries JSON values using JSONPath expressions ($.field, $.items[0], etc.)")
		fmt.Println("    Example: jpath user:123 \"$.profile.email\" extracts email from user profile")
		fmt.Println("  - jsonquery searches for JSON values where field matches value exactly")
		fmt.Println("    Example: jsonquery name \"Alice\" finds all entries where name field = \"Alice\"")
		fmt.Println("  - stats command shows key count, data types, size distribution, and common patterns")
		fmt.Println("  - search command supports wildcard (*,?) and regex patterns with --regex flag")
		fmt.Println("")
		fmt.Println("Advanced Features:")
		fmt.Println("  📊 Real-time monitoring: rocksdb-cli --db <path> --watch <cf>")
		fmt.Println("  📄 CSV export: rocksdb-cli --db <path> --export-cf <cf> --export-file file.csv")
		fmt.Println("  🔍 Complex search: search --key=user:* --value=admin --regex --limit=10")
		fmt.Println("  🎯 Range scanning: scan users user:1000 user:2000 --limit=50 --reverse")
		fmt.Println("")
		fmt.Println("For comprehensive documentation visit: https://github.com/rocksdb-cli")
	case "exit", "quit":
		return false
	default:
		fmt.Println("Unknown command. Type 'help' for available commands.")
	}
	return true
}

// formatDatabaseStats formats and displays database-wide statistics
func (h *Handler) formatDatabaseStats(stats *db.DatabaseStats, detailed, pretty bool) {
	if pretty {
		// Output as pretty JSON
		if data, err := json.MarshalIndent(stats, "", "  "); err == nil {
			fmt.Println(string(data))
		} else {
			fmt.Printf("Error formatting stats: %v\n", err)
		}
		return
	}

	fmt.Println("=== Database Statistics ===")
	fmt.Printf("Column Families: %d\n", stats.ColumnFamilyCount)
	fmt.Printf("Total Keys: %s\n", formatNumber(stats.TotalKeyCount))
	fmt.Printf("Total Size: %s\n", formatBytes(stats.TotalSize))
	fmt.Printf("Last Updated: %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05"))
	fmt.Println()

	if detailed {
		fmt.Println("=== Column Family Details ===")
		for _, cf := range stats.ColumnFamilies {
			h.formatCFStats(&cf, false, false)
			fmt.Println()
		}
	} else {
		fmt.Println("Column Family Summary:")
		for _, cf := range stats.ColumnFamilies {
			fmt.Printf("  %-20s %8s keys  %10s\n",
				cf.Name,
				formatNumber(cf.KeyCount),
				formatBytes(cf.TotalKeySize+cf.TotalValueSize))
		}
	}
}

// formatCFStats formats and displays column family statistics
func (h *Handler) formatCFStats(stats *db.CFStats, detailed, pretty bool) {
	if pretty {
		// Output as pretty JSON
		if data, err := json.MarshalIndent(stats, "", "  "); err == nil {
			fmt.Println(string(data))
		} else {
			fmt.Printf("Error formatting stats: %v\n", err)
		}
		return
	}

	fmt.Printf("=== Column Family: %s ===\n", stats.Name)
	fmt.Printf("Keys: %s\n", formatNumber(stats.KeyCount))
	fmt.Printf("Total Key Size: %s\n", formatBytes(stats.TotalKeySize))
	fmt.Printf("Total Value Size: %s\n", formatBytes(stats.TotalValueSize))
	fmt.Printf("Average Key Size: %.1f bytes\n", stats.AverageKeySize)
	fmt.Printf("Average Value Size: %.1f bytes\n", stats.AverageValueSize)
	fmt.Printf("Last Updated: %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05"))

	if detailed || len(stats.DataTypeDistribution) > 0 {
		fmt.Println("\nData Type Distribution:")
		// Sort data types by count (descending)
		type dataTypeCount struct {
			dataType db.DataType
			count    int64
		}
		var sortedTypes []dataTypeCount
		for dt, count := range stats.DataTypeDistribution {
			sortedTypes = append(sortedTypes, dataTypeCount{dt, count})
		}
		sort.Slice(sortedTypes, func(i, j int) bool {
			return sortedTypes[i].count > sortedTypes[j].count
		})

		for _, dtc := range sortedTypes {
			percentage := float64(dtc.count) / float64(stats.KeyCount) * 100
			fmt.Printf("  %-12s %8s (%5.1f%%)\n",
				dtc.dataType,
				formatNumber(dtc.count),
				percentage)
		}
	}

	if detailed {
		if len(stats.CommonPrefixes) > 0 {
			fmt.Println("\nCommon Key Prefixes:")
			// Sort prefixes by count (descending)
			type prefixCount struct {
				prefix string
				count  int64
			}
			var sortedPrefixes []prefixCount
			for prefix, count := range stats.CommonPrefixes {
				sortedPrefixes = append(sortedPrefixes, prefixCount{prefix, count})
			}
			sort.Slice(sortedPrefixes, func(i, j int) bool {
				return sortedPrefixes[i].count > sortedPrefixes[j].count
			})

			// Show top 10 prefixes
			limit := 10
			if len(sortedPrefixes) < limit {
				limit = len(sortedPrefixes)
			}
			for i := 0; i < limit; i++ {
				pc := sortedPrefixes[i]
				percentage := float64(pc.count) / float64(stats.KeyCount) * 100
				fmt.Printf("  %-15s %8s (%5.1f%%)\n",
					pc.prefix,
					formatNumber(pc.count),
					percentage)
			}
		}

		if len(stats.SampleKeys) > 0 {
			fmt.Println("\nSample Keys:")
			for i, key := range stats.SampleKeys {
				if i >= 5 { // Limit to first 5 keys
					fmt.Printf("  ... and %d more\n", len(stats.SampleKeys)-5)
					break
				}
				fmt.Printf("  %s\n", util.FormatKey(key))
			}
		}
	}
}

// formatNumber formats large numbers with K/M/B suffixes
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	} else if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	} else if n < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	} else {
		return fmt.Sprintf("%.1fB", float64(n)/1000000000)
	}
}

// formatBytes formats byte counts with appropriate units
func formatBytes(bytes int64) string {
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

// formatSearchResults formats and displays search results
func (h *Handler) formatSearchResults(results *db.SearchResults, pretty bool) {
	if len(results.Results) == 0 {
		fmt.Println("No matches found")
		fmt.Printf("Query took: %s\n", results.QueryTime)
		return
	}

	// Show header with result count and timing
	limitedText := ""
	if results.Limited {
		limitedText = " (limited)"
	}
	fmt.Printf("Found %d matches%s in %s\n", results.Total, limitedText, results.QueryTime)
	fmt.Println()

	// Display results
	for i, result := range results.Results {
		// Show result number
		fmt.Printf("[%d] Key: %s", i+1, util.FormatKey(result.Key))

		// Show which fields matched
		if len(result.MatchedFields) > 0 {
			fmt.Printf(" (matched: %s)", strings.Join(result.MatchedFields, ", "))
		}
		fmt.Println()

		// Show value if not keys-only
		if result.Value != "" {
			valueToDisplay := result.Value
			if pretty {
				valueToDisplay = formatValue(result.Value, true)
			}
			// Indent value for better readability
			valueLines := strings.Split(valueToDisplay, "\n")
			for _, line := range valueLines {
				fmt.Printf("    %s\n", line)
			}
		}

		// Add separator between results (except for last one)
		if i < len(results.Results)-1 {
			fmt.Println()
		}
	}

	// Show footer with timing
	fmt.Printf("\nQuery completed in %s\n", results.QueryTime)
}

// ExecuteGraphChainCommand executes the graphchain agent command
func (h *Handler) ExecuteGraphChainCommand(input string) bool {
	trimmed := strings.TrimSpace(input)

	// Handle 'memory' command
	if trimmed == "memory" {
		h.showMemoryHelp()
		return true
	}

	// Handle memory subcommands
	if strings.HasPrefix(trimmed, "memory ") {
		parts := strings.Fields(trimmed)
		if len(parts) >= 2 {
			switch parts[1] {
			case "stats":
				h.showMemoryStats()
				return true
			case "history":
				n := 5 // default to last 5 turns
				if len(parts) >= 3 {
					if parsed, err := strconv.Atoi(parts[2]); err == nil && parsed > 0 {
						n = parsed
					}
				}
				h.showMemoryHistory(n)
				return true
			case "clear":
				h.clearMemory()
				return true
			default:
				fmt.Println("Unknown memory command. Use 'memory' for help.")
				return true
			}
		}
	}

	// Handle other graphchain commands here
	// For now, return false to indicate the command wasn't handled
	return false
}

// showMemoryHelp displays help for memory commands
func (h *Handler) showMemoryHelp() {
	fmt.Println("\n🧠 Memory Commands:")
	fmt.Println("  memory           - Show this help")
	fmt.Println("  memory stats     - Show memory usage statistics")
	fmt.Println("  memory history [n] - Show last n conversation turns (default: 5)")
	fmt.Println("  memory clear     - Clear conversation history")
	fmt.Println()
}

// showMemoryStats displays memory usage statistics
func (h *Handler) showMemoryStats() {
	if h.GraphChainAgent == nil {
		fmt.Println("❌ GraphChain agent not initialized")
		return
	}

	if !h.GraphChainAgent.IsMemoryEnabled() {
		fmt.Println("🚫 Memory is not enabled. Enable it in the config file.")
		return
	}

	stats := h.GraphChainAgent.GetMemoryStats()
	if stats == nil {
		fmt.Println("❌ Could not retrieve memory statistics")
		return
	}

	fmt.Printf("\n📊 Memory Statistics:\n")
	fmt.Printf("  Total Conversations: %d\n", stats.TotalTurns)
	fmt.Printf("  Memory Capacity: %d\n", stats.MaxSize)
	fmt.Printf("  Memory Usage: %.1f%%\n", stats.MemoryUsage)
	fmt.Printf("  Total Characters: %d\n", stats.TotalChars)

	if stats.TotalTurns > 0 {
		fmt.Printf("  Oldest Conversation: %s\n", stats.OldestTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Newest Conversation: %s\n", stats.NewestTime.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()
}

// showMemoryHistory displays recent conversation history
func (h *Handler) showMemoryHistory(n int) {
	if h.GraphChainAgent == nil {
		fmt.Println("❌ GraphChain agent not initialized")
		return
	}

	if !h.GraphChainAgent.IsMemoryEnabled() {
		fmt.Println("🚫 Memory is not enabled. Enable it in the config file.")
		return
	}

	history := h.GraphChainAgent.GetConversationHistory(n)
	if len(history) == 0 {
		fmt.Println("📝 No conversation history available")
		return
	}

	fmt.Printf("\n📚 Last %d Conversation(s):\n", len(history))
	fmt.Println(strings.Repeat("─", 60))

	for i, turn := range history {
		fmt.Printf("\n💬 Conversation %d (%s):\n", len(history)-i, turn.Timestamp.Format("15:04:05"))

		if turn.UserQuery != "" {
			fmt.Printf("🙋 User: %s\n", turn.UserQuery)
		}

		if turn.AgentResponse != "" {
			// Truncate long responses for display
			response := turn.AgentResponse
			if len(response) > 200 {
				response = response[:200] + "..."
			}
			fmt.Printf("🤖 Agent: %s\n", response)
		}

		if turn.ExecutionTime > 0 {
			fmt.Printf("⏱️  Time: %v\n", turn.ExecutionTime)
		}
	}
	fmt.Println()
}

// clearMemory clears the conversation history
func (h *Handler) clearMemory() {
	if h.GraphChainAgent == nil {
		fmt.Println("❌ GraphChain agent not initialized")
		return
	}

	if !h.GraphChainAgent.IsMemoryEnabled() {
		fmt.Println("🚫 Memory is not enabled. Enable it in the config file.")
		return
	}

	err := h.GraphChainAgent.ClearMemory(context.Background())
	if err != nil {
		fmt.Printf("❌ Failed to clear memory: %v\n", err)
		return
	}

	fmt.Println("✅ Conversation memory cleared successfully")
}
