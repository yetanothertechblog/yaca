#!/bin/sh
set -e

# Configuration
REPO="yetanothertechblog/go-tui-agent"
BINARY_NAME="yaca"
INSTALL_DIR="$HOME/.local/bin"

# Fetch latest release tag from GitHub API
VERSION=$(curl -sI "https://github.com/${REPO}/releases/latest" | grep -i "^location:" | sed 's|.*/||' | tr -d '\r')
if [ -z "$VERSION" ]; then
    echo "Failed to detect latest release, falling back to v0.0.1-beta"
    VERSION="v0.0.1-beta"
fi

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)
        echo "Error: unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    darwin|linux) ;;
    *)
        echo "Error: unsupported OS: $OS"
        exit 1
        ;;
esac

ASSET="${BINARY_NAME}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"

echo "Installing ${BINARY_NAME} ${VERSION} (${OS}/${ARCH})..."

# Download
mkdir -p "$INSTALL_DIR"
HTTP_CODE=$(curl -sL -w "%{http_code}" "$URL" -o "${INSTALL_DIR}/${BINARY_NAME}")
if [ "$HTTP_CODE" != "200" ]; then
    rm -f "${INSTALL_DIR}/${BINARY_NAME}"
    echo "Error: failed to download ${ASSET} (HTTP ${HTTP_CODE})"
    echo "Check available assets at: https://github.com/${REPO}/releases/tag/${VERSION}"
    exit 1
fi
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

# Add to PATH if needed
case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        SHELL_NAME=$(basename "$SHELL")
        case "$SHELL_NAME" in
            zsh)  RC="$HOME/.zshrc" ;;
            bash) RC="$HOME/.bashrc" ;;
            *)    RC="$HOME/.profile" ;;
        esac
        if ! grep -q "export PATH.*${INSTALL_DIR}" "$RC" 2>/dev/null; then
            echo "" >> "$RC"
            echo "export PATH=\"${INSTALL_DIR}:\$PATH\"" >> "$RC"
        fi
        echo "Added ${INSTALL_DIR} to PATH in ${RC}"
        echo "Run 'source ${RC}' or open a new terminal."
        ;;
esac

echo "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
echo "Run '${BINARY_NAME}' to get started."
