import { vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import CatalogNodeTypeahead from '../CatalogNodeTypeahead'

// Simple fetch mock helper
function mockFetch(response: any) {
  (global as any).fetch = vi.fn(() =>
    Promise.resolve({ ok: true, json: () => Promise.resolve(response) })
  )
}

describe('CatalogNodeTypeahead', () => {
  beforeEach(() => {
  vi.resetAllMocks()
  })

  it('sends parent_id when searching and renders options', async () => {
    const nodes = [
      { id: '1', node_name: 'public', qualified_path: '/public', catalog_type: 'schema' },
    ]
    mockFetch(nodes)

    render(
      <CatalogNodeTypeahead
        nodeType="schema"
        value={''}
        onChange={() => {}}
        parentId={'parent-123'}
        label="Schema"
        placeholder="Select schema..."
      />
    )

    // Open the autocomplete
    const input = screen.getByPlaceholderText('Select schema...') as HTMLInputElement
    await userEvent.click(input)

    await waitFor(() => {
      expect((global as any).fetch).toHaveBeenCalled()
      const args = (global as any).fetch.mock.calls[0][0]
      expect(args).toContain('parent_id=parent-123')
    })

    // Option should be visible
    await screen.findByText('public')
  })

  it('supports multiple selection and returns ids via onChange', async () => {
    const nodes = [
      { id: 'c1', node_name: 'id', qualified_path: '/public.table.id', catalog_type: 'column' },
      { id: 'c2', node_name: 'name', qualified_path: '/public.table.name', catalog_type: 'column' },
    ]
    mockFetch(nodes)

  const onChange = vi.fn()
    render(
      <CatalogNodeTypeahead
        nodeType="column"
        value={[]}
        onChange={onChange}
        multiple
        label="Columns"
        placeholder="Select columns..."
      />
    )

    const input = screen.getByPlaceholderText('Select columns...') as HTMLInputElement
    await userEvent.click(input)

    // wait for options and select first
    await screen.findByText('id')
    await userEvent.click(screen.getByText('id'))

    await waitFor(() => {
      // Expect onChange called with array of ids
      expect(onChange).toHaveBeenCalled()
      const val = onChange.mock.calls[0][0]
      expect(Array.isArray(val)).toBe(true)
      expect(val[0]).toBe('c1')
    })
  })
})
