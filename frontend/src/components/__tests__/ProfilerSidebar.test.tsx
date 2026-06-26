import { render, screen, waitFor as _waitFor, fireEvent } from '@testing-library/react'
import { vi } from 'vitest'
import ProfilerPage from '../ProfilerPage'
import * as TenantHook from '../../contexts/TenantContext'

describe('Profiler sidebar', () => {
  beforeEach(() => {
  // mock tenant hook
  const tenantSpy: any = vi.spyOn(TenantHook as any, 'useTenant');
  tenantSpy.mockReturnValue({ tenant: { id: 't1' } as any, datasource: { id: 'd1' } as any, product: null, setSelection: () => {}, clearSelection: () => {}, isSelected: true } as any)
    // reset fetch mock
    (global as any).fetch = vi.fn()
  })

  afterEach(() => {
    vi.resetAllMocks()
    vi.restoreAllMocks()
  })

  it('renders schemas and allows refresh', async () => {
  const schemas = [ { id: 's1', node_name: 'public', qualified_path: '/public' } ];
    (global as any).fetch = vi.fn((url: string) => {
      if (String(url).includes('type=schema')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(schemas) } as any)
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) } as any)
    })

    render(<ProfilerPage />)

    // Click refresh to trigger fetchSchemas
    const refresh = await screen.findByLabelText('refresh-schemas')
    fireEvent.click(refresh)

  // Schema should appear after fetch resolves
  expect(await screen.findByText('public')).toBeInTheDocument()
  })
})
