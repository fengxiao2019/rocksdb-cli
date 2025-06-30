# Docker Setup for RocksDB CLI

This document explains how to use Docker for development and CI/CD with pre-built RocksDB environment.

## Overview

We use a two-stage Docker setup:
1. **Base Image** (`ghcr.io/fengxiao2019/rocksdb-cli-base:latest`) - Contains RocksDB v10.2.1 + Go 1.21
2. **Test Image** (`Dockerfile.test`) - Uses base image + project code for testing

## Benefits

- ✅ **Fast CI**: Avoid recompiling RocksDB (~15-20 min → ~2-3 min)
- ✅ **Consistent Environment**: Same RocksDB version everywhere
- ✅ **Free**: GitHub Container Registry is free for public repos

## Setup Instructions

### 1. Create GitHub Personal Access Token

1. Go to https://github.com/settings/tokens
2. Generate new token (classic) with these permissions:
   - ✅ `write:packages`
   - ✅ `read:packages`

### 2. Login to GitHub Container Registry

```bash
echo "YOUR_GITHUB_TOKEN" | docker login ghcr.io -u fengxiao2019 --password-stdin
```

### 3. Build and Push Base Image

```bash
# This takes 15-20 minutes (one-time setup)
./scripts/build-and-push-ghcr.sh fengxiao2019
```

### 4. Test Locally

```bash
# Build test image
docker build -t rocksdb-cli-test -f Dockerfile.test .

# Run tests
docker run --rm rocksdb-cli-test
```

## CI/CD Integration

The GitHub Actions workflow (`.github/workflows/build.yml`) automatically uses the pre-built base image:

```yaml
- name: Build and test in Docker
  run: |
    docker build -t rocksdb-cli-test -f Dockerfile.test .
    docker run --rm rocksdb-cli-test
```

## Updating Base Image

When you need to update RocksDB version or dependencies:

1. Update `Dockerfile.base`
2. Run `./scripts/build-and-push-ghcr.sh fengxiao2019`
3. The updated image will be used in next CI runs

## Files

- `Dockerfile.base` - Base image with RocksDB + Go
- `Dockerfile.test` - Test image using base image
- `scripts/build-and-push-ghcr.sh` - Build/push script for GHCR
- `scripts/build-and-push-base-image.sh` - Alternative for Docker Hub

## Troubleshooting

### Authentication Issues
```bash
# Re-login to GHCR
docker logout ghcr.io
echo "YOUR_TOKEN" | docker login ghcr.io -u fengxiao2019 --password-stdin
```

### Base Image Not Found
Make sure you've pushed the base image:
```bash
docker pull ghcr.io/fengxiao2019/rocksdb-cli-base:latest
```

### Build Failures
Check if you have enough disk space and memory:
```bash
docker system prune -f  # Clean up space
``` 