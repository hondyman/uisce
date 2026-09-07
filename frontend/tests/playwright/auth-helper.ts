/**
 * Shared E2E auth helpers — Keycloak password-grant token acquisition
 * and localStorage seeding. Borrowed from frontend/e2e/a11y/auth.ts and
 * adapted for use across all playwright smoke tests.
 *
 * Env vars required (one of):
 *   E2E_JWT          — pre-minted JWT (CI path)
 *   E2E_KC_USER      — Keycloak user for password grant (local path)
 *   E2E_KC_PASS      — Keycloak password
 *
 * Optional:
 *   E2E_KC_ISSUER    — Keycloak issuer URL
 *                        (default: https://100.84.50.65:8443/realms/uisce)
 *   E2E_TENANT_ID     — Tenant ID for X-Tenant-ID header
 *                        (default: 99e99e99-99e9-49e9-89e9-99e99e99e999)
 */

export const E2E_JWT = process.env.E2E_JWT;
export const KC_USER = process.env.E2E_KC_USER;
export const KC_PASS = process.env.E2E_KC_PASS;
export const KC_ISSUER =
  process.env.E2E_KC_ISSUER ?? 'https://100.84.50.65:8443/realms/uisce';

export const E2E_TENANT_ID =
  process.env.E2E_TENANT_ID ?? '99e99e99-99e9-49e9-89e9-99e99e99e999';

interface KcTokenCache {
  token: string;
  exp: number;
}

let _cachedKcToken: KcTokenCache | null = null;

export async function getKeycloakToken(): Promise<string> {
  if (_cachedKcToken && Date.now() < _cachedKcToken.exp - 60_000) {
    return _cachedKcToken.token;
  }
  if (!KC_USER || !KC_PASS) {
    throw new Error(
      'E2E_KC_USER or E2E_KC_PASS not set; ' +
        'set E2E_JWT for CI or E2E_KC_USER+E2E_KC_PASS for local',
    );
  }
  const res = await fetch(`${KC_ISSUER}/protocol/openid-connect/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'password',
      username: KC_USER,
      password: KC_PASS,
      client_id: 'semlayer-frontend',
      scope: 'openid profile email',
    }),
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Keycloak token fetch failed: ${res.status} ${body}`);
  }
  const data = (await res.json()) as { access_token: string; expires_in?: number };
  if (!data.access_token) {
    throw new Error('No access_token in Keycloak response');
  }
  const ttl = (data.expires_in ?? 300) * 1000;
  _cachedKcToken = { token: data.access_token, exp: Date.now() + ttl };
  return _cachedKcToken.token;
}

export const FALLBACK_JWT =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJpYXQiOjE3MDAwMDAwMDAsImV4cCI6OTk5OTk5OTk5OX0.fake';

export const E2E_USER = {
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

export const EXPIRES_AT = Date.now() + 86400 * 1000;

/**
 * Build the localStorage object the app expects for auth state.
 * Seeds: auth_token, auth_user, auth_expires_at, selected_tenant, appLocale
 */
export function buildAuthStorage(jwt: string) {
  return {
    auth_token: jwt,
    auth_user: JSON.stringify(E2E_USER),
    auth_expires_at: EXPIRES_AT.toString(),
    selected_tenant: JSON.stringify({
      id: E2E_TENANT_ID,
      display_name: 'Northwind Traders',
    }),
    appLocale: 'en',
  };
}

/**
 * Resolve the JWT for use in HTTP headers.
 * Tries: E2E_JWT env var → Keycloak password grant → throws.
 */
export async function resolveJwt(): Promise<string> {
  if (E2E_JWT) return E2E_JWT;
  if (KC_USER) return getKeycloakToken();
  throw new Error(
    'Neither E2E_JWT nor E2E_KC_USER is set. ' +
      'Set E2E_JWT (CI) or E2E_KC_USER+E2E_KC_PASS (local).',
  );
}
