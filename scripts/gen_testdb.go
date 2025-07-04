package main

import (
	"encoding/binary"
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
	cfNames := []string{"default", "users", "products", "logs", "binary_keys"}
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

	// Helper function to create uint64 byte array key
	longToBytes := func(val uint64) []byte {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, val)
		return buf
	}

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
			"user:1001":    `{"id":1001,"name":"Alice","email":"alice@example.com","age":25}`,
			"user:1002":    `{"id":1002,"name":"Bob","email":"bob@example.com","age":30}`,
			"user:1003":    `{"id":1003,"name":"Charlie","email":"charlie@example.com","age":35}`,
			"user:1004":    `{"id":1004,"name":"Diana","email":"diana@example.com","age":28}`,
			"user:1005":    `{"id":1005,"name":"Eve","email":"eve@example.com","age":32}`,
			"admin:001":    `{"id":"admin001","name":"Admin User","role":"administrator","permissions":["read","write","delete"]}`,
			"guest:001":    `{"id":"guest001","name":"Guest User","role":"guest","permissions":["read"]}`,
			"user:nested1": `{"user_id":"123","profile":"{\"name\":\"Alice\",\"age\":30,\"city\":\"New York\"}","preferences":"{\"theme\":\"dark\",\"notifications\":true}"}`,
			"user:nested2": `{"event_id":"evt_001","metadata":"{\"timestamp\":\"2024-01-15T10:30:00Z\",\"source\":\"api\",\"details\":{\"user_agent\":\"Mozilla/5.0\",\"ip\":\"192.168.1.1\"}}","payload":"{\"users\":[{\"id\":1,\"name\":\"Alice\"},{\"id\":2,\"name\":\"Bob\"}],\"action\":\"login\"}"}`,
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
			"order:12345":  `{"id":"order_123","description":"Customer order for electronics","order_data":"{\"items\":[{\"product\":\"laptop\",\"price\":999.99},{\"product\":\"mouse\",\"price\":29.99}],\"total\":1029.98}","note":"Priority shipping requested"}`,
			"order:67890":  `{"order_id":"67890","customer_info":"{\"name\":\"John Doe\",\"address\":{\"street\":\"123 Main St\",\"city\":\"Springfield\",\"zip\":\"12345\"}}","items":"[{\"id\":\"item1\",\"name\":\"Widget A\"},{\"id\":\"item2\",\"name\":\"Widget B\"}]"}`,
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

	// Binary keys test data
	binaryKeys := make(map[string]string)

	// Add long integer keys (8 bytes, big-endian)
	longKeys := []uint64{
		123456789,            // Normal integer
		18446744073709551615, // Max uint64
		0,                    // Zero
		4294967295,           // Max uint32
		1234567890123456789,  // Large number
	}

	for _, val := range longKeys {
		key := string(longToBytes(val))
		binaryKeys[key] = fmt.Sprintf("Value for long key: %d", val)
	}

	// Add some mixed binary keys
	mixedBinaryKeys := []struct {
		key   []byte
		value string
	}{
		{[]byte{0xFF, 0xAA, 0xBB, 0xCC}, "Binary key with 4 bytes"},
		{[]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}, "Binary key with 6 bytes"},
		{[]byte{0xDE, 0xAD, 0xBE, 0xEF}, "Classic binary pattern"},
		{append([]byte("key:"), 0x00, 0xFF, 0x00), "Mixed ASCII and binary"},
		{[]byte{0x01, 0x02, 0x03, 0x04, 0x05}, "Sequential bytes"},
	}

	for _, item := range mixedBinaryKeys {
		binaryKeys[string(item.key)] = item.value
	}

	testData["binary_keys"] = binaryKeys

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
				// For binary keys, show hex representation
				keyDisplay := k
				if cfName == "binary_keys" {
					keyDisplay = fmt.Sprintf("0x%x", []byte(k))
				}
				// Show first few characters of value for confirmation
				displayValue := v
				if len(displayValue) > 50 {
					displayValue = displayValue[:47] + "..."
				}
				fmt.Printf("  %s: %s\n", keyDisplay, displayValue)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Test database generated successfully at: %s\n", dbPath)
	fmt.Printf("Column families created: %v\n", cfNames)
	fmt.Println("\nTry these commands to explore the data:")
	fmt.Println("  go run cmd/main.go --db", dbPath)
	fmt.Println("  > listcf")
	fmt.Println("  > usecf binary_keys")
	fmt.Println("  > scan")
	fmt.Println("  > scan * *")
	fmt.Println("\nTest binary key formatting:")
	fmt.Println("  > usecf binary_keys")
	fmt.Println("  > get 0x0000000075bcd15") // Will show 123456789
	fmt.Println("  > prefix 0x")             // Show all binary keys
	fmt.Println("  > scan")                  // Show all entries with formatted keys
}
