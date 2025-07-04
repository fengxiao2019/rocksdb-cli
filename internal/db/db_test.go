package db

import (
	"fmt"
	"os"
	"path/filepath"
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
