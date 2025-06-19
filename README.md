# rocksdb-cli

An interactive RocksDB command-line tool written in Go, with support for multiple column families (CF).

## Features
- Interactive command-line (REPL)
- Query by key
- Query by prefix
- Insert/Update key-value pairs
- Multiple column family (CF) management and operations
- JSON pretty print support for values
- Clear structure, easy to maintain and extend

## Project Structure
```
rocksdb-cli/
├── cmd/                # Main program entry
│   └── main.go
├── internal/
│   ├── db/            # RocksDB wrapper
│   │   └── db.go
│   ├── repl/          # Interactive command-line
│   │   └── repl.go
│   └── command/       # Command handling
│       └── command.go
├── scripts/           # Helper scripts
│   └── gen_testdb.go  # Generate test rocksdb database
├── go.mod
└── README.md
```

## macOS Installation and Build Process

### 1. Install RocksDB and Dependencies

```sh
brew install rocksdb snappy lz4 zstd bzip2
```

### 2. Configure Environment Variables (Recommended to add to ~/.zshrc or ~/.bash_profile)

```sh
export CGO_CFLAGS="-I/opt/homebrew/Cellar/rocksdb/10.2.1/include"
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/rocksdb/10.2.1/lib -L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"
```

> Note: Replace 10.2.1 with your actual RocksDB version.

Run `source ~/.zshrc` to apply environment variables.

### 3. Install Go Dependencies

```sh
go mod tidy
```

### 4. Build and Test

```sh
go build ./...
go test ./...
```

### 5. Common Issues
- If header or library files are not found, verify environment variable paths match installation paths.
- If you see errors like `library 'snappy' not found`, ensure all dependencies are installed via brew.
- Linker warnings about duplicate libraries can be ignored, they don't affect functionality.

## Usage
1. Run: `go run cmd/main.go --db /path/to/rocksdb`
2. Enter commands in REPL:

### Basic Commands
There are two ways to use commands:

1. Set current CF and use simplified commands:
```
usecf mycf       # Set current CF
get key          # Use current CF
put key value    # Use current CF
prefix pre       # Use current CF
```

2. Explicitly specify CF in commands:
```
get mycf key
put mycf key value
prefix mycf pre
```

### Command Reference
- `usecf <cf>`                   Switch to a column family
- `get [<cf>] <key> [--pretty]`  Query key in CF, with optional JSON pretty print
- `put [<cf>] <key> <value>`     Insert/Update key-value in CF
- `prefix [<cf>] <prefix>`       Query by prefix in CF
- `listcf`                       List all column families
- `createcf <cf>`                Create new column family
- `dropcf <cf>`                  Drop column family
- `exit` or `quit`               Exit

### JSON Pretty Print
When retrieving values that are JSON formatted, you can use the `--pretty` flag with the `get` command to format the output:

```
# Store a JSON value
put mycf user1 {"name":"John","age":30,"hobbies":["reading","coding"]}

# Regular get (single line)
get mycf user1
{"name":"John","age":30,"hobbies":["reading","coding"]}

# Pretty printed get
get mycf user1 --pretty
{
  "name": "John",
  "age": 30,
  "hobbies": [
    "reading",
    "coding"
  ]
}
```

If the value is not valid JSON, it will be displayed as is.

## Complete Solution

### 1. **Ensure testdb is created with column family**

Your original `gen_testdb.go` script used `grocksdb.OpenDb` (single CF), not `OpenDbColumnFamilies` (multiple CF).  
This will cause OpenWithCFs to open, even though there's a default CF name, there's no default CF handle.

### 2. **Correct testdb generation script**

Please change `scripts/gen_testdb.go` to use `OpenDbColumnFamilies` to create and write to default CF:

```go
package main

import (
	"fmt"
	"os"
	"github.com/linxGnu/grocksdb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: gen_testdb <db_path>")
		os.Exit(1)
	}
	dbPath := os.Args[1]
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	cfNames := []string{"default"}
	cfOpts := []*grocksdb.Options{grocksdb.NewDefaultOptions()}
	db, cfHandles, err := grocksdb.OpenDbColumnFamilies(opts, dbPath, cfNames, cfOpts)
	if err != nil {
		panic(err)
	}
	wo := grocksdb.NewDefaultWriteOptions()
	defer db.Close()
	defer wo.Destroy()

	kvs := map[string]string{
		"foo": "bar",
		"foo2": "baz",
		"fop": "zzz",
		"hello": "world",
		"prefix1": "v1",
		"prefix2": "v2",
	}
	for k, v := range kvs {
		err := db.PutCF(wo, cfHandles[0], []byte(k), []byte(v))
		if err != nil {
			fmt.Printf("写入 %s 失败: %v\n", k, err)
		}
	}
	fmt.Println("testdb 生成完毕:", dbPath)
}
```

### 3. **Regenerate testdb and test**

```sh
go run scripts/gen_testdb.go ./testdb
go run cmd/main.go --db ./testdb
# Then usecf default, get foo should return data
``` 