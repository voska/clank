#!/usr/bin/env bash
# scripts/install.sh — download pre-built clank binary
set -euo pipefail

INSTALL_DIR="${1:-.}"
REPO="voska/clank"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/')
if [ -z "$LATEST" ]; then
  echo "clank: could not determine latest version" >&2
  exit 1
fi

URL="https://github.com/$REPO/releases/download/v${LATEST}/clank_${LATEST}_${OS}_${ARCH}.tar.gz"
echo "clank: downloading v${LATEST} for ${OS}/${ARCH}..."

mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" | tar xz -C "$INSTALL_DIR" clank
chmod +x "$INSTALL_DIR/clank"
echo "clank: installed to $INSTALL_DIR/clank"
