# vbas-autosuggest zsh integration (M2)
#
# Usage: source this file from your .zshrc:
#
#   export VBAS_SPECS_DIR=/path/to/vbas/specs
#   source /path/to/vbas/shell/zsh/vbas.zsh
#
# On Tab, calls `vbas complete --interactive`. The Go binary owns the
# dropdown UI (drawn on /dev/tty) and writes the chosen value to stdout.
#
# vbas exit code dispatch:
#   0 + non-empty stdout → replace trailing token with the value
#   0 + empty stdout     → user cancelled (Esc/Ctrl-C); leave buffer alone
#   non-zero             → no spec / no matches / error → default zsh completion

: ${VBAS_BIN:=vbas}

if (( ! ${+commands[$VBAS_BIN]} )) && [[ ! -x "$VBAS_BIN" ]]; then
  echo "vbas: binary '$VBAS_BIN' not found in PATH; hook not installed" >&2
  return 1
fi

_vbas_widget() {
  emulate -L zsh

  local buffer="$LBUFFER"
  local rbuffer="$RBUFFER"

  local pick rc
  pick="$("$VBAS_BIN" complete --buffer "$buffer" --cursor "$CURSOR" --interactive 2>/dev/null)"
  rc=$?

  if (( rc != 0 )); then
    zle expand-or-complete
    return
  fi

  if [[ -z "$pick" ]]; then
    # User cancelled — repaint and leave buffer untouched.
    zle redisplay
    return
  fi

  # Replace trailing partial token (or append if buffer ends in space).
  local newbuf
  if [[ -z "$buffer" || "$buffer" == *' ' ]]; then
    newbuf="${buffer}${pick} "
  else
    local prefix="${buffer% *}"
    if [[ "$prefix" == "$buffer" ]]; then
      newbuf="${pick} "
    else
      newbuf="${prefix} ${pick} "
    fi
  fi

  LBUFFER="$newbuf"
  RBUFFER="$rbuffer"
  zle redisplay
}

zle -N _vbas_widget
bindkey '^I' _vbas_widget
