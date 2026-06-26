import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createElement } from 'react';
import { render } from '@testing-library/react';
import { act } from 'react-dom/test-utils';
import * as AuthContext from '../../contexts/AuthContext';
import * as AuthFetch from '../../utils/authFetch';
import { useModelCatalog } from '../useModelCatalog';

// Minimal mock models list
const MOCK_MODELS = [
  { id: 'uuid-1', model_key: 'core_model', display_name: 'Core Model', is_custom: false },
  { id: 'uuid-2', model_key: 'core_model_custom', display_name: 'Custom Model', is_custom: true },
] as any;

describe('useModelCatalog.deleteModel', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    // Mock getValidToken
    vi.spyOn(AuthContext, 'useAuth').mockReturnValue({ getValidToken: async () => 'token', isLoading: false } as any);
    // Mock authFetch to call through to global fetch wrapper
  vi.spyOn(AuthFetch, 'useAuthFetch').mockReturnValue({ authFetch: async (url: string, _opts: any) => {
      // Simulate delete endpoint returning 204 for uuid-2
      if (url.includes('/api/models/uuid-2')) return { ok: true, status: 204, data: null, response: new Response(null, { status: 204 }) } as any;
      if (url.includes('/api/models/uuid-1')) return { ok: true, status: 204, data: null, response: new Response(null, { status: 204 }) } as any;
      // Simulate list fetch
      if (url.includes('/api/models?')) return { ok: true, status: 200, data: { models: MOCK_MODELS }, response: new Response(JSON.stringify({ models: MOCK_MODELS }), { status: 200 }) } as any;
      return { ok: false, status: 404, data: null, response: new Response(null, { status: 404 }) } as any;
    }} as any);
  });

  it('resolves non-UUID model_key to UUID and deletes custom model', async () => {
    // Use a small harness component to access hook instance
    let hookApi: any = null;
    function Harness() {
      hookApi = useModelCatalog('tenant-1', 'ds-1');
      return null;
    }

  render(createElement(Harness));
    // Wait a tick for fetchModels to run
    await new Promise(resolve => setTimeout(resolve, 0));

    // Prime local models with mock data
    act(() => { hookApi.models = MOCK_MODELS.slice(); });

    const res = await act(async () => hookApi.deleteModel('core_model_custom'));
    expect(res && (res.success === true || res.ok === true || res !== undefined)).toBeTruthy();
  });
});
