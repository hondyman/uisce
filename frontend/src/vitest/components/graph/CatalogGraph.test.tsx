import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';

// Mock reactflow to avoid the heavy renderer/DOM APIs it needs in jsdom,
// matching the established pattern in src/vitest/RuleLineageGraph.test.tsx.
vi.mock('reactflow', () => ({
  __esModule: true,
  default: ({ nodes, edges }: any) => (
    <div>
      <div data-testid="nodes">{nodes.length}</div>
      <div data-testid="edges">{edges.length}</div>
    </div>
  ),
  Background: () => null,
  BackgroundVariant: { Dots: 'dots' },
  Controls: () => null,
  Handle: () => null,
  Position: { Top: 'top', Bottom: 'bottom' },
  ReactFlowProvider: ({ children }: any) => <>{children}</>,
  useNodesState: (initial: any[]) => {
    const [state, setState] = React.useState(initial);
    return [state, setState, () => {}];
  },
  useEdgesState: (initial: any[]) => {
    const [state, setState] = React.useState(initial);
    return [state, setState, () => {}];
  },
}));

vi.mock('@/api/viewDefinitions', () => ({
  useViewDefinition: () => ({
    data: {
      id: 'view-1',
      view_key: 'erd',
      display_name: 'ERD',
      is_core: true,
      is_active: true,
      config: { layout: { algorithm: 'dagre', direction: 'LR' } },
    },
  }),
  useCatalogGraph: () => ({
    data: {
      nodes: [
        { id: 'n1', type: 'table', label: 'orders', properties: {} },
        { id: 'n2', type: 'column', label: 'id', properties: {} },
      ],
      edges: [{ id: 'n1-n2', source: 'n1', target: 'n2', type: 'belongs_to' }],
    },
    isLoading: false,
    error: null,
  }),
}));

import { CatalogGraph } from '@/components/graph/CatalogGraph';

describe('CatalogGraph', () => {
  it('fetches the normalized graph for a view + root node and renders it', async () => {
    render(<CatalogGraph viewDefinitionId="view-1" rootNodeId="n1" />);

    await waitFor(() => {
      expect(screen.getByTestId('nodes').textContent).toBe('2');
      expect(screen.getByTestId('edges').textContent).toBe('1');
    });
  });
});
