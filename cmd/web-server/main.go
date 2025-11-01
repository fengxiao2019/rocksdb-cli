package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"rocksdb-cli/internal/api"
	"rocksdb-cli/internal/db"
)

var (
	dbPath   string
	port     string
	readOnly bool
)

func init() {
	flag.StringVar(&dbPath, "db", "", "Path to RocksDB database (required)")
	flag.StringVar(&port, "port", "8080", "Port to listen on")
	flag.BoolVar(&readOnly, "readonly", true, "Open database in read-only mode (recommended)")
}

func main() {
	flag.Parse()

	// Validate required flags
	if dbPath == "" {
		fmt.Println("Error: --db flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Open database
	fmt.Printf("Opening database: %s (read-only: %v)\n", dbPath, readOnly)
	var rdb db.KeyValueDB
	var err error

	if readOnly {
		rdb, err = db.OpenReadOnly(dbPath)
	} else {
		rdb, err = db.Open(dbPath)
	}

	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer rdb.Close()

	fmt.Printf("✅ Database opened successfully\n")

	// Setup router
	router := api.SetupRouter(rdb)

	// Start server
	addr := ":" + port
	fmt.Printf("\n🚀 RocksDB Web API Server starting on http://localhost%s\n", addr)
	fmt.Printf("   Database: %s\n", dbPath)
	fmt.Printf("   Read-only: %v\n", readOnly)
	fmt.Printf("\n📋 API Endpoints:\n")
	fmt.Printf("   GET  /                         - API info\n")
	fmt.Printf("   GET  /api/v1/health            - Health check\n")
	fmt.Printf("   GET  /api/v1/cf                - List column families\n")
	fmt.Printf("   GET  /api/v1/stats             - Database statistics\n")
	fmt.Printf("   GET  /api/v1/cf/:cf/get/:key   - Get value by key\n")
	fmt.Printf("   POST /api/v1/cf/:cf/put        - Put key-value pair\n")
	fmt.Printf("   POST /api/v1/cf/:cf/scan       - Scan entries\n")
	fmt.Printf("   POST /api/v1/cf/:cf/prefix     - Prefix scan\n")
	fmt.Printf("   POST /api/v1/cf/:cf/search     - Advanced search\n")
	fmt.Printf("   POST /api/v1/cf/:cf/jsonquery  - JSON query\n")
	fmt.Printf("\n💡 Try: curl http://localhost%s/api/v1/health\n\n", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
