import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'

// Mock reactflow so CatalogGraph's inner ReactFlow renders as a plain div we
// can inspect — this is a unit test of ImprovedErdDiagram's node-mapping
// logic, not a real ReactFlow render.
vi.mock('reactflow', () => ({
  __esModule: true,
  default: ({ nodes }: any) => (
    <div data-testid="node-types">{nodes.map((n: any) => `${n.id}:${n.type}`).join(',')}</div>
  ),
  Background: () => null,
  BackgroundVariant: { Dots: 'dots' },
  MiniMap: () => null,
  Panel: ({ children }: any) => <div>{children}</div>,
  Handle: () => null,
  Position: { Top: 'top', Bottom: 'bottom', Left: 'left', Right: 'right' },
  ReactFlowProvider: ({ children }: any) => <>{children}</>,
  useNodesState: (initial: any[]) => {
    const [state, setState] = React.useState(initial);
    return [state, setState, () => {}];
  },
  useEdgesState: (initial: any[]) => {
    const [state, setState] = React.useState(initial);
    return [state, setState, () => {}];
  },
}))

vi.mock('../../../../pages/TabbedModal/ERD/ErdSidebar', () => ({ default: () => null }))
vi.mock('../../../../pages/TabbedModal/ERD/ErdMinimap', () => ({ default: () => null }))
vi.mock('../../../../pages/TabbedModal/ERD/ErdInfoPanel', () => ({ default: () => null }))

import ImprovedErdDiagram from '@/pages/TabbedModal/ERD/ImprovedErdDiagram'

describe('ImprovedErdDiagram', () => {
  it('preserves each node\'s own type instead of forcing every node to professionalTable', () => {
    const nodes = [
      { id: 'customers', type: 'table', position: { x: 0, y: 0 }, data: { label: 'customers', columns: [] } },
      { id: 'orders_view', type: 'view', position: { x: 0, y: 0 }, data: { label: 'orders_view', columns: [] } },
      { id: 'legacy', position: { x: 0, y: 0 }, data: { label: 'legacy', columns: [] } },
    ]

    render(
      <ImprovedErdDiagram
        nodes={nodes as any}
        edges={[]}
        nodeTypes={{}}
        showColumns={false}
        showMiniMap={false}
        highlightedItem={null}
        zoomLevel={1}
        onInit={() => {}}
        onNodeClick={() => {}}
        onEdgeClick={() => {}}
        onPaneClick={() => {}}
        onMoveEnd={() => {}}
        onZoomChange={() => {}}
        onToggleColumns={() => {}}
        onToggleMiniMap={() => {}}
        onFitView={() => {}}
      />
    )

    const rendered = screen.getByTestId('node-types').textContent || ''
    // 'view' must survive — previously every node was force-overwritten to
    // 'professionalTable', discarding this distinction.
    expect(rendered).toContain('orders_view:view')
    expect(rendered).toContain('customers:table')
    // A node with no explicit type still falls back to the table renderer.
    expect(rendered).toContain('legacy:table')
  })
})
