import React from 'react';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';

/**
 * The golden-record drawer built in Page Studio from building blocks
 * (Drawer, sections of TextBlock / KeyValue / DataGrid with row detail and
 * generated columns): for the same record and API it shows exactly what the
 * hand-built GoldenDrawer showed - recorded below before that was retired.
 */

const iso = '2026-09-01T10:00:00Z';
const detail = {
  id: 'g1', code: 'P-001', name: 'Widget Ltd', selected_version: 2,
  versions: [
    { id: 'v2', version: 2, status: 'PUBLISHED', is_current: true, attributes: { name: 'Widget Ltd', base_currency: 'USD' }, winning_sources: {}, dq_score: 92, identity_confidence: 0.97, knowledge_at: iso, published_at: iso },
    { id: 'v1', version: 1, status: 'SUPERSEDED', is_current: false, attributes: { name: 'Widget', base_currency: 'USD' }, winning_sources: {}, dq_score: 80, identity_confidence: 0.9, knowledge_at: iso },
  ],
  fields: [
    { name: 'name', value: 'Widget Ltd', source: 'BLOOMBERG', confidence: 0.5 },
    { name: 'base_currency', value: 'USD', source: 'FACTSET', confidence: 1 },
  ],
  sources: [{ source: 'BLOOMBERG', source_key: 'BBG1', method: 'DETERMINISTIC', score: 1, matched_keys: '', status: 'ACTIVE', updated_at: iso }],
  identifiers: [{ type: 'ISIN', value: 'US0001', is_primary: true, source: 'BLOOMBERG' }, { type: 'CUSIP', value: '0001', is_primary: false }],
  exceptions: [],
  decisions: [
    { version: 2, field: 'name', value: 'Widget Ltd', source: 'BLOOMBERG', reason: 'SOURCE_PRIORITY: Bloomberg ranks first for NAME',
      competing: [
        { source_id: 's1', source: 'BLOOMBERG', source_key: 'BBG1', value: 'Widget Ltd', as_of: iso },
        { source_id: 's2', source: 'FACTSET', source_key: 'FS1', value: 'Widget Limited', as_of: iso, selected: false, note: 'fails the name rule' },
      ] },
  ],
  terms: { name: 'Product name', base_currency: 'Base currency' },
};

const api = vi.hoisted(() => ({
  profiles: vi.fn(), policy: vi.fn(), golden: vi.fn(), runs: vi.fn(), exceptions: vi.fn(), candidates: vi.fn(), overrides: vi.fn(),
  goldenById: vi.fn(), proposeOverride: vi.fn(),
}));
vi.mock('../../features/mastering/api', async (orig) => {
  const real = await orig<typeof import('../../features/mastering/api')>();
  return { ...real, masteringApi: { ...real.masteringApi, ...api } };
});

import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { masteringConsoleBlueprint } from '../../pages/page-studio/app/blueprints/masteringConsole';

beforeAll(loadRuleEngine, 30000);
beforeEach(() => {
  Object.values(api).forEach((f) => f.mockReset());
  api.profiles.mockResolvedValue({ profiles: [{ id: 'p1', entity_cd: 'PRODUCT', display_name: 'Product', bo_key: 'product', kind: 'RECORD' }] });
  api.policy.mockResolvedValue({ policy: { mode: 'APPROVAL', approvals_required: 2, high_risk_attributes: [], high_risk_approvals: 2 }, can_edit: true });
  api.golden.mockResolvedValue({ golden: [{ id: 'g1', code: 'P-001', name: 'Widget Ltd', version: 2, status: 'PUBLISHED', sources: 1, winning_sources: {}, updated_at: iso }] });
  api.runs.mockResolvedValue({ runs: [] });
  api.exceptions.mockResolvedValue({ exceptions: [] });
  api.candidates.mockResolvedValue({ candidates: [] });
  api.overrides.mockResolvedValue({ overrides: [] });
  api.goldenById.mockResolvedValue({ golden: detail });
  api.proposeOverride.mockResolvedValue({ override: {} });
});

function mount(node: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}><MemoryRouter>{node}</MemoryRouter></QueryClientProvider>);
}

const squash = (s: string | null | undefined) => (s ?? '').replace(/[\s​]+/g, '');
// Dates render in the test machine's timezone.
const DT = /[A-Z][a-z]{2}\d{1,2},\d{4},\d{1,2}:\d{2}(AM|PM)/g;

