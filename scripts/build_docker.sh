#!/bin/bash

# Build Linux executable using Docker

set -e

APP_NAME="rocksdb-cli"
VERSION=${VERSION:-"v1.0.0"}
BUILD_DIR="build"

echo "Building $APP_NAME $VERSION for Linux using Docker..."

# Create build directory
mkdir -p $BUILD_DIR

# Build Docker image and extract executable
docker build -f docker/Dockerfile.linux -t $APP_NAME-builder .

# Create temporary container and copy executable
CONTAINER_ID=$(docker create $APP_NAME-builder)
docker cp $CONTAINER_ID:/workspace/build/rocksdb-cli-linux-amd64 $BUILD_DIR/
docker rm $CONTAINER_ID

echo "Linux build completed!"
echo "Executable: $BUILD_DIR/rocksdb-cli-linux-amd64"
ls -la $BUILD_DIR/rocksdb-cli-linux-amd64 