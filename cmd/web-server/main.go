package main

import (
	"flag"
	"fmt"
	"log"

	"rocksdb-cli/internal/api"
	"rocksdb-cli/internal/service"
)

var (
	dbPath   string
	port     string
	readOnly bool
	webUI    bool
)

func init() {
	flag.StringVar(&dbPath, "db", "", "Path to RocksDB database (required)")
	flag.StringVar(&port, "port", "8080", "Port to listen on")
	flag.BoolVar(&readOnly, "readonly", true, "Open database in read-only mode (recommended)")
	flag.BoolVar(&webUI, "ui", true, "Enable Web UI with dynamic database selection")
}

func main() {
	flag.Parse()

	// Require database path
	if dbPath == "" {
		log.Fatal("Error: --db flag is required. Please specify a database path.\nUsage: web-server --db /path/to/database")
	}

	// Create DBManager for dynamic database management
	dbManager := service.NewDBManager()

	// Auto-connect to the specified database
	fmt.Printf("Connecting to database: %s (read-only mode enforced)\n", dbPath)
	if _, err := dbManager.Connect(dbPath); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	fmt.Printf("✅ Database connected successfully\n")

	// Setup router with UI support
	router := api.SetupRouterWithUI(dbManager)

	// Start server
	addr := ":" + port
	fmt.Printf("\n🚀 RocksDB Web Server starting on http://localhost%s\n", addr)
	fmt.Printf("   Database: %s\n", dbPath)
	fmt.Printf("   Read-only: %v\n", readOnly)
	fmt.Printf("\n📋 API Endpoints:\n")
	fmt.Printf("   GET  /                              - Web UI (if enabled)\n")
	fmt.Printf("   GET  /api/v1/health                 - Health check\n")
	fmt.Printf("   GET  /api/v1/databases/list         - List available databases\n")
	fmt.Printf("   POST /api/v1/databases/connect      - Connect to database\n")
	fmt.Printf("   POST /api/v1/databases/disconnect   - Disconnect from database\n")
	fmt.Printf("   GET  /api/v1/databases/current      - Get current database info\n")
	fmt.Printf("   GET  /api/v1/cf                     - List column families\n")
	fmt.Printf("   GET  /api/v1/stats                  - Database statistics\n")
	fmt.Printf("   GET  /api/v1/cf/:cf/get/:key        - Get value by key\n")
	fmt.Printf("   POST /api/v1/cf/:cf/put             - Put key-value pair\n")
	fmt.Printf("   POST /api/v1/cf/:cf/scan            - Scan entries\n")
	fmt.Printf("\n💡 Open in browser: http://localhost%s\n\n", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
