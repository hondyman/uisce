import { useEffect, useMemo, useState } from 'react';
import { evaluateRuleWasm } from '../../../rules/wasmRuntime';
import type { ConditionNode } from './appModel';
import type { Scope } from './bindings';

/**
 * Page conditions run through the one rule engine: the RuleNode JSON goes to
 * rule_engine.wasm's evaluateRule unchanged. This file only decides WHICH
 * scope values the engine sees - it never compares anything itself.
 */

/** The scope paths a condition reads - only these are sent to the engine. */
function fieldsOf(node: ConditionNode, into: Set<string> = new Set()): Set<string> {
  if (node.type === 'condition') into.add(node.field);
  else node.conditions.forEach((c) => fieldsOf(c, into));
  return into;
}

/**
 * A minimal context holding just the referenced paths, with the scope's real
 * nesting. A path stops at its first absent level, which is sent as null -
 * never an invented empty object - so "profile not loaded" stays empty even
 * when another path reads a field beneath it.
 */
function contextFor(node: ConditionNode, scope: Scope): Record<string, unknown> {
  const ctx: Record<string, unknown> = {};
  for (const path of fieldsOf(node)) {
    const segs = path.split('.');
    let cur = ctx;
    let src: unknown = scope;
    for (let i = 0; i < segs.length; i++) {
      const s = segs[i];
      const v = src !== null && typeof src === 'object' ? (src as Record<string, unknown>)[s] : undefined;
      if (i === segs.length - 1 || v === undefined || v === null || typeof v !== 'object') {
        if (!(s in cur) || cur[s] === null || typeof cur[s] !== 'object') cur[s] = v === undefined ? null : (i === segs.length - 1 ? v : null);
        break;
      }
      // Always a plain object: an array's .length or .0 then reads as a field.
      if (typeof cur[s] !== 'object' || cur[s] === null) cur[s] = {};
      cur = cur[s] as Record<string, unknown>;
      src = v;
    }
  }
  return ctx;
}

const cache = new Map<string, boolean>();

export async function evaluateCondition(node: ConditionNode | undefined, scope: Scope): Promise<boolean> {
  if (!node) return true;
  const ctx = contextFor(node, scope);
  const key = JSON.stringify([node, ctx]);
  const hit = cache.get(key);
  if (hit !== undefined) return hit;
  let ok: boolean;
  try {
    ok = await evaluateRuleWasm(node, ctx);
  } catch {
    // An unresolvable condition hides/disables rather than showing
    // something the author meant to gate.
    ok = false;
  }
  if (cache.size > 5000) cache.clear();
  cache.set(key, ok);
  return ok;
}

/**
 * A condition's current value. `pending` is what to assume until the engine
 * answers (first render): visible things default hidden so a gated button
 * never flashes on.
 */
export function useCondition(node: ConditionNode | undefined, scope: Scope, pending = false): boolean {
  const ctxKey = useMemo(() => (node ? JSON.stringify([node, contextFor(node, scope)]) : ''), [node, scope]);
  const [state, setState] = useState<{ key: string; value: boolean }>(() => ({
    key: ctxKey, value: node ? (cache.get(ctxKey) ?? pending) : true,
  }));
  useEffect(() => {
    if (!node) return;
    let live = true;
    void evaluateCondition(node, scope).then((v) => { if (live) setState({ key: ctxKey, value: v }); });
    return () => { live = false; };
    // ctxKey captures node + the values it reads.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ctxKey]);
  if (!node) return true;
  if (state.key === ctxKey) return state.value;
  return cache.get(ctxKey) ?? state.value;
}
