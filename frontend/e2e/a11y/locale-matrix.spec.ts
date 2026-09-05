import { test, expect } from '@playwright/test';
import {
  E2E_JWT,
  KC_USER,
  E2E_USER,
  EXPIRES_AT,
  getKeycloakToken,
  FALLBACK_JWT,
} from './auth';

/**
 * Locale matrix: verifies the i18n infrastructure is wired correctly.
 * Auth seeding shared with baseline.spec.ts via auth.ts.
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

  test.fixme(
    'Spanish locale renders translated nav chrome',
    async ({ page }) => {
      await page.goto('/es', { waitUntil: 'networkidle' });
      const bodyText = await page.locator('body').innerText();
      expect(bodyText.length).toBeGreaterThan(10);
      const spanishPatterns = [/configuración/i, /administración/i, /usuarios/i, /inicio/i];
      const hasSpanish = spanishPatterns.some(p => p.test(bodyText));
      expect(hasSpanish).toBe(true);
    },
    'MainNavigation.tsx hardcodes Platform/Organization/Security/System labels — ' +
    'nav chrome is Phase 3 extraction backlog; i18n bundle loads but chrome labels ' +
    'are not yet translated. When chrome is extracted to i18n keys, update the ' +
    'assertion to check for a genuinely translated string.',
  );

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

  test.fixme(
    'root redirect preserves query string across locale redirect',
    async ({ page }) => {
      await page.goto('/?foo=bar&baz=qux', { waitUntil: 'networkidle' });
      const url = page.url();
      expect(url).toContain('foo=bar');
      expect(url).toContain('baz=qux');
    },
    'RootRedirect in localeShell.tsx navigates to /{locale} without search/hash. ' +
    'Fix: useLocation() in RootRedirect, append search+hash to Navigate target. ' +
    'This is the original Blocker 1 regression — resolveLocaleRoute was fixed but ' +
    'RootRedirect was not in the enumerated site list.',
  );

  test.fixme(
    'locale-to-locale navigation preserves query string',
    async ({ page }) => {
      await page.goto('/en/catalog?view=table', { waitUntil: 'networkidle' });
      const enUrl = page.url();
      await page.goto('/es/catalog?view=table', { waitUntil: 'networkidle' });
      const esUrl = page.url();
      expect(esUrl).toContain('view=table');
      expect(esUrl).not.toBe(enUrl);
    },
    'RootRedirect fix (above) is the pre-req; then /en/catalog?view=table → ' +
    '/es/catalog?view=table should preserve the query via resolveLocaleRoute ' +
    'search/hash composition. Currently unit-tested only; no browser verification.',
  );
});
