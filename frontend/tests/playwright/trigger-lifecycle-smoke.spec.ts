/**
 * Trigger lifecycle smoke test.
 *
 * Covers the full Tier-1 trigger path:
 *   1. Authenticate (E2E_JWT or Keycloak password grant)
 *   2. Seed localStorage with auth state
 *   3. POST a trigger via /api/v1/triggers
 *   4. GET the trigger list and assert it appears
 *   5. DB check: tenant_id is correct (cross-tenant isolation), is_active = true
 *   6. PUT to toggle is_active → false
 *   7. GET again and assert is_active flipped
 *   8. DB check: is_active = false (direct DB query, not API round-trip)
 *   9. PUT to toggle is_active → true (cleanup)
 *  10. DB check: is_active = true (direct DB query, not API round-trip)
 *
 * UI leg: there is no dedicated trigger authoring page in AppRoutes — the
 * trigger management surface is not yet implemented in the frontend. The UI
 * navigation step is deferred until the surface exists.
 *
 * Run:
 *   npx playwright test trigger-lifecycle-smoke.spec.ts \
 *     --config=playwright.smoke.config.ts \
 *     --reporter=list
 *
 * Env vars:
 *   E2E_JWT           — pre-minted JWT (CI path)
 *   E2E_KC_USER       — Keycloak user for password grant (local)
 *   E2E_KC_PASS       — Keycloak password
 *   E2E_KC_ISSUER     — optional (default: https://100.84.50.65:8443/realms/uisce)
 *   E2E_TENANT_ID     — optional (default: 99e99e99-99e9-49e9-89e9-99e99e99e999)
 *   SMOKE_BACKEND_URL — backend URL (default: http://localhost:8080)
 *   SMOKE_DB_URL      — psql DSN for DB assertions
 *                        (default: postgres://postgres@100.84.50.65:5432/alpha?sslmode=verify-full&sslcert=...&sslkey=...&sslrootcert=...)
 */

import { test, expect } from '@playwright/test';
import { execSync } from 'child_process';
import {
  resolveJwt,
  buildAuthStorage,
  E2E_TENANT_ID,
} from './auth-helper';

const BACKEND_URL = process.env.SMOKE_BACKEND_URL ?? 'http://localhost:8080';

const DEFAULT_DB_URL =
  'postgres://postgres@100.84.50.65:5432/alpha?' +
  'sslmode=verify-full' +
  '&sslcert=' + encodeURIComponent('/Users/eganpj/.uisce/certs/postgres-client.crt') +
  '&sslkey=' + encodeURIComponent('/Users/eganpj/.uisce/certs/postgres-client.key') +
  '&sslrootcert=' + encodeURIComponent('/Users/eganpj/.uisce/certs/ca.crt');

const DB_URL = process.env.SMOKE_DB_URL ?? DEFAULT_DB_URL;

/** Run a psql query and return the output string. */
function psql(query: string): string {
  return execSync(`psql "${DB_URL}" -t -c "${query.replace(/"/g, '\\"')}"`, {
    encoding: 'utf-8',
    stdio: ['pipe', 'pipe', 'pipe'],
  }).trim();
}

/** Find a trigger by its UUID in the list (handles base64-encoded IDs from API). */
function findTrigger(
  triggers: Record<string, unknown>[],
  triggerId: string,
): Record<string, unknown> | null {
  const b64Id = Buffer.from(triggerId).toString('base64');
  return triggers.find(t =>
    t['id'] === triggerId || t['id'] === b64Id,
  ) ?? null;
}

