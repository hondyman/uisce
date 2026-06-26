import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TestHarness } from './testHarness'
import ValidationRuleEditor from '@/components/validation/ValidationRuleEditor'

describe('ValidationRuleEditor - create flow', () => {
  it('opens the create dialog when Add Rule is clicked', async () => {
    const fetchMock = globalThis.fetch as unknown as vi.Mock
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify([]), { status: 200, headers: { 'Content-Type': 'application/json' } })
    )

    render(
      <TestHarness>
        <ValidationRuleEditor />
      </TestHarness>
    )

    const addButton = await screen.findByRole('button', { name: /Add Rule/i })
    fireEvent.click(addButton)

    expect(await screen.findByText('Create New Rule')).toBeInTheDocument()
  })
})