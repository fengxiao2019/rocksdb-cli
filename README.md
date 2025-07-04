# rocksdb-cli

An interactive RocksDB command-line tool written in Go, with support for multiple column families (CF) and AI-powered natural language queries via GraphChain Agent.

## Table of Contents
- [Quick Start](#quick-start)
- [Features](#features)
- [GraphChain Agent (AI-Powered)](#graphchain-agent-ai-powered)
  - [Quick Start with GraphChain](#quick-start-with-graphchain)
  - [Configuration](#configuration)
  - [Natural Language Examples](#natural-language-examples)
  - [Supported LLM Providers](#supported-llm-providers)
- [Installation and Build Process](#installation-and-build-process)
- [Usage](#usage)
  - [Command Line Help](#command-line-help)
  - [Interactive Mode](#interactive-mode)
  - [Direct Command Usage](#direct-command-usage)
    - [Prefix Scan](#prefix-scan)
    - [Range Scan](#range-scan)
- [Interactive Commands](#interactive-commands)
- [JSON Features](#json-pretty-print)
- [Generate Test Database](#generate-test-database)
- [MCP Server Support](#mcp-server-support)

## Quick Start

```bash
# Interactive mode
rocksdb-cli --db /path/to/database

# AI-powered mode (GraphChain Agent)
rocksdb-cli --db /path/to/database --graphchain

# Find keys by prefix
rocksdb-cli --db /path/to/database --prefix users --prefix-key "user:"

# Scan key ranges
rocksdb-cli --db /path/to/database --scan users --start "user:1000" --end "user:2000" --pretty

# Get last entry
rocksdb-cli --db /path/to/database --last users --pretty
```

## Features
- **🤖 GraphChain Agent** - AI-powered natural language queries using LLMs (OpenAI, Ollama, Google AI)
- **Interactive command-line (REPL)** - Full-featured REPL with command history
- **Query by key** with JSON pretty print support
- **Prefix scanning** - Find keys starting with specific patterns (both interactive and command-line)
- **Range scanning** - Scan key ranges with flexible options (reverse, limit, keys-only)
- **JSON field querying** - Search entries by JSON field values
- **Data manipulation** - Insert/Update key-value pairs
- **Column family management** - Full support for multiple column families
- **CSV export functionality** - Export column families to CSV files
- **Real-time monitoring** - Watch mode for live data changes
- **Docker support** - Easy deployment with pre-built Docker images
- **Read-only mode** - Safe concurrent access for production environments
- **MCP Server** - Model Context Protocol server for AI integration

## GraphChain Agent (AI-Powered)

GraphChain Agent transforms your RocksDB interactions using natural language processing. Instead of remembering specific commands, simply ask questions in plain English!

### Quick Start with GraphChain

```bash
# Start GraphChain Agent
rocksdb-cli --db /path/to/database --graphchain

# With custom configuration
rocksdb-cli --db /path/to/database --graphchain --config custom-graphchain.yaml

# Docker mode
docker run -it --rm -v "/path/to/db:/data" -v "$PWD/config:/config" \
  rocksdb-cli --db /data --graphchain --config /config/graphchain.yaml
```

### Configuration

Create a configuration file (default: `config/graphchain.yaml`):

```yaml
graphchain:
  llm:
    provider: "ollama"              # openai, googleai, ollama
    model: "llama2"                 # Model name
    api_key: "${OPENAI_API_KEY}"    # API key (not needed for Ollama)
    base_url: "http://localhost:11434"  # Ollama URL
    timeout: "30s"                  # Request timeout
  
  agent:
    max_iterations: 10              # Max tool iterations
    tool_timeout: "10s"             # Tool execution timeout
    enable_memory: true             # Enable conversation memory
    memory_size: 100                # Max conversation history
  
  security:
    enable_audit: true              # Enable audit logging
    read_only_mode: false           # Restrict to read operations
    max_query_complexity: 10        # Max query complexity
    allowed_operations: ["get", "scan", "prefix", "jsonquery", "search", "stats"]
  
  context:
    enable_auto_discovery: true     # Auto-discover database structure
    update_interval: "5m"           # Context refresh interval
    max_context_size: 4096          # Max context tokens
```

### Natural Language Examples

Once in GraphChain mode, you can ask natural questions:

#### Database Exploration
```
🤖 GraphChain Agent > Show me all column families in the database
🤖 GraphChain Agent > What's in the users column family?
🤖 GraphChain Agent > How many keys are in the logs table?
🤖 GraphChain Agent > Give me some statistics about the database
```

#### Data Queries
```
🤖 GraphChain Agent > Find all users named Alice
🤖 GraphChain Agent > Show me the last 5 entries in the logs column family
🤖 GraphChain Agent > Get all keys that start with "user:" in the users table
🤖 GraphChain Agent > Find JSON records where age is greater than 30
🤖 GraphChain Agent > Search for entries containing "error" in the value
```

#### Complex Operations
```
🤖 GraphChain Agent > Export the users column family to a CSV file
🤖 GraphChain Agent > Show me all product records where category is "electronics"
🤖 GraphChain Agent > Find the most recent log entry with level "ERROR"
🤖 GraphChain Agent > Compare the size of different column families
```

#### Data Analysis
```
🤖 GraphChain Agent > What types of data are stored in the database?
🤖 GraphChain Agent > Show me the key patterns used in the users table
🤖 GraphChain Agent > How is the data distributed across column families?
🤖 GraphChain Agent > Find unusual or interesting patterns in the data
```

### Supported LLM Providers

#### 1. Ollama (Local, Recommended)
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Start Ollama service
ollama serve

# Pull a model
ollama pull llama2        # Or llama3, codellama, mistral, etc.
```

**Configuration:**
```yaml
graphchain:
  llm:
    provider: "ollama"
    model: "llama2"
    base_url: "http://localhost:11434"
```

#### 2. OpenAI
```bash
export OPENAI_API_KEY="your-api-key-here"
```

**Configuration:**
```yaml
graphchain:
  llm:
    provider: "openai"
    model: "gpt-4"
    api_key: "${OPENAI_API_KEY}"
```

#### 3. Google AI (Gemini)
```bash
export GOOGLE_AI_API_KEY="your-api-key-here"
```

**Configuration:**
```yaml
graphchain:
  llm:
    provider: "googleai"
    model: "gemini-pro"
    api_key: "${GOOGLE_AI_API_KEY}"
```

### GraphChain Agent Features

- **🧠 Intelligent Query Planning**: Automatically selects the right tools for your questions
- **🔍 Context Awareness**: Understands database structure and content patterns
- **💬 Natural Conversation**: Ask follow-up questions and maintain context
- **🛡️ Security & Auditing**: Configurable permissions and audit logging
- **⚡ Performance Optimized**: Efficient tool selection and execution
- **🔧 Extensible**: Easy to add new tools and capabilities

### Troubleshooting GraphChain

**Common Issues:**

1. **Ollama Connection Failed**
   ```bash
   # Check if Ollama is running
   curl http://localhost:11434/api/tags
   
   # Start Ollama if not running
   ollama serve
   ```

2. **Model Not Found**
   ```bash
   # List available models
   ollama list
   
   # Pull required model
   ollama pull llama2
   ```

3. **API Key Issues (OpenAI/Google)**
   ```bash
   # Verify environment variable
   echo $OPENAI_API_KEY
   
   # Or set in config file
   api_key: "your-actual-key-here"
   ```

4. **Permission Errors**
   - Check `read_only_mode` setting in config
   - Verify `allowed_operations` includes needed operations

## Project Structure
```
rocksdb-cli/
├── cmd/                    # Main program entry
│   ├── main.go
│   └── mcp-server/        # MCP server implementation
│       └── main.go
├── internal/
│   ├── db/                # RocksDB wrapper
│   │   └── db.go
│   ├── repl/              # Interactive command-line
│   │   └── repl.go
│   ├── command/           # Command handling
│   │   └── command.go
│   ├── graphchain/        # GraphChain Agent implementation
│   │   ├── agent.go       # Core agent logic
│   │   ├── config.go      # Configuration management
│   │   ├── tools.go       # Database tools for LLM
│   │   ├── context.go     # Database context management
│   │   └── audit.go       # Audit and security
│   └── mcp/               # MCP server components
│       ├── tools.go
│       ├── resources.go
│       └── transport.go
├── config/                # Configuration files
│   ├── graphchain.yaml    # GraphChain config
│   └── mcp-server.yaml    # MCP server config
├── scripts/               # Helper scripts
│   └── gen_testdb.go      # Generate test database
├── Dockerfile             # Docker configuration
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

# Prefix scan
docker run --rm -v "/path/to/your/db:/data" rocksdb-cli --db /data --prefix users --prefix-key "user:" --pretty

# Range scan
docker run --rm -v "/path/to/your/db:/data" rocksdb-cli --db /data --scan users --start "user:1000" --limit 10

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

#### Prefix Scan
```sh
# Scan keys starting with a specific prefix
rocksdb-cli --db /path/to/db --prefix users --prefix-key "user:"

# Prefix scan with pretty JSON formatting
rocksdb-cli --db /path/to/db --prefix users --prefix-key "user:" --pretty

# Example output:
# user:1001: {"id":1001,"name":"Alice","email":"alice@example.com"}
# user:1002: {"id":1002,"name":"Bob","email":"bob@example.com"}
```

#### Range Scan
```sh
# Scan all entries in a column family
rocksdb-cli --db /path/to/db --scan users

# Scan with range
rocksdb-cli --db /path/to/db --scan users --start "user:1000" --end "user:2000"

# Scan with options
rocksdb-cli --db /path/to/db --scan users --start "user:1000" --limit 10 --reverse

# Keys only (no values)
rocksdb-cli --db /path/to/db --scan users --keys-only
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
prefix [<cf>] <prefix> [--pretty]  # Query by key prefix (supports --pretty for JSON)
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
usecf users                      # Set current CF
get user:1001                    # Use current CF
put user:1006 {"name":"Alice","age":25}
prefix user:                     # Use current CF for prefix scan
prefix user: --pretty            # Use current CF with pretty JSON formatting
```

2. **Explicitly specify CF in commands:**
```
get users user:1001                      # Specify CF in command
put users user:1006 {"name":"Alice","age":25}
prefix users user:                       # Specify CF for prefix scan
prefix users user: --pretty              # Specify CF with pretty formatting
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
> prefix user:              # Get all users starting with "user:"
> prefix user: --pretty     # Get all users with pretty JSON formatting
> get user:1001 --pretty    # Get specific user with JSON formatting
> jsonquery name Alice      # Find users named Alice
> jsonquery users age 25    # Find users aged 25
> scan user:1001 user:1005  # Scan range of users
> export users users.csv    # Export to CSV
> usecf logs               # Switch to logs
> prefix error:             # Get all error logs starting with "error:"
> jsonquery level ERROR     # Find error logs by JSON field
> watch logs --interval 1s  # Watch for new log entries
```

## MCP Server Support

RocksDB CLI includes a Model Context Protocol (MCP) server that enables integration with AI assistants like Claude Desktop, allowing AI tools to interact with your RocksDB databases through natural language.

### Quick Start with MCP Server

```bash
# Start MCP server (read-only mode, recommended)
./cmd/mcp-server/rocksdb-mcp-server --db /path/to/database --readonly

# With configuration file
./cmd/mcp-server/rocksdb-mcp-server --config config/mcp-server.yaml
```

### Claude Desktop Integration

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "rocksdb": {
      "command": "/path/to/rocksdb-mcp-server",
      "args": ["--db", "/path/to/database", "--readonly"]
    }
  }
}
```

Then interact with your database using natural language:
- "Show me all users in the database"
- "Export the products table to CSV"
- "Find all logs with error level"

### Available MCP Tools

The MCP server provides these tools for AI assistants:
- **Database Operations**: Get, scan, prefix search
- **Column Family Management**: List, create, drop CFs
- **Data Export**: CSV export functionality
- **JSON Queries**: Search by JSON field values

### MCP vs GraphChain Agent

| Feature | MCP Server | GraphChain Agent |
|---------|------------|------------------|
| **Integration** | External AI (Claude Desktop) | Built-in AI agent |
| **Protocol** | Standard MCP protocol | Direct LLM integration |
| **Setup** | Requires Claude Desktop | Self-contained |
| **Security** | Tool-level permissions | Full security controls |

> 📖 **For comprehensive MCP documentation**, including detailed configuration, API reference, security considerations, and troubleshooting, see **[docs/MCP_SERVER_README.md](docs/MCP_SERVER_README.md)**.

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

# Prefix Scan Caveats for Binary/Uint64 Keys

> For column families with binary or uint64 keys, the prefix command only matches byte prefixes, not numeric string prefixes. For example, `prefix 0x00` matches all keys starting with byte 0x00, but `prefix 123` only matches the key with value 123 as an 8-byte big-endian integer. 