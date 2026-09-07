/**
 * Locale shell routing matrix. These tests exercise `resolveLocaleRoute`
 * — the single pure function the shell calls. If the shell's
 * `<Navigate to={route.to} />` rendering changes, that's a 1-line JSX
 * change with no logic. The actual redirect / canonical-render decision
 * lives in exactly one place (locales.ts) and is what we test.
 *
 * Coverage of the 5-URL matrix:
 *   URL 1: `/`                            — RootRedirect (separate function)
 *   URL 2: `/es/dashboard`                — kind: 'ok' branch
 *   URL 3: `/dashboard?tab=2`             — kind: 'redirect' no-locale (full path)
 *   URL 3b: `/dashboard#section-3`        — kind: 'redirect' no-locale (full path + hash)
 *   URL 4: `/ar/dashboard`                — kind: 'ok' branch + isRtl
 *   URL 5: `/auth/callback?code=…`         — kind: 'redirect' (auth isn't a locale)
 *
 * URLs 4 (RTL render) and 5 (OAuth round-trip) require a running backend;
 * they assert the route decision here; full MUI rendering happens in
 * the backend-up session.
 */
import { describe, it, expect } from 'vitest';
import {
  ACTIVE_LOCALES,
  DEFAULT_LOCALE,
  getPreferredLocale,
  isRtl,
  matchActiveLocale,
  resolveLocaleRoute,
} from './locales';

describe('resolveLocaleRoute — 5-URL matrix', () => {
  it('URL 2: `/es/dashboard` resolves to kind: "ok" with locale "es"', () => {
    const route = resolveLocaleRoute('/es/dashboard');
    expect(route.kind).toBe('ok');
    if (route.kind === 'ok') expect(route.locale).toBe('es');
  });

  it('URL 3: `/dashboard?tab=2` heals to /en/dashboard?tab=2 (full path preserved)', () => {
    const route = resolveLocaleRoute('/dashboard', '?tab=2', '', 'en');
    expect(route.kind).toBe('redirect');
    if (route.kind === 'redirect') {
      expect(route.to).toBe('/en/dashboard?tab=2');
    }
  });

  it('URL 3b: `/dashboard#section-3` heals to /en/dashboard#section-3 (hash preserved)', () => {
    const route = resolveLocaleRoute('/dashboard', '', '#section-3', 'en');
    expect(route.kind).toBe('redirect');
    if (route.kind === 'redirect') {
      expect(route.to).toBe('/en/dashboard#section-3');
    }
  });

  it('URL 3c: `/dashboard?tab=2#section-3` heals preserving both query AND hash', () => {
    const route = resolveLocaleRoute('/dashboard', '?tab=2', '#section-3', 'en');
    expect(route.kind).toBe('redirect');
    if (route.kind === 'redirect') {
      expect(route.to).toBe('/en/dashboard?tab=2#section-3');
    }
  });

  it('URL 4: `/ar/dashboard` resolves to kind: "ok" with locale "ar" (RTL eligible)', () => {
    const route = resolveLocaleRoute('/ar/dashboard');
    expect(route.kind).toBe('ok');
    if (route.kind === 'ok') expect(route.locale).toBe('ar');
    expect(isRtl('ar')).toBe(true);
  });

  it('URL 5: `/auth/callback?code=…` heals to /en/auth/callback?code=… (auth isn\'t a locale)', () => {
    const route = resolveLocaleRoute('/auth/callback', '?code=abc&state=xyz', '', 'en');
    expect(route.kind).toBe('redirect');
    if (route.kind === 'redirect') {
      expect(route.to).toBe('/en/auth/callback?code=abc&state=xyz');
    }
  });

  it('non-canonical prefix `/ES/dashboard` redirects to canonical `/es/dashboard`', () => {
    const route = resolveLocaleRoute('/ES/dashboard');
    expect(route.kind).toBe('redirect');
    if (route.kind === 'redirect') {
      // Strips the first segment, keeps the rest verbatim
      expect(route.to).toBe('/es/dashboard');
    }
  });

  it('non-canonical `/PT-BR/dashboard` redirects to canonical `/pt-BR/dashboard`', () => {
    const route = resolveLocaleRoute('/PT-BR/dashboard');
    expect(route.kind).toBe('redirect');
    if (route.kind === 'redirect') {
      expect(route.to).toBe('/pt-BR/dashboard');
    }
  });

  it('`/` (root) heals to /es (preferred locale, trailing slash collapsed)', () => {
    const route = resolveLocaleRoute('/', '', '', 'es');
    expect(route.kind).toBe('redirect');
    if (route.kind === 'redirect') expect(route.to).toBe('/es');
  });

  it('unknown first segment `/foo/dashboard` heals (foo isn\'t a locale)', () => {
    const route = resolveLocaleRoute('/foo/dashboard', '', '', 'en');
    expect(route.kind).toBe('redirect');
    if (route.kind === 'redirect') expect(route.to).toBe('/en/foo/dashboard');
  });
});

