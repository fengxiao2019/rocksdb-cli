# Makefile for rocksdb-cli

APP_NAME := rocksdb-cli
VERSION := v1.0.0
BUILD_DIR := build

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build:
	@echo "Building $(APP_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd

# Build for current platform only (CGO cross-compilation is complex)
.PHONY: build-native
build-native:
	@echo "Building $(APP_NAME) $(VERSION) for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd
	@echo "Build completed! Executable: $(BUILD_DIR)/$(APP_NAME)"
	@ls -la $(BUILD_DIR)/$(APP_NAME)

# Build minimal version (without Web UI)
.PHONY: build-minimal
build-minimal:
	@echo "Building $(APP_NAME)-minimal (without Web UI)..."
	@mkdir -p $(BUILD_DIR)
	@chmod +x scripts/build-minimal.sh
	@./scripts/build-minimal.sh

# Build using Docker for Linux
.PHONY: build-linux-docker
build-linux-docker:
	@echo "Building $(APP_NAME) for Linux using Docker..."
	@chmod +x scripts/build_docker.sh
	@./scripts/build_docker.sh


# Clean build directory
.PHONY: clean
clean:
	@echo "Cleaning build directory..."
	rm -rf $(BUILD_DIR)

# Test
.PHONY: test
test:
	go test ./...

# Test with coverage
.PHONY: test-coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Run locally
.PHONY: run
run:
	go run ./cmd --db ./testdb

# Generate test database
.PHONY: gen-testdb
gen-testdb:
	go run scripts/gen_testdb.go ./testdb

# Create release package
.PHONY: release
release:
	@chmod +x scripts/release.sh
	@./scripts/release.sh

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build              - Build for current platform (with Web UI)"
	@echo "  build-minimal      - Build minimal version (without Web UI, faster)"
	@echo "  build-native       - Build for current platform (alias for build)"
	@echo "  build-linux-docker - Build for Linux using Docker"
	@echo "  clean              - Clean build directory"
	@echo "  test               - Run tests"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  deps               - Install dependencies"
	@echo "  run                - Run locally with testdb"
	@echo "  gen-testdb         - Generate test database"
	@echo "  release            - Create release package"
	@echo "  help               - Show this help" 