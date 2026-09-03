import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';

// CatalogGraph itself is already covered by its own test; here we're
// verifying ImpactGraph's own logic (direction filtering + stats), so mock
// the shared renderer down to something inspectable.
vi.mock('@/components/graph/CatalogGraph', () => ({
  CatalogGraph: (props: any) => (
    <div data-testid="graph-node-ids">{props.graphData.nodes.map((n: any) => n.id).join(',')}</div>
  ),
}));

vi.mock('@/features/impact-analysis/api/impactApi', () => ({
  impactApi: {
    getLineageGraph: vi.fn().mockResolvedValue({
      nodes: [
        { id: 'root', type: 'business_object', label: 'Root', properties: {} },
        { id: 'up1', type: 'table', label: 'Upstream 1', properties: { metadata: { direction: 'upstream' } } },
        { id: 'down1', type: 'column', label: 'Downstream 1', properties: { metadata: { direction: 'downstream' } } },
      ],
      edges: [
        { id: 'e1', source: 'up1', target: 'root', type: 'depends_on', properties: {} },
        { id: 'e2', source: 'root', target: 'down1', type: 'depends_on', properties: {} },
      ],
    }),
    getGraph: vi.fn(),
  },
}));

import { ImpactGraph } from '@/features/impact-analysis/components/ImpactGraph';

describe('ImpactGraph', () => {
  it('reports upstream/downstream stats over the full graph regardless of direction filter', async () => {
    const onStatsUpdate = vi.fn();
    render(
      <ImpactGraph
        nodeType="business_object"
        nodeId="root"
        directionMode="upstream"
        useLineageAPI
        onStatsUpdate={onStatsUpdate}
      />
    );

    await waitFor(() => {
      expect(onStatsUpdate).toHaveBeenCalledWith({ upstreamCount: 1, downstreamCount: 1, totalCount: 2 });
    });
  });

  it('filters rendered nodes by directionMode while keeping the root', async () => {
    render(<ImpactGraph nodeType="business_object" nodeId="root" directionMode="upstream" useLineageAPI />);

    await waitFor(() => {
      const ids = screen.getByTestId('graph-node-ids').textContent!.split(',').sort();
      // 'downstream'-tagged node is filtered out under directionMode="upstream"; root always stays.
      expect(ids).toEqual(['root', 'up1']);
    });
  });
});
