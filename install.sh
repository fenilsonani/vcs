#!/bin/bash

set -e

# VCS Hyperdrive Installation Script
# Downloads pre-built binaries - no Go required!

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

# Check for Homebrew on macOS (recommended)
if [[ "$OS" == "darwin" ]] && command -v brew >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Homebrew detected! Installing via Homebrew...${NC}"
    echo ""
    
    # Install directly from formula URL
    echo -e "${YELLOW}📥 Installing VCS Hyperdrive from formula...${NC}"
    
    echo -e "${YELLOW}📦 Installing VCS Hyperdrive...${NC}"
    brew install https://raw.githubusercontent.com/fenilsonani/vcs/main/homebrew/vcs.rb
    
    echo -e "${GREEN}✅ Installation complete via Homebrew!${NC}"
    echo ""
    echo -e "${BLUE}Verify installation:${NC}"
    echo "  vcs --version"
    echo "  vcs --check-hardware"
    echo ""
    exit 0
fi

# Download pre-built binary
echo -e "${YELLOW}📥 Downloading pre-built binary...${NC}"

# Construct download URL
BINARY_FILE="vcs-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_FILE}"

# Create temporary directory
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

echo -e "${YELLOW}🌐 Downloading from: ${DOWNLOAD_URL}${NC}"

# Download binary
if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$DOWNLOAD_URL" -o "$BINARY_NAME" || {
        echo -e "${RED}❌ Download failed. Building from source as fallback...${NC}"
        build_from_source
        exit 0
    }
elif command -v wget >/dev/null 2>&1; then
    wget -q "$DOWNLOAD_URL" -O "$BINARY_NAME" || {
        echo -e "${RED}❌ Download failed. Building from source as fallback...${NC}"
        build_from_source
        exit 0
    }
else
    echo -e "${RED}❌ Neither curl nor wget found${NC}"
    build_from_source
    exit 0
fi

# Make binary executable
chmod +x "$BINARY_NAME"

# Determine installation directory
if [[ -w "/usr/local/bin" ]]; then
    INSTALL_DIR="/usr/local/bin"
elif [[ -w "$HOME/.local/bin" ]]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
else
    INSTALL_DIR="/usr/local/bin"
fi

echo -e "${YELLOW}📦 Installing to $INSTALL_DIR...${NC}"

# Install binary
if [[ ! -w "$INSTALL_DIR" ]]; then
    echo -e "${YELLOW}🔐 Installing with sudo (password may be required)...${NC}"
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
else
    mv "$BINARY_NAME" "$INSTALL_DIR/"
fi

# Check if directory is in PATH
if [[ "$INSTALL_DIR" == "$HOME/.local/bin" ]] && [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo ""
    echo -e "${YELLOW}📝 Note: $HOME/.local/bin is not in your PATH${NC}"
    echo -e "${BLUE}Add this to your shell profile:${NC}"
    if [[ "$SHELL" == *"zsh"* ]]; then
        echo "  echo 'export PATH=\$PATH:$HOME/.local/bin' >> ~/.zshrc"
        echo "  source ~/.zshrc"
    elif [[ "$SHELL" == *"bash"* ]]; then
        echo "  echo 'export PATH=\$PATH:$HOME/.local/bin' >> ~/.bashrc"
        echo "  source ~/.bashrc"
    else
        echo "  export PATH=\$PATH:$HOME/.local/bin"
    fi
    echo ""
fi

# Clean up
cd /
rm -rf "$TEMP_DIR"

echo -e "${GREEN}✅ Installation complete!${NC}"

# Fallback function to build from source
build_from_source() {
    echo -e "${YELLOW}🔨 Building from source (requires Go)...${NC}"
    
    if ! command -v go >/dev/null 2>&1; then
        echo -e "${RED}❌ Go not found${NC}"
        echo -e "${YELLOW}Please install Go from https://golang.org/dl/${NC}"
        echo -e "${YELLOW}Or use Homebrew on macOS: brew install go${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}📥 Cloning repository...${NC}"
    git clone "https://github.com/$REPO.git" vcs
    cd vcs
    
    echo -e "${YELLOW}🔨 Building VCS Hyperdrive...${NC}"
    go build -ldflags "-s -w" -o "$BINARY_NAME" ./cmd/vcs
    
    # Install
    if [[ -w "/usr/local/bin" ]]; then
        mv "$BINARY_NAME" "/usr/local/bin/"
    else
        sudo mv "$BINARY_NAME" "/usr/local/bin/"
    fi
}

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