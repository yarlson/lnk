#!/bin/bash

# Lnk installer script
# Downloads and installs the latest release of lnk

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub repository
REPO="yarlson/lnk"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="lnk"

# Detect OS and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux)   os="Linux" ;;
        Darwin)  os="Darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="Windows" ;;
        *)
            echo -e "${RED}Error: Unsupported operating system $(uname -s)${NC}"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64) arch="x86_64" ;;
        arm64|aarch64) arch="arm64" ;;
        *)
            echo -e "${RED}Error: Unsupported architecture $(uname -m)${NC}"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Get the latest release version
get_latest_version() {
    curl -s "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install_lnk() {
    local platform version
    
    echo -e "${BLUE}🔗 Installing lnk...${NC}"
    
    platform=$(detect_platform)
    version=$(get_latest_version)
    
    if [ -z "$version" ]; then
        echo -e "${RED}Error: Failed to get latest version${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}Latest version: ${version}${NC}"
    echo -e "${BLUE}Platform: ${platform}${NC}"
    
    # Download URL
    local filename="lnk_${platform}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${version}/${filename}"
    
    echo -e "${BLUE}Downloading ${url}...${NC}"
    
    # Create temporary directory
    local tmp_dir=$(mktemp -d)
    cd "$tmp_dir"
    
    # Download the binary
    if ! curl -sL "$url" -o "$filename"; then
        echo -e "${RED}Error: Failed to download ${url}${NC}"
        exit 1
    fi
    
    # Extract the binary
    if ! tar -xzf "$filename"; then
        echo -e "${RED}Error: Failed to extract ${filename}${NC}"
        exit 1
    fi
    
    # Make binary executable
    chmod +x "$BINARY_NAME"
    
    # Install to system directory
    echo -e "${YELLOW}Installing to ${INSTALL_DIR} (requires sudo)...${NC}"
    if ! sudo mv "$BINARY_NAME" "$INSTALL_DIR/"; then
        echo -e "${RED}Error: Failed to install binary${NC}"
        exit 1
    fi
    
    # Cleanup
    cd - > /dev/null
    rm -rf "$tmp_dir"
    
    echo -e "${GREEN}✅ lnk installed successfully!${NC}"
    echo -e "${GREEN}Run 'lnk --help' to get started.${NC}"
}

# Check if running with --help
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Lnk installer script"
    echo ""
    echo "Usage: curl -sSL https://raw.githubusercontent.com/yarlson/lnk/main/install.sh | bash"
    echo ""
    echo "This script will:"
    echo "  1. Detect your OS and architecture"
    echo "  2. Download the latest lnk release"
    echo "  3. Install it to /usr/local/bin (requires sudo)"
    exit 0
fi

# Run the installer
install_lnk 