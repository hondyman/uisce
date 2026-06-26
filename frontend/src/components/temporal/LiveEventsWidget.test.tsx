import { render, screen, waitFor } from '@testing-library/react'
import { vi } from 'vitest'

vi.mock('../../services/temporalService', () => ({
  default: {
    fetchTriggerEvents: vi.fn(),
  }
}))

import temporalService from '../../services/temporalService'
import LiveEventsWidget from './LiveEventsWidget'

describe('LiveEventsWidget', () => {
  it('renders events from temporalService', async () => {
    (temporalService.fetchTriggerEvents as any).mockResolvedValueOnce([
      { type: 'test.event', id: 'e1' },
      { type: 'other.event', id: 'e2' },
    ])

    render(<LiveEventsWidget />)

    await waitFor(() => expect(screen.getByText('test.event')).toBeInTheDocument())
    expect(screen.getByText('other.event')).toBeInTheDocument()
  })
})
