#!/bin/bash

set -e

# VCS Hyperdrive Installation Script
# Supports macOS (Apple Silicon & Intel) and Linux

VERSION="v1.0.0"
REPO="fenilsonani/vcs"
BINARY_NAME="vcs"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

echo -e "${BLUE}${BOLD}🚀 VCS Hyperdrive Installation${NC}"
echo -e "${BLUE}The World's Fastest Version Control System${NC}"
echo ""

# Normalize architecture names
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
    *)
        echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Normalize OS names
case "$OS" in
    Linux) OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)
        echo -e "${RED}❌ Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

echo -e "${YELLOW}🔍 Detected: $OS/$ARCH${NC}"

# Check for Homebrew on macOS
if [[ "$OS" == "darwin" ]] && command -v brew >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Homebrew detected! Using Homebrew installation...${NC}"
    echo ""
    echo -e "${BLUE}Running: ${BOLD}brew tap fenilsonani/vcs${NC}"
    echo -e "${BLUE}Running: ${BOLD}brew install vcs${NC}"
    echo ""
    echo -e "${YELLOW}Please run these commands manually:${NC}"
    echo "  brew tap fenilsonani/vcs"
    echo "  brew install vcs"
    echo ""
    echo -e "${GREEN}🎉 Homebrew installation recommended for macOS!${NC}"
    exit 0
fi

# Check if Go is available for building from source
if command -v go >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Go detected - building from source${NC}"
    
    # Create temporary directory
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    echo -e "${YELLOW}📥 Cloning repository...${NC}"
    git clone "https://github.com/$REPO.git" vcs
    cd vcs
    
    echo -e "${YELLOW}🔨 Building VCS Hyperdrive...${NC}"
    go build -ldflags "-s -w" -o "$BINARY_NAME" ./cmd/vcs
    
    # Install to /usr/local/bin
    INSTALL_DIR="/usr/local/bin"
    echo -e "${YELLOW}📦 Installing to $INSTALL_DIR...${NC}"
    
    if [[ ! -w "$INSTALL_DIR" ]]; then
        echo -e "${YELLOW}🔐 Installing with sudo...${NC}"
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
        sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        mv "$BINARY_NAME" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    # Clean up
    cd /
    rm -rf "$TEMP_DIR"
    
    echo -e "${GREEN}✅ Installation complete!${NC}"
    
else
    echo -e "${RED}❌ Go not found and pre-built binaries not available yet${NC}"
    echo -e "${YELLOW}Please install Go and try again, or use Homebrew on macOS${NC}"
    echo ""
    echo "Install Go: https://golang.org/dl/"
    echo "Install Homebrew: https://brew.sh/"
    exit 1
fi

echo ""
echo -e "${GREEN}${BOLD}🎉 VCS Hyperdrive installed successfully!${NC}"
echo ""
echo -e "${BLUE}Quick start:${NC}"
echo "  vcs --version"
echo "  vcs --check-hardware"
echo "  vcs benchmark --quick"
echo ""
echo -e "${BLUE}Initialize a repository:${NC}"
echo "  vcs init"
echo ""
echo -e "${BLUE}Clone a repository (lightning fast!):${NC}"
echo "  vcs clone https://github.com/user/repo.git"
echo ""
echo -e "${YELLOW}📖 Documentation: https://github.com/$REPO/tree/main/docs${NC}"
echo -e "${GREEN}⚡ Enjoy 1000x+ faster version control!${NC}"