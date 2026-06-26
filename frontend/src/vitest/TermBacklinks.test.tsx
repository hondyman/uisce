import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'

vi.mock('@/services/rulesApi', () => ({
  rulesApi: {
    getRulesForTerm: vi.fn()
  }
}))

import { rulesApi } from '@/services/rulesApi'
import { TermBacklinks } from '@/components/terms/TermBacklinks'

describe('TermBacklinks', () => {
  it('renders rules returned by the API', async () => {
    (rulesApi.getRulesForTerm as any).mockResolvedValue([
      { id: '1', name: 'Rule A' },
      { id: '2', name: 'Rule B' }
    ])

    render(<TermBacklinks termId="term-123" />)

    expect(await screen.findByText('Rule A')).toBeInTheDocument()
    expect(await screen.findByText('Rule B')).toBeInTheDocument()
  })
})
