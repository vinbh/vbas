/**
 * Imports curated specs from withfig/autocomplete into ../../specs/fig/.
 *
 * For each command listed in commands.txt:
 *   1. Resolve its TypeScript spec inside a shallow clone of the Fig repo
 *   2. Dynamically import it (tsx handles the .ts at runtime)
 *   3. JSON-stringify the default export (function values stripped)
 *   4. Prepend MIT-license metadata keys and write specs/fig/<cmd>.json
 *
 * Dynamic generators (functions inside Fig specs) are dropped during step 3.
 * vbas doesn't execute JS yet — that's M6 (goja). Static name/description
 * /subcommand/option fields cover ~90% of practical completions.
 *
 * Run with:
 *   cd tools/import-fig
 *   npm install
 *   npm run import           # uses cached fig clone if present
 *   npm run import -- --refresh   # git pulls the fig clone first
 */

import { execSync } from 'node:child_process';
import {
  existsSync,
  mkdirSync,
  readFileSync,
  writeFileSync,
} from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const FIG_REPO = 'https://github.com/withfig/autocomplete.git';
const FIG_CLONE = '/tmp/vbas-fig-autocomplete';

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(SCRIPT_DIR, '..', '..');
const OUT_DIR = path.join(REPO_ROOT, 'specs', 'fig');
const COMMANDS_FILE = path.join(SCRIPT_DIR, 'commands.txt');

interface Result {
  cmd: string;
  bytes?: number;
  reason?: string;
}

async function main() {
  // 1. Clone or refresh the Fig repo.
  if (!existsSync(FIG_CLONE)) {
    log(`cloning ${FIG_REPO} → ${FIG_CLONE}`);
    execSync(`git clone --depth 1 ${FIG_REPO} ${FIG_CLONE}`, { stdio: 'inherit' });
  } else if (process.argv.includes('--refresh')) {
    log(`refreshing ${FIG_CLONE}`);
    execSync(`git pull --depth 1 --rebase`, { cwd: FIG_CLONE, stdio: 'inherit' });
  }

  // 2. Make sure Fig's runtime deps are installed (some specs import shared
  //    helpers via @withfig/autocomplete-shared etc.).
  if (!existsSync(path.join(FIG_CLONE, 'node_modules'))) {
    log(`installing fig deps (one-time, ~30s)`);
    execSync(`npm install --no-audit --no-fund --no-progress`, {
      cwd: FIG_CLONE,
      stdio: 'inherit',
    });
  }

  // 3. Read curated list.
  const commands = readFileSync(COMMANDS_FILE, 'utf8')
    .split('\n')
    .map((s) => s.trim())
    .filter((s) => s && !s.startsWith('#'));

  // 4. Ensure output dir.
  mkdirSync(OUT_DIR, { recursive: true });

  // 5. Import each.
  const results: Result[] = [];
  for (const cmd of commands) {
    const specPath = findSpecPath(cmd);
    if (!specPath) {
      results.push({ cmd, reason: 'no spec in fig repo' });
      continue;
    }
    try {
      const mod = await import(specPath);
      const spec = mod.default ?? mod.completionSpec;
      if (!spec) throw new Error('no default export or `completionSpec`');

      const cleaned = JSON.stringify(
        spec,
        (_k, v) => (typeof v === 'function' ? undefined : v),
        2,
      );
      const wrapped = wrapWithMetadata(cmd, cleaned);
      const outPath = path.join(OUT_DIR, `${cmd}.json`);
      writeFileSync(outPath, wrapped);
      results.push({ cmd, bytes: Buffer.byteLength(wrapped) });
    } catch (e: any) {
      results.push({ cmd, reason: e?.message ?? String(e) });
    }
  }

  // 6. Report.
  const ok = results.filter((r) => r.bytes !== undefined);
  const failed = results.filter((r) => r.reason !== undefined);
  log('');
  log(`imported ${ok.length} / ${commands.length}`);
  for (const r of ok) {
    log(`  OK   ${r.cmd.padEnd(14)} ${(r.bytes! / 1024).toFixed(1).padStart(6)} KB`);
  }
  if (failed.length) {
    log('');
    log(`skipped (${failed.length}):`);
    for (const r of failed) {
      log(`  SKIP ${r.cmd.padEnd(14)} ${r.reason}`);
    }
  }
}

function findSpecPath(cmd: string): string | null {
  const candidates = [
    path.join(FIG_CLONE, 'src', `${cmd}.ts`),
    path.join(FIG_CLONE, 'src', cmd, 'index.ts'),
    path.join(FIG_CLONE, 'src', cmd, `_${cmd}.ts`),
  ];
  for (const c of candidates) {
    if (existsSync(c)) return c;
  }
  return null;
}

// Add MIT-provenance keys at the top of the JSON. Go's decoder ignores
// unknown fields, so these don't break vbas's parser. Keeping them in-file
// (rather than a separate LICENSES/ doc) means the provenance never gets
// separated from the data when someone copies a single spec around.
function wrapWithMetadata(cmd: string, json: string): string {
  const obj = JSON.parse(json);
  const wrapped = {
    _license: 'MIT',
    _source: `https://github.com/withfig/autocomplete/blob/master/src/${cmd}.ts`,
    _generated: 'tools/import-fig/import.ts',
    ...obj,
  };
  return JSON.stringify(wrapped, null, 2) + '\n';
}

function log(msg: string) {
  console.log(msg);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
