import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'

// Mocks: tenant context and rulesApi used by the component
vi.mock('@/contexts/TenantContext', () => ({
  useTenant: () => ({ tenant: { id: 'tenant-1' }, datasource: { id: 'ds-1' } })
}))

vi.mock('@/services/rulesApi', () => ({
  rulesApi: {
    fetchRuleDiff: vi.fn()
  }
}))

// Mock monaco to simulate 'unavailable' (no DiffEditor export)
vi.mock('@monaco-editor/react', () => ({}))

import { rulesApi } from '@/services/rulesApi'
import { RuleDiffViewer } from '@/components/rules/RuleDiffViewer'

describe('RuleDiffViewer', () => {
  it('renders the diff editor fallback when Monaco DiffEditor is unavailable', async () => {
    ;(rulesApi.fetchRuleDiff as unknown as vi.Mock).mockResolvedValue({
      base: {
        name: 'Base Rule',
        severity: 'low',
        scope: 'all',
        expression: 'a: 1'
      },
      current: {
        name: 'Current Rule',
        severity: 'high',
        scope: 'all',
        expression: 'a: 2'
      },
      diffs: [
        { field: 'expression', old: 'a: 1', new: 'a: 2' }
      ]
    })

    render(<RuleDiffViewer isOpen={true} onClose={() => {}} boId="bo-1" ruleId="r-1" />)

    // Wait for the fetched diff to be shown (and the Show DSL Diff button to render)
    await waitFor(() => expect(screen.getByText(/Show DSL Diff/i)).toBeInTheDocument())

    const user = userEvent.setup()
    await user.click(screen.getByText(/Show DSL Diff/i))

    // The lazy import will resolve to the fallback component because we mocked the module without a DiffEditor
    await waitFor(() => expect(screen.getByText(/Diff editor unavailable in this environment/i)).toBeInTheDocument())
  })
})