# rocksdb-cli

An interactive RocksDB command-line tool written in Go, with support for multiple column families (CF).

## Features
- Interactive command-line (REPL)
- Query by key with JSON pretty print support
- Query by prefix and range scanning
- **JSON field querying** - Search entries by JSON field values
- Insert/Update key-value pairs
- Multiple column family (CF) management and operations
- CSV export functionality
- Real-time monitoring with watch mode
- Docker support for easy deployment
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
├── Dockerfile         # Docker build configuration
├── build_docker.sh    # Docker build script
├── go.mod
└── README.md
```

## Installation and Build Process

### Option 1: Docker (Recommended)

Docker provides the easiest way to use rocksdb-cli without dealing with native dependencies.

#### Prerequisites
- Docker installed on your system

#### Building Docker Image
```sh
# Build Docker image (automatically detects proxy if needed)
./build_docker.sh

# Or build manually
docker build -t rocksdb-cli .
```

#### Using Docker Image
```sh
# Get help
docker run --rm rocksdb-cli --help

# Interactive mode with your database
docker run -it --rm -v "/path/to/your/db:/data" rocksdb-cli --db /data

# Command-line usage
docker run --rm -v "/path/to/your/db:/data" rocksdb-cli --db /data --last users --pretty

# CSV export
docker run --rm -v "/path/to/your/db:/data" -v "$PWD:/output" rocksdb-cli --db /data --export-cf users --export-file /output/users.csv

# Watch mode
docker run -it --rm -v "/path/to/your/db:/data" rocksdb-cli --db /data --watch logs --interval 500ms
```

#### Docker with Proxy Support
If you're behind a corporate firewall or using a proxy, the build script automatically detects and uses your proxy settings:

```sh
# Set proxy environment variables (if needed)
export HTTP_PROXY="http://your-proxy-server:port"
export HTTPS_PROXY="http://your-proxy-server:port"

# Build with proxy support
./build_docker.sh
```

Alternatively, you can manually specify proxy settings:
```sh
# Manual proxy build
docker build \
    --build-arg HTTP_PROXY="http://your-proxy-server:port" \
    --build-arg HTTPS_PROXY="http://your-proxy-server:port" \
    -t rocksdb-cli .
```

### Option 2: Native Build

For better performance or development purposes, you can build natively.

#### Prerequisites

RocksDB CLI requires RocksDB C++ libraries to be installed on your system.

##### macOS
```sh
brew install rocksdb snappy lz4 zstd bzip2

