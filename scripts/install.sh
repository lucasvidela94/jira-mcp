#!/usr/bin/env bash
set -euo pipefail

# Install jira-mcp from GitHub releases.
# Usage: curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/main/scripts/install.sh | bash

REPO="lucasvidela94/jira-mcp"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  msys*|cygwin*|mingw*|nt|win*) OS="windows" ;;
  *) echo "unsupported OS: $OS" >&2; exit 1 ;;
esac

VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "could not determine latest version" >&2
  exit 1
fi

ARCHIVE="jira-mcp_${VERSION#v}_${OS}_${ARCH}.tar.gz"
if [ "$OS" = "windows" ]; then
  ARCHIVE="jira-mcp_${VERSION#v}_${OS}_${ARCH}.zip"
fi

URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo "Installing jira-mcp ${VERSION} for ${OS}/${ARCH}..."

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "$URL" -o "$TMP_DIR/$ARCHIVE"

case "$ARCHIVE" in
  *.zip) unzip -q "$TMP_DIR/$ARCHIVE" -d "$TMP_DIR" ;;
  *) tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR" ;;
esac

mkdir -p "$INSTALL_DIR"
cp "$TMP_DIR/jira-mcp" "$INSTALL_DIR/jira-mcp"
chmod +x "$INSTALL_DIR/jira-mcp"

echo "jira-mcp installed to ${INSTALL_DIR}/jira-mcp"

if ! command -v jira-mcp >/dev/null 2>&1; then
  echo ""
  echo "Note: ${INSTALL_DIR} is not in your PATH. Add it to your shell profile:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi
