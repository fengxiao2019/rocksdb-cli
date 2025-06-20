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
	exportCF := flag.String("export-cf", "", "Column family to export")
	exportFile := flag.String("export-file", "", "Output CSV file path")
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

	// If export parameters are provided, perform export and exit
	if *exportCF != "" && *exportFile != "" {
		err := rdb.ExportToCSV(*exportCF, *exportFile)
		if err != nil {
			fmt.Printf("Export failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully exported column family '%s' to '%s'\n", *exportCF, *exportFile)
		return
	}

	// If only one export parameter is provided, show usage
	if *exportCF != "" || *exportFile != "" {
		fmt.Println("Both --export-cf and --export-file must be specified for export")
		fmt.Println("Usage: rocksdb-cli --db <path> --export-cf <cf> --export-file <file.csv>")
		os.Exit(1)
	}

	// Start interactive REPL if no export parameters
	repl.Start(rdb)
}
