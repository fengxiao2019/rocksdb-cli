package command

import (
	"encoding/json"
	"fmt"
	"rocksdb-cli/internal/db"
	"strconv"
	"strings"
)

type ReplState struct {
	CurrentCF string
}

type Handler struct {
	DB    db.KeyValueDB
	State interface{} // *ReplState, used to manage the active column family
}

func prettyPrintJSON(val string) string {
	var jsonData interface{}
	if err := json.Unmarshal([]byte(val), &jsonData); err != nil {
		return val // If not valid JSON, return as is
	}
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return val // If can't pretty print, return as is
	}
	return string(prettyJSON)
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
		var start, end []byte

		// Parse column family and range
		switch len(args) {
		case 1: // scan <start>
			if s, ok := h.State.(*ReplState); ok && s != nil {
				cf = s.CurrentCF
				start = []byte(args[0])
			} else {
				fmt.Println("No current column family set")
				return true
			}
		case 2: // scan <start> <end> or scan <cf> <start>
			if s, ok := h.State.(*ReplState); ok && s != nil {
				cf = s.CurrentCF
				start = []byte(args[0])
				end = []byte(args[1])
			} else {
				cf = args[0]
				start = []byte(args[1])
			}
		case 3: // scan <cf> <start> <end>
			cf = args[0]
			start = []byte(args[1])
			end = []byte(args[2])
		default:
			fmt.Println("Usage: scan [<cf>] [start] [end] [--limit=N] [--reverse] [--values=no]")
			return true
		}

		// Parse options
		opts := db.ScanOptions{
			Values: true, // Default to showing values
		}

		if limit, ok := flags["limit"]; ok {
			if n, err := strconv.Atoi(limit); err == nil && n > 0 {
				opts.Limit = n
			} else {
				fmt.Println("Invalid limit value")
				return true
			}
		}

		if _, ok := flags["reverse"]; ok {
			opts.Reverse = true
		}

		if val, ok := flags["values"]; ok && val == "no" {
			opts.Values = false
		}

		result, err := h.DB.ScanCF(cf, start, end, opts)
		if err != nil {
			fmt.Printf("Scan failed: %v\n", err)
		} else {
			for k, v := range result {
				if opts.Values {
					fmt.Printf("%s: %s\n", k, v)
				} else {
					fmt.Printf("%s\n", k)
				}
			}
		}
	case "get":
		cf, key, pretty := "", "", false
		switch len(parts) {
		case 2: // get <key>
			if s, ok := h.State.(*ReplState); ok && s != nil {
				cf = s.CurrentCF
				key = parts[1]
			} else {
				fmt.Println("No current column family set")
				return true
			}
		case 3:
			if parts[2] == "--pretty" { // get <key> --pretty
				if s, ok := h.State.(*ReplState); ok && s != nil {
					cf = s.CurrentCF
					key = parts[1]
					pretty = true
				} else {
					fmt.Println("No current column family set")
					return true
				}
			} else { // get <cf> <key>
				cf = parts[1]
				key = parts[2]
			}
		case 4: // get <cf> <key> --pretty
			if parts[3] != "--pretty" {
				fmt.Println("Usage: get [<cf>] <key> [--pretty]")
				return true
			}
			cf = parts[1]
			key = parts[2]
			pretty = true
		default:
			fmt.Println("Usage: get [<cf>] <key> [--pretty]")
			return true
		}
		val, err := h.DB.GetCF(cf, key)
		if err != nil {
			fmt.Printf("Query failed: %v\n", err)
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
			fmt.Printf("Write failed: %v\n", err)
		} else {
			fmt.Println("OK")
		}
	case "prefix":
		cf, prefix := "", ""
		if len(parts) == 2 && h.State != nil {
			if s, ok := h.State.(*ReplState); ok {
				cf = s.CurrentCF
				prefix = parts[1]
			}
		} else if len(parts) == 3 {
			cf = parts[1]
			prefix = parts[2]
		} else {
			fmt.Println("Usage: prefix [<cf>] <prefix>")
			return true
		}
		result, err := h.DB.PrefixScanCF(cf, prefix, 20)
		if err != nil {
			fmt.Printf("Query failed: %v\n", err)
		} else {
			for k, v := range result {
				fmt.Printf("%s: %s\n", k, v)
			}
		}
	case "listcf":
		cfs, err := h.DB.ListCFs()
		if err != nil {
			fmt.Printf("List CF failed: %v\n", err)
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
			fmt.Printf("Create CF failed: %v\n", err)
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
			fmt.Printf("Drop CF failed: %v\n", err)
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
			fmt.Println("Usage: export [<cf>] <file_path>")
			fmt.Println("  Export column family data to CSV file")
			return true
		}

		err := h.DB.ExportToCSV(cf, filePath)
		if err != nil {
			fmt.Printf("Export failed: %v\n", err)
		} else {
			fmt.Printf("Successfully exported column family '%s' to '%s'\n", cf, filePath)
		}
	case "last":
		cf := ""
		if len(parts) == 1 && h.State != nil {
			// last - use current CF
			if s, ok := h.State.(*ReplState); ok && s != nil {
				cf = s.CurrentCF
			} else {
				fmt.Println("No current column family set")
				return true
			}
		} else if len(parts) == 2 {
			// last <cf>
			cf = parts[1]
		} else {
			fmt.Println("Usage: last [<cf>]")
			fmt.Println("  Get the last key-value pair from column family")
			return true
		}

		key, value, err := h.DB.GetLastCF(cf)
		if err != nil {
			fmt.Printf("Get last failed: %v\n", err)
		} else {
			fmt.Printf("Last entry in '%s': %s = %s\n", cf, key, value)
		}
	case "help":
		fmt.Println("Available commands:")
		fmt.Println("  usecf <cf>                    - Switch current column family")
		fmt.Println("  get [<cf>] <key> [--pretty]   - Query by key (use --pretty for JSON formatting)")
		fmt.Println("  put [<cf>] <key> <value>      - Insert/Update key-value pair")
		fmt.Println("  prefix [<cf>] <prefix>        - Query by key prefix")
		fmt.Println("  scan [<cf>] [start] [end]     - Scan range with options")
		fmt.Println("    Options: --limit=N --reverse --values=no")
		fmt.Println("  last [<cf>]                   - Get last key-value pair from CF")
		fmt.Println("  export [<cf>] <file_path>     - Export CF to CSV file")
		fmt.Println("  listcf                        - List all column families")
		fmt.Println("  createcf <cf>                 - Create new column family")
		fmt.Println("  dropcf <cf>                   - Drop column family")
		fmt.Println("  help                          - Show this help message")
		fmt.Println("  exit/quit                     - Exit the CLI")
		fmt.Println("")
		fmt.Println("Notes:")
		fmt.Println("  - Commands without [<cf>] use current column family")
		fmt.Println("  - Current column family is shown in prompt: rocksdb[current_cf]>")
	case "exit", "quit":
		return false
	default:
		fmt.Println("Unknown command. Type 'help' for available commands.")
	}
	return true
}
