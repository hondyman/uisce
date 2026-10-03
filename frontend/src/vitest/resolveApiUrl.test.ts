import { describe, it, expect, vi, afterEach } from 'vitest';
import { resolveApiUrl } from '../utils/resolveApiUrl';

/**
 * Regression guard for "Failed to construct 'URL': Invalid URL" on /views.
 *
 * resolveApiUrl() returns a bare relative path ('/api/x') whenever the Vite
 * proxy path is in play — deliberately, so the dev proxy still sees /api/*.
 * But `new URL('/api/x')` throws, because the one-argument form has no base.
 * Every call site that builds a URL object therefore has to opt into
 * `absolute`; these tests pin both halves of that contract.
 *
 * Note on env: Vitest gives each module its own `import.meta.env` object, so
 * reassigning it from the test file does NOT reach the module under test (a
 * throwaway probe confirmed the module kept seeing the original values). The
 * proxy env is therefore the real, load-time one, and these tests exercise the
 * branches that are actually reachable in a dev checkout.
 */

// Read the real jsdom origin rather than hardcoding it: the absolute form is
// resolved against window.location.origin, which is the default jsdom value.
const ORIGIN = window.location.origin;

describe('resolveApiUrl', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('returns a relative path by default so the Vite proxy still intercepts /api', () => {
    // This is the dev/proxy path in a normal checkout.
    expect(resolveApiUrl('/api/views')).toBe('/api/views');
  });

  it('returns an absolute URL when asked, even on the relative dev path', () => {
    expect(resolveApiUrl('/api/views', true)).toBe(`${ORIGIN}/api/views`);
  });

  it('produces a value `new URL()` accepts — the crash this guards against', () => {
    // This exact call threw "Failed to construct 'URL': Invalid URL" before
    // the fix, and took the whole /views page down on mount.
    const u = new URL(resolveApiUrl('/api/views', true));
    u.searchParams.set('page', '1');
    u.searchParams.set('tenant_id', 't-1');
    expect(u.pathname).toBe('/api/views');
    expect(u.searchParams.get('page')).toBe('1');
    expect(u.searchParams.get('tenant_id')).toBe('t-1');
  });

  it('stays same-origin and /api-rooted, so the proxy still matches the path', () => {
    const resolved = resolveApiUrl('/api/views', true);
    const u = new URL(resolved);
    expect(u.origin).toBe(ORIGIN);
    expect(u.pathname.startsWith('/api/')).toBe(true);
  });

  it('passes absolute input URLs through untouched in both modes', () => {
    expect(resolveApiUrl('https://example.test/api/x')).toBe('https://example.test/api/x');
    expect(resolveApiUrl('https://example.test/api/x', true)).toBe('https://example.test/api/x');
  });

  it('handles a relative path with a query string', () => {
    const u = new URL(resolveApiUrl('/api/views?page=2', true));
    expect(u.pathname).toBe('/api/views');
    expect(u.searchParams.get('page')).toBe('2');
  });
});
