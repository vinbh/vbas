# vbas bash integration
#
# Usage: source this file from your .bashrc:
#
#   source ~/.config/vbas/vbas.bash
#
# Requires bash 4.3+ (bind -x became reliable in 4.3).
#
# Two triggers:
#   Tab   — opens the vbas dropdown if a spec exists for the current command;
#            does nothing for unknown commands (see note below).
#   Space — inserts space, then auto-opens dropdown after any known command.
#
# Note on Tab fallthrough: bash's bind -x does not allow calling readline's
# internal complete builtin from within a bound function, so Tab cannot fall
# through to native bash completion when vbas has no spec. For commands vbas
# doesn't know about, Tab is a no-op; use Ctrl-I (same key) after removing
# the binding if you need it back, or just rely on the Space auto-trigger.

: ${VBAS_BIN:=vbas}

if ! command -v "$VBAS_BIN" &>/dev/null && [[ ! -x "$VBAS_BIN" ]]; then
  echo "vbas: binary '$VBAS_BIN' not found in PATH; hook not installed" >&2
  return 1
fi

if (( BASH_VERSINFO[0] < 4 || ( BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] < 3 ) )); then
  echo "vbas: bash 4.3+ required (have $BASH_VERSION); hook not installed" >&2
  return 1
fi

# ----------------------------------------------------------------------------
# Specs dir auto-detection
# ----------------------------------------------------------------------------

# Resolve VBAS_SPECS_DIR from this file's own location when it isn't set.
# Supports two layouts without the user having to export anything:
#
#   Standard (~/.config/vbas/):
#     vbas.bash sits alongside specs/ → specs_dir = thisdir/specs
#
#   Homebrew (…/share/vbas/):
#     vbas.bash is at …/share/vbas/shell/bash/vbas.bash
#     specs are at   …/share/vbas/specs/
#     → specs_dir = thisdir/../../specs (two levels up)
if [[ -z "${VBAS_SPECS_DIR:-}" ]]; then
  _vbas_thisdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
  if   [[ -d "$_vbas_thisdir/specs" ]];      then VBAS_SPECS_DIR="$_vbas_thisdir/specs"
  elif [[ -d "$_vbas_thisdir/../../specs" ]]; then VBAS_SPECS_DIR="$(cd "$_vbas_thisdir/../../specs" && pwd -P)"
  fi
  unset _vbas_thisdir
fi

# ----------------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------------

# Mirrors the Go loader's lookup order: hand-rolled <specs>/<cmd>.json wins
# over imported <specs>/fig/<cmd>.json. Falls back to ~/.config/vbas/specs
# when VBAS_SPECS_DIR is unset.
_vbas_has_spec() {
  local specs_dir="${VBAS_SPECS_DIR:-$HOME/.config/vbas/specs}"
  [[ -f "$specs_dir/$1.json" || -f "$specs_dir/fig/$1.json" ]]
}

_VBAS_CASCADED_PREFIX=""

_vbas_should_suppress() {
  [[ -n "$_VBAS_CASCADED_PREFIX" && "$READLINE_LINE" == "$_VBAS_CASCADED_PREFIX"* ]]
}

