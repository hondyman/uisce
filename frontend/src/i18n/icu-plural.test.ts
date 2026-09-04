/**
 * ICU plural smoke test. Locks Fix A (the ICU migration of
 * search.results_count from {{count}} to ICU message format) and the
 * `lang`/`dir` sync handler in src/i18n.ts.
 *
 * If the ICU plugin isn't wired, the first test returns the raw
 * "{count, plural, …}" string — Fix A was a no-op and every migrated
 * key is broken.
 */
import { describe, it, expect, beforeAll } from 'vitest';
import i18n from '../i18n';

beforeAll(async () => {
  // Ensure i18next is initialized; init() resolves once plugins are ready.
  if (!i18n.isInitialized) {
    await new Promise((resolve) => i18n.on('initialized', resolve));
  }
});

describe('ICU plurals', () => {
  it('search.results_count renders count-driven plurals', () => {
    expect(i18n.t('search.results_count', { count: 0 })).toBe('0 results');
    expect(i18n.t('search.results_count', { count: 1 })).toBe('1 result');
    expect(i18n.t('search.results_count', { count: 2 })).toBe('2 results');
    expect(i18n.t('search.results_count', { count: 5 })).toBe('5 results');
  });

  it('returns literal ICU syntax if plugin is misconfigured', () => {
    // This is a NEGATIVE test — if the ICU plugin is wired, this passes
    // (because the rendered output IS the count). If it's not wired, the
    // i18n.t call would return the raw `{count, plural, ...}` string and
    // this test would fail. The previous test is the positive gate.
    expect(i18n.t('search.results_count', { count: 1 })).not.toContain('{count');
  });
});

describe('lang/dir sync handler', () => {
  it('updates <html lang> and dir on changeLanguage', async () => {
    document.documentElement.lang = 'en';
    document.documentElement.dir = 'ltr';

    await i18n.changeLanguage('ar');
    expect(document.documentElement.lang).toBe('ar');
    expect(document.documentElement.dir).toBe('rtl');

    await i18n.changeLanguage('en');
    expect(document.documentElement.lang).toBe('en');
    expect(document.documentElement.dir).toBe('ltr');
  });

  it('Japanese locale sets LTR direction (RTL set covers only ar/he/fa)', async () => {
    await i18n.changeLanguage('ja');
    expect(document.documentElement.lang).toBe('ja');
    expect(document.documentElement.dir).toBe('ltr');

    await i18n.changeLanguage('en');
  });
});
