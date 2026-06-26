import { vi, describe, it, expect, beforeEach } from 'vitest'

vi.mock('../api', () => ({
  fetchAPI: vi.fn(),
}))

import { listDomains, searchDomains } from './domains'
import { fetchAPI } from '../api'

beforeEach(() => {
  (fetchAPI as unknown as ReturnType<typeof vi.fn>).mockReset()
})

describe('domains API helpers', () => {
  it('calls fetchAPI with /data-domains for listDomains', async () => {
    ;(fetchAPI as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce([])
    await listDomains()
    expect((fetchAPI as unknown as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe('/data-domains')
  })

  it('calls search with encoded query on /data-domains/search', async () => {
    ;(fetchAPI as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce([])
    await searchDomains("my query")
    expect((fetchAPI as unknown as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe('/data-domains/search?q=my%20query')
  })
})
