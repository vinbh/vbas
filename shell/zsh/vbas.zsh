# vbas-autosuggest zsh integration (M2 + M4 + M4.2)
#
# Usage: source this file from your .zshrc:
#
#   export VBAS_SPECS_DIR=/path/to/vbas/specs
#   source /path/to/vbas/shell/zsh/vbas.zsh
#
# Two triggers, both opening the same dropdown UI:
#
#   * Tab — explicit user trigger (always fires; bypasses suppression).
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
# Specs dir auto-detection
# ----------------------------------------------------------------------------

# Resolve VBAS_SPECS_DIR from this file's own location when it isn't set.
# Supports two layouts without the user having to export anything:
#
#   Standard (~/.config/vbas/):
#     vbas.zsh sits alongside specs/ → specs_dir = thisdir/specs
#
#   Homebrew (…/share/vbas/):
#     vbas.zsh is at …/share/vbas/shell/zsh/vbas.zsh
#     specs are at  …/share/vbas/specs/
#     → specs_dir = thisdir/../../specs (two levels up)
if [[ -z "${VBAS_SPECS_DIR:-}" ]]; then
  _vbas_thisdir="${${(%):-%x}:A:h}"
  if   [[ -d "$_vbas_thisdir/specs" ]];      then VBAS_SPECS_DIR="$_vbas_thisdir/specs"
  elif [[ -d "$_vbas_thisdir/../../specs" ]]; then VBAS_SPECS_DIR="${_vbas_thisdir}/../../specs"
    VBAS_SPECS_DIR="$(cd "$VBAS_SPECS_DIR" && pwd -P)"
  fi
  unset _vbas_thisdir
fi

# ----------------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------------

# Cheap stat per call — fine on every keystroke.
# Mirrors the Go loader's lookup order: hand-rolled $VBAS_SPECS_DIR/<cmd>.json
# wins, then fall back to imported $VBAS_SPECS_DIR/fig/<cmd>.json (M5+).
# Falls back to ~/.config/vbas/specs when VBAS_SPECS_DIR is not set.
_vbas_has_spec() {
  local specs_dir="${VBAS_SPECS_DIR:-$HOME/.config/vbas/specs}"
  [[ -f "$specs_dir/$1.json" || -f "$specs_dir/fig/$1.json" ]]
}

# After auto-trigger fires once, we suppress further auto-triggers as
# long as the user keeps extending the same line. Tab is unaffected.
typeset -g _VBAS_CASCADED_PREFIX=""

_vbas_should_suppress() {
  [[ -n "$_VBAS_CASCADED_PREFIX" && "$LBUFFER" == "$_VBAS_CASCADED_PREFIX"* ]]
}

# Release the cascaded-prefix suppression if LBUFFER is shorter than the
# prefix we recorded. That means the user backspaced past where the cascade
# fired, so they should be allowed to re-trigger by typing forward again.
# Called at the top of the smart_space / smart_self_insert widgets BEFORE
# zle .self-insert, so we see the pre-insert buffer state.
_vbas_release_if_backspaced() {
  if [[ -n "$_VBAS_CASCADED_PREFIX" ]] && (( ${#LBUFFER} < ${#_VBAS_CASCADED_PREFIX} )); then
    _VBAS_CASCADED_PREFIX=""
  fi
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
  # Directory picks (ending in /) get no trailing space so the cascade can
  # immediately drill into the selected directory on the next iteration.
  local newbuf trail
  [[ "$pick" == */ ]] && trail="" || trail=" "
  if [[ -z "$buffer" || "$buffer" == *' ' ]]; then
    newbuf="${buffer}${pick}${trail}"
  else
    local prefix="${buffer% *}"
    if [[ "$prefix" == "$buffer" ]]; then
      newbuf="${pick}${trail}"
    else
      newbuf="${prefix} ${pick}${trail}"
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
    # Continue when buffer ends in a space (new token position) or in /
    # (user picked a directory; drill into it without inserting a space).
    [[ "$LBUFFER" == *' ' || "$LBUFFER" == */ ]] || break
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
zle -N _vbas_reset_state
# Chain into zle-line-init rather than replacing it — add-zle-hook-widget
# appends to the hook list so prompt themes (p10k, starship, etc.) keep
# their own zle-line-init hooks intact.
autoload -Uz add-zle-hook-widget 2>/dev/null
if (( ${+functions[add-zle-hook-widget]} )); then
  add-zle-hook-widget zle-line-init _vbas_reset_state
else
  # Pre-zsh-5.3 fallback: replace (may conflict with theme's zle-line-init).
  zle -N zle-line-init _vbas_reset_state
fi

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
  _vbas_release_if_backspaced
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

# M4.1 (auto-open on typing the command name, before space) was removed.
# It fired too eagerly when a command name was a prefix of another (e.g.
# "vi" triggered while the user was still typing "vim"). Space is the
# unambiguous trigger; Tab still works as an explicit fallback.

# ----------------------------------------------------------------------------
# Pre-warm daemon so the first completion is instant
# ----------------------------------------------------------------------------

_vbas_ensure_daemon() {
  local args=(daemon)
  [[ -n "${VBAS_SPECS_DIR:-}"  ]] && args+=(--specs  "$VBAS_SPECS_DIR")
  [[ -n "${VBAS_SOCKET:-}"     ]] && args+=(--socket "$VBAS_SOCKET")
  "$VBAS_BIN" "${args[@]}" &>/dev/null
}
# Run in background; exits silently if a daemon is already listening.
_vbas_ensure_daemon &!
