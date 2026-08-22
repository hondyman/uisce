import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react/pure';
import { waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// Mock TenantContext BEFORE importing the hook
vi.mock('../../../../contexts/TenantContext', () => ({
  useTenant: () => ({ tenant: { id: 'tenant-1' } }),
}));

// Capture the URL the hook calls
const apiCalls: Array<{ url: string; init?: RequestInit }> = [];
vi.mock('../../../../utils/apiClient', () => ({
  default: vi.fn(async (url: string, init?: RequestInit) => {
    apiCalls.push({ url, init });
    if (url.includes('/api/business-objects/')) {
      return {
        relatedObjects: [
          { relatedObjectName: 'Customer Order', relatedObjectId: 'bo-customer', relationshipType: 'HAS_ORDER', description: 'A customer has orders' },
        ],
        semanticFields: [
          { semanticTermName: 'Order ID', semanticTermId: 'sem-order', edgeTypeName: 'HAS_BUSINESS_TERM' },
        ],
        availableTerms: [],
      };
    }
    // node-graph endpoint
    return {
      node: { id: 'sem-1', node_name: 'Order ID' },
      edges: [
        {
          id: 'edge-1',
          subject_node_id: 'sem-1',
          object_node_id: 'bt-1',
          edge_type_name: 'MAPS_TO',
          relationship_type: 'maps_to',
          predicate: 'MAPS_TO',
        },
      ],
      connected_nodes: [
        { id: 'bt-1', node_name: 'Account Code' },
      ],
      nodes: [{ id: 'bt-1', node_name: 'Account Code' }],
    };
  }),
}));

import { useEntityRelationships } from '@/features/glossary/hooks/useEntityRelationships';

function makeWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
  return wrapper;
}

describe('useEntityRelationships', () => {
  beforeEach(() => {
    apiCalls.length = 0;
  });

  it('routes business_object to /api/business-objects/:id/relationships and normalizes', async () => {
    const { result } = renderHook(
      () => useEntityRelationships('business_object', 'bo-1'),
      { wrapper: makeWrapper() }
    );
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await waitFor(() => expect(result.current.data).toBeDefined());

    expect(apiCalls).toHaveLength(1);
    expect(apiCalls[0].url).toBe('/api/business-objects/bo-1/relationships');

    const data = result.current.data!;
    expect(data.source).toBe('business-object');
    // 2 synthetic edges: 1 from relatedObjects + 1 from semanticFields
    expect(data.edges).toHaveLength(2);
    // First edge: BO ↔ BO
    const boEdge: any = data.edges.find((e: any) => e.predicate === 'HAS_ORDER')!;
    expect(boEdge).toBeDefined();
    expect(boEdge.subject_node_id).toBe('bo-1');
    expect(boEdge.object_node_id).toBe('bo-customer');
    expect(boEdge.id).toContain('synthetic::bo-rel::bo-1::bo-customer::HAS_ORDER');
    expect(boEdge.description).toBe('A customer has orders');
    // Second edge: BO ↔ semantic term
    const semEdge: any = data.edges.find((e: any) => e.predicate === 'HAS_BUSINESS_TERM')!;
    expect(semEdge).toBeDefined();
    expect(semEdge.object_node_id).toBe('sem-order');
    expect(semEdge.id).toContain('synthetic::bo-sem::bo-1::sem-order::HAS_BUSINESS_TERM');
    // Focal BO is always in nodes
    expect(data.nodes.find((n: any) => n.id === 'bo-1')).toBeDefined();
  });

  it('routes business_term to /api/glossary/node-graph and does NOT normalize', async () => {
    const { result } = renderHook(
      () => useEntityRelationships('business_term', 'bt-1'),
      { wrapper: makeWrapper() }
    );
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await waitFor(() => expect(result.current.data).toBeDefined());

    expect(apiCalls).toHaveLength(1);
    expect(apiCalls[0].url).toContain('/api/glossary/node-graph');
    expect(apiCalls[0].url).toContain('node_id=bt-1');
    expect(apiCalls[0].url).toContain('tenant_id=tenant-1');

    const data = result.current.data!;
    expect(data.source).toBe('node-graph');
    expect(data.edges).toHaveLength(1);
    expect(data.edges[0].predicate).toBe('MAPS_TO');
    expect(data.edges[0].edge_type_name).toBe('MAPS_TO');
    expect(data.edges[0].subject_node_id).toBe('sem-1');
    expect(data.edges[0].object_node_id).toBe('bt-1');
  });

  it('routes semantic_term to /api/glossary/node-graph', async () => {
    const { result } = renderHook(
      () => useEntityRelationships('semantic_term', 'sem-1'),
      { wrapper: makeWrapper() }
    );
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(apiCalls[0].url).toContain('/api/glossary/node-graph');
  });

  it('does not fire when entityId is missing', async () => {
    const { result } = renderHook(
      () => useEntityRelationships('business_term', null),
      { wrapper: makeWrapper() }
    );
    await new Promise((r) => setTimeout(r, 50));
    expect(result.current.data).toBeUndefined();
    expect(apiCalls).toHaveLength(0);
  });

  it('skips synthetic edge generation when BO payload is empty', async () => {
    // Override the mock for this test only
    const apiModule = await import('../../../../utils/apiClient');
    (apiModule.default as any).mockImplementationOnce(async () => ({
      relatedObjects: [],
      semanticFields: [],
    }));
    const { result } = renderHook(
      () => useEntityRelationships('business_object', 'bo-empty'),
      { wrapper: makeWrapper() }
    );
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(result.current.data!.edges).toHaveLength(0);
    // Focal node is still in the list even with no relationships
    expect(result.current.data!.nodes).toHaveLength(1);
    expect(result.current.data!.nodes[0].id).toBe('bo-empty');
  });
});
