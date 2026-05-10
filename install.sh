#!/usr/bin/env bash
#
# peek installer.
#
# Works in two modes:
#   1. Pre-built release archive: a "peek" binary sits next to this script
#      (extracted from a GitHub release tarball). No Go required.
#   2. Source install: builds from ./cmd/peek. Requires Go 1.21+.
#
# Detects installed shells (zsh, bash) and offers to wire each one.
# Idempotent — running it again refreshes the install.
#
# Override the binary location with PREFIX (default $HOME/.local):
#   PREFIX=/usr/local ./install.sh
#
# To uninstall, run ./uninstall.sh.

set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="$PREFIX/bin"
CONFIG_DIR="$HOME/.config/peek"

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# ----------------------------------------------------------------------------
# 1. Build or locate binary
# ----------------------------------------------------------------------------

if [[ -f "./peek" ]]; then
  echo "==> using pre-built binary"
  mkdir -p ./bin
  cp ./peek ./bin/peek
  chmod +x ./bin/peek
elif command -v go &>/dev/null; then
  echo "==> building from source"
  go build -o ./bin/peek ./cmd/peek
else
  echo "error: no pre-built binary found and 'go' is not in PATH." >&2
  echo "  Download a pre-built release from https://github.com/vinbh/peek/releases" >&2
  echo "  or install Go from https://go.dev/dl/ and re-run this script." >&2
  exit 1
fi

# ----------------------------------------------------------------------------
# 2. Install files
# ----------------------------------------------------------------------------

echo "==> installing"
mkdir -p "$BIN_DIR" "$CONFIG_DIR/specs"
install -m 0755 ./bin/peek          "$BIN_DIR/peek"
install -m 0644 ./shell/zsh/peek.zsh  "$CONFIG_DIR/peek.zsh"
install -m 0644 ./shell/bash/peek.bash "$CONFIG_DIR/peek.bash"
cp -R ./specs/. "$CONFIG_DIR/specs/"
chmod -R u+rw,go-w "$CONFIG_DIR/specs"

# Stop any running daemon so the next completion picks up the fresh binary.
pkill -KILL -f 'peek daemon' 2>/dev/null || true

cat <<EOF

peek installed:
  binary  →  $BIN_DIR/peek
  zsh     →  $CONFIG_DIR/peek.zsh
  bash    →  $CONFIG_DIR/peek.bash
  specs   →  $CONFIG_DIR/specs/

EOF

if ! echo ":$PATH:" | grep -q ":$BIN_DIR:"; then
  echo "WARNING: $BIN_DIR is not in your \$PATH."
  echo "  Add to your shell rc:  export PATH=\"$BIN_DIR:\$PATH\""
  echo ""
fi

# ----------------------------------------------------------------------------
# 3. Wire detected shells
# ----------------------------------------------------------------------------

# Offer to add a source line to a shell's rc file.
# Reads answer from /dev/tty so it works even when stdin is a pipe (curl|bash).
_offer_rc() {
  local label="$1" rc="$2" hook="$3"
  local src="source $hook"

  if grep -qF "$src" "$rc" 2>/dev/null; then
    echo "  $label: already enabled in $rc"
    return
  fi

  printf "  Enable peek in %s (%s)? [Y/n] " "$label" "$(basename "$rc")"
  local yn
  read -r yn < /dev/tty
  case "${yn:-Y}" in
    [Yy]*|"")
      printf '\n# peek autosuggest\n%s\n' "$src" >> "$rc"
      echo "  Added. Run: source $rc" ;;
    *)
      echo "  Skipped. Add manually to $rc:"
      echo "      $src" ;;
  esac
}

_noninteractive_hint() {
  local label="$1" rc="$2" hook="$3"
  echo "  $label: add to $rc:"
  echo "      source $hook"
}

IS_INTERACTIVE=0
[[ -t 0 ]] || [[ -t 1 ]] && IS_INTERACTIVE=1

echo "==> shell integration"

HAS_ZSH=0;  command -v zsh  &>/dev/null && HAS_ZSH=1
HAS_BASH=0; command -v bash &>/dev/null && HAS_BASH=1

if (( HAS_ZSH )); then
  if (( IS_INTERACTIVE )); then
    _offer_rc "zsh" "$HOME/.zshrc" "$CONFIG_DIR/peek.zsh"
  else
    _noninteractive_hint "zsh" "$HOME/.zshrc" "$CONFIG_DIR/peek.zsh"
  fi
fi

if (( HAS_BASH )); then
  if (( IS_INTERACTIVE )); then
    _offer_rc "bash" "$HOME/.bashrc" "$CONFIG_DIR/peek.bash"
  else
    _noninteractive_hint "bash" "$HOME/.bashrc" "$CONFIG_DIR/peek.bash"
  fi
fi

echo ""
