import i18n from '../../../i18n';
import type { Binding, TextSpec } from './appModel';

/**
 * Binding resolution: path lookup and string templates, nothing else. This
 * is data plumbing, not evaluation - no operators, no function calls - so
 * it never becomes a second rule engine (see appModel.ts).
 */

export type Scope = Record<string, unknown>;

const TEMPLATE = /\{\{\s*([^}]+?)\s*\}\}/g;
const WHOLE = /^\{\{\s*([^}]+?)\s*\}\}$/;

/** Reads a dotted path (vars.entity, queries.golden.data.0.code, row.winning_sources). */
export function getPath(scope: unknown, path: string): unknown {
  let cur: unknown = scope;
  for (const seg of path.split('.')) {
    if (cur === null || cur === undefined) return undefined;
    if (typeof cur !== 'object') return undefined;
    cur = (cur as Record<string, unknown>)[seg];
  }
  return cur;
}

const stringify = (v: unknown): string =>
  v === undefined || v === null ? '' : typeof v === 'object' ? JSON.stringify(v) : String(v);

/** Resolves one binding. A string that is exactly one {{path}} keeps the raw value's type. */
export function resolve(binding: Binding, scope: Scope): unknown {
  if (typeof binding !== 'string') return binding;
  const whole = WHOLE.exec(binding);
  if (whole) return getPath(scope, whole[1]);
  if (!binding.includes('{{')) return binding;
  return binding.replace(TEMPLATE, (_, p: string) => stringify(getPath(scope, p)));
}

/** Resolves every binding in a params/props object (one level, plus nested plain objects). */
export function resolveAll(obj: Record<string, Binding> | undefined, scope: Scope): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  if (!obj) return out;
  for (const [k, v] of Object.entries(obj)) {
    out[k] = v && typeof v === 'object' && !Array.isArray(v) ? resolveAll(v as Record<string, Binding>, scope) : resolve(v, scope);
  }
  return out;
}

/** A string that looks like an i18n key and is one, translated; anything else as written. */
function translateMaybe(s: string, params?: Record<string, unknown>): string {
  if (/^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)+$/.test(s) && i18n.exists(s)) return i18n.t(s, params) as string;
  return s;
}

/** Display text: templates resolved, then translated when it names an i18n key. */
export function text(spec: TextSpec | undefined, scope: Scope): string {
  if (spec === undefined || spec === null) return '';
  if (typeof spec === 'string') return translateMaybe(stringify(resolve(spec, scope)));
  const params = resolveAll(spec.params, scope);
  const key = stringify(resolve(spec.t, scope));
  // With the params: a pluralised key exists only as key_one / key_other, which i18next finds from count.
  return i18n.exists(key, params) ? (i18n.t(key, params) as string) : key;
}
