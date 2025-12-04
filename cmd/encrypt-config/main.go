package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"rocksdb-cli/internal/crypto"

	"golang.org/x/term"
)

const (
	defaultKeyFile = ".crypto-key"
)

func main() {
	var (
		generateKey bool
		keyFile     string
		keyUUID     string
		decrypt     bool
		value       string
		interactive bool
	)

	flag.BoolVar(&generateKey, "generate-key", false, "Generate a new encryption key (UUID)")
	flag.StringVar(&keyFile, "key-file", defaultKeyFile, "Path to key file (will be created if -generate-key is used)")
	flag.StringVar(&keyUUID, "key", "", "Encryption key (UUID) - if not provided, will read from key file or prompt")
	flag.BoolVar(&decrypt, "decrypt", false, "Decrypt mode (default is encrypt)")
	flag.StringVar(&value, "value", "", "Value to encrypt/decrypt (if not provided, will read from stdin)")
	flag.BoolVar(&interactive, "interactive", false, "Interactive mode - prompt for multiple values")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Encrypt or decrypt sensitive configuration values using AES-256-GCM.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate new encryption key\n")
		fmt.Fprintf(os.Stderr, "  %s -generate-key\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Encrypt a value interactively\n")
		fmt.Fprintf(os.Stderr, "  %s -value \"my-secret-api-key\"\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Encrypt from stdin (secure input)\n")
		fmt.Fprintf(os.Stderr, "  echo \"my-secret\" | %s\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Decrypt a value\n")
		fmt.Fprintf(os.Stderr, "  %s -decrypt -value \"ENC:AES256:...\"\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Interactive mode\n")
		fmt.Fprintf(os.Stderr, "  %s -interactive\n\n", os.Args[0])
	}

	flag.Parse()

	// Handle key generation
	if generateKey {
		if err := generateAndSaveKey(keyFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Get encryption key
	if keyUUID == "" {
		var err error
		keyUUID, err = getKey(keyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "\nRun with -generate-key to create a new encryption key.\n")
			os.Exit(1)
		}
	}

	// Handle interactive mode
	if interactive {
		if err := interactiveMode(keyUUID, decrypt); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Get value to process
	if value == "" {
		var err error
		value, err = readFromStdin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
	}

	// Process value
	var result string
	var err error

	if decrypt {
		result, err = crypto.Decrypt(value, keyUUID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decrypting: %v\n", err)
			os.Exit(1)
		}
	} else {
		result, err = crypto.Encrypt(value, keyUUID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encrypting: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println(result)
}

func generateAndSaveKey(keyFile string) error {
	// Generate new key
	keyUUID := crypto.GenerateKey()

	// Create directory if needed
	dir := filepath.Dir(keyFile)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Save key to file with secure permissions
	if err := os.WriteFile(keyFile, []byte(keyUUID), 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	fmt.Printf("✓ Generated new encryption key: %s\n", keyUUID)
	fmt.Printf("✓ Saved to: %s\n", keyFile)
	fmt.Printf("\nIMPORTANT: Keep this key secure!\n")
	fmt.Printf("Set environment variable: export ROCKSDB_CRYPTO_KEY=%s\n", keyUUID)
	fmt.Printf("Or set in your shell profile for persistence.\n")

	return nil
}

func getKey(keyFile string) (string, error) {
	// Try to read from environment variable first
	if key := os.Getenv("ROCKSDB_CRYPTO_KEY"); key != "" {
		return key, nil
	}

	// Try to read from key file
	if _, err := os.Stat(keyFile); err == nil {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read key file: %w", err)
		}
		key := strings.TrimSpace(string(data))
		if key != "" {
			return key, nil
		}
	}

	return "", fmt.Errorf("encryption key not found (checked env var ROCKSDB_CRYPTO_KEY and file %s)", keyFile)
}

func readFromStdin() (string, error) {
	// Check if stdin is a terminal (interactive) or pipe
	if term.IsTerminal(int(syscall.Stdin)) {
		fmt.Fprint(os.Stderr, "Enter value (will be hidden): ")
		data, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr) // New line after hidden input
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// Read from pipe
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

func interactiveMode(keyUUID string, decrypt bool) error {
	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Println("Interactive Encryption Mode")
	fmt.Println("---------------------------")
	if decrypt {
		fmt.Println("Mode: DECRYPT")
	} else {
		fmt.Println("Mode: ENCRYPT")
	}
	fmt.Println("Enter values to process (one per line, empty line to exit)")
	fmt.Println()

	for {
		if decrypt {
			fmt.Print("Encrypted value: ")
		} else {
			fmt.Print("Plain value: ")
		}

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		var result string
		var err error

		if decrypt {
			result, err = crypto.Decrypt(line, keyUUID)
		} else {
			result, err = crypto.Encrypt(line, keyUUID)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		if decrypt {
			fmt.Printf("Decrypted: %s\n\n", result)
		} else {
			fmt.Printf("Encrypted: %s\n\n", result)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}


