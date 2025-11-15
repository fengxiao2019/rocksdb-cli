// Package repl provides an interactive REPL (Read-Eval-Print Loop) for RocksDB CLI.
//
// This package handles Windows-specific issues related to signal handling and channel closures.
// On Windows, the go-prompt library and multiple signal handlers can cause race conditions
// leading to "close of closed channel" panics. The exit handling has been made thread-safe
// and uses os.Exit() instead of panic() to avoid conflicts with go-prompt's internal
// signal handling.
package repl

import (
	"fmt"
	"os"
	"os/exec"
	"rocksdb-cli/internal/command"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/util"
	"runtime"
	"sync"

	prompt "github.com/c-bata/go-prompt"
)

var (
	// Global flag to track if we're in the exit process
	exiting   = false
	exitMutex sync.Mutex
)

func Start(rdb db.KeyValueDB) {
	// Enable color highlighting for interactive mode
	util.EnableColor()

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

	// Create exit handler function
	exitHandler := func() {
		exitMutex.Lock()
		if exiting {
			exitMutex.Unlock()
			return
		}
		exiting = true
		exitMutex.Unlock()

		fmt.Println("Bye.")
		restoreTerminal()
		os.Exit(0)
	}

	p := prompt.New(
		func(in string) {
			if !handler.Execute(in) {
				exitHandler()
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
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn: func(buf *prompt.Buffer) {
				exitHandler()
			},
		}),
	)

	// Ensure terminal is always restored on exit
	defer restoreTerminal()

	// Set up safer panic recovery that doesn't interfere with signal handlers
	defer func() {
		if r := recover(); r != nil {
			// Only handle our own panics, let others propagate
			if r == "exit" {
				// Clean exit - already handled above
				return
			}
			// Re-panic for other panics
			panic(r)
		}
	}()

	p.Run()
}

// isWSL checks if we're running in Windows Subsystem for Linux
func isWSL() bool {
	return os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSLENV") != ""
}

// isWindows checks if we're running on Windows
func isWindows() bool {
	return runtime.GOOS == "windows"
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

// restoreTerminal restores terminal to normal state (echo enabled, cursor visible)
func restoreTerminal() {
	// Restore echo
	cmd := exec.Command("stty", "echo")
	_ = cmd.Run()
	
	// Show cursor and reset attributes
	fmt.Print("\033[?25h") // Show cursor
	fmt.Print("\033[0m")   // Reset attributes
}

func completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}
