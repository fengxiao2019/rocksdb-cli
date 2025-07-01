#!/bin/bash
# Script to build and push base Docker image to GitHub Container Registry (GHCR)
# GHCR is free for public repositories

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if GitHub username is provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: GitHub username is required${NC}"
    echo "Usage: $0 <github-username>"
    echo "Example: $0 myusername"
    exit 1
fi

GITHUB_USERNAME=$1
IMAGE_NAME="rocksdb-cli-base"
TAG="latest"
FULL_IMAGE_NAME="ghcr.io/${GITHUB_USERNAME}/${IMAGE_NAME}:${TAG}"

echo -e "${YELLOW}Building base image: ${FULL_IMAGE_NAME}${NC}"
echo "This will take several minutes as RocksDB needs to be compiled..."

# Build the base image for AMD64 (GitHub Actions platform)
docker build -f Dockerfile.base --platform linux/amd64 -t "${FULL_IMAGE_NAME}" .

echo -e "${GREEN}Base image built successfully!${NC}"

# Ask if user wants to push to GHCR
read -p "Do you want to push this image to GitHub Container Registry? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Pushing to GitHub Container Registry...${NC}"
    
    # Check if user is logged in to GHCR
    if ! docker info | grep -q "ghcr.io"; then
        echo -e "${YELLOW}Please log in to GitHub Container Registry first:${NC}"
        echo "1. Create a GitHub Personal Access Token with 'write:packages' scope"
        echo "2. Run: echo \$GITHUB_TOKEN | docker login ghcr.io -u ${GITHUB_USERNAME} --password-stdin"
        read -p "Press Enter after logging in..."
    fi
    
    # Push the image
    docker push "${FULL_IMAGE_NAME}"
    
    echo -e "${GREEN}Image pushed successfully!${NC}"
    echo -e "${YELLOW}Now you can update Dockerfile.test to use:${NC}"
    echo "FROM ${FULL_IMAGE_NAME}"
    
    # Update Dockerfile.test automatically
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s|FROM your-dockerhub-username/rocksdb-cli-base:latest|FROM ${FULL_IMAGE_NAME}|g" Dockerfile.test
    else
        # Linux
        sed -i "s|FROM your-dockerhub-username/rocksdb-cli-base:latest|FROM ${FULL_IMAGE_NAME}|g" Dockerfile.test
    fi
    
    echo -e "${GREEN}Dockerfile.test updated automatically!${NC}"
else
    echo -e "${YELLOW}Skipping push. You can push later with:${NC}"
    echo "docker push ${FULL_IMAGE_NAME}"
fi

echo -e "${GREEN}Done!${NC}"
echo -e "${YELLOW}Note: Make sure your GitHub repository is public for free GHCR usage${NC}" 