#!/bin/sh
set -eu

REPO="KirillBaranov/kb-labs-create"
BINARY="kb-create"
DEST="${HOME}/.local/bin/${BINARY}"
VERSION="latest"

usage() {
  cat <<'EOF'
Usage: install.sh [--version <tag>]

Options:
  --version <tag>   Install specific release tag (example: v1.2.3)
  -h, --help        Show this help
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      shift
      if [ "$#" -eq 0 ]; then
        echo "Error: --version requires a value (example: v1.2.3)." >&2
        exit 1
      fi
      VERSION="$1"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Error: unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

if ! command -v curl >/dev/null 2>&1; then
  echo "Error: curl is required but not found in PATH." >&2
  exit 1
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
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

if [ "$VERSION" = "latest" ]; then
  BASE_URL="https://github.com/${REPO}/releases/latest/download"
else
  BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
fi

BINARY_FILE="${BINARY}-${OS}-${ARCH}"
BINARY_URL="${BASE_URL}/${BINARY_FILE}"
CHECKSUMS_URL="${BASE_URL}/checksums.txt"

TMP_BIN="$(mktemp)"
TMP_SUM="$(mktemp)"
cleanup() {
  rm -f "$TMP_BIN" "$TMP_SUM"
}
trap cleanup EXIT

echo "Downloading ${BINARY_FILE}..."
curl -fsSL "$BINARY_URL" -o "$TMP_BIN"

echo "Downloading checksums..."
curl -fsSL "$CHECKSUMS_URL" -o "$TMP_SUM"

EXPECTED="$(grep "  ${BINARY_FILE}$" "$TMP_SUM" | awk '{print $1}' | head -n 1)"
if [ -z "$EXPECTED" ]; then
  echo "Error: checksum for ${BINARY_FILE} not found in checksums.txt." >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$TMP_BIN" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "$TMP_BIN" | awk '{print $1}')"
else
  echo "Error: neither sha256sum nor shasum found for checksum verification." >&2
  exit 1
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Error: checksum mismatch for ${BINARY_FILE}." >&2
  echo "Expected: $EXPECTED" >&2
  echo "Actual:   $ACTUAL" >&2
  exit 1
fi

chmod +x "$TMP_BIN"
mkdir -p "$(dirname "$DEST")"
mv "$TMP_BIN" "$DEST"

case ":$PATH:" in
  *":${HOME}/.local/bin:"*) ;;
  *)
    echo ""
    echo "Add to your shell profile (~/.zshrc or ~/.bashrc):"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    ;;
esac

echo ""
echo "✓ ${BINARY} installed to $DEST"
echo "✓ Checksum verified (${BINARY_FILE})"
echo ""
echo "Get started:"
echo "  kb-create my-project"
echo "  kb-create status"