test.describe.serial('Trigger lifecycle smoke', () => {
  let jwt: string;
  let triggerId: string;

  test.beforeAll(async () => {
    jwt = await resolveJwt();
  });

  test.beforeEach(async ({ page }) => {
    await page.context().addInitScript(
      (auth) => {
        for (const [k, v] of Object.entries(auth)) {
          localStorage.setItem(k, v as string);
        }
      },
      buildAuthStorage(jwt),
    );
    await page.context().setExtraHTTPHeaders({ 'X-Tenant-ID': E2E_TENANT_ID });
  });

  test('pre-flight: /api/v1/triggers/types is reachable', async ({ request }) => {
    const res = await request.fetch(`${BACKEND_URL}/api/v1/triggers/types`);
    expect(res.status(), await res.text()).toBe(200);
    const types = (await res.json()) as { id: string; key: string }[];
    expect(types.length, 'at least one trigger type should exist').toBeGreaterThan(0);
  });

  test('pre-flight: /api/v1/triggers/executions is reachable (200 with no runs)', async ({ request }) => {
    const res = await request.fetch(`${BACKEND_URL}/api/v1/triggers/executions`, {
      headers: {
        Authorization: `Bearer ${jwt}`,
        'X-Tenant-ID': E2E_TENANT_ID,
      },
    });
    expect(res.status(), `executions: ${res.status()} ${await res.text()}`).toBe(200);
  });

  test('step 1: create trigger via POST /api/v1/triggers', async ({ request }) => {
    triggerId = await createTriggerViaRequest(request, jwt, E2E_TENANT_ID);
    expect(triggerId, 'create response must include id').toBeTruthy();
  });

  test('step 2: trigger appears in GET /api/v1/triggers list', async ({ request }) => {
    const triggers = await listTriggersViaRequest(request, jwt, E2E_TENANT_ID);
    const found = findTrigger(triggers, triggerId);
    expect(found, `trigger ${triggerId} not found in list`).not.toBeNull();
    expect(found?.['is_active'], 'newly created trigger should be active').toBe(true);
  });

  test('step 3: DB check — tenant_id correct and is_active=true after creation', () => {
    const row = psql(
      `SELECT tenant_id, is_active FROM validation_triggers WHERE id = '${triggerId}'`,
    );
    expect(row, `trigger ${triggerId} not found in DB`).toBeTruthy();
    const [dbTenantId, dbIsActive] = row.split('|').map((s: string) => s.trim());
    expect(dbTenantId?.toLowerCase(), `tenant_id should be ${E2E_TENANT_ID}, got: ${row}`).toBe(E2E_TENANT_ID.toLowerCase());
    expect(dbIsActive, `is_active should be true, got: ${row}`).toBe('t');
  });

  test('step 4: toggle is_active → false via PUT', async ({ request }) => {
    await toggleTriggerViaRequest(request, jwt, E2E_TENANT_ID, triggerId, false);
  });

  test('step 5: is_active is false in GET list', async ({ request }) => {
    const triggers = await listTriggersViaRequest(request, jwt, E2E_TENANT_ID);
    const found = findTrigger(triggers, triggerId);
    expect(found, `trigger ${triggerId} not found after toggle`).not.toBeNull();
    expect(found?.['is_active'], 'is_active should be false after toggle').toBe(false);
  });

  test('step 6: DB check — is_active=false directly in database', () => {
    const row = psql(
      `SELECT is_active FROM validation_triggers WHERE id = '${triggerId}'`,
    );
    expect(row, `trigger ${triggerId} not found in DB`).toBeTruthy();
    const [dbIsActive] = row.split('|').map((s: string) => s.trim());
    expect(dbIsActive, `is_active should be f after toggle, got: ${row}`).toBe('f');
  });

  test('step 7: toggle is_active → true (cleanup)', async ({ request }) => {
    await toggleTriggerViaRequest(request, jwt, E2E_TENANT_ID, triggerId, true);
  });

  test('step 8: is_active is true in GET list after re-activation', async ({ request }) => {
    const triggers = await listTriggersViaRequest(request, jwt, E2E_TENANT_ID);
    const found = findTrigger(triggers, triggerId);
    expect(found, `trigger ${triggerId} not found after re-activation`).not.toBeNull();
    expect(found?.['is_active'], 'is_active should be true after re-activation').toBe(true);
  });

  test('step 9: DB check — is_active=true directly in database after re-activation', () => {
    const row = psql(
      `SELECT is_active FROM validation_triggers WHERE id = '${triggerId}'`,
    );
    expect(row, `trigger ${triggerId} not found in DB`).toBeTruthy();
    const [dbIsActive] = row.split('|').map((s: string) => s.trim());
    expect(dbIsActive, `is_active should be t after re-activation, got: ${row}`).toBe('t');
  });
});

/** Request-based helpers */
async function createTriggerViaRequest(
  request: import('@playwright/test').APIRequestContext,
  jwt: string,
  tenantId: string,
  triggerType = 'row_insert',
  targetEntity = 'northwind.category',
): Promise<string> {
  const res = await request.fetch(`${BACKEND_URL}/api/v1/triggers`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${jwt}`,
      'X-Tenant-ID': tenantId,
      'Content-Type': 'application/json',
    },
    data: JSON.stringify({
      trigger_type: triggerType,
      target_entity: targetEntity,
      is_active: true,
    }),
  });
  expect(res.status(), `create trigger HTTP ${res.status()} body: ${await res.text()}`).toBe(201);
  const body = (await res.json()) as { id: string };
  return body.id;
}

async function listTriggersViaRequest(
  request: import('@playwright/test').APIRequestContext,
  jwt: string,
  tenantId: string,
): Promise<Record<string, unknown>[]> {
  const res = await request.fetch(`${BACKEND_URL}/api/v1/triggers`, {
    headers: {
      Authorization: `Bearer ${jwt}`,
      'X-Tenant-ID': tenantId,
    },
  });
  expect(res.status(), `list triggers HTTP ${res.status()} body: ${await res.text()}`).toBe(200);
  const body = (await res.json()) as Record<string, unknown>[] | null;
  return body ?? [];
}

async function toggleTriggerViaRequest(
  request: import('@playwright/test').APIRequestContext,
  jwt: string,
  tenantId: string,
  triggerId: string,
  isActive: boolean,
): Promise<void> {
  const b64Id = Buffer.from(triggerId).toString('base64');
  for (const id of [triggerId, b64Id]) {
    const res = await request.fetch(`${BACKEND_URL}/api/v1/triggers/${id}`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${jwt}`,
        'X-Tenant-ID': tenantId,
        'Content-Type': 'application/json',
      },
      data: JSON.stringify({ is_active: isActive }),
    });
    if (res.ok()) return;
  }
  throw new Error(`toggle failed for trigger ${triggerId}`);
}
