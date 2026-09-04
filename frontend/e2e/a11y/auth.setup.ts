import { test as setup } from '@playwright/test';

/**
 * Phase 0 axe baseline: auth fixture.
 *
 * Most app routes are gated behind ProtectedRoute → TenantScope → JWT. Without
 * a real session, axe would crawl the login page 140 times. We seed storage
 * with a JWT-shaped payload and a tenant, then expose it via storageState so
 * axe specs run authenticated.
 *
 * oidc-client's loadUser() calls Keycloak on init and clears auth on failure.
 * We use page.addInitScript() to seed localStorage BEFORE the app bootstraps,
 * so the seeded auth is in place when loadUser() runs. This prevents the
 * Keycloak call from racing against our seed.
 *
 * To upgrade to real auth: obtain a Keycloak access_token for a seeded test user
 * and set E2E_JWT in the environment.
 */
const E2E_JWT = process.env.E2E_JWT ?? 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJpYXQiOjE3MDAwMDAwMDAsImV4cCI6OTk5OTk5OTk5OX0.fake';
const EXPIRES_AT = Date.now() + 86400 * 1000;

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

setup('seed auth storage via addInitScript', async ({ page, context }) => {
  await page.context().addInitScript(
    ({ jwt, user, expiresAt }) => {
      localStorage.setItem('auth_token', jwt);
      localStorage.setItem('auth_user', JSON.stringify(user));
      localStorage.setItem('auth_expires_at', expiresAt.toString());
      localStorage.setItem('selected_tenant', JSON.stringify({
        id: '00000000-0000-0000-0000-000000000000',
        display_name: 'Dev Tenant',
      }));
      localStorage.setItem('appLocale', 'en');
    },
    { jwt: E2E_JWT, user: E2E_USER, expiresAt: EXPIRES_AT },
  );
  await context.storageState({ path: 'e2e/a11y/.auth-storage.json' });
});
