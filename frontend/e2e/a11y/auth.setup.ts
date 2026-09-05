import { test as setup } from '@playwright/test';
import {
  E2E_JWT,
  KC_USER,
  E2E_USER,
  EXPIRES_AT,
  getKeycloakToken,
  FALLBACK_JWT,
} from './auth';

/**
 * Phase 0 axe baseline: auth fixture.
 *
 * Token generation strategy (in priority order):
 * 1. E2E_JWT env var         — CI/remote sets this; no secrets in tracked files
 * 2. Keycloak password grant — if E2E_KC_USER + E2E_KC_PASS are set; token is
 *    fetched once per test run and memoized at module level.
 * 3. Fake JWT               — fallback for local dev where Keycloak is unreachable;
 *    localStorage-only auth works for route-level axe scanning.
 *
 * The fake JWT is accepted by the SPA's client-side routing gate (ProtectedRoute
 * checks token presence). API calls fail silently — visible as console errors, not
 * as redirects. The negative control (see baseline.spec.ts) detects this case.
 */

setup('seed auth storage via addInitScript', async ({ page, context }) => {
  let jwt: string;
  if (E2E_JWT) {
    jwt = E2E_JWT;
  } else if (KC_USER) {
    jwt = await getKeycloakToken();
  } else {
    console.warn(
      '[auth.setup] No E2E_JWT and no E2E_KC_USER — using fallback fake JWT. ' +
      'API calls will fail silently. Set both env vars for a real token.',
    );
    jwt = FALLBACK_JWT;
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
