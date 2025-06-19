package repl

import (
	"fmt"
	"rocksdb-cli/internal/command"
	"rocksdb-cli/internal/db"

	prompt "github.com/c-bata/go-prompt"
)

func Start(rdb db.KeyValueDB) {
	state := &command.ReplState{CurrentCF: "default"}
	handler := &command.Handler{DB: rdb, State: state}
	fmt.Println("Welcome to rocksdb-cli with column family support. Type 'help' for commands, 'exit/quit' to quit.")
	p := prompt.New(
		func(in string) {
			if !handler.Execute(in) {
				fmt.Println("Bye.")
				// Exit REPL
				panic("exit")
			}
		},
		completer,
		prompt.OptionLivePrefix(func() (string, bool) {
			return fmt.Sprintf("rocksdb[%s]> ", state.CurrentCF), true
		}),
	)
	defer func() {
		if r := recover(); r != nil && r != "exit" {
			panic(r)
		}
	}()
	p.Run()
}

func completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "usecf", Description: "usecf <cf> Switch current column family"},
		{Text: "get", Description: "get [<cf>] <key> Query by CF"},
		{Text: "put", Description: "put [<cf>] <key> <value> Insert/Update"},
		{Text: "prefix", Description: "prefix [<cf>] <prefix> Query by CF prefix"},
		{Text: "scan", Description: "scan [<cf>] [start] [end] Scan range with options"},
		{Text: "listcf", Description: "List all column families"},
		{Text: "createcf", Description: "createcf <cf> Create new column family"},
		{Text: "dropcf", Description: "dropcf <cf> Drop column family"},
		{Text: "exit", Description: "Exit"},
		{Text: "quit", Description: "Exit"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}
