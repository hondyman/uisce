import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { vi } from 'vitest'
import ProfilerPage from '../ProfilerPage'
import { TenantProvider } from '../../contexts/TenantContext'

beforeEach(() => {
  localStorage.setItem('selected_tenant', JSON.stringify({ id: 't1', display_name: 'T1' }))
  localStorage.setItem('selected_product', JSON.stringify({ id: 'p1', alpha_product: { product_name: 'P1' } }))
  localStorage.setItem('selected_datasource', JSON.stringify({ id: 'ds1', source_name: 'DS1' }))
  vi.stubGlobal('fetch', vi.fn())
})

afterEach(() => {
  localStorage.removeItem('selected_tenant')
  localStorage.removeItem('selected_product')
  localStorage.removeItem('selected_datasource')
  vi.restoreAllMocks()
})

test('runs profiler and shows results', async () => {
  // mock schemas
  (fetch as any)
    .mockResolvedValueOnce({ ok: true, json: async () => [{ id: 's1', node_name: 'public' }] }) // schemas
    .mockResolvedValueOnce({ ok: true, json: async () => [{ id: 't1', node_name: 'employees', parent_id: 's1', qualified_path: '/public/employees' }] }) // tables fetch (when selecting schema)
    .mockResolvedValueOnce({ ok: true, json: async () => [{ id: 'c1', node_name: 'id', parent_id: 't1', properties: { data_type: 'int', is_nullable: false, ordinal_position: 1 } }] }) // columns fetch (when selecting table)
    .mockResolvedValueOnce({ ok: true, json: async () => ({ jobId: 'job-123' }) }) // POST /api/profiler/profile
    .mockResolvedValueOnce({ ok: true, json: async () => ({ profiles: [ { Schema: 'public', TableName: 'employees', ColumnName: 'id', DataType: 'int', Cardinality: 100 } ] }) }) // GET /api/profiler/results

  render(
    <TenantProvider>
      <ProfilerPage />
    </TenantProvider>
  )

  // wait for schema select to populate
  await waitFor(() => expect(fetch).toHaveBeenCalled())

  const schemaSelect = await screen.findByTestId('profiler-schema-select')
  // choose the first schema by setting the native input value inside the MUI Select
  const schemaNativeInput = schemaSelect.querySelector('input')
  expect(schemaNativeInput).not.toBeNull()
  fireEvent.change(schemaNativeInput as Element, { target: { value: 'public' } })

  // wait for tables to be fetched and the table select to become enabled
  const tableSelect = await screen.findByTestId('profiler-table-select')
  await waitFor(() => expect(tableSelect).not.toHaveAttribute('aria-disabled', 'true'))
  const tableNativeInput = tableSelect.querySelector('input')
  expect(tableNativeInput).not.toBeNull()
  fireEvent.change(tableNativeInput as Element, { target: { value: 'employees' } })

  const runButton = await screen.findByTestId('profiler-run-button')
  fireEvent.click(runButton)

  // wait for results to be fetched and rendered
  await waitFor(() => expect(screen.getByText(/Profile Summary for public.employees/)).toBeInTheDocument())
  expect(screen.getByText('id')).toBeInTheDocument()
})
