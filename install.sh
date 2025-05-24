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

# Fallback version if redirect fails
FALLBACK_VERSION="v0.0.2"

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

# Get latest version by following redirect
get_latest_version() {
    echo -e "${BLUE}Getting latest release version...${NC}" >&2
    
    # Get redirect location from releases/latest
    local redirect_url
    redirect_url=$(curl -s -I "https://github.com/${REPO}/releases/latest" | grep -i "^location:" | sed 's/\r$//' | cut -d' ' -f2-)
    
    if [ -z "$redirect_url" ]; then
        echo -e "${YELLOW}⚠ Could not get redirect URL, using fallback version ${FALLBACK_VERSION}${NC}" >&2
        echo "$FALLBACK_VERSION"
        return 0
    fi
    
    # Extract version from redirect URL (format: https://github.com/user/repo/releases/tag/v1.2.3)
    local version
    version=$(echo "$redirect_url" | sed -E 's|.*/releases/tag/([^/]*)\s*$|\1|')
    
    if [ -z "$version" ] || [ "$version" = "$redirect_url" ]; then
        echo -e "${YELLOW}⚠ Could not parse version from redirect URL: $redirect_url${NC}" >&2
        echo -e "${YELLOW}Using fallback version ${FALLBACK_VERSION}${NC}" >&2
        echo "$FALLBACK_VERSION"
        return 0
    fi
    
    echo "$version"
}

# Get version to install
get_version() {
    # Allow override via environment variable
    if [ -n "$LNK_VERSION" ]; then
        echo "$LNK_VERSION"
    elif [ -n "$1" ]; then
        echo "$1"
    else
        get_latest_version
    fi
}

# Download and install
install_lnk() {
    local platform version
    
    echo -e "${BLUE}🔗 Installing lnk...${NC}"
    
    platform=$(detect_platform)
    version=$(get_version "$1")
    
    echo -e "${BLUE}Version: ${version}${NC}"
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
        echo -e "${YELLOW}Please check if the release exists at: https://github.com/${REPO}/releases/tag/${version}${NC}"
        echo -e "${YELLOW}Available releases: https://github.com/${REPO}/releases${NC}"
        exit 1
    fi
    
    # Check if we got an HTML error page instead of the binary
    if file "$filename" 2>/dev/null | grep -q "HTML"; then
        echo -e "${RED}Error: Downloaded file appears to be an HTML page (404 error)${NC}"
        echo -e "${YELLOW}The release ${version} might not exist.${NC}"
        echo -e "${YELLOW}Available releases: https://github.com/${REPO}/releases${NC}"
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
    
    # Test the installation
    if command -v lnk >/dev/null 2>&1; then
        echo -e "${GREEN}Installed version: $(lnk --version)${NC}"
    fi
}

# Check if running with --help
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Lnk installer script"
    echo ""
    echo "Usage:"
    echo "  curl -sSL https://raw.githubusercontent.com/yarlson/lnk/main/install.sh | bash"
    echo "  curl -sSL https://raw.githubusercontent.com/yarlson/lnk/main/install.sh | bash -s v0.0.1"
    echo "  LNK_VERSION=v0.0.1 curl -sSL https://raw.githubusercontent.com/yarlson/lnk/main/install.sh | bash"
    echo ""
    echo "This script will:"
    echo "  1. Detect your OS and architecture"
    echo "  2. Auto-detect the latest release by following GitHub redirects"
    echo "  3. Download and install to /usr/local/bin (requires sudo)"
    echo ""
    echo "Environment variables:"
    echo "  LNK_VERSION - Specify version to install (e.g., v0.0.1)"
    echo ""
    echo "Manual installation: https://github.com/yarlson/lnk/releases"
    exit 0
fi

# Run the installer
install_lnk "$1" 