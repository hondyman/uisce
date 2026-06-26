import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest'
import { useABAC } from '../hooks/useABAC'

describe('useABAC.evaluate', () => {
  let origFetch: any
  beforeEach(() => {
    origFetch = global.fetch
  })
  afterEach(() => {
    global.fetch = origFetch
    vi.resetAllMocks()
  })

  it('returns true when backend replies {allowed: true}', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ allowed: true }),
      headers: { get: () => 'application/json' }
    })

    const { evaluate } = useABAC()
    const res = await evaluate('create', 'workflow')
    expect(res).toBe(true)
  })

  it('returns false when backend replies {allowed: false}', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ allowed: false }),
      headers: { get: () => 'application/json' }
    })

    const { evaluate } = useABAC()
    const res = await evaluate('delete', 'workflow')
    expect(res).toBe(false)
  })

  it('falls back to allow on localhost after retries', async () => {
    // simulate network error
    global.fetch = vi.fn().mockRejectedValue(new Error('network'))
    // ensure location hostname is localhost for fallback
    Object.defineProperty(window, 'location', {
      value: new URL('http://localhost'),
      writable: true
    })

    const { evaluate } = useABAC()
    const res = await evaluate('create', 'workflow')
    expect(res).toBe(true)
  })
})
