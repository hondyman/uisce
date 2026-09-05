#!/usr/bin/env node
/**
 * a11y-ratchet.mjs
 *
 * Compares the current aggregate against a frozen baseline JSON.
 *
 * Gate semantics:
 * - FAIL on any rule increase (regression — blocks)
 * - WARN on any rule decrease (improvement — prompts deliberate re-freeze)
 * - FAIL on denominator shrink (fewer routes measured — re-freeze required)
 * - WARN on denominator growth (more routes measured — re-freeze recommended)
 *
 * Shrinks are never silent: a floor that drops means the previous floor's
 * denominator was wrong. The escape hatch is a commit that rewrites
 * baseline-frozen.json with FREEZE-REASON: in the commit message.
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
const warnings = [];

// repViolations: fail on increase, warn on decrease
const repDelta = c.repViolations - f.repViolations;
if (repDelta > 0) {
  failed = true;
  failures.push(`repViolations REGRESSION: ${f.repViolations} → ${c.repViolations} (+${repDelta})`);
} else if (repDelta < 0) {
  warnings.push(`repViolations IMPROVED: ${f.repViolations} → ${c.repViolations} (${repDelta}). Verify and re-freeze.`);
}

// Denominator: fail on shrink, warn on growth
const denomDelta = repDenomCurrent - repDenomFrozen;
if (denomDelta < 0) {
  failed = true;
  failures.push(`repDenominator SHRANK: ${repDenomFrozen} → ${repDenomCurrent} (Δ${denomDelta}). Must re-freeze.`);
} else if (denomDelta > 0) {
  warnings.push(`repDenominator GREW: ${repDenomFrozen} → ${repDenomCurrent} (+${denomDelta}). Re-freeze recommended.`);
}

// Per-rule: fail on increase, warn on decrease
for (const rule of [...allRules].sort()) {
  const fc = c.byRule?.[rule]?.count ?? 0;
  const ff = f.byRule?.[rule]?.count ?? 0;
  const delta = fc - ff;
  if (delta > 0) {
    failed = true;
    failures.push(`rule ${rule} REGRESSION: ${ff} → ${fc} (+${delta})`);
  } else if (delta < 0) {
    warnings.push(`rule ${rule} IMPROVED: ${ff} → ${fc} (${delta}). Verify and re-freeze.`);
  }
}

console.log(`=== a11y-ratchet ===`);
console.log(`  Current: ${currentPath}`);
console.log(`  Frozen:  ${frozenPath}`);
console.log(`  repViolations:  frozen=${f.repViolations} current=${c.repViolations} Δ=${repDelta >= 0 ? '+' : ''}${repDelta}`);
console.log(`  repDenominator: frozen=${repDenomFrozen} current=${repDenomCurrent} Δ=${denomDelta >= 0 ? '+' : ''}${denomDelta}`);

if (warnings.length > 0) {
  console.log(`  Warnings (verify and re-freeze if expected):`);
  for (const w of warnings) console.log(`    ⚠ ${w}`);
}

if (failed) {
  console.error(`  ❌ FAIL — regressions detected:`);
  for (const msg of failures) console.error(`    • ${msg}`);
  console.error('');
  console.error('  To re-freeze: commit rewrites baseline-frozen.json with:');
  console.error(`    FREEZE-REASON: ${failures.map(f => f.split(' ')[0]).join(' | ')}`);
  process.exit(1);
} else if (warnings.length > 0) {
  console.log(`  ✅ PASS — no regressions`);
  console.log(`     (${warnings.length} improvement(s) detected — re-freeze to update the floor)`);
  process.exit(0);
} else {
  console.log(`  ✅ PASS — current matches frozen floor`);
  process.exit(0);
}
