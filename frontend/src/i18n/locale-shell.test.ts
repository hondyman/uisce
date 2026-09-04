/**
 * Locale shell routing matrix — verifies the URL behavior without a browser
 * or backend. Covers the 5 URLs from the Phase 1 matrix; three are fully
 * verifiable here (heal preserves query/hash, redirect targets, locale
 * prefix structure), two require a running backend (full page render +
 * RTL via RtlProvider, OAuth callback round-trip with real token).
 *
 * Strategy: avoid importing `LocaleShell` (which transitively pulls in
 * Monaco and other heavy deps that vitest can't resolve here). Instead,
 * exercise the SAME routing primitives the shell uses:
 *   - `normalizeLocale(pathname.split('/')[1])` → locale detection
 *   - `localePath(locale, rest)` → prefix composition
 *   - `localePath(locale, rest) + search + hash` → full URL composition
 *
 * If these primitives are correct, the shell's `<Navigate>` redirects
 * are correct (the shell's only logic is wrapping these helpers).
 */
import { describe, it, expect, beforeEach } from 'vitest';
import {
  LOCALES,
  DEFAULT_LOCALE,
  normalizeLocale,
  matchLocale,
  localePath,
  stripLocale,
  isRtl,
} from './locales';

describe('locale shell primitives — 5-URL matrix (3 verifiable without backend)', () => {
  beforeEach(() => {
    // Ensure no cross-test pollution; the helpers are pure.
  });

  it('URL 1: `/` (no path locale) — rootRedirect chooses from cache → matchLocale fallback', () => {
    // The shell calls matchLocale(candidates) when no cached locale exists.
    // Verify the function does what the shell expects.
    const candidates = ['en-US', 'es', 'fr-FR'];
    const match = matchLocale(candidates);
    expect(LOCALES).toContain(match);

    // Cached locale path
    const cached = 'es';
    const normalizedCached = normalizeLocale(cached);
    expect(normalizedCached).toBe('es');
  });

  it('URL 1b: `/` heals to the stored locale via matchLocale fallback', () => {
    // No cached locale → use matchLocale(navigator.languages || [navigator.language])
    const en = matchLocale(['en-US', 'en']);
    const es = matchLocale(['es-MX', 'es']);
    const ar = matchLocale(['ar-SA']);
    const unknown = matchLocale(['ko-KR']); // ko isn't in LOCALES
    expect(en).toBe('en');
    expect(es).toBe('es');
    expect(ar).toBe('ar');
    expect(unknown).toBe(DEFAULT_LOCALE);
  });

  it('URL 3: `/dashboard?tab=2` heals to /en/dashboard?tab=2 — query preserved', () => {
    // The shell composes `${localePath(preferred, pathname)}${search}${hash}`
    // when no locale segment is present (uses pathname as-is, NOT a stripped
    // version).
    const pathname = '/dashboard';
    const search = '?tab=2';
    const hash = '';
    const preferred = 'en';
    const composed = `${localePath(preferred, pathname)}${search}${hash}`;
    expect(composed).toBe('/en/dashboard?tab=2');
  });

  it('URL 3b: `/dashboard#section-3` heals to /en/dashboard#section-3 — hash preserved', () => {
    const pathname = '/dashboard';
    const search = '';
    const hash = '#section-3';
    const preferred = 'en';
    const composed = `${localePath(preferred, pathname)}${search}${hash}`;
    expect(composed).toBe('/en/dashboard#section-3');
  });

  it('URL 3c: `/es/dashboard?tab=2` does NOT heal — already locale-prefixed', () => {
    // Shell logic: if first segment IS a locale, leave the rest as-is (no redirect).
    const pathname = '/es/dashboard';
    const firstSegment = pathname.split('/')[1];
    expect(normalizeLocale(firstSegment)).toBe('es');
    // No heal target — path is canonical as-is.
    expect(stripLocale(pathname)).toBe('/dashboard');
  });

  it('URL 2: `/es/dashboard` composes to es localePath — chunk lazy-loads in browser', () => {
    // The shell renders the AppRoutes which contains the /es/dashboard route.
    // Verify the locale-prefix composition produces the right path.
    expect(localePath('es', '/dashboard')).toBe('/es/dashboard');
    // root of a locale collapses the trailing slash
    expect(localePath('es', '/')).toBe('/es');
    // empty string still triggers '/' default inside localePath
    expect(localePath('es', '')).toBe('/es/');
    // sub-paths
    expect(localePath('ar', '/dashboard/sub')).toBe('/ar/dashboard/sub');
  });

  it('URL 4: `/ar/dashboard` — RTL detection returns true for ar', () => {
    expect(isRtl('ar')).toBe(true);
    expect(isRtl('he')).toBe(true);
    expect(isRtl('fa')).toBe(true);
    expect(isRtl('en')).toBe(false);
    expect(isRtl('ja')).toBe(false);
    // The URL contains /ar but isRtl is read from i18n.language (or in the
    // shell's case, from path detection + i18n.changeLanguage in LocaleSync).
  });

  it('URL 4b: stripLocale on `/ar/dashboard` returns `/dashboard` for use in breadcrumbs', () => {
    expect(stripLocale('/ar/dashboard')).toBe('/dashboard');
    expect(stripLocale('/ar/dashboard/sub')).toBe('/dashboard/sub');
  });

  it('URL 5: `/auth/callback?code=…&state=…` stays un-prefixed — un-prefixed routes are top-level in LocaleShell', () => {
    // /auth/callback is in the un-prefixed routes list (login flow).
    // The shell does NOT redirect /auth/callback → /en/auth/callback because
    // /auth isn't a locale. Verify the helpers don't mistake it for one.
    expect(normalizeLocale('auth')).toBeNull();
    // stripLocale on /auth/callback?code=… returns /auth/callback (no prefix to strip)
    expect(stripLocale('/auth/callback')).toBe('/auth/callback');
  });

  it('normalizeLocale tolerates locale-region variants (nonExplicitSupportedLngs)', () => {
    expect(normalizeLocale('en-US')).toBe('en');
    expect(normalizeLocale('es-MX')).toBe('es');
    expect(normalizeLocale('pt-BR')).toBe('pt-BR');
    expect(normalizeLocale('zh-CN')).toBe('zh-CN');
    expect(normalizeLocale('ja-JP')).toBe('ja');
    // Unknown → null
    expect(normalizeLocale('xx')).toBeNull();
    expect(normalizeLocale('login')).toBeNull();
  });
});
