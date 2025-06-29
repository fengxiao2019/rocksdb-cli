# RocksDB CLI MCP Server Integration

This document provides comprehensive information about the Model Context Protocol (MCP) server integration for the RocksDB CLI tool.

## Overview

The MCP server integration allows AI models to interact with RocksDB databases through a standardized protocol. This enables AI assistants to perform database operations, query data, and manage column families through natural language commands.

## Features

### Core Capabilities

- **Database Operations**: Get, put, scan, and prefix search operations
- **Column Family Management**: Create, drop, list, and manage column families
- **Query Support**: JSON field queries and CSV export functionality
- **Real-time Monitoring**: Get latest entries and database statistics
- **Multi-transport Support**: STDIO, TCP, WebSocket, and Unix socket transports
- **Security**: Read-only mode support and configurable access controls

### MCP Tools Available

| Tool | Function | Description |
|------|----------|-------------|
| `rocksdb_get` | Get key value | Retrieve a value by key from specified column family |
| `rocksdb_put` | Put key-value | Store a key-value pair in specified column family |
| `rocksdb_scan` | Scan range | Scan a range of keys with optional filters |
| `rocksdb_prefix_scan` | Prefix search | Find all keys with a given prefix |
| `rocksdb_list_column_families` | List CFs | List all available column families |
| `rocksdb_create_column_family` | Create CF | Create a new column family |
| `rocksdb_drop_column_family` | Drop CF | Delete a column family |
| `rocksdb_export_to_csv` | Export data | Export column family data to CSV |
| `rocksdb_json_query` | JSON query | Query entries by JSON field values |
| `rocksdb_get_last` | Get latest | Retrieve the most recent entry |

### MCP Prompts Available

| Prompt | Purpose | Description |
|--------|---------|-------------|
| `rocksdb_data_analysis` | Data analysis | Generate prompts for exploring RocksDB data |
| `rocksdb_query_generator` | Query generation | Create optimized queries for specific use cases |
| `rocksdb_troubleshooting` | Troubleshooting | Generate debugging prompts for RocksDB issues |
| `rocksdb_schema_design` | Schema design | Design column family structures |

## Installation and Setup

### Prerequisites

- Go 1.22 or newer
- RocksDB development libraries
- Required system dependencies (snappy, lz4, zstd, bzip2)

### Building the MCP Server

```bash
# Clone and build
cd rocksdb-cli
go build -o rocksdb-mcp-server ./cmd/mcp-server

# Or build with CGO flags for RocksDB
CGO_CFLAGS="-I/opt/homebrew/Cellar/rocksdb/*/include" \
CGO_LDFLAGS="-L/opt/homebrew/Cellar/rocksdb/*/lib -L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd" \
go build -o rocksdb-mcp-server ./cmd/mcp-server
```

## Configuration

### Configuration File

Create a YAML or JSON configuration file:

```yaml
# config.yaml
name: "RocksDB MCP Server"
version: "1.0.0"
description: "MCP server for RocksDB operations"

database_path: "./data/rocksdb"
read_only: false

transport:
  type: "stdio"  # stdio, tcp, websocket, unix
  host: "localhost"
  port: 8080
  timeout: 30s

max_concurrent_sessions: 10
session_timeout: 5m

enable_all_tools: true
enabled_tools: []
disabled_tools: []
enable_resources: true

log_level: "info"
```

### Environment Variables

Set required environment variables for RocksDB:

```bash
export CGO_CFLAGS="-I/opt/homebrew/Cellar/rocksdb/*/include"
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/rocksdb/*/lib -L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"
```

## Usage

### Starting the MCP Server

#### STDIO Mode (Default)
```bash
./rocksdb-mcp-server --db ./data/rocksdb
```

#### With Configuration File
```bash
./rocksdb-mcp-server --config config.yaml
```

#### Read-Only Mode
```bash
./rocksdb-mcp-server --db ./data/rocksdb --readonly
```

#### Different Transport Types
```bash
# TCP transport
./rocksdb-mcp-server --db ./data/rocksdb --transport tcp --port 8080

# WebSocket transport  
./rocksdb-mcp-server --db ./data/rocksdb --transport websocket --port 8080

# Unix socket transport
./rocksdb-mcp-server --db ./data/rocksdb --transport unix --socket /tmp/rocksdb-mcp.sock
```

### Command Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `--config` | Path to configuration file | - |
| `--db` | Path to RocksDB database | - |
| `--readonly` | Open database in read-only mode | false |
| `--transport` | Transport type (stdio, tcp, websocket, unix) | stdio |
| `--host` | Host for TCP/WebSocket transport | localhost |
| `--port` | Port for TCP/WebSocket transport | 8080 |
| `--socket` | Unix socket path | /tmp/rocksdb-mcp.sock |

## MCP Client Integration

### Claude Desktop Integration

