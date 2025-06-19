package main

import (
	"flag"
	"fmt"
	"os"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/repl"
)

func main() {
	dbPath := flag.String("db", "", "Path to RocksDB database")
	flag.Parse()
	if *dbPath == "" {
		fmt.Println("Please specify --db parameter pointing to RocksDB path")
		os.Exit(1)
	}
	rdb, err := db.Open(*dbPath)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer rdb.Close()
	repl.Start(rdb)
}
