#!/usr/bin/env node
/**
 * Build-time tsc ratchet. Fails CI if the current tsc output introduces
 * errors not present in docs/ts-baseline.txt (the pre-batch snapshot).
 *
 * Run before vite build so type errors block the bundle.
 *
 * tsc emits errors as multi-line blocks. Each block starts with a line
 * matching /<file>.ts:<col>? error TS<code>/ and continues with indented
 * continuation lines until the next error header. We compare block-sets,
 * not line-sets, so multi-line errors are compared atomically.
 *
 * Usage:
 *   node scripts/ts-ratchet.mjs              # gate; non-zero exit on new errors
 *   node scripts/ts-ratchet.mjs --regenerate # rewrite the baseline from current tsc
 */
import { execSync } from 'node:child_process';
import { readFileSync, existsSync, writeFileSync } from 'node:fs';
import path from 'node:path';

const BASELINE_PATH = path.resolve('docs/ts-baseline.txt');

function stripProjectRoot(line) {
  const projectRoot = process.cwd();
  let out = line;
  if (projectRoot) out = out.split(projectRoot).join('');
  // Strip leading slashes inside any quoted string — tsc's Namespace/import
  // output uses paths like "/src/..." or "/Users/.../src/..." with a leading
  // slash that breaks the diff. Collapse to project-relative.
  out = out.replace(/"\/+/g, '"');
  return out;
}

function normalizeLine(line) {
  return stripProjectRoot(line).replace(/\(\d+,\d+\)/, '').trimEnd();
}

function parseErrorBlocks(raw) {
  const HEADER_RE = /^\S.*: error TS\d+:/;
  const lines = raw.split('\n');
  const blocks = [];
  let current = [];
  for (const line of lines) {
    if (HEADER_RE.test(line)) {
      if (current.length) blocks.push(current);
      current = [line];
    } else if (line.trim() === '') {
      if (current.length) blocks.push(current);
      current = [];
    } else if (current.length) {
      current.push(line);
    }
  }
  if (current.length) blocks.push(current);
  return blocks;
}

function blockToKey(block) {
  return block.map(normalizeLine).join('\n');
}

function runTsc() {
  try {
    return execSync('npx tsc --noEmit --pretty false', {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
      maxBuffer: 64 * 1024 * 1024,
    });
  } catch (err) {
    return (err.stdout || '') + (err.stderr || '');
  }
}

if (process.argv.includes('--regenerate')) {
  const raw = runTsc();
  const blocks = parseErrorBlocks(raw);
  const keys = blocks.map(blockToKey);
  // Use a delimiter that can't appear inside an error block. JSON-encode each
  // block so multi-line content is preserved exactly when we split back out.
  const out = keys.map((k) => JSON.stringify(k)).sort().join('\n') + '\n';
  writeFileSync(BASELINE_PATH, out);
  console.log(`Regenerated ${BASELINE_PATH}: ${blocks.length} unique error blocks`);
  process.exit(0);
}

if (!existsSync(BASELINE_PATH)) {
  console.error(`Missing baseline at ${BASELINE_PATH}. Regenerating now...`);
  const raw = runTsc();
  const blocks = parseErrorBlocks(raw);
  const keys = blocks.map(blockToKey);
  const out = keys.map((k) => JSON.stringify(k)).sort().join('\n') + '\n';
  writeFileSync(BASELINE_PATH, out);
  console.error(`Wrote ${blocks.length} error blocks. Re-run to gate.`);
  process.exit(2);
}

// Read baseline: each line is a JSON-encoded block.
const baselineText = readFileSync(BASELINE_PATH, 'utf8');
const baseline = new Set(
  baselineText
    .split('\n')
    .filter(Boolean)
    .map((line) => JSON.parse(line)),
);

const currentRaw = runTsc();
const currentBlocks = parseErrorBlocks(currentRaw);
const currentKeys = currentBlocks.map(blockToKey);
const current = new Set(currentKeys);

const fresh = currentKeys.filter((k) => !baseline.has(k));
const removed = [...baseline].filter((k) => !current.has(k));

if (fresh.length === 0) {
  console.log(
    `ratchet OK: ${currentKeys.length} error blocks, 0 new (baseline ${baseline.size}, ${removed.length} resolved since)`,
  );
  process.exit(0);
}

console.error(`\n=== RATCHET FAILED: ${fresh.length} new tsc error blocks ===\n`);
for (const key of fresh) {
  console.error('---');
  console.error(key);
}
console.error(`\nTo accept these as known, regenerate docs/ts-baseline.txt after review:`);
console.error(`  node scripts/ts-ratchet.mjs --regenerate`);
process.exit(1);