Add to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "rocksdb": {
      "command": "/path/to/rocksdb-mcp-server",
      "args": ["--db", "/path/to/your/rocksdb", "--readonly"],
      "env": {
        "CGO_CFLAGS": "-I/opt/homebrew/Cellar/rocksdb/*/include",
        "CGO_LDFLAGS": "-L/opt/homebrew/Cellar/rocksdb/*/lib -L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"
      }
    }
  }
}
```

### Example AI Interactions

Once integrated, you can interact with your RocksDB through natural language:

```
"What column families are available in my database?"
"Get the value for key 'user:123' from the users column family"
"Show me all keys that start with 'product:' from the catalog column family"
"Export the user data to a CSV file"
"What's the latest entry in the logs column family?"
```

## API Reference

### Tool Parameters

#### rocksdb_get
- `key` (required): The key to retrieve
- `column_family` (optional): Column family name (default: "default")
- `pretty` (optional): Pretty-print JSON output (default: false)

#### rocksdb_put
- `key` (required): The key to store
- `value` (required): The value to store
- `column_family` (optional): Column family name (default: "default")

#### rocksdb_scan
- `column_family` (optional): Column family name (default: "default")
- `start_key` (optional): Start key for range scan
- `end_key` (optional): End key for range scan
- `limit` (optional): Maximum number of results
- `reverse` (optional): Scan in reverse order (default: false)
- `values_only` (optional): Return only values (default: false)

#### rocksdb_prefix_scan
- `prefix` (required): Key prefix to search for
- `column_family` (optional): Column family name (default: "default")
- `limit` (optional): Maximum number of results

### Error Handling

The MCP server provides detailed error information:

- **Key not found**: When requested key doesn't exist
- **Column family not found**: When specified CF doesn't exist
- **Read-only mode**: When write operations are attempted in read-only mode
- **Invalid parameters**: When required parameters are missing or invalid

## Security Considerations

### Read-Only Mode

For production use with AI assistants, enable read-only mode:

```bash
./rocksdb-mcp-server --db ./data/rocksdb --readonly
```

This prevents any write operations while allowing safe data exploration.

### Access Control

Configure specific tools to limit AI capabilities:

```yaml
enable_all_tools: false
enabled_tools:
  - "rocksdb_get"
  - "rocksdb_scan"
  - "rocksdb_prefix_scan"
  - "rocksdb_list_column_families"
disabled_tools:
  - "rocksdb_put"
  - "rocksdb_create_column_family"
  - "rocksdb_drop_column_family"
```

## Testing

### Running Tests

```bash
# Run all MCP tests
go test ./internal/mcp/... -v

# Run with coverage
go test ./internal/mcp/... -v -cover

# Run specific test
go test ./internal/mcp/... -run TestConfigValidation -v
```

### Test Coverage

The test suite covers:

- Configuration loading and validation
- Tool manager functionality
- Database operation mocking
- Error handling scenarios
- Read-only mode enforcement

## Troubleshooting

### Common Issues

#### 1. CGO Compilation Errors
```bash
# Ensure correct environment variables
export CGO_CFLAGS="-I/opt/homebrew/Cellar/rocksdb/*/include"
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/rocksdb/*/lib -lrocksdb -lstdc++ -lm -lz"
```

#### 2. Database Not Found
```bash
# Create database directory
mkdir -p ./data/rocksdb

# Or use existing CLI to create test data
go run scripts/gen_testdb.go ./data/rocksdb
```

#### 3. Permission Issues
```bash
# Check database directory permissions
chmod 755 ./data/rocksdb
```

#### 4. Port Already in Use (TCP mode)
```bash
# Use different port
./rocksdb-mcp-server --db ./data/rocksdb --transport tcp --port 8081
```

### Debug Mode

Enable verbose logging for troubleshooting:

```yaml
log_level: "debug"
```

## Performance Considerations

### Memory Usage

- Each session maintains its own read options
- Large scan operations are limited by configuration
- Consider session timeout for memory management

### Concurrency

- Multiple concurrent sessions supported
- Configure `max_concurrent_sessions` based on system resources
- Read-only mode allows unlimited concurrent access

### Network Transport

- STDIO: Lowest latency, direct process communication
- TCP: Network access, higher latency but more flexible
- WebSocket: Web browser compatibility
- Unix Socket: Local high-performance IPC

## Contributing

### Adding New Tools

1. Define tool in `internal/mcp/tools.go`
2. Implement handler function
3. Register tool in `RegisterTools()`
4. Add tests in `internal/mcp/tools_test.go`

### Adding New Prompts

1. Define prompt in `internal/mcp/prompts.go`
2. Implement handler function
3. Register prompt in `RegisterPrompts()`
4. Add documentation

## License

This MCP server integration follows the same license as the main RocksDB CLI project.

## Support

For issues and questions:

1. Check this documentation
2. Review test cases for examples
3. Check the main RocksDB CLI README
4. Submit issues with detailed error messages and configuration

---

**Note**: This MCP server integration extends the existing RocksDB CLI tool with AI-friendly interfaces. Ensure you understand the security implications before using in production environments. 