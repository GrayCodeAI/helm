#!/usr/bin/env bash
#
# HELM Installer
# https://github.com/yourname/helm
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/yourname/helm/main/install.sh | bash
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

BINARY_NAME="helm"
INSTALL_DIR="${HOME}/.local/bin"
VERSION="latest"
REPO="yourname/helm"

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║            HELM Installer              ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *)
        echo -e "${RED}Error: Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "Detected: ${GREEN}${OS}/${ARCH}${NC}"

if [[ "$VERSION" == "latest" ]]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "$VERSION" ]]; then
        echo -e "${RED}Error: Could not determine latest version${NC}"
        exit 1
    fi
fi

VERSION_NUM="${VERSION#v}"
echo -e "Installing: ${GREEN}${VERSION}${NC}"

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/helm_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

echo -e "Downloading..."
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/helm.tar.gz"; then
    echo -e "${RED}Error: Download failed${NC}"
    echo "URL: $DOWNLOAD_URL"
    echo ""
    echo "Or build from source:"
    echo "  git clone https://github.com/${REPO}.git"
    echo "  cd helm && go build ./cmd/helm"
    exit 1
fi

echo -e "Extracting..."
tar -xzf "$TMP_DIR/helm.tar.gz" -C "$TMP_DIR"

mkdir -p "$INSTALL_DIR"
echo -e "Installing to ${GREEN}${INSTALL_DIR}/${BINARY_NAME}${NC}"
mv "$TMP_DIR/helm" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo -e "${YELLOW}Note: ${INSTALL_DIR} is not in your PATH${NC}"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Installation successful!           ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "Binary: ${GREEN}${INSTALL_DIR}/${BINARY_NAME}${NC}"
echo ""
echo "Get started:"
echo "  helm init        # Initialize a project"
echo "  helm             # Launch the TUI"
echo "  helm --help      # Show help"
