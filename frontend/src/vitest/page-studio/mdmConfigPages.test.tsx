import React from 'react';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';

/**
 * The mastering configuration pages (source hierarchy, match rules, vendor
 * registry): rows as mastering reads them, the gold copy's read-only with an
 * Override, the tenant's own editable, and approvals decided by someone
 * other than the proposer.
 */

const iso = '2026-09-01T10:00:00Z';
const cfg = vi.hoisted(() => ({ table: vi.fn(), changes: vi.fn(), propose: vi.fn(), decide: vi.fn() }));
const profiles = vi.hoisted(() => vi.fn());

vi.mock('../../features/mastering/api', async (orig) => {
  const real = await orig<typeof import('../../features/mastering/api')>();
  return { ...real, masteringConfigApi: cfg, masteringApi: { ...real.masteringApi, profiles } };
});

import '../../studio-core/registerDomains';
import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { matchRulesBlueprint, sourceHierarchyBlueprint, vendorRegistryBlueprint } from '../../pages/page-studio/app/blueprints/mdmConfig';
import ConfigRowEditor from '../../features/mastering/ConfigRowEditor';
import type { CorePageDefinition } from '../../types/pageStudio';

beforeAll(loadRuleEngine, 30000);

const hierarchyColumns = [
  { name: 'field_group', type: 'text', required: true, key: true },
  { name: 'product_type_cd', type: 'text', required: false, key: true },
  { name: 'source', type: 'source', required: true, key: true },
  { name: 'priority', type: 'integer', required: true, key: false },
];
const goldRow = { id: 'g1', origin: 'core', inherited: true, overridden: false, pending: false, values: { field_group: 'NAME', product_type_cd: 'ALL', source: 'REFINITIV', priority: 10 } };
const ownRow = { id: 'o1', origin: 'tenant', inherited: false, overridden: false, pending: true, values: { field_group: 'NAME', product_type_cd: null, source: 'FACTSET', priority: 1 } };

beforeEach(() => {
  Object.values(cfg).forEach((f) => f.mockReset());
  profiles.mockResolvedValue({ profiles: [
    { id: 'p', entity_cd: 'PRODUCT', display_name: 'Product', kind: 'RECORD' },
    { id: 'q', entity_cd: 'PRICE', display_name: 'Price', kind: 'TIMESERIES' },
  ] });
  cfg.table.mockImplementation(async (kind: string) => (kind === 'source_system'
    ? { can_edit: true, table: { kind, columns: [{ name: 'code', type: 'text', required: true, key: true }, { name: 'display_name', type: 'text', required: true, key: false }],
      rows: [{ id: 's1', origin: 'core', inherited: true, overridden: false, pending: false, values: { code: 'FACTSET', display_name: 'FactSet', metadata: { description: 'Research' } } }] } }
    : kind === 'match_rule'
      ? { can_edit: false, table: { kind, columns: [{ name: 'rule_cd', type: 'text', required: true, key: true }],
        rows: [{ id: 'm1', origin: 'core', inherited: true, overridden: false, pending: false, values: {
          rule_cd: 'PRODUCT_NAME_FUZZY', rule_name: 'Name with currency', deterministic_keys: [], fuzzy_keys: [{ field: 'name', method: 'trigram', weight: 0.7 }],
          threshold_auto_match: 0.95, threshold_review: 0.8, priority: 20 } }] } }
      : { can_edit: true, table: { kind, columns: hierarchyColumns, rows: [ownRow, goldRow] } }));
  cfg.changes.mockImplementation(async (p: { status?: string }) => ({
    can_decide: true,
    changes: [{ id: 'c1', kind: 'source_priority', action: 'upsert', values: { field_group: 'NAME', source: 'FACTSET', priority: 1 }, before: null,
      reason: 'Cleaner names', status: 'pending', requested_by: 'u2', requested_by_name: 'Ann', requested_at: iso, mine: false }].filter(() => p.status === 'pending' || !p.status),
  }));
  cfg.decide.mockResolvedValue({ change: {} });
  cfg.propose.mockResolvedValue({ change: {} });
});

