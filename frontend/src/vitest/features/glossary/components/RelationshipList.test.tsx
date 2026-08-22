import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('@/api/glossary', async () => {
  const actual = await vi.importActual<any>('@/api/glossary');
  return {
    ...actual,
    useDeleteTermEdge: () => ({ mutateAsync: vi.fn().mockResolvedValue(undefined) }),
  };
});

import { RelationshipList } from '@/features/glossary/components/RelationshipList';
import type { CatalogEdge } from '@/api/glossary';
import { getPredicate } from '@/features/glossary/constants/predicates';

function makeEdge(overrides: Partial<CatalogEdge> = {}): CatalogEdge {
  return {
    id: 'edge-test',
    subject_node_id: 'a',
    object_node_id: 'b',
    object_node_type_id: 'obj-type',
    edge_type_name: 'IRRELEVANT',
    relationship_type: 'irrelevant',
    properties: {},
    is_active: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    tenant_id: 'test-tenant',
    core_id: null,
    ...overrides,
  } as CatalogEdge;
}

function makeEdges(): CatalogEdge[] {
  return [
    makeEdge({ id: 'e1', subject_node_id: 'a', object_node_id: 'b', edge_type_name: 'IS_SPECIALIZATION_OF' }),
    makeEdge({ id: 'e2', subject_node_id: 'a', object_node_id: 'b', edge_type_name: 'MAPS_TO' }),
    makeEdge({ id: 'e3', subject_node_id: 'a', object_node_id: 'c', edge_type_name: 'BO_RELATIONSHIP', relationship_type: 'BO_RELATIONSHIP' }),
  ];
}

function renderList(edges: CatalogEdge[]) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <RelationshipList
        edges={edges}
        nodes={[
          { id: 'a', node_name: 'A', catalog_type_name: 'business_term' },
          { id: 'b', node_name: 'B', catalog_type_name: 'semantic_term' },
          { id: 'c', node_name: 'C', catalog_type_name: 'business_object' },
        ]}
        selectedNodeId="a"
        darkMode={true}
      />
    </QueryClientProvider>
  );
}

describe('RelationshipList — predicate resolution', () => {
  it('renders the specialisation badge with the registry color/icon', () => {
    renderList(makeEdges());
    // Badge contains both the icon and label concatenated. Use a regex that matches both.
    expect(screen.getByText(/🔻 Specialization/i)).toBeInTheDocument();
  });

  it('renders the maps_to badge from the registry', () => {
    renderList(makeEdges());
    expect(screen.getByText(/🧠 Maps to/i)).toBeInTheDocument();
  });

  it('falls back to the registry icon when edge_type_name is missing but predicate is set', () => {
    renderList(makeEdges());
    expect(screen.getByText(/🏢 BO relationship/i)).toBeInTheDocument();
  });

  it('falls back to RELATED_TO when all aliases are missing', () => {
    const edges = [
      makeEdge({ id: 'fallback', subject_node_id: 'a', object_node_id: 'b', edge_type_name: undefined, relationship_type: undefined }),
    ];
    renderList(edges);
    expect(screen.getByText(/🔗 Relation/i)).toBeInTheDocument();
  });
});

describe('getPredicate registry', () => {
  it('returns FALLBACK_PREDICATE for unknown keys', () => {
    const meta = getPredicate('NOT_A_REAL_PREDICATE');
    expect(meta.key).toBe('NOT_A_REAL_PREDICATE');
    expect(meta.label).toBe('Relation');
  });

  it('returns the canonical metadata for known predicates', () => {
    const meta = getPredicate('IS_SPECIALIZATION_OF');
    expect(meta.icon).toBe('🔻');
    expect(meta.color).toBe('#6366F1');
    expect(meta.direction).toBe('outbound');
  });
});
