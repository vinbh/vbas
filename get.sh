#!/usr/bin/env bash
#
# vbas one-line installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/vinbh/vbas/main/get.sh | bash
#
# What it does:
#   1. Detects OS and architecture
#   2. Downloads the latest release tarball from GitHub
#   3. Extracts it to a temp directory and runs install.sh
#   4. Cleans up
#
# The install puts:
#   binary  -> $HOME/.local/bin/vbas  (override: PREFIX=/usr/local)
#   hook    -> $HOME/.config/vbas/vbas.zsh
#   specs   -> $HOME/.config/vbas/specs/
#
# After install, add this line to your ~/.zshrc:
#   source ~/.config/vbas/vbas.zsh

set -euo pipefail

REPO="vinbh/vbas"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "error: unsupported OS: $OS" >&2
    echo "  Download manually from https://github.com/$REPO/releases" >&2
    exit 1
    ;;
esac

# Detect arch
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  aarch64|arm64)   ARCH="arm64" ;;
  *)
    echo "error: unsupported architecture: $ARCH" >&2
    echo "  Download manually from https://github.com/$REPO/releases" >&2
    exit 1
    ;;
esac

# Resolve latest version tag
echo "==> fetching latest release"
if command -v curl &>/dev/null; then
  FETCH="curl -fsSL"
elif command -v wget &>/dev/null; then
  FETCH="wget -qO-"
else
  echo "error: curl or wget is required" >&2
  exit 1
fi

TAG="$($FETCH "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' \
  | head -1 \
  | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"

if [[ -z "$TAG" ]]; then
  echo "error: could not determine latest release tag" >&2
  echo "  Check https://github.com/$REPO/releases" >&2
  exit 1
fi

echo "==> installing vbas $TAG for ${OS}/${ARCH}"

ARCHIVE="vbas_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$TAG/$ARCHIVE"
DEST="$TMPDIR/$ARCHIVE"

$FETCH "$URL" -o "$DEST" 2>/dev/null || {
  # wget -O syntax
  $FETCH "$URL" > "$DEST"
}

tar -xzf "$DEST" -C "$TMPDIR"

# The archive extracts to a directory named after the archive (without .tar.gz)
EXTRACTED="$TMPDIR/vbas_${OS}_${ARCH}"
if [[ ! -d "$EXTRACTED" ]]; then
  echo "error: unexpected archive layout — expected $EXTRACTED" >&2
  exit 1
fi

cd "$EXTRACTED"
bash install.sh

echo ""
echo "Add this to your ~/.zshrc to enable vbas:"
echo "    source \$HOME/.config/vbas/vbas.zsh"
