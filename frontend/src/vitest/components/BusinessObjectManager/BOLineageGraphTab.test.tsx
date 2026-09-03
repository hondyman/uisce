import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mirrors the pattern in src/vitest/components/graph/CatalogGraph.test.tsx —
// mock reactflow to avoid its heavy jsdom-unfriendly renderer, and mock
// CatalogGraph itself since this test is about BOLineageGraphTab's own data
// transform (condenseBoGraph), not the shared renderer (already covered by
// CatalogGraph's own tests).
vi.mock('@/components/graph/CatalogGraph', () => ({
  CatalogGraph: (props: any) => (
    <div>
      <div data-testid="graph-nodes">{JSON.stringify(props.graphData.nodes)}</div>
      <div data-testid="graph-edges">{JSON.stringify(props.graphData.edges)}</div>
    </div>
  ),
}));

vi.mock('@/components/BusinessObjectManager/NodeDetailDrawer', () => ({
  NodeDetailDrawer: () => null,
}));
vi.mock('@/components/BusinessObjectManager/GraphLegend', () => ({
  GraphLegend: () => null,
}));

import { BOLineageGraphTab } from '@/components/BusinessObjectManager/BOLineageGraphTab';

describe('BOLineageGraphTab', () => {
  beforeEach(() => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        nodes: [
          { id: 'BO:bo1', type: 'bo', label: 'Customer', data: { name: 'Customer', description: 'A customer' } },
          { id: 'BO:bo2', type: 'related_bo', label: 'Order', data: { name: 'Order' } },
          {
            id: 'term1',
            type: 'term',
            label: 'customer_id',
            data: { termName: 'customer_id', termType: 'dimension', subtypeName: 'Core' },
          },
        ],
        edges: [
          { id: 'e1', source: 'BO:bo1', target: 'BO:bo2', type: 'relates_to' },
          { id: 'e2', source: 'BO:bo1', target: 'term1', type: 'contains' },
        ],
      }),
    }) as any;
  });

  it('condenses BO/related_bo nodes, attaches terms, and drops non-BO edges', async () => {
    render(<BOLineageGraphTab boId="bo1" />);

    await waitFor(() => {
      expect(screen.getByTestId('graph-nodes')).toBeTruthy();
    });

    const nodes = JSON.parse(screen.getByTestId('graph-nodes').textContent!);
    const edges = JSON.parse(screen.getByTestId('graph-edges').textContent!);

    // Only bo/related_bo nodes survive condensing.
    expect(nodes.map((n: any) => n.id).sort()).toEqual(['BO:bo1', 'BO:bo2']);

    // The BO node carries its terms for BONode to group/render internally.
    const boNode = nodes.find((n: any) => n.id === 'BO:bo1');
    expect(boNode.properties.terms).toHaveLength(1);
    expect(boNode.properties.terms[0].nodeName).toBe('customer_id');
    expect(boNode.properties.termCount).toBe(1);

    // Only the BO<->BO edge survives; the BO->term edge is dropped (terms
    // render inside the BO node itself, not as separate graph edges).
    expect(edges).toHaveLength(1);
    expect(edges[0]).toMatchObject({ source: 'BO:bo1', target: 'BO:bo2' });
  });
});
