// Package repl provides an interactive REPL (Read-Eval-Print Loop) for RocksDB CLI.
package repl

import (
	"os"
	"testing"
	"time"
)

// TestExitHandling tests that the exit handling is thread-safe and doesn't cause panics
func TestExitHandling(t *testing.T) {
	// Test that multiple concurrent exit attempts don't cause panics
	for i := 0; i < 10; i++ {
		go func() {
			exitMutex.Lock()
			if exiting {
				exitMutex.Unlock()
				return
			}
			exiting = true
			exitMutex.Unlock()

			// Reset for next test
			time.Sleep(1 * time.Millisecond)
			exitMutex.Lock()
			exiting = false
			exitMutex.Unlock()
		}()
	}

	// Wait for all goroutines to complete
	time.Sleep(10 * time.Millisecond)

	// Test should complete without panics
	t.Log("Exit handling test completed successfully")
}

// TestWSLDetection tests WSL detection functionality
func TestWSLDetection(t *testing.T) {
	// Save original environment
	originalWSLDistro := os.Getenv("WSL_DISTRO_NAME")
	originalWSLEnv := os.Getenv("WSLENV")

	defer func() {
		// Restore original environment
		os.Setenv("WSL_DISTRO_NAME", originalWSLDistro)
		os.Setenv("WSLENV", originalWSLEnv)
	}()

	// Test WSL detection with WSL_DISTRO_NAME
	os.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	os.Unsetenv("WSLENV")
	if !isWSL() {
		t.Error("Expected isWSL() to return true when WSL_DISTRO_NAME is set")
	}

	// Test WSL detection with WSLENV
	os.Unsetenv("WSL_DISTRO_NAME")
	os.Setenv("WSLENV", "PATH/l")
	if !isWSL() {
		t.Error("Expected isWSL() to return true when WSLENV is set")
	}

	// Test no WSL detection
	os.Unsetenv("WSL_DISTRO_NAME")
	os.Unsetenv("WSLENV")
	if isWSL() {
		t.Error("Expected isWSL() to return false when neither WSL env var is set")
	}
}

// TestWindowsDetection tests Windows OS detection
func TestWindowsDetection(t *testing.T) {
	// This test will return different results on different OS
	// but should not panic
	result := isWindows()
	t.Logf("isWindows() returned: %v", result)
}
