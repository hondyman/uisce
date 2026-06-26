import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { TestHarness } from './testHarness'

vi.mock('@/services/rulesApi', () => ({
  rulesApi: {
    overrideRule: vi.fn(),
    deleteRule: vi.fn(),
    fetchPromotionImpact: vi.fn()
  }
}))

import ValidationRuleEditor from '@/components/validation/ValidationRuleEditor'

describe('ValidationRuleEditor (skeleton)', () => {
  it('renders the validation rules heading', () => {
    const fetchMock = globalThis.fetch as unknown as vi.Mock
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify([]), { status: 200, headers: { 'Content-Type': 'application/json' } })
    )

    render(
      <TestHarness>
        <ValidationRuleEditor />
      </TestHarness>
    )

    expect(screen.getByText('Validation Rules')).toBeInTheDocument()
  })
})
