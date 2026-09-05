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

const EXCLUDED_RULES = ['aria-progressbar-name'];
const EXCLUDED_RULES_SET = new Set(EXCLUDED_RULES);

const currentPath = process.argv.includes('--current')
  ? process.argv[process.argv.indexOf('--current') + 1]
  : DEFAULT_CURRENT;
const frozenPath = process.argv.includes('--frozen')
  ? process.argv[process.argv.indexOf('--frozen') + 1]
  : DEFAULT_FROZEN;

const rawCurrent = JSON.parse(readFileSync(currentPath, 'utf8'));
const rawFrozen  = JSON.parse(readFileSync(frozenPath,  'utf8'));

// Normalize current: support both old { metadata: {...} } and new { totals, violations } formats
function normalizeCurrent(raw) {
  if (raw.metadata) {
    return raw.metadata;
  }
  // New aggregate format: { date, routes, totals, byTag, violations }
  // Compute byRule from violations array
  const byRule = {};
  for (const v of raw.violations ?? []) {
    if (!byRule[v.id]) {
      byRule[v.id] = { count: 0, impact: v.impact ?? 'serious', description: v.description ?? '' };
    }
    byRule[v.id].count++;
  }
  // Compute gated repViolations (excludes EXCLUDED_RULES like aria-progressbar-name)
  const gatedCount = Object.entries(byRule)
    .filter(([id]) => !EXCLUDED_RULES_SET.has(id))
    .reduce((sum, [, data]) => sum + data.count, 0);
  return {
    totalRoutes: raw.totals?.routes ?? raw.routes?.length ?? 0,
    paramRoutes: 0,
    nonRepClean: 0,
    nonRepWithViolations: 0,
    crashedRoutes: 0,
    repRoutesWithViolations: 0,
    repViolations: raw.totals?.violations ?? raw.violations?.length ?? 0,
    totalViolations: raw.totals?.violations ?? raw.violations?.length ?? 0,
    routesWithViolations: 0,
    byRule,
    routes: [],
    _gatedRepViolations: gatedCount,
  };
}

const c = normalizeCurrent(rawCurrent);
const f = rawFrozen.metadata;

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
// Gated repViolations excludes EXCLUDED_RULES (structural MUI spinner violations)
const gatedRepViolations = (routes) => routes
  .filter(r => !r.wasNonRep && !r.crashed)
  .reduce((sum, r) => sum + (
    Array.isArray(r.violationIds)
      ? r.violationIds.filter(id => !EXCLUDED_RULES_SET.has(id)).length
      : 0
  ), 0);

const cGated = (c.routes ?? []).length > 0 ? gatedRepViolations(c.routes) : (c._gatedRepViolations ?? c.repViolations);
const fGated = (f.routes ?? []).length > 0 ? gatedRepViolations(f.routes) : f.repViolations;
const repDeltaGated = cGated - fGated;
const repDelta = c.repViolations - f.repViolations;

if (repDeltaGated > 0) {
  failed = true;
  failures.push(`repViolations (gated) REGRESSION: ${fGated} → ${cGated} (+${repDeltaGated})`);
} else if (repDeltaGated < 0) {
  warnings.push(`repViolations (gated) IMPROVED: ${fGated} → ${cGated} (${repDeltaGated}). Verify and re-freeze.`);
}

// Total repViolations (all rules, informational only)
if (repDelta !== repDeltaGated) {
  warnings.push(`repViolations (total, incl. excluded): ${f.repViolations} → ${c.repViolations} (${repDelta >= 0 ? '+' : ''}${repDelta})`);
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
// Excluded rules (aria-progressbar-name): tracked separately, not gated
for (const rule of [...allRules].sort()) {
  if (EXCLUDED_RULES_SET.has(rule)) {
    const fc = c.byRule?.[rule]?.count ?? 0;
    const ff = f.byRule?.[rule]?.count ?? 0;
    const delta = fc - ff;
    if (delta !== 0) {
      warnings.push(`[tracked] rule ${rule}: ${ff} → ${fc} (${delta >= 0 ? '+' : ''}${delta}) — excluded from gate (MUI spinner structural; fix: A11yCircularProgress wrapper)`);
    }
    continue;
  }
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
console.log(`  repViolations (gated):  frozen=${fGated} current=${cGated} Δ=${repDeltaGated >= 0 ? '+' : ''}${repDeltaGated}  [excludes: ${EXCLUDED_RULES.join(', ')}]`);
console.log(`  repViolations (total):  frozen=${f.repViolations} current=${c.repViolations} Δ=${repDelta >= 0 ? '+' : ''}${repDelta}  [informational — includes excluded rules]`);
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
