import '@testing-library/jest-dom';

// Seed tenant scope expected by many components
try {
  if (!localStorage.getItem('selected_tenant')) {
    localStorage.setItem('selected_tenant', JSON.stringify({ id: '00000000-0000-0000-0000-000000000000', display_name: 'Test Tenant' }));
  }
  if (!localStorage.getItem('selected_product')) {
    localStorage.setItem('selected_product', JSON.stringify({ id: '00000000-0000-0000-0000-000000000001', alpha_product: { product_name: 'Test' } }));
  }
  if (!localStorage.getItem('selected_datasource')) {
    localStorage.setItem('selected_datasource', JSON.stringify({ id: '00000000-0000-0000-0000-000000000002', source_name: 'TestDS' }));
  }
} catch (e) {
  // ignore in non-browser test environments
}

// Provide a basic fetch shim if not present (some modules reference window.fetch)
if (typeof (globalThis as any).fetch === 'undefined') {
  // Minimal fetch shim returning an empty successful response.
  // Tests that need richer behavior should mock fetch per-test.
  // @ts-ignore
  globalThis.fetch = async (_input: any, _init?: any) => {
    return {
      ok: true,
      status: 200,
      json: async () => ({}),
      text: async () => '',
      headers: {
        get: () => null,
      },
    };
  };
}

// Ensure jsdom has a base URL origin for relative URL handling in tests
if (typeof window !== 'undefined' && (!window.location || !window.location.origin)) {
  // @ts-ignore
  delete (window as any).location;
  // Provide a simple location with origin
  // @ts-ignore
  window.location = new URL('http://localhost');
}

// noop scrollTo used by some components
if (typeof window !== 'undefined' && typeof window.scrollTo === 'undefined') {
  // @ts-ignore
  window.scrollTo = () => {};
}
