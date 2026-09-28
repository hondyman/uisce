import React from 'react';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';

/**
 * The MDM screens rebuilt as Page Studio pages (staging bindings, data
 * pipelines): each blueprint renders the domain's data through its
 * registered operations and its actions call the domain's API.
 */

const iso = '2026-09-01T10:00:00Z';

const sb = vi.hoisted(() => ({ list: vi.fn(), changes: vi.fn(), propose: vi.fn(), approve: vi.fn(), reject: vi.fn(), withdraw: vi.fn() }));
const me = vi.hoisted(() => vi.fn());
const dp = vi.hoisted(() => ({ list: vi.fn(), remove: vi.fn() }));

vi.mock('../../features/staging-bindings/api', async (orig) => {
  const real = await orig<typeof import('../../features/staging-bindings/api')>();
  return { ...real, stagingBindingsApi: sb };
});
vi.mock('../../features/message-catalog/api', async (orig) => {
  const real = await orig<typeof import('../../features/message-catalog/api')>();
  return { ...real, msgcatApi: { ...real.msgcatApi, me } };
});
vi.mock('../../features/data-pipelines/api', async (orig) => {
  const real = await orig<typeof import('../../features/data-pipelines/api')>();
  return { ...real, pipelinesApi: { ...real.pipelinesApi, ...dp } };
});
vi.mock('../../features/data-pipelines/PipelineEditorPage', () => ({ default: () => <div data-testid="pipeline-editor" /> }));

import '../../studio-core/registerDomains';
import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { stagingBindingsBlueprint } from '../../pages/page-studio/app/blueprints/stagingBindings';
import { dataPipelinesBlueprint, dataPipelineEditorBlueprint } from '../../pages/page-studio/app/blueprints/dataPipelines';
import type { CorePageDefinition } from '../../types/pageStudio';

beforeAll(loadRuleEngine, 30000);

const binding = {
  id: 'b1', tenant_id: 't', bo_key: 'product', bo_name: 'Product', staging_table: 'staging.ff_product',
  fields: { isin: 'ISIN_CD', name: 'PROD_NM' }, version: 2, origin: 'core', inherited: false, updated_at: iso,
};
const change = {
  id: 'c1', tenant_id: 't', bo_key: 'product', staging_table: 'staging.ff_product', action: 'upsert',
  before: { isin: 'ISIN_CD' }, fields: { isin: 'ISIN', name: 'PROD_NM' }, status: 'pending', reason: 'Vendor renamed',
  requested_by: 'u2', requested_by_name: 'Ann', requested_at: iso,
};

beforeEach(() => {
  Object.values(sb).forEach((f) => f.mockReset());
  Object.values(dp).forEach((f) => f.mockReset());
  sb.list.mockResolvedValue({ bindings: [binding] });
  sb.changes.mockImplementation(async (status?: string) => ({ changes: status === 'pending' ? [change] : [change] }));
  sb.approve.mockResolvedValue({ change: { ...change, status: 'applied' } });
  me.mockResolvedValue({ user_id: 'u1' });
  dp.list.mockResolvedValue([{ id: 'p1', name: 'FactSet products', spec: { nodes: [], edges: [] }, last_modified_at: iso }]);
  dp.remove.mockResolvedValue({});
});

function mount(bp: Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'>) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} filterBar={bp.filterBar} components={bp.components}
          dataSources={[]} tenantId="t1" app={bp.app} padded />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('staging bindings as a studio page', () => {
  it('lists bindings, counts approvals, and opens the editor on a binding', async () => {
    mount(stagingBindingsBlueprint());
    await screen.findByText('staging.ff_product');
    expect(screen.getByText('Product')).toBeTruthy();
    // Plural i18n label with a count from the row.
    expect(screen.getByText('2 fields')).toBeTruthy();
    await waitFor(() => expect(screen.getAllByRole('tab').map((t: HTMLElement) => t.textContent).join('|')).toMatch(/1/));

    const row = screen.getByText('staging.ff_product').closest('tr') as HTMLElement;
    fireEvent.click(within(row).getAllByRole('button')[0]);
    // The studio-built editor opens on that binding (configEditorsParity covers the editor itself).
    expect(within(await screen.findByRole('dialog')).getByText('Propose a change to the binding')).toBeTruthy();
  });

  it('shows what a proposal changes and approves it with the comment', async () => {
    mount(stagingBindingsBlueprint());
    const approvalsTab = (await screen.findAllByRole('tab'))[1];
    fireEvent.click(approvalsTab);
    // One chip per changed field: isin re-pointed, name added.
    expect(await screen.findByText('isin: ISIN_CD → ISIN')).toBeTruthy();
    expect(screen.getByText('name: PROD_NM')).toBeTruthy();
    const row = screen.getByText('isin: ISIN_CD → ISIN').closest('tr') as HTMLElement;
    fireEvent.change(within(row).getByRole('textbox'), { target: { value: 'ok' } });
    fireEvent.click(within(row).getByRole('button', { name: /approve/i }));
    await waitFor(() => expect(sb.approve).toHaveBeenCalledWith('c1', 'ok'));
  });
});

describe('data pipelines as studio pages', () => {
  it('lists pipelines with their summary', async () => {
    mount(dataPipelinesBlueprint());
    expect(await screen.findByText('FactSet products')).toBeTruthy();
    expect(screen.getByText('Empty pipeline')).toBeTruthy();
  });

  it('asks before deleting, then deletes', async () => {
    mount(dataPipelinesBlueprint());
    await screen.findByText('FactSet products');
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('Delete "FactSet products"?')).toBeTruthy();
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }));
    await waitFor(() => expect(dp.remove).toHaveBeenCalledWith('p1'));
  });

  it('the editor page places the domain editor', async () => {
    mount(dataPipelineEditorBlueprint());
    expect(await screen.findByTestId('pipeline-editor')).toBeTruthy();
  });
});
