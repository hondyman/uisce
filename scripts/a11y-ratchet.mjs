#!/usr/bin/env node
/**
 * a11y-ratchet.mjs
 *
 * Compares the current aggregate against a frozen baseline JSON.
 * Fails if any rule increases OR denominator shrinks — the only sanctioned
 * escape hatch is a commit that rewrites the frozen baseline with a
 * commit message containing FREEZE-REASON: and the per-rule delta.
 *
 * Usage: node scripts/a11y-ratchet.mjs [--current <path>] [--frozen <path>]
 */

import { readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));

const DEFAULT_CURRENT = join(__dirname, '..', 'frontend', 'docs', 'a11y', `baseline-${new Date().toISOString().slice(0,10)}.json`);
const DEFAULT_FROZEN  = join(__dirname, '..', 'frontend', 'docs', 'a11y', 'baseline-frozen.json');

const currentPath = process.argv.includes('--current')
  ? process.argv[process.argv.indexOf('--current') + 1]
  : DEFAULT_CURRENT;
const frozenPath = process.argv.includes('--frozen')
  ? process.argv[process.argv.indexOf('--frozen') + 1]
  : DEFAULT_FROZEN;

const current = JSON.parse(readFileSync(currentPath, 'utf8'));
const frozen  = JSON.parse(readFileSync(frozenPath,  'utf8'));

const c = current.metadata;
const f = frozen.metadata;

const allRules = new Set([
  ...Object.keys(c.byRule || {}),
  ...Object.keys(f.byRule || {}),
]);

const repDenomCurrent = c.totalRoutes - c.nonRepClean - c.nonRepWithViolations - c.crashedRoutes;
const repDenomFrozen  = f.totalRoutes - f.nonRepClean - f.nonRepWithViolations - f.crashedRoutes;

let failed = false;
const failures = [];

if (c.repViolations !== f.repViolations) {
  failed = true;
  failures.push(`repViolations: ${f.repViolations} → ${c.repViolations} (Δ${c.repViolations - f.repViolations >= 0 ? '+' : ''}${c.repViolations - f.repViolations})`);
}

if (repDenomCurrent < repDenomFrozen) {
  failed = true;
  failures.push(`repDenominator SHRANK: ${repDenomFrozen} → ${repDenomCurrent} (Δ${repDenomCurrent - repDenomFrozen}) — must be documented in FREEZE-REASON`);
}

for (const rule of allRules) {
  const fc = c.byRule?.[rule]?.count ?? 0;
  const ff = f.byRule?.[rule]?.count ?? 0;
  if (fc > ff) {
    failed = true;
    failures.push(`rule ${rule}: ${ff} → ${fc} (Δ+${fc - ff}) — must be documented in FREEZE-REASON`);
  }
}

if (failed) {
  console.error('❌ RATCHET FAILED');
  console.error(`  Current:  ${currentPath}`);
  console.error(`  Frozen:   ${frozenPath}`);
  for (const msg of failures) {
    console.error(`  • ${msg}`);
  }
  console.error('');
  console.error('To re-freeze with a documented reason, add to the commit message:');
  console.error(`  FREEZE-REASON: ${failures.join(' | ')}`);
  process.exit(1);
} else {
  console.log('✅ RATCHET PASSED — no regressions detected');
  console.log(`  Frozen repViolations:  ${f.repViolations}`);
  console.log(`  Current repViolations: ${c.repViolations}`);
  console.log(`  Frozen repDenominator: ${repDenomFrozen}`);
  console.log(`  Current repDenominator:${repDenomCurrent}`);
  console.log(`  Per-rule deltas:`);
  for (const rule of [...allRules].sort()) {
    const fc = c.byRule?.[rule]?.count ?? 0;
    const ff = f.byRule?.[rule]?.count ?? 0;
    const delta = fc - ff;
    if (delta !== 0) {
      console.log(`    ${rule}: ${ff} → ${fc} (${delta >= 0 ? '+' : ''}${delta})`);
    }
  }
  process.exit(0);
}
