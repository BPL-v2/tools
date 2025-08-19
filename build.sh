#!/bin/bash

# Build script for guild stash report tool
# Builds binaries for Windows, Mac, and Linux

set -e  # Exit on any error

echo "Building guild stash report binaries..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build for Linux (amd64)
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o bin/bpl-tools-linux-amd64 .

# Build for Windows (amd64)
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o bin/bpl-tools-windows-amd64.exe .

# Build for macOS (amd64 - Intel)
echo "Building for macOS (amd64 - Intel)..."
GOOS=darwin GOARCH=amd64 go build -o bin/bpl-tools-darwin-amd64 .

# Build for macOS (arm64 - Apple Silicon)
echo "Building for macOS (arm64 - Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -o bin/bpl-tools-darwin-arm64 .

echo "Build complete! Binaries are in guild_stash_tool/bin/"
echo ""
echo "Generated files:"
ls -la bin/
