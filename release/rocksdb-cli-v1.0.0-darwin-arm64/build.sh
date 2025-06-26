#!/bin/bash

# Build script for rocksdb-cli
# Generates executables for different platforms

set -e

APP_NAME="rocksdb-cli"
VERSION=${VERSION:-"v1.0.0"}
BUILD_DIR="build"

# Create build directory
mkdir -p $BUILD_DIR

echo "Building $APP_NAME $VERSION..."

# Build for current platform (macOS/Linux)
echo "Building for current platform..."
go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/$APP_NAME ./cmd

# Build for Linux (amd64)
echo "Building for Linux amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/${APP_NAME}-linux-amd64 ./cmd

# Build for Linux (arm64)
echo "Building for Linux arm64..."
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/${APP_NAME}-linux-arm64 ./cmd

# Build for macOS (amd64)
echo "Building for macOS amd64..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/${APP_NAME}-darwin-amd64 ./cmd

# Build for macOS (arm64 - Apple Silicon)
echo "Building for macOS arm64..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/${APP_NAME}-darwin-arm64 ./cmd

echo "Build completed! Executables are in the $BUILD_DIR directory:"
ls -la $BUILD_DIR/

echo ""
echo "Note: Windows builds require special setup due to RocksDB C++ dependencies."
echo "See README.md for Windows build instructions." 