#!/usr/bin/env node
/**
 * i18n coverage report. Compares each locale against `en` (the canonical
 * source) and reports three numbers per locale:
 *   - missing:   keys present in en but absent in locale
 *   - extra:     keys present in locale but absent in en
 *   - identity:  keys present in both with byte-identical values — these
 *                look "translated" but are actually English fallbacks.
 *                The 5 skeleton locales (de/pt-BR/ja/zh-CN/ar) currently
 *                have all keys identity with en, so naive coverage
 *                would report 100% for them. Real translation completion
 *                = (present-and-not-identity) / total-en-keys.
 *
 * Usage:
 *   node scripts/i18n-diff.mjs                  # report only, exit 0
 *   node scripts/i18n-diff.mjs --strict         # exit 1 if skeleton locales
 *                                              # have any identity-as-translation
 *   node scripts/i18n-diff.mjs --threshold=N    # fail if any ACTIVE locale has
 *                                              # identity > N keys (vendor gate)
 *
 * Exit codes:
 *   0 = coverage within thresholds (or --strict not set)
 *   1 = any locale missing keys OR has identity-as-translation
 *   2 = no source locale file found
 */
import { readFileSync, readdirSync, statSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import path from 'node:path';

const SRC = 'src/locales';
const SOURCE = 'en';
const ACTIVE_LOCALES = new Set(['en', 'es', 'fr']);
const STRICT = process.argv.includes('--strict');

// Keys whose value is expected to be byte-identical to English across
// all locales: brand names, acronyms, units, etc. These SHOULD NOT
// count as untranslated work. Phase 2 vendor briefs MUST keep this list
// in sync; Phase 3 vendor acceptance uses it as the gate for "0 missing
// identity" deliverables.
//
// Add to this list ONLY when the key's English value is a known
// non-translatable (brand, unit, code, etc.). Do NOT add keys whose
// values are full sentences in English — those ARE untranslated work.
const IDENTITY_ALLOWLIST = new Set([
  'app.name',
]);

function isIdentityExpected(key) {
  return IDENTITY_ALLOWLIST.has(key);
}

function loadJson(p) {
  return JSON.parse(readFileSync(p, 'utf8'));
}

function flatLeaves(obj, prefix = '') {
  const out = {};
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      Object.assign(out, flatLeaves(v, key));
    } else {
      out[key] = v;
    }
  }
  return out;
}

const sourcePath = path.join(SRC, `${SOURCE}.json`);
if (!existsSync(sourcePath)) {
  console.error(`Missing source locale: ${sourcePath}`);
  process.exit(2);
}

const sourceFlat = flatLeaves(loadJson(sourcePath));
const sourceKeys = Object.keys(sourceFlat);
const locales = readdirSync(SRC)
  .filter((f) => f.endsWith('.json') && f !== `${SOURCE}.json`)
  .map((f) => f.replace(/\.json$/, ''))
  .sort();

let failed = false;
const summary = {
  date: new Date().toISOString(),
  source: SOURCE,
  sourceKeys: sourceKeys.length,
  locales: {},
};

console.log(`Source (${SOURCE}): ${sourceKeys.length} keys\n`);
console.log(
  'Locale       present  missing  identity  translated  translated-pct  extra',
);
console.log(
  '-----------  -------  -------  --------  ----------  --------------  -----',
);