function mount(node: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}><MemoryRouter>{node}</MemoryRouter></QueryClientProvider>);
}
const page = (bp: Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'>) => (
  <RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} filterBar={bp.filterBar} components={bp.components} dataSources={[]} tenantId="t1" app={bp.app} padded />
);

describe('source hierarchy page', () => {
  it('shows own and core rows with the right actions', async () => {
    mount(page(sourceHierarchyBlueprint()));
    const own = (await screen.findByText('FACTSET')).closest('tr') as HTMLElement;
    const core = screen.getByText('REFINITIV').closest('tr') as HTMLElement;
    expect(within(own).getByText('Yours')).toBeTruthy();
    expect(within(own).getByText('Change pending')).toBeTruthy();
    expect(within(own).getByRole('button', { name: 'Propose change' })).toBeTruthy();
    expect(within(core).getByText('Core')).toBeTruthy();
    expect(within(core).queryByRole('button', { name: 'Propose change' })).toBeNull();
    expect(within(core).getByRole('button', { name: 'Override' })).toBeTruthy();
    // Wildcard scope reads as "All"; the loaded entity is the first profile.
    expect(within(core).getByText('All')).toBeTruthy();
    expect(cfg.table).toHaveBeenCalledWith('source_priority', 'product');
  });

  it('approvals: someone else\'s proposal is approved with the comment', async () => {
    mount(page(sourceHierarchyBlueprint()));
    const approvals = (await screen.findAllByRole('tab'))[1];
    await waitFor(() => expect(approvals.textContent).toMatch(/1/));
    fireEvent.click(approvals);
    const row = (await screen.findByText('priority: 1')).closest('tr') as HTMLElement;
    fireEvent.change(within(row).getByRole('textbox'), { target: { value: 'agreed' } });
    fireEvent.click(within(row).getByRole('button', { name: 'Approve' }));
    await waitFor(() => expect(cfg.decide).toHaveBeenCalledWith('c1', 'approve', 'agreed'));
  });
});

describe('match rules page', () => {
  it('lists record entities only and is read-only for non-administrators', async () => {
    mount(page(matchRulesBlueprint()));
    expect(await screen.findByText('PRODUCT_NAME_FUZZY')).toBeTruthy();
    expect(screen.getByText('name (trigram, 0.7)')).toBeTruthy();
    expect(screen.getByText(/Read-only: only administrators/)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Override' })).toBeNull();
  });
});

describe('vendor registry page', () => {
  it('lists the registry without an entity', async () => {
    mount(page(vendorRegistryBlueprint()));
    expect(await screen.findByText('FactSet')).toBeTruthy();
    expect(screen.getByText('Research')).toBeTruthy();
    expect(cfg.table).toHaveBeenCalledWith('source_system', undefined);
  });
});

describe('row editor', () => {
  it('an override keeps the key, sends no target, and goes for approval', async () => {
    const onProposed = vi.fn();
    mount(<ConfigRowEditor kind="source_priority" entity="product" mode="override" row={goldRow as never} onClose={() => {}} onProposed={onProposed} />);
    const group = await screen.findByLabelText(/Field group/);
    expect((group as HTMLInputElement).disabled).toBe(true);
    const priority = screen.getByLabelText(/Priority/);
    fireEvent.change(priority, { target: { value: '1' } });
    fireEvent.click(screen.getByRole('button', { name: 'Send for approval' }));
    await waitFor(() => expect(cfg.propose).toHaveBeenCalledWith(expect.objectContaining({
      kind: 'source_priority', entity: 'product', action: 'upsert', target_id: undefined,
      values: expect.objectContaining({ field_group: 'NAME', source: 'REFINITIV', priority: 1 }),
    })));
    expect(onProposed).toHaveBeenCalled();
  });
});
