#!/bin/bash

# Release packaging script for rocksdb-cli

set -e

APP_NAME="rocksdb-cli"
VERSION=${VERSION:-"v1.0.0"}
BUILD_DIR="build"
RELEASE_DIR="release"

echo "Packaging $APP_NAME $VERSION for release..."

# Create release directory
mkdir -p $RELEASE_DIR

# Build for current platform
make build

# Detect current platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

PLATFORM="${OS}-${ARCH}"
RELEASE_NAME="${APP_NAME}-${VERSION}-${PLATFORM}"

echo "Creating release package for $PLATFORM..."

# Create release package directory
PACKAGE_DIR="$RELEASE_DIR/$RELEASE_NAME"
mkdir -p $PACKAGE_DIR

# Copy executable
cp $BUILD_DIR/$APP_NAME $PACKAGE_DIR/

# Copy documentation
cp README.md $PACKAGE_DIR/
cp -r scripts/ $PACKAGE_DIR/

# Create a simple usage guide
cat > $PACKAGE_DIR/USAGE.txt << EOF
RocksDB CLI $VERSION

Quick Start:
1. Ensure RocksDB C++ libraries are installed on your system
2. Run: ./$APP_NAME --db /path/to/your/rocksdb

For detailed installation instructions, see README.md

Examples:
  ./$APP_NAME --db ./testdb                    # Interactive mode
  ./$APP_NAME --db ./testdb --last users       # Get last entry
  ./$APP_NAME --db ./testdb --watch logs       # Watch for new entries

For more information: https://github.com/your-repo/rocksdb-cli
EOF

# Create archive
cd $RELEASE_DIR
tar -czf "${RELEASE_NAME}.tar.gz" "$RELEASE_NAME/"

echo "Release package created: $RELEASE_DIR/${RELEASE_NAME}.tar.gz"
echo "Contents:"
tar -tzf "${RELEASE_NAME}.tar.gz" | head -10

cd ..
echo "Release packaging completed!" 