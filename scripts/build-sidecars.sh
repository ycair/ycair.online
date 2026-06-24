#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$PROJECT_ROOT/src-tauri/bin"
CORE_DIR="$PROJECT_ROOT/core"
SIGNAL_DIR="$PROJECT_ROOT/signaling-server"

echo "=== ycair.online Cross-Platform Build ==="
echo ""

mkdir -p "$BIN_DIR"

echo "--- ycair-core ---"

echo "Building ycair-core for macOS ARM64..."
cd "$CORE_DIR"
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
    go build -ldflags="-s -w" \
    -o "$BIN_DIR/ycair-core-aarch64-apple-darwin" .
echo "  -> $BIN_DIR/ycair-core-aarch64-apple-darwin"

echo "Building ycair-core for macOS x86_64..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o "$BIN_DIR/ycair-core-x86_64-apple-darwin" . 2>/dev/null || {
    echo "  -> SKIPPED (cross-compiler not available)"
}

echo "Building ycair-core for Linux x86_64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o "$BIN_DIR/ycair-core-x86_64-unknown-linux-gnu" .
echo "  -> $BIN_DIR/ycair-core-x86_64-unknown-linux-gnu"

echo "Building ycair-core for Windows x86_64..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o "$BIN_DIR/ycair-core-x86_64-pc-windows-msvc.exe" .
echo "  -> $BIN_DIR/ycair-core-x86_64-pc-windows-msvc.exe"

echo ""
echo "--- Signaling Server ---"
echo "Building signaling server..."
cd "$SIGNAL_DIR"
CGO_ENABLED=0 go build -ldflags="-s -w" -o "$BIN_DIR/signaling-server" .
echo "  -> $BIN_DIR/signaling-server"

echo ""
echo "=== Build Complete ==="
echo ""
ls -lh "$BIN_DIR/"
