# Building RocksDB CLI

This guide covers building RocksDB CLI for different platforms.

## Quick Start

For most users, building for your current platform is sufficient:

```bash
make build
```

This creates `build/rocksdb-cli` executable for your current platform.

## Platform-Specific Instructions

### macOS

```bash
# Install dependencies
brew install rocksdb snappy lz4 zstd bzip2

# Set environment variables (add to ~/.zshrc)
export CGO_CFLAGS="-I/opt/homebrew/Cellar/rocksdb/*/include"
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/rocksdb/*/lib -L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"

# Build
make build
```

### Linux (Ubuntu/Debian)

```bash
# Install dependencies
sudo apt-get update
sudo apt-get install librocksdb-dev libsnappy-dev liblz4-dev libzstd-dev libbz2-dev build-essential

# Build
make build
```

### Linux (Using Docker)

Build Linux executables without installing dependencies locally:

```bash
make build-linux-docker
```

This creates `build/rocksdb-cli-linux-amd64`.

### Windows

#### Option 1: WSL (Recommended)

```bash
# Install WSL
wsl --install

# In WSL, follow Linux instructions
sudo apt-get update
sudo apt-get install librocksdb-dev libsnappy-dev liblz4-dev libzstd-dev libbz2-dev build-essential
make build
```

#### Option 2: Native Windows

1. Install [vcpkg](https://github.com/Microsoft/vcpkg)
2. Install dependencies:
   ```cmd
   vcpkg install rocksdb:x64-windows snappy:x64-windows lz4:x64-windows zstd:x64-windows
   ```
3. Set environment variables:
   ```cmd
   set CGO_CFLAGS=-I%VCPKG_ROOT%\installed\x64-windows\include
   set CGO_LDFLAGS=-L%VCPKG_ROOT%\installed\x64-windows\lib -lrocksdb -lsnappy -llz4 -lzstd
   set CGO_ENABLED=1
   ```
4. Build:
   ```cmd
   scripts\build.bat
   ```

## Cross-Platform Building

### Why Cross-Compilation is Complex

RocksDB CLI uses CGO (C bindings) to interface with the RocksDB C++ library. This means:

1. **Native libraries required**: Each target platform needs RocksDB C++ libraries installed
2. **CGO limitations**: Go's cross-compilation doesn't work with CGO by default
3. **Platform-specific linking**: Different platforms have different library paths and linking requirements

### Solutions for Cross-Platform Builds

#### 1. Native Compilation (Recommended)

Build on each target platform:
- **macOS**: Use the macOS instructions above
- **Linux**: Use Linux instructions or Docker
- **Windows**: Use WSL or native Windows setup

#### 2. Docker-Based Building

Use Docker to build for Linux from any platform:

```bash
# Build Linux executable
make build-linux-docker

# Or manually
docker build -f docker/Dockerfile.linux -t rocksdb-cli-builder .
```

#### 3. GitHub Actions (Automated)

The repository includes GitHub Actions workflows:

1. **Push to main**: Triggers builds for all platforms
2. **Create release**: Generates downloadable binaries
3. **Artifacts**: Available from the Actions tab

To create a release:
```bash
git tag v1.0.0
git push origin v1.0.0
```

This triggers the release workflow and creates binaries for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Build Targets

Available Make targets:

```bash
make build              # Build for current platform
make build-native       # Same as build
make build-linux-docker # Build Linux using Docker
make build-cross        # Show cross-compilation help
make release            # Create release package
make clean              # Clean build directory
make test               # Run tests
make help               # Show all targets
```

## Troubleshooting

### Common Issues

1. **Header files not found**
   - Verify CGO_CFLAGS points to correct RocksDB include directory
   - Check RocksDB installation path

2. **Library linking errors**
   - Verify CGO_LDFLAGS includes all required libraries
   - Ensure all dependencies (snappy, lz4, zstd, bzip2) are installed

3. **Cross-compilation fails**
   - Use native compilation or Docker instead
   - CGO cross-compilation requires complex toolchain setup

4. **Windows build issues**
   - Use WSL for easier setup
   - Ensure proper C++ toolchain (MinGW or MSVC)

### Environment Variables

For persistent setup, add to your shell profile:

**macOS (~/.zshrc)**:
```bash
export CGO_CFLAGS="-I/opt/homebrew/Cellar/rocksdb/*/include"
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/rocksdb/*/lib -L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"
```

**Linux (~/.bashrc)**:
```bash
# Usually not needed if using package manager
export CGO_ENABLED=1
```

**Windows (cmd)**:
```cmd
set CGO_CFLAGS=-I%VCPKG_ROOT%\installed\x64-windows\include
set CGO_LDFLAGS=-L%VCPKG_ROOT%\installed\x64-windows\lib -lrocksdb -lsnappy -llz4 -lzstd
set CGO_ENABLED=1
```

## Release Packaging

Create a release package:

```bash
make release
```

This creates a `release/` directory with:
- Executable for current platform
- Documentation
- Usage guide
- Compressed archive

## Continuous Integration

The project uses GitHub Actions for:
- **Automated testing** on push/PR
- **Cross-platform builds** for releases
- **Artifact generation** for downloads

See `.github/workflows/build.yml` for the complete CI configuration. 