#!/bin/sh
set -e

REPO="anivaryam/brokit"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
  *) echo "Unsupported OS: $OS" && exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

# Get latest version
VERSION="$(curl -sSf https://api.github.com/repos/${REPO}/releases/latest | grep '"tag_name"' | cut -d'"' -f4)"
if [ -z "$VERSION" ]; then
  echo "Failed to fetch latest version"
  exit 1
fi

# Download
EXT="tar.gz"
if [ "$OS" = "windows" ]; then
  EXT="zip"
fi

URL="https://github.com/${REPO}/releases/download/${VERSION}/brokit_${OS}_${ARCH}.${EXT}"
echo "Downloading brokit ${VERSION} for ${OS}/${ARCH}..."

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -sSfL "$URL" -o "${TMP}/brokit.${EXT}"

# Extract
if [ "$EXT" = "zip" ]; then
  unzip -q "${TMP}/brokit.${EXT}" -d "$TMP"
else
  tar -xzf "${TMP}/brokit.${EXT}" -C "$TMP"
fi

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/brokit" "${INSTALL_DIR}/brokit"
else
  sudo mv "${TMP}/brokit" "${INSTALL_DIR}/brokit"
fi

chmod +x "${INSTALL_DIR}/brokit"
echo "brokit ${VERSION} installed to ${INSTALL_DIR}/brokit"
