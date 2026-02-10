#!/bin/sh
# CoralMux Relay installer
# Usage: curl -sSL https://raw.githubusercontent.com/user/coralmux/main/install.sh | sh

set -e

REPO="openclaw/coralmux"
BINARY="coralmux-relay"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    darwin|linux) ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

PLATFORM="${OS}-${ARCH}"
echo "Detected platform: $PLATFORM"

# Get latest release
echo "Fetching latest release..."
LATEST=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "Failed to fetch latest release"
    exit 1
fi

echo "Latest version: $LATEST"

# Download
URL="https://github.com/${REPO}/releases/download/${LATEST}/${BINARY}-${PLATFORM}.tar.gz"
echo "Downloading from $URL..."

TMP=$(mktemp -d)
curl -sSL "$URL" -o "$TMP/coralmux.tar.gz"
tar -xzf "$TMP/coralmux.tar.gz" -C "$TMP"

# Install
echo "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP/${BINARY}-${PLATFORM}" "$INSTALL_DIR/$BINARY"
else
    sudo mv "$TMP/${BINARY}-${PLATFORM}" "$INSTALL_DIR/$BINARY"
fi
chmod +x "$INSTALL_DIR/$BINARY"

# Cleanup
rm -rf "$TMP"

echo ""
echo "✅ CoralMux Relay installed successfully!"
echo ""
echo "Quick start:"
echo "  $BINARY -addr :8080                    # Development (no TLS)"
echo "  $BINARY -domain relay.example.com      # Production (auto TLS)"
echo ""
echo "Create a pairing token:"
echo "  curl -X POST http://localhost:8080/api/v1/pair -H 'X-Admin-Key: your-secret'"
echo ""
