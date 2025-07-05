# RocksDB CLI - Complete Help Guide

## Overview

RocksDB CLI is a powerful command-line tool for interacting with RocksDB databases. It offers three main modes of operation:

1. **Traditional CLI** - Direct command-line operations and interactive REPL
2. **GraphChain Agent** - AI-powered natural language queries 🤖
3. **MCP Server** - Integration with AI assistants like Claude Desktop 🔗

## Quick Reference

### Starting the Application

```bash
# Interactive REPL mode
rocksdb-cli --db /path/to/database

# Read-only mode (safe for production)
rocksdb-cli --db /path/to/database --read-only

# AI-powered GraphChain Agent
rocksdb-cli --db /path/to/database --graphchain

# Help and version
rocksdb-cli --help
```

### Basic Operations

| Operation | Command | Example |
|-----------|---------|---------|
| **Get key** | `get [<cf>] <key> [--pretty]` | `get users user:1001 --pretty` |
| **Put data** | `put [<cf>] <key> <value>` | `put users user:1001 {"name":"Alice"}` |
| **Scan range** | `scan [<cf>] [start] [end]` | `scan users user:1000 user:2000` |
| **Prefix search** | `prefix [<cf>] <prefix>` | `prefix users user:` |
| **JSON query** | `jsonquery [<cf>] <field> <value>` | `jsonquery users name Alice` |
| **Export CSV** | `export [<cf>] <file> [--sep=<sep>]` | `export users /tmp/users.csv --sep=";"` |

## Interactive Commands Reference

### Column Family Management
```bash
usecf <cf>                   # Switch to column family
listcf                       # List all column families
createcf <cf>                # Create new column family
dropcf <cf>                  # Drop column family (warning: destructive!)
```

### Data Operations
```bash
# Basic operations
get [<cf>] <key> [--pretty]             # Get value by key
put [<cf>] <key> <value>                # Insert/update key-value pair
last [<cf>] [--pretty]                  # Get last entry in column family

# Search operations
prefix [<cf>] <prefix> [--pretty]       # Find keys with prefix
scan [<cf>] [start] [end] [options]     # Scan key range
jsonquery [<cf>] <field> <value> [--pretty]  # Query JSON field
search [<cf>] [options]                 # Fuzzy search

# Utility operations
export [<cf>] <file_path> [--sep=<sep>]   # Export to CSV (default separator: ,)
                                         # Use --sep=";" for semicolon, --sep="\\t" for tab

# Examples:
export users users.csv --sep=";"
export logs logs.tsv --sep="\\t"

stats [<cf>] [--detailed] [--pretty]   # Show statistics
help                                    # Show help
exit/quit                               # Exit CLI
```

### Command Line Operations
```bash
# Direct operations (non-interactive)
rocksdb-cli --db <path> --last <cf> [--pretty]
rocksdb-cli --db <path> --prefix <cf> --prefix-key <prefix> [--pretty]
rocksdb-cli --db <path> --scan <cf> [--start <key>] [--end <key>] [options]
rocksdb-cli --db <path> --export-cf <cf> --export-file <file>

# Real-time monitoring
rocksdb-cli --db <path> --watch <cf> [--interval <duration>]

# Search operations
rocksdb-cli --db <path> --search <cf> --search-key <pattern> [options]
rocksdb-cli --db <path> --search <cf> --search-value <pattern> [options]
```

## GraphChain Agent (AI-Powered) 🤖

### Starting GraphChain
```bash
# Basic startup
rocksdb-cli --db /path/to/database --graphchain

# With custom configuration
rocksdb-cli --db /path/to/database --graphchain --config custom.yaml
```

### Natural Language Examples

**Database Exploration:**
- "Show me all column families"
- "What's in the users table?"
- "How many keys are in the database?"
- "Give me database statistics"

**Data Queries:**
- "Find all users named Alice"
- "Show me the last 10 entries in logs"
- "Get all keys starting with 'user:'"
- "Find JSON records where age > 30"

**Complex Operations:**
- "Export users to CSV file"
- "Find products with category 'electronics'"
- "Show me error logs from today"
- "Compare sizes of different column families"

### GraphChain Configuration

Create `config/graphchain.yaml`:

