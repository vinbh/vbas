#!/usr/bin/env bash
#
# Removes what install.sh installed. Hand-rolled specs you've placed at
# ~/.config/vbas/specs/<cmd>.json (i.e. NOT in the imported fig/ subdir)
# are preserved.

set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="$PREFIX/bin"
CONFIG_DIR="$HOME/.config/vbas"

pkill -KILL -f 'vbas daemon' 2>/dev/null || true
rm -f "$BIN_DIR/vbas"
rm -f "$CONFIG_DIR/vbas.zsh"
rm -rf "$CONFIG_DIR/specs/fig"

# Tidy up empty parent dirs so an uninstall+reinstall is clean.
rmdir "$CONFIG_DIR/specs" 2>/dev/null || true
rmdir "$CONFIG_DIR" 2>/dev/null || true

echo "vbas uninstalled."
if [[ -d "$CONFIG_DIR" ]]; then
  echo "(kept $CONFIG_DIR — it still contains hand-rolled specs)"
fi
echo
echo "Remove this line from your ~/.zshrc:"
echo "    source $CONFIG_DIR/vbas.zsh"
