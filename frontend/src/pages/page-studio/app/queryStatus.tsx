import React from 'react';
import { Box, Chip, Stack, Tooltip, Typography } from '@mui/material';
import { getOperation, missingParams } from '../../../studio-core/operations/registry';
import type { PageAppModel, PageQuery } from './appModel';
import { useAppRuntime, type QueryState } from './AppRuntime';
import { getPath, resolveAll, type Scope } from './bindings';
import { useCondition } from './conditions';

/**
 * Live query status on the design canvas: for each widget, the queries it
 * reads and what they are doing right now - rows, loading, an error, or
 * waiting (with why: a required param still empty, or its Run only when not
 * holding). The most common "why is this empty?" is answered at a glance.
 */

export type QueryStatusKind = 'rows' | 'empty' | 'ready' | 'loading' | 'error' | 'waiting' | 'unknown';

export interface QueryStatus {
  kind: QueryStatusKind;
  /** Short text for the chip: "42 rows", "no rows", "loading…", "error", "waiting". */
  label: string;
  /** Why, in words (an error message, the params it is waiting for). */
  detail?: string;
  rows?: number;
}

/** The queries a widget reads: named by a `query` setting, or inside any binding or condition (`queries.<id>`). */
export function queriesReadBy(comp: { props?: Record<string, unknown>; visibleWhen?: unknown }, declared: string[]): string[] {
  const known = new Set(declared);
  const found: string[] = [];
  const add = (id: string) => { if (known.has(id) && !found.includes(id)) found.push(id); };
  const walk = (v: unknown, key?: string) => {
    if (typeof v === 'string') {
      if (key === 'query') add(v);
      for (const m of v.matchAll(/queries\.([A-Za-z0-9_]+)/g)) add(m[1]);
    } else if (Array.isArray(v)) v.forEach((x) => walk(x));
    else if (v && typeof v === 'object') for (const [k, x] of Object.entries(v)) walk(x, k);
  };
  walk(comp.props);
  walk(comp.visibleWhen);
  return found;
}

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

/** What a query is doing, from its state in the page scope. `rowsPath` says where its list is, when the widget reads a list inside it. */
export function summarizeQuery(state: QueryState | undefined, opts: { rowsPath?: string; waitingFor?: string[]; paused?: boolean; operationKnown?: boolean } = {}): QueryStatus {
  if (opts.operationKnown === false) return { kind: 'error', label: 'no operation', detail: 'This query names an operation that is not registered.' };
  if (!state) return { kind: 'unknown', label: 'not run' };
  if (state.error) {
    const message = state.error instanceof Error ? state.error.message : String(state.error);
    return { kind: 'error', label: 'error', detail: message };
  }
  if (state.isLoading) return { kind: 'loading', label: 'loading…' };
  if (state.data === undefined) {
    if (opts.paused) return { kind: 'waiting', label: 'paused', detail: 'Its "Run only when" condition does not hold yet.' };
    if (opts.waitingFor?.length) return { kind: 'waiting', label: 'waiting', detail: `Waiting for ${opts.waitingFor.join(', ')} to have a value.` };
    return { kind: 'waiting', label: 'waiting', detail: 'It has not run yet.' };
  }
  const list = opts.rowsPath ? getPath(state.data, opts.rowsPath) : state.data;
  // The widget reads a list at a path the data does not have: the usual cause of an empty grid.
  if (opts.rowsPath && !Array.isArray(list)) {
    const keys = state.data && typeof state.data === 'object' && !Array.isArray(state.data) ? Object.keys(state.data as object) : [];
    return {
      kind: 'empty', label: `no list at ${opts.rowsPath}`,
      detail: `The query ran, but there is no list at "${opts.rowsPath}".${keys.length ? ` The data has: ${keys.slice(0, 8).join(', ')}.` : Array.isArray(state.data) ? ' The data is itself a list: leave the rows path empty.' : ''}`,
    };
  }
  if (Array.isArray(list)) {
    return list.length
      ? { kind: 'rows', label: plural(list.length, 'row', 'rows'), rows: list.length }
      : { kind: 'empty', label: 'no rows', rows: 0, detail: opts.rowsPath ? `The query ran, but nothing is at "${opts.rowsPath}" - or the list is empty.` : 'The query ran and returned no rows.' };
  }
  if (state.data === null) return { kind: 'empty', label: 'no data', detail: 'The query ran and returned nothing.' };
  return { kind: 'ready', label: 'ready' };
}

const COLOR: Record<QueryStatusKind, 'success' | 'warning' | 'error' | 'info' | 'default'> = {
  rows: 'success', empty: 'warning', ready: 'success', loading: 'info', error: 'error', waiting: 'default', unknown: 'default',
};

const short = (v: unknown) => { const s = JSON.stringify(v); return s.length > 240 ? `${s.slice(0, 240)}…` : s; };

/** One query's chip, with a tooltip of what it runs, with which params, and why it is waiting or failed. */
export function QueryStatusChip({ query, scope, rowsPath }: { query: PageQuery; scope: Scope; rowsPath?: string }) {
  const op = getOperation(query.operation);
  const state = (scope.queries as Record<string, QueryState> | undefined)?.[query.id];
  const enabled = useCondition(query.enabledWhen, scope, true);
  const params = resolveAll(query.params, scope);
  const status = summarizeQuery(state, {
    rowsPath, operationKnown: !!op, paused: !!query.enabledWhen && !enabled,
    waitingFor: op ? missingParams(op, params) : [],
  });
  return (
    <Tooltip
      title={(
        <Box sx={{ maxWidth: 360 }}>
          <Typography variant="caption" component="div" fontFamily="monospace">{query.operation || '(no operation)'}</Typography>
          <Typography variant="caption" component="div">{status.label}{status.detail ? ` - ${status.detail}` : ''}</Typography>
          {Object.keys(params).length > 0 && <Typography variant="caption" component="div" fontFamily="monospace" sx={{ mt: 0.5, wordBreak: 'break-all' }}>{short(params)}</Typography>}
        </Box>
      )}>
      <Chip size="small" color={COLOR[status.kind]} variant="outlined" label={`${query.id} · ${status.label}`}
        sx={{ height: 18, fontSize: 10, bgcolor: 'background.paper', '& .MuiChip-label': { px: 0.75 } }} />
    </Tooltip>
  );
}

/** The chips for what a widget reads (up to three; the rest as a count). */
export function WidgetQueryStatus({ comp, app }: { comp: { props?: Record<string, unknown>; visibleWhen?: unknown }; app?: PageAppModel }) {
  const { scope } = useAppRuntime();
  const queries = app?.queries ?? [];
  const ids = queriesReadBy(comp, queries.map((q) => q.id));
  if (ids.length === 0) return null;
  const rowsPath = typeof comp.props?.rowsPath === 'string' ? comp.props.rowsPath : undefined;
  const own = comp.props?.query;
  return (
    <Stack direction="row" spacing={0.5} sx={{ position: 'absolute', top: 4, left: 4, zIndex: 2, maxWidth: '70%', flexWrap: 'wrap', rowGap: 0.5 }} data-testid="widget-query-status">
      {ids.slice(0, 3).map((id) => (
        <QueryStatusChip key={id} query={queries.find((q) => q.id === id)!} scope={scope} rowsPath={id === own ? rowsPath : undefined} />
      ))}
      {ids.length > 3 && <Chip size="small" variant="outlined" label={`+${ids.length - 3}`} sx={{ height: 18, fontSize: 10, bgcolor: 'background.paper' }} />}
    </Stack>
  );
}
