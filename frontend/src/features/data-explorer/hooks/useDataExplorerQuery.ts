/**
 * useDataExplorerQuery — runs the explorer's query against the chosen source.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  executeExplorer,
  previewExplorerPlan,
  previewExplorerSQL,
} from '../services/dataExplorerApi';
import type {
  ExplorerQueryState,
  ExplorerResult,
  ExplorerSource,
} from '../types/dataExplorerTypes';
import { devError } from '../../../utils/devLogger';

export interface UseDataExplorerQueryArgs {
  source: ExplorerSource | null;
  state: ExplorerQueryState;
}

export interface UseDataExplorerQueryResult {
  result: ExplorerResult | null;
  isLoading: boolean;
  isPreviewing: boolean;
  error: string | null;
  run: () => Promise<void>;
  refreshPreview: () => Promise<void>;
  lastRunAt: number | null;
}

function shallowEqualArrays<T>(a: T[], b: T[], eq: (x: T, y: T) => boolean): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i += 1) {
    if (!eq(a[i], b[i])) return false;
  }
  return true;
}

function dimensionsEqual(
  a: ExplorerQueryState['dimensions'],
  b: ExplorerQueryState['dimensions']
): boolean {
  return shallowEqualArrays(a, b, (x, y) => x.fieldId === y.fieldId && x.granularity === y.granularity);
}

function measuresEqual(
  a: ExplorerQueryState['measures'],
  b: ExplorerQueryState['measures']
): boolean {
  return shallowEqualArrays(a, b, (x, y) => x.fieldId === y.fieldId && x.agg === y.agg);
}

function filtersEqual(
  a: ExplorerQueryState['filters'],
  b: ExplorerQueryState['filters']
): boolean {
  return shallowEqualArrays(a, b, (x, y) => {
    if (x.fieldId !== y.fieldId || x.operator !== y.operator) return false;
    if (x.values.length !== y.values.length) return false;
    return x.values.every((v, i) => v === y.values[i]);
  });
}

function sortsEqual(a: ExplorerQueryState['sorts'], b: ExplorerQueryState['sorts']): boolean {
  return shallowEqualArrays(a, b, (x, y) => x.fieldId === y.fieldId && x.direction === y.direction);
}

export function statesEqual(a: ExplorerQueryState, b: ExplorerQueryState): boolean {
  return (
    a.sourceId === b.sourceId &&
    a.bindingId === b.bindingId &&
    a.limit === b.limit &&
    dimensionsEqual(a.dimensions, b.dimensions) &&
    dimensionsEqual(a.timeDimensions, b.timeDimensions) &&
    measuresEqual(a.measures, b.measures) &&
    filtersEqual(a.filters, b.filters) &&
    sortsEqual(a.sorts, b.sorts)
  );
}

export function useDataExplorerQuery({
  source,
  state,
}: UseDataExplorerQueryArgs): UseDataExplorerQueryResult {
  const [result, setResult] = useState<ExplorerResult | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastRunAt, setLastRunAt] = useState<number | null>(null);
  const lastRef = useRef<ExplorerQueryState | null>(null);

  const hasContent =
    state.dimensions.length +
      state.measures.length +
      state.timeDimensions.length >
    0;

  const refreshPreview = useCallback(async () => {
    if (!source) return;
    setIsPreviewing(true);
    try {
      const [sql, plan] = await Promise.all([
        previewExplorerSQL(source, state),
        previewExplorerPlan(source, state).catch(() => null),
      ]);
      setResult((prev) => {
        if (!prev) {
          return {
            columns: [],
            rows: [],
            sql,
            plan: plan ?? undefined,
            rowCount: 0,
            executionTimeMs: 0,
          };
        }
        return { ...prev, sql, plan: plan ?? prev.plan };
      });
    } catch (err) {
      devError('previewExplorerSQL failed', err);
    } finally {
      setIsPreviewing(false);
    }
  }, [source, state]);

  const run = useCallback(async () => {
    if (!source) return;
    if (!hasContent) {
      setResult({
        columns: [],
        rows: [],
        sql: '-- Select dimensions or measures to generate SQL.',
        rowCount: 0,
        executionTimeMs: 0,
      });
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const next = await executeExplorer(source, state);
      setResult(next);
      setLastRunAt(Date.now());
      lastRef.current = state;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to execute query.';
      setError(message);
      setResult((prev) => prev ?? { columns: [], rows: [], sql: '', rowCount: 0, executionTimeMs: 0 });
    } finally {
      setIsLoading(false);
    }
  }, [source, state, hasContent]);

  const lastState = lastRef.current;
  const previewSql = useMemo(() => state, [state]);

  useEffect(() => {
    if (!source) return undefined;
    const handle = setTimeout(() => {
      refreshPreview();
    }, 350);
    return () => clearTimeout(handle);
  }, [source, previewSql, refreshPreview]);

  useEffect(() => {
    if (!source) return;
    if (!lastState) return;
    if (!statesEqual(lastState, state)) {
      // query state changed but did not re-run — show stale SQL rather than silently changing
    }
  }, [source, lastState, state]);

  return useMemo(
    () => ({
      result,
      isLoading,
      isPreviewing,
      error,
      run,
      refreshPreview,
      lastRunAt,
    }),
    [result, isLoading, isPreviewing, error, run, refreshPreview, lastRunAt]
  );
}
