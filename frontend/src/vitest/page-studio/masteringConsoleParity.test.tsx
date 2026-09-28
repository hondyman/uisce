import React from 'react';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';

/**
 * Acceptance test for the page application model: the hand-built mastering
 * console and the same console built as a Page Studio page (the blueprint)
 * render the same header, tabs, counts and cells from the same API, and
 * the studio page's actions call the same operations.
 */

const iso = '2026-09-01T10:00:00Z';
const fixtures = {
  profiles: [{ id: 'p1', entity_cd: 'PRODUCT', display_name: 'Product', bo_key: 'product', anchor_table: 't', inherited: false, is_active: true, kind: 'RECORD' }],
  policy: { policy: { mode: 'APPROVAL', approvals_required: 2, high_risk_attributes: [], high_risk_approvals: 2 }, can_edit: true },
  golden: [
    { id: 'g1', code: 'P-001', name: 'Widget', version: 3, status: 'PUBLISHED', is_current: true, dq_score: 92, identity_confidence: 0.97, sources: 2, winning_sources: { name: 'BBG', price: 'RTR' }, updated_at: iso },
    { id: 'g2', code: 'P-002', name: 'Gadget', version: 1, status: 'REVIEW', is_current: true, dq_score: 71, identity_confidence: 0.64, sources: 1, winning_sources: { name: 'RTR' }, updated_at: iso },
  ],
  runs: [{ id: 'r1', entity_cd: 'PRODUCT', idempotency_key: 'k', trigger: 'manual', status: 'COMPLETED', stage: 'done', counts: { records: 10, valid: 10, published: 8, exceptions: 2 }, started_by: 'Sam', started_at: iso }],
  exceptions: [{ id: 'x1', type: 'MISSING_REQUIRED', severity: 'ERROR', description: 'Name missing', golden_id: 'g1', detected_at: iso, status: 'OPEN' }],
  candidates: [{ id: 'c1', a: 'g1', a_code: 'P-001', a_name: 'Widget', b: 'g2', b_code: 'P-002', b_name: 'Gadget', score: 0.91, rule: 'fuzzy name' }],
  overrides: [{
    id: 'o1', golden_id: 'g1', golden_code: 'P-001', golden_name: 'Widget', attribute: 'name', action: 'SET', value: 'Widget Ltd', previous_value: 'Widget',
    reason: 'Legal name', requested_by_name: 'Sam', requested_at: iso, status: 'PENDING', mode: 'APPROVAL', approvals: 0, approvals_required: 2, mine: false, voted: false, active: false,
  }],
};

const api = vi.hoisted(() => ({
  profiles: vi.fn(), policy: vi.fn(), golden: vi.fn(), runs: vi.fn(), exceptions: vi.fn(), candidates: vi.fn(), overrides: vi.fn(),
  resolve: vi.fn(), voteOverride: vi.fn(), decide: vi.fn(), loads: vi.fn(),
}));

vi.mock('../../features/mastering/api', async (orig) => {
  const real = await orig<typeof import('../../features/mastering/api')>();
  return { ...real, masteringApi: { ...real.masteringApi, ...api } };
});
// The drawer is the domain's own component; here it only has to open with the right record.
vi.mock('../../features/mastering/GoldenDrawer', () => ({
  default: ({ id }: { id: string | null }) => (id ? <div data-testid="golden-drawer">drawer:{id}</div> : null),
}));

import MasteringPage from '../../features/mastering/MasteringPage';
import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { masteringConsoleBlueprint } from '../../pages/page-studio/app/blueprints/masteringConsole';

beforeAll(loadRuleEngine, 30000);

beforeEach(() => {
  Object.values(api).forEach((f) => f.mockReset());
  api.profiles.mockResolvedValue({ profiles: fixtures.profiles });
  api.policy.mockResolvedValue(fixtures.policy);
  api.golden.mockResolvedValue({ golden: fixtures.golden });
  api.runs.mockResolvedValue({ runs: fixtures.runs });
  api.exceptions.mockImplementation(async (_e: string, status?: string) => ({ exceptions: status ? [] : fixtures.exceptions }));
  api.candidates.mockResolvedValue({ candidates: fixtures.candidates });
  api.overrides.mockImplementation(async () => ({ overrides: fixtures.overrides }));
  api.resolve.mockResolvedValue({ ok: true });
  api.voteOverride.mockResolvedValue({ override: fixtures.overrides[0] });
});

function mount(node: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}><MemoryRouter>{node}</MemoryRouter></QueryClientProvider>);
}

function studioPage() {
  const bp = masteringConsoleBlueprint();
  return (
    <RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} filterBar={bp.filterBar} components={bp.components}
      dataSources={[]} tenantId="t1" app={bp.app} padded />
  );
}

