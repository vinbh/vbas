# vbas

**Amazon Q-style autocomplete for the Linux terminal — free, source-available, no Node required.**

<p align="center">
  <img src="demo/git.gif" alt="vbas git subcommand completion" width="860"/>
</p>

Press **space** after any known command and a dropdown opens. Type to filter, arrows to navigate, Enter to pick. The next level cascades automatically — subcommands → flags, all without leaving the terminal.

> ⚠️ **Pre-release.** Works well for daily use on 63 built-in commands. Live completions (branch names, kubectl contexts) land in M6.

---

## Install

You need **Go 1.21+** and **zsh**.

```bash
git clone https://github.com/vinbh/vbas.git
cd vbas
./install.sh
```

Add one line to your `~/.zshrc`:

```bash
source ~/.config/vbas/vbas.zsh
```

Open a new shell and start typing. `./uninstall.sh` removes everything (hand-rolled specs are preserved).

---

## More demos

**Directory navigation — drill down level by level**

<p align="center">
  <img src="demo/cd.gif" alt="vbas cd directory drill-down" width="860"/>
</p>

Pick a folder and the dropdown immediately opens inside it. Navigate to nested paths without typing a single character of the path.

**File completion — files first, then flags**

<p align="center">
  <img src="demo/files.gif" alt="vbas vim file completion" width="860"/>
</p>

For `vim`, `cat`, `ls`, `cp`, `rm`, `scp` and most other commands — real filesystem entries appear at the top of the dropdown, flags below.

---

## What it covers

| Category | Commands |
|---|---|
| Dev tools | `git`, `docker`, `kubectl`, `helm`, `terraform`, `make`, `gh`, `podman` |
| Languages | `go`, `cargo`, `rustc`, `python`, `pip`, `node`, `npm`, `yarn`, `pnpm` |
| Cloud | `aws`, `gcloud` |
| Shell | `ls`, `find`, `grep`, `sed`, `xargs`, `tar`, `cat`, `less`, `head`, `tail`, `man`, `mv`, `cp`, `rm`, `chmod`, `chown`, `cd` |
| Network | `curl`, `wget`, `ssh`, `scp`, `rsync`, `ping`, `nc`, `ssh-keygen` |
| System | `systemctl`, `apt`, `brew`, `ps`, `kill`, `top`, `htop`, `df`, `du` |
| Databases | `psql`, `mysql`, `sqlite3` |
| Data | `jq` |
| Editors / muxers | `vim`, `vi`, `nvim`, `tmux`, `screen` |

63 commands imported from [Fig autocomplete](https://github.com/withfig/autocomplete) (MIT). To add more: edit [`tools/import-fig/commands.txt`](./tools/import-fig/commands.txt) and re-run the importer.

---

## Why?

[Amazon Q CLI](https://aws.amazon.com/q/developer/cli/)'s dropdown autocomplete is great — but it's **macOS-only**. On Linux you get `zsh-autosuggestions` (single-line ghost text) or zsh's built-in completion (functional but not beautiful).

**vbas** brings the same Q-style experience to Linux: inline dropdown, descriptions visible while you type, cascading through subcommand levels. Built in Go — single static binary, sub-millisecond latency, no runtime deps.

---

## What's not there yet

| | Lands in |
|---|---|
| Live completions — `git checkout <branch>`, `kubectl` contexts, `aws` regions | M6 (embedded JS engine) |
| More of Fig's ~3000-CLI catalog | M5 polish |
| bash and fish shells | M7 |
| LLM fallback for unknown commands | post-M7 |
| Packages (deb/rpm/AUR/Homebrew) | M7+ |

**No floating overlay** — Linux has no universal cursor-pixel API and Wayland blocks the trick. vbas renders inline (same as `fzf`, `atuin`). No telemetry, no network calls.

---

## How it works

```
zsh widget (space or Tab)
      │
      ▼
  vbas client ──unix socket──▶ vbas-daemon
      │                              │
      │                              ├─ load spec for first token
      │                              └─ match remaining input
      ◀──── []Suggestion JSON ───────┘
      ▼
  /dev/tty raw mode → ANSI dropdown → pick returned to zsh
```

The daemon lazy-spawns on first use (fork + Setsid, no systemd). If it can't start, vbas answers in-process so you're never left without completions.

---

## Hacking on it

```bash
go build -o ./bin/vbas ./cmd/vbas
export PATH="$PWD/bin:$PATH"
export VBAS_SPECS_DIR="$PWD/specs"
source ./shell/zsh/vbas.zsh
# After rebuilding: pkill -KILL -f 'vbas daemon'
```

```bash
go test ./...
go test -race ./...
```

To regenerate the demo GIFs (requires [vhs](https://github.com/charmbracelet/vhs), ffmpeg, ttyd):

```bash
vhs demo/git.tape
vhs demo/cd.tape
vhs demo/files.tape
```

---

## Roadmap

- [x] **M0** — scaffold
- [x] **M1** — zsh Tab completion via JSON specs
- [x] **M2** — ANSI dropdown + type-to-filter
- [x] **M3** — long-running daemon (lazy auto-spawn, in-process fallback)
- [x] **M4** — auto-open on space · cascading subcommand levels
- [x] **M5** — 63 commands imported from [Fig autocomplete](https://github.com/withfig/autocomplete)
- [x] **M5.5** — file / folder generators (`cd`, `vim`, `ls`, `cp`, …)
- [ ] **M6** — embedded JS engine for live completions (branches, contexts, regions)
- [ ] **M7+** — bash/fish · history ranking · LLM fallback · packaging

---

## Comparison

|  | vbas | zsh-autosuggestions | Amazon Q / Fig | inshellisense |
|---|---|---|---|---|
| Linux native | ✅ | ✅ | ❌ macOS only | ✅ |
| Dropdown with descriptions | ✅ | ❌ ghost text | ✅ | ✅ |
| File/folder completion | ✅ | ❌ | ✅ | ✅ |
| Type-to-filter in dropdown | ✅ | n/a | ✅ | ✅ |
| Standalone binary | ✅ Go | ✅ shell | ❌ | ❌ Node |
| Source-available | ✅ PolyForm-NC | ✅ MIT | ❌ | ✅ MIT |

---

## License

[**PolyForm Noncommercial 1.0.0**](./LICENSE) — free for personal, hobby, research, and educational use.

For **commercial use**, [open an issue](https://github.com/vinbh/vbas/issues/new) or email `vinayakbhatt@mit.tc`.

Specs from [Fig autocomplete](https://github.com/withfig/autocomplete) remain MIT-licensed; their headers are preserved in `specs/fig/`.

## Contributing

PRs welcome. By submitting you agree to the [CLA](./CLA.md), which allows future commercial dual-licensing.

To add a spec: drop a `specs/<cmd>.json` — it overrides anything in `specs/fig/`. The format is a subset of [Fig's spec schema](https://fig.io/docs/reference/spec).

## Credits

- [`withfig/autocomplete`](https://github.com/withfig/autocomplete) — spec catalog (MIT)
- [`charmbracelet/vhs`](https://github.com/charmbracelet/vhs) — demo GIF tooling
- [`microsoft/inshellisense`](https://github.com/microsoft/inshellisense) — Node-based Fig port
- [`carapace-sh/carapace`](https://github.com/carapace-sh/carapace) — Go completion engine
- [`zsh-users/zsh-autosuggestions`](https://github.com/zsh-users/zsh-autosuggestions) — the Linux ghost-text standard
- `fzf`, `atuin`, `fish` — UI patterns
