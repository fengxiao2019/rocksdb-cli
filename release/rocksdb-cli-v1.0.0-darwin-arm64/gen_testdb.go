package main

import (
	"fmt"
	"os"

	"github.com/linxGnu/grocksdb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gen_testdb <db_path>")
		os.Exit(1)
	}
	dbPath := os.Args[1]

	// Create DB options
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)

	// Define column families
	cfNames := []string{"default", "users", "products", "logs"}
	cfOpts := make([]*grocksdb.Options, len(cfNames))
	for i := range cfOpts {
		cfOpts[i] = grocksdb.NewDefaultOptions()
	}

	// Open DB with column families
	db, cfHandles, err := grocksdb.OpenDbColumnFamilies(opts, dbPath, cfNames, cfOpts)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	defer func() {
		for _, handle := range cfHandles {
			handle.Destroy()
		}
	}()

	// Create write options
	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()

	// Test data for different column families
	testData := map[string]map[string]string{
		"default": {
			"key001":   "value001",
			"key002":   "value002",
			"key010":   "value010",
			"key020":   "value020",
			"key100":   "value100",
			"app_name": "rocksdb-cli",
			"version":  "1.0.0",
			"config":   `{"debug":true,"port":8080}`,
			"test":     "data",
			"foo":      "bar",
			"hello":    "world",
		},
		"users": {
			"user:1001": `{"id":1001,"name":"Alice","email":"alice@example.com","age":25}`,
			"user:1002": `{"id":1002,"name":"Bob","email":"bob@example.com","age":30}`,
			"user:1003": `{"id":1003,"name":"Charlie","email":"charlie@example.com","age":35}`,
			"user:1004": `{"id":1004,"name":"Diana","email":"diana@example.com","age":28}`,
			"user:1005": `{"id":1005,"name":"Eve","email":"eve@example.com","age":32}`,
			"admin:001": `{"id":"admin001","name":"Admin User","role":"administrator","permissions":["read","write","delete"]}`,
			"guest:001": `{"id":"guest001","name":"Guest User","role":"guest","permissions":["read"]}`,
		},
		"products": {
			"prod:apple":   `{"name":"Apple","price":1.50,"category":"fruit","stock":100}`,
			"prod:banana":  `{"name":"Banana","price":0.80,"category":"fruit","stock":150}`,
			"prod:carrot":  `{"name":"Carrot","price":2.00,"category":"vegetable","stock":80}`,
			"prod:laptop":  `{"name":"Laptop","price":999.99,"category":"electronics","stock":5}`,
			"prod:mouse":   `{"name":"Mouse","price":25.99,"category":"electronics","stock":20}`,
			"sku:ABC123":   "Apple iPhone 14",
			"sku:DEF456":   "Samsung Galaxy S23",
			"sku:GHI789":   "Google Pixel 7",
			"category:001": "Electronics",
			"category:002": "Clothing",
		},
		"logs": {
			"2024-01-01T10:00:00": `{"level":"INFO","message":"Application started","service":"api"}`,
			"2024-01-01T10:01:00": `{"level":"DEBUG","message":"Database connected","service":"db"}`,
			"2024-01-01T10:02:00": `{"level":"WARN","message":"High memory usage detected","service":"monitor"}`,
			"2024-01-01T10:03:00": `{"level":"ERROR","message":"Failed to process request","service":"api","error":"timeout"}`,
			"2024-01-01T10:04:00": `{"level":"INFO","message":"Request processed successfully","service":"api","duration":"150ms"}`,
			"error:001":           "Database connection failed",
			"error:002":           "Invalid user credentials",
			"error:003":           "Permission denied",
			"metric:cpu":          "75.5",
			"metric:memory":       "82.3",
			"metric:disk":         "45.8",
		},
	}

	// Write data to column families
	for cfName, kvs := range testData {
		// Find the column family index
		cfIndex := -1
		for i, name := range cfNames {
			if name == cfName {
				cfIndex = i
				break
			}
		}
		if cfIndex == -1 {
			fmt.Printf("Column family %s not found\n", cfName)
			continue
		}

		fmt.Printf("Writing %d entries to column family '%s':\n", len(kvs), cfName)
		for k, v := range kvs {
			err := db.PutCF(wo, cfHandles[cfIndex], []byte(k), []byte(v))
			if err != nil {
				fmt.Printf("  Failed to write %s: %v\n", k, err)
			} else {
				// Show first few characters of value for confirmation
				displayValue := v
				if len(displayValue) > 50 {
					displayValue = displayValue[:47] + "..."
				}
				fmt.Printf("  %s: %s\n", k, displayValue)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Test database generated successfully at: %s\n", dbPath)
	fmt.Printf("Column families created: %v\n", cfNames)
	fmt.Println("\nTry these commands to explore the data:")
	fmt.Println("  go run cmd/main.go --db", dbPath)
	fmt.Println("  > listcf")
	fmt.Println("  > usecf users")
	fmt.Println("  > prefix user:")
	fmt.Println("  > scan user:1001 user:1005")
	fmt.Println("  > get user:1001 --pretty")
}