const squash = (s: string | null | undefined) => (s ?? '').replace(/\s+/g, '');
const cellsOf = (table: HTMLElement) => Array.from(table.querySelectorAll('tbody tr')).map((tr) => Array.from(tr.querySelectorAll('td')).map((td) => squash(td.textContent)));
const tabsText = () => screen.getAllByRole('tab').map((t: HTMLElement) => squash(t.textContent));

async function snapshot() {
  await screen.findByText('P-001');
  // Counts arrive with their own queries.
  await waitFor(() => expect(tabsText().join('|')).toMatch(/Exceptions1/));
  await waitFor(() => expect(tabsText().join('|')).toMatch(/Matchreview1/));
  await waitFor(() => expect(tabsText().join('|')).toMatch(/Overrides1/));
  return {
    heading: squash(screen.getByRole('heading', { level: 5 }).textContent),
    buttons: screen.getAllByRole('button').map((b: HTMLElement) => squash(b.textContent)).filter((t: string) => /load|approv|policy/i.test(t)).sort(),
    tabs: tabsText(),
    golden: cellsOf(screen.getByRole('table')),
  };
}

describe('mastering console: hand-built vs Page Studio', () => {
  it('renders the same header, tabs with counts, and golden records', async () => {
    const hand = mount(<MasteringPage />);
    const a = await snapshot();
    hand.unmount();

    mount(studioPage());
    const b = await snapshot();

    expect(b.heading).toBe(a.heading);
    expect(b.buttons).toEqual(a.buttons);
    expect(b.tabs).toEqual(a.tabs);
    expect(b.golden).toEqual(a.golden);
    expect(b.golden).toHaveLength(2);
  }, 30000);

  it.each(['Runs', 'Exceptions', 'Match review', 'Overrides'])('renders the same %s tab', async (label) => {
    const open = async () => {
      await screen.findByText('P-001');
      fireEvent.click(screen.getAllByRole('tab').find((t: HTMLElement) => t.textContent?.startsWith(label))!);
      const table = await waitFor(() => {
        const t = screen.getByRole('table');
        expect(cellsOf(t).length).toBeGreaterThan(0);
        // Row buttons are gated by rule-engine conditions; wait for them to settle.
        return t;
      });
      await new Promise((r) => setTimeout(r, 150));
      return cellsOf(table);
    };
    const hand = mount(<MasteringPage />);
    const a = await open();
    hand.unmount();
    mount(studioPage());
    const b = await open();
    expect(b).toEqual(a);
  }, 30000);

  it('opens the record drawer from a golden row', async () => {
    mount(studioPage());
    fireEvent.click(await screen.findByText('P-002'));
    expect(await screen.findByTestId('golden-drawer')).toHaveTextContent('drawer:g2');
  }, 30000);

  it('row actions call the domain operations with the row and inline input', async () => {
    mount(studioPage());
    await screen.findByText('P-001');
    fireEvent.click(screen.getAllByRole('tab').find((t: HTMLElement) => t.textContent?.startsWith('Exceptions'))!);
    const table = await screen.findByRole('table');
    fireEvent.click(await within(table).findByRole('button', { name: /resolve/i }));
    await waitFor(() => expect(api.resolve).toHaveBeenCalledWith('product', 'x1', 'RESOLVED', undefined));

    fireEvent.click(screen.getAllByRole('tab').find((t: HTMLElement) => t.textContent?.startsWith('Overrides'))!);
    const input = await screen.findByPlaceholderText(/comment/i);
    fireEvent.change(input, { target: { value: 'checked the filing' } });
    fireEvent.click(await screen.findByRole('button', { name: /^approve$/i }));
    await waitFor(() => expect(api.voteOverride).toHaveBeenCalledWith('product', 'o1', true, 'checked the filing'));
  }, 30000);
});

describe('Page Studio editor on the blueprint', () => {
  it('opens in design mode with the header, grids and domain overlays on the canvas, and inspects a grid', async () => {
    const { default: PageEditor } = await import('../../pages/page-studio/PageEditor');
    const bp = masteringConsoleBlueprint();
    const page = { ...bp, id: '', createdAt: iso, updatedAt: iso, tenantId: 't1' };
    mount(<PageEditor page={page} onSave={() => {}} />);
    // Live data on the design canvas.
    expect(await screen.findByText('P-001')).toBeInTheDocument();
    // Overlays show as chips in design, not as open drawers.
    expect(screen.getByText('Golden record drawer · overlay')).toBeInTheDocument();
    expect(screen.queryByTestId('golden-drawer')).toBeNull();
    // Selecting the grid opens its inspector with its columns.
    fireEvent.click(screen.getByText('P-001'));
    expect(await screen.findByText('golden_grid')).toBeInTheDocument();
    expect(screen.getByText('Columns')).toBeInTheDocument();
    // The App tab lists the page's queries.
    fireEvent.click(screen.getByRole('tab', { name: /^app$/i }));
    expect((await screen.findAllByDisplayValue('exceptionsOpen')).length).toBe(1);
    expect(screen.getAllByDisplayValue('golden').length).toBeGreaterThan(0);
  }, 30000);
});
