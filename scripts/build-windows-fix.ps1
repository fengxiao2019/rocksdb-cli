# Enhanced Windows Build Script for rocksdb-cli
# This script addresses common build blocking issues
#
# Usage:
#   .\scripts\build-windows-fix.ps1              # Build CLI only
#   .\scripts\build-windows-fix.ps1 -WebBuild    # Build Web Server with UI
#   .\scripts\build-windows-fix.ps1 -Clean       # Clean before build
#   .\scripts\build-windows-fix.ps1 -Verbose     # Verbose output
#   .\scripts\build-windows-fix.ps1 -WebBuild -Clean -Version "v1.2.0"  # Combined options

param(
    [switch]$Verbose,
    [switch]$Clean,
    [switch]$WebBuild,
    [string]$Version = "v1.0.0"
)

$ErrorActionPreference = "Stop"

Write-Host "=== RocksDB CLI Windows Build Script ===" -ForegroundColor Cyan
Write-Host ""

if ($WebBuild) {
    Write-Host "Mode: Web Server Build (with UI)" -ForegroundColor Cyan
} else {
    Write-Host "Mode: CLI Build" -ForegroundColor Cyan
}
Write-Host ""

# 1. Check prerequisites
Write-Host "[1/6] Checking prerequisites..." -ForegroundColor Yellow

if (-not $env:VCPKG_ROOT) {
    Write-Host "ERROR: VCPKG_ROOT environment variable is not set!" -ForegroundColor Red
    Write-Host "Please set VCPKG_ROOT to your vcpkg installation directory." -ForegroundColor Red
    exit 1
}

if (-not (Test-Path "$env:VCPKG_ROOT\installed\x64-windows\lib\rocksdb.lib")) {
    Write-Host "ERROR: rocksdb.lib not found!" -ForegroundColor Red
    Write-Host "Please install RocksDB: vcpkg install rocksdb:x64-windows" -ForegroundColor Red
    exit 1
}

Write-Host "✓ VCPKG_ROOT: $env:VCPKG_ROOT" -ForegroundColor Green

# 2. Verify GCC compiler
Write-Host "[2/6] Verifying GCC compiler..." -ForegroundColor Yellow

$gccPath = Get-Command gcc -ErrorAction SilentlyContinue
if (-not $gccPath) {
    Write-Host "ERROR: GCC not found in PATH!" -ForegroundColor Red
    Write-Host "Please install Strawberry Perl from https://strawberryperl.com" -ForegroundColor Red
    exit 1
}

Write-Host "✓ GCC found: $($gccPath.Source)" -ForegroundColor Green
& gcc --version | Select-Object -First 1

# 3. Check required libraries
Write-Host "[3/6] Checking required libraries..." -ForegroundColor Yellow

$requiredLibs = @("rocksdb.lib", "snappy.lib", "lz4.lib", "zstd.lib")
foreach ($lib in $requiredLibs) {
    $libPath = "$env:VCPKG_ROOT\installed\x64-windows\lib\$lib"
    if (Test-Path $libPath) {
        Write-Host "✓ Found: $lib" -ForegroundColor Green
    } else {
        Write-Host "✗ Missing: $lib" -ForegroundColor Red
        $missingLibs = $true
    }
}

if ($missingLibs) {
    Write-Host ""
    Write-Host "Install missing libraries with:" -ForegroundColor Yellow
    Write-Host "vcpkg install rocksdb:x64-windows snappy:x64-windows lz4:x64-windows zstd:x64-windows" -ForegroundColor Yellow
    exit 1
}

# 4. Detect toolchain and set CGO environment variables
Write-Host "[4/6] Detecting toolchain and setting CGO variables..." -ForegroundColor Yellow
Write-Host ""

# Check for GCC
$hasGCC = Get-Command gcc -ErrorAction SilentlyContinue

