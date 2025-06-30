# RocksDB MCP Server Startup Script for Windows PowerShell
# This script starts the MCP server with proper environment configuration
# for integration with Cursor and other MCP clients on Windows.

param(
    [string]$Config = "",
    [string]$Database = "",
    [switch]$ReadWrite = $false,
    [switch]$Help = $false
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Get the directory where this script is located
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

# No RocksDB environment setup needed - using statically compiled binary

# Default configuration
$DefaultDbPath = ".\testdb"
$DefaultConfig = ".\config\mcp-server.yaml"
$BinaryName = "rocksdb-mcp-server.exe"

# Function to show usage
function Show-Usage {
    Write-Host "Usage: .\start-mcp-server.ps1 [OPTIONS]" -ForegroundColor Green
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Yellow
    Write-Host "  -Config PATH      Use specific config file (default: $DefaultConfig)"
    Write-Host "  -Database PATH    Use specific database path (default: $DefaultDbPath)"
    Write-Host "  -ReadWrite        Start in read-write mode (default: read-only)"
    Write-Host "  -Help             Show this help message"
    Write-Host ""
    Write-Host "Environment Variables:" -ForegroundColor Yellow
    Write-Host "  MCP_DB_PATH       Database path override"
    Write-Host "  MCP_CONFIG        Config file path override"
    Write-Host "  MCP_READ_WRITE    Set to 'true' for read-write mode"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Cyan
    Write-Host "  .\start-mcp-server.ps1                              # Start with defaults (read-only)"
    Write-Host "  .\start-mcp-server.ps1 -Database .\mydb            # Use custom database"
    Write-Host "  .\start-mcp-server.ps1 -ReadWrite                  # Enable write operations"
    Write-Host "  .\start-mcp-server.ps1 -Config .\my-config.yaml    # Use custom config"
    Write-Host ""
    Write-Host "Windows Prerequisites:" -ForegroundColor Yellow
    Write-Host "  - RocksDB libraries installed (via vcpkg or manual installation)"
    Write-Host "  - Visual C++ Redistributable packages"
    Write-Host "  - .NET runtime (if required by the application)"
}

# Show help if requested
if ($Help) {
    Show-Usage
    exit 0
}

# Check if the binary exists
if (-not (Test-Path $BinaryName)) {
    Write-Error "Error: MCP server binary '$BinaryName' not found in $ScriptDir"
    Write-Host "Please ensure the binary is in the same directory as this script." -ForegroundColor Red
    Write-Host "Expected file: $ScriptDir\$BinaryName" -ForegroundColor Yellow
    exit 1
}

# Override with environment variables if set
$DbPath = if ($env:MCP_DB_PATH) { $env:MCP_DB_PATH } 
          elseif ($Database) { $Database } 
          else { $DefaultDbPath }

$ConfigFile = if ($env:MCP_CONFIG) { $env:MCP_CONFIG }
              elseif ($Config) { $Config }
              else { $DefaultConfig }

$IsReadOnly = -not $ReadWrite
if ($env:MCP_READ_WRITE -eq "true") {
    $IsReadOnly = $false
}

# Check if testdb exists
if (-not (Test-Path $DbPath)) {
    Write-Warning "Database path '$DbPath' not found"
    Write-Host "You may need to:" -ForegroundColor Yellow
    Write-Host "  1. Generate test data first: go run scripts\gen_testdb.go $DbPath" -ForegroundColor Gray
    Write-Host "  2. Or specify an existing database path with -Database parameter" -ForegroundColor Gray
}

# Build the command arguments
$Args = @()

# Add database path
$Args += "--db"
$Args += $DbPath

# Add read-only flag if enabled
if ($IsReadOnly) {
    $Args += "--readonly"
}

# Add config file if it exists
if (Test-Path $ConfigFile) {
    $Args += "--config"
    $Args += $ConfigFile
} else {
    Write-Warning "Config file '$ConfigFile' not found, using defaults"
}

# Log startup information
Write-Host "Starting RocksDB MCP Server..." -ForegroundColor Green
Write-Host "  Binary: $BinaryName" -ForegroundColor Gray
Write-Host "  Database: $DbPath" -ForegroundColor Gray
Write-Host "  Config: $ConfigFile" -ForegroundColor Gray
Write-Host "  Mode: $(if ($IsReadOnly) { 'read-only' } else { 'read-write' })" -ForegroundColor Gray
Write-Host "  Script Location: $ScriptDir" -ForegroundColor Gray
Write-Host ""


try {
    # Start the MCP server
    Write-Host "Executing: .\$BinaryName $($Args -join ' ')" -ForegroundColor Cyan
    & ".\$BinaryName" @Args
}
catch {
    Write-Error "Failed to start MCP server: $_"
    Write-Host ""
    Write-Host "Troubleshooting tips:" -ForegroundColor Yellow
    Write-Host "  3. Verify the database path exists and is accessible" -ForegroundColor Gray
    Write-Host "  4. Run PowerShell as Administrator if needed" -ForegroundColor Gray
    Write-Host "  5. Check Windows Event Viewer for detailed error information" -ForegroundColor Gray
    exit 1
} 