package db

import (
	"path/filepath"
	"testing"
)

func TestDB_Basic(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Put
	err = db.Put("foo", "bar")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get
	val, err := db.Get("foo")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "bar" {
		t.Errorf("Get value = %s, want bar", val)
	}

	// Not found
	_, err = db.Get("notfound")
	if err == nil {
		t.Error("Get notfound should error")
	}

	// PrefixScan
	_ = db.Put("prefix1", "v1")
	_ = db.Put("prefix2", "v2")
	_ = db.Put("other", "v3")
	res, err := db.PrefixScan("prefix", 10)
	if err != nil {
		t.Fatalf("PrefixScan failed: %v", err)
	}
	if len(res) != 2 || res["prefix1"] != "v1" || res["prefix2"] != "v2" {
		t.Errorf("PrefixScan result = %#v", res)
	}
}

func TestDB_PutGetPrefixScan(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Put
	err = db.Put("foo", "bar")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get
	val, err := db.Get("foo")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "bar" {
		t.Errorf("Get value = %s, want bar", val)
	}

	// PrefixScan
	_ = db.Put("foo2", "baz")
	_ = db.Put("fop", "zzz")
	result, err := db.PrefixScan("foo", 10)
	if err != nil {
		t.Fatalf("PrefixScan failed: %v", err)
	}
	if len(result) != 2 || result["foo"] != "bar" || result["foo2"] != "baz" {
		t.Errorf("PrefixScan result = %#v, want keys foo, foo2", result)
	}
}
