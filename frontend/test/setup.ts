import '@testing-library/jest-dom'
import { vi } from 'vitest'

// Ensure a jsdom-like location is present so code using `new URL('/path', location.origin)` or
// `location.origin` resolves correctly during tests.
;(globalThis as any).location = (globalThis as any).location || {
  origin: process.env.VITE_TEST_BASE_URL || 'http://localhost:5173',
  href: process.env.VITE_TEST_BASE_URL || 'http://localhost:5173/',
  pathname: '/',
  search: '',
  hash: ''
}

// Install a robust fetch shim that always exists in the test environment. It resolves
// relative URLs (starting with '/') against VITE_BACKEND_TARGET or a sensible default
// so node's fetch implementation (undici) won't throw Invalid URL errors.
const BACKEND_TARGET = process.env.VITE_BACKEND_TARGET || 'http://localhost:29080'

const defaultFetch = async (input: any, init?: any) => {
  let url = typeof input === 'string' ? input : input?.url || ''
  if (typeof url === 'string' && url.startsWith('/')) {
    // Resolve relative paths against backend target by default so tests that call
    // fetch('/api/...') will hit an absolute URL instead of throwing.
    url = `${BACKEND_TARGET}${url}`
  }
  // Delegate to the real global fetch if available (node >=18 / undici).
  if ((globalThis as any).__realFetch) {
    return (globalThis as any).__realFetch(url, init)
  }
  // If no real fetch present, return a default empty JSON response so tests can proceed.
  return Promise.resolve(new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } }))
}

// Preserve any existing fetch implementation under __realFetch so individual tests can
// still spy/mock global.fetch if needed.
if (!(globalThis as any).__realFetch && (globalThis as any).fetch) {
  ;(globalThis as any).__realFetch = (globalThis as any).fetch
}

// Always install our wrapper so relative URLs and base resolution are handled consistently.
;(globalThis as any).fetch = vi.fn(defaultFetch)

export { vi }
