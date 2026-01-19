#!/bin/bash
# =============================================================================
# Argus Installation Script
# =============================================================================
# This script installs Argus with the secure privilege separation model.
#
# Run as root: sudo ./scripts/install.sh
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Installation paths
INSTALL_BIN="/usr/local/bin"
CONFIG_DIR="/etc/argus"
SUDOERS_DIR="/etc/sudoers.d"
GROUP_NAME="argus-users"

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                     Argus Installation Script                     ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}Error: This script must be run as root${NC}"
    echo "Usage: sudo $0"
    exit 1
fi

# Determine script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo -e "${YELLOW}Step 1: Building release binary...${NC}"
cd "$PROJECT_DIR"
if command -v go &> /dev/null; then
    go build -ldflags="-s -w" -trimpath -o argus ./cmd/argus
    echo -e "${GREEN}✓ Built argus binary${NC}"
else
    if [[ ! -f "argus" ]]; then
        echo -e "${RED}Error: Go not installed and no pre-built binary found${NC}"
        exit 1
    fi
    echo -e "${YELLOW}⚠ Using existing binary (Go not installed)${NC}"
fi

echo -e "${YELLOW}Step 2: Installing binaries...${NC}"
# Install the actual binary as argus-bin
install -Dm755 argus "$INSTALL_BIN/argus-bin"
echo -e "${GREEN}✓ Installed binary to $INSTALL_BIN/argus-bin${NC}"

# Install the wrapper script as argus
install -Dm755 scripts/argus-wrapper.sh "$INSTALL_BIN/argus"
echo -e "${GREEN}✓ Installed wrapper to $INSTALL_BIN/argus${NC}"

echo -e "${YELLOW}Step 3: Setting up configuration...${NC}"
# Create config directory
mkdir -p "$CONFIG_DIR"
# Install default config if it doesn't exist
if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
    install -Dm644 configs/default.yaml "$CONFIG_DIR/config.yaml"
    echo -e "${GREEN}✓ Installed default config to $CONFIG_DIR/config.yaml${NC}"
else
    echo -e "${YELLOW}⚠ Config already exists, skipping${NC}"
fi

echo -e "${YELLOW}Step 4: Creating argus-users group...${NC}"
if ! getent group "$GROUP_NAME" > /dev/null 2>&1; then
    groupadd "$GROUP_NAME"
    echo -e "${GREEN}✓ Created group: $GROUP_NAME${NC}"
else
    echo -e "${YELLOW}⚠ Group $GROUP_NAME already exists${NC}"
fi

echo -e "${YELLOW}Step 5: Installing sudoers rule...${NC}"
install -Dm440 scripts/argus.sudoers "$SUDOERS_DIR/argus"
# Verify sudoers syntax
if visudo -c -f "$SUDOERS_DIR/argus" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Installed sudoers rule to $SUDOERS_DIR/argus${NC}"
else
    echo -e "${RED}Error: Invalid sudoers syntax!${NC}"
    rm -f "$SUDOERS_DIR/argus"
    exit 1
fi

echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                     Installation Complete!                        ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "To allow a user to run Argus without password prompts:"
echo -e "  ${BLUE}sudo usermod -aG $GROUP_NAME <username>${NC}"
echo ""
echo -e "Then log out and back in for group changes to take effect."
echo ""
echo -e "Usage:"
echo -e "  ${BLUE}sudo argus${NC}           # Run with elevated privileges"
echo -e "  ${BLUE}argus${NC}                # Run without sudo (limited access)"
echo ""
echo -e "Config file: ${BLUE}$CONFIG_DIR/config.yaml${NC}"
echo ""
