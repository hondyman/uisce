import React from 'react'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { vi } from 'vitest'
import DbScanner from '../DbScanner'
import { TenantProvider } from '../../contexts/TenantContext'
import { ScopeProvider } from '../../contexts/ScopeContext'
import { SnackbarProvider } from 'notistack'
import { ConfirmProvider } from '../../components/ConfirmProvider'

// simple helper to render with providers
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

describe('DbScanner', () => {
  beforeEach(() => {
    // seed tenant scope in localStorage so TenantProvider picks it up in tests
    localStorage.setItem('selected_tenant', JSON.stringify({ id: 't-1', display_name: 'Test Tenant' }))
    localStorage.setItem('selected_product', JSON.stringify({ id: 'p-1', alpha_product: { product_name: 'Test Product' } }))
    localStorage.setItem('selected_datasource', JSON.stringify({ id: 'ds1', source_name: 'Test Datasource' }))
    // mock fetch for schemas, tables, columns and scan
    global.fetch = vi.fn((input: RequestInfo) => {
      const url = String(input)
      if (url.includes('type=schema')) {
        return Promise.resolve(new Response(JSON.stringify([{ id: 's1', node_name: 'public' }])))
      }
      if (url.includes('type=table')) {
        return Promise.resolve(new Response(JSON.stringify([{ id: 't1', node_name: 'products' }])))
      }
      if (url.includes('type=column')) {
        return Promise.resolve(new Response(JSON.stringify([{ id: 'c1', node_name: 'id', properties: { data_type: 'int', is_nullable: false } } ])))
      }
      if (url.includes('/api/catalog/scan')) {
        return Promise.resolve(new Response(JSON.stringify({ message: 'scan ok', results: [{ tenant_instance_id: 'ds1', success: true }] })))
      }
      return Promise.resolve(new Response('{}'))
    }) as any
  })

  afterEach(() => {
    (global.fetch as any) = undefined
    localStorage.removeItem('selected_tenant')
    localStorage.removeItem('selected_product')
    localStorage.removeItem('selected_datasource')
    vi.resetAllMocks()
  })

  it('loads schemas and tables and allows applying selection to scope', async () => {
    const mockRefresh = vi.fn(() => Promise.resolve())
    renderWithProviders(<DbScanner refreshMappings={mockRefresh} />)

    // wait for schema to load
    await waitFor(() => expect(screen.getByText('public')).toBeInTheDocument())

    // expand schema by clicking its label
    fireEvent.click(screen.getByText('public'))

    // wait for table to appear
    await waitFor(() => expect(screen.getByText('products')).toBeInTheDocument())

    // click table to select
    fireEvent.click(screen.getByText('products'))

    // Apply to Semantic Mapper
    const applyButton = screen.getByRole('button', { name: /Apply to Semantic Mapper/i })
    expect(applyButton).toBeEnabled()
    fireEvent.click(applyButton)

    // refreshMappings should be called
    await waitFor(() => expect(mockRefresh).toHaveBeenCalled())
  })
})
