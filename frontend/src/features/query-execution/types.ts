/**
 * Canonical result shape consumed by QueryResultsPanel (Layer 2).
 *
 * Every backend response - previewQuery, executeQuery, runSavedQuery,
 * SSRS preview, future EXPLAIN endpoints - funnels through an adapter that
 * produces a QueryResultSet, so the panel only ever sees this one shape.
 * A fourth consumer adds an adapter, not a fork.
 */
export type ResultColumnType =
  | 'string'
  | 'number'
  | 'date'
  | 'boolean'
  | 'id'
  | 'unknown';

export interface ResultColumn {
  /** Physical column name as returned by the backend (e.g. "order_id"). */
  name: string;
  /** Optional human-readable label, e.g. the BO term display name. */
  label?: string;
  type?: ResultColumnType;
  nullable?: boolean;
}

export interface QueryResultSetMeta {
  /** True if the backend truncated at `limit` and the user might want more. */
  rowLimitHit?: boolean;
  /** Backend-reported execution time. Never fabricated. */
  durationMs?: number;
  /** SQL dialect the backend compiled to. */
  dialect?: string;
}

export interface QueryResultSet {
  columns: ResultColumn[];
  rows: Record<string, unknown>[];
  meta?: QueryResultSetMeta;
}
