package db

import (
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestDB_ColumnFamilies tests column family operations using table-driven tests
func TestDB_ColumnFamilies(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name      string
		setup     func(*testing.T, *DB)
		operation func(*testing.T, *DB) error
		validate  func(*testing.T, *DB) error
		wantErr   bool
	}{
		{
			name: "initial state should have default CF",
			operation: func(t *testing.T, db *DB) error {
				cfs, err := db.ListCFs()
				if err != nil {
					return err
				}
				if len(cfs) != 1 || cfs[0] != "default" {
					t.Errorf("Expected only default CF, got %v", cfs)
				}
				return nil
			},
		},
		{
			name: "create new column family",
			operation: func(t *testing.T, db *DB) error {
				return db.CreateCF("cf1")
			},
			validate: func(t *testing.T, db *DB) error {
				cfs, err := db.ListCFs()
				if err != nil {
					return err
				}
				if len(cfs) != 2 || !contains(cfs, "cf1") {
					t.Errorf("Expected default and cf1, got %v", cfs)
				}
				return nil
			},
		},
		{
			name: "put and get operations on new CF",
			setup: func(t *testing.T, db *DB) {
				if err := db.CreateCF("testcf"); err != nil {
					t.Fatalf("CreateCF failed: %v", err)
				}
			},
			operation: func(t *testing.T, db *DB) error {
				if err := db.PutCF("testcf", "key1", "value1"); err != nil {
					return err
				}
				val, err := db.GetCF("testcf", "key1")
				if err != nil {
					return err
				}
				if val != "value1" {
					t.Errorf("GetCF value = %s, want value1", val)
				}
				return nil
			},
		},
		{
			name: "prefix scan on CF",
			setup: func(t *testing.T, db *DB) {
				if err := db.CreateCF("prefixcf"); err != nil {
					t.Fatalf("CreateCF failed: %v", err)
				}
				db.PutCF("prefixcf", "key1", "value1")
				db.PutCF("prefixcf", "key2", "value2")
				db.PutCF("prefixcf", "other", "value3")
			},
			operation: func(t *testing.T, db *DB) error {
				result, err := db.PrefixScanCF("prefixcf", "key", 10)
				if err != nil {
					return err
				}
				if len(result) != 2 || result["key1"] != "value1" || result["key2"] != "value2" {
					t.Errorf("PrefixScanCF result = %v, want keys key1, key2", result)
				}
				return nil
			},
		},
		{
			name: "operations on default CF still work",
			operation: func(t *testing.T, db *DB) error {
				if err := db.PutCF("default", "foo", "bar"); err != nil {
					return err
				}
				val, err := db.GetCF("default", "foo")
				if err != nil {
					return err
				}
				if val != "bar" {
					t.Errorf("GetCF on default value = %s, want bar", val)
				}
				return nil
			},
		},
		{
			name: "get from nonexistent CF should fail",
			operation: func(t *testing.T, db *DB) error {
				_, err := db.GetCF("nonexistent", "key")
				if err == nil {
					t.Error("GetCF on nonexistent CF should fail")
				}
				return nil // We expect this to "fail" but that's the correct behavior
			},
		},
		{
			name: "put to nonexistent CF should fail",
			operation: func(t *testing.T, db *DB) error {
				err := db.PutCF("nonexistent", "key", "value")
				if err == nil {
					t.Error("PutCF on nonexistent CF should fail")
				}
				return nil // We expect this to "fail" but that's the correct behavior
			},
		},
		{
			name: "drop column family",
			setup: func(t *testing.T, db *DB) {
				if err := db.CreateCF("dropme"); err != nil {
					t.Fatalf("CreateCF failed: %v", err)
				}
			},
			operation: func(t *testing.T, db *DB) error {
				return db.DropCF("dropme")
			},
			validate: func(t *testing.T, db *DB) error {
				cfs, err := db.ListCFs()
				if err != nil {
					return err
				}
				if contains(cfs, "dropme") {
					t.Errorf("Column family 'dropme' should have been dropped, got %v", cfs)
				}
				return nil
			},
		},
		{
			name: "drop default CF should fail",
			operation: func(t *testing.T, db *DB) error {
				err := db.DropCF("default")
				if err == nil {
					t.Error("Dropping default CF should fail")
				}
				return nil // We expect this to "fail" but that's the correct behavior
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := tt.operation(t, db)
			if (err != nil) != tt.wantErr {
				t.Errorf("operation error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				if err := tt.validate(t, db); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

// TestDB_Scan tests scanning operations using table-driven tests
func TestDB_Scan(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Setup test data
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
		"key5": "value5",
	}

	for k, v := range testData {
		if err := db.PutCF("default", k, v); err != nil {
			t.Fatalf("PutCF failed: %v", err)
		}
	}

	tests := []struct {
		name     string
		start    []byte
		end      []byte
		opts     ScanOptions
		validate func(*testing.T, map[string]string)
		wantErr  bool
	}{
		{
			name:  "forward scan with no end key",
			start: []byte("key2"),
			end:   nil,
			opts:  ScanOptions{Values: true},
			validate: func(t *testing.T, result map[string]string) {
				if len(result) != 4 {
					t.Errorf("Forward scan result count = %d, want 4", len(result))
				}
				if v, ok := result["key2"]; !ok || v != "value2" {
					t.Errorf("Forward scan result missing or wrong value for key2: got %v", v)
				}
				if v, ok := result["key5"]; !ok || v != "value5" {
					t.Errorf("Forward scan result missing or wrong value for key5: got %v", v)
				}
			},
		},
		{
			name:  "reverse scan with end key",
			start: []byte("key2"),
			end:   []byte("key4"),
			opts:  ScanOptions{Reverse: true, Values: true},
			validate: func(t *testing.T, result map[string]string) {
				if len(result) != 2 {
					t.Errorf("Reverse scan result count = %d, want 2", len(result))
				}
				if v, ok := result["key3"]; !ok || v != "value3" {
					t.Errorf("Reverse scan result missing or wrong value for key3: got %v", v)
				}
				if v, ok := result["key2"]; !ok || v != "value2" {
					t.Errorf("Reverse scan result missing or wrong value for key2: got %v", v)
				}
			},
		},
		{
			name:  "reverse scan with only start key",
			start: []byte("key3"),
			end:   nil,
			opts:  ScanOptions{Reverse: true, Values: true},
			validate: func(t *testing.T, result map[string]string) {
				// Should start from key3 and scan backwards to key1
				expectedKeys := []string{"key3", "key2", "key1"}
				if len(result) != len(expectedKeys) {
					t.Errorf("Reverse scan result count = %d, want %d", len(result), len(expectedKeys))
				}
				for _, key := range expectedKeys {
					if v, ok := result[key]; !ok || v != "value"+key[3:] {
						t.Errorf("Reverse scan result missing or wrong value for %s: got %v", key, v)
					}
				}
				// Verify that key4 and key5 are NOT included (since we start from key3)
				if _, ok := result["key4"]; ok {
					t.Errorf("Reverse scan should not include key4 when starting from key3")
				}
				if _, ok := result["key5"]; ok {
					t.Errorf("Reverse scan should not include key5 when starting from key3")
				}
			},
		},
		{
			name:  "scan with limit",
			start: []byte("key1"),
			end:   nil,
			opts:  ScanOptions{Limit: 2},
			validate: func(t *testing.T, result map[string]string) {
				if len(result) != 2 {
					t.Errorf("Scan with limit result count = %d, want 2", len(result))
				}
			},
		},
		{
			name:  "scan without values",
			start: []byte("key1"),
			end:   nil,
			opts:  ScanOptions{Values: false},
			validate: func(t *testing.T, result map[string]string) {
				for _, v := range result {
					if v != "" {
						t.Errorf("Scan without values should return empty values, got %v", v)
					}
				}
			},
		},
		{
			name:    "scan on non-existent CF should fail",
			start:   []byte("key1"),
			end:     nil,
			opts:    ScanOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := "default"
			if tt.wantErr {
				cf = "nonexistent" // Use nonexistent CF for error test
			}

			result, err := db.ScanCF(cf, tt.start, tt.end, tt.opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("ScanCF error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestDB_ScanPaginated tests cursor-based pagination for ScanCF
func TestDB_ScanPaginated(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Insert 10 keys: key01 ... key10
	total := 10
	for i := 1; i <= total; i++ {
		k := fmt.Sprintf("key%02d", i)
		v := fmt.Sprintf("value%02d", i)
		if err := db.PutCF("default", k, v); err != nil {
			t.Fatalf("PutCF failed: %v", err)
		}
	}

	scanPage := func(startAfter string, limit int) (ScanPageResult, error) {
		return db.ScanCFPage("default", nil, nil, ScanOptions{Values: true, Limit: limit, StartAfter: startAfter})
	}

	// First page
	page1, err := scanPage("", 3)
	if err != nil {
		t.Fatalf("scanPage failed: %v", err)
	}
	if len(page1.Results) != 3 {
		t.Errorf("First page result count = %d, want 3", len(page1.Results))
	}
	if !page1.HasMore {
		t.Errorf("First page should have more")
	}
	if page1.NextCursor == "" {
		t.Errorf("First page should have next cursor")
	}

	// Middle page
	page2, err := scanPage(page1.NextCursor, 3)
	if err != nil {
		t.Fatalf("scanPage failed: %v", err)
	}
	if len(page2.Results) != 3 {
		t.Errorf("Second page result count = %d, want 3", len(page2.Results))
	}
	if !page2.HasMore {
		t.Errorf("Second page should have more")
	}
	if page2.NextCursor == "" {
		t.Errorf("Second page should have next cursor")
	}

	// Last page
	pageLast, err := scanPage(page2.NextCursor, 5)
	if err != nil {
		t.Fatalf("scanPage failed: %v", err)
	}
	if len(pageLast.Results) != 4 {
		t.Errorf("Last page result count = %d, want 4", len(pageLast.Results))
	}
	if pageLast.HasMore {
		t.Errorf("Last page should not have more")
	}
	if pageLast.NextCursor != "" {
		t.Errorf("Last page should not have next cursor")
	}

	// Edge: empty page
	emptyPage, err := scanPage("key10", 3)
	if err != nil {
		t.Fatalf("scanPage failed: %v", err)
	}
	if len(emptyPage.Results) != 0 {
		t.Errorf("Empty page result count = %d, want 0", len(emptyPage.Results))
		for k := range emptyPage.Results {
			t.Logf("Empty page returned key: %s", k)
		}
	}
	if emptyPage.HasMore {
		t.Errorf("Empty page should not have more")
	}
	if emptyPage.NextCursor != "" {
		t.Errorf("Empty page should not have next cursor")
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestDB_ExportToCSVWithSep(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	cf := "exportcf"
	err = db.CreateCF(cf)
	if err != nil {
		t.Fatalf("CreateCF failed: %v", err)
	}
	// 插入测试数据
	testData := map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}
	for k, v := range testData {
		err := db.PutCF(cf, k, v)
		if err != nil {
			t.Fatalf("PutCF failed: %v", err)
		}
	}

	t.Run("comma separator", func(t *testing.T) {
		csvPath := filepath.Join(dir, "out_comma.csv")
		err := db.ExportToCSV(cf, csvPath, ",")
		if err != nil {
			t.Fatalf("ExportToCSV failed: %v", err)
		}
		data, err := os.ReadFile(csvPath)
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		content := string(data)
		if !(containsLine(content, "Key,Value") && containsLine(content, "k1,v1")) {
			t.Errorf("CSV content missing expected lines: %s", content)
		}
	})

	t.Run("semicolon separator", func(t *testing.T) {
		csvPath := filepath.Join(dir, "out_semicolon.csv")
		err := db.ExportToCSV(cf, csvPath, ";")
		if err != nil {
			t.Fatalf("ExportToCSV failed: %v", err)
		}
		data, err := os.ReadFile(csvPath)
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		content := string(data)
		if !(containsLine(content, "Key;Value") && containsLine(content, "k2;v2")) {
			t.Errorf("CSV content missing expected lines: %s", content)
		}
	})

	t.Run("tab separator", func(t *testing.T) {
		csvPath := filepath.Join(dir, "out_tab.csv")
		err := db.ExportToCSV(cf, csvPath, "\t")
		if err != nil {
			t.Fatalf("ExportToCSV failed: %v", err)
		}
		data, err := os.ReadFile(csvPath)
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		content := string(data)
		if !(containsLine(content, "Key\tValue") && containsLine(content, "k3\tv3")) {
			t.Errorf("CSV content missing expected lines: %s", content)
		}
	})
}

func TestDB_ExportSearchResultsToCSV(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	cf := "test_search_export"

	// Create column family and add test data
	err = db.CreateCF(cf)
	if err != nil {
		t.Fatalf("Failed to create CF: %v", err)
	}

	// Add test data
	testData := map[string]string{
		"user:1001":   `{"name":"Alice","type":"admin"}`,
		"user:1002":   `{"name":"Bob","type":"user"}`,
		"user:1003":   `{"name":"Charlie","type":"admin"}`,
		"product:001": `{"name":"Widget","category":"tools"}`,
		"product:002": `{"name":"Gadget","category":"electronics"}`,
	}

	for key, value := range testData {
		err := db.PutCF(cf, key, value)
		if err != nil {
			t.Fatalf("Failed to put key %s: %v", key, err)
		}
	}

	tests := []struct {
		name        string
		opts        SearchOptions
		separator   string
		wantRecords int
		wantHeaders []string
	}{
		{
			name: "export search by key pattern",
			opts: SearchOptions{
				KeyPattern: "user:",
				Limit:      10,
			},
			separator:   ",",
			wantRecords: 3, // 3 user records
			wantHeaders: []string{"Key", "Value"},
		},
		{
			name: "export search by value pattern",
			opts: SearchOptions{
				ValuePattern: "admin",
				Limit:        10,
			},
			separator:   ",",
			wantRecords: 2, // 2 admin records
			wantHeaders: []string{"Key", "Value"},
		},
		{
			name: "export keys only",
			opts: SearchOptions{
				KeyPattern: "product:",
				KeysOnly:   true,
				Limit:      10,
			},
			separator:   ",",
			wantRecords: 2, // 2 product records
			wantHeaders: []string{"Key"},
		},
		{
			name: "export with semicolon separator",
			opts: SearchOptions{
				KeyPattern: "user:",
				Limit:      10,
			},
			separator:   ";",
			wantRecords: 3,
			wantHeaders: []string{"Key", "Value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csvPath := filepath.Join(t.TempDir(), "search_export.csv")

			err := db.ExportSearchResultsToCSV(cf, csvPath, tt.separator, tt.opts)
			if err != nil {
				t.Fatalf("ExportSearchResultsToCSV failed: %v", err)
			}

			// Verify the exported file
			file, err := os.Open(csvPath)
			if err != nil {
				t.Fatalf("Failed to open exported file: %v", err)
			}
			defer file.Close()

			reader := csv.NewReader(file)
			if tt.separator != "" {
				reader.Comma = rune(tt.separator[0])
			}

			records, err := reader.ReadAll()
			if err != nil {
				t.Fatalf("Failed to read CSV: %v", err)
			}

			// Check header
			if len(records) == 0 {
				t.Fatal("No records found in CSV")
			}

			if !reflect.DeepEqual(records[0], tt.wantHeaders) {
				t.Errorf("Expected headers %v, got %v", tt.wantHeaders, records[0])
			}

			// Check number of data records (excluding header)
			if len(records)-1 != tt.wantRecords {
				t.Errorf("Expected %d data records, got %d", tt.wantRecords, len(records)-1)
			}

			// Verify content structure
			for i := 1; i < len(records); i++ {
				record := records[i]
				if tt.opts.KeysOnly {
					if len(record) != 1 {
						t.Errorf("Keys-only record should have 1 field, got %d", len(record))
					}
				} else {
					if len(record) != 2 {
						t.Errorf("Full record should have 2 fields, got %d", len(record))
					}
				}
			}
		})
	}
}

func TestDB_ExportSearchResultsToCSV_Errors(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Test with non-existent column family
	csvPath := filepath.Join(t.TempDir(), "error_test.csv")
	opts := SearchOptions{KeyPattern: "test"}

	err = db.ExportSearchResultsToCSV("non_existent_cf", csvPath, ",", opts)
	if !errors.Is(err, ErrColumnFamilyNotFound) {
		t.Errorf("Expected ErrColumnFamilyNotFound, got: %v", err)
	}

	// Test with invalid separator
	cf := "test_cf"
	err = db.CreateCF(cf)
	if err != nil {
		t.Fatalf("Failed to create CF: %v", err)
	}

	err = db.ExportSearchResultsToCSV(cf, csvPath, "invalid_sep", opts)
	if err == nil || !strings.Contains(err.Error(), "CSV separator must be a single character") {
		t.Errorf("Expected separator error, got: %v", err)
	}
}

// containsLine checks if content contains a line with the given substring
func containsLine(content, substr string) bool {
	for _, line := range splitLines(content) {
		if line == substr {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

func TestDB_SearchCF_NumericKeyAfterPagination(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	cf := "test_numeric_keys"

	// Create column family
	err = db.CreateCF(cf)
	if err != nil {
		t.Fatalf("Failed to create CF: %v", err)
	}

	// Add test data with numeric keys (8-byte uint64 big-endian)
	numericKeys := []uint64{
		1000,
		2000,
		3000,
		4000,
		5000,
		10000, // This should come after 5000 in numeric order
		20000, // But might come before in string order
	}

	for i, keyNum := range numericKeys {
		// Convert uint64 to 8-byte big-endian binary key
		keyBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(keyBytes, keyNum)
		keyStr := string(keyBytes)

		value := fmt.Sprintf("value%d", i)
		err := db.PutCF(cf, keyStr, value)
		if err != nil {
			t.Fatalf("Failed to put numeric key %d: %v", keyNum, err)
		}
	}

	// Test 1: Search without after parameter to establish baseline
	opts := SearchOptions{
		KeyPattern: "*",
		Limit:      3,
	}

	results, err := db.SearchCF(cf, opts)
	if err != nil {
		t.Fatalf("Initial search failed: %v", err)
	}

	if len(results.Results) != 3 {
		t.Fatalf("Expected 3 results in first page, got %d", len(results.Results))
	}

	t.Logf("First page keys: %v", func() []string {
		keys := make([]string, len(results.Results))
		for i, r := range results.Results {
			// Format binary key as number for readability
			if len(r.Key) == 8 {
				val := binary.BigEndian.Uint64([]byte(r.Key))
				keys[i] = fmt.Sprintf("%d", val)
			} else {
				keys[i] = r.Key
			}
		}
		return keys
	}())

	// Test 2: Use string representation of a number as after parameter
	// This should work for numeric keys - user passes "3000"
	optsWithNumericAfter := SearchOptions{
		KeyPattern: "*",
		Limit:      3,
		After:      "3000", // User passes string representation of number
	}

	numericAfterResults, err := db.SearchCF(cf, optsWithNumericAfter)
	if err != nil {
		t.Fatalf("Search with numeric after failed: %v", err)
	}

	t.Logf("Results after '3000': %d", len(numericAfterResults.Results))

	// This is where the bug manifests - string comparison "3000" vs binary keys
	// will not work correctly
	if len(numericAfterResults.Results) == 0 {
		t.Error("BUG: No results when using numeric string as after parameter for binary numeric keys")
	}

	// Test 3: Use cursor from previous search (this should work)
	if results.HasMore {
		optsWithCursor := SearchOptions{
			KeyPattern: "*",
			Limit:      3,
			After:      results.NextCursor,
		}

		cursorResults, err := db.SearchCF(cf, optsWithCursor)
		if err != nil {
			t.Fatalf("Search with cursor after failed: %v", err)
		}

		t.Logf("Results with cursor: %d", len(cursorResults.Results))

		// Verify no duplicate keys between pages
		firstPageKeys := make(map[string]bool)
		for _, result := range results.Results {
			firstPageKeys[result.Key] = true
		}

		for _, result := range cursorResults.Results {
			if firstPageKeys[result.Key] {
				t.Error("Found duplicate key between pages when using cursor")
			}
		}
	}

	// Test 4: Verify the numeric ordering is correct
	// Get all results and check they are in ascending numeric order
	allOpts := SearchOptions{
		KeyPattern: "*",
		Limit:      100,
	}

	allResults, err := db.SearchCF(cf, allOpts)
	if err != nil {
		t.Fatalf("Search for all results failed: %v", err)
	}

	if len(allResults.Results) != len(numericKeys) {
		t.Fatalf("Expected %d total results, got %d", len(numericKeys), len(allResults.Results))
	}

	// Check numeric ordering
	var prevNum uint64 = 0
	for i, result := range allResults.Results {
		if len(result.Key) == 8 {
			currentNum := binary.BigEndian.Uint64([]byte(result.Key))
			if i > 0 && currentNum <= prevNum {
				t.Errorf("Results not in ascending numeric order: %d <= %d at position %d", currentNum, prevNum, i)
			}
			prevNum = currentNum
		}
	}
}
