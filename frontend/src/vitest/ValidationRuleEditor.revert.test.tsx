import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TestHarness } from './testHarness'
import ValidationRuleEditor from '@/components/validation/ValidationRuleEditor'

describe('ValidationRuleEditor - edit flow', () => {
  it('opens edit dialog when Edit is clicked', async () => {
    const fetchMock = globalThis.fetch as unknown as vi.Mock
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify([
          {
            id: 'r1',
            name: 'Rule One',
            bp_name: 'bp1',
            step_name: 'step1',
            condition_json: '{}'
          }
        ]),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    )

    render(
      <TestHarness>
        <ValidationRuleEditor />
      </TestHarness>
    )

    expect(await screen.findByText('Rule One')).toBeInTheDocument()

    const editButtons = await screen.findAllByTitle('Edit')
    fireEvent.click(editButtons[0])

    expect(await screen.findByText('Edit Rule')).toBeInTheDocument()
  })
})