/**
 * useQueryExecution - the single hook that owns run / lazy-compile / error
 * normalization for any consumer of QueryResultsPanel.
 *
 * Callers wire only: they pass a `buildDef` closure that produces the
 * current QueryDef (or null when the query isn't runnable), and the hook
 * returns typed state and callbacks. The hook itself:
 *   - calls previewQuery on demand for the Compiled SQL tab (lazy, only
 *     when the tab opens, never on mount)
 *   - calls executeQuery for Run, normalizing errors through friendlyQueryError
 *   - never fabricates results, SQL, or timing data
 *
 * Phase 1 ships the wiring surface; Phase 2 will migrate LiveQueryTab and
 * Phase 3 will migrate FilterBuilderPanel onto this same hook.
 */
import { useCallback, useEffect, useState } from 'react';
import type { QueryDef } from '../query-builder/types/queryDef';
import type { QueryResultSet } from './types';
import { fetchCompiledSql, runExecute, type CompiledSql } from './queryExecutionApi';
import { friendlyQueryError } from './errorMessage';

export interface UseQueryExecutionArgs {
  /** Re-evaluated on every run/compile. Return null to disable. */
  buildDef: () => QueryDef | null;
  /** Auto-run execute on mount. Default false. */
  autoRunOnMount?: boolean;
}

export interface UseQueryExecution {
  // execute state
  run: () => Promise<void>;
  running: boolean;
  runError: string | null;
  resultSet: QueryResultSet | null;
  // compile state
  requestCompileSql: () => Promise<void>;
  sqlLoading: boolean;
  sqlError: string | null;
  sql: CompiledSql | null;
  // control
  reset: () => void;
}

export function useQueryExecution({
  buildDef,
  autoRunOnMount = false,
}: UseQueryExecutionArgs): UseQueryExecution {
  const [resultSet, setResultSet] = useState<QueryResultSet | null>(null);
  const [running, setRunning] = useState(false);
  const [runError, setRunError] = useState<string | null>(null);

  const [sql, setSql] = useState<CompiledSql | null>(null);
  const [sqlLoading, setSqlLoading] = useState(false);
  const [sqlError, setSqlError] = useState<string | null>(null);

  const run = useCallback(async () => {
    const def = buildDef();
    if (!def) return;
    setRunning(true);
    setRunError(null);
    try {
      setResultSet(await runExecute(def));
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      setRunError(friendlyQueryError(msg));
    } finally {
      setRunning(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [buildDef]);

  const requestCompileSql = useCallback(async () => {
    const def = buildDef();
    if (!def) return;
    setSqlLoading(true);
    setSqlError(null);
    try {
      setSql(await fetchCompiledSql(def));
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      setSqlError(friendlyQueryError(msg));
    } finally {
      setSqlLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [buildDef]);

  const reset = useCallback(() => {
    setResultSet(null);
    setRunError(null);
    setSql(null);
    setSqlError(null);
  }, []);

  useEffect(() => {
    if (autoRunOnMount) run();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return {
    run,
    running,
    runError,
    resultSet,
    requestCompileSql,
    sqlLoading,
    sqlError,
    sql,
    reset,
  };
}
