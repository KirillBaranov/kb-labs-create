#!/bin/sh
set -e

REPO="kb-labs/create"
BINARY="kb-create"
DEST="${HOME}/.local/bin/${BINARY}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  darwin|linux) ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

URL="https://github.com/${REPO}/releases/latest/download/${BINARY}-${OS}-${ARCH}"

echo "Downloading kb-create (${OS}/${ARCH})..."
curl -fsSL "$URL" -o /tmp/kb-create-download
chmod +x /tmp/kb-create-download

# Install to ~/.local/bin (no sudo needed)
mkdir -p "$(dirname "$DEST")"
mv /tmp/kb-create-download "$DEST"

# Add to PATH hint if not already there
case ":$PATH:" in
  *":${HOME}/.local/bin:"*) ;;
  *)
    echo ""
    echo "Add to your shell profile (~/.zshrc or ~/.bashrc):"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    ;;
esac

echo ""
echo "✓ kb-create installed to $DEST"
echo ""
echo "  Get started:"
echo "    kb-create my-project"
