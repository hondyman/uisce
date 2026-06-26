import { useEffect } from 'react';
import { render } from '@testing-library/react';
import { vi, describe, it, expect } from 'vitest';
import { waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useUpdateNodeType, nodeTypesKeys } from '../nodeTypes';

// A small harness to call the hook imperatively
function Harness({ onDone }: { onDone: () => void }) {
  const mutation = useUpdateNodeType();
  useEffect(() => {
    (async () => {
      try {
        await mutation.mutateAsync({ id: 'node-1', tenantId: 'tenant-1', data: { catalog_type_name: 'x' } });
      } catch (e) {
        // swallow for test
      }
      onDone();
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return null;
}

describe('useUpdateNodeType', () => {
  it('invalidates list and detail queries on success (uses variables)', async () => {
    // Mock fetch to return a successful response with JSON body
    const mockResponse = { id: 'node-1', tenant_id: 'tenant-1' };
    const originalFetch = global.fetch;
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify(mockResponse),
    } as any);

    // Create a QueryClient and spy on invalidateQueries
    const qc = new QueryClient();
    qc.invalidateQueries = vi.fn();

    const onDone = vi.fn();

    render(
      <QueryClientProvider client={qc}>
        <Harness onDone={() => onDone()} />
      </QueryClientProvider>
    );

    // Wait for the harness to call onDone
    await waitFor(() => {
      expect(onDone).toHaveBeenCalled();
    });

    // Assert invalidateQueries was called for list and detail
    expect(qc.invalidateQueries).toHaveBeenCalled();
    // Find a call that had the list key
    const calledWithList = (qc.invalidateQueries as any).mock.calls.find((c: any[]) => {
      return c[0] && Array.isArray(c[0].queryKey) && c[0].queryKey.join('|').includes('list');
    });
    expect(calledWithList).toBeTruthy();

    const expectedListKey = nodeTypesKeys.list('tenant-1');
    const expectedDetailKey = nodeTypesKeys.detail('node-1', 'tenant-1');

    // Check that one of the calls matched the list key exactly
    const sawList = (qc.invalidateQueries as any).mock.calls.some((c: any[]) => {
      return JSON.stringify(c[0].queryKey) === JSON.stringify(expectedListKey);
    });
    const sawDetail = (qc.invalidateQueries as any).mock.calls.some((c: any[]) => {
      return JSON.stringify(c[0].queryKey) === JSON.stringify(expectedDetailKey);
    });

    expect(sawList).toBe(true);
    expect(sawDetail).toBe(true);

    // restore fetch
    global.fetch = originalFetch;
  });
});
