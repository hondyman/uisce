import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'

// Mock reactflow to avoid heavy renderer
vi.mock('reactflow', () => ({
  __esModule: true,
  default: ({ nodes, edges }: any) => (
    <div>
      <div data-testid="nodes">{nodes.length}</div>
      <div data-testid="edges">{edges.length}</div>
    </div>
  ),
  Background: () => null
}))

import { RuleLineageGraph } from '@/components/rules/RuleLineageGraph'

describe('RuleLineageGraph', () => {
  it('renders node and edge counts', () => {
    const lineage = {
      nodes: [
        { id: '1', type: 'rule', name: 'R1' },
        { id: '2', type: 'field', name: 'customer.name' }
      ],
      edges: [
        { from: '1', to: '2', type: 'uses_field' }
      ]
    }

    render(<RuleLineageGraph lineage={lineage} />)

    expect(screen.getByTestId('nodes').textContent).toBe('2')
    expect(screen.getByTestId('edges').textContent).toBe('1')
  })
})