for (const locale of locales) {
  const file = path.join(SRC, `${locale}.json`);
  let localeFlat;
  try {
    statSync(file);
    localeFlat = flatLeaves(loadJson(file));
  } catch {
    console.log(
      `  ${locale.padEnd(10)}  FILE MISSING`,
    );
    summary.locales[locale] = { error: 'file_missing' };
    failed = true;
    continue;
  }

  const localeKeys = Object.keys(localeFlat);
  const missing = sourceKeys.filter((k) => !(k in localeFlat));
  // Identity = present in both AND value is byte-identical to en.
  // Empty values are also "identity" (no translation work done).
  // Keys in IDENTITY_ALLOWLIST are excluded — they're expected to match.
  const identity = localeKeys.filter(
    (k) =>
      k in sourceFlat &&
      (localeFlat[k] === sourceFlat[k] || localeFlat[k] === '') &&
      !isIdentityExpected(k),
  );
  const translated = localeKeys.length - identity.length;
  const translatedPct = sourceKeys.length > 0
    ? ((translated / sourceKeys.length) * 100).toFixed(1)
    : '0.0';
  const extra = localeKeys.filter((k) => !(k in sourceFlat));

  const isActive = ACTIVE_LOCALES.has(locale);
  const status = missing.length === 0 && identity.length === 0 ? 'OK' : 'GAP';
  if (isActive && missing.length > 0) failed = true;

  // Vendor acceptance gate: an ACTIVE locale is "delivered" when it has
  // zero missing keys AND zero untranslated-identity keys (allowing for
  // the explicit IDENTITY_ALLOWLIST above). Pure `status === 'OK'` would
  // also work here; we keep both for visibility in the table.
  const delivered = isActive && missing.length === 0 && identity.length === 0;

  console.log(
    `  ${locale.padEnd(10)}  ` +
    `${String(localeKeys.length).padStart(7)}  ` +
    `${String(missing.length).padStart(7)}  ` +
    `${String(identity.length).padStart(8)}  ` +
    `${String(translated).padStart(10)}  ` +
    `${translatedPct.padStart(12)}%  ` +
    `${String(extra.length).padStart(5)}  ${status}${isActive ? '' : ' (skeleton)'}${delivered ? ' [DELIVERED]' : ''}`,
  );

  if (missing.length > 0 && missing.length <= 8) {
    for (const k of missing) console.log(`      missing: ${k}`);
  } else if (missing.length > 8) {
    for (const k of missing.slice(0, 5)) console.log(`      missing: ${k}`);
    console.log(`      ... and ${missing.length - 5} more missing`);
  }

  summary.locales[locale] = {
    active: isActive,
    present: localeKeys.length,
    missing: missing.length,
    identity: identity.length,
    translated,
    translatedPct: parseFloat(translatedPct),
    extra: extra.length,
    status,
  };
}

mkdirSync('scripts/out', { recursive: true });
writeFileSync(
  'scripts/out/i18n-coverage.json',
  JSON.stringify(summary, null, 2),
);

console.log('');

// Parse --threshold=N (vendor acceptance: fail if any ACTIVE locale has
// identity-key count > N). Default: --threshold=0 (current behavior).
// Realistic Phase 3 acceptance: --threshold=20 (a few stragglers OK).
const thresholdArg = process.argv.find((a) => a.startsWith('--threshold='));
const IDENTITY_THRESHOLD = thresholdArg ? parseInt(thresholdArg.split('=')[1], 10) : 0;

if (failed) {
  console.error('FAIL: at least one active locale has missing keys.');
  process.exit(1);
}
const overThreshold = Object.entries(summary.locales).filter(
  ([, v]) => v.active && v.identity > IDENTITY_THRESHOLD,
);
if (overThreshold.length > 0) {
  console.error(
    `FAIL: ${overThreshold.length} active locale(s) exceed identity-threshold ${IDENTITY_THRESHOLD} (--threshold=${IDENTITY_THRESHOLD}):`,
  );
  for (const [locale, v] of overThreshold) {
    console.error(`  ${locale}: identity=${v.identity}`);
  }
  process.exit(1);
}
if (STRICT) {
  const skeletons = Object.entries(summary.locales).filter(
    ([, v]) => !v.error && v.identity > 0 && !v.active,
  );
  if (skeletons.length > 0) {
    console.error(
      `FAIL (--strict): ${skeletons.length} skeleton locale(s) have all-key identity — translation never started.`,
    );
    process.exit(1);
  }
}
console.log('OK: no active locale has missing keys.');
