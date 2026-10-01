import React from 'react';
import { beforeAll, describe, expect, it } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';
import { registerOperations } from '../../studio-core/operations/registry';
import { AppRuntimeProvider } from '../../pages/page-studio/app/AppRuntime';
import { queriesReadBy, summarizeQuery, WidgetQueryStatus } from '../../pages/page-studio/app/queryStatus';
import type { PageAppModel } from '../../pages/page-studio/app/appModel';

/**
 * Live query status on the design canvas: which queries a widget reads, and
 * what each is doing - rows, no rows, loading, an error, waiting for a
 * param, or paused by its Run only when.
 */

registerOperations([
  { id: 'qs.rows', domain: 'qs', kind: 'query', label: 'Rows', params: [], run: async () => ({ rows: [1, 2, 3], total: 3 }) },
  { id: 'qs.one', domain: 'qs', kind: 'query', label: 'One', params: [], run: async () => ({ rows: [1] }) },
  { id: 'qs.none', domain: 'qs', kind: 'query', label: 'None', params: [], run: async () => ({ rows: [] }) },
  { id: 'qs.slow', domain: 'qs', kind: 'query', label: 'Slow', params: [], run: () => new Promise(() => {}) },
  { id: 'qs.boom', domain: 'qs', kind: 'query', label: 'Boom', params: [], run: async () => { throw new Error('the server said no'); } },
  { id: 'qs.needs', domain: 'qs', kind: 'query', label: 'Needs', params: [{ name: 'id', type: 'string', required: true }, { name: 'kind', type: 'string', required: true }], run: async () => [] },
]);

beforeAll(loadRuleEngine, 30000);

describe('which queries a widget reads', () => {
  const declared = ['golden', 'profile', 'runs'];
  it('finds a named query, queries inside bindings and conditions, once each, and only declared ones', () => {
    expect(queriesReadBy({ props: { query: 'golden', rowsPath: 'rows', title: '{{queries.profile.data.name}} and {{queries.profile.data.code}}' } }, declared)).toEqual(['golden', 'profile']);
    expect(queriesReadBy({
      props: { columns: [{ cell: { value: '{{queries.runs.data.0.id}}' } }], optionsFrom: { query: 'profile' } },
      visibleWhen: { type: 'condition', field: 'queries.golden.data.length', operator: 'is_true' },
    }, declared)).toEqual(['runs', 'profile', 'golden']);
    expect(queriesReadBy({ props: { query: 'not_declared', text: '{{queries.ghost.data}}' } }, declared)).toEqual([]);
    expect(queriesReadBy({}, declared)).toEqual([]);
  });
});

describe('what a query is doing', () => {
  const s = (data: unknown, more: Partial<{ isLoading: boolean; error: unknown }> = {}) => ({ data, isLoading: false, isFetching: false, error: null, ...more });
  it('counts rows where the widget reads them', () => {
    expect(summarizeQuery(s({ rows: [1, 2] }), { rowsPath: 'rows' })).toMatchObject({ kind: 'rows', label: '2 rows', rows: 2 });
    expect(summarizeQuery(s({ rows: [1] }), { rowsPath: 'rows' }).label).toBe('1 row');
    expect(summarizeQuery(s([1, 2, 3]))).toMatchObject({ kind: 'rows', label: '3 rows' });
  });
  it('says why a list is empty - the likeliest cause of an empty widget', () => {
    expect(summarizeQuery(s({ rows: [] }), { rowsPath: 'rows' })).toMatchObject({ kind: 'empty', label: 'no rows', detail: expect.stringContaining('"rows"') });
    // A wrong rows path - the usual cause of an empty grid - names what the data does have.
    expect(summarizeQuery(s({ other: [1], total: 1 }), { rowsPath: 'rows' })).toMatchObject({
      kind: 'empty', label: 'no list at rows', detail: 'The query ran, but there is no list at "rows". The data has: other, total.',
    });
    expect(summarizeQuery(s([1, 2]), { rowsPath: 'rows' }).detail).toContain('The data is itself a list');
    expect(summarizeQuery(s(null))).toMatchObject({ kind: 'empty', label: 'no data' });
  });
  it('tells loading, errors, waiting and paused apart', () => {
    expect(summarizeQuery(s(undefined, { isLoading: true })).kind).toBe('loading');
    expect(summarizeQuery(s(undefined, { error: new Error('boom') }))).toMatchObject({ kind: 'error', detail: 'boom' });
    expect(summarizeQuery(s(undefined), { waitingFor: ['id', 'kind'] })).toMatchObject({ kind: 'waiting', detail: 'Waiting for id, kind to have a value.' });
    expect(summarizeQuery(s(undefined), { paused: true })).toMatchObject({ kind: 'waiting', label: 'paused' });
    expect(summarizeQuery(undefined)).toMatchObject({ kind: 'unknown' });
    expect(summarizeQuery(s({}), { operationKnown: false })).toMatchObject({ kind: 'error', label: 'no operation' });
    expect(summarizeQuery(s({ a: 1 }))).toMatchObject({ kind: 'ready' });
  });
});

