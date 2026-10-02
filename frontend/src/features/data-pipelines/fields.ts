import type { Column, ColumnType, Spec, SpecNode, TargetField } from './api';

/**
 * Field flow through a pipeline, so every picker offers only fields that
 * exist at that step. The graph is a forest (one input per node).
 *
 * sourceFields: fields a source produces (from the file contract, or the BO
 * schema looked up by the caller).
 */
export type SourceFieldLookup = (node: SpecNode) => Column[] | undefined;

export function parentOf(spec: Spec, id: string): SpecNode | undefined {
  const e = spec.edges.find(x => x.to === id);
  return e ? spec.nodes.find(n => n.id === e.from) : undefined;
}

export function childrenOf(spec: Spec, id: string): SpecNode[] {
  return spec.edges.filter(e => e.from === id)
    .map(e => spec.nodes.find(n => n.id === e.to))
    .filter((n): n is SpecNode => !!n);
}

/** Fields leaving node id. */
export function fieldsOut(spec: Spec, id: string, lookup: SourceFieldLookup, seen = new Set<string>()): Column[] {
  if (seen.has(id)) return []; // cycles are a validation error, not a crash
  seen.add(id);
  const node = spec.nodes.find(n => n.id === id);
  if (!node) return [];
  switch (node.type) {
    case 'file_source': {
      const cols = (node.config as { columns?: Column[] }).columns;
      return cols && cols.length ? cols : lookup(node) ?? [];
    }
    case 'bo_source':
      return lookup(node) ?? [];
    case 'map': {
      const cfg = node.config as { fields?: { from: string; to: string; transform?: string }[]; keep_unmapped?: boolean };
      const inCols = fieldsIn(spec, id, lookup, seen);
      const byName = new Map(inCols.map(c => [c.name, c]));
      const mapped = (cfg.fields ?? []).filter(f => f.to).map(f => ({
        name: f.to,
        type: transformType(f.transform) ?? byName.get(f.from)?.type ?? 'string',
      } as Column));
      if (!cfg.keep_unmapped) return mapped;
      const used = new Set((cfg.fields ?? []).map(f => f.from));
      return [...mapped, ...inCols.filter(c => !used.has(c.name) && !mapped.some(m => m.name === c.name))];
    }
    default: // validate, rule_check and sinks pass rows through unchanged
      return fieldsIn(spec, id, lookup, seen);
  }
}

/** Fields arriving at node id. */
export function fieldsIn(spec: Spec, id: string, lookup: SourceFieldLookup, seen = new Set<string>()): Column[] {
  const p = parentOf(spec, id);
  return p ? fieldsOut(spec, p.id, lookup, seen) : [];
}

function transformType(t?: string): ColumnType | undefined {
  switch (t) {
    case 'to_date': return 'date';
    case 'to_number': return 'decimal';
    case 'trim': case 'upper': case 'lower': case 'lookup': return 'string';
  }
  return undefined;
}

/**
 * The destination a map step feeds, found by walking down through
 * pass-through steps. A map feeding several sinks targets the first.
 */
export function downstreamSink(spec: Spec, id: string): SpecNode | undefined {
  const queue = [...childrenOf(spec, id)];
  const seen = new Set<string>();
  while (queue.length) {
    const n = queue.shift()!;
    if (seen.has(n.id)) continue;
    seen.add(n.id);
    if (n.type === 'bo_sink' || n.type === 'staging_sink' || n.type === 'file_sink') return n;
    if (n.type !== 'map') queue.push(...childrenOf(spec, n.id));
  }
  return undefined;
}

/** Required targets not produced by the mapping. */
export function missingRequired(mappedTo: string[], targets: TargetField[]): TargetField[] {
  const have = new Set(mappedTo);
  return targets.filter(t => t.required && !have.has(t.name));
}

let seq = 0;
export function newNodeId(kind: string, spec: Spec): string {
  let id: string;
  do { id = `${kind}_${++seq}`; } while (spec.nodes.some(n => n.id === id));
  return id;
}
