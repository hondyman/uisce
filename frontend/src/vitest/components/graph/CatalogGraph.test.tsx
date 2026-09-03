import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'

// Mock reactflow to avoid the heavy renderer — capture what CatalogGraph hands it.
vi.mock('reactflow', () => ({
  __esModule: true,
  default: ({ nodes, edges, nodeTypes, children }: any) => (
    <div>
      <div data-testid="node-ids">{nodes.map((n: any) => n.id).join(',')}</div>
      <div data-testid="node-types">{nodes.map((n: any) => n.type).join(',')}</div>
      <div data-testid="edge-count">{edges.length}</div>
      <div data-testid="registered-types">{Object.keys(nodeTypes).sort().join(',')}</div>
      {children}
    </div>
  ),
  Background: () => null,
  BackgroundVariant: { Dots: 'dots' },
  MiniMap: () => <div data-testid="minimap" />,
  Panel: ({ children }: any) => <div data-testid="legend-panel">{children}</div>,
}))

import CatalogGraph from '@/components/graph/CatalogGraph'

describe('CatalogGraph', () => {
  it('passes pre-fetched nodes/edges through untouched with layout="none"', () => {
    const graphData = {
      nodes: [
        { id: 'a', type: 'table', position: { x: 0, y: 0 }, data: { label: 'A' } },
        { id: 'b', type: 'view', position: { x: 100, y: 0 }, data: { label: 'B' } },
      ],
      edges: [{ id: 'a-b', source: 'a', target: 'b' }],
    }

    render(<CatalogGraph mode="pre-fetched" graphData={graphData} layout="none" grouping={[]} />)

    expect(screen.getByTestId('node-ids').textContent).toBe('a,b')
    expect(screen.getByTestId('node-types').textContent).toBe('table,view')
    expect(screen.getByTestId('edge-count').textContent).toBe('1')
  })

  it('merges nodeTypeOverrides into the registered node types', () => {
    const CustomNode = () => <div>custom</div>
    render(
      <CatalogGraph
        mode="pre-fetched"
        graphData={{ nodes: [], edges: [] }}
        layout="none"
        grouping={[]}
        nodeTypeOverrides={{ table: CustomNode }}
      />
    )

    expect(screen.getByTestId('registered-types').textContent).toContain('table')
  })

  it('shows the minimap only when showMiniMap is true', () => {
    const { rerender } = render(
      <CatalogGraph mode="pre-fetched" graphData={{ nodes: [], edges: [] }} layout="none" grouping={[]} />
    )
    expect(screen.queryByTestId('minimap')).toBeNull()

    rerender(
      <CatalogGraph
        mode="pre-fetched"
        graphData={{ nodes: [], edges: [] }}
        layout="none"
        grouping={[]}
        showMiniMap
      />
    )
    expect(screen.queryByTestId('minimap')).not.toBeNull()
  })

  it('collapses a group into a cluster node once it exceeds the threshold', () => {
    const graphData = {
      nodes: [
        { id: '1', type: 'column', position: { x: 0, y: 0 }, data: {} },
        { id: '2', type: 'column', position: { x: 0, y: 0 }, data: {} },
        { id: '3', type: 'column', position: { x: 0, y: 0 }, data: {} },
      ],
      edges: [],
    }

    render(
      <CatalogGraph
        mode="pre-fetched"
        graphData={graphData}
        layout="none"
        grouping={[{ key: (n) => n.type || 'unknown', threshold: 2 }]}
      />
    )

    expect(screen.getByTestId('node-ids').textContent).toBe('cluster-column')
  })
})
