import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'

import BusinessObjectBindingWizard from '../../../components/BusinessObjectManager/BusinessObjectBindingWizard'
import type { WizardSemanticTerm, CatalogTableNode } from '../../../components/BusinessObjectManager/bindingWizard.types'

vi.mock('../../../contexts/TenantContext', () => ({
  useTenant: () => ({
    tenant: { id: 'tenant-1', name: 'Test Tenant' },
    datasource: {
      id: 'ds-1',
      name: 'Northwind',
      alpha_tenant_instance_id: 'ds-1',
    },
  }),
}))

vi.mock('../../../hooks/useNotification', () => ({
  useNotification: () => ({
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  }),
}))

const mockCreateBusinessObject = vi.fn()
const mockFetchCatalogTables = vi.fn()
const mockFetchSemanticTermsByTable = vi.fn()
const mockFetchAllSemanticTerms = vi.fn()
const mockFetchRelatedSemanticTerms = vi.fn()
const mockFetchCalculatedSemanticTerms = vi.fn()

vi.mock('../../../components/BusinessObjectManager/bindingWizard.service', async () => {
  const actual = await vi.importActual<typeof import('../../../components/BusinessObjectManager/bindingWizard.service')>(
    '../../../components/BusinessObjectManager/bindingWizard.service'
  )
  return {
    ...actual,
    createBusinessObject: (...args: any[]) => mockCreateBusinessObject(...args),
    fetchCatalogTables: (...args: any[]) => mockFetchCatalogTables(...args),
    fetchSemanticTermsByTable: (...args: any[]) => mockFetchSemanticTermsByTable(...args),
    fetchAllSemanticTerms: (...args: any[]) => mockFetchAllSemanticTerms(...args),
    fetchRelatedSemanticTerms: (...args: any[]) => mockFetchRelatedSemanticTerms(...args),
    fetchCalculatedSemanticTerms: (...args: any[]) => mockFetchCalculatedSemanticTerms(...args),
  }
})

describe('BusinessObjectBindingWizard', () => {
  const tables: CatalogTableNode[] = [
    { node_id: 'tbl-1', node_name: 'Customers', qualified_path: 'public.Customers' },
  ]

  const terms: WizardSemanticTerm[] = [
    {
      termNodeId: 'term-1',
      termKey: 'customer_name',
      termName: 'Customer Name',
      dataType: 'text',
      role: 'DIMENSION',
      eligibilitySource: 'DIRECT',
      mappings: [
        { columnNodeId: 'col-1', columnName: 'CompanyName', tableNodeId: 'tbl-1', tableName: 'Customers', isPrimarySource: true },
      ],
    },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
    mockFetchCatalogTables.mockResolvedValue(tables)
    mockFetchSemanticTermsByTable.mockResolvedValue(terms)
    mockFetchAllSemanticTerms.mockResolvedValue([])
    mockFetchRelatedSemanticTerms.mockResolvedValue([])
    mockFetchCalculatedSemanticTerms.mockResolvedValue([])
    mockCreateBusinessObject.mockResolvedValue({ id: 'bo-1' })
  })

  it('renders the wizard and creates a business object', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn()

    render(<BusinessObjectBindingWizard open onClose={vi.fn()} onSave={onSave} />)

    // Definition section
    await waitFor(() => expect(screen.getByPlaceholderText('e.g. Customer')).toBeInTheDocument())
    await user.type(screen.getByPlaceholderText('e.g. Customer'), 'Customer')

    // Select driving table
    const tableInput = screen.getByLabelText(/driving table/i)
    await user.click(tableInput)
    await waitFor(() => expect(screen.getByText('Customers')).toBeInTheDocument())
    await user.click(screen.getByText('Customers'))

    // Wait for direct terms and select Customer Name
    await waitFor(() => expect(screen.getByText('Customer Name')).toBeInTheDocument())
    await user.click(screen.getByText('Customer Name'))

    // Selected fields table should appear
    await waitFor(() => expect(screen.getByText(/selected fields/i)).toBeInTheDocument())
    expect(screen.getAllByText('Customer Name').length).toBeGreaterThanOrEqual(1)

    // Save draft
    await user.click(screen.getByRole('button', { name: /save draft/i }))

    await waitFor(() => {
      expect(mockCreateBusinessObject).toHaveBeenCalled()
    })

    expect(mockCreateBusinessObject).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Customer',
        driver_table_id: 'tbl-1',
        config: expect.objectContaining({
          fields: expect.arrayContaining([
            expect.objectContaining({
              semanticTermId: 'term-1',
              sourceColumnName: 'CompanyName',
              bindingStatus: 'RESOLVED',
            }),
          ]),
        }),
      })
    )
    expect(onSave).toHaveBeenCalledWith('bo-1')
  })

  it('shows manual terms when the manual tab is active', async () => {
    const user = userEvent.setup()
    const manualTerms: WizardSemanticTerm[] = [
      {
        termNodeId: 'term-2',
        termKey: 'custom_field',
        termName: 'Custom Field',
        eligibilitySource: 'MANUAL',
        mappings: [],
      },
    ]
    mockFetchAllSemanticTerms.mockResolvedValue(manualTerms)

    render(<BusinessObjectBindingWizard open onClose={vi.fn()} />)

    await waitFor(() => expect(screen.getByRole('tab', { name: /manual/i })).toBeInTheDocument())
    await user.click(screen.getByRole('tab', { name: /manual/i }))

    await waitFor(() => expect(screen.getByText('Custom Field')).toBeInTheDocument())
    await user.click(screen.getByText('Custom Field'))

    await waitFor(() => expect(screen.getAllByText('Custom Field').length).toBeGreaterThanOrEqual(1))
  })
})
