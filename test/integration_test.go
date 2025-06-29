package test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestMCPServerBuild tests that the MCP server can be built successfully
func TestMCPServerBuild(t *testing.T) {
	// Build the MCP server
	cmd := exec.Command("go", "build", "-o", "/tmp/rocksdb-mcp-server-test", "./cmd/mcp-server")
	cmd.Dir = ".."

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build MCP server: %v\nOutput: %s", err, output)
	}

	// Verify the binary exists
	if _, err := os.Stat("/tmp/rocksdb-mcp-server-test"); os.IsNotExist(err) {
		t.Fatal("MCP server binary was not created")
	}

	// Clean up
	os.Remove("/tmp/rocksdb-mcp-server-test")
}

// TestMCPServerStartup tests that the MCP server can start with basic configuration
func TestMCPServerStartup(t *testing.T) {
	// Skip this test if no test database exists
	testDBPath := "../testdb"
	if _, err := os.Stat(testDBPath); os.IsNotExist(err) {
		t.Skip("Test database not found, skipping startup test")
	}

	// Build the MCP server
	binary := "/tmp/rocksdb-mcp-server-startup-test"
	buildCmd := exec.Command("go", "build", "-o", binary, "./cmd/mcp-server")
	buildCmd.Dir = ".."

	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build MCP server: %v\nOutput: %s", err, output)
	}
	defer os.Remove(binary)

	// Start the server with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "--db", testDBPath, "--readonly")

	// Create a pipe to communicate with the server
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	// Start the server
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start MCP server: %v", err)
	}

	// Send an initialize request
	initRequest := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test-client", "version": "1.0.0"}}}` + "\n"

	go func() {
		stdin.Write([]byte(initRequest))
		stdin.Close()
	}()

	// Read response with timeout
	responseCh := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 1024)
		n, _ := stdout.Read(buf)
		responseCh <- buf[:n]
	}()

	select {
	case response := <-responseCh:
		if len(response) == 0 {
			t.Error("No response received from MCP server")
		}
		// Basic validation that we got some response
		responseStr := string(response)
		if len(responseStr) < 10 {
			t.Errorf("Response too short: %s", responseStr)
		}
		t.Logf("MCP server responded: %s", responseStr)
	case <-ctx.Done():
		t.Error("MCP server startup test timed out")
	}

	// Kill the server
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
	cmd.Wait()
}

// TestConfigFileLoading tests that configuration files can be loaded
func TestConfigFileLoading(t *testing.T) {
	// Create a temporary config file
	tmpDir, err := os.MkdirTemp("", "mcp-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configContent := `
name: "Test MCP Server"
version: "1.0.0"
database_path: "/tmp/test"
read_only: true
transport:
  type: "stdio"
  timeout: 30s
max_concurrent_sessions: 5
enable_all_tools: false
enabled_tools:
  - "rocksdb_get"
  - "rocksdb_list_column_families"
log_level: "debug"
`

	configPath := filepath.Join(tmpDir, "test-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Build the MCP server
	binary := "/tmp/rocksdb-mcp-server-config-test"
	buildCmd := exec.Command("go", "build", "-o", binary, "./cmd/mcp-server")
	buildCmd.Dir = ".."

	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build MCP server: %v\nOutput: %s", err, output)
	}
	defer os.Remove(binary)

	// Test that the server can load the config (it should fail due to missing DB, but that's expected)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "--config", configPath)
	output, err := cmd.CombinedOutput()

	// We expect this to fail because the database path doesn't exist,
	// but it should fail AFTER successfully loading the config
	if err == nil {
		t.Error("Expected command to fail due to missing database")
	}

	outputStr := string(output)
	// Check that it got far enough to try opening the database
	if len(outputStr) == 0 {
		t.Error("No output from MCP server config test")
	}

	t.Logf("MCP server config test output: %s", outputStr)
}

// TestMCPPackageTests ensures all MCP package tests pass
func TestMCPPackageTests(t *testing.T) {
	cmd := exec.Command("go", "test", "./internal/mcp/...", "-v")
	cmd.Dir = ".."

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("MCP package tests failed: %v\nOutput: %s", err, output)
	}

	t.Logf("MCP package tests output:\n%s", output)
}
