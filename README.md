# vbas

**vb-autosuggest** — a Linux terminal autosuggest tool inspired by Amazon Q CLI's dropdown completions. Source-available, free for personal use.

> **Status:** very early. M1 ships Tab-driven completion for a small set of `git` subcommands in zsh. Most of the roadmap is not yet built.

## Roadmap

- [x] **M0** — repo scaffold
- [ ] **M1** — zsh Tab completion via static specs (in progress)
- [ ] **M2** — in-terminal dropdown UI (ANSI-rendered)
- [ ] **M3** — long-running daemon over Unix socket for sub-10ms latency
- [ ] **M4** — as-you-type suggestions on every keystroke
- [ ] **M5** — broader spec coverage (transpile from [Fig autocomplete](https://github.com/withfig/autocomplete))
- [ ] **M6** — embedded JS engine (`goja`) for full Fig spec compatibility
- [ ] **M7+** — bash adapter, history-based ranking, LLM fallback, packaging (deb/rpm/AUR/brew)

## Try it (M1)

You'll need Go 1.21+ and zsh.

```bash
go build -o ./bin/vbas ./cmd/vbas
export PATH="$PWD/bin:$PATH"
export VBAS_SPECS_DIR="$PWD/specs"
source ./shell/zsh/vbas.zsh
```

Now type `git ch` and press <kbd>Tab</kbd> — it should complete to `git checkout`. Tab on commands without a spec falls through to your normal zsh completion.

To uninstall, just don't source the script again. The widget binds Tab only inside that shell session.

## How it works

```
zsh widget (Tab) ──exec──▶ vbas binary ──read──▶ specs/*.json
                                │
                                ▼
                          first matching suggestion
                                │
                                ▼
                          replace BUFFER, redraw
```

Every Tab spawns a fresh `vbas` process. That's fine for M1 (~10ms cold start), but won't scale to as-you-type — M3 introduces a long-running daemon.

## Architecture (target)

```
shell hook  ──(unix socket)──▶  vbas-daemon  ──▶  spec engine
                                      │              ├─ specs/*.json (M1+)
                                      │              └─ goja JS runtime (M6+)
                                      ▼
                                ANSI dropdown renderer (M2+)
```

## License

[PolyForm Noncommercial 1.0.0](./LICENSE) — free for personal, research, hobby, educational, and noncommercial use.

For commercial use (use by a for-profit organization in the course of business), contact the maintainer.

Bundled CLI specs imported from [Fig autocomplete](https://github.com/withfig/autocomplete) (when added in M5) remain under their original MIT license; their headers will be preserved.

## Contributing

PRs welcome. By submitting a contribution you agree to the [Contributor License Agreement](./CLA.md), which lets the project relicense your contribution as part of future commercial dual-licensing.

## Prior art and credits

- [`microsoft/inshellisense`](https://github.com/microsoft/inshellisense) — Node-based Fig port
- [`withfig/autocomplete`](https://github.com/withfig/autocomplete) — Fig spec catalog (MIT)
- [`carapace-sh/carapace`](https://github.com/carapace-sh/carapace) — Go completion engine
- [`zsh-users/zsh-autosuggestions`](https://github.com/zsh-users/zsh-autosuggestions) — inspiration for shell-side integration
