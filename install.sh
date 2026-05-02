#!/usr/bin/env bash
#
# vbas installer.
#
# Builds ./cmd/vbas and copies the binary + zsh hook + specs into your
# home directory. Idempotent — running it again refreshes the install.
#
# Override the binary location with PREFIX (default $HOME/.local):
#   PREFIX=/usr/local ./install.sh
#
# To uninstall, run ./uninstall.sh.

set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="$PREFIX/bin"
CONFIG_DIR="$HOME/.config/vbas"

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

echo "==> building"
go build -o ./bin/vbas ./cmd/vbas

echo "==> installing"
mkdir -p "$BIN_DIR" "$CONFIG_DIR/specs"
install -m 0755 ./bin/vbas "$BIN_DIR/vbas"
install -m 0644 ./shell/zsh/vbas.zsh "$CONFIG_DIR/vbas.zsh"
cp -R ./specs/. "$CONFIG_DIR/specs/"
chmod -R u+rw,go-w "$CONFIG_DIR/specs"

# Stop any running daemon so the next completion picks up the fresh binary.
pkill -KILL -f 'vbas daemon' 2>/dev/null || true

cat <<EOF

vbas installed:
  binary  →  $BIN_DIR/vbas
  hook    →  $CONFIG_DIR/vbas.zsh
  specs   →  $CONFIG_DIR/specs/

EOF

if ! echo ":$PATH:" | grep -q ":$BIN_DIR:"; then
  cat <<EOF
WARNING: $BIN_DIR is not in your \$PATH.
Add this to your ~/.zshrc:
    export PATH="$BIN_DIR:\$PATH"

EOF
fi

cat <<EOF
To enable in zsh, add this line to your ~/.zshrc (one time):
    source $CONFIG_DIR/vbas.zsh

Then open a new shell, or run:  source ~/.zshrc
EOF
