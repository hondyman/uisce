import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';

// Seed localStorage with a tenant selection so tenantScope helpers return a valid scope
const TENANT_KEY = 'selected_tenant';
const PRODUCT_KEY = 'selected_product';
const DATASOURCE_KEY = 'selected_datasource';


describe('setupTenantFetch proxying behavior (integration)', () => {
  let originalFetch: any;
  let fetchSpy: any;

  beforeEach(async () => {
    originalFetch = globalThis.fetch;
    // Provide a spyable fetch that just returns a simple Response
    fetchSpy = vi.fn(() => {
      // Return a minimal Response-like object for consumers that call .ok / .json
      const body = JSON.stringify({});
      return Promise.resolve(new Response(body, { status: 200, headers: { 'Content-Type': 'application/json' } }));
    });
    globalThis.fetch = fetchSpy;

    // Simulate dev mode and configured API base pointing at localhost
    try { (import.meta as any).env = { DEV: true, VITE_API_BASE_URL: 'http://localhost:9090' }; } catch (e) {}
    // Ensure window.location looks like a dev frontend
    // jsdom default origin is http://localhost
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: new URL('http://localhost:3000')
    });
    // Seed localStorage with a valid tenant selection
    localStorage.setItem(TENANT_KEY, JSON.stringify({ id: '00000000-0000-0000-0000-000000000000', display_name: 'Test Tenant' }));
    localStorage.setItem(PRODUCT_KEY, JSON.stringify({ id: 'prod-1', alpha_product: { product_name: 'Test Product' } }));
    localStorage.setItem(DATASOURCE_KEY, JSON.stringify({ id: '11111111-1111-1111-1111-111111111111', source_name: 'Test DS' }));
    // Now import the module which will patch window.fetch
    await import('../../../setupTenantFetch');
  });

  afterEach(() => {
    // restore
    globalThis.fetch = originalFetch;
    vi.resetModules();
    try { delete (import.meta as any).env; } catch (e) {}
  });

  it('rewrites localhost configured API requests to use frontend origin (proxy) and appends tenant params', async () => {
    // Call a relative API path which should be enforced and rewritten
    await window.fetch('/api/semantic-mappings');

    expect(fetchSpy).toHaveBeenCalled();
    const calledWith = fetchSpy.mock.calls[0][0] as string;
    // Should be absolute URL pointing at the frontend origin (http://localhost:3000)
    expect(calledWith.startsWith('http://localhost:3000')).toBe(true);
    // Should include tenant query params
    expect(calledWith).toContain('tenant_id=00000000-0000-0000-0000-000000000000');
    expect(calledWith).toContain('tenant_instance_id=11111111-1111-1111-1111-111111111111');
  });
});
