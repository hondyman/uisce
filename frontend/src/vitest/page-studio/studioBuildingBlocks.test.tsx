import React from 'react';
import { beforeAll, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';
import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { registerOperations } from '../../studio-core/operations/registry';
import type { ComponentDefinition, PageLayout } from '../../types/pageStudio';
import type { ConditionNode, PageAppModel } from '../../pages/page-studio/app/appModel';

/**
 * The studio's building blocks, configured the way a page author would:
 * Drawer / Dialog / TabSet containers holding KeyValue, Timeline and Form
 * widgets - no code beyond registered operations.
 */

beforeAll(loadRuleEngine, 30000);

const save = vi.fn(async (p: Record<string, unknown>) => ({ ok: true, ...p }));
registerOperations([
  {
    id: 'test.record', domain: 'test', kind: 'query', label: 'A record', params: [{ name: 'id', type: 'string', required: true }],
    run: async (p) => ({
      id: p.id, code: 'P-001', name: 'Widget', status: 'PUBLISHED',
      versions: [{ id: 'v2', version: 2, note: 'Name restated', at: '2026-09-02T10:00:00Z' }, { id: 'v1', version: 1, note: 'First publish', at: '2026-09-01T10:00:00Z' }],
    }),
  },
  {
    id: 'test.sources', domain: 'test', kind: 'query', label: 'Sources', params: [],
    run: async () => [{ code: 'BLOOMBERG', name: 'Bloomberg' }, { code: 'FACTSET', name: 'FactSet' }],
  },
  { id: 'test.save', domain: 'test', kind: 'mutation', label: 'Save', params: [{ name: 'source', type: 'string' }, { name: 'priority', type: 'number' }], run: save },
]);

const cond = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });

function page(nodes: Record<string, { type: string; children?: string[]; props?: Record<string, unknown> }>, components: Record<string, Omit<ComponentDefinition, 'id'>>, app: PageAppModel) {
  const layout: PageLayout = { root: 'root', nodes: Object.fromEntries(Object.entries(nodes).map(([id, n]) => [id, { id, ...n }])) as PageLayout['nodes'] };
  const comps = Object.fromEntries(Object.entries(components).map(([id, c]) => [id, { id, ...c }]));
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <RuntimePage name="t" slug="t" tabs={[{ id: 'main', label: 'Main', layout }]} components={comps} dataSources={[]} tenantId="t1" app={{ chrome: 'none', ...app }} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('studio building blocks', () => {
  it('a button opens a Drawer holding a KeyValue and a selectable Timeline; closing clears it', async () => {
    page(
      {
        root: { type: 'Column', children: ['open_btn', 'drawer'] },
        drawer: {
          type: 'Drawer', children: ['kv', 'versions'],
          props: { title: 'Record {{queries.record.data.code}}', openWhen: cond('vars.recordId', 'is_not_empty'), onClose: [{ kind: 'setVariable', name: 'recordId', value: null }] },
        },
      },
      {
        open_btn: { type: 'ActionButton', props: { label: 'Open P-001', onClick: [{ kind: 'setVariable', name: 'recordId', value: 'g1' }] } },
        kv: { type: 'KeyValue', props: { source: '{{queries.record.data}}', items: [{ label: 'Code', value: '{{data.code}}' }, { label: 'Name', value: '{{data.name}}' }] } },
        versions: {
          type: 'Timeline', props: {
            items: '{{queries.record.data.versions}}', title: 'v{{row.version}}', subtitle: '{{row.note}}', time: '{{row.at}}',
            selectedWhen: cond('row.version', 'equals', '{{vars.version}}'), onItemClick: [{ kind: 'setVariable', name: 'version', value: '{{row.version}}' }],
          },
        },
      },
      {
        variables: [{ name: 'recordId' }, { name: 'version' }],
        queries: [{ id: 'record', operation: 'test.record', params: { id: '{{vars.recordId}}' } }],
      },
    );
    expect(screen.queryByText('Record P-001')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Open P-001' }));
    const drawer = await screen.findByRole('presentation');
    expect(await within(drawer).findByText('Record P-001')).toBeTruthy();
    expect(within(drawer).getByText('Widget')).toBeTruthy();
    expect(within(drawer).getByText('Name restated')).toBeTruthy();
    fireEvent.click(within(drawer).getByText('v1'));
    fireEvent.click(within(drawer).getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(screen.queryByText('Record P-001')).toBeNull());
  }, 30000);

  it('a Dialog with a Form (options from a query) submits through its footer button', async () => {
    page(
      {
        root: { type: 'Column', children: ['add_btn', 'dialog'] },
        dialog: {
          type: 'Dialog', children: ['form'],
          props: {
            title: 'Propose a ranking', openWhen: cond('vars.editing', 'is_true'), onClose: [{ kind: 'setVariable', name: 'editing', value: false }],
            buttons: [{
              label: 'Send', variant: 'contained', disabledWhen: cond('vars.draft.source', 'is_empty'),
              onClick: [{ kind: 'runOperation', operation: 'test.save', params: { source: '{{vars.draft.source}}', priority: '{{vars.draft.priority}}' } },
                { kind: 'setVariable', name: 'editing', value: false }],
            }],
          },
        },
      },
      {
        add_btn: { type: 'ActionButton', props: { label: 'Add', onClick: [{ kind: 'setVariable', name: 'editing', value: true }] } },
        form: {
          type: 'Form', props: {
            variable: 'draft', fields: [
              { name: 'source', label: 'Source', kind: 'select', required: true, optionsFrom: { query: 'sources', valueField: 'code', labelField: 'name' } },
              { name: 'priority', label: 'Priority', kind: 'number', default: 10 },
            ],
          },
        },
      },
      { variables: [{ name: 'editing', default: false }, { name: 'draft' }], queries: [{ id: 'sources', operation: 'test.sources', params: {} }] },
    );
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('Propose a ranking')).toBeTruthy();
    await waitFor(() => expect((within(dialog).getByRole('button', { name: 'Send' }) as HTMLButtonElement).disabled).toBe(true));
    fireEvent.mouseDown(within(dialog).getByLabelText(/Source/));
    fireEvent.click(await screen.findByRole('option', { name: 'FactSet' }));
    await waitFor(() => expect((within(dialog).getByRole('button', { name: 'Send' }) as HTMLButtonElement).disabled).toBe(false));
    fireEvent.click(within(dialog).getByRole('button', { name: 'Send' }));
    await waitFor(() => expect(save).toHaveBeenCalledWith({ source: 'FACTSET', priority: 10 }));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  }, 30000);

  it('a TabSet shows one body per tab, with a badge', async () => {
    page(
      {
        root: { type: 'Column', children: ['tabs'] },
        tabs: { type: 'TabSet', children: ['t_values', 't_versions'], props: { tabs: [{ id: 'values', label: 'Values' }, { id: 'versions', label: 'Versions', badge: '{{queries.record.data.versions.length}}' }] } },
        t_values: { type: 'Column', children: ['values_text'] },
        t_versions: { type: 'Column', children: ['versions_text'] },
      },
      {
        values_text: { type: 'AlertBanner', props: { text: 'All values' } },
        versions_text: { type: 'AlertBanner', props: { text: 'Every version' } },
      },
      { variables: [{ name: 'rid', default: 'g1' }], queries: [{ id: 'record', operation: 'test.record', params: { id: '{{vars.rid}}' } }] },
    );
    expect(await screen.findByText('All values')).toBeTruthy();
    expect(screen.queryByText('Every version')).toBeNull();
    const versions = screen.getByRole('tab', { name: /Versions/ });
    await waitFor(() => expect(versions.textContent).toMatch(/2/));
    fireEvent.click(versions);
    expect(await screen.findByText('Every version')).toBeTruthy();
  }, 30000);
});

describe('the building blocks in Page Designer', () => {
  it('shows a Drawer as an editable region with its open condition, and inspects it', async () => {
    const { default: PageEditor } = await import('../../pages/page-studio/PageEditor');
    const layout: PageLayout = {
      root: 'root', nodes: {
        root: { id: 'root', type: 'Column', children: ['drawer'] },
        drawer: {
          id: 'drawer', type: 'Drawer', children: ['kv'],
          props: { title: 'Record details', openWhen: cond('vars.recordId', 'is_not_empty'), onClose: [{ kind: 'setVariable', name: 'recordId', value: null }] },
        },
      },
    };
    const iso = '2026-09-01T10:00:00Z';
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <PageEditor onSave={() => {}} page={{
            id: '', name: 'Designer test', slug: 'designer-test', createdAt: iso, updatedAt: iso, dataSources: [],
            layout, tabs: [{ id: 'main', label: 'Main', layout }],
            components: { kv: { id: 'kv', type: 'KeyValue', props: { source: '{{vars.sample}}', items: [{ label: 'Code', value: '{{data.code}}' }] } } },
            app: { variables: [{ name: 'recordId' }, { name: 'sample', default: { code: 'P-001' } }] },
          }} />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    // Always shown on the canvas (closed at runtime), with what opens it.
    expect(await screen.findByText('Record details')).toBeTruthy();
    expect(screen.getByText(/opens when vars\.recordId is not empty/)).toBeTruthy();
    // Its content is edited in place.
    expect(screen.getByText('P-001')).toBeTruthy();
    fireEvent.click(screen.getByText('Record details'));
    expect(await screen.findByText('Opening and closing')).toBeTruthy();
    expect(screen.getByText('Footer buttons')).toBeTruthy();
  }, 30000);
});
