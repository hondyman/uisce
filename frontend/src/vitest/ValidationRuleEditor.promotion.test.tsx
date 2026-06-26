import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TestHarness } from './testHarness'
import ValidationRuleEditor from '@/components/validation/ValidationRuleEditor'

describe('ValidationRuleEditor - templates tab', () => {
  it('shows template section when creating a new rule', async () => {
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

    expect(await screen.findByText(/Start from Template/i)).toBeInTheDocument()
  })
})