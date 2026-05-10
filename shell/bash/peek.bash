# peek bash integration
#
# Usage: source this file from your .bashrc:
#
#   source ~/.config/peek/peek.bash
#
# Requires bash 4.3+ (bind -x became reliable in 4.3).
#
# Two triggers:
#   Tab   — opens the peek dropdown if a spec exists for the current command;
#            does nothing for unknown commands (see note below).
#   Space — inserts space, then auto-opens dropdown after any known command.
#
# Note on Tab fallthrough: bash's bind -x does not allow calling readline's
# internal complete builtin from within a bound function, so Tab cannot fall
# through to native bash completion when peek has no spec. For commands peek
# doesn't know about, Tab is a no-op; use Ctrl-I (same key) after removing
# the binding if you need it back, or just rely on the Space auto-trigger.

: ${PEEK_BIN:=peek}

if ! command -v "$PEEK_BIN" &>/dev/null && [[ ! -x "$PEEK_BIN" ]]; then
  echo "peek: binary '$PEEK_BIN' not found in PATH; hook not installed" >&2
  return 1
fi

if (( BASH_VERSINFO[0] < 4 || ( BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] < 3 ) )); then
  echo "peek: bash 4.3+ required (have $BASH_VERSION); hook not installed" >&2
  return 1
fi

# ----------------------------------------------------------------------------
# Specs dir auto-detection
# ----------------------------------------------------------------------------

# Resolve PEEK_SPECS_DIR from this file's own location when it isn't set.
# Supports two layouts without the user having to export anything:
#
#   Standard (~/.config/peek/):
#     peek.bash sits alongside specs/ → specs_dir = thisdir/specs
#
#   Homebrew / deb / rpm (…/share/peek/):
#     peek.bash is at …/share/peek/shell/bash/peek.bash
#     specs are at    …/share/peek/specs/
#     → specs_dir = thisdir/../../specs (two levels up)
if [[ -z "${PEEK_SPECS_DIR:-}" ]]; then
  _peek_thisdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
  if   [[ -d "$_peek_thisdir/specs" ]];      then PEEK_SPECS_DIR="$_peek_thisdir/specs"
  elif [[ -d "$_peek_thisdir/../../specs" ]]; then PEEK_SPECS_DIR="$(cd "$_peek_thisdir/../../specs" && pwd -P)"
  fi
  unset _peek_thisdir
fi

# ----------------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------------

# Mirrors the Go loader's lookup order: hand-rolled <specs>/<cmd>.json wins
# over imported <specs>/fig/<cmd>.json. Falls back to ~/.config/peek/specs
# when PEEK_SPECS_DIR is unset.
_peek_has_spec() {
  local specs_dir="${PEEK_SPECS_DIR:-$HOME/.config/peek/specs}"
  [[ -f "$specs_dir/$1.json" || -f "$specs_dir/fig/$1.json" ]]
}

_PEEK_CASCADED_PREFIX=""

_peek_should_suppress() {
  [[ -n "$_PEEK_CASCADED_PREFIX" && "$READLINE_LINE" == "$_PEEK_CASCADED_PREFIX"* ]]
}

_peek_release_if_backspaced() {
  if [[ -n "$_PEEK_CASCADED_PREFIX" ]] && (( ${#READLINE_LINE} < ${#_PEEK_CASCADED_PREFIX} )); then
    _PEEK_CASCADED_PREFIX=""
  fi
}

# Invoke the peek dropdown for the current readline buffer. Returns:
#   0 — pick applied to READLINE_LINE / READLINE_POINT
#   1 — no spec or no matches (peek exited 2)
#   2 — user cancelled (pick was empty)
_peek_bash_dropdown_core() {
  local buffer="$READLINE_LINE"
  local cursor="$READLINE_POINT"

  local pick rc
  pick="$("$PEEK_BIN" complete --buffer "$buffer" --cursor "$cursor" --interactive 2>/dev/null)"
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
_peek_bash_cascade() {
  while true; do
    [[ "$READLINE_LINE" == *' ' || "$READLINE_LINE" == */ ]] || break
    local first_token="${READLINE_LINE%% *}"
    [[ -n "$first_token" ]] && _peek_has_spec "$first_token" || break

    local trimmed="${READLINE_LINE% }"
    local last="${trimmed##* }"
    [[ "$last" != -* ]] || break

    _peek_bash_dropdown_core
    case $? in
      0) ;;
      *) break ;;
    esac
  done
}

# Reset suppression state at the start of each new prompt.
_peek_reset_state() {
  _PEEK_CASCADED_PREFIX=""
}
# Prepend to PROMPT_COMMAND rather than replacing it (preserves existing hooks).
PROMPT_COMMAND="_peek_reset_state${PROMPT_COMMAND:+; $PROMPT_COMMAND}"

# ----------------------------------------------------------------------------
# Tab — explicit dropdown trigger
# ----------------------------------------------------------------------------

_peek_bash_widget() {
  local first_token="${READLINE_LINE%% *}"
  if [[ -z "$first_token" ]] || ! _peek_has_spec "$first_token"; then
    return 0
  fi

  _peek_bash_dropdown_core
  case $? in
    0) _peek_bash_cascade
       _PEEK_CASCADED_PREFIX="$READLINE_LINE" ;;
  esac
}
bind -x '"\t": _peek_bash_widget'

# ----------------------------------------------------------------------------
# Space — auto-opens dropdown after a known command
# ----------------------------------------------------------------------------

_peek_bash_smart_space() {
  _peek_release_if_backspaced

  # Insert the space at the cursor position.
  READLINE_LINE="${READLINE_LINE:0:$READLINE_POINT} ${READLINE_LINE:$READLINE_POINT}"
  READLINE_POINT=$(( READLINE_POINT + 1 ))

  _peek_should_suppress && return

  # Only auto-trigger when the text up to the cursor ends in a space
  # (i.e. cursor is at a new token boundary, not editing mid-line).
  local before="${READLINE_LINE:0:$READLINE_POINT}"
  if [[ "$before" == *' ' ]]; then
    local first_token="${before%% *}"
    if [[ -n "$first_token" ]] && _peek_has_spec "$first_token"; then
      _peek_bash_cascade
      _PEEK_CASCADED_PREFIX="$READLINE_LINE"
    fi
  fi
}
bind -x '" ": _peek_bash_smart_space'

# ----------------------------------------------------------------------------
# Pre-warm daemon so the first completion is instant
# ----------------------------------------------------------------------------

_peek_ensure_daemon() {
  local args=(daemon)
  [[ -n "${PEEK_SPECS_DIR:-}" ]] && args+=(--specs  "$PEEK_SPECS_DIR")
  [[ -n "${PEEK_SOCKET:-}"    ]] && args+=(--socket "$PEEK_SOCKET")
  "$PEEK_BIN" "${args[@]}" &>/dev/null
}
# Run in background; exits silently if a daemon is already listening.
_peek_ensure_daemon &>/dev/null &
disown
