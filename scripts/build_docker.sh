#!/bin/bash

# Docker build script for rocksdb-cli
# This script handles proxy issues and builds the Docker image

set -e

echo "🐳 Building RocksDB CLI Docker image..."

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "❌ Error: Docker is not running"
    exit 1
fi

# Clear any proxy settings that might interfere
export HTTP_PROXY=""
export HTTPS_PROXY=""
export http_proxy=""
export https_proxy=""
export no_proxy="*"

# Build image with proxy bypass
echo "📦 Building Docker image (this may take a few minutes)..."

# Try Alpine first (faster), fallback to Ubuntu if it fails
if docker build \
    --build-arg HTTP_PROXY="" \
    --build-arg HTTPS_PROXY="" \
    --build-arg http_proxy="" \
    --build-arg https_proxy="" \
    --build-arg no_proxy="*" \
    -f docker/Dockerfile.manual \
    -t rocksdb-cli:latest \
    . ; then
    echo "✅ Successfully built rocksdb-cli:latest"
else
    echo "❌ Both builds failed. Please check the error messages above."
    exit 1
fi

# Test the built image
echo "🧪 Testing the built image..."
if docker run --rm rocksdb-cli:latest --help >/dev/null 2>&1; then
    echo "✅ Docker image test passed!"
    echo ""
    echo "🎉 Docker image 'rocksdb-cli:latest' is ready to use!"
    echo ""
    echo "Usage examples:"
    echo "  # Run with a database"
    echo "  docker run -it --rm -v \"/path/to/your/db:/data\" rocksdb-cli:latest --db /data"
    echo ""
    echo "  # Get help"
    echo "  docker run --rm rocksdb-cli:latest --help"
else
    echo "❌ Docker image test failed"
    exit 1
fi 