import { test as setup } from '@playwright/test';

/**
 * Phase 0 axe baseline: auth fixture.
 *
 * Most app routes are gated behind ProtectedRoute → TenantScope → JWT. Without
 * a real session, axe would crawl the login page 140 times. We seed storage
 * with a JWT-shaped payload and a tenant, then expose it via storageState so
 * axe specs run authenticated.
 *
 * NOTE: this is a structural fixture for offline/CI use; the real JWT comes
 * from the auth-service in production. Replace LOCAL_STORAGE_AUTH with an
 * apiClient login call when the auth fixture is wired up.
 */
const LOCAL_STORAGE_AUTH = {
  auth_token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJpYXQiOjE3MDAwMDAwMDAsImV4cCI6OTk5OTk5OTk5OX0.fake',
  user: {
    id: '00000000-0000-0000-0000-000000000001',
    username: 'a11y-fixture',
    email: 'a11y@example.com',
    role: 'admin',
  },
  selected_tenant: JSON.stringify({
    id: '00000000-0000-0000-0000-000000000000',
    display_name: 'Dev Tenant',
  }),
  appLocale: 'en',
};

setup('seed auth storage', async ({ page, context }) => {
  await page.goto('/');
  await page.evaluate((auth) => {
    for (const [k, v] of Object.entries(auth)) {
      localStorage.setItem(k, v);
    }
  }, LOCAL_STORAGE_AUTH);
  await context.storageState({ path: 'e2e/a11y/.auth-storage.json' });
});
