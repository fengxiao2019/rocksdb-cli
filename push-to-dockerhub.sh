#!/bin/bash
# Script to tag and push rocksdb-base image to Docker Hub
#
# Usage:
#   1. First login: docker login
#   2. Run this script: bash push-to-dockerhub.sh YOUR_DOCKERHUB_USERNAME

set -e

if [ -z "$1" ]; then
    echo "Error: Please provide your Docker Hub username"
    echo "Usage: bash push-to-dockerhub.sh YOUR_DOCKERHUB_USERNAME"
    exit 1
fi

DOCKERHUB_USERNAME="$1"
IMAGE_NAME="rocksdb-base"
VERSION="10.2.1"

echo "=========================================="
echo "Tagging and pushing to Docker Hub"
echo "=========================================="
echo "Username: $DOCKERHUB_USERNAME"
echo "Image: $IMAGE_NAME"
echo "Version: $VERSION"
echo "=========================================="

# Tag the image
echo "Tagging image..."
docker tag ${IMAGE_NAME}:${VERSION} ${DOCKERHUB_USERNAME}/${IMAGE_NAME}:${VERSION}
docker tag ${IMAGE_NAME}:${VERSION} ${DOCKERHUB_USERNAME}/${IMAGE_NAME}:latest

echo "✓ Tagged as ${DOCKERHUB_USERNAME}/${IMAGE_NAME}:${VERSION}"
echo "✓ Tagged as ${DOCKERHUB_USERNAME}/${IMAGE_NAME}:latest"

# Push the images
echo ""
echo "Pushing to Docker Hub..."
docker push ${DOCKERHUB_USERNAME}/${IMAGE_NAME}:${VERSION}
docker push ${DOCKERHUB_USERNAME}/${IMAGE_NAME}:latest

echo ""
echo "=========================================="
echo "✓ Successfully pushed to Docker Hub!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Update Dockerfile.quick line 10 to:"
echo "   FROM ${DOCKERHUB_USERNAME}/${IMAGE_NAME}:${VERSION} AS builder"
echo ""
echo "2. Build your application quickly with:"
echo "   docker build -f Dockerfile.quick -t rocksdb-cli ."
echo ""
