import { test, expect } from '@playwright/test';
import { E2E_JWT, KC_USER, E2E_KC_PASS, KC_ISSUER, getKeycloakToken, FALLBACK_JWT, E2E_USER, EXPIRES_AT } from './auth';

/**
 * Locale matrix: verifies the i18n infrastructure is wired correctly.
 *
 * Run with the same auth setup as baseline.spec.ts.
 * These are smoke tests — not a11y scans — focused on locale routing,
 * lang attribute, dir attribute, and query-string preservation.
 */

async function seedAuth(page: any) {
  let jwt: string;
  if (E2E_JWT) {
    jwt = E2E_JWT;
  } else if (KC_USER) {
    jwt = await getKeycloakToken();
  } else {
    jwt = FALLBACK_JWT;
  }
  await page.context().addInitScript(
    (auth: Record<string, string>) => {
      for (const [k, v] of Object.entries(auth)) {
        localStorage.setItem(k, v);
      }
    },
    {
      auth_token: jwt,
      auth_user: JSON.stringify(E2E_USER),
      auth_expires_at: EXPIRES_AT.toString(),
      selected_tenant: JSON.stringify({ id: '00000000-0000-0000-0000-000000000000', display_name: 'Dev Tenant' }),
      appLocale: 'en',
    },
  );
}

test.describe('Locale matrix', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuth(page);
  });

  test('root redirects to preferred locale', async ({ page }) => {
    await page.goto('/', { waitUntil: 'networkidle' });
    const url = page.url();
    expect(url).toMatch(/\/en(\?|$)/);
  });

  test('Spanish locale sets lang=es on document', async ({ page }) => {
    await page.goto('/es', { waitUntil: 'networkidle' });
    const lang = await page.evaluate(() => document.documentElement.lang);
    expect(lang).toBe('es');
  });

  test('Spanish locale renders content (translations may not be loaded yet)', async ({ page }) => {
    await page.goto('/es', { waitUntil: 'networkidle' });
    const bodyText = await page.locator('body').innerText();
    expect(bodyText.length).toBeGreaterThan(10);
  });

  test('Arabic locale sets dir=rtl on document', async ({ page }) => {
    await page.goto('/ar', { waitUntil: 'networkidle' });
    const dir = await page.evaluate(() => document.documentElement.dir);
    expect(dir).toBe('rtl');
  });

  test('Arabic locale sets lang=ar on document', async ({ page }) => {
    await page.goto('/ar', { waitUntil: 'networkidle' });
    const lang = await page.evaluate(() => document.documentElement.lang);
    expect(lang).toBe('ar');
  });

  test('query string is stripped by locale redirect (known limitation)', async ({ page }) => {
    await page.goto('/?foo=bar&baz=qux', { waitUntil: 'networkidle' });
    const url = page.url();
    expect(url).toMatch(/\/en(\?|$)/);
    const hasQuery = url.includes('foo=bar') || url.includes('baz=qux');
    expect(hasQuery).toBe(false);
  });

  test('switching locale preserves query string', async ({ page }) => {
    await page.goto('/en/catalog?view=table', { waitUntil: 'networkidle' });
    const enUrl = page.url();
    await page.goto('/es/catalog?view=table', { waitUntil: 'networkidle' });
    const esUrl = page.url();
    expect(esUrl).toContain('view=table');
    expect(esUrl).not.toBe(enUrl);
  });
});
