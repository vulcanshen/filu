#!/bin/sh
# filu installer for macOS / Linux (unix-only — on Windows use WSL).
# Usage: curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/install.sh | sh

set -e

REPO="vulcanshen/filu"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux*)  OS="linux" ;;
  darwin*) OS="darwin" ;;
  *) echo "Error: filu is unix-only (no Windows build — use WSL). Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Error: unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
echo "Fetching latest release..."
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/')
echo "Latest version: $VERSION"

# Install dir
if [ "$(id -u)" = "0" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="$HOME/.local/bin"
fi

FILENAME="filu_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/$FILENAME"

# Download
TMPDIR=$(mktemp -d)
echo "Downloading $FILENAME..."
curl -fsSL "$DOWNLOAD_URL" -o "$TMPDIR/$FILENAME"

# Extract
echo "Extracting..."
tar xzf "$TMPDIR/$FILENAME" -C "$TMPDIR"

# Install
mkdir -p "$INSTALL_DIR"
cp "$TMPDIR/filu" "$INSTALL_DIR/filu"
chmod +x "$INSTALL_DIR/filu"
rm -rf "$TMPDIR"

echo ""
echo "filu $VERSION installed to $INSTALL_DIR"

# Check if install dir is in PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "WARNING: $INSTALL_DIR is not in your PATH. Add it by running:"
    echo ""
    echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.$(basename "$SHELL")rc && source ~/.$(basename "$SHELL")rc"
    ;;
esac

# Search (/) relies on external tools — warn (don't auto-install: package names
# vary across distros). brew installs handle these as formula dependencies.
missing=""
command -v rg >/dev/null 2>&1 || missing="$missing ripgrep"
command -v fd >/dev/null 2>&1 || missing="$missing fd"
if [ -n "$missing" ]; then
  echo ""
  echo "NOTE: filu's Search (/) uses ripgrep (content filter, required) and fd"
  echo "(file listing, optional — falls back to a slower built-in walk)."
  echo "Not found:$missing. Install with your package manager, e.g.:"
  echo "  macOS:          brew install ripgrep fd"
  echo "  Debian/Ubuntu:  sudo apt install ripgrep fd-find     # fd binary is 'fdfind'"
  echo "  Fedora:         sudo dnf install ripgrep fd-find"
  echo "  Arch:           sudo pacman -S ripgrep fd"
fi

echo ""
echo "Run 'filu' to launch."
