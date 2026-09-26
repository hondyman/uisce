import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import apiClient from '../../utils/apiClient';
import {
  beginScopeRestore,
  markScopeRestored,
  readCachedSelection,
  resetScopeRestoreGate,
  whenTenantScopeReady,
} from '../../utils/tenantScope';

const PLACEHOLDER_DS = '11111111-1111-1111-1111-111111111111';
const TENANT = '99e99e99-99e9-49e9-89e9-99e99e99e999';
const REAL_DS = '441f62c9-aad1-481d-9aab-62943fa11cd3';

function sentHeaders(fetchMock: ReturnType<typeof vi.fn<any[], Promise<Response>>>, call = 0): Headers {
  return fetchMock.mock.calls[call][1].headers as Headers;
}

describe('tenant scope request gating', () => {
  let fetchMock: ReturnType<typeof vi.fn<any[], Promise<Response>>>;

  beforeEach(() => {
    localStorage.clear();
    resetScopeRestoreGate();
    fetchMock = vi.fn<any[], Promise<Response>>(async () => new Response('{"ok":true}', { status: 200, headers: { 'content-type': 'application/json' } }));
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
    resetScopeRestoreGate();
    localStorage.clear();
  });

  it('never sends the placeholder datasource id, and forgets it', async () => {
    localStorage.setItem('selected_datasource', JSON.stringify({ id: PLACEHOLDER_DS, source_name: 'dev' }));
    localStorage.setItem('operating_scope', JSON.stringify({ tenantId: TENANT, datasourceId: REAL_DS }));

    expect(readCachedSelection().datasource?.id).toBe(REAL_DS);
    expect(localStorage.getItem('selected_datasource')).toBeNull();

    await apiClient('/api/schedules/');
    expect(sentHeaders(fetchMock).get('X-Tenant-Datasource-ID')).toBe(REAL_DS);
  });

  it('holds a scoped request until the Operating Scope is restored', async () => {
    beginScopeRestore(); // AccessProvider mounted, restore under way, nothing cached

    const pending = apiClient('/api/schedules/kinds');
    await Promise.resolve();
    await Promise.resolve();
    expect(fetchMock).not.toHaveBeenCalled();

    // AccessContext.setDatasourceScope writes the selection, then releases.
    localStorage.setItem('selected_tenant', JSON.stringify({ id: TENANT }));
    localStorage.setItem('selected_datasource', JSON.stringify({ id: REAL_DS }));
    markScopeRestored();
    await pending;

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(sentHeaders(fetchMock).get('X-Tenant-Datasource-ID')).toBe(REAL_DS);
  });

  it('does not hold the requests that restore the scope', async () => {
    beginScopeRestore();
    await apiClient('/api/tenants/accessible');
    await apiClient('/api/auth/refresh', { method: 'POST' });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('does not hold when a real datasource is already cached', async () => {
    beginScopeRestore();
    localStorage.setItem('selected_datasource', JSON.stringify({ id: REAL_DS }));
    await apiClient('/api/schedules/');
    expect(sentHeaders(fetchMock).get('X-Tenant-Datasource-ID')).toBe(REAL_DS);
  });

  it('gives up after the timeout and omits the header rather than guessing', async () => {
    vi.useFakeTimers();
    beginScopeRestore();
    const pending = apiClient('/api/schedules/');
    await vi.advanceTimersByTimeAsync(5000);
    await pending;
    expect(sentHeaders(fetchMock).has('X-Tenant-Datasource-ID')).toBe(false);
  });

  it('is a no-op without an AccessProvider restoring', async () => {
    await expect(whenTenantScopeReady('/api/schedules/')).resolves.toBeUndefined();
  });
});