/** What the hand-built GoldenDrawer rendered for this record. */
const HAND_BUILT = {
  title: 'WidgetLtd',
  chips: ['100%', '50%', 'BLOOMBERG', 'CUSIP0001', 'DQ92', 'FACTSET', 'ISINUS0001·BLOOMBERG', 'Identifiermatch', 'Identity97%', 'Published', 'Published', 'Superseded', 'Version2', 'showing'],
  tables: [
    [['name', 'WidgetLtdwasWidget', 'BLOOMBERG', '50%', ''], ['base_currency', 'USD', 'FACTSET', '100%', '']],
    [['BLOOMBERG', 'BBG1', 'Identifiermatch100%', '<datetime>']],
    [['v2showing', 'Published', 'DQ92', '<datetime>'], ['v1', 'Superseded', 'DQ80', '<datetime>']],
  ],
};
/** The drawer's content: title, chips, and every table's rows as squashed cell text. */
function snapshot(drawer: HTMLElement) {
  const tables = Array.from(drawer.querySelectorAll('table')).map((t) =>
    Array.from(t.querySelectorAll(':scope > tbody > tr')).map((tr) => Array.from(tr.querySelectorAll(':scope > td')).map((td) => squash(td.textContent)))
      .filter((cells) => cells.some((c) => c !== '')).map((cells) => cells.map((c) => c.replace(DT, '<datetime>'))));
  return {
    title: squash(drawer.querySelector('h6')?.textContent),
    chips: Array.from(drawer.querySelectorAll('.MuiChip-label')).map((c) => squash(c.textContent)).filter(Boolean).sort(),
    tables,
  };
}

describe('golden record drawer built in Page Studio', () => {
  it('shows what the hand-built drawer showed: title, chips, values with sources and confidence, identifiers, sources, versions', async () => {
    const a = HAND_BUILT;

    const bp = masteringConsoleBlueprint();
    mount(<RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} filterBar={bp.filterBar} components={bp.components} dataSources={[]} tenantId="t1" app={bp.app} />);
    fireEvent.click(await screen.findByText('P-001'));
    const studioDrawer = await screen.findByRole('presentation');
    await within(studioDrawer).findByText('BBG1');
    await waitFor(() => expect(within(studioDrawer).getAllByText('US0001', { exact: false }).length).toBeGreaterThan(0));
    const b = snapshot(studioDrawer);

    expect(b.title).toBe(a.title);
    // Header, identifier, source, method and status chips.
    for (const chip of a.chips) expect(b.chips).toContain(chip);
    // Values (attribute, value, source, confidence), sources, versions - as tables, in order.
    expect(b.tables).toEqual(a.tables);
    expect(b.chips).toEqual(a.chips);
  }, 30000);

  it('the why detail lists the competing values, the excluded one struck through', async () => {
    const bp = masteringConsoleBlueprint();
    mount(<RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} filterBar={bp.filterBar} components={bp.components} dataSources={[]} tenantId="t1" app={bp.app} />);
    fireEvent.click(await screen.findByText('P-001'));
    const drawer = await screen.findByRole('presentation');
    fireEvent.click(await within(drawer).findByRole('button', { name: 'Show detail' }));
    expect(await within(drawer).findByText('SOURCE_PRIORITY: Bloomberg ranks first for NAME')).toBeTruthy();
    const excluded = within(drawer).getByText('Widget Limited');
    expect(getComputedStyle(excluded).textDecoration).toMatch(/line-through/);
    expect(within(drawer).getByText('fails the name rule')).toBeTruthy();
  }, 30000);

  it('overriding an attribute proposes it with the reason', async () => {
    const bp = masteringConsoleBlueprint();
    mount(<RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} filterBar={bp.filterBar} components={bp.components} dataSources={[]} tenantId="t1" app={bp.app} />);
    fireEvent.click(await screen.findByText('P-001'));
    const drawer = await screen.findByRole('presentation');
    const edit = (await within(drawer).findAllByRole('button', { name: /base_currency/ }))[0];
    fireEvent.click(edit);
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(/Needs approval from 2 people/)).toBeTruthy();
    const value = within(dialog).getByLabelText(/New value/i) as HTMLInputElement;
    expect(value.value).toBe('USD');
    fireEvent.change(value, { target: { value: 'EUR' } });
    fireEvent.change(within(dialog).getByLabelText(/Reason/i), { target: { value: 'Share class moved' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /propose/i }));
    await waitFor(() => expect(api.proposeOverride).toHaveBeenCalledWith('product', 'g1', { attribute: 'base_currency', action: 'SET', value: 'EUR', reason: 'Share class moved' }));
  }, 30000);
});