# Check if libraries were built with MSVC (x64-windows) or MinGW (x64-mingw-dynamic)
$libPathMSVC = "$env:VCPKG_ROOT\installed\x64-windows\lib"
$libPathMinGW = "$env:VCPKG_ROOT\installed\x64-mingw-dynamic\lib"

$hasMSVCLibs = Test-Path "$libPathMSVC\rocksdb.lib"
$hasMinGWLibs = (Test-Path "$libPathMinGW\librocksdb.dll.a") -or (Test-Path "$libPathMinGW\librocksdb-shared.dll.a") -or (Test-Path "$libPathMinGW\librocksdb.a")

Write-Host "Library Detection:" -ForegroundColor Cyan
Write-Host "  MSVC libs (x64-windows): $(if ($hasMSVCLibs) { '✓ Found' } else { '✗ Not found' })" -ForegroundColor $(if ($hasMSVCLibs) { 'Green' } else { 'Gray' })
Write-Host "  MinGW libs (x64-mingw-dynamic): $(if ($hasMinGWLibs) { '✓ Found' } else { '✗ Not found' })" -ForegroundColor $(if ($hasMinGWLibs) { 'Green' } else { 'Gray' })
Write-Host ""

# Determine which toolchain to use
$usingMinGW = $false

if ($hasMinGWLibs -and $hasGCC) {
    # Best case: MinGW libraries + GCC compiler
    Write-Host "✓ Using MinGW toolchain (GCC + MinGW libraries)" -ForegroundColor Green
    $libPath = $libPathMinGW
    $includePath = "$env:VCPKG_ROOT\installed\x64-mingw-dynamic\include"
    $binPath = "$env:VCPKG_ROOT\installed\x64-mingw-dynamic\bin"
    $usingMinGW = $true
    
} elseif ($hasMSVCLibs -and -not $hasGCC) {
    # MSVC libraries but no GCC - this is problematic for CGO
    Write-Host "⚠ CRITICAL: You have MSVC libraries but CGO requires GCC!" -ForegroundColor Red
    Write-Host ""
    Write-Host "MSVC libraries (.lib files) are incompatible with CGO's build process." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Recommended Solutions (choose one):" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Option 1: Install MinGW-compatible libraries (RECOMMENDED)" -ForegroundColor Green
    Write-Host "  vcpkg install rocksdb:x64-mingw-dynamic snappy:x64-mingw-dynamic lz4:x64-mingw-dynamic zstd:x64-mingw-dynamic" -ForegroundColor White
    Write-Host "  Then install Strawberry Perl: https://strawberryperl.com/" -ForegroundColor White
    Write-Host ""
    Write-Host "Option 2: Use WSL (Windows Subsystem for Linux)" -ForegroundColor Yellow
    Write-Host "  wsl --install" -ForegroundColor White
    Write-Host "  # Then build inside WSL using Linux tools" -ForegroundColor White
    Write-Host ""
    Write-Host "Option 3: Use Docker" -ForegroundColor Yellow
    Write-Host "  docker run --rm -v ${PWD}:/app -w /app golang:1.24 go build -o rocksdb-cli ./cmd" -ForegroundColor White
    Write-Host ""
    exit 1
    
} elseif ($hasMSVCLibs -and $hasGCC) {
    # MSVC libraries + GCC = Incompatible!
    Write-Host "⚠ CRITICAL: Toolchain mismatch detected!" -ForegroundColor Red
    Write-Host ""
    Write-Host "You have:" -ForegroundColor Yellow
    Write-Host "  - MSVC-compiled libraries (.lib files) from x64-windows triplet" -ForegroundColor White
    Write-Host "  - GCC compiler from Strawberry Perl" -ForegroundColor White
    Write-Host ""
    Write-Host "These are INCOMPATIBLE! GCC's linker cannot use MSVC .lib files." -ForegroundColor Red
    Write-Host ""
    Write-Host "Solutions:" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "1. INSTALL MinGW libraries (RECOMMENDED - quickest fix):" -ForegroundColor Green
    Write-Host "   vcpkg install rocksdb:x64-mingw-dynamic snappy:x64-mingw-dynamic lz4:x64-mingw-dynamic zstd:x64-mingw-dynamic" -ForegroundColor White
    Write-Host "   Then re-run this script" -ForegroundColor White
    Write-Host ""
    Write-Host "2. Remove GCC and use pure MSVC (difficult with CGO):" -ForegroundColor Yellow  
    Write-Host "   Remove C:\Strawberry from PATH" -ForegroundColor White
    Write-Host "   Use WSL or Docker instead (see Option 3)" -ForegroundColor White
    Write-Host ""
    Write-Host "3. Use WSL (RECOMMENDED for production builds):" -ForegroundColor Green
    Write-Host "   wsl --install" -ForegroundColor White
    Write-Host "   # Build inside WSL with native Linux tools" -ForegroundColor White
    Write-Host ""
    Write-Host "Press Ctrl+C to exit, or Enter to continue anyway (will likely fail)..." -ForegroundColor Yellow
    Read-Host
    
    # Try anyway with MinGW (will likely fail)
    Write-Host "⚠ Attempting build with mismatched toolchain..." -ForegroundColor Yellow
    $libPath = $libPathMSVC
    $includePath = "$env:VCPKG_ROOT\installed\x64-windows\include"
    $usingMinGW = $false
    
} elseif ($hasMinGWLibs -and -not $hasGCC) {
    # MinGW libraries but no GCC
    Write-Host "⚠ ERROR: MinGW libraries found but GCC compiler not in PATH!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please install Strawberry Perl to get GCC:" -ForegroundColor Yellow
    Write-Host "  1. Download from: https://strawberryperl.com/" -ForegroundColor White
    Write-Host "  2. Install and restart your terminal" -ForegroundColor White
    Write-Host "  3. Or manually add to PATH: C:\Strawberry\c\bin" -ForegroundColor White
    Write-Host ""
    exit 1
} else {
    Write-Host "ERROR: No RocksDB libraries found!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please install RocksDB with MinGW triplet:" -ForegroundColor Yellow
    Write-Host "  vcpkg install rocksdb:x64-mingw-dynamic snappy:x64-mingw-dynamic lz4:x64-mingw-dynamic zstd:x64-mingw-dynamic" -ForegroundColor White
    Write-Host ""
    exit 1
}

# Configure CGO environment
$env:CGO_ENABLED = "1"

if ($usingMinGW) {
    # MinGW toolchain with proper configuration
    $env:CC = "gcc"
    $env:CXX = "g++"
    $env:CGO_CFLAGS = "-I$includePath"
    # Use -L for library path and -l for library names (without 'lib' prefix and '.a' extension)
    $env:CGO_LDFLAGS = "-L$libPath -lrocksdb -lsnappy -llz4 -lzstd -lstdc++ -lws2_32 -lrpcrt4 -lshlwapi"
    
    # Ensure DLLs can be found at runtime
    $env:PATH = "$binPath;$env:PATH"
    
    Write-Host ""
    Write-Host "CGO Configuration:" -ForegroundColor Cyan
    Write-Host "  Compiler: GCC (MinGW)" -ForegroundColor Green
    Write-Host "  Libraries: x64-mingw-dynamic" -ForegroundColor Green
} else {
    # Fallback attempt (will likely fail)
    $env:CC = "gcc"
    $env:CXX = "g++"
    $env:CGO_CFLAGS = "-I$includePath"
    # This will probably fail because GCC can't link MSVC .lib files
    $env:CGO_LDFLAGS = "-L$libPath -lrocksdb -lsnappy -llz4 -lzstd -lstdc++ -lws2_32 -lrpcrt4"
    
    Write-Host ""
    Write-Host "CGO Configuration:" -ForegroundColor Yellow
    Write-Host "  Compiler: GCC (MinGW)" -ForegroundColor Yellow
    Write-Host "  Libraries: MSVC (INCOMPATIBLE - expect failure)" -ForegroundColor Red
}