_vbas_release_if_backspaced() {
  if [[ -n "$_VBAS_CASCADED_PREFIX" ]] && (( ${#READLINE_LINE} < ${#_VBAS_CASCADED_PREFIX} )); then
    _VBAS_CASCADED_PREFIX=""
  fi
}

# Invoke the vbas dropdown for the current readline buffer. Returns:
#   0 — pick applied to READLINE_LINE / READLINE_POINT
#   1 — no spec or no matches (vbas exited 2)
#   2 — user cancelled (pick was empty)
_vbas_bash_dropdown_core() {
  local buffer="$READLINE_LINE"
  local cursor="$READLINE_POINT"

  local pick rc
  pick="$("$VBAS_BIN" complete --buffer "$buffer" --cursor "$cursor" --interactive 2>/dev/null)"
  rc=$?

  if (( rc != 0 )); then
    return 1
  fi
  if [[ -z "$pick" ]]; then
    return 2
  fi

  # Build new buffer — same logic as the zsh hook.
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

  READLINE_LINE="$newbuf"
  READLINE_POINT="${#newbuf}"
  return 0
}

# Cascade: keep opening the next-level dropdown after each pick.
# Stops on an option flag (-x), no-match, cancel, or when the buffer no
# longer ends in a space or slash.
#
# Bash has no equivalent of zsh's `zle -R`, so intermediate buffer states
# are not flushed to the terminal between cascade levels. The dropdown UI
# still works correctly because each invocation receives --buffer with the
# updated value; the only effect is that the prompt text behind the dropdown
# briefly shows the previous state.
_vbas_bash_cascade() {
  while true; do
    [[ "$READLINE_LINE" == *' ' || "$READLINE_LINE" == */ ]] || break
    local first_token="${READLINE_LINE%% *}"
    [[ -n "$first_token" ]] && _vbas_has_spec "$first_token" || break

    local trimmed="${READLINE_LINE% }"
    local last="${trimmed##* }"
    [[ "$last" != -* ]] || break

    _vbas_bash_dropdown_core
    case $? in
      0) ;;
      *) break ;;
    esac
  done
}

# Reset suppression state at the start of each new prompt.
_vbas_reset_state() {
  _VBAS_CASCADED_PREFIX=""
}
# Prepend to PROMPT_COMMAND rather than replacing it (preserves existing hooks).
PROMPT_COMMAND="_vbas_reset_state${PROMPT_COMMAND:+; $PROMPT_COMMAND}"

# ----------------------------------------------------------------------------
# Tab — explicit dropdown trigger
# ----------------------------------------------------------------------------

_vbas_bash_widget() {
  local first_token="${READLINE_LINE%% *}"
  if [[ -z "$first_token" ]] || ! _vbas_has_spec "$first_token"; then
    return 0
  fi

  _vbas_bash_dropdown_core
  case $? in
    0) _vbas_bash_cascade
       _VBAS_CASCADED_PREFIX="$READLINE_LINE" ;;
  esac
}
bind -x '"\t": _vbas_bash_widget'

# ----------------------------------------------------------------------------
# Space — auto-opens dropdown after a known command
# ----------------------------------------------------------------------------

_vbas_bash_smart_space() {
  _vbas_release_if_backspaced

  # Insert the space at the cursor position.
  READLINE_LINE="${READLINE_LINE:0:$READLINE_POINT} ${READLINE_LINE:$READLINE_POINT}"
  READLINE_POINT=$(( READLINE_POINT + 1 ))

  _vbas_should_suppress && return

  # Only auto-trigger when the text up to the cursor ends in a space
  # (i.e. cursor is at a new token boundary, not editing mid-line).
  local before="${READLINE_LINE:0:$READLINE_POINT}"
  if [[ "$before" == *' ' ]]; then
    local first_token="${before%% *}"
    if [[ -n "$first_token" ]] && _vbas_has_spec "$first_token"; then
      _vbas_bash_cascade
      _VBAS_CASCADED_PREFIX="$READLINE_LINE"
    fi
  fi
}
bind -x '" ": _vbas_bash_smart_space'

# ----------------------------------------------------------------------------
# Pre-warm daemon so the first completion is instant
# ----------------------------------------------------------------------------

_vbas_ensure_daemon() {
  local args=(daemon)
  [[ -n "${VBAS_SPECS_DIR:-}" ]] && args+=(--specs  "$VBAS_SPECS_DIR")
  [[ -n "${VBAS_SOCKET:-}"    ]] && args+=(--socket "$VBAS_SOCKET")
  "$VBAS_BIN" "${args[@]}" &>/dev/null
}
# Run in background; exits silently if a daemon is already listening.
_vbas_ensure_daemon &>/dev/null &
disown
