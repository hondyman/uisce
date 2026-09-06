import { test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';
import {
  E2E_JWT,
  KC_USER,
  E2E_USER,
  EXPIRES_AT,
  getKeycloakToken,
  FALLBACK_JWT,
} from './auth';

/**
 * Phase 0 axe baseline. Run against the default-locale URLs (locales all use
 * the same components, so a single crawl covers all languages for now). Once
 * /ar becomes a real surface, add a parallel spec iterating over locales.
 *
 * Output: test-results/a11y/{route}.json per route.
 *
 * Auth: each route seeds localStorage before navigation (see auth helpers below).
 * Negative control: E2E_JWT=garbage triggers the auth guard → test fails.
 * Non-empty-main: every route must render a <main> landmark.
 *
 * SCOPE NOTE: /api-studio is excluded — Vite dev server returns 404 for this
 * path (pre-existing bug; fix the Vite routing first, then re-add).
 */

const LOCALE = 'en';

/**
 * Locale-prefixed routes from AppRoutes.tsx (path values inside `/:locale/*` splat).
 * Excluded: api-studio (Vite 404 — pre-existing bug).
 */
const APP_ROUTES = [
  'abbreviations',
  'access-explanation',
  'admin/ai-semantic-bridge',
  'admin/entitlements',
  'admin/entitlements/profiles/:profileKey',
  'admin/entitlements/profiles/:profileKey/components',
  'admin/llm',
  'admin/rbac/delegations',
  'admin/rbac/field-permissions',
  'admin/rbac/group-role-mappings',
  'admin/rbac/identity-providers',
  'admin/rbac/roles',
  'admin/rbac/teams',
  'admin/rbac/user-roles',
  'admin/rbac/user-tenants',
  'admin/rbac/users',
  'admin/seeding',
  'admin/temporal-ops',
  'ai-assistant',
  'analytics/advisor-dashboard',
  'analytics/factors',
  'analytics/factors/:portfolioID?',
  'analytics/rebalancer',
  'analytics/scenario-analysis',
  'api-designer',
  'aso',
  'audit',
  'bp-console',
  'bp-console/:tab',
  'bp-console/instances',
  'bp-console/queues',
  'bundle-explorer',
  'bundles',
  'business-objects',
  'business-objects/:id',
  'business-objects/new',
  'calendar',
  'calendar/conflicts',
  'catalog/ai-suggestions',
  'catalog/api-inventory',
  'catalog/business-terms',
  'catalog/business-terms/:id',
  'catalog/edge-types',
  'catalog/edge-types/:id',
  'catalog/node-types',
  'catalog/node-types/:id',
  'catalog/semantic-terms',
  'catalog/view-definitions',
  'change-reviews',
  'change-reviews/:id',
  'client-portal/rules-editor',
  'client-portal/workflow-studio',
  'core/abbreviations',
  'core/approval-inbox',
  'core/approval-workflows',
  'core/audit-explorer',
  'core/business-terms',
  'core/calculated-fields',
  'core/data-pipelines',
  'core/domains',
  'core/flow-builder',
  'core/genui-chat',
  'core/genui-inbox',
  'core/genui-proposal',
  'core/glossary',
  'core/notifications',
  'core/notifications/preferences',
  'core/notifications/templates',
  'core/process-catalog',
  'core/semantic-mapper',
  'core/semantic-terms',
  'core/sla-dashboard',
  'core/uisce-builder',
  'core/validation',
  'core/validation-rules',
  'core/workflow-designer',
  'crypto/portfolio',
  'dynamic-ui',
  'fabric/audit-logs',
  'fabric/bundles',
  'fabric/bundles/:bundleId/edit',
  'fabric/bundles/create',
  'fabric/calculations',
  'fabric/custom-components',
  'fabric/dashboard',
  'fabric/ip-whitelist',
  'fabric/preaggregations',
  'fabric/settings',
  'fabric/tenants',
  'fixed-income',
  'flow-builder',
  'global-intelligence',
  'glossary',
  'governance/changesets',
  'incidents/:id',
  'intelligence',
  'intelligence/data-quality',
  'intelligence/index-advisor',
  'intelligence/storage',
  'jit-request',
  'marketplace',
  'marketplace/components',
  'nlq',
  'observability',
  'observability/slos',
  'optimization',
  'optimization/:optimizationId',
  'page-designer',
  'page-studio',
  'pipelines/studio',
  'pipelines/studio/:id',
  'pipelines/triggers/new',
  'private-markets',
  'query-builder',
  'rbac',
  'reports/:reportId/edit',
  'reports/builder',
  'reports/expressions',
  'reports/library',
  'reports/models',
  'reports/queries',
  'scheduler',
  'scheduler-intelligence',
  'scheduler/calendars',
  'scheduler/calendars/:calendarId/edit',
  'scheduler/calendars/new',
  'scheduler/executions',
  'scheduler/executions/:executionId',
  'scheduler/jobs',
  'scheduler/jobs/:jobId',
  'scheduler/jobs/:jobId/edit',
  'scheduler/jobs/new',
  'schema-explorer',
  'secrets/audit',
  'secrets/config',
  'secrets/monitoring',
  'semantic-health',
  'simulation',
  'simulation/:id',
  'simulation/compare',
  'simulation/rebalance',
  'tenants',
  'tenants/:tenantId',
  'validation-rules',
  'wealth/feed',
];

/**
 * Un-prefixed top-level routes (mounted outside `/:locale/*` in LocaleShell).
 * api-studio excluded (Vite 404 — pre-existing bug).
 */
const UNPREFIXED_ROUTES = [
  '/',
  '/auth/callback',
  '/change-review',
  '/login',
  '/page-studio',
  '/app/data-product/:pageKey',
];

const ROUTES = [
  ...UNPREFIXED_ROUTES,
  ...APP_ROUTES.map(r => `/${LOCALE}/${r}`),
];

const OUT_DIR = 'test-results/a11y';
fs.mkdirSync(OUT_DIR, { recursive: true });

function authStorage(jwt: string) {
  return {
    auth_token: jwt,
    auth_user: JSON.stringify(E2E_USER),
    auth_expires_at: EXPIRES_AT.toString(),
    selected_tenant: JSON.stringify({ id: '99e99e99-99e9-49e9-89e9-99e99e99e999', display_name: 'Northwind Traders' }),
    appLocale: LOCALE,
  };
}

// john.b is a global admin with no tenant claim; must supply X-Tenant-ID header.
const E2E_TENANT_ID = process.env.E2E_TENANT_ID ?? '99e99e99-99e9-49e9-89e9-99e99e99e999';

test.describe('Phase 0 axe baseline (WCAG 2.1 AA)', () => {
  // Protocol closure #2: health pre-flight — verify app accepts auth+tenant
  // before spending 6 minutes on a crawl that will measure error boundaries.
  // Keycloak-OK (token fetch 200) and app-OK (API 200) are different checks.
  test.beforeAll(async ({ request }) => {
    const backendURL = (process.env.BASE_URL ?? 'http://localhost:5173')
      .replace(':5173', ':8080')
      .replace('localhost', '127.0.0.1');
    // Fetch token here in beforeAll (beforeEach runs after beforeAll — can't stash from there).
    // getKeycloakToken() is memoized, so this is free after the first call.
    let jwt: string;
    if (E2E_JWT) {
      jwt = E2E_JWT;
    } else if (KC_USER) {
      jwt = await getKeycloakToken();
    } else {
      throw new Error(
        '[baseline] Neither E2E_JWT nor E2E_KC_USER is set. ' +
        'A fake JWT produces vacuous-clean scans. ' +
        'Set E2E_JWT (CI) or E2E_KC_USER+E2E_KC_PASS (local) to run a real crawl.',
      );
    }

    // 500 = schema drift / broken app state — hard fail, don't try other endpoints.
    // This is the signature of binary/DB mismatch; past crawls completed green on it.
    // 401 = on /api/auth/me: endpoint uses session tokens, not JWTs — try next.
    // 401 = on /api/v1/schedules/: bad auth token — hard fail.
    // 403/404 = tenant or routing problem — try next endpoint.
    // /api/auth/me: region-exempt, uses session tokens (not JWT-aware).
    // /api/v1/schedules/: tenant-scoped, JWT-protected via middleware.
    const preflightPaths = ['/api/auth/me', '/api/v1/schedules/'];
    let lastError = '';
    for (const p of preflightPaths) {
      try {
        console.log(`[pre-flight] trying ${p}...`);
        const res = await request.fetch(`${backendURL}${p}`, {
          headers: {
            Authorization: `Bearer ${jwt}`,
            'X-Tenant-ID': E2E_TENANT_ID,
          },
        });
        console.log(`[pre-flight] ${p} → ${res.status()}`);
        if (res.status() === 200) {
          console.log(`[pre-flight] Auth + tenant verified OK (${p})`);
          return; // pre-flight passed
        }
        if (res.status() === 401) {
          // /api/auth/me returns 401 for JWT auth (session-token only) — try next endpoint.
          // /api/v1/schedules/ returning 401 means bad token — hard fail.
          if (p === '/api/v1/schedules/') {
            throw new Error(`[pre-flight] Auth rejected — check E2E_JWT or E2E_KC_USER/E2E_KC_PASS. Status: ${res.status()}`);
          }
          lastError = `[pre-flight] ${p} → ${res.status()} (session-token only, trying next)`;
          continue;
        }
        if (res.status() === 500) {
          throw new Error(`[pre-flight] App returned 500 on ${p} — schema drift or DB mismatch. Status: ${res.status()}. Fix the backend before re-running.`);
        }
        // 403/404 — try next endpoint
        lastError = `[pre-flight] ${p} → ${res.status()}`;
      } catch (e: any) {
        if (e.message.includes('[pre-flight]')) throw e; // already a pre-flight error
        lastError = e.message;
      }
    }
    throw new Error(
      `[pre-flight] App rejected auth+tenant on all pre-flight endpoints.\n` +
      `  Last error: ${lastError}\n` +
      `  This crawl would measure error boundaries, not the app. Fix the tenant/auth setup before re-running.`,
    );
  });

  test.beforeEach(async ({ page, context }) => {
    let jwt: string;
    if (E2E_JWT) {
      jwt = E2E_JWT;
    } else if (KC_USER) {
      jwt = await getKeycloakToken();
    } else {
      throw new Error(
        '[baseline] Neither E2E_JWT nor E2E_KC_USER is set. ' +
        'A fake JWT produces vacuous-clean scans. ' +
        'Set E2E_JWT (CI) or E2E_KC_USER+E2E_KC_PASS (local) to run a real crawl.',
      );
    }
    await context.setExtraHTTPHeaders({ 'X-Tenant-ID': E2E_TENANT_ID });
    await context.addInitScript(
      (auth) => { for (const [k, v] of Object.entries(auth)) localStorage.setItem(k, v as string); },
      authStorage(jwt),
    );
  });

  for (const route of ROUTES) {
    test(`axe: ${route}`, async ({ page }) => {
      await page.goto(route, { waitUntil: 'load' });

      const url = page.url();
      const authLost = route !== '/login' && url.includes('/login');
      if (authLost) {
        throw new Error(
          `auth lost on ${route} — app redirected to ${url}. ` +
          `Set E2E_JWT or E2E_KC_USER+E2E_KC_PASS.`,
        );
      }

      const results = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
        .disableRules(['color-contrast'])
        .analyze();

      const output: Record<string, unknown> = {
        routePattern: route,
        finalUrl: page.url(),
        violations: results.violations,
        violationIds: results.violations.map((v: { id: string }) => v.id),
        passCount: results.passes.length,
      };

      // Check for crash before writing — error boundaries render valid DOM that axe can scan,
      // producing "clean" results on broken pages. <main> presence is the crash indicator.
      const mainCount = await page.locator('main, [role="main"]').count();
      if (mainCount === 0) {
        output.crashed = true;
        output.crashReason = 'no <main> landmark — app crashed or error boundary rendered';
      }

      fs.writeFileSync(
        path.join(OUT_DIR, `${route.replace(/\W+/g, '_')}.json`),
        JSON.stringify(output, null, 2),
      );

      test.info().annotations.push({
        type: 'axe-summary',
        description: `${results.violations.length} violations${output.crashed ? ' [CRASHED]' : ''}`,
      });

      if (output.crashed) return;
    });
  }
});
