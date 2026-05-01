# vbas

**Smart command suggestions for the Linux terminal — Q-style dropdown, source-available.**

```
$ git c
▍ checkout              Switch branches or restore working tree files
▍ cherry-pick           Apply changes from existing commits
▍ clean                 Remove untracked files from the working tree
▍ clone                 Clone a repository into a new directory
▍ commit                Record changes to the repository
─── vbas ─── ↑↓ select · ⏎ accept · esc cancel · 1/5
```

Tab opens a dropdown · type to filter · arrows navigate · Tab/Enter accept · Esc cancel.

> ⚠️ **Pre-MVP.** Currently ships with a hand-rolled `git` spec only. The catalog grows in M5 (Fig spec import). Usable, fun, not yet a daily driver.

## Why?

[Amazon Q CLI](https://aws.amazon.com/q/developer/cli/)'s smart shell autocomplete is excellent — but it's macOS-only. [Fig](https://fig.io/), its predecessor, was also macOS-first. On Linux you've had two real choices: [`zsh-autosuggestions`](https://github.com/zsh-users/zsh-autosuggestions) (single-line greyed-out ghost text), or zsh's built-in completion (functional but visually noisy).

**vbas brings the Q-style inline dropdown to Linux** — built in Go, free for personal use, source-available, designed to grow.

## What works today

- **Tab opens an inline dropdown** of subcommands and flags, sourced from a JSON spec
- **Type to filter** — case-insensitive substring match on name or description
- **Arrow keys navigate**, Tab/Enter accept, Esc cancel
- **Distinctive UI** — cyan left-edge bar, position counter, key hints in the footer (so you can tell vbas apart from zsh's default menuselect at a glance)
- **Built-in spec for `git`** (~22 subcommands and common flags)
- **Single static Go binary** — no Node, no runtime deps

## What it can't do yet

| Capability | Lands in |
|---|---|
| As-you-type suggestions (no Tab needed) | M4 |
| Sub-millisecond steady-state latency | M3 (long-running daemon) |
| Spec coverage for thousands of CLIs | M5 (import from Fig autocomplete) |
| bash and fish | M7 |
| LLM fallback for unknown commands | post-M7 |

A few things vbas **won't** do, intentionally:

- **No floating overlay** like Q on macOS. Linux has no universal cursor-pixel API and Wayland blocks the trick for security. vbas renders inline below your prompt — same approach as `fzf`, `atuin`, `inshellisense`. Closest practical Q-like experience on Linux.
- **No telemetry, no network calls.** Everything runs locally.

## Try it

You'll need **Go 1.21+** and **zsh**.

```bash
git clone https://github.com/vinbh/vbas.git
cd vbas
go build -o ./bin/vbas ./cmd/vbas

export PATH="$PWD/bin:$PATH"
export VBAS_SPECS_DIR="$PWD/specs"

# In a zsh session:
source ./shell/zsh/vbas.zsh

# Try it:
git c<Tab>      # dropdown of 5 subcommands starting with "c"
git <Tab>       # full list — type letters to filter
git commit -<Tab>   # flag dropdown
```

Make it permanent by adding the `export` lines and the `source` line to your `~/.zshrc`.

## How it works

```
zsh widget on Tab ──exec──▶ vbas binary
                                │
                                ├─▶ load JSON spec for the first token (e.g. "git")
                                ├─▶ match remaining input against spec
                                └─▶ if 2+ matches: open /dev/tty, raw mode,
                                     draw ANSI dropdown, run key loop,
                                     return picked value to stdout
                                              │
                  zsh widget reads stdout ◀───┘
                  zsh widget replaces last token with picked value
```

Today: a fresh process per Tab keypress (~10ms cold start — fine for M2).
M3 introduces a long-running daemon over a Unix socket — sub-millisecond steady-state, which is what makes M4 (as-you-type) feasible.

## Roadmap

- [x] **M0** — repo scaffold (license, CLA, layout)
- [x] **M1** — zsh Tab completion via static JSON specs
- [x] **M2** — in-terminal ANSI dropdown UI + type-to-filter ← *you are here*
- [ ] **M3** — long-running daemon over Unix socket
- [ ] **M4** — as-you-type suggestions on every keystroke
- [ ] **M5** — broader spec coverage (transpile from [Fig autocomplete](https://github.com/withfig/autocomplete))
- [ ] **M6** — embedded `goja` JS engine for full Fig spec compatibility
- [ ] **M7+** — bash/fish adapters · history-based ranking · LLM fallback · packaging (deb/rpm/AUR/brew)

## How vbas compares

|  | vbas (today) | zsh-autosuggestions | Amazon Q / Fig | inshellisense |
|---|---|---|---|---|
| Linux native | ✅ | ✅ | ❌ macOS only | ✅ |
| Dropdown with descriptions | ✅ | ❌ inline ghost only | ✅ | ✅ |
| Type-to-filter inside dropdown | ✅ | n/a | ✅ | ✅ |
| Source-available | ✅ PolyForm-NC | ✅ MIT | ❌ proprietary | ✅ MIT |
| Standalone binary | ✅ Go | ✅ pure shell | ❌ | ❌ Node runtime |
| As-you-type | ❌ (M4) | ✅ | ✅ | ✅ |

## License

[**PolyForm Noncommercial 1.0.0**](./LICENSE) — free for personal, hobby, research, and educational use.

For **commercial use** (use within a for-profit organization in the course of business), [open an issue](https://github.com/vinbh/vbas/issues/new) or email the maintainer (`vinayakbhatt@mit.tc`).

Bundled CLI specs imported from [Fig autocomplete](https://github.com/withfig/autocomplete) (when added in M5) remain under their original MIT license; headers will be preserved.

## Contributing

PRs welcome. By submitting a contribution you agree to the [Contributor License Agreement](./CLA.md), which lets the project relicense your contribution as part of future commercial dual-licensing.

If you want to add a spec for your favorite CLI, the JSON format is a strict subset of [Fig's spec schema](https://fig.io/docs/reference/spec) — see `specs/git.json` for a template.

## Prior art and credits

vbas stands on the shoulders of:

- [`withfig/autocomplete`](https://github.com/withfig/autocomplete) — the Fig spec catalog (MIT)
- [`microsoft/inshellisense`](https://github.com/microsoft/inshellisense) — the Node-based Fig port for the terminal
- [`carapace-sh/carapace`](https://github.com/carapace-sh/carapace) — Go-based multi-shell completion engine
- [`zsh-users/zsh-autosuggestions`](https://github.com/zsh-users/zsh-autosuggestions) — the inline-ghost Linux mainstay
- `fzf`, `atuin`, `fish` — for the in-terminal UI patterns vbas borrows from
