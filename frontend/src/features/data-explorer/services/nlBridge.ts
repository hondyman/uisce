/**
 * Bridge from the NL engine's ParsedIntent / GeneratedQuery shape into
 * the Data Explorer's ExplorerQueryState so a natural-language prompt
 * populates the dim/measure/filter chips.
 */

import type { ExplorerField, ExplorerSource, ExplorerQueryState, AggFn, FilterSelection } from '../types/dataExplorerTypes';

export interface NLParseResult {
  dimensions: string[];
  measures: string[];
  filters: FilterSelection[];
  sql?: string;
  confidence?: number;
}

/**
 * Translate a backend NL response (GeneratedQuery + ParsedIntent fields)
 * into a partial ExplorerQueryState diff.
 *
 * The matcher is intentionally lenient: server-side metric/dim names
 * are looked up against the explore source's fields by name OR by
 * term-key so language mismatches degrade gracefully (unmatched terms
 * are dropped from the diff, leaving existing state intact).
 */
export function applyNLToExplorerQueryState(
  prev: ExplorerQueryState,
  source: ExplorerSource,
  parsed: NLParseResult
): ExplorerQueryState {
  const fieldByName = new Map<string, ExplorerField>();
  const fieldByLower = new Map<string, ExplorerField>();
  source.fields.forEach((f) => {
    fieldByName.set(f.name, f);
    fieldByName.set(f.displayName, f);
    fieldByLower.set(f.name.toLowerCase(), f);
    fieldByLower.set(f.displayName.toLowerCase(), f);
  });

  const lookup = (name: string): ExplorerField | undefined => {
    return fieldByName.get(name) || fieldByLower.get(name.toLowerCase());
  };

  const dimensions = parsed.dimensions
    .map((name) => lookup(name))
    .filter((f): f is ExplorerField => f !== undefined && f.category === 'dimension')
    .map((f) => ({ fieldId: f.id }));

  const timeDimensions = parsed.dimensions
    .map((name) => lookup(name))
    .filter((f): f is ExplorerField => f !== undefined && f.category === 'time')
    .map((f) => ({ fieldId: f.id, granularity: 'month' as const }));

  const measures = parsed.measures
    .map((name) => lookup(name))
    .filter((f): f is ExplorerField => f !== undefined && f.category === 'measure')
    .map((f) => ({ fieldId: f.id, agg: (f.defaultAggregation || 'SUM') as AggFn }));

  return {
    ...prev,
    dimensions: dedupeDimensionField(dimensions, timeDimensions),
    measures: dedupeMeasureField(measures),
    filters: parsed.filters,
  };
}

function dedupeDimensionField(
  dims: { fieldId: string }[],
  times: { fieldId: string; granularity: 'month' }[]
): ExplorerQueryState['dimensions'] {
  const seen = new Set<string>();
  const out: ExplorerQueryState['dimensions'] = [];
  times.forEach((t) => {
    if (!seen.has(t.fieldId)) {
      seen.add(t.fieldId);
      out.push({ fieldId: t.fieldId, granularity: t.granularity });
    }
  });
  dims.forEach((d) => {
    if (!seen.has(d.fieldId)) {
      seen.add(d.fieldId);
      out.push({ fieldId: d.fieldId });
    }
  });
  return out;
}

function dedupeMeasureField(list: { fieldId: string; agg: AggFn }[]) {
  const seen = new Set<string>();
  const out = [] as { fieldId: string; agg: AggFn }[];
  list.forEach((m) => {
    if (!seen.has(m.fieldId)) {
      seen.add(m.fieldId);
      out.push(m);
    }
  });
  return out;
}
