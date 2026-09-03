#!/usr/bin/env node
/**
 * i18n diff: report locale coverage gaps so CI can fail when a translation
 * is missing. Compares each locale against `en` (the canonical source).
 *
 * Usage: node scripts/i18n-diff.mjs
 * Exit 0 = all locales match en. Exit 1 = missing keys (CI fail).
 */
import { readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';

const SRC = 'src/locales';
const SOURCE = 'en';

function flat(obj, prefix = '') {
  const out = [];
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      out.push(...flat(v, key));
    } else {
      out.push(key);
    }
  }
  return out;
}

const sourceKeys = flat(JSON.parse(readFileSync(path.join(SRC, `${SOURCE}.json`), 'utf8')));
const locales = readdirSync(SRC)
  .filter((f) => f.endsWith('.json') && f !== `${SOURCE}.json`)
  .map((f) => f.replace(/\.json$/, ''));

let failed = false;
console.log(`Source (${SOURCE}): ${sourceKeys.length} keys\n`);
console.log('Coverage by locale:');
for (const locale of locales) {
  const file = path.join(SRC, `${locale}.json`);
  try { statSync(file); } catch { console.log(`  ${locale}: FILE MISSING`); failed = true; continue; }
  const keys = new Set(flat(JSON.parse(readFileSync(file, 'utf8'))));
  const missing = sourceKeys.filter((k) => !keys.has(k));
  const extra = [...keys].filter((k) => !sourceKeys.includes(k));
  const pct = ((keys.size / sourceKeys.length) * 100).toFixed(1);
  const status = missing.length === 0 ? 'OK' : `MISSING ${missing.length}`;
  console.log(`  ${locale.padEnd(6)} ${keys.size.toString().padStart(4)}/${sourceKeys.length} (${pct}%)  ${status}  extra:${extra.length}`);
  if (missing.length > 0) {
    failed = true;
    if (missing.length <= 20) {
      for (const k of missing) console.log(`      - ${k}`);
    } else {
      for (const k of missing.slice(0, 10)) console.log(`      - ${k}`);
      console.log(`      ... and ${missing.length - 10} more`);
    }
  }
}

if (failed) {
  console.error('\nFAIL: locale coverage is below 100% of source.');
  process.exit(1);
}
console.log('\nOK: all locales match source coverage.');
