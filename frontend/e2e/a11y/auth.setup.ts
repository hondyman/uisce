import { test as setup } from '@playwright/test';

/**
 * Phase 0 axe baseline: auth fixture.
 *
 * Token generation strategy (in priority order):
 * 1. E2E_JWT env var         — CI/remote sets this; no secrets in tracked files
 * 2. Keycloak password grant — if E2E_KC_USER + E2E_KC_PASS are set; fetches
 *    a real OIDC token at fixture setup time (not hardcoded, not tracked)
 * 3. Fake JWT               — fallback for local dev where Keycloak is unreachable;
 *    localStorage-only auth works for route-level axe scanning
 *
 * The fake JWT is accepted by the SPA's client-side routing gate (ProtectedRoute
 * checks token presence). API calls fail silently — visible as console errors, not
 * as redirects. The negative control (see baseline.spec.ts) detects this case.
 */
const E2E_JWT = process.env.E2E_JWT;
const KC_USER = process.env.E2E_KC_USER;
const KC_PASS = process.env.E2E_KC_PASS;
const KC_ISSUER = process.env.E2E_KC_ISSUER ?? 'https://100.84.50.65:8443/realms/uisce';

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

async function fetchKeycloakToken(): Promise<string | null> {
  if (!KC_USER || !KC_PASS) return null;
  try {
    const res = await fetch(
      `${KC_ISSUER}/protocol/openid-connect/token`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
          grant_type: 'password',
          username: KC_USER,
          password: KC_PASS,
          client_id: 'semlayer-frontend',
          scope: 'openid profile email',
        }),
      },
    );
    if (!res.ok) {
      console.warn(`[auth.setup] Keycloak token fetch failed: ${res.status} ${await res.text()}`);
      return null;
    }
    const data = await res.json() as { access_token: string };
    return data.access_token ?? null;
  } catch (err) {
    console.warn(`[auth.setup] Keycloak unreachable: ${err}`);
    return null;
  }
}

const FALLBACK_JWT = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJpYXQiOjE3MDAwMDAwMDAsImV4cCI6OTk5OTk5OTk5OX0.fake';

setup('seed auth storage via addInitScript', async ({ page, context }) => {
  const jwt = E2E_JWT ?? (await fetchKeycloakToken()) ?? FALLBACK_JWT;

  if (!E2E_JWT && !KC_USER) {
    console.warn(
      '[auth.setup] No E2E_JWT and no E2E_KC_USER — using fallback fake JWT. ' +
      'API calls will fail silently. Set both env vars for a real token.',
    );
  }

  await page.context().addInitScript(
    (auth) => {
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
  await context.storageState({ path: 'e2e/a11y/.auth-storage.json' });
});
