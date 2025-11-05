#!/bin/bash
# Build script for minimal rocksdb-cli (without Web UI)
# This significantly reduces build time and binary size

set -e  # Exit on error

APP_NAME="rocksdb-cli-minimal"
VERSION="${VERSION:-v1.0.0}"
BUILD_DIR="build"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================"
echo "Building ${APP_NAME} ${VERSION}"
echo "Minimal build: Web UI DISABLED"
echo "========================================"
echo

# Create build directory
mkdir -p ${BUILD_DIR}

# Detect platform
GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

echo "Platform: ${GOOS}/${GOARCH}"
echo

# Set CGO environment
export CGO_ENABLED=1

# Build with minimal tag
echo "Building ${APP_NAME}..."
go build -tags=minimal \
         -ldflags "-X main.version=${VERSION}" \
         -o ${BUILD_DIR}/${APP_NAME}-${GOOS}-${GOARCH} \
         ./cmd

if [ $? -eq 0 ]; then
    echo
    echo -e "${GREEN}========================================"
    echo "Build completed successfully!"
    echo -e "========================================${NC}"
    echo
    echo "Executable: ${BUILD_DIR}/${APP_NAME}-${GOOS}-${GOARCH}"
    echo

    # Show file size
    if command -v du &> /dev/null; then
        SIZE=$(du -h ${BUILD_DIR}/${APP_NAME}-${GOOS}-${GOARCH} | cut -f1)
        echo "Size: ${SIZE}"
    fi

    echo
    echo -e "${YELLOW}Note: Web UI is NOT available in this build.${NC}"
    echo "To enable Web UI, use 'make build' instead."
    echo
else
    echo
    echo -e "${RED}========================================"
    echo "Build FAILED!"
    echo -e "========================================${NC}"
    echo
    echo "Make sure:"
    echo "  1. RocksDB C++ libraries are installed"
    echo "  2. CGO is properly configured"
    echo "  3. Go 1.24+ is installed"
    echo
    echo "See BUILD.md for installation instructions."
    exit 1
fi