function Page({ app, comp }: { app: PageAppModel; comp: { props?: Record<string, unknown> } }) {
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><MemoryRouter>
      <AppRuntimeProvider mode="design" app={app}>
        <WidgetQueryStatus comp={comp} app={app} />
      </AppRuntimeProvider>
    </MemoryRouter></QueryClientProvider>
  );
}

describe('the chips, through the real runtime', () => {
  const app: PageAppModel = {
    variables: [{ name: 'go', default: false }, { name: 'chosen' }],
    queries: [
      { id: 'ok', operation: 'qs.rows' }, { id: 'empty', operation: 'qs.none' }, { id: 'slow', operation: 'qs.slow' }, { id: 'boom', operation: 'qs.boom' },
      { id: 'needs', operation: 'qs.needs', params: { id: '{{vars.chosen}}', kind: 'x' } },
      // Its own operation: queries are cached by operation and params, so one sharing `ok`'s would show `ok`'s rows.
      { id: 'gated', operation: 'qs.one', enabledWhen: { type: 'condition', field: 'vars.go', operator: 'is_true' } },
      { id: 'ghost', operation: 'no.such.op' },
    ],
  };
  const show = (id: string, rowsPath?: string) => render(<Page app={app} comp={{ props: { query: id, ...(rowsPath ? { rowsPath } : {}) } }} />);

  it('counts the rows the widget reads', async () => {
    show('ok', 'rows');
    expect(await screen.findByText('ok · 3 rows')).toBeTruthy();
  }, 30000);

  it('shows an empty list as no rows, a slow one as loading, a failed one as an error with its message', async () => {
    const { unmount } = show('empty', 'rows');
    expect(await screen.findByText('empty · no rows')).toBeTruthy();
    unmount();
    const slow = show('slow');
    expect(await screen.findByText('slow · loading…')).toBeTruthy();
    slow.unmount();
    show('boom');
    const chip = await screen.findByText('boom · error');
    fireEvent.mouseOver(chip);
    expect(await screen.findByText(/the server said no/)).toBeTruthy();
  }, 30000);

  it('says what a waiting query is waiting for, and that a gated one is paused', async () => {
    const { unmount } = show('needs');
    const waiting = await screen.findByText('needs · waiting');
    fireEvent.mouseOver(waiting);
    // Only the param that is missing: kind has a value.
    expect(await screen.findByText(/Waiting for id to have a value\./)).toBeTruthy();
    unmount();
    show('gated');
    expect(await screen.findByText('gated · paused')).toBeTruthy();
  }, 30000);

  it('flags an operation that is not registered', async () => {
    show('ghost');
    expect(await screen.findByText('ghost · no operation')).toBeTruthy();
  }, 30000);

  it('shows several, three at most, with a count for the rest', async () => {
    render(<Page app={app} comp={{ props: { text: '{{queries.ok.data}} {{queries.empty.data}} {{queries.slow.data}} {{queries.boom.data}} {{queries.gated.data}}' } }} />);
    await waitFor(() => expect(screen.getByTestId('widget-query-status').querySelectorAll('.MuiChip-root')).toHaveLength(4));
    expect(screen.getByText('+2')).toBeTruthy();
  }, 30000);

  it('shows nothing for a widget that reads no query', () => {
    render(<Page app={app} comp={{ props: { text: 'Hello' } }} />);
    expect(screen.queryByTestId('widget-query-status')).toBeNull();
  });
});
