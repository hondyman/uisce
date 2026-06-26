import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { TestHarness } from './testHarness'
import ValidationRuleEditor from '@/components/validation/ValidationRuleEditor'

describe('ValidationRuleEditor - empty state', () => {
  it('shows an empty state when no rules are returned', async () => {
    const fetchMock = globalThis.fetch as unknown as vi.Mock
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify([]), { status: 200, headers: { 'Content-Type': 'application/json' } })
    )

    render(
      <TestHarness>
        <ValidationRuleEditor />
      </TestHarness>
    )

    expect(await screen.findByText('No rules found')).toBeInTheDocument()
  })
})