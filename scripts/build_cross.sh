#!/bin/bash

# Cross-compilation build script for rocksdb-cli
# Note: Cross-compilation with CGO requires target system libraries

set -e

APP_NAME="rocksdb-cli"
VERSION=${VERSION:-"v1.0.0"}
BUILD_DIR="build"

# Create build directory
mkdir -p $BUILD_DIR

echo "Building $APP_NAME $VERSION..."

# Build for current platform (always works)
echo "Building for current platform..."
go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/$APP_NAME ./cmd

echo "Build completed for current platform!"
echo ""
echo "Note: Cross-compilation with CGO (required for RocksDB) is complex."
echo "For other platforms, you should:"
echo "1. Use the target platform to build natively"
echo "2. Use Docker containers with the target OS"
echo "3. Use GitHub Actions for automated cross-platform builds"
echo ""
echo "Current executable: $BUILD_DIR/$APP_NAME"
ls -la $BUILD_DIR/$APP_NAME 