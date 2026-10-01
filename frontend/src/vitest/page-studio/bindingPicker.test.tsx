import React, { useState } from 'react';
import { beforeAll, describe, expect, it } from 'vitest';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';
import { registerOperations } from '../../studio-core/operations/registry';
import { AppRuntimeProvider } from '../../pages/page-studio/app/AppRuntime';
import { BindingField, ConditionEditor } from '../../pages/page-studio/app/editors';
import { browsableScope, entriesUnder, flattenScope } from '../../pages/page-studio/app/bindingPicker';
import type { ConditionNode } from '../../pages/page-studio/app/appModel';

/**
 * The binding picker: the page's live state as a tree (variables, route,
 * each query's real result), searchable, with the names that exist only in
 * the setting's context; a pick lands in the field as {{path}} (a binding)
 * or as the bare path (a condition).
 */

registerOperations([{
  id: 'test.vendors', domain: 'test', kind: 'query', label: 'Vendors', params: [],
  run: async () => ({ can_edit: true, rows: [{ code: 'FACTSET', display_name: 'FactSet', meta: { tier: 1 } }, { code: 'ICE', display_name: 'ICE' }] }),
}]);

beforeAll(loadRuleEngine, 30000);

describe('scope helpers', () => {
  const data = { vars: { entity: 'product', open: false }, queries: { v: { data: { rows: [{ code: 'A', n: 2 }], total: 1 } } } };
  it('lists an object by key and a list by its length and first item', () => {
    expect(entriesUnder(data, '').map((e) => `${e.path}:${e.type}`)).toEqual(['vars:object', 'queries:object']);
    expect(entriesUnder(data.queries.v.data.rows, 'queries.v.data.rows').map((e) => `${e.name}:${e.type}:${e.preview}`))
      .toEqual(['length:number:1', '0:object:code, n']);
  });
  it('flattens to every path for search, to a depth', () => {
    const paths = flattenScope(data).map((e) => e.path);
    expect(paths).toContain('queries.v.data.rows.0.code');
    expect(paths).toContain('vars.entity');
    expect(flattenScope(data, 1).map((e) => e.path)).toEqual(['vars', 'queries']);
  });
  it('shows each query as data, loading and error', () => {
    const s = browsableScope({ vars: { a: 1 }, route: { id: 'p1' }, queries: { q: { data: [1], isLoading: false, error: new Error('boom') } } });
    expect(s).toEqual({ vars: { a: 1 }, route: { id: 'p1' }, queries: { q: { data: [1], isLoading: false, error: 'boom' } } });
  });
});

function Harness() {
  const [value, setValue] = useState<unknown>('');
  const [when, setWhen] = useState<ConditionNode | undefined>({ type: 'condition', field: '', operator: 'is_true' });
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><MemoryRouter>
      <AppRuntimeProvider mode="design" app={{ variables: [{ name: 'entity', default: 'product' }], queries: [{ id: 'vendors', operation: 'test.vendors', params: {} }] }}>
        <BindingField label="Value" value={value} onChange={setValue} paths={['vars.entity', 'row.code', 'row.display_name']} />
        <ConditionEditor label="Show when" value={when} onChange={setWhen} paths={['vars.entity']} />
        <div data-testid="value">{String(value)}</div>
        <div data-testid="when">{JSON.stringify(when)}</div>
      </AppRuntimeProvider>
    </MemoryRouter></QueryClientProvider>
  );
}

describe('binding picker in a field', () => {
  it('browses live query data and inserts the picked path as a binding', async () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole('button', { name: 'Browse data for Value' }));
    const panel = await screen.findByRole('presentation');
    // What this setting can read here, then the page's live state.
    expect(within(panel).getByRole('button', { name: 'Use row.code' })).toBeTruthy();
    // The top level starts open: the variables with their current values, and each query.
    expect(within(panel).getByRole('button', { name: 'Use vars.entity' }).textContent).toContain('product');
    // The query has loaded: its real shape is there to walk.
    fireEvent.click(await within(panel).findByRole('button', { name: 'Expand queries.vendors' }));
    fireEvent.click(await within(panel).findByRole('button', { name: 'Expand queries.vendors.data' }));
    expect(within(panel).getByRole('button', { name: 'Use queries.vendors.data.can_edit' }).textContent).toContain('yes/no');
    expect(within(panel).getByRole('button', { name: 'Use queries.vendors.data.rows' }).textContent).toContain('list (2)');
    fireEvent.click(within(panel).getByRole('button', { name: 'Use queries.vendors.data.can_edit' }));
    expect(screen.getByTestId('value').textContent).toBe('{{queries.vendors.data.can_edit}}');
  }, 30000);

  it('searches every path, and a condition takes the bare path', async () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole('button', { name: 'Browse data for the condition' }));
    const panel = await screen.findByRole('presentation');
    await within(panel).findByRole('button', { name: 'Expand queries.vendors' });
    fireEvent.change(within(panel).getByRole('textbox', { name: 'Search data' }), { target: { value: 'tier' } });
    fireEvent.click(await within(panel).findByRole('button', { name: 'Use queries.vendors.data.rows.0.meta.tier' }));
    expect(JSON.parse(screen.getByTestId('when').textContent!).field).toBe('queries.vendors.data.rows.0.meta.tier');
  }, 30000);
});
