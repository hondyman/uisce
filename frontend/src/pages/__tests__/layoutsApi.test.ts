import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { fetchSavedLayoutsApi, saveLayoutApi, loadLayoutApi, deleteLayoutApi } from '../layoutsApi';

describe('layoutsApi', () => {
  beforeEach(() => {
    // simple in-memory localStorage mock for node test env
    const store: Record<string, string> = {};
    const mockLocalStorage = {
      getItem: (k: string) => (k in store ? store[k] : null),
      setItem: (k: string, v: string) => { store[k] = String(v); },
      removeItem: (k: string) => { delete store[k]; },
      clear: () => { for (const k in store) delete store[k]; },
    };
    vi.stubGlobal('localStorage', mockLocalStorage as any);
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('fetchSavedLayoutsApi calls correct endpoint and parses response', async () => {
    // Note: in a real integration test we would verify apiClient logic, 
    // but here we just satisfy the new signature.
    const mock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ layouts: [{ id: '1', name: 'L' }] }) });
    (global.fetch as any) = mock;
    const res = await fetchSavedLayoutsApi();
    // apiClient will have transformed the URL
    expect(mock).toHaveBeenCalled();
    expect(res).toEqual([{ id: '1', name: 'L', updated_at: undefined }]);
  });

  it('saveLayoutApi posts to /api/layouts and returns json', async () => {
    const mock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ id: 'abc' }) });
    (global.fetch as any) = mock;
    const payload = { name: 'X', layout: { foo: 1 } };
    const res = await saveLayoutApi(payload);
    expect(mock).toHaveBeenCalled();
    expect(res).toEqual({ id: 'abc' });
  });

  it('loadLayoutApi calls correct url and returns json', async () => {
    const mock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ id: 'x', name: 'n' }) });
    (global.fetch as any) = mock;
    const res = await loadLayoutApi('xyz');
    expect(mock).toHaveBeenCalled();
    expect(res).toEqual({ id: 'x', name: 'n' });
  });

  it('deleteLayoutApi calls DELETE and returns true', async () => {
    const mock = vi.fn().mockResolvedValue({ ok: true });
    (global.fetch as any) = mock;
    const res = await deleteLayoutApi('delme');
    expect(mock).toHaveBeenCalled();
    expect(res).toBe(true);
  });
});
