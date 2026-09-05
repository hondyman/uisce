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
    selected_tenant: JSON.stringify({ id: '00000000-0000-0000-0000-000000000000', display_name: 'Dev Tenant' }),
    appLocale: LOCALE,
  };
}

test.describe('Phase 0 axe baseline (WCAG 2.1 AA)', () => {
  test.beforeEach(async ({ page, context }) => {
    let jwt: string;
    if (E2E_JWT) {
      jwt = E2E_JWT;
    } else if (KC_USER) {
      jwt = await getKeycloakToken();
    } else {
      console.warn('[baseline] using fallback fake JWT — set E2E_JWT or E2E_KC_USER+E2E_KC_PASS');
      jwt = FALLBACK_JWT;
    }
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

      const output = {
        routePattern: route,
        finalUrl: page.url(),
        violations: results.violations,
        violationIds: results.violations.map((v: { id: string }) => v.id),
      };
      fs.writeFileSync(
        path.join(OUT_DIR, `${route.replace(/\W+/g, '_')}.json`),
        JSON.stringify(output, null, 2),
      );

      test.info().annotations.push({
        type: 'axe-summary',
        description: `${results.violations.length} violations`,
      });

      const mainCount = await page.locator('main, [role="main"]').count();
      if (mainCount === 0) {
        throw new Error(`no <main> landmark on ${route} — page may have failed to render`);
      }
    });
  }
});
