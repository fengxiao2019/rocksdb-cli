#!/bin/bash

# RocksDB MCP Server Startup Script
# This script starts the MCP server with proper environment configuration
# for integration with Cursor and other MCP clients.

set -e  # Exit on any error

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Default configuration
DEFAULT_DB_PATH="./testdb"
DEFAULT_CONFIG="./config/mcp-server.yaml"
BINARY_NAME="rocksdb-mcp-server"

# Check if the binary exists
if [ ! -f "$BINARY_NAME" ]; then
    echo "Error: MCP server binary '$BINARY_NAME' not found in $SCRIPT_DIR"
    echo "Please build the server first with: go build -o $BINARY_NAME ./cmd/mcp-server"
    exit 1
fi

# Check if testdb exists
if [ ! -d "$DEFAULT_DB_PATH" ]; then
    echo "Warning: Test database '$DEFAULT_DB_PATH' not found"
    echo "You may need to generate test data first with: go run scripts/gen_testdb.go $DEFAULT_DB_PATH"
fi

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --config PATH     Use specific config file (default: $DEFAULT_CONFIG)"
    echo "  --db PATH         Use specific database path (default: $DEFAULT_DB_PATH)"
    echo "  --read-write      Start in read-write mode (default: read-only)"
    echo "  --help            Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  MCP_DB_PATH       Database path override"
    echo "  MCP_CONFIG        Config file path override"
    echo "  MCP_READ_WRITE    Set to 'true' for read-write mode"
    echo ""
    echo "Examples:"
    echo "  $0                           # Start with defaults (read-only)"
    echo "  $0 --db ./mydb               # Use custom database"
    echo "  $0 --read-write              # Enable write operations"
    echo "  $0 --config ./my-config.yaml # Use custom config"
}

# Parse command line arguments
DB_PATH="${MCP_DB_PATH:-$DEFAULT_DB_PATH}"
CONFIG_FILE="${MCP_CONFIG:-$DEFAULT_CONFIG}"
READ_ONLY=true  # Default to read-only for safety

while [[ $# -gt 0 ]]; do
    case $1 in
        --config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        --db)
            DB_PATH="$2"
            shift 2
            ;;
        --read-write)
            READ_ONLY=false
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Override read-only setting from environment
if [ "${MCP_READ_WRITE}" = "true" ]; then
    READ_ONLY=false
fi

# Build the command arguments
ARGS=()

# Add database path
ARGS+=("--db" "$DB_PATH")

# Add read-only flag if enabled
if [ "$READ_ONLY" = true ]; then
    ARGS+=("--readonly")
fi

# Add config file if it exists
if [ -f "$CONFIG_FILE" ]; then
    ARGS+=("--config" "$CONFIG_FILE")
else
    echo "Warning: Config file '$CONFIG_FILE' not found, using defaults"
fi

# Log startup information
echo "Starting RocksDB MCP Server..."
echo "  Database: $DB_PATH"
echo "  Config: $CONFIG_FILE"
echo "  Mode: $([ "$READ_ONLY" = true ] && echo "read-only" || echo "read-write")"
echo "  Binary: $BINARY_NAME"
echo ""

# Start the MCP server
exec "./$BINARY_NAME" "${ARGS[@]}" 