describe('matchActiveLocale — auto-detection restrictions', () => {
  it('skips skeleton locales (ja/ko/etc.) and falls back to DEFAULT_LOCALE', () => {
    // ja is in LOCALES (skeleton file exists) but not in ACTIVE_LOCALES
    expect(ACTIVE_LOCALES.includes('ja' as never)).toBe(false);
    // So an auto-detected ja-JP browser lands on /en/, not /ja/
    expect(matchActiveLocale(['ja-JP', 'ja'], ACTIVE_LOCALES)).toBe(DEFAULT_LOCALE);
  });

  it('returns a real translation when one matches', () => {
    expect(matchActiveLocale(['es-MX', 'es'], ACTIVE_LOCALES)).toBe('es');
    expect(matchActiveLocale(['fr-FR', 'fr'], ACTIVE_LOCALES)).toBe('fr');
    expect(matchActiveLocale(['en-US', 'en'], ACTIVE_LOCALES)).toBe('en');
  });

  it('falls through to DEFAULT_LOCALE when no candidate is in ACTIVE_LOCALES', () => {
    expect(matchActiveLocale(['de-DE', 'ko-KR'], ACTIVE_LOCALES)).toBe(DEFAULT_LOCALE);
  });
});

describe('getPreferredLocale — cache + candidates resolution', () => {
  it('honors ACTIVE_LOCALES cache hit', () => {
    expect(getPreferredLocale('es', ['en-US', 'en'])).toBe('es');
    expect(getPreferredLocale('fr', ['en-US', 'en'])).toBe('fr');
  });

  it('REJECTS skeleton-locale cache hit — the cache-shaped hole fix', () => {
    // An explicit /ja visit sets localStorage.appLocale='ja' via i18next.
    // Without this filter, every un-prefixed URL would heal to /ja/ with
    // skeleton English, bypassing the ACTIVE_LOCALES gate through the side
    // door. getPreferredLocale must filter the cache through ACTIVE too.
    expect(getPreferredLocale('ja', ['en-US', 'en'])).not.toBe('ja');
    expect(getPreferredLocale('ja', ['en-US', 'en'])).toBe(DEFAULT_LOCALE);
    expect(getPreferredLocale('de', ['en-US', 'en'])).toBe(DEFAULT_LOCALE);
    expect(getPreferredLocale('zh-CN', ['en-US', 'en'])).toBe(DEFAULT_LOCALE);
    expect(getPreferredLocale('ar', ['en-US', 'en'])).toBe(DEFAULT_LOCALE);
  });

  it('falls back to candidate matching when cache is empty', () => {
    expect(getPreferredLocale(null, ['es-MX', 'es'])).toBe('es');
    expect(getPreferredLocale(null, ['fr-FR', 'fr'])).toBe('fr');
    expect(getPreferredLocale(null, ['en-US', 'en'])).toBe('en');
  });

  it('falls back to candidate matching when cache is skeleton but candidates are ACTIVE', () => {
    expect(getPreferredLocale('ja', ['es-MX', 'es'])).toBe('es');
    expect(getPreferredLocale('de', ['fr-FR', 'fr'])).toBe('fr');
  });

  it('falls back to DEFAULT_LOCALE when nothing matches', () => {
    expect(getPreferredLocale(null, [])).toBe(DEFAULT_LOCALE);
    expect(getPreferredLocale(null, ['ko-KR', 'de-DE'])).toBe(DEFAULT_LOCALE);
    expect(getPreferredLocale('ja', ['ko-KR'])).toBe(DEFAULT_LOCALE);
  });

  it('treats null vs empty vs unknown cache the same', () => {
    const a = getPreferredLocale(null, ['en-US']);
    const b = getPreferredLocale('', ['en-US']);
    const c = getPreferredLocale('xx-unknown', ['en-US']);
    expect(a).toBe('en');
    expect(b).toBe('en');
    expect(c).toBe('en');
  });
});
