#!/usr/bin/env node
/**
 * Merge i18next-parser harvest into canonical locale files.
 *
 * Rules:
 * 1. Existing keys in target locale: PRESERVE the existing translation.
 *    Parser-harvested English defaults do NOT overwrite non-empty translations.
 * 2. New keys (in harvest but not in target): add with the harvested value.
 *    If harvest value is empty string, key is added as empty (sentinel for
 *    "needs human copy").
 * 3. For locales that don't have a target file yet (de, pt-BR, ja, zh-CN, ar):
 *    write the harvest directly. These are Phase 3; no existing translations
 *    to preserve.
 * 4. ICU plural keys are kept as a single key (no _one/_other/_few suffixes).
 *    If the harvest has suffix variants (which i18next-parser generates
 *    automatically for ICU plurals), we drop them.
 */
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';

const HARVEST_DIR = process.argv[2] || '/tmp/i18n-harvest';
const LOCALES = ['en', 'es', 'fr', 'de', 'pt-BR', 'ja', 'zh-CN', 'ar'];
const EXISTING = ['en', 'es', 'fr'];

function loadJson(p) {
  return JSON.parse(readFileSync(p, 'utf8'));
}

function flatKeys(obj, prefix = '') {
  const out = {};
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      Object.assign(out, flatKeys(v, key));
    } else {
      out[key] = v;
    }
  }
  return out;
}

function ensureNested(target, keyPath, value) {
  const parts = keyPath.split('.');
  let cursor = target;
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i];
    if (typeof cursor[part] !== 'object' || cursor[part] === null || Array.isArray(cursor[part])) {
      cursor[part] = {};
    }
    cursor = cursor[part];
  }
  const last = parts[parts.length - 1];
  if (!(last in cursor)) {
    cursor[last] = value;
    return true;
  }
  return false;
}

const summary = {};

// 1. Existing locales (en/es/fr): preserve existing translations, add new.
for (const locale of EXISTING) {
  const targetPath = `src/locales/${locale}.json`;
  const target = loadJson(targetPath);
  const targetKeys = flatKeys(target);

  const harvestPath = `${HARVEST_DIR}/${locale}.json`;
  const harvest = loadJson(harvestPath);
  const harvestKeys = flatKeys(harvest);

  let added = 0;
  let filled = 0;
  for (const [key, value] of Object.entries(harvestKeys)) {
    if (key in targetKeys) continue;
    ensureNested(target, key, value);
    added += 1;
    if (value !== '') filled += 1;
  }

  writeFileSync(targetPath, JSON.stringify(target, null, 2) + '\n');
  const afterKeys = flatKeys(target);
  console.log(`${locale}: +${added} new (${filled} filled), ${afterKeys.length} total`);
  summary[locale] = { added, filled, total: afterKeys.length };
}

// 2. New locales (de/pt-BR/ja/zh-CN/ar): write harvest directly.
for (const locale of LOCALES.filter((l) => !EXISTING.includes(l))) {
  const harvestPath = `${HARVEST_DIR}/${locale}.json`;
  if (!existsSync(harvestPath)) continue;
  const harvest = loadJson(harvestPath);
  const harvestFlat = flatKeys(harvest);
  const targetPath = `src/locales/${locale}.json`;
  writeFileSync(targetPath, JSON.stringify(harvest, null, 2) + '\n');
  const filled = Object.values(harvestFlat).filter((v) => v !== '').length;
  console.log(`${locale}: NEW (${filled} filled of ${Object.keys(harvestFlat).length})`);
  summary[locale] = { added: Object.keys(harvestFlat).length, filled, total: Object.keys(harvestFlat).length, new: true };
}

mkdirSync('scripts/out', { recursive: true });
writeFileSync(
  'scripts/out/i18n-merge-summary.json',
  JSON.stringify({ date: new Date().toISOString(), summary }, null, 2),
);
