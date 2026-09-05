#!/usr/bin/env node
/**
 * Aggregate per-route axe results from test-results/a11y/*.json into a single
 * baseline file at docs/a11y/baseline-{date}.json. Grouped by WCAG SC so
 * Phase 2 can prioritize by impact.
 */
import { readdirSync, readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import path from 'node:path';

const SRC = 'test-results/a11y';
const DST = 'docs/a11y';
const date = new Date().toISOString().slice(0, 10);

if (!existsSync(SRC)) {
  console.error(`No axe results found at ${SRC}. Run the baseline spec first.`);
  process.exit(1);
}

const files = readdirSync(SRC).filter((f) => f.endsWith('.json'));
const violations = [];
const passes = [];

for (const file of files) {
  const raw = JSON.parse(readFileSync(path.join(SRC, file), 'utf8'));
  const route = file.replace(/^_+|_+\.json$/g, '');
  for (const v of raw.violations ?? []) {
    violations.push({
      route,
      id: v.id,
      impact: v.impact,
      description: v.description,
      help: v.help,
      helpUrl: v.helpUrl,
      tags: v.tags,
      nodes: v.nodes.length,
      sampleTargets: v.nodes.slice(0, 3).map((n) => n.target),
    });
  }
  for (const p of raw.passes ?? []) {
    passes.push({ route, id: p.id, impact: p.impact });
  }
}

// Group by WCAG SC tag (wcag2a, wcag21aa, etc.)
const byTag = {};
for (const v of violations) {
  for (const tag of v.tags ?? []) {
    if (tag.startsWith('wcag')) {
      (byTag[tag] ??= []).push({ route: v.route, id: v.id, impact: v.impact, nodes: v.nodes });
    }
  }
}

// Tripwire: if any single rule appears on every route, the crawl likely hit the
// same page (e.g. Vite error overlay) and the data is invalid.
const byRule = {};
for (const v of violations) {
  byRule[v.id] = (byRule[v.id] || 0) + 1;
}
const routeCount = files.length;
for (const [rule, count] of Object.entries(byRule)) {
  if (count === routeCount) {
    console.warn(`\n⚠️  TRIPWIRE: rule '${rule}' appears on ALL ${routeCount} routes — this indicates`);
    console.warn(`   the crawl may have measured the same page everywhere (e.g. unmount/unauth redirect).`);
    console.warn(`   Review test-results/a11y/ samples before trusting this baseline.\n`);
  }
}

const summary = {
  date,
  generatedBy: 'frontend/scripts/aggregate-a11y-baseline.mjs',
  routes: files.map((f) => f.replace(/^_+|_+\.json$/g, '')),
  totals: {
    violations: violations.length,
    passes: passes.length,
    impact: {
      critical: violations.filter((v) => v.impact === 'critical').length,
      serious: violations.filter((v) => v.impact === 'serious').length,
      moderate: violations.filter((v) => v.impact === 'moderate').length,
      minor: violations.filter((v) => v.impact === 'minor').length,
    },
    routes: files.length,
  },
  byTag,
  violations,
};

mkdirSync(DST, { recursive: true });
const out = path.join(DST, `baseline-${date}.json`);
writeFileSync(out, JSON.stringify(summary, null, 2));
console.log(`Wrote ${out}`);
console.log(`  ${summary.totals.violations} violations across ${summary.totals.routes} routes`);
console.log(`  Impact: critical=${summary.totals.impact.critical} serious=${summary.totals.impact.serious} moderate=${summary.totals.impact.moderate} minor=${summary.totals.impact.minor}`);
console.log(`  Top SCs:`);
const scCounts = Object.entries(byTag).map(([tag, vs]) => [tag, vs.length]).sort((a, b) => b[1] - a[1]);
for (const [tag, count] of scCounts.slice(0, 10)) {
  console.log(`    ${tag}: ${count}`);
}