```yaml
graphchain:
  llm:
    provider: "ollama"          # ollama, openai, googleai
    model: "llama2"             # Model name
    api_key: "${API_KEY}"       # API key (not needed for Ollama)
    base_url: "http://localhost:11434"
    timeout: "30s"
  
  agent:
    max_iterations: 10
    tool_timeout: "10s"
    enable_memory: true
    memory_size: 100
  
  security:
    enable_audit: true
    read_only_mode: false
    allowed_operations: ["get", "scan", "prefix", "jsonquery"]
```

### LLM Provider Setup

**Ollama (Recommended for local use):**
```bash
# Install and start Ollama
curl -fsSL https://ollama.ai/install.sh | sh
ollama serve

# Pull a model
ollama pull llama2
```

**OpenAI:**
```bash
export OPENAI_API_KEY="your-api-key"
```

**Google AI:**
```bash
export GOOGLE_AI_API_KEY="your-api-key"
```

## MCP Server Integration 🔗

### Starting MCP Server
```bash
# Basic startup (read-only mode)
./cmd/mcp-server/rocksdb-mcp-server --db /path/to/database

# With configuration file
./cmd/mcp-server/rocksdb-mcp-server --config config/mcp-server.yaml
```

### Claude Desktop Integration

1. **Add to `claude_desktop_config.json`:**
```json
{
  "mcpServers": {
    "rocksdb": {
      "command": "/path/to/rocksdb-mcp-server",
      "args": ["--db", "/path/to/database"],
      "env": {
        "MCP_LOG_LEVEL": "info"
      }
    }
  }
}
```

2. **Restart Claude Desktop**

3. **Use natural language with Claude:**
   - "Show me users in the RocksDB database"
   - "Export the products table to CSV"
   - "Find all logs with error level"

### MCP Tools Available
- `rocksdb_get` - Get value by key
- `rocksdb_scan` - Scan key ranges
- `rocksdb_prefix_scan` - Prefix scanning
- `rocksdb_list_column_families` - List CFs
- `rocksdb_get_last` - Get last entry
- `rocksdb_json_query` - Query JSON fields
- `rocksdb_export_to_csv` - Export to CSV

## Advanced Features

### Search Command Options
```bash
search [<cf>] [options]

Options:
  --key=<pattern>       Search in key names
  --value=<pattern>     Search in values
  --regex               Use regex patterns
  --case-sensitive      Case sensitive search
  --limit=N             Limit results
  --keys-only           Show only keys
  --pretty              Pretty format JSON
```

**Search Examples:**
```bash
search --key=user:*                    # Keys starting with 'user:'
search users --value=Alice             # Values containing 'Alice'
search --key=*temp* --value=*active*   # Both key and value patterns
search --key=user:[0-9]+ --regex       # Regex pattern
search --value=error --limit=10         # Limit results
```

### Scan Command Options
```bash
scan [<cf>] [start] [end] [options]

Options:
  --limit=N        Limit number of results
  --reverse        Scan in reverse order
  --values=no      Show only keys
  --timestamp      Show timestamp interpretation
```

**Scan Examples:**
```bash
scan users                           # Scan all users
scan users user:1000 user:2000       # Range scan
scan users * * --limit=10            # First 10 entries
scan users user:9999 * --reverse     # Reverse from key
scan logs --timestamp                # Show timestamps
```

### JSON Query Features
```bash
jsonquery [<cf>] <field> <value> [--pretty]
```

**Supported field types:**
- **String**: `jsonquery users name "Alice"`
- **Number**: `jsonquery users age 30`
- **Boolean**: `jsonquery users active true`
- **Null**: `jsonquery users deleted null`

### Statistics Command
```bash
stats [<cf>] [--detailed] [--pretty]
```

**Examples:**
```bash
stats                    # Database overview
stats users              # Users column family stats
stats --detailed         # Detailed database stats
stats users --pretty     # JSON format output
```

### Real-time Monitoring
```bash
rocksdb-cli --db <path> --watch <cf> [--interval <duration>]
```

**Examples:**
```bash
rocksdb-cli --db ./db --watch logs
rocksdb-cli --db ./db --watch users --interval 500ms
rocksdb-cli --db ./db --watch metrics --interval 5s
```

## Best Practices

### Security Considerations
1. **Use read-only mode** for production databases:
   ```bash
   rocksdb-cli --db /prod/db --read-only
   ```

2. **Enable audit logging** for GraphChain/MCP:
   ```yaml
   security:
     enable_audit: true
   ```

