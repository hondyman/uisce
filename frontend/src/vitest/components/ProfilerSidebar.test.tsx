import { render, screen, fireEvent } from '@testing-library/react'
import { vi } from 'vitest'
import ProfilerPage from '../../components/ProfilerPage'

// EXPLICIT STUB: useTenant is stubbed, so this file is evidence about ProfilerPage's
// rendering only — not about the tenant context contract. See
// fixtures/tenantContextStub.ts for the fixture law and what this stub does NOT prove.
//
// This file was converted from a `vi.spyOn(TenantHook, 'useTenant')` mock. The spy had
// to be torn down in `afterEach` via `vi.restoreAllMocks()`, and it restored the real hook
// globally — a namespace spy is process-wide state, so a test that forgets the teardown
// silently changes every later test in the same worker. The `vi.mock` form is scoped to
// this module graph and cannot leak.
//
// vi.mock factories are hoisted above imports, so the fixture is pulled in dynamically
// rather than referenced as a binding.
vi.mock('../../contexts/TenantContext', async (importOriginal) => {
  const { applyTenantContextStub } = await import('../fixtures/tenantContextStub')
  // ProfilerPage reads `tenant` and `datasource` and renders schemas for them, so this
  // is a tenant-scoped path: the default stub (provider present, no tenant selected)
  // would not exercise what this test asserts. Overriding is explicit, not implicit.
  return applyTenantContextStub(importOriginal, {
    tenant: { id: 't1' } as any,
    datasource: { id: 'd1' } as any,
    isSelected: true,
  })
})

describe('Profiler sidebar', () => {
  beforeEach(() => {
    // reset fetch mock
    (global as any).fetch = vi.fn()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  it('renders schemas and allows refresh', async () => {
  const schemas = [ { id: 's1', node_name: 'public', qualified_path: '/public' } ];
    // The matcher keys on the URL the component actually requests. This test was
    // dormant outside the vitest include glob, so its old `type=schema` matcher was
    // never exercised and quietly bit-rotted: `fetchSchemas` calls
    // `api/rest/catalog-nodes?node_type_id=...`, which fell through to the `[]`
    // branch below, so no schema ever rendered. Verified by probe: with the old
    // matcher the original `vi.spyOn` version fails identically, so the fault is
    // pre-existing and not introduced by the mock rewrite.
    (global as any).fetch = vi.fn((url: string) => {
      if (String(url).includes('catalog-nodes')) {
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
