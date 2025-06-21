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

## Installation and Build Process

### Prerequisites

RocksDB CLI requires RocksDB C++ libraries to be installed on your system.

#### macOS
```sh
brew install rocksdb snappy lz4 zstd bzip2

# Configure environment variables (add to ~/.zshrc or ~/.bash_profile)
export CGO_CFLAGS="-I/opt/homebrew/Cellar/rocksdb/10.2.1/include"
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/rocksdb/10.2.1/lib -L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"

# Apply environment variables
source ~/.zshrc
```

> Note: Replace 10.2.1 with your actual RocksDB version.

#### Linux (Ubuntu/Debian)
```sh
sudo apt-get update
sudo apt-get install librocksdb-dev libsnappy-dev liblz4-dev libzstd-dev libbz2-dev build-essential
```

#### Linux (CentOS/RHEL)
```sh
sudo yum install rocksdb-devel snappy-devel lz4-devel libzstd-devel bzip2-devel gcc-c++
```

#### Windows
Windows builds require more setup due to C++ dependencies. You have several options:

**Option 1: Use WSL (Recommended)**
```bash
# Install WSL and Ubuntu, then follow Linux instructions above
wsl --install
```

**Option 2: Native Windows Build**
1. Install [vcpkg](https://github.com/Microsoft/vcpkg):
```cmd
git clone https://github.com/Microsoft/vcpkg.git
cd vcpkg
.\bootstrap-vcpkg.bat
```

2. Install RocksDB and dependencies:
```cmd
.\vcpkg install rocksdb:x64-windows
.\vcpkg install snappy:x64-windows
.\vcpkg install lz4:x64-windows
.\vcpkg install zstd:x64-windows
```

3. Set environment variables:
```cmd
set CGO_CFLAGS=-I%VCPKG_ROOT%\installed\x64-windows\include
set CGO_LDFLAGS=-L%VCPKG_ROOT%\installed\x64-windows\lib -lrocksdb -lsnappy -llz4 -lzstd
set CGO_ENABLED=1
```

### Building Executables

#### Install Dependencies
```sh
go mod tidy
```

#### Quick Build (Current Platform)
```sh
# Using Make
make build

# Or directly with Go
go build -o build/rocksdb-cli ./cmd
```

#### Cross-Platform Building

**Important Note**: RocksDB CLI uses CGO (C bindings), which makes cross-compilation complex. Each platform needs its native C++ libraries.

**Option 1: Native Build (Recommended)**
Build on each target platform:
```sh
# On any platform
make build
# or
make build-native
```

**Option 2: Docker Build (Linux)**
Build Linux executables using Docker:
```sh
# Build Linux executable using Docker
make build-linux-docker

# Or manually
chmod +x scripts/build_docker.sh
./scripts/build_docker.sh
```

**Option 3: GitHub Actions (Automated)**
The repository includes GitHub Actions workflows that automatically build for all platforms:
- Push to `main` branch triggers builds
- Create a release to generate downloadable binaries
- Artifacts are available for download from Actions tab

**Option 4: Windows Build**
On Windows systems:
```cmd
# After setting up vcpkg and RocksDB (see Windows prerequisites above)
scripts\build.bat
```

#### Supported Platforms
- **Linux**: amd64, arm64 (via Docker or native)
- **macOS**: amd64 (Intel), arm64 (Apple Silicon) (native only)
- **Windows**: amd64 (native with proper setup)

Built executables are placed in the `build/` directory:
- `rocksdb-cli` (current platform)
- `rocksdb-cli-linux-amd64` (Linux)
- `rocksdb-cli-windows-amd64.exe` (Windows)

#### Running Tests
```sh
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Common Issues
- If header or library files are not found, verify environment variable paths match installation paths.
- If you see errors like `library 'snappy' not found`, ensure all dependencies are installed via package manager.
- Linker warnings about duplicate libraries can be ignored, they don't affect functionality.
- For Windows builds, ensure CGO is enabled and proper C++ toolchain is available.

## Usage

### Interactive Mode
1. Run: `go run cmd/main.go --db /path/to/rocksdb`
2. Enter commands in REPL:

### CSV Export Mode
You can also export any column family to a CSV file directly from the command line:

```sh
# Export specific column family to CSV file
rocksdb-cli --db /path/to/rocksdb --export-cf <column_family> --export-file <output.csv>

# Examples
rocksdb-cli --db ./testdb --export-cf users --export-file users.csv
rocksdb-cli --db ./testdb --export-cf products --export-file products.csv
```

The CSV export includes:
- Header row with "Key" and "Value" columns
- All key-value pairs from the specified column family
- Proper CSV escaping for special characters in JSON values

### Get Last Entry Mode
You can retrieve the last (lexicographically largest) key-value pair from any column family:

```sh
# Get last entry from specific column family
rocksdb-cli --db /path/to/rocksdb --last <column_family>

# Examples
rocksdb-cli --db ./testdb --last users
rocksdb-cli --db ./testdb --last products
```

### Watch Mode (Continuous Monitoring)
Monitor a column family for new entries in real-time, similar to `ping -t`:

```sh
# Watch specific column family for new entries
rocksdb-cli --db /path/to/rocksdb --watch <column_family> [--interval <duration>]

# Examples
rocksdb-cli --db ./testdb --watch users                    # Default 1s interval
rocksdb-cli --db ./testdb --watch logs --interval 500ms    # Custom interval
rocksdb-cli --db ./testdb --watch products --interval 2s   # 2 second interval
```

Watch mode features:
- Shows initial last entry when starting
- Detects and displays new entries as they are added
- Configurable polling interval (default: 1 second)
- Graceful shutdown with Ctrl+C
- Timestamps for each new entry detected

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

#### Interactive Commands
- `usecf <cf>`                   Switch to a column family
- `get [<cf>] <key> [--pretty]`  Query key in CF, with optional JSON pretty print
- `put [<cf>] <key> <value>`     Insert/Update key-value in CF
- `prefix [<cf>] <prefix>`       Query by prefix in CF
- `scan [<cf>] [start] [end] [options]` Scan range of keys in CF with advanced options
- `export [<cf>] <file_path>`    Export column family data to CSV file
- `listcf`                       List all column families
- `createcf <cf>`                Create new column family
- `dropcf <cf>`                  Drop column family
- `exit` or `quit`               Exit

#### Command Line Options
- `--db <path>`                  Path to RocksDB database (required)
- `--export-cf <cf>`             Column family to export (use with --export-file)
- `--export-file <path>`         Output CSV file path (use with --export-cf)
- `--last <cf>`                  Get the last key-value pair from column family
- `--watch <cf>`                 Watch for new entries in column family (like ping -t)
- `--interval <duration>`        Watch interval (default: 1s, use with --watch)

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