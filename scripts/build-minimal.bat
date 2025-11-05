@echo off
REM Build script for minimal rocksdb-cli on Windows (without Web UI)
REM This significantly reduces build time and binary size

setlocal

set APP_NAME=rocksdb-cli-minimal
if "%VERSION%"=="" set VERSION=v1.0.0
set BUILD_DIR=build

REM Create build directory
if not exist %BUILD_DIR% mkdir %BUILD_DIR%

echo ========================================
echo Building %APP_NAME% %VERSION% for Windows
echo Minimal build: Web UI DISABLED
echo ========================================
echo.

REM Set CGO environment for Windows
set CGO_ENABLED=1
set CC=gcc

REM Optional: Set VCPKG paths if using vcpkg
REM Uncomment and adjust if needed:
REM set CGO_CFLAGS=-I%VCPKG_ROOT%\installed\x64-windows\include
REM set CGO_LDFLAGS=-L%VCPKG_ROOT%\installed\x64-windows\lib -lrocksdb

REM Build for Windows amd64 with minimal tag
echo Building for Windows amd64 (minimal)...
set GOOS=windows
set GOARCH=amd64
go build -tags=minimal -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\%APP_NAME%-windows-amd64.exe .\cmd

if %ERRORLEVEL% neq 0 (
    echo.
    echo ========================================
    echo Build FAILED!
    echo ========================================
    echo.
    echo Make sure:
    echo   1. RocksDB C++ libraries are installed
    echo   2. CGO_CFLAGS and CGO_LDFLAGS are set correctly
    echo   3. Go 1.24+ is installed
    echo.
    echo See README.md for installation instructions.
    exit /b 1
)

echo.
echo ========================================
echo Build completed successfully!
echo ========================================
echo.
echo Executable: %BUILD_DIR%\%APP_NAME%-windows-amd64.exe
echo.

REM Show file size
for %%A in (%BUILD_DIR%\%APP_NAME%-windows-amd64.exe) do (
    echo Size: %%~zA bytes
)

echo.
echo Note: Web UI is NOT available in this build.
echo To enable Web UI, use 'build.bat' instead.
echo.

endlocal
