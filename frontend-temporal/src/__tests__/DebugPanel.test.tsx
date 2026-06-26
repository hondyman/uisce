import React from 'react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import DebugPanel from '../components/DebugPanel'

describe('DebugPanel', () => {
  let origFetch: any
  beforeEach(() => {
    origFetch = global.fetch
  })
  afterEach(() => {
    global.fetch = origFetch
    vi.resetAllMocks()
  })

  it('renders metrics and events from the endpoints', async () => {
    global.fetch = vi.fn((input: any) => {
      const url = typeof input === 'string' ? input : input.url
      if (url.includes('/api/_debug/amqp-metrics')) {
        return Promise.resolve({ ok: true, json: async () => ({ amqp: 'ok' }), headers: { get: () => 'application/json' } })
      }
      if (url.includes('/api/v1/triggers/events')) {
        return Promise.resolve({ ok: true, json: async () => ([{ type: 'test.event', payload: { k: 'v' } }]) })
      }
      if (url.includes('/api/_debug/publish-event')) {
        return Promise.resolve({ ok: true, text: async () => 'published' })
      }
      return Promise.resolve({ ok: false })
    })

    render(<DebugPanel />)

    await waitFor(() => expect(screen.getByText(/AMQP Metrics/i)).toBeTruthy())
    await waitFor(() => expect(screen.getByText(/test.event/)).toBeTruthy())
  })
})
