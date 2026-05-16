---
name: peek-project-overview
description: "Source-available terminal autosuggest — M0–M8 shipped, v0.3.0 released (2026-05-16), renamed from vbas to peek, deb/rpm/apk packages added"
metadata: 
  node_type: memory
  type: project
  originSessionId: 359993fe-1b21-460a-b941-fe860c086556
---

User is building **peek** (formerly vbas) at `/home/vb666/workspace/vbas` → `github.com/vinbh/peek` — a source-available Linux/macOS terminal autosuggest tool inspired by Amazon Q CLI / Fig dropdown completions.

**Why:** User personally wants this on Linux and may dual-license commercially later. License is **PolyForm Noncommercial 1.0.0** + CLA.

**How to apply:**
- Prioritize Linux/macOS/WSL portability. Favor in-terminal ANSI rendering over floating overlays.
- Stack: **Go** (latency-sensitive), **zsh + bash** (fish deferred), **JSON specs** as strict subset of Fig's TS spec schema.
- User picks polish before advancing milestones — offer polish steps and ask, rather than rushing to next milestone.

## Status as of 2026-05-16

**Latest tag: `v0.3.0`** — released on GitHub with pre-built binaries, Homebrew formula, and deb/rpm/apk/archlinux packages.
**Latest commit on `origin/main`: `a18868e`** (clean, fully pushed).

### Everything shipped (M0–M8):

**Rename: vbas → peek** (done 2026-05-16, commit `a5bc0ae` + `a18868e`)
- Go module: `github.com/vinbh/peek`
- Binary: `peek`, config dir: `~/.config/peek/`, env vars: `PEEK_SOCKET`, `PEEK_SPECS_DIR`, `PEEK_BIN`
- Shell hooks: `shell/zsh/peek.zsh`, `shell/bash/peek.bash`
- Socket paths: `$XDG_RUNTIME_DIR/peek/peek.sock`, fallback `/tmp/peek-$UID.sock`
- GitHub repo: `github.com/vinbh/peek` (renamed; old `vinbh/vbas` redirects)
- Footer brand: `">_ peek"` in dropdown UI

**M8 — nfpms (deb/rpm/apk/archlinux packages)** (commit `75b045d`)
- GoReleaser `nfpms:` section generates `.deb`, `.rpm`, `.apk`, `.archlinux` as release artifacts
- Package layout: binary `/usr/bin/peek`, specs `/usr/share/peek/specs/` (tree copy), hooks `/usr/share/peek/shell/`
- `scripts/pkg-postinstall.sh` — prints source instructions after install
- PolyForm-NC license blocks entry into official Debian/Fedora/pacman repos; AUR/COPR/self-hosted are fine

**M8 — Catalog + go install + Homebrew tap** (commits `769ab58`, `5f6225d`)
- **121 commands** from Fig autocomplete catalog (MIT licensed, headers preserved in `specs/fig/`)
- `peek setup` subcommand — extracts embedded specs+hooks to `~/.config/peek/`, wires rc files
- Homebrew tap: `vinbh/homebrew-peek` (formula `peek`); GoReleaser auto-updates on future tags
- Shell hooks auto-detect specs dir from their own file location (no env var export needed)
- `assets.go` at repo root: `//go:embed specs` + `//go:embed shell/zsh/peek.zsh shell/bash/peek.bash`

**M7 — Bash support** (earlier)
- `shell/bash/peek.bash` — full bash 4.3+ integration: Tab + Space, cascade, suppression, PROMPT_COMMAND prepend
- `install.sh` detects zsh + bash, wires both rc files; reads from `/dev/tty` so works under `curl | bash`
- `zle-line-init` uses `add-zle-hook-widget` (compatible with p10k, starship)

**M0–M6** — scaffold, Tab, dropdown+filter, daemon, auto-open+cascade, 63→121 commands, filepath generators, live generators (git branches, kubectl, docker, aws)

### All install paths:
```bash
# 1. One-liner (no Go needed)
curl -fsSL https://raw.githubusercontent.com/vinbh/peek/main/get.sh | bash

# 2. go install
go install github.com/vinbh/peek/cmd/peek@latest && peek setup

# 3. Homebrew
brew install vinbh/peek/peek

# 4. deb/rpm/apk — download from GitHub releases
```

## Pending / incomplete

- **homebrew-peek PAT setup**: `HOMEBREW_TAP_GITHUB_TOKEN` secret on `vinbh/peek` repo needs a fine-grained PAT (Contents R/W on `vinbh/homebrew-peek`). User needs to create PAT and set secret manually — `gh` CLI not available.
- **M9+**: fish shell, history ranking, LLM fallback, AUR/COPR packages, tokenizer improvements (pipes, redirects, quoted paths)
- **Bash Tab limitation**: `bind -x` cannot fall through to readline's built-in `complete` — Tab is a no-op for commands without a peek spec; Space trigger is unaffected.
- **Option-arg state not modeled**: `git commit -m <Tab>` still shows option flags instead of "next token is string arg"

## Key technical notes

- **Spec lookup precedence:** `<dir>/<cmd>.json` (hand-rolled) wins over `<dir>/fig/<cmd>.json` (imported). Both Go loader and `_peek_has_spec` in shell hooks mirror this — keep in sync.
- **Fig schema "single-or-array" patterns:** `Names`, `Args`, `Templates`, `Generators` use custom `UnmarshalJSON` accepting `T` or `T[]`.
- **Homebrew layout:** binary `<prefix>/bin/peek`, specs `<prefix>/share/peek/specs/`, hooks `<prefix>/share/peek/shell/{zsh,bash}/`. Shell hooks detect via `thisdir/../../specs`.
- **GoReleaser brews:** `skip_upload: auto` (skips pre-releases). First formula pushed manually for v0.3.0 because repo was empty; future releases are automatic.
- **Re-importing specs:** `cd tools/import-fig && npm run import`. Adds to `specs/fig/` — don't hand-edit those files.
- **Daemon:** auto-spawns on first completion, pre-warmed at shell startup via `_peek_ensure_daemon`. Kill after rebuild: `pkill -KILL -f 'peek daemon'`.
- **`assets.go`** must stay at repo root — only location where embed paths `specs` and `shell/...` are valid (can't traverse `..`).
- **CLAUDE.md is gitignored** — local-only, never committed.
- **Module path:** `github.com/vinbh/peek` (not `vb666/peek`). Keep imports consistent.
- **Demo GIFs:** generated via `vhs demo/*.tape` (charmbracelet/vhs + ffmpeg + ttyd). Tape files exist at `demo/git.tape`, `demo/cd.tape`, `demo/files.tape`.

## Prior art
`microsoft/inshellisense`, `withfig/autocomplete` (MIT specs), `carapace-sh/carapace`, `zsh-autosuggestions`, `fzf`, `atuin`.
