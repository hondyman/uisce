export const E2E_JWT = process.env.E2E_JWT;
export const KC_USER = process.env.E2E_KC_USER;
export const KC_PASS = process.env.E2E_KC_PASS;
export const KC_ISSUER = process.env.E2E_KC_ISSUER ?? 'https://100.84.50.65:8443/realms/uisce';

interface KcTokenCache {
  token: string;
  exp: number;
}

let _cachedKcToken: KcTokenCache | null = null;

export async function getKeycloakToken(): Promise<string> {
  if (_cachedKcToken && Date.now() < _cachedKcToken.exp - 60_000) {
    return _cachedKcToken.token;
  }
  if (!KC_USER || !KC_PASS) throw new Error('E2E_KC_USER or E2E_KC_PASS not set');
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
    const body = await res.text();
    throw new Error(`Keycloak token fetch failed: ${res.status} ${body}`);
  }
  const data = await res.json() as { access_token: string; expires_in?: number };
  if (!data.access_token) throw new Error('No access_token in Keycloak response');
  const ttl = (data.expires_in ?? 300) * 1000;
  _cachedKcToken = { token: data.access_token, exp: Date.now() + ttl };
  return _cachedKcToken.token;
}

export const FALLBACK_JWT = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJpYXQiOjE3MDAwMDAwMDAsImV4cCI6OTk5OTk5OTk5OX0.fake';

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
