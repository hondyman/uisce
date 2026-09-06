/**
 * Native i18next suffix-plural smoke test. Locks the migration of
 * `search.results_count` from ICU syntax (`{count, plural, =0 {...}
 * one {...} other {...}}`) to native suffix keys (`results_count_one`,
 * `results_count_other`), driven by `Intl.PluralRules`.
 *
 * If suffix plurals stop working, `t('search.results_count', { count: 1 })`
 * would return `'search.results_count'` (raw key) instead of `'1 result'`.
 * Build + ratchet can't see this — only the runtime assertion catches it.
 *
 * Counterfactual: had we shipped the v2 ICU plugin patch (with a 2021-era
 * dependency that produces a namespace where a constructor was expected,
 * silently returning raw ICU strings) without this test, every plural
 * search result in the UI would have rendered literally as the ICU syntax.
 */
import { describe, it, expect, beforeAll } from 'vitest';
import i18n from '../i18n';

beforeAll(async () => {
  if (!i18n.isInitialized) {
    await new Promise((resolve) => i18n.on('initialized', resolve));
  }
});

describe('native suffix plurals', () => {
  it('search.results_count picks _one / _other by count', () => {
    // en-US CLDR plural rules: 1 → 'one', everything else → 'other'
    expect(i18n.t('search.results_count', { count: 0 })).toBe('0 results');
    expect(i18n.t('search.results_count', { count: 1 })).toBe('1 result');
    expect(i18n.t('search.results_count', { count: 2 })).toBe('2 results');
    expect(i18n.t('search.results_count', { count: 5 })).toBe('5 results');
    expect(i18n.t('search.results_count', { count: 42 })).toBe('42 results');
  });

  it('without count, returns the base key (i18next default — callsite must pass count)', () => {
    // i18next native suffix plurals require `count` to pick a suffix.
    // Without it, the base key is returned. This is correct i18next
    // behavior; the call site (ProfessionalSearchInput.tsx:206) passes
    // count. Document the contract.
    expect(i18n.t('search.results_count')).toBe('search.results_count');
  });

  it('es plural — the only ACTIVE non-en locale; runtime-verifies its translation', () => {
    // ES CLDR plural rules: 1 → 'one', everything else → 'other'
    // This guards against a vendor delivery or hand-edit dropping the
    // es translation or breaking its placeholder substitution.
    return i18n.changeLanguage('es').then(() => {
      expect(i18n.t('search.results_count', { count: 1 })).toBe('1 resultado');
      expect(i18n.t('search.results_count', { count: 5 })).toBe('5 resultados');
      expect(i18n.t('search.results_count', { count: 0 })).toBe('0 resultados');
    });
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
