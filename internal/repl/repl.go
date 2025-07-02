package repl

import (
	"fmt"
	"os"
	"os/exec"
	"rocksdb-cli/internal/command"
	"rocksdb-cli/internal/db"

	prompt "github.com/c-bata/go-prompt"
)

func Start(rdb db.KeyValueDB) {
	state := &command.ReplState{CurrentCF: "default"}
	handler := &command.Handler{DB: rdb, State: state}

	if rdb.IsReadOnly() {
		fmt.Println("Welcome to rocksdb-cli with column family support (READ-ONLY MODE).")
		fmt.Println("Type 'help' for available commands, 'exit' or 'quit' to exit.")
		fmt.Println("Note: Write operations are disabled in read-only mode.")
	} else {
		fmt.Println("Welcome to rocksdb-cli with column family support.")
		fmt.Println("Type 'help' for available commands, 'exit' or 'quit' to exit.")
	}

	p := prompt.New(
		func(in string) {
			if !handler.Execute(in) {
				fmt.Println("Bye.")
				// Only fix terminal on WSL
				if isWSL() {
					fixWSLTerminal()
				}
				// Exit REPL
				panic("exit")
			}
		},
		completer,
		prompt.OptionLivePrefix(func() (string, bool) {
			readOnlyFlag := ""
			if rdb.IsReadOnly() {
				readOnlyFlag = "[READ-ONLY]"
			}
			return fmt.Sprintf("rocksdb%s[%s]> ", readOnlyFlag, state.CurrentCF), true
		}),
	)
	defer func() {
		if r := recover(); r != nil && r != "exit" {
			panic(r)
		}
	}()
	p.Run()
}

// isWSL checks if we're running in Windows Subsystem for Linux
func isWSL() bool {
	return os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSLENV") != ""
}

// fixWSLTerminal restores terminal input visibility for WSL
func fixWSLTerminal() {
	// Method 1: Use reset command (most effective for WSL)
	cmd := exec.Command("reset")
	_ = cmd.Run()

	// Method 2: Ensure echo is enabled
	cmd = exec.Command("stty", "echo")
	_ = cmd.Run()

	// Method 3: Send terminal escape sequence to restore echo
	fmt.Print("\033[?25h") // Show cursor
	fmt.Print("\033[0m")   // Reset attributes
}

func completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "usecf", Description: "usecf <cf> Switch current column family"},
		{Text: "get", Description: "get [<cf>] <key> [--pretty] Query by key (--pretty for JSON)"},
		{Text: "put", Description: "put [<cf>] <key> <value> Insert/Update (disabled in read-only)"},
		{Text: "prefix", Description: "prefix [<cf>] <prefix> Query by key prefix"},
		{Text: "scan", Description: "scan [<cf>] [start] [end] [--limit=N] [--reverse] [--values=no]"},
		{Text: "last", Description: "last [<cf>] [--pretty] Get last key-value pair from CF"},
		{Text: "export", Description: "export [<cf>] <file_path> Export CF to CSV file"},
		{Text: "jsonquery", Description: "jsonquery [<cf>] <field> <value> [--pretty] Query by JSON field value"},
		{Text: "stats", Description: "stats [<cf>] [--detailed] [--pretty] Show database/CF statistics"},
		{Text: "listcf", Description: "List all column families"},
		{Text: "createcf", Description: "createcf <cf> Create new column family (disabled in read-only)"},
		{Text: "dropcf", Description: "dropcf <cf> Drop column family (disabled in read-only)"},
		{Text: "help", Description: "Show help with all available commands"},
		{Text: "exit", Description: "Exit"},
		{Text: "quit", Description: "Exit"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}
