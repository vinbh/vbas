# vbas-autosuggest zsh integration (M2 + M4 + M4.1 + M4.2)
#
# Usage: source this file from your .zshrc:
#
#   export VBAS_SPECS_DIR=/path/to/vbas/specs
#   source /path/to/vbas/shell/zsh/vbas.zsh
#
# Three triggers, all opening the same dropdown UI:
#
#   * Tab — explicit user trigger (always fires; bypasses suppression).
#   * Typing a known command name (e.g., `git`) — auto-trigger via the
#     self-insert override; opens the subcommand dropdown immediately.
#   * Space after a known command — auto-trigger via the space binding.
#
# After one auto-trigger has fired on a command line, further typing on
# the same line does NOT re-open the dropdown (M4.2). This avoids the
# "I'm typing a positional arg, please stop showing me option flags"
# problem. Tab still works as an explicit override. State resets when
# the prompt redraws (zle-line-init).

: ${VBAS_BIN:=vbas}

if (( ! ${+commands[$VBAS_BIN]} )) && [[ ! -x "$VBAS_BIN" ]]; then
  echo "vbas: binary '$VBAS_BIN' not found in PATH; hook not installed" >&2
  return 1
fi

# ----------------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------------

# Cheap stat per call — fine on every keystroke.
_vbas_has_spec() {
  [[ -n "$VBAS_SPECS_DIR" && -f "$VBAS_SPECS_DIR/$1.json" ]]
}

# After auto-trigger fires once, we suppress further auto-triggers as
# long as the user keeps extending the same line. Tab is unaffected.
typeset -g _VBAS_CASCADED_PREFIX=""

_vbas_should_suppress() {
  [[ -n "$_VBAS_CASCADED_PREFIX" && "$LBUFFER" == "$_VBAS_CASCADED_PREFIX"* ]]
}

# Invoke vbas's interactive dropdown for the current LBUFFER. Returns:
#   0 — pick applied to LBUFFER
#   1 — vbas had no matches (caller decides whether to fall through)
#   2 — user cancelled (LBUFFER unchanged from caller's perspective)
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

# Cascade: keep opening the next-level dropdown after each pick.
# Stops on option flag (-x), no-match, cancel, or buffer not ending in space.
_vbas_cascade() {
  while true; do
    [[ "$LBUFFER" == *' ' ]] || break
    local first_token="${LBUFFER%% *}"
    [[ -n "$first_token" ]] && _vbas_has_spec "$first_token" || break

    # If the most recent completed token is an option flag, stop —
    # cascading would just re-show the same option list.
    local trimmed="${LBUFFER% }"
    local last="${trimmed##* }"
    [[ "$last" != -* ]] || break

    # Flush LBUFFER to the terminal BEFORE spawning the dropdown.
    # ZLE normally only redraws after a widget returns; if we skip this,
    # the dropdown subprocess would save its anchor cursor at the screen
    # state from the last keystroke (one char short of LBUFFER) and the
    # last typed char would appear missing until after the dropdown
    # closes and the widget returns.
    zle -R

    _vbas_dropdown_core
    case $? in
      0) ;;       # picked something; check if next level cascades
      *) break ;; # cancelled or no matches — stop
    esac
  done
  zle redisplay
}

# Reset auto-trigger state at the start of each new prompt.
_vbas_reset_state() {
  _VBAS_CASCADED_PREFIX=""
}
zle -N zle-line-init _vbas_reset_state

# ----------------------------------------------------------------------------
# Tab — explicit dropdown trigger (always fires)
# ----------------------------------------------------------------------------

_vbas_widget() {
  emulate -L zsh
  _vbas_dropdown_core
  case $? in
    0) _vbas_cascade
       _VBAS_CASCADED_PREFIX="$LBUFFER" ;;
    1) zle expand-or-complete ;;
    2) zle redisplay ;;
  esac
}
zle -N _vbas_widget
bindkey '^I' _vbas_widget

# ----------------------------------------------------------------------------
# M4 — Space after a known command auto-opens the dropdown
# ----------------------------------------------------------------------------

_vbas_smart_space() {
  emulate -L zsh
  zle .self-insert

  _vbas_should_suppress && return

  if [[ "$LBUFFER" == *' ' ]]; then
    local first_token="${LBUFFER%% *}"
    if [[ -n "$first_token" ]] && _vbas_has_spec "$first_token"; then
      _vbas_cascade
      _VBAS_CASCADED_PREFIX="$LBUFFER"
    fi
  fi
}
zle -N _vbas_smart_space
bindkey ' ' _vbas_smart_space

# ----------------------------------------------------------------------------
# M4.1 — typing a known command name (no space yet) also auto-opens
# ----------------------------------------------------------------------------

# Overrides the default self-insert. Every printable char goes through
# here; we only act when LBUFFER is exactly a known command name with
# no space yet. The space binding (above) takes precedence for ' '.
_vbas_smart_self_insert() {
  emulate -L zsh
  zle .self-insert

  _vbas_should_suppress && return

  if [[ "$LBUFFER" != *' '* ]] && _vbas_has_spec "$LBUFFER"; then
    local before="$LBUFFER"
    LBUFFER="$LBUFFER "
    _vbas_cascade

    # If cascade fired but the user dismissed without picking anything,
    # roll back the auto-inserted space so their typing flow continues
    # naturally (no confusing extra whitespace).
    if [[ "$LBUFFER" == "$before " ]]; then
      LBUFFER="$before"
    fi

    _VBAS_CASCADED_PREFIX="$LBUFFER"
  fi
}
zle -N self-insert _vbas_smart_self_insert