3. **Restrict operations** in config files:
   ```yaml
   allowed_operations: ["get", "scan", "prefix", "jsonquery"]
   ```

### Performance Tips
1. **Use limits** for large datasets:
   ```bash
   scan users --limit=1000
   prefix users user: --limit=100
   ```

2. **Use specific ranges** instead of full scans:
   ```bash
   scan users user:1000 user:2000    # Better than: scan users
   ```

3. **Enable read-only mode** for better concurrent access

### Data Management
1. **Regular exports** for backups:
   ```bash
   export users /backup/users-$(date +%Y%m%d).csv
   ```

2. **Monitor database statistics**:
   ```bash
   stats --detailed
   ```

3. **Use appropriate column families** for data organization

## Troubleshooting

### Common Issues

**1. Database Access Errors**
```bash
# Check database path and permissions
ls -la /path/to/database
rocksdb-cli --db /path/to/database --read-only
```

**2. GraphChain Connection Issues**
```bash
# Check Ollama status
curl http://localhost:11434/api/tags
ollama list

# Check logs
tail -f graphchain_audit.log
```

**3. MCP Server Problems**
```bash
# Test MCP server directly
./rocksdb-mcp-server --db ./testdb --debug

# Check Claude Desktop logs
tail -f ~/.claude/logs/claude_desktop.log
```

**4. Performance Issues**
- Use `--limit` for large scans
- Enable read-only mode for better concurrency
- Use specific key ranges instead of full scans

### Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| "Key not found" | Key doesn't exist | Check key name and column family |
| "Column family not found" | CF doesn't exist | Use `listcf` to see available CFs |
| "Read-only mode" | Write attempted in RO mode | Remove `--read-only` flag |
| "Database closed" | DB connection lost | Restart CLI |

## Examples and Use Cases

### Data Exploration
```bash
# Discover database structure
listcf
stats --detailed

# Explore data patterns
prefix users user:
jsonquery users active true
scan logs --timestamp --limit=10
```

### Data Analysis
```bash
# Find patterns in keys
search --key=*:session:* --limit=100
search --key=cache:* --value=*expired*

# Analyze JSON data
jsonquery products category "electronics"
jsonquery users age 25 --pretty
```

### Data Export and Backup
```bash
# Export specific data
export users /backup/users.csv
export logs /backup/logs-$(date +%Y%m%d).csv

# Conditional exports using search
search logs --value=ERROR --limit=1000 > errors.txt
```

### Monitoring and Debugging
```bash
# Real-time monitoring
rocksdb-cli --db ./db --watch events --interval 1s

# Debug with timestamps
scan logs --timestamp --reverse --limit=20

# Error investigation
jsonquery logs level ERROR --pretty
search logs --value=exception --limit=50
```

## Configuration Files

### GraphChain Configuration (`config/graphchain.yaml`)
```yaml
graphchain:
  llm:
    provider: "ollama"
    model: "llama2"
    base_url: "http://localhost:11434"
    timeout: "30s"
  agent:
    max_iterations: 10
    enable_memory: true
    memory_size: 100
  security:
    enable_audit: true
    read_only_mode: false
  context:
    enable_auto_discovery: true
    update_interval: "5m"
```

### MCP Server Configuration (`config/mcp-server.yaml`)
```yaml
name: "RocksDB MCP Server"
version: "1.0.0"
database_path: "./testdb"
read_only: true
transport:
  type: "stdio"
  timeout: 30s
log_level: "info"
```

## Additional Resources

- **Project Repository**: https://github.com/rocksdb-cli
- **RocksDB Documentation**: https://rocksdb.org/
- **Ollama Models**: https://ollama.ai/library
- **MCP Protocol**: https://modelcontextprotocol.io/
- **LangChain Go**: https://github.com/tmc/langchaingo

## Support

For issues and questions:
1. Check this help document
2. Review the main README.md
3. Check existing GitHub issues
4. Create a new issue with detailed reproduction steps

---

*This help document covers RocksDB CLI v1.0+. For older versions, some features may not be available.* 

[Note] For binary/uint64 keys, prefix only matches byte prefixes, not numeric string prefixes. Example: `prefix 0x00` matches all keys starting with byte 0x00; `prefix 123` only matches the key with value 123 as 8-byte big-endian. 