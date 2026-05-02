# vbas-autosuggest zsh integration (M2 + M4)
#
# Usage: source this file from your .zshrc:
#
#   export VBAS_SPECS_DIR=/path/to/vbas/specs
#   source /path/to/vbas/shell/zsh/vbas.zsh
#
# Two triggers open the same dropdown UI:
#
#   * Tab — explicit user trigger. Falls through to default zsh
#     completion if vbas has no spec for the first token.
#
#   * Space — auto-trigger. After typing a space when the buffer
#     starts with a known command (one with a JSON spec), the
#     dropdown opens automatically. Repeated picks cascade through
#     subcommand levels until the user lands on an option (—-flag)
#     or there are no more matches.
#
# vbas's own dropdown (in Go) handles type-to-filter, descriptions,
# and selection — see internal/ui/dropdown.go.

: ${VBAS_BIN:=vbas}

if (( ! ${+commands[$VBAS_BIN]} )) && [[ ! -x "$VBAS_BIN" ]]; then
  echo "vbas: binary '$VBAS_BIN' not found in PATH; hook not installed" >&2
  return 1
fi

# ----------------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------------

# _vbas_has_spec — is there a spec file for command name $1?
# Cheap stat per call; fine on every space keystroke.
_vbas_has_spec() {
  [[ -n "$VBAS_SPECS_DIR" && -f "$VBAS_SPECS_DIR/$1.json" ]]
}

# _vbas_dropdown_core — invoke vbas's interactive dropdown.
#
# Returns:
#   0 — the user picked something; LBUFFER updated
#   1 — vbas had no matches (caller decides whether to fall through)
#   2 — user cancelled (Esc/Ctrl-C); LBUFFER unchanged
_vbas_dropdown_core() {
  local buffer="$LBUFFER"
  local rbuffer="$RBUFFER"

  local pick rc
  pick="$("$VBAS_BIN" complete --buffer "$buffer" --cursor "$CURSOR" --interactive 2>/dev/null)"
  rc=$?

  if (( rc != 0 )); then
    return 1
  fi
  if [[ -z "$pick" ]]; then
    return 2
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
  return 0
}

# _vbas_cascade — after a successful pick, keep opening the dropdown
# for the next level as long as it's useful.
#
# Stops when:
#   - LBUFFER doesn't end in a space (user mid-token)
#   - first token has no spec
#   - the most recently completed token is an option (avoids re-showing
#     the same option list at the same nesting level)
#   - dropdown returns no matches or user cancels
_vbas_cascade() {
  while true; do
    [[ "$LBUFFER" == *' ' ]] || break

    local first_token="${LBUFFER%% *}"
    [[ -n "$first_token" ]] && _vbas_has_spec "$first_token" || break

    # If the last completed token is an option (-x / --foo), cascading
    # would just re-show the same options. Stop and let the user fill
    # in the option's argument.
    local trimmed="${LBUFFER% }"
    local last="${trimmed##* }"
    [[ "$last" != -* ]] || break

    _vbas_dropdown_core
    case $? in
      0) ;;       # picked, look for further cascade
      *) break ;; # cancelled or no matches — stop
    esac
  done
  zle redisplay
}

# ----------------------------------------------------------------------------
# Tab — explicit dropdown trigger
# ----------------------------------------------------------------------------

_vbas_widget() {
  emulate -L zsh
  _vbas_dropdown_core
  case $? in
    0) _vbas_cascade ;;
    1) zle expand-or-complete ;;  # fall back to default zsh completion
    2) zle redisplay ;;
  esac
}

zle -N _vbas_widget
bindkey '^I' _vbas_widget

# ----------------------------------------------------------------------------
# Space — smart self-insert that auto-opens the dropdown
# ----------------------------------------------------------------------------

_vbas_smart_space() {
  emulate -L zsh
  zle .self-insert  # insert the space first

  # Auto-open only when:
  #   - buffer now ends in a space (it should, since we just inserted one)
  #   - the first token of the buffer has a spec
  if [[ "$LBUFFER" == *' ' ]]; then
    local first_token="${LBUFFER%% *}"
    if [[ -n "$first_token" ]] && _vbas_has_spec "$first_token"; then
      _vbas_cascade
    fi
  fi
}

zle -N _vbas_smart_space
bindkey ' ' _vbas_smart_space
