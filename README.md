# vbas

**Amazon Q-style autocomplete for the Linux terminal. Free, source-available, no Node required.**

```
$ git <space>
▍ checkout              Switch branches or restore working tree files
▍ cherry-pick           Apply changes from existing commits
▍ clean                 Remove untracked files from the working tree
▍ clone                 Clone a repository into a new directory
▍ commit                Record changes to the repository
─── vbas ─── ↑↓ select · ⏎ accept · esc cancel · 1/5
```

Type a known command, press space — a dropdown opens. Type to filter, arrows to move, Enter to accept. That's it.

> ⚠️ **Pre-release.** Works well for daily use on the 63 built-in commands. Dynamic completions (live branch names, kubectl contexts) land in M6.

---

## What it looks like

**Subcommand completion with cascading levels**

```
$ docker <space>
▍ build                 Build an image from a Dockerfile
▍ compose               Docker Compose
▍ container             Manage containers
▍ exec                  Execute a command in a running container
▍ images                List images
▍ pull                  Download an image from a registry
▍ push                  Upload an image to a registry
▍ run                   Create and run a new container from an image
─── vbas ─── ↑↓ select · ⏎ accept · esc cancel · 1/17
```

Pick `run` → flag dropdown opens automatically.

```
$ docker run <space>
▍ --detach              Run container in background and print container ID
▍ --env                 Set environment variables
▍ --interactive         Keep STDIN open even if not attached
▍ --name                Assign a name to the container
▍ --network             Connect a container to a network
▍ --publish             Publish a container's port(s) to the host
▍ --rm                  Automatically remove the container on exit
▍ --volume              Bind mount a volume
─── vbas ─── ↑↓ select · ⏎ accept · esc cancel · 1/30
```

**File and folder completion**

```
$ cd <space>
▍ Documents/
▍ Downloads/
▍ Pictures/
▍ workspace/
─── vbas ─── ↑↓ select · ⏎ accept · esc cancel · 1/4
```

Pick `workspace/` → automatically drills in:

```
$ cd workspace/
▍ vbas/
▍ myproject/
▍ scripts/
─── vbas ─── ↑↓ select · ⏎ accept · esc cancel · 1/3
```

Works for `vim`, `cat`, `ls`, `cp`, `mv`, `rm`, `scp`, and most other commands that take file paths.

**Type to filter — ranked by relevance**

```
$ git c
▍ checkout              Switch branches or restore working tree files
▍ cherry-pick           Apply changes from existing commits
▍ clean                 Remove untracked files from the working tree
▍ clone                 Clone a repository into a new directory
▍ commit                Record changes to the repository
─── vbas ─── ↑↓ select · ⏎ accept · esc cancel · 5/28
```

Name-prefix matches float to the top; description matches come after.

---

## Why?

[Amazon Q CLI](https://aws.amazon.com/q/developer/cli/)'s dropdown autocomplete is the best shell UX on a terminal — but it's **macOS-only**. [Fig](https://fig.io/), its predecessor, was the same. On Linux the alternatives are `zsh-autosuggestions` (single-line ghost text) or built-in zsh completion (functional, not pretty).

**vbas brings the same experience to Linux**: inline dropdown, descriptions visible while you type, cascading through subcommand levels. Built in Go — single static binary, sub-millisecond latency, no Node or Python runtime.

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

Open a new shell and start typing. `./uninstall.sh` removes everything; hand-rolled specs in `~/.config/vbas/specs/` are preserved.

---

## Built-in commands (63 today)

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

Specs are imported from [Fig autocomplete](https://github.com/withfig/autocomplete) (MIT). To add more: edit [`tools/import-fig/commands.txt`](./tools/import-fig/commands.txt) and re-run `npm run import`.

---

## What's not there yet

| | Lands in |
|---|---|
| Live completions — `git checkout <branch>`, `kubectl` contexts, `aws` regions | M6 (JS runtime) |
| More of Fig's ~3000-CLI catalog | M5 polish |
| bash and fish shells | M7 |
| LLM fallback for unknown commands | post-M7 |
| Packages (deb/rpm/AUR/Homebrew) | M7+ |

**No floating overlay** — Linux has no universal cursor-pixel API and Wayland blocks the trick. vbas renders inline, same as `fzf` and `atuin`. No telemetry, no network calls.

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

The daemon is lazy-spawned on first use (fork + Setsid, no systemd plumbing). If it can't start, vbas answers in-process so the user is never blocked.

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

To add a spec: the JSON format is a subset of [Fig's spec schema](https://fig.io/docs/reference/spec). Drop a file in `specs/<cmd>.json` — it takes precedence over anything in `specs/fig/`.

## Credits

- [`withfig/autocomplete`](https://github.com/withfig/autocomplete) — spec catalog (MIT)
- [`microsoft/inshellisense`](https://github.com/microsoft/inshellisense) — Node-based Fig port
- [`carapace-sh/carapace`](https://github.com/carapace-sh/carapace) — Go completion engine
- [`zsh-users/zsh-autosuggestions`](https://github.com/zsh-users/zsh-autosuggestions) — the Linux ghost-text standard
- `fzf`, `atuin`, `fish` — UI patterns
