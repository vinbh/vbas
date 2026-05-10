<div align="center">

# vbas

**Amazon Q-style dropdown autocomplete for Linux. Works in zsh and bash. Free, source-available, no Node required.**

[![Go](https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macOS-lightgrey?style=flat-square)](https://github.com/vinbh/vbas/releases)
[![Commands](https://img.shields.io/badge/commands-121-58A6FF?style=flat-square)](./specs)
[![License](https://img.shields.io/badge/license-PolyForm--NC-D2A8FF?style=flat-square)](./LICENSE)
[![Release](https://img.shields.io/github/v/release/vinbh/vbas?style=flat-square&color=3FB950)](https://github.com/vinbh/vbas/releases)

<br/>

<img src="demo/git.gif" alt="vbas git subcommand completion" width="860"/>

<br/>

*Press **space** after any known command. A dropdown opens. Type to filter. Arrows to navigate. Enter to pick. Cascades automatically through subcommand levels.*

</div>

---

## Why?

[Amazon Q CLI](https://aws.amazon.com/q/developer/cli/) ships the best terminal autocomplete available - an inline dropdown with descriptions, cascading through subcommand levels. It is **macOS-only**.

On Linux you get `zsh-autosuggestions` (single-line ghost text) or your shell's built-in TAB completion (functional but not visual). **vbas** fills the gap: the same dropdown experience, in zsh and bash, as a single static Go binary. No Node, no Electron, no runtime deps. Sub-millisecond latency via a persistent background daemon.

---

## Features

**Live completions from your actual environment**

`git checkout <Tab>` lists your real branches. `git add <Tab>` shows your modified files. `kubectl exec <Tab>` lists running pods. `docker run <Tab>` shows local images. Completions run the same shell commands Fig's specs define, in your cwd, at completion time.

**Directory drill-down**

<p align="center">
  <img src="demo/cd.gif" alt="vbas cd directory drill-down" width="860"/>
</p>

Pick a folder and the dropdown immediately opens inside it. Navigate deep paths without typing a single slash.

**Files before flags**

<p align="center">
  <img src="demo/files.gif" alt="vbas file completion" width="860"/>
</p>

For `vim`, `cat`, `ls`, `cp`, `rm`, `scp` and most file commands - real filesystem entries appear at the top of the dropdown, flags below.

---

## Install

### One-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/vinbh/vbas/main/get.sh | bash
```

Detects your OS and architecture, downloads the right binary from [Releases](https://github.com/vinbh/vbas/releases/latest), and installs everything. No Go required.

### Via `go install` (requires Go 1.21+)

```bash
go install github.com/vinbh/vbas/cmd/vbas@latest
vbas setup
```

`vbas setup` extracts the embedded specs and hooks to `~/.config/vbas/` and offers to wire your shell rc files automatically.

### From source

```bash
git clone https://github.com/vinbh/vbas.git
cd vbas
./install.sh
```

### Enable in your shell

The installer prompts you automatically. To add manually:

**zsh** — add to `~/.zshrc`:
```bash
source ~/.config/vbas/vbas.zsh
```

**bash** (4.3+) — add to `~/.bashrc`:
```bash
source ~/.config/vbas/vbas.bash
```

Open a new shell and start typing. `./uninstall.sh` removes everything (hand-rolled specs are preserved).

---

## What it covers

| Category | Commands |
|---|---|
| Dev tools | `git`, `docker`, `docker-compose`, `kubectl`, `helm`, `terraform`, `make`, `gh`, `podman` |
| Kubernetes | `kind`, `minikube`, `k3d`, `kubectx`, `kubens` |
| IaC / infra | `pulumi`, `ansible`, `ansible-playbook`, `vault` |
| Cloud | `aws`, `gcloud`, `doctl`, `fly`, `vercel`, `netlify` |
| Languages | `go`, `cargo`, `rustc`, `python`, `pip`, `node`, `npm`, `yarn`, `pnpm`, `ruby`, `gem`, `bundle`, `swift`, `dotnet`, `java` |
| Version managers | `nvm`, `rbenv`, `pyenv`, `asdf` |
| Python packaging | `poetry`, `pipenv`, `conda` |
| Build tools | `cmake`, `gradle`, `mvn`, `ant`, `bazel` |
| Shell utils | `ls`, `find`, `grep`, `sed`, `xargs`, `tar`, `cut`, `sort`, `uniq`, `wc`, `tee`, `diff`, `mv`, `cp`, `rm`, `ln`, `mkdir`, `touch`, `chmod`, `chown`, `cd` |
| Viewing | `cat`, `less`, `head`, `tail`, `man`, `bat` |
| Network | `curl`, `wget`, `ssh`, `scp`, `rsync`, `ping`, `nc`, `ssh-keygen`, `nmap`, `dig`, `traceroute` |
| System | `systemctl`, `apt`, `brew`, `ps`, `kill`, `top`, `htop`, `df`, `du`, `lsof`, `uname`, `crontab` |
| Databases | `psql`, `mysql`, `sqlite3`, `mongosh` |
| Data | `jq`, `svn` |
| Editors / muxers | `vim`, `vi`, `nvim`, `emacs`, `nano`, `tmux`, `screen` |
| Dev workflow | `direnv`, `pre-commit`, `goreleaser`, `act`, `gpg` |

121 commands imported from [Fig autocomplete](https://github.com/withfig/autocomplete) (MIT). To add more: edit [`tools/import-fig/commands.txt`](./tools/import-fig/commands.txt) and re-run the importer.

---

## Comparison

|  | vbas | zsh-autosuggestions | Amazon Q / Fig | inshellisense |
|---|---|---|---|---|
| Linux native | ✅ | ✅ | ❌ macOS only | ✅ |
| Dropdown with descriptions | ✅ | ❌ ghost text | ✅ | ✅ |
| Live completions (branches, pods) | ✅ | ❌ | ✅ | ✅ |
| File/folder completion | ✅ | ❌ | ✅ | ✅ |
| Type-to-filter in dropdown | ✅ | n/a | ✅ | ✅ |
| Standalone binary | ✅ Go | ✅ shell | ❌ | ❌ Node |
| Source-available | ✅ PolyForm-NC | ✅ MIT | ❌ | ✅ MIT |

**No floating overlay.** Linux has no universal cursor-pixel API and Wayland blocks the trick. vbas renders inline (same as `fzf`, `atuin`). No telemetry, no network calls.

---

## What's not there yet

| | Lands in |
|---|---|
| fish shell | M9+ |
| LLM fallback for unknown commands | M9+ |
| Packages (deb/rpm/AUR/Homebrew) | M9+ |

---

## How it works

```
zsh / bash widget (space or Tab)
      |
      v
  vbas client --unix socket--> vbas-daemon
      |                              |
      |                              +- load spec for first token
      |                              +- run generators (git branch, kubectl get, ...)
      |                              +- prefix-match remaining input
      <---- []Suggestion JSON -------+
      v
  /dev/tty raw mode -> ANSI dropdown -> pick returned to shell
```

The daemon lazy-spawns on first use (fork + Setsid, no systemd required). If it can't start, vbas answers in-process so you're never left without completions. Generator scripts run with a 5-second timeout in your shell's cwd.

---

## Hacking on it

```bash
go build -o ./bin/vbas ./cmd/vbas
export PATH="$PWD/bin:$PATH"
export VBAS_SPECS_DIR="$PWD/specs"
source ./shell/zsh/vbas.zsh   # or: source ./shell/bash/vbas.bash
# After rebuilding: pkill -KILL -f 'vbas daemon'
```

To test the `go install` path locally:

```bash
go build -o ./bin/vbas ./cmd/vbas
./bin/vbas setup
```

```bash
go test ./...
```

To regenerate the demo GIFs (requires [vhs](https://github.com/charmbracelet/vhs), ffmpeg, ttyd):

```bash
vhs demo/git.tape
vhs demo/cd.tape
vhs demo/files.tape
```

---

## Roadmap

- [x] **M0** - scaffold
- [x] **M1** - zsh Tab completion via JSON specs
- [x] **M2** - ANSI dropdown + type-to-filter
- [x] **M3** - long-running daemon (lazy auto-spawn, in-process fallback)
- [x] **M4** - auto-open on space, cascading subcommand levels
- [x] **M5** - 63 commands imported from [Fig autocomplete](https://github.com/withfig/autocomplete)
- [x] **M5.5** - file / folder generators (`cd`, `vim`, `ls`, `cp`, ...)
- [x] **M6** - live completions via script generators (git branches, kubectl resources, docker containers, aws profiles)
- [x] **M7** - bash shell support (4.3+), unified multi-shell installer
- [x] **M8** - 121-command catalog (2x), `go install` + embedded specs, `vbas setup`
- [ ] **M9+** - fish shell, history ranking, LLM fallback, packaging

---

## Contributing

PRs welcome. By submitting you agree to the [CLA](./CLA.md), which allows future commercial dual-licensing.

To add a spec: drop a `specs/<cmd>.json` - it overrides anything in `specs/fig/`. The format is a subset of [Fig's spec schema](https://fig.io/docs/reference/spec).

## License

[**PolyForm Noncommercial 1.0.0**](./LICENSE) - free for personal, hobby, research, and educational use.

For **commercial use**, [open an issue](https://github.com/vinbh/vbas/issues/new) or email `vinayakbhatt@mit.tc`.

Specs from [Fig autocomplete](https://github.com/withfig/autocomplete) remain MIT-licensed; their headers are preserved in `specs/fig/`.

## Credits

- [`withfig/autocomplete`](https://github.com/withfig/autocomplete) - spec catalog (MIT)
- [`charmbracelet/vhs`](https://github.com/charmbracelet/vhs) - demo GIF tooling
- [`microsoft/inshellisense`](https://github.com/microsoft/inshellisense) - Node-based Fig port
- [`carapace-sh/carapace`](https://github.com/carapace-sh/carapace) - Go completion engine
- [`zsh-users/zsh-autosuggestions`](https://github.com/zsh-users/zsh-autosuggestions) - the Linux ghost-text standard
- `fzf`, `atuin`, `fish` - UI patterns
