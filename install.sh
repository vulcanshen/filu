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

# ── Search tools (ripgrep + fd) ──────────────────────────────────────────────
# filu's finders use ripgrep (Find `f`, content search — required) and fd
# (Search `/` + Goto `go` listing — optional; filu falls back to a built-in Go
# walk when it is absent). A brew install pulls these in as formula dependencies;
# a raw install.sh has no package manager to lean on, so — for whatever is
# missing — we fetch the same kind of static binary we ship for filu itself: no
# sudo, and the binaries land named `rg`/`fd` (not Debian's `fdfind`, which filu
# would not find). Downloads go into INSTALL_DIR; existing installs are left be.

# Rust target triple shared by ripgrep and fd release assets.
case "$ARCH" in
  amd64) RUST_ARCH="x86_64" ;;
  arm64) RUST_ARCH="aarch64" ;;
esac
case "$OS" in
  darwin) RUST_OS="apple-darwin" ;;
  linux)  RUST_OS="unknown-linux-musl" ;;  # musl = static, distro-agnostic
esac
TARGET="${RUST_ARCH}-${RUST_OS}"

# install_tool <label> <repo> <asset-prefix> <binary> — drop the latest release
# binary into INSTALL_DIR. A missing asset for this platform (e.g. fd ships no
# Intel-macOS build) is a soft skip, never a hard failure.
install_tool() {
  label="$1"; repo="$2"; prefix="$3"; bin="$4"
  tag=$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" 2>/dev/null \
        | grep -oE '"tag_name":[[:space:]]*"[^"]+"' | head -1 | sed 's/.*"\([^"]*\)"$/\1/')
  if [ -z "$tag" ]; then
    echo "  ! could not resolve latest $label — skipping"
    return 0
  fi
  asset="${prefix}-${tag}-${TARGET}.tar.gz"
  url="https://github.com/$repo/releases/download/$tag/$asset"
  td=$(mktemp -d)
  if ! curl -fsSL "$url" -o "$td/$asset" 2>/dev/null; then
    echo "  ! no $TARGET build for $label $tag — skipping"
    rm -rf "$td"; return 0
  fi
  if ! tar xzf "$td/$asset" -C "$td" 2>/dev/null; then
    echo "  ! could not extract $label — skipping"
    rm -rf "$td"; return 0
  fi
  binpath=$(find "$td" -type f -name "$bin" 2>/dev/null | head -1)
  if [ -z "$binpath" ]; then
    echo "  ! $bin not found in $label archive — skipping"
    rm -rf "$td"; return 0
  fi
  cp "$binpath" "$INSTALL_DIR/$bin"
  chmod +x "$INSTALL_DIR/$bin"
  rm -rf "$td"
  echo "  + $label $tag -> $INSTALL_DIR/$bin"
}

need_rg=false; command -v rg >/dev/null 2>&1 || need_rg=true
need_fd=false; command -v fd >/dev/null 2>&1 || need_fd=true
if [ "$need_rg" = true ] || [ "$need_fd" = true ]; then
  echo ""
  echo "Installing search tools into $INSTALL_DIR..."
  if [ "$need_rg" = true ]; then install_tool ripgrep BurntSushi/ripgrep ripgrep rg; fi
  if [ "$need_fd" = true ]; then install_tool fd sharkdp/fd fd fd; fi
fi

# Anything still absent (not on PATH and not freshly dropped into INSTALL_DIR) —
# point the user at their package manager. filu still runs; only Find needs rg.
still=""
if ! command -v rg >/dev/null 2>&1 && [ ! -x "$INSTALL_DIR/rg" ]; then still="$still ripgrep"; fi
if ! command -v fd >/dev/null 2>&1 && [ ! -x "$INSTALL_DIR/fd" ]; then still="$still fd"; fi
if [ -n "$still" ]; then
  echo ""
  echo "NOTE: still missing:$still — filu runs, but Find (f) needs ripgrep"
  echo "(fd only speeds up listing; it falls back to a built-in walk). Install via:"
  echo "  macOS:          brew install ripgrep fd"
  echo "  Debian/Ubuntu:  sudo apt install ripgrep fd-find     # fd binary is 'fdfind'"
  echo "  Fedora:         sudo dnf install ripgrep fd-find"
  echo "  Arch:           sudo pacman -S ripgrep fd"
fi

echo ""
echo "Run 'filu' to launch."
