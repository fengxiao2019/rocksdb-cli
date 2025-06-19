package command

import (
	"fmt"
	"rocksdb-cli/internal/db"
	"strings"
)

type ReplState struct {
	CurrentCF string
}

type Handler struct {
	DB    db.KeyValueDB
	State interface{} // *ReplState, used to manage the active column family
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
	case "get":
		cf, key := "", ""
		if len(parts) == 2 && h.State != nil {
			if s, ok := h.State.(*ReplState); ok {
				cf = s.CurrentCF
				key = parts[1]
			}
		} else if len(parts) == 3 {
			cf = parts[1]
			key = parts[2]
		} else {
			fmt.Println("Usage: get [<cf>] <key>")
			return true
		}
		val, err := h.DB.GetCF(cf, key)
		if err != nil {
			fmt.Printf("Query failed: %v\n", err)
		} else {
			fmt.Printf("%s\n", val)
		}
	case "put":
		cf, key, value := "", "", ""
		if len(parts) >= 3 && h.State != nil {
			if s, ok := h.State.(*ReplState); ok {
				cf = s.CurrentCF
				key = parts[1]
				value = strings.Join(parts[2:], " ")
			}
		} else if len(parts) >= 4 {
			cf = parts[1]
			key = parts[2]
			value = strings.Join(parts[3:], " ")
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
	case "exit", "quit":
		return false
	default:
		fmt.Println("Unknown command")
	}
	return true
}
