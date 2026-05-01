# vbas-autosuggest zsh integration (M1)
#
# Usage: source this file from your .zshrc:
#
#   export VBAS_SPECS_DIR=/path/to/vb-autosuggest/specs
#   source /path/to/vb-autosuggest/shell/zsh/vbas.zsh
#
# Behavior: on Tab, asks `vbas complete` for a suggestion. If exactly one
# match, completes inline. If multiple, defers to default zsh completion
# (the dropdown UI lands in M2). If no match or no spec, also defers.

# Resolve binary, allow override.
: ${VBAS_BIN:=vbas}

# Refuse to install hook if the binary isn't reachable.
if (( ! ${+commands[$VBAS_BIN]} )) && [[ ! -x "$VBAS_BIN" ]]; then
  echo "vbas: binary '$VBAS_BIN' not found in PATH; hook not installed" >&2
  return 1
fi

_vbas_widget() {
  emulate -L zsh

  local buffer="$LBUFFER"
  local rbuffer="$RBUFFER"

  # Get suggestions from vbas. One per line.
  local -a suggestions
  suggestions=("${(@f)$("$VBAS_BIN" complete --buffer "$buffer" --cursor "$CURSOR" 2>/dev/null)}")

  # Strip empty trailing element from `${(@f)}` split.
  if (( ${#suggestions} > 0 )) && [[ -z "${suggestions[-1]}" ]]; then
    suggestions=("${suggestions[@]:0:${#suggestions[@]}-1}")
  fi

  if (( ${#suggestions} == 0 )); then
    zle expand-or-complete
    return
  fi

  if (( ${#suggestions} > 1 )); then
    # Multiple candidates — fall through until M2 lands a real picker.
    zle expand-or-complete
    return
  fi

  local pick="${suggestions[1]}"

  # Replace the trailing partial token (or append if buffer ends with space).
  local newbuf
  if [[ -z "$buffer" || "$buffer" == *' ' ]]; then
    newbuf="${buffer}${pick} "
  else
    # Strip last word, append suggestion.
    local prefix="${buffer% *}"
    if [[ "$prefix" == "$buffer" ]]; then
      # Single-token buffer with no spaces.
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
