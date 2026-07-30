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

# filu keeps state under its config dir: XDG_CONFIG_HOME wins on every platform
# (matching the binary), else macOS uses Library/Application Support and Linux
# ~/.config.
if [ -n "$XDG_CONFIG_HOME" ]; then
  CONFIG_DIR="$XDG_CONFIG_HOME/filu"
elif [ "$(uname -s)" = "Darwin" ]; then
  CONFIG_DIR="$HOME/Library/Application Support/filu"
else
  CONFIG_DIR="$HOME/.config/filu"
fi
if [ -d "$CONFIG_DIR" ]; then
  printf "Remove filu state in %s? [y/N]: " "$CONFIG_DIR"
  # Read the answer from the terminal, not stdin: under `curl ... | sh` stdin is
  # the script itself, so a plain `read` consumes script text (or hits EOF)
  # instead of the keypress. No controlling terminal (cron, nohup) -> keep the
  # state, the safe default, rather than deleting on a misread.
  if read -r answer < /dev/tty 2>/dev/null; then
    case "$answer" in
      y|Y|yes|YES)
        rm -rf "$CONFIG_DIR"
        echo "removed $CONFIG_DIR"
        ;;
      *)
        echo "kept $CONFIG_DIR"
        ;;
    esac
  else
    echo ""
    echo "kept $CONFIG_DIR (no terminal to confirm on)"
  fi
fi

echo ""
echo "filu uninstalled. (ripgrep / fd, if installed for Search, are left in place.)"
