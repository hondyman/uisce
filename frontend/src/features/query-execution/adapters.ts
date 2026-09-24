/**
 * Adapters from backend response shapes into the canonical QueryResultSet.
 *
 * Every consumer of useQueryExecution normalizes its backend response here
 * - QueryResultsPanel only ever accepts QueryResultSet, so a new consumer
 * (SSRS preview, future EXPLAIN, Report Builder) adds a 15-line adapter
 * instead of widening the panel's prop surface.
 */
import type { QueryExecuteResult } from '../query-builder/types/queryDef';
import type { QueryResultSet, ResultColumn } from './types';

function normalizeType(raw: string | undefined): ResultColumn['type'] {
  if (!raw) return 'unknown';
  const t = raw.toLowerCase();
  if (t === 'uuid' || t === 'id') return 'id';
  if (t === 'text' || t === 'varchar' || t === 'char' || t === 'string') return 'string';
  if (t === 'int' || t === 'integer' || t === 'bigint' || t === 'numeric' ||
      t === 'decimal' || t === 'float' || t === 'double' || t === 'number') return 'number';
  if (t === 'date' || t === 'time' || t === 'timestamp' || t === 'datetime' ||
      t === 'timestamptz') return 'date';
  if (t === 'bool' || t === 'boolean') return 'boolean';
  return 'unknown';
}

export function liveQueryResultToSet(res: QueryExecuteResult): QueryResultSet {
  return {
    columns: res.columns.map((c) => ({
      name: c.name,
      label: undefined,
      type: normalizeType(c.type),
      nullable: false,
    })),
    rows: res.rows,
    meta: {
      rowLimitHit: false,
      durationMs: res.executionTimeMs,
      dialect: 'postgres',
    },
  };
}

export interface SavedQueryRunShape {
  columns: { name: string; type?: string }[];
  rows: Record<string, unknown>[];
  /**
   * Total row count as reported by the saved-query `execute` endpoint. The
   * assumption is that the API returns the complete result set inline — i.e.
   * `rows.length` always equals `rowCount`. If the API ever switches to
   * pagination with truncated `rows`, this interface must change and the
   * adapter must fall back to `rows.length` (with a comment explaining the
   * silent misreport). Marked optional because older response shapes
   * predated the field.
   */
  rowCount?: number;
  /** Optional - older saved-query responses may not carry an execution time. */
  executionTimeMs?: number;
}

export function savedQueryResultToSet(res: SavedQueryRunShape): QueryResultSet {
  return {
    columns: res.columns.map((c) => ({
      name: c.name,
      label: undefined,
      type: normalizeType(c.type),
      nullable: false,
    })),
    rows: res.rows,
    meta: {
      rowLimitHit: false,
      durationMs: res.executionTimeMs,
      dialect: 'postgres',
    },
  };
}
