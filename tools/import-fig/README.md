# tools/import-fig

Imports curated CLI specs from [`withfig/autocomplete`](https://github.com/withfig/autocomplete) into [`specs/fig/`](../../specs/fig/) at the repo root.

This is a **dev tool** — needed only when you want to regenerate or extend the imported spec catalog. End users never run it; vbas ships the resulting JSON files inside the Go binary's spec directory.

## Run

```bash
cd tools/import-fig
npm install               # one-time, installs tsx
npm run import            # uses cached /tmp/vbas-fig-autocomplete clone if present
npm run import -- --refresh   # `git pull` the fig clone before importing
```

First run takes ~1 minute (shallow-clones Fig + installs their deps into `/tmp/vbas-fig-autocomplete`). Subsequent runs reuse the clone and complete in seconds.

## Output

One JSON file per command at `specs/fig/<cmd>.json`. Each file has a small metadata header preserving upstream provenance:

```json
{
  "_license": "MIT",
  "_source": "https://github.com/withfig/autocomplete/blob/master/src/<cmd>.ts",
  "_generated": "tools/import-fig/import.ts",
  "name": "<cmd>",
  "subcommands": [...]
}
```

## What gets stripped

Fig specs are TypeScript and often contain runtime functions:

- **`generators`** — dynamic completion via shell scripts (e.g., live `git branch` names). Function fields are dropped during JSON serialization; the static parts of the generator object remain.
- **`postProcess` / `filterTerm` / etc.** — all function values are dropped.

Static fields (`name`, `description`, `subcommands`, `options`, positional `args`) survive intact and cover the bulk of practical completion. **Dynamic completion lands in M6** when vbas embeds a JS engine (`goja`).

## Editing the curated list

`commands.txt`, one per line, `#` for comments. Add or remove entries and re-run `npm run import`.

If a command exists in `commands.txt` but the script can't find a matching spec in the Fig repo, the run prints `SKIP <cmd> no spec in fig repo` and continues. No partial output is left behind.

## Hand-tuning a spec

Hand-rolled specs at `specs/<cmd>.json` take precedence over `specs/fig/<cmd>.json` — the loader checks the top-level dir first, then falls back to `fig/`. So you can override an imported spec by writing your own at `specs/<cmd>.json` without forking Fig.