Write-Host "✓ CGO_ENABLED = 1" -ForegroundColor Green
Write-Host "✓ CGO_CFLAGS set" -ForegroundColor Green
Write-Host "✓ CGO_LDFLAGS set" -ForegroundColor Green

# 5. Clean if requested
if ($Clean) {
    Write-Host "[5/6] Cleaning previous builds..." -ForegroundColor Yellow
    if (Test-Path "build") {
        Remove-Item -Recurse -Force build
    }
    if (Test-Path "rocksdb-cli.exe") {
        Remove-Item -Force rocksdb-cli.exe
    }
    if (Test-Path "internal\webui\dist") {
        Remove-Item -Recurse -Force "internal\webui\dist"
    }
    if (Test-Path "web-ui\dist") {
        Remove-Item -Recurse -Force "web-ui\dist"
    }
    Write-Host "✓ Cleaned" -ForegroundColor Green
}

# 5.5. Build Web UI if requested
if ($WebBuild) {
    Write-Host "[5.5/6] Building Web UI..." -ForegroundColor Yellow
    Write-Host ""
    
    # Check if Node.js is installed
    $nodeCmd = Get-Command node -ErrorAction SilentlyContinue
    if (-not $nodeCmd) {
        Write-Host "ERROR: Node.js not found in PATH!" -ForegroundColor Red
        Write-Host "Please install Node.js from: https://nodejs.org/" -ForegroundColor Red
        exit 1
    }
    
    $npmCmd = Get-Command npm -ErrorAction SilentlyContinue
    if (-not $npmCmd) {
        Write-Host "ERROR: npm not found in PATH!" -ForegroundColor Red
        Write-Host "Please ensure npm is installed (comes with Node.js)" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "✓ Node.js version: $(node --version)" -ForegroundColor Green
    Write-Host "✓ npm version: $(npm --version)" -ForegroundColor Green
    Write-Host ""
    
    # Navigate to web-ui directory and build
    Push-Location "web-ui"
    try {
        Write-Host "Installing npm dependencies..." -ForegroundColor Yellow
        & npm install
        if ($LASTEXITCODE -ne 0) {
            throw "npm install failed"
        }
        
        Write-Host "Building React app..." -ForegroundColor Yellow
        & npm run build
        if ($LASTEXITCODE -ne 0) {
            throw "npm build failed"
        }
        
        Write-Host "✓ Web UI build completed" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Web UI build failed: $_" -ForegroundColor Red
        Pop-Location
        exit 1
    } finally {
        Pop-Location
    }
    
    # Copy built files to internal/webui/dist
    Write-Host "Copying web UI files to internal\webui\dist..." -ForegroundColor Yellow
    
    if (Test-Path "internal\webui\dist") {
        Remove-Item -Recurse -Force "internal\webui\dist"
    }
    
    New-Item -ItemType Directory -Path "internal\webui\dist" -Force | Out-Null
    Copy-Item -Path "web-ui\dist\*" -Destination "internal\webui\dist\" -Recurse -Force
    
    Write-Host "✓ Web UI files copied" -ForegroundColor Green
    Write-Host ""
}

# 6. Build
Write-Host "[6/6] Building rocksdb-cli..." -ForegroundColor Yellow
Write-Host ""

$buildDir = "build"
if (-not (Test-Path $buildDir)) {
    New-Item -ItemType Directory -Path $buildDir | Out-Null
}

# Determine output file and build target
if ($WebBuild) {
    $outputFile = "$buildDir\web-server-windows-amd64.exe"
    $buildTarget = ".\cmd\web-server"
    Write-Host "Building: Web Server with UI" -ForegroundColor Cyan
} else {
    $outputFile = "$buildDir\rocksdb-cli-windows-amd64.exe"
    $buildTarget = ".\cmd"
    Write-Host "Building: CLI" -ForegroundColor Cyan
}

$ldflags = "-X main.version=$Version"

Write-Host "Output: $outputFile" -ForegroundColor Cyan
Write-Host "Version: $Version" -ForegroundColor Cyan
Write-Host ""

$buildArgs = @(
    "build",
    "-ldflags", $ldflags,
    "-o", $outputFile,
    $buildTarget
)

if ($Verbose) {
    $buildArgs = @("-v", "-x") + $buildArgs
    Write-Host "Running: go $($buildArgs -join ' ')" -ForegroundColor Cyan
    Write-Host ""
}

Write-Host "This may take 5-10 minutes on first build..." -ForegroundColor Yellow
Write-Host "If it appears to hang, check Task Manager for gcc.exe processes" -ForegroundColor Yellow
Write-Host ""

$sw = [System.Diagnostics.Stopwatch]::StartNew()

try {
    if ($Verbose) {
        & go @buildArgs
    } else {
        & go @buildArgs 2>&1 | Write-Host
    }
    
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed with exit code $LASTEXITCODE"
    }
    
    $sw.Stop()
    
    Write-Host ""
    Write-Host "=== Build Successful! ===" -ForegroundColor Green
    Write-Host ""
    Write-Host "✓ Executable: $outputFile" -ForegroundColor Green
    Write-Host "✓ Build time: $($sw.Elapsed.ToString('mm\:ss'))" -ForegroundColor Green
    Write-Host ""
    
    # Copy DLLs if using MinGW
    if ($usingMinGW) {
        Write-Host "[6/6] Copying runtime DLLs..." -ForegroundColor Yellow
        
        $dllsToCopy = @(
            "librocksdb-shared.dll",
            "libsnappy.dll",
            "liblz4.dll",
            "libzstd.dll",
            "zlib1.dll",
            "libstdc++-6.dll",
            "libgcc_s_seh-1.dll",
            "libwinpthread-1.dll"
        )
        
        $sourceDirs = @($binPath, "C:\Strawberry\c\bin")
        $copiedCount = 0
        
        foreach ($dll in $dllsToCopy) {
            foreach ($sourceDir in $sourceDirs) {
                $sourcePath = Join-Path $sourceDir $dll
                if (Test-Path $sourcePath) {
                    $destPath = Join-Path $buildDir $dll
                    if (-not (Test-Path $destPath) -or 
                        (Get-Item $sourcePath).LastWriteTime -gt (Get-Item $destPath).LastWriteTime) {
                        Copy-Item -Path $sourcePath -Destination $destPath -Force -ErrorAction SilentlyContinue
                        $copiedCount++
                    }
                    break
                }
            }
        }
        
        if ($copiedCount -gt 0) {
            Write-Host "✓ Copied $copiedCount runtime DLL(s)" -ForegroundColor Green
        }
        Write-Host ""
    }
    
    Write-Host "Test it with:" -ForegroundColor Cyan
    if ($WebBuild) {
        Write-Host "  $outputFile --db=./testdb" -ForegroundColor White
        Write-Host "  # Then open browser to: http://localhost:8080" -ForegroundColor White
    } else {
        Write-Host "  $outputFile --help" -ForegroundColor White
        Write-Host "  $outputFile repl --db=./testdb" -ForegroundColor White
    }
    Write-Host ""
    
} catch {
    Write-Host ""
    Write-Host "=== Build Failed! ===" -ForegroundColor Red
    Write-Host ""
    Write-Host "Error: $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "Troubleshooting steps:" -ForegroundColor Yellow
    Write-Host "1. Try running with -Verbose flag to see detailed output" -ForegroundColor White
    Write-Host "2. Check if gcc.exe is running in Task Manager (it might just be slow)" -ForegroundColor White
    Write-Host "3. Verify all libraries are installed: vcpkg list" -ForegroundColor White
    Write-Host "4. Try building with verbose mode: go build -v -x -o test.exe .\cmd" -ForegroundColor White
    Write-Host ""
    exit 1
}

