/**
 * Placeholder parity lint. Verifies that every locale's translation:
 *   1. Uses the same set of {{placeholders}} as en (no missing, no extra).
 *   2. Has no stray single-brace tokens ({x}) outside of valid ICU-style
 *    syntax (which we don't use — native i18next requires {{x}}).
 *
 * This is the systemic version of the {{count}} vs {count} near-miss from
 * the ICU → native-suffix-plural migration. It catches both translator
 * mistakes (single-brace tokens from TMS exports) and developer mistakes
 * (forgetting to interpolate a value).
 *
 * Adds ~150ms to the test suite; runs as part of `pnpm test:vitest`.
 */
import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';

const LOCALES_DIR = path.resolve(__dirname, '../locales');
const SOURCE = 'en';

const DOUBLE_BRACE_RE = /\{\{(\w+)\}\}/g;
const SINGLE_BRACE_RE = /(?<!\{)\{[a-zA-Z_]\w*\}(?!\})/g;
const ICU_PATTERN = /\{\w+,\s*(plural|select|number|date|time)\b/;

function flatLeaves(obj: Record<string, unknown>, prefix = ''): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      Object.assign(out, flatLeaves(v as Record<string, unknown>, key));
    } else {
      out[key] = v;
    }
  }
  return out;
}

function placeholders(value: string): Set<string> {
  return new Set(Array.from(value.matchAll(DOUBLE_BRACE_RE), (m): string => m[1] ?? ''));
}

function loadLocale(name: string) {
  return flatLeaves(JSON.parse(readFileSync(path.join(LOCALES_DIR, `${name}.json`), 'utf8')));
}

const locales = readdirSync(LOCALES_DIR)
  .filter((f) => f.endsWith('.json') && f !== `${SOURCE}.json`)
  .map((f) => f.replace(/\.json$/, ''));

describe('placeholder parity across locales', () => {
  const sourceKeys = loadLocale(SOURCE);

  for (const locale of locales) {
    describe(`${locale}`, () => {
      const localeKeys = loadLocale(locale);

      it('has no stray single-brace tokens outside ICU syntax', () => {
        for (const [key, value] of Object.entries(localeKeys)) {
          if (typeof value !== 'string') continue;
          // ICU messages intentionally use {x, plural, …} — exempt those.
          if (ICU_PATTERN.test(value)) continue;
          const stray = value.match(SINGLE_BRACE_RE);
          expect(
            stray,
            `${locale} :: ${key} contains stray single-brace token(s): ${stray?.join(', ') ?? ''}\n  value: ${JSON.stringify(value)}`,
          ).toBeNull();
        }
      });

      it('uses the same {{placeholders}} as en (when key exists in both)', () => {
        for (const [key, value] of Object.entries(localeKeys)) {
          if (typeof value !== 'string') continue;
          if (!(key in sourceKeys)) continue; // extra key in locale — not a placeholder mismatch
          const sourceVal = sourceKeys[key];
          if (typeof sourceVal !== 'string') continue;
          const srcPH = placeholders(sourceVal);
          const locPH = placeholders(value);
          expect(
            Array.from(locPH).sort(),
            `${locale} :: ${key}\n  en placeholders: ${JSON.stringify(Array.from(srcPH).sort())}\n  ${locale} placeholders: ${JSON.stringify(Array.from(locPH).sort())}\n  en value: ${JSON.stringify(sourceVal)}\n  ${locale} value: ${JSON.stringify(value)}`,
          ).toEqual(Array.from(srcPH).sort());
        }
      });
    });
  }
});
