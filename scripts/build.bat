@echo off
REM Build script for rocksdb-cli on Windows
REM Requires RocksDB C++ libraries to be installed

setlocal

set APP_NAME=rocksdb-cli
if "%VERSION%"=="" set VERSION=v1.0.0
set BUILD_DIR=build

REM Create build directory
if not exist %BUILD_DIR% mkdir %BUILD_DIR%

echo Building %APP_NAME% %VERSION% for Windows...

REM Set CGO environment for Windows (adjust paths as needed)
REM You need to install RocksDB C++ libraries first
REM See README.md for installation instructions

set CGO_ENABLED=1
set CC=gcc

REM Build for Windows amd64
echo Building for Windows amd64...
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\%APP_NAME%-windows-amd64.exe .\cmd

if %ERRORLEVEL% neq 0 (
    echo Build failed! Make sure RocksDB C++ libraries are properly installed.
    echo See README.md for installation instructions.
    exit /b 1
)

echo Build completed! Executable: %BUILD_DIR%\%APP_NAME%-windows-amd64.exe

endlocal 