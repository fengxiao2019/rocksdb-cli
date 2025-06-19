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

	// First create DB with default CF
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	// First open with just default CF to create the DB
	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		panic(err)
	}

	// Create additional column families
	cfOpts := grocksdb.NewDefaultOptions()
	handles, err := db.CreateColumnFamilies(cfOpts, []string{"cf1", "cf2"})
	if err != nil {
		panic(err)
	}
	for _, handle := range handles {
		handle.Destroy()
	}
	db.Close()

	// Now reopen with all column families
	cfNames := []string{"default", "cf1", "cf2"}
	cfOpts2 := make([]*grocksdb.Options, len(cfNames))
	for i := range cfOpts2 {
		cfOpts2[i] = grocksdb.NewDefaultOptions()
	}

	// Open DB with column families
	db, cfHandles, err := grocksdb.OpenDbColumnFamilies(opts, dbPath, cfNames, cfOpts2)
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
	data := map[int]map[string]string{
		0: { // default CF
			"foo":  "bar",
			"foo2": "baz",
		},
		1: { // cf1
			"hello":   "world",
			"prefix1": "v1",
		},
		2: { // cf2
			"fop":     "zzz",
			"prefix2": "v2",
		},
	}

	// Write data to different column families
	for cfIdx, kvs := range data {
		for k, v := range kvs {
			err := db.PutCF(wo, cfHandles[cfIdx], []byte(k), []byte(v))
			if err != nil {
				fmt.Printf("Failed to write %s to CF %s: %v\n", k, cfNames[cfIdx], err)
			}
		}
	}

	fmt.Printf("Test database generated at: %s\n", dbPath)
	fmt.Println("Column families created:", cfNames)
}
