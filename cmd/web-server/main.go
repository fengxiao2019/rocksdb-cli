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
	flag.StringVar(&dbPath, "db", "", "Path to RocksDB database (optional for web UI mode)")
	flag.StringVar(&port, "port", "8080", "Port to listen on")
	flag.BoolVar(&readOnly, "readonly", true, "Open database in read-only mode (recommended)")
	flag.BoolVar(&webUI, "ui", true, "Enable Web UI with dynamic database selection")
}

func main() {
	flag.Parse()

	// Create DBManager for dynamic database management
	dbManager := service.NewDBManager()

	// If dbPath is provided, auto-connect to it
	if dbPath != "" {
		fmt.Printf("Auto-connecting to database: %s (read-only mode enforced)\n", dbPath)
		if _, err := dbManager.Connect(dbPath); err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		fmt.Printf("✅ Database connected successfully\n")
	} else {
		fmt.Printf("🌐 Starting in Web UI mode - database selection via UI\n")
	}

	// Setup router with UI support
	router := api.SetupRouterWithUI(dbManager)

	// Start server
	addr := ":" + port
	fmt.Printf("\n🚀 RocksDB Web Server starting on http://localhost%s\n", addr)
	if dbPath != "" {
		fmt.Printf("   Database: %s\n", dbPath)
		fmt.Printf("   Read-only: %v\n", readOnly)
	} else {
		fmt.Printf("   Mode: Web UI - Select database in browser\n")
	}
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
