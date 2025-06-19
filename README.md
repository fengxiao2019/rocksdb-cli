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
- `scan [<cf>] [start] [end] [options]` Scan range of keys in CF with advanced options
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

### Range Scanning
The `scan` command provides powerful range scanning capabilities with various options:

```
# Basic range scan (forward)
scan key1 key5              # Scan from key1 to key5 (exclusive)

# Scan with explicit column family
scan mycf key1 key5         # Scan in specific column family

# Reverse scan
scan key1 key5 --reverse    # Scan backwards from key5 to key1

# Scan with limit
scan key1 key5 --limit=10   # Return at most 10 results

# Scan without values (keys only)
scan key1 key5 --values=no  # Return only keys, not values

# Scan from start to end of database
scan key1                   # Scan from key1 to end

# Combined options
scan mycf key1 key5 --reverse --limit=5 --values=no
```

**Scan Options:**
- `--reverse`: Scan in reverse order (from end to start)
- `--limit=N`: Limit results to N entries
- `--values=no`: Return only keys without values

**Range Behavior:**
- Forward scan: includes start key, excludes end key `[start, end)`
- Reverse scan: starts just before end key, goes backwards to start key
- If no end key provided, scans to the end/beginning of the database

## Generate Test Database

The included `gen_testdb.go` script creates a comprehensive test database with multiple column families and sample data:

```sh
# Generate test database
go run scripts/gen_testdb.go ./testdb

# Start the CLI with the test database
go run cmd/main.go --db ./testdb
```

The test database includes:
- **default**: Basic key-value pairs and configuration data
- **users**: User profiles in JSON format with different roles
- **products**: Product information including electronics and groceries
- **logs**: Application logs with different severity levels and metrics

### Example Usage with Test Data

```sh
# List all column families
> listcf

# Switch to users column family
> usecf users

# Get all users with prefix
> prefix user:

# Scan range of users
> scan user:1001 user:1005

# Get user details with pretty JSON formatting
> get user:1001 --pretty

# Switch to products and explore
> usecf products
> prefix prod:
> scan sku:ABC123 sku:GHI789 --reverse

# View logs with different filters
> usecf logs
> prefix error:
> scan 2024-01-01T10:00:00 2024-01-01T10:05:00 --limit=3
``` 