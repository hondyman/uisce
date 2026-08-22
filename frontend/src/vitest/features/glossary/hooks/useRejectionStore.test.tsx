import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react/pure';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../../../contexts/TenantContext', () => ({
  useTenant: () => ({ tenant: { id: 'tenant-1' } }),
}));

const apiCalls: Array<{ url: string; init?: RequestInit }> = [];
const rejectionStore: any[] = [];

vi.mock('../../../../utils/apiClient', () => ({
  default: vi.fn(async (url: string, init?: RequestInit) => {
    apiCalls.push({ url, init });
    if (url.includes('/api/semantic-mapper/rejections') && (!init || init.method === undefined || init.method === 'GET')) {
      return { data: rejectionStore };
    }
    if (url.includes('/api/semantic-mapper/rejections') && init?.method === 'POST') {
      const body = JSON.parse(init.body as string);
      const record = {
        rejection_id: `rej-${rejectionStore.length + 1}`,
        source_node_id: body.source_node_id,
        rejected_target_id: body.rejected_target_id,
        edge_type_id: body.edge_type_id,
        reason: body.reason ?? '',
      };
      rejectionStore.push(record);
      return { status: 'recorded' };
    }
    return null;
  }),
}));

import { useRejectionStore } from '@/features/glossary/hooks/useRejectionStore';

function makeWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
  return wrapper;
}

describe('useRejectionStore', () => {
  beforeEach(() => {
    apiCalls.length = 0;
    rejectionStore.length = 0;
  });

  it('lists existing rejections', async () => {
    rejectionStore.push(
      { rejection_id: '1', source_node_id: 'A', rejected_target_id: 'B', edge_type_id: 'MAPS_TO' },
      { rejection_id: '2', source_node_id: 'A', rejected_target_id: 'C', edge_type_id: 'IS_SPECIALIZATION_OF' },
    );
    const { result } = renderHook(() => useRejectionStore(), { wrapper: makeWrapper() });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.rejections).toHaveLength(2);
    expect(apiCalls.some((c) => c.url === '/api/semantic-mapper/rejections')).toBe(true);
  });

  it('isRejected returns true for exact (source, target, edgeType) triplet', async () => {
    rejectionStore.push(
      { rejection_id: '1', source_node_id: 'A', rejected_target_id: 'B', edge_type_id: 'MAPS_TO' },
    );
    const { result } = renderHook(() => useRejectionStore(), { wrapper: makeWrapper() });
    await waitFor(() => expect(result.current.rejections.length).toBeGreaterThan(0));
    expect(result.current.isRejected('A', 'B', 'MAPS_TO')).toBe(true);
    expect(result.current.isRejected('A', 'B', 'OTHER')).toBe(false);
    expect(result.current.isRejected('X', 'Y', 'MAPS_TO')).toBe(false);
  });

  it('isRejected without edgeType returns true for any rejection of the (source, target) pair', async () => {
    rejectionStore.push(
      { rejection_id: '1', source_node_id: 'A', rejected_target_id: 'B', edge_type_id: 'MAPS_TO' },
    );
    const { result } = renderHook(() => useRejectionStore(), { wrapper: makeWrapper() });
    await waitFor(() => expect(result.current.rejections.length).toBeGreaterThan(0));
    expect(result.current.isRejected('A', 'B')).toBe(true);
  });

  it('recordRejection POSTs to the rejections endpoint and refreshes the cache', async () => {
    const { result } = renderHook(() => useRejectionStore(), { wrapper: makeWrapper() });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await act(async () => {
      await result.current.recordRejection({
        sourceNodeId: 'src-1',
        targetNodeId: 'tgt-1',
        predicate: 'MAPS_TO',
        edgeTypeId: 'edge-type-uuid-1',
        reason: 'not relevant',
      });
    });
    // Wait until the invalidation-driven refetch has populated the cache.
    await waitFor(() => expect(result.current.rejections.length).toBe(1));
    expect(result.current.rejections[0]).toMatchObject({
      source_node_id: 'src-1',
      rejected_target_id: 'tgt-1',
      edge_type_id: 'edge-type-uuid-1',
    });
    // Verify the POST was issued with the correct body shape
    const postCall = apiCalls.find((c) => c.init?.method === 'POST');
    expect(postCall).toBeDefined();
    expect(JSON.parse(postCall!.init!.body as string)).toEqual({
      source_node_id: 'src-1',
      rejected_target_id: 'tgt-1',
      edge_type_id: 'edge-type-uuid-1',
      reason: 'not relevant',
    });
  });

  it('falls back to predicate as edgeTypeId when not provided', async () => {
    const { result } = renderHook(() => useRejectionStore(), { wrapper: makeWrapper() });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await act(async () => {
      await result.current.recordRejection({
        sourceNodeId: 's',
        targetNodeId: 't',
        predicate: 'IS_PEER_IDENTIFIER_OF',
      });
    });
    expect(result.current.rejections[0].edge_type_id).toBe('IS_PEER_IDENTIFIER_OF');
  });

  it('returns empty list when no tenant', async () => {
    vi.doMock('../../../../contexts/TenantContext', () => ({
      useTenant: () => ({ tenant: null }),
    }));
    const { result } = renderHook(() => useRejectionStore(), { wrapper: makeWrapper() });
    await new Promise((r) => setTimeout(r, 50));
    expect(result.current.rejections).toEqual([]);
    vi.doUnmock('../../../../contexts/TenantContext');
  });
});
