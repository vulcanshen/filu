#!/bin/sh
# filu uninstaller for macOS / Linux.
# Usage: curl -fsSL https://raw.githubusercontent.com/vulcanshen/filu/main/uninstall.sh | sh

set -e

CANDIDATES="$HOME/.local/bin/filu /usr/local/bin/filu"

FOUND=""
for path in $CANDIDATES; do
  if [ -f "$path" ]; then
    FOUND="$path"
    break
  fi
done

if [ -z "$FOUND" ]; then
  echo "filu not found in expected locations."
  echo "Checked: $CANDIDATES"
  exit 1
fi

rm "$FOUND"
echo "removed $FOUND"

# filu keeps state under the OS config dir (os.UserConfigDir()/filu).
if [ "$(uname -s)" = "Darwin" ]; then
  CONFIG_DIR="$HOME/Library/Application Support/filu"
else
  CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/filu"
fi
if [ -d "$CONFIG_DIR" ]; then
  printf "Remove filu state in %s? [y/N]: " "$CONFIG_DIR"
  read -r answer
  case "$answer" in
    y|Y|yes|YES)
      rm -rf "$CONFIG_DIR"
      echo "removed $CONFIG_DIR"
      ;;
    *)
      echo "kept $CONFIG_DIR"
      ;;
  esac
fi

echo ""
echo "filu uninstalled. (ripgrep / fd, if installed for Search, are left in place.)"
