#!/usr/bin/env node
/**
 * aggregate-a11y-baseline.mjs
 *
 * Reads per-route axe JSON files from `frontend/test-results/a11y/*.json` and
 * produces a single aggregate baseline artifact in `frontend/docs/a11y/`.
 *
 * Usage: node scripts/aggregate-a11y-baseline.mjs [--output <path>]
 *
 * JSON format (self-describing, committed in baseline.spec.ts):
 *   { routePattern: string, finalUrl: string, axe: { violations, passes, ... } }
 *
 * Backwards-compatible with old format:
 *   { violations, passes, ... }  (no routePattern/finalUrl)
 *
 * isParam is derived from ':' in routePattern — no hardcoded list.
 * A route is "non-representative clean" if isParam=true AND violations=0.
 */
import { readFileSync, writeFileSync, readdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const FRONTEND = join(__dirname, '..', 'frontend');
const RESULTS_DIR = join(FRONTEND, 'test-results', 'a11y');
const DEFAULT_OUTPUT = join(FRONTEND, 'docs', 'a11y', `baseline-${new Date().toISOString().slice(0, 10)}.json`);

const outputPath = process.argv[2] === '--output' ? process.argv[3] : DEFAULT_OUTPUT;

const files = readdirSync(RESULTS_DIR).filter(f => f.endsWith('.json'));

const result = {
  metadata: {
    generated: new Date().toISOString(),
    tool: 'axe-core playwright',
    browser: 'chromium',
    totalRoutes: files.length,
    paramRoutes: 0,
    nonRepClean: 0,
    nonRepWithViolations: 0,
    repRoutesWithViolations: 0,
    repViolations: 0,
    totalViolations: 0,
    routesWithViolations: 0,
    byRule: {},
    routes: [],
  },
};

for (const file of files) {
  const raw = readFileSync(join(RESULTS_DIR, file), 'utf8');
  const d = JSON.parse(raw);

  // New stripped format: { routePattern, finalUrl, violations, violationIds }
  // Legacy format: { routePattern, finalUrl, axe: { violations, passes, ... } }
  // Oldest format: { violations, passes, ... } at root level (no routePattern)
  const routePattern = d.routePattern ?? ('/' + file.replace(/^_/, '').replace(/_/g, '/').replace(/\.json$/, ''));
  const finalUrl = d.finalUrl ?? null;
  let violations;
  if (Array.isArray(d.violations)) {
    // New stripped format: violations at root
    violations = d.violations;
  } else if (d.axe && Array.isArray(d.axe.violations)) {
    // Legacy wrapped format
    violations = d.axe.violations;
  } else {
    // Oldest format
    violations = d.violations ?? [];
  }
  const isParam = routePattern.includes(':');
  const wasRedirected = finalUrl && finalUrl !== `http://localhost:5173${routePattern}`;
  // Non-representative: param route visited literally (non-rep clean),
  // OR param route that redirected (caught by the router's fallback/error handler)
  const wasNonRep = isParam && (violations.length === 0 || wasRedirected);

  if (isParam) result.metadata.paramRoutes++;
  if (wasNonRep && violations.length === 0) result.metadata.nonRepClean++;
  if (wasNonRep && violations.length > 0) result.metadata.nonRepWithViolations++;
  result.metadata.totalViolations += violations.length;
  if (violations.length > 0) result.metadata.routesWithViolations++;
  if (!wasNonRep && violations.length > 0) result.metadata.repRoutesWithViolations++;
  if (!wasNonRep) result.metadata.repViolations += violations.length;

  for (const v of violations) {
    result.metadata.byRule[v.id] = result.metadata.byRule[v.id] ||
      { count: 0, impact: v.impact, description: v.description };
    result.metadata.byRule[v.id].count++;
  }

  result.metadata.routes.push({
    routePattern,
    finalUrl,
    isParam,
    wasNonRep,
    violations: violations.length,
    violationIds: violations.map(v => v.id),
  });
}

result.metadata.byRule = Object.entries(result.metadata.byRule)
  .sort((a, b) => b[1].count - a[1].count)
  .reduce((o, [k, v]) => ({ ...o, [k]: v }), {});

writeFileSync(outputPath, JSON.stringify(result, null, 2));
console.log(`Written: ${outputPath}`);
console.log(`  Routes: ${result.metadata.totalRoutes}`);
console.log(`  Param routes: ${result.metadata.nonRepClean} clean non-rep + ${result.metadata.nonRepWithViolations} non-rep w/violations / ${result.metadata.paramRoutes} total`);
console.log(`  Representative denominator: ${result.metadata.totalRoutes - result.metadata.nonRepClean - result.metadata.nonRepWithViolations}`);
console.log(`  Representative violations: ${result.metadata.repViolations} across ${result.metadata.repRoutesWithViolations} routes`);
console.log(`  Total violations (incl. non-rep): ${result.metadata.totalViolations} across ${result.metadata.routesWithViolations} routes`);
console.log(`  Rules: ${Object.keys(result.metadata.byRule).length}`);
