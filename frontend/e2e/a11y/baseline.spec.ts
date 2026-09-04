import { test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';

/**
 * Phase 0 axe baseline. Run against the default-locale URLs (locales all use
 * the same components, so a single crawl covers all languages for now). Once
 * /ar becomes a real surface, add a parallel spec iterating over locales.
 *
 * Output: test-results/a11y/{route}.json per route.
 *
 * SCOPE NOTE: /api-studio is excluded — Vite dev server returns 404 for this
 * path (pre-existing bug; scope decision: fix the Vite routing first).
 */

const E2E_JWT = process.env.E2E_JWT ?? 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJpYXQiOjE3MDAwMDAwMDAsImV4cCI6OTk5OTk5OTk5OX0.fake';

const E2E_USER = {
  id: '00000000-0000-0000-0000-000000000001',
  email: 'a11y@example.com',
  name: 'A11y Fixture',
  role: 'admin',
  organization: 'E2E Test',
  permissions: [],
  is_active: true,
  roles: ['admin', 'user'],
  is_core_admin: true,
  isCoreAdmin: true,
  is_admin: true,
  is_global_admin: true,
};

const ROUTES = [
  '/',
  '/en/',
  '/en/dashboard',
  '/en/scheduler/jobs',
  '/en/reports/library',
  '/en/admin/rbac/roles',
  '/en/business-objects',
  '/en/core/glossary',
  '/login',
];

const EXPIRES_AT = Date.now() + 86400 * 1000;

const AUTH_STORAGE = {
  auth_token: E2E_JWT,
  auth_user: JSON.stringify(E2E_USER),
  auth_expires_at: EXPIRES_AT.toString(),
  selected_tenant: JSON.stringify({ id: '00000000-0000-0000-0000-000000000000', display_name: 'Dev Tenant' }),
  appLocale: 'en',
};

test.describe('Phase 0 axe baseline (WCAG 2.1 AA)', () => {
  test.beforeEach(async ({ page, context }) => {
    await context.addInitScript(
      (auth) => {
        for (const [k, v] of Object.entries(auth)) {
          localStorage.setItem(k, v);
        }
      },
      AUTH_STORAGE,
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
          `Token may be expired or rejected by oidc-client. ` +
          `Set E2E_JWT to a real Keycloak access_token.`,
        );
      }

      const results = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
        .disableRules(['color-contrast'])
        .analyze();

      const dir = 'test-results/a11y';
      fs.mkdirSync(dir, { recursive: true });
      fs.writeFileSync(
        path.join(dir, `${route.replace(/\W+/g, '_')}.json`),
        JSON.stringify(results, null, 2),
      );

      test.info().annotations.push({
        type: 'axe-summary',
        description: `${results.violations.length} rule violations, ${results.passes.length} passes`,
      });
    });
  }
});