# Configure environment variables (add to ~/.zshrc or ~/.bash_profile)
export CGO_CFLAGS="-I/opt/homebrew/Cellar/rocksdb/*/include"
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/rocksdb/*/lib -L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"

# Apply environment variables
source ~/.zshrc
```

##### Linux (Ubuntu/Debian)
```sh
sudo apt-get update
sudo apt-get install librocksdb-dev libsnappy-dev liblz4-dev libzstd-dev libbz2-dev build-essential
```

##### Linux (CentOS/RHEL)
```sh
sudo yum install rocksdb-devel snappy-devel lz4-devel libzstd-devel bzip2-devel gcc-c++
```

##### Windows
For Windows, we recommend using Docker or WSL:

**Option 1: Use WSL (Recommended)**
```bash
# Install WSL and Ubuntu, then follow Linux instructions above
wsl --install
```

**Option 2: Native Windows Build**
Requires complex setup with vcpkg and Visual Studio. Docker is much easier.

#### Building Native Executable
```sh
# Install dependencies
go mod tidy

# Quick build (current platform)
go build -o rocksdb-cli ./cmd

# Or using make
make build
```

#### Running Tests
```sh
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

## Usage

### Command Line Help
```sh
# Show comprehensive help
rocksdb-cli --help

# Docker version
docker run --rm rocksdb-cli --help
```

### Interactive Mode
Start the interactive REPL:

```sh
# Native
rocksdb-cli --db /path/to/rocksdb

# Docker
docker run -it --rm -v "/path/to/db:/data" rocksdb-cli --db /data
```

### Direct Command Usage

#### Get Last Entry
```sh
# Get last entry from column family
rocksdb-cli --db /path/to/db --last users

# With pretty JSON formatting
rocksdb-cli --db /path/to/db --last users --pretty
```

#### CSV Export
```sh
# Export column family to CSV
rocksdb-cli --db /path/to/db --export-cf users --export-file users.csv
```

#### Watch Mode (Real-time Monitoring)
```sh
# Monitor column family for new entries
rocksdb-cli --db /path/to/db --watch users
rocksdb-cli --db /path/to/db --watch logs --interval 500ms
```

### Interactive Commands

Once in interactive mode, you can use these commands:

#### Basic Operations
```
# Column family management
usecf <cf>                   # Switch current column family
listcf                       # List all column families
createcf <cf>                # Create new column family
dropcf <cf>                  # Drop column family

# Data operations
get [<cf>] <key> [--pretty]  # Query by key (use --pretty for JSON formatting)
put [<cf>] <key> <value>     # Insert/Update key-value pair
prefix [<cf>] <prefix>       # Query by key prefix
last [<cf>] [--pretty]       # Get last key-value pair from CF

# Advanced operations
scan [<cf>] [start] [end] [options]  # Scan range with options
jsonquery [<cf>] <field> <value> [--pretty]  # Query by JSON field value
export [<cf>] <file_path>    # Export CF to CSV file

# Help and exit
help                         # Show interactive help
exit/quit                    # Exit the CLI
```

#### Command Usage Patterns
There are two ways to use commands:

1. **Set current CF and use simplified commands:**
```
usecf users       # Set current CF
get user:1001     # Use current CF
put user:1006 {"name":"Alice","age":25}
prefix user:      # Use current CF
```

2. **Explicitly specify CF in commands:**
```
get users user:1001
put users user:1006 {"name":"Alice","age":25}
prefix users user:
```

### JSON Pretty Print
When retrieving JSON values, use the `--pretty` flag for formatted output:

```
# Store JSON value
put users user:1001 {"name":"John","age":30,"hobbies":["reading","coding"]}

# Regular get (single line)
get users user:1001
{"name":"John","age":30,"hobbies":["reading","coding"]}

# Pretty printed get
get users user:1001 --pretty
{
  "name": "John",
  "age": 30,
  "hobbies": [
    "reading",
    "coding"
  ]
}
```

### JSON Field Querying
The `jsonquery` command allows you to search for entries based on JSON field values:

```
# Query by string field
jsonquery users name Alice
Found 1 entries in 'users' where field 'name' = 'Alice':
user:1001: {"id":1001,"name":"Alice","email":"alice@example.com","age":25}

# Query by number field
jsonquery users age 30
Found 1 entries in 'users' where field 'age' = '30':
user:1002: {"id":1002,"name":"Bob","age":30}

# Query with explicit column family
jsonquery products category fruit
Found 2 entries in 'products' where field 'category' = 'fruit':
prod:apple: {"name":"Apple","price":1.50,"category":"fruit"}
prod:banana: {"name":"Banana","price":0.80,"category":"fruit"}

# Query with pretty JSON output
jsonquery users name Alice --pretty
Found 1 entries in 'users' where field 'name' = 'Alice':
user:1001: {
  "age": 25,
  "email": "alice@example.com",
  "id": 1001,
  "name": "Alice"
}

# Using current column family
usecf logs
jsonquery level ERROR
Found entries where field 'level' = 'ERROR' in current CF
```

**Supported field types:**
- **String**: Exact match (`"Alice"`)
- **Number**: Numeric comparison (`30`, `1.5`)
- **Boolean**: Boolean comparison (`true`, `false`)
- **Null**: Null comparison (`null`)

### Range Scanning
The `scan` command provides powerful range scanning with various options:

```
# Basic range scan
scan users user:1001 user:1005

# Scan with options
scan users user:1001 user:1005 --reverse --limit=10 --values=no

# Available options:
# --reverse    : Scan in reverse order
# --limit=N    : Limit results to N entries
# --values=no  : Return only keys without values
```

## Generate Test Database

Create a comprehensive test database with sample data:

```sh
# Generate test database
go run scripts/gen_testdb.go ./testdb

# Use with CLI
rocksdb-cli --db ./testdb

# Use with Docker
docker run -it --rm -v "$PWD/testdb:/data" rocksdb-cli --db /data
```

The test database includes:
- **default**: Basic key-value pairs and configuration data
- **users**: User profiles in JSON format
- **products**: Product information with categories
- **logs**: Application logs with different severity levels

### Example Usage with Test Data

```sh
# Interactive mode
rocksdb-cli --db ./testdb

# In REPL:
> listcf                     # List all column families
> usecf users               # Switch to users
> prefix user:              # Get all users
> get user:1001 --pretty    # Get user with JSON formatting
> jsonquery name Alice      # Find users named Alice
> jsonquery users age 25    # Find users aged 25
> scan user:1001 user:1005  # Scan range of users
> export users users.csv    # Export to CSV
> usecf logs               # Switch to logs
> jsonquery level ERROR     # Find error logs
> watch logs --interval 1s  # Watch for new log entries
```

## Docker Technical Details

The Docker image includes:
- **RocksDB v10.2.1** - manually compiled for compatibility with grocksdb v1.10.1
- **Multi-stage build** - optimized for size and security
- **Non-root user** - runs as `rocksdb` user for security
- **Debian bullseye** base - for compatibility and stability

### Build Process
1. **Build stage**: Compiles RocksDB and Go application
2. **Runtime stage**: Minimal image with only runtime dependencies
3. **Total build time**: ~6-7 minutes (RocksDB compilation takes most time)
4. **Final image size**: ~200MB

### Proxy Support
The Docker build automatically handles proxy configurations:
- Detects `HTTP_PROXY` and `HTTPS_PROXY` environment variables
- Passes them to the build process if present
- No manual configuration needed in most cases

## Performance Notes

- **Docker**: Slight overhead but consistent across platforms
- **Native**: Best performance, platform-specific
- **Memory**: RocksDB compilation requires ~2GB RAM in Docker
- **Storage**: Built image is ~200MB

## Contributing

1. Fork the repository
2. Create your feature branch
3. Make changes and add tests
4. Ensure Docker build works: `./build_docker.sh`
5. Submit a pull request

## License

MIT License - see LICENSE file for details 