#!/usr/bin/env node
/**
 * aggregate-a11y-baseline.mjs
 *
 * Reads per-route axe JSON files from `frontend/test-results/a11y/*.json` and
 * produces a single aggregate baseline artifact in `frontend/docs/a11y/`.
 *
 * Usage: node scripts/aggregate-a11y-baseline.mjs [--output <path>]
 *
 * Output format: JSON with metadata + per-route array.
 * Denominator flags:
 *   - param routes (URL contains :param) are marked isParam=true — these scan a
 *     literal ":param" segment and are non-representative (catch-all → error/redirect).
 *   - Clean param routes (0 violations) are counted separately as non-representative
 *     in metadata.cleanParamRoutes.
 */
import { readFileSync, writeFileSync, readdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const FRONTEND = join(__dirname, '..', 'frontend');
const RESULTS_DIR = join(FRONTEND, 'test-results', 'a11y');
const DEFAULT_OUTPUT = join(FRONTEND, 'docs', 'a11y', `baseline-${new Date().toISOString().slice(0, 10)}.json`);

const paramRouteKeys = new Set([
  '_app_data_product_pageKey',
  '_en_admin_entitlements_profiles_profileKey',
  '_en_admin_entitlements_profiles_profileKey_components',
  '_en_analytics_factors_portfolioID_',
  '_en_bp_console_tab',
  '_en_business_objects_id',
  '_en_change_reviews_id',
  '_en_incidents_id',
  '_en_optimization_optimizationId',
  '_en_pipelines_studio_id',
  '_en_reports_reportId_edit',
  '_en_scheduler_calendars_calendarId_edit',
  '_en_scheduler_executions_executionId',
  '_en_scheduler_jobs_jobId',
  '_en_scheduler_jobs_jobId_edit',
  '_en_simulation_id',
  '_en_tenants_tenantId',
]);

const outputPath = process.argv[2] === '--output' ? process.argv[3] : DEFAULT_OUTPUT;

const files = readdirSync(RESULTS_DIR).filter(f => f.endsWith('.json'));

const result = {
  metadata: {
    generated: new Date().toISOString(),
    tool: 'axe-core playwright',
    browser: 'chromium',
    totalRoutes: files.length,
    paramRoutes: paramRouteKeys.size,
    cleanParamRoutes: 0,
    totalViolations: 0,
    totalPasses: 0,
    routesWithViolations: 0,
    byRule: {},
    routes: [],
  },
};

for (const file of files) {
  const raw = readFileSync(join(RESULTS_DIR, file), 'utf8');
  const d = JSON.parse(raw);
  const route = '/' + file.replace(/^_/, '').replace(/_/g, '/').replace(/\.json$/, '');
  const key = file.replace(/\.json$/, '');
  const isParam = paramRouteKeys.has(key);

  if (isParam && d.violations.length === 0) result.metadata.cleanParamRoutes++;

  result.metadata.totalViolations += d.violations.length;
  result.metadata.totalPasses += d.passes.length;
  if (d.violations.length > 0) result.metadata.routesWithViolations++;

  for (const v of d.violations) {
    result.metadata.byRule[v.id] = result.metadata.byRule[v.id] ||
      { count: 0, impact: v.impact, description: v.description };
    result.metadata.byRule[v.id].count++;
  }

  result.metadata.routes.push({
    route,
    violations: d.violations.length,
    passes: d.passes.length,
    isParam,
    violationIds: d.violations.map(v => v.id),
  });
}

result.metadata.byRule = Object.entries(result.metadata.byRule)
  .sort((a, b) => b[1].count - a[1].count)
  .reduce((o, [k, v]) => ({ ...o, [k]: v }), {});

writeFileSync(outputPath, JSON.stringify(result, null, 2));
console.log(`Written: ${outputPath}`);
console.log(`  Routes: ${result.metadata.totalRoutes}`);
console.log(`  Param routes (non-representative if clean): ${result.metadata.cleanParamRoutes} / ${result.metadata.paramRoutes}`);
console.log(`  Violations: ${result.metadata.totalViolations} across ${result.metadata.routesWithViolations} routes`);
console.log(`  Rules: ${Object.keys(result.metadata.byRule).length}`);
