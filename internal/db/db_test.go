package db

import (
	"path/filepath"
	"testing"
)

func TestDB_ColumnFamilies(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Test ListCFs - should have default CF
	cfs, err := db.ListCFs()
	if err != nil {
		t.Fatalf("ListCFs failed: %v", err)
	}
	if len(cfs) != 1 || cfs[0] != "default" {
		t.Errorf("Expected only default CF, got %v", cfs)
	}

	// Test CreateCF
	err = db.CreateCF("cf1")
	if err != nil {
		t.Fatalf("CreateCF failed: %v", err)
	}

	// Verify CF was created
	cfs, err = db.ListCFs()
	if err != nil {
		t.Fatalf("ListCFs failed: %v", err)
	}
	if len(cfs) != 2 || !contains(cfs, "cf1") {
		t.Errorf("Expected default and cf1, got %v", cfs)
	}

	// Test operations on new CF
	err = db.PutCF("cf1", "key1", "value1")
	if err != nil {
		t.Fatalf("PutCF failed: %v", err)
	}

	val, err := db.GetCF("cf1", "key1")
	if err != nil {
		t.Fatalf("GetCF failed: %v", err)
	}
	if val != "value1" {
		t.Errorf("GetCF value = %s, want value1", val)
	}

	// Test PrefixScan on CF
	err = db.PutCF("cf1", "key2", "value2")
	if err != nil {
		t.Fatalf("PutCF failed: %v", err)
	}
	err = db.PutCF("cf1", "other", "value3")
	if err != nil {
		t.Fatalf("PutCF failed: %v", err)
	}

	result, err := db.PrefixScanCF("cf1", "key", 10)
	if err != nil {
		t.Fatalf("PrefixScanCF failed: %v", err)
	}
	if len(result) != 2 || result["key1"] != "value1" || result["key2"] != "value2" {
		t.Errorf("PrefixScanCF result = %v, want keys key1, key2", result)
	}

	// Test operations on default CF still work
	err = db.PutCF("default", "foo", "bar")
	if err != nil {
		t.Fatalf("PutCF on default failed: %v", err)
	}
	val, err = db.GetCF("default", "foo")
	if err != nil {
		t.Fatalf("GetCF on default failed: %v", err)
	}
	if val != "bar" {
		t.Errorf("GetCF on default value = %s, want bar", val)
	}

	// Test error cases
	_, err = db.GetCF("nonexistent", "key")
	if err == nil {
		t.Error("GetCF on nonexistent CF should fail")
	}

	err = db.PutCF("nonexistent", "key", "value")
	if err == nil {
		t.Error("PutCF on nonexistent CF should fail")
	}

	// Test DropCF
	err = db.DropCF("cf1")
	if err != nil {
		t.Fatalf("DropCF failed: %v", err)
	}

	// Verify CF was dropped
	cfs, err = db.ListCFs()
	if err != nil {
		t.Fatalf("ListCFs failed: %v", err)
	}
	if len(cfs) != 1 || cfs[0] != "default" {
		t.Errorf("Expected only default CF after drop, got %v", cfs)
	}

	// Test dropping default CF should fail
	err = db.DropCF("default")
	if err == nil {
		t.Error("Dropping default CF should fail")
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
