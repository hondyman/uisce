/* eslint-disable no-console */

import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { vi } from 'vitest'
import SemanticMapper from '../SemanticMapper'
import { TenantProvider } from '../../contexts/TenantContext'
import { ScopeProvider } from '../../contexts/ScopeContext'
import { devDebug } from '../../utils/devLogger';

beforeEach(() => {
  localStorage.setItem('selected_tenant', JSON.stringify({ id: 't1', display_name: 'T1' }))
  localStorage.setItem('selected_product', JSON.stringify({ id: 'p1', alpha_product: { product_name: 'P1' } }))
  localStorage.setItem('selected_datasource', JSON.stringify({ id: 'ds1', source_name: 'DS1' }))
  // stub a URL-aware fetch so other background requests (mappings, etc.) do not
  // consume the catalog/profiler responses meant for this test.
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === 'string' ? input : String((input as Request).url || input)
    // eslint-disable-next-line no-console
    devDebug('[mock fetch] ', url, init?.method || 'GET')
    // Schemas
    if (url.includes('/api/semantic-mappings')) {
      return { ok: true, json: async () => [] }
    }
    if (url.includes('/api/catalog/nodes') && url.includes('type=schema')) {
      // eslint-disable-next-line no-console
      devDebug('[mock fetch] returning schemas')
      return { ok: true, json: async () => [{ id: 's1', node_name: 'public' }] }
    }
    // Tables for schema s1
    if (url.includes('/api/catalog/nodes') && url.includes('type=table')) {
      // eslint-disable-next-line no-console
      devDebug('[mock fetch] returning tables for schema')
      return { ok: true, json: async () => [ { id: 't1', node_name: 'employees', parent_id: 's1', qualified_path: '/public/employees' }, { id: 't2', node_name: 'departments', parent_id: 's1', qualified_path: '/public/departments' } ] }
    }
    // Columns for t1
    if (url.includes('/api/catalog/nodes') && url.includes('type=column') && url.includes('parent_id=t1')) {
      // eslint-disable-next-line no-console
      devDebug('[mock fetch] returning columns for t1')
      return { ok: true, json: async () => [{ id: 'c1', node_name: 'id', parent_id: 't1', properties: { data_type: 'int', ordinal_position: 1 } }] }
    }
    // Columns for t2
    if (url.includes('/api/catalog/nodes') && url.includes('type=column') && url.includes('parent_id=t2')) {
      // eslint-disable-next-line no-console
      devDebug('[mock fetch] returning columns for t2')
      return { ok: true, json: async () => [{ id: 'c2', node_name: 'dept_id', parent_id: 't2', properties: { data_type: 'int', ordinal_position: 1 } }] }
    }
    // POST profile
    if (url.includes('/api/profiler/profile') && init?.method === 'POST') {
      return { ok: true, json: async () => ({ jobId: 'job-1' }) }
    }
    // GET results
    if (url.includes('/api/profiler/results')) {
      return { ok: true, json: async () => ({ profiles: [ { Schema: 'public', TableName: 'employees', ColumnName: 'id', DataType: 'int', Cardinality: 100 }, { Schema: 'public', TableName: 'departments', ColumnName: 'dept_id', DataType: 'int', Cardinality: 50 } ] }) }
    }
    // default: empty successful response
    return { ok: true, json: async () => ({}) }
  }))
})

afterEach(() => {
  localStorage.removeItem('selected_tenant')
  localStorage.removeItem('selected_product')
  localStorage.removeItem('selected_datasource')
  vi.restoreAllMocks()
})

test('scanner -> profiler multi-table flow shows snackbar and ColumnId mapping', async () => {
  // sequence: schemas, tables for schema, columns for table1, columns for table2, POST profile, GET results
  // Note: fetch is stubbed above to return based on URL; no sequential mocks here.

  render(
    <TenantProvider>
      <ScopeProvider>
        <SemanticMapper />
      </ScopeProvider>
    </TenantProvider>
  )

  // wait for schemas to be fetched and scanner to render
    // wait until the fetch mock was called for schemas (avoid races with other background fetches)
    await waitFor(() => {
      const calls = (fetch as any).mock.calls || []
      return calls.some((c: any) => String(c[0]).includes('type=schema'))
    })

  // click the scanner refresh icon to ensure schemas are loaded in the DOM
  const refreshBtns = await screen.findAllByTestId('RefreshIcon')
  // click the first one (scanner refresh)
  fireEvent.click(refreshBtns[0])

  // give the scanner a moment to render the schema list
  await waitFor(() => expect(screen.getByText('Database')).toBeInTheDocument())

  // Expand schema and wait for tables
  const schema = await screen.findByTestId('schema-public')
  fireEvent.click(schema)

  // Wait for table items then select both via checkboxes
  const table1Checkbox = await screen.findByLabelText('profile-select-employees')
  const table2Checkbox = await screen.findByLabelText('profile-select-departments')
  // click both
  fireEvent.click(table1Checkbox)
  fireEvent.click(table2Checkbox)

  // Click Run profile in the scanner area (it should switch to Profile tab)
  const runProfileBtn = await screen.findByTestId('run-profile')
  fireEvent.click(runProfileBtn)

  // Wait for snackbar to appear
  await waitFor(() => expect(screen.getByText(/Profiling started/)).toBeInTheDocument())

  // Switch to Profile tab should have been automatic; wait for profile summary
  await waitFor(() => expect(screen.getByText(/Profile Summary for/)).toBeInTheDocument())

  // Confirm profiled columns are visible (by name) and have cardinality chips
  const col1 = await screen.findByText('id')
  const col2 = await screen.findByText('dept_id')
  expect(col1).toBeInTheDocument()
  expect(col2).toBeInTheDocument()

  // Cardinality chips from profiler results
  const chip100 = await screen.findByText('100 unique')
  const chip50 = await screen.findByText('50 unique')
  expect(chip100).toBeInTheDocument()
  expect(chip50).toBeInTheDocument()
})