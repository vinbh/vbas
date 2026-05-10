# peek zsh integration
#
# Usage: source this file from your .zshrc:
#
#   source ~/.config/peek/peek.zsh
#
# Two triggers, both opening the same dropdown UI:
#
#   * Tab   — explicit user trigger (always fires; bypasses suppression).
#   * Space after a known command — auto-trigger via the space binding.
#
# After one auto-trigger has fired on a command line, further typing on
# the same line does NOT re-open the dropdown. This avoids the
# "I'm typing a positional arg, please stop showing me option flags"
# problem. Tab still works as an explicit override. State resets when
# the prompt redraws (zle-line-init).

: ${PEEK_BIN:=peek}

if (( ! ${+commands[$PEEK_BIN]} )) && [[ ! -x "$PEEK_BIN" ]]; then
  echo "peek: binary '$PEEK_BIN' not found in PATH; hook not installed" >&2
  return 1
fi

# ----------------------------------------------------------------------------
# Specs dir auto-detection
# ----------------------------------------------------------------------------

# Resolve PEEK_SPECS_DIR from this file's own location when it isn't set.
# Supports two layouts without the user having to export anything:
#
#   Standard (~/.config/peek/):
#     peek.zsh sits alongside specs/ → specs_dir = thisdir/specs
#
#   Homebrew / deb / rpm (…/share/peek/):
#     peek.zsh is at …/share/peek/shell/zsh/peek.zsh
#     specs are at   …/share/peek/specs/
#     → specs_dir = thisdir/../../specs (two levels up)
if [[ -z "${PEEK_SPECS_DIR:-}" ]]; then
  _peek_thisdir="${${(%):-%x}:A:h}"
  if   [[ -d "$_peek_thisdir/specs" ]];      then PEEK_SPECS_DIR="$_peek_thisdir/specs"
  elif [[ -d "$_peek_thisdir/../../specs" ]]; then PEEK_SPECS_DIR="${_peek_thisdir}/../../specs"
    PEEK_SPECS_DIR="$(cd "$PEEK_SPECS_DIR" && pwd -P)"
  fi
  unset _peek_thisdir
fi

# ----------------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------------

# Cheap stat per call — fine on every keystroke.
# Mirrors the Go loader's lookup order: hand-rolled $PEEK_SPECS_DIR/<cmd>.json
# wins, then fall back to imported $PEEK_SPECS_DIR/fig/<cmd>.json (M5+).
# Falls back to ~/.config/peek/specs when PEEK_SPECS_DIR is not set.
_peek_has_spec() {
  local specs_dir="${PEEK_SPECS_DIR:-$HOME/.config/peek/specs}"
  [[ -f "$specs_dir/$1.json" || -f "$specs_dir/fig/$1.json" ]]
}

# After auto-trigger fires once, we suppress further auto-triggers as
# long as the user keeps extending the same line. Tab is unaffected.
typeset -g _PEEK_CASCADED_PREFIX=""

_peek_should_suppress() {
  [[ -n "$_PEEK_CASCADED_PREFIX" && "$LBUFFER" == "$_PEEK_CASCADED_PREFIX"* ]]
}

# Release the cascaded-prefix suppression if LBUFFER is shorter than the
# prefix we recorded. That means the user backspaced past where the cascade
# fired, so they should be allowed to re-trigger by typing forward again.
# Called at the top of the smart_space widget BEFORE zle .self-insert, so
# we see the pre-insert buffer state.
_peek_release_if_backspaced() {
  if [[ -n "$_PEEK_CASCADED_PREFIX" ]] && (( ${#LBUFFER} < ${#_PEEK_CASCADED_PREFIX} )); then
    _PEEK_CASCADED_PREFIX=""
  fi
}

# Invoke peek's interactive dropdown for the current LBUFFER. Returns:
#   0 — pick applied to LBUFFER
#   1 — peek had no matches (caller decides whether to fall through)
#   2 — user cancelled (LBUFFER unchanged from caller's perspective)
_peek_dropdown_core() {
  local buffer="$LBUFFER"
  local rbuffer="$RBUFFER"

  local pick rc
  pick="$("$PEEK_BIN" complete --buffer "$buffer" --cursor "$CURSOR" --interactive 2>/dev/null)"
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
_peek_cascade() {
  while true; do
    # Continue when buffer ends in a space (new token position) or in /
    # (user picked a directory; drill into it without inserting a space).
    [[ "$LBUFFER" == *' ' || "$LBUFFER" == */ ]] || break
    local first_token="${LBUFFER%% *}"
    [[ -n "$first_token" ]] && _peek_has_spec "$first_token" || break

    # If the most recent completed token is an option flag, stop —
    # cascading would just re-show the same option list.
    local trimmed="${LBUFFER% }"
    local last="${trimmed##* }"
    [[ "$last" != -* ]] || break

    # Flush LBUFFER to the terminal BEFORE spawning the dropdown.
    # ZLE normally only redraws after a widget returns; if we skip this,
    # the dropdown subprocess would save its cursor anchor at the screen
    # state from the last keystroke (one char short of LBUFFER) and the
    # last typed char would appear missing until after the dropdown
    # closes and the widget returns.
    zle -R

    _peek_dropdown_core
    case $? in
      0) ;;       # picked something; check if next level cascades
      *) break ;; # cancelled or no matches — stop
    esac
  done
  zle redisplay
}

# Reset auto-trigger state at the start of each new prompt.
_peek_reset_state() {
  _PEEK_CASCADED_PREFIX=""
}
zle -N _peek_reset_state
# Chain into zle-line-init rather than replacing it — add-zle-hook-widget
# appends to the hook list so prompt themes (p10k, starship, etc.) keep
# their own zle-line-init hooks intact.
autoload -Uz add-zle-hook-widget 2>/dev/null
if (( ${+functions[add-zle-hook-widget]} )); then
  add-zle-hook-widget zle-line-init _peek_reset_state
else
  # Pre-zsh-5.3 fallback: replace (may conflict with theme's zle-line-init).
  zle -N zle-line-init _peek_reset_state
fi

# ----------------------------------------------------------------------------
# Tab — explicit dropdown trigger (always fires)
# ----------------------------------------------------------------------------

_peek_widget() {
  emulate -L zsh
  _peek_dropdown_core
  case $? in
    0) _peek_cascade
       _PEEK_CASCADED_PREFIX="$LBUFFER" ;;
    1) zle expand-or-complete ;;
    2) zle redisplay ;;
  esac
}
zle -N _peek_widget
bindkey '^I' _peek_widget

# ----------------------------------------------------------------------------
# M4 — Space after a known command auto-opens the dropdown
# ----------------------------------------------------------------------------

_peek_smart_space() {
  emulate -L zsh
  _peek_release_if_backspaced
  zle .self-insert

  _peek_should_suppress && return

  if [[ "$LBUFFER" == *' ' ]]; then
    local first_token="${LBUFFER%% *}"
    if [[ -n "$first_token" ]] && _peek_has_spec "$first_token"; then
      _peek_cascade
      _PEEK_CASCADED_PREFIX="$LBUFFER"
    fi
  fi
}
zle -N _peek_smart_space
bindkey ' ' _peek_smart_space

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
_peek_ensure_daemon &!
