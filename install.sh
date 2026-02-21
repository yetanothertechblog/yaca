#!/bin/bash

set -e

# Configuration
BINARY_NAME="yaca"
INSTALL_DIR="$HOME/.local/bin"
SHELL_PROFILE="$HOME/.zshrc"  # Default to zsh, can be changed to .bashrc

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Installing YACA (Yet Another Coding Assistant)${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go first.${NC}"
    echo "Visit https://golang.org/dl/ to download and install Go"
    exit 1
fi

echo -e "${YELLOW}✅ Go found: $(go version)${NC}"

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Build the binary
echo -e "${YELLOW}🔨 Building YACA binary...${NC}"
go build -o "$BINARY_NAME"
echo -e "${GREEN}✅ Binary built successfully${NC}"

# Check if binary already exists and handle idempotency
if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    echo -e "${YELLOW}⚠️  Binary already exists in $INSTALL_DIR, replacing it...${NC}"
fi

# Install the binary
mv "$BINARY_NAME" "$INSTALL_DIR/"
echo -e "${GREEN}✅ Binary installed to $INSTALL_DIR/$BINARY_NAME${NC}"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}📝 Adding $INSTALL_DIR to PATH in $SHELL_PROFILE${NC}"
    
    # Check if shell profile already has the PATH entry
    if ! grep -q "export PATH.*$INSTALL_DIR" "$SHELL_PROFILE" 2>/dev/null; then
        echo "" >> "$SHELL_PROFILE"
        echo "# YACA - Add local bin directory to PATH" >> "$SHELL_PROFILE"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_PROFILE"
        echo -e "${GREEN}✅ Added PATH entry to $SHELL_PROFILE${NC}"
    else
        echo -e "${GREEN}✅ PATH entry already exists in $SHELL_PROFILE${NC}"
    fi
    
    echo -e "${YELLOW}💡 Restart your shell or run 'source $SHELL_PROFILE' to use the command immediately${NC}"
else
    echo -e "${GREEN}✅ $INSTALL_DIR is already in PATH${NC}"
fi

# Test the installation
echo -e "${YELLOW}🧪 Testing installation...${NC}"
if command -v "$BINARY_NAME" &> /dev/null; then
    echo -e "${GREEN}✅ $BINARY_NAME is available in PATH${NC}"
    echo -e "${YELLOW}   Version info:$(cd "$INSTALL_DIR" && ./"$BINARY_NAME" --help | head -1)${NC}"
else
    echo -e "${RED}❌ $BINARY_NAME not found in PATH. Please check your installation.${NC}"
    exit 1
fi

echo -e "${GREEN}🎉 YACA installation completed successfully!${NC}"
echo -e "${YELLOW}   Run '$BINARY_NAME --help' to get started${NC}"