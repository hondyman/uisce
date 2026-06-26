import React from 'react'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { vi } from 'vitest'
import DbScanner from '../DbScanner'
import { TenantProvider } from '../../contexts/TenantContext'
import { ScopeProvider } from '../../contexts/ScopeContext'
import { SnackbarProvider } from 'notistack'
import { ConfirmProvider } from '../../components/ConfirmProvider'

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <TenantProvider>
      <ScopeProvider>
        <SnackbarProvider>
          <ConfirmProvider>
            {ui}
          </ConfirmProvider>
        </SnackbarProvider>
      </ScopeProvider>
    </TenantProvider>
  )
}

describe('DbScanner integration', () => {
  beforeEach(() => {
    // seed tenant scope
    localStorage.setItem('selected_tenant', JSON.stringify({ id: 't-1', display_name: 'Test Tenant' }))
    localStorage.setItem('selected_product', JSON.stringify({ id: 'p-1', alpha_product: { product_name: 'Test Product' } }))
    localStorage.setItem('selected_datasource', JSON.stringify({ id: 'ds1', source_name: 'Test Datasource' }))
  })

  afterEach(() => {
    (global.fetch as any) = undefined
    localStorage.removeItem('selected_tenant')
    localStorage.removeItem('selected_product')
    localStorage.removeItem('selected_datasource')
    vi.resetAllMocks()
  })

  it('loads first page, filters via server q param, and appends more rows when Load more is clicked', async () => {
  // prepare paged responses: first call returns a full page (200 items) so hasMore=true,
  // second call returns 1 more item to be appended
  const columnsPage1 = Array.from({ length: 200 }).map((_, i) => ({ id: `c${i + 1}`, node_name: `col${i + 1}`, properties: { data_type: i % 2 === 0 ? 'int' : 'text' } }))
  // make one of the later items have the human-friendly name 'description' for filtering
  columnsPage1[2].node_name = 'id'
  columnsPage1[3].node_name = 'name'
  const columnsPage2 = [{ id: 'c201', node_name: 'description', properties: { data_type: 'text' } }]

    let call = 0
    global.fetch = vi.fn((input: RequestInfo) => {
      const url = String(input)
      if (url.includes('type=schema')) return Promise.resolve(new Response(JSON.stringify([{ id: 's1', node_name: 'public' }])))
      if (url.includes('type=table')) return Promise.resolve(new Response(JSON.stringify([{ id: 't1', node_name: 'products' }])))
      if (url.includes('type=column')) {
        call++
        if (call === 1) return Promise.resolve(new Response(JSON.stringify(columnsPage1)))
        if (call === 2) return Promise.resolve(new Response(JSON.stringify(columnsPage2)))
        return Promise.resolve(new Response(JSON.stringify([])))
      }
      if (url.includes('/api/catalog/scan')) return Promise.resolve(new Response(JSON.stringify({ message: 'scan ok', results: [{ tenant_instance_id: 'ds1', success: true }] })))
      return Promise.resolve(new Response('{}'))
    }) as any

    const rendered = renderWithProviders(<DbScanner />)

    // wait for schema
    await waitFor(() => expect(screen.getByText('public')).toBeInTheDocument())

  // expand and click the table via data-testids
  fireEvent.click(screen.getByTestId('schema-public'))
  await waitFor(() => expect(screen.getByTestId('table-products')).toBeInTheDocument())
  fireEvent.click(screen.getByTestId('table-products'))

    // some initial columns should be visible (tbody rows count >= 2)
    await waitFor(() => {
      const rows = rendered.container.querySelectorAll('tbody tr')
      expect(rows.length).toBeGreaterThanOrEqual(2)
    })

    // now click Load more
  const loadMore = await screen.findByTestId('load-more')
  fireEvent.click(loadMore)

    // new column appears (now rows >= 201)
    await waitFor(() => {
      const rows = rendered.container.querySelectorAll('tbody tr')
      expect(rows.length).toBeGreaterThanOrEqual(201)
    })

    // Now test server-side filtering: when typing 'desc' the fetch should be replaced so next call returns only 'description'
    global.fetch = vi.fn((input: RequestInfo) => {
      const url = String(input)
      if (url.includes('type=column') && url.includes('q=desc')) {
        return Promise.resolve(new Response(JSON.stringify([{ id: 'c3', node_name: 'description', properties: { data_type: 'text' } }])))
      }
      return Promise.resolve(new Response(JSON.stringify([])))
    }) as any

    const input = screen.getByPlaceholderText('Filter columns (name, type, props)') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'desc' } })

    // after filter, only description row should be visible (rows === 1)
    await waitFor(() => {
      const rows = rendered.container.querySelectorAll('tbody tr')
      expect(rows.length).toBeGreaterThanOrEqual(1)
      expect(screen.getByText('description')).toBeInTheDocument()
    })
  })
})
