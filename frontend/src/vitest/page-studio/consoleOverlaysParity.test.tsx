import React from 'react';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';

/**
 * The mastering console's price drawer, override-policy dialog and run
 * wizard, built in Page Studio from blocks (Drawer, Dialog, Form, KeyValue,
 * DataGrid, footer buttons). For the same API they show what the hand-built
 * PriceDrawer / PolicyDialog / RunDialog showed - recorded below (squashed
 * text of each) before those were retired - and do the same things.
 */

const iso = '2026-09-01T10:00:00Z';
const cands = [
  { source: 'BLOOMBERG', value: 101.25, currency: 'USD', as_of: iso, rank: 1, diff_pct: 0 },
  { source: 'REFINITIV', value: 101.3, currency: 'USD', as_of: iso, rank: 2, diff_pct: 0.05, stale: true },
  { source: 'ICE', value: 108, currency: 'USD', as_of: iso, rank: 3, diff_pct: 6.67, excluded: 'outside tolerance', note: 'check feed' },
];
const POLICY = {
  policy: { entity_cd: 'PRODUCT', mode: 'APPROVAL', approvals_required: 2, high_risk_attributes: ['base_currency'], high_risk_approvals: 3, inherited: false, updated_by: 'alice', attributes: ['name', 'base_currency', 'isin'] },
  can_edit: true,
};
const PRICE = {
  id: 'gp1', entity_id: 'g9', code: 'P-009', name: 'Widget Fund', price_type: 'NAV', date: '2026-09-01',
  versions: [
    { id: 'pv2', version: 2, value: 101.25, currency: 'USD', winner: 'BLOOMBERG', status: 'PUBLISHED', is_current: true, is_stale: false, confidence: 0.9, knowledge_at: iso,
      provenance: { price_type: 'NAV', strategy: 'SOURCE_PRIORITY', reason: 'highest ranked source with a fresh quote', ranking: ['BLOOMBERG', 'REFINITIV', 'ICE'], candidates: cands,
        threshold: { warning: 1, error: 5, critical: 10 }, controls: [{ control: 'DAY_OVER_DAY', level: 'WARNING', action: 'FLAG', pct: 1.5 }],
        prior: { date: '2026-08-31', value: 99.75 }, change_pct: 1.5 } },
    { id: 'pv1', version: 1, value: 100, currency: 'USD', winner: 'REFINITIV', status: 'SUPERSEDED', is_current: false, is_stale: false, confidence: 0.7, knowledge_at: iso,
      provenance: { price_type: 'NAV', strategy: 'SOURCE_PRIORITY', reason: 'only source', ranking: ['REFINITIV'], candidates: [cands[1]], threshold: {} } },
  ],
  exceptions: [{ id: 'x1', type: 'PRICE_VARIANCE', severity: 'WARNING', description: 'ICE disagrees by 6.67%', status: 'OPEN', detected_at: iso }],
  variances: [{ id: 'va1', source_a: 'BLOOMBERG', source_b: 'ICE', price_a: 101.25, price_b: 108, variance_pct: 6.67, severity: 'ERROR', status: 'OPEN' }],
};
const LOADS = [
  { id: 'l1', source: 'BLOOMBERG', run_ref: 'R-100', started_at: iso, received_rows: 120 },
  { id: 'l2', source: 'FACTSET', started_at: iso, received_rows: 40, mastering_run_id: 'r9', mastering_status: 'COMPLETED' },
];
const PREVIEW = {
  counts: { records: 120, valid: 118, invalid: 2, xref: 0, deterministic: 100, fuzzy: 3, review: 1, new: 14, conflicts: 0, published: 117, held_for_review: 0, unchanged: 0, exceptions: 2 },
  exceptions: [{ code: 'MISSING_ISIN', severity: 'ERROR', message: 'Row 7 has no ISIN' }, { code: 'NAME_CHANGED', severity: 'WARNING', message: 'Row 9 renamed' }],
  unmastered_fields: ['legacy_code'],
};

const api = vi.hoisted(() => ({
  profiles: vi.fn(), policy: vi.fn(), setPolicy: vi.fn(), golden: vi.fn(), prices: vi.fn(), runs: vi.fn(), exceptions: vi.fn(), candidates: vi.fn(),
  overrides: vi.fn(), priceById: vi.fn(), loads: vi.fn(), preview: vi.fn(), runToEnd: vi.fn(), proposeOverride: vi.fn(),
}));
vi.mock('../../features/mastering/api', async (orig) => {
  const real = await orig<typeof import('../../features/mastering/api')>();
  return { ...real, masteringApi: { ...real.masteringApi, ...api } };
});
const sb = vi.hoisted(() => ({ list: vi.fn() }));
vi.mock('../../features/staging-bindings/api', async (orig) => {
  const real = await orig<typeof import('../../features/staging-bindings/api')>();
  return { ...real, stagingBindingsApi: { ...real.stagingBindingsApi, ...sb } };
});

import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { masteringConsoleBlueprint } from '../../pages/page-studio/app/blueprints/masteringConsole';

let series = false;
beforeAll(loadRuleEngine, 30000);
beforeEach(() => {
  Object.values(api).forEach((f) => f.mockReset());
  api.profiles.mockImplementation(async () => ({
    profiles: [series
      ? { id: 'p2', entity_cd: 'PRICE', display_name: 'Price', bo_key: 'price', kind: 'TIMESERIES' }
      : { id: 'p1', entity_cd: 'PRODUCT', display_name: 'Product', bo_key: 'product', kind: 'RECORD' }],
  }));
  api.policy.mockResolvedValue(POLICY);
  api.setPolicy.mockResolvedValue(POLICY);
  api.golden.mockResolvedValue({ golden: [] });
  api.prices.mockResolvedValue({ prices: { date: '2026-09-01', dates: ['2026-09-01'], prices: [
    { id: 'gp1', entity_id: 'g9', code: 'P-009', name: 'Widget Fund', price_type: 'NAV', date: '2026-09-01', value: 101.25, currency: 'USD', winner: 'BLOOMBERG', sources: 3, status: 'PUBLISHED', version: 2, is_current: true, is_stale: false, updated_at: iso },
  ] } });
  api.runs.mockResolvedValue({ runs: [] });
  api.exceptions.mockResolvedValue({ exceptions: [] });
  api.candidates.mockResolvedValue({ candidates: [] });
  api.overrides.mockResolvedValue({ overrides: [] });
  api.priceById.mockResolvedValue({ price: PRICE });
  api.proposeOverride.mockResolvedValue({ override: {} });
  api.loads.mockResolvedValue({ loads: LOADS });
  api.preview.mockResolvedValue({ preview: PREVIEW });
  sb.list.mockResolvedValue({ bindings: [{ bo_key: 'product', staging_table: 'stg_product' }, { bo_key: 'price', staging_table: 'stg_price' }] });
});

async function openPolicy() {
  const b = await screen.findByRole('button', { name: /Overrides: 2 approvals/ });
  await waitFor(() => expect((b as HTMLButtonElement).disabled).toBe(false));
  fireEvent.click(b);
}

function mount() {
  const bp = masteringConsoleBlueprint();
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}><MemoryRouter>
      <RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} filterBar={bp.filterBar} components={bp.components} dataSources={[]} tenantId="t1" app={bp.app} />
    </MemoryRouter></QueryClientProvider>,
  );
}

// Dates render in the test machine's timezone; required fields add an asterisk the hand-built ones lacked.
const DT = /[A-Z][a-z]{2}\d{1,2},\d{4},\d{1,2}:\d{2}(AM|PM)/g;
const squash = (el: HTMLElement) => (el.textContent ?? '').replace(/[\s​]+/g, '').replace(DT, '<dt>').replace(/\*/g, '');
const chips = (el: HTMLElement) => Array.from(el.querySelectorAll('.MuiChip-label')).map((c) => (c.textContent ?? '').replace(/\s+/g, '')).filter(Boolean).sort();

/** What the hand-built components rendered for these fixtures. */
const HAND_BUILT = {
  policy: 'OverridepolicyHowstewardsmaychangegoldendataforthisentity-overridingavalueormergingduplicates.Everychangeisloggedwithitsreasoneitherway.Requireapproval(maker-checker)Allowdirectoverrides,withoutapprovalApprovalsrequired2ApprovalsrequiredHigh-riskattributesbase_currencyHigh-riskattributesOverridesoftheseneedmoreapprovals.Approvalsforhigh-riskattributes3Approvalsforhigh-riskattributesSetbyalice.CancelSavepolicy',
  // After the header (the studio drawer puts the title above the code line).
  price: '101.25USDPublishedVersion2Confidence90%CorrectthispriceBLOOMBERG—highestrankedsourcewithafreshquotePriorgoldenprice99.75on2026-08-31·moved1.5%EveryquoteconsideredSourcePricevsgoldenRankAsofBLOOMBERGwon101.25USD0%1<dt>REFINITIV101.3USD0.05%2<dt>staleICE108USD6.67%3<dt>excluded:outsidetolerancecheckfeedRanking:BLOOMBERG›REFINITIV›ICE·thresholdswarning1%/error5%/critical10%ControlsDay-over-daymove:1.5%(warning,flag)SourcesthatdisagreeBLOOMBERG101.25ICE1086.67%ERROROPENOpenexceptionsPRICE_VARIANCEICEdisagreesby6.67%Versions-selectonetoseeitsvaluesv2101.25USDBLOOMBERGPublished<dt>v1100USDREFINITIVSuperseded<dt>Everyversioniskeptwithwhenitwasknown:whatthepricewasbelievedtobeatanytimecanbereadback.',
  priceChips: ['Confidence90%', 'Day-over-daymove:1.5%(warning,flag)', 'ERROR', 'PRICE_VARIANCE', 'Published', 'Published', 'Superseded', 'Version2', 'stale', 'won'],
  runOpen: 'MasteraProductloadLoadLoadStagingtablestg_productStagingtableWheretheloadlanded;readthroughitsstagingbinding.CancelPreviewRun',
  runOptions: ['BLOOMBERG·R-100·<dt>120rows', 'FACTSET·l2·<dt>40rows·mastered:Completed'],
  runPreviewed: 'MasteraProductloadLoadBLOOMBERG·R-100·<dt>120rowsLoadStagingtablestg_productStagingtableWheretheloadlanded;readthroughitsstagingbinding.Whatarunwoulddo(nothingwaskept)Records120Rejected2Identifiermatch100Fuzzymatch3Toreview1New14Published117Exceptions2MISSING_ISINRow7hasnoISINNAME_CHANGEDRow9renamedBoundbutnotmastered(nogoldencolumn):legacy_codeCancelPreviewRun',
  previewChips: ['Exceptions2', 'Fuzzymatch3', 'Identifiermatch100', 'MISSING_ISIN', 'NAME_CHANGED', 'New14', 'Published117', 'Records120', 'Rejected2', 'Toreview1'],
};

describe('override policy dialog built in Page Studio', () => {
  it('shows what the hand-built dialog showed and saves the policy as numbers, without the attribute list', async () => {
    series = false;
    mount();
    await openPolicy();
    const d = await screen.findByRole('dialog');
    await within(d).findByText('base_currency');
    await waitFor(() => expect(squash(d)).toBe(HAND_BUILT.policy));

    fireEvent.mouseDown(within(d).getAllByRole('combobox')[0]);
    fireEvent.click(within(await screen.findByRole('listbox')).getByRole('option', { name: '3' }));
    fireEvent.click(within(d).getByRole('button', { name: 'Save policy' }));
    await waitFor(() => expect(api.setPolicy).toHaveBeenCalledWith('product', {
      entity_cd: 'PRODUCT', mode: 'APPROVAL', approvals_required: 3, high_risk_attributes: ['base_currency'], high_risk_approvals: 3, inherited: false, updated_by: 'alice',
    }));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  }, 30000);

  it('direct mode hides the approval fields and warns', async () => {
    series = false;
    mount();
    await openPolicy();
    const d = await screen.findByRole('dialog');
    fireEvent.click(await within(d).findByLabelText(/Allow direct overrides/));
    expect(await within(d).findByText(/take effect immediately/)).toBeTruthy();
    await waitFor(() => expect(within(d).queryByText('High-risk attributes')).toBeNull());
  }, 30000);

  it('is read-only for non-administrators', async () => {
    series = false;
    api.policy.mockResolvedValue({ ...POLICY, can_edit: false });
    mount();
    await openPolicy();
    const d = await screen.findByRole('dialog');
    expect(await within(d).findByText(/administrators/i)).toBeTruthy();
    expect(within(d).queryByRole('button', { name: 'Save policy' })).toBeNull();
    expect((within(d).getByLabelText(/Allow direct overrides/) as HTMLInputElement).disabled).toBe(true);
  }, 30000);
});

describe('golden price drawer built in Page Studio', () => {
  it('shows what the hand-built drawer showed', async () => {
    series = true;
    mount();
    fireEvent.click(await screen.findByText('Widget Fund'));
    const d = await screen.findByRole('presentation');
    await within(d).findAllByText('REFINITIV');
    expect(within(d).getByText('P-009 · NAV · 2026-09-01')).toBeTruthy();
    // The one addition: the version on show is marked "showing" (as in the record drawer).
    await waitFor(() => expect(squash(d).replace(/^WidgetFundP-009·NAV·2026-09-01/, '').replace('v2showing', 'v2')).toBe(HAND_BUILT.price));
    expect(chips(d).filter((c) => c !== 'showing')).toEqual(HAND_BUILT.priceChips);
    expect(getComputedStyle(within(d).getByText('108')).textDecoration).toMatch(/line-through/);
  }, 30000);

  it('correcting the price proposes an override of TYPE@date, numbers as numbers', async () => {
    series = true;
    mount();
    fireEvent.click(await screen.findByText('Widget Fund'));
    const d = await screen.findByRole('presentation');
    fireEvent.click(await within(d).findByRole('button', { name: 'Correct this price' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(/Needs approval from 2 people/)).toBeTruthy();
    const value = within(dialog).getByLabelText(/New value/i) as HTMLInputElement;
    expect(value.value).toBe('101.25');
    fireEvent.change(value, { target: { value: '101.5' } });
    fireEvent.change(within(dialog).getByLabelText(/Reason/i), { target: { value: 'Vendor restated' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /propose/i }));
    await waitFor(() => expect(api.proposeOverride).toHaveBeenCalledWith('price', 'gp1', { attribute: 'NAV@2026-09-01', action: 'SET', value: 101.5, reason: 'Vendor restated' }));
  }, 30000);
});

describe('run wizard built in Page Studio', () => {
  it('shows what the hand-built wizard showed, previews the chosen load, then runs it', async () => {
    series = false;
    let finish: (v: unknown) => void = () => {};
    api.runToEnd.mockImplementation((_e: string, _r: unknown, progress: (r: { stage: string }) => void) => {
      progress({ stage: 'MATCH' });
      return new Promise((res) => { finish = res; });
    });
    mount();
    const start = await screen.findByRole('button', { name: 'Master a load' });
    await waitFor(() => expect((start as HTMLButtonElement).disabled).toBe(false));
    fireEvent.click(start);
    const d = await screen.findByRole('dialog');
    await waitFor(() => expect(squash(d)).toBe(HAND_BUILT.runOpen));
    expect((within(d).getByRole('button', { name: 'Run' }) as HTMLButtonElement).disabled).toBe(true);

    fireEvent.mouseDown(within(d).getAllByRole('combobox')[0]);
    const options = within(await screen.findByRole('listbox')).getAllByRole('option');
    expect(options.map((o: HTMLElement) => squash(o))).toEqual(HAND_BUILT.runOptions);
    fireEvent.click(options[0]);
    await waitFor(() => expect((within(d).getByRole('button', { name: 'Preview' }) as HTMLButtonElement).disabled).toBe(false));
    fireEvent.click(within(d).getByRole('button', { name: 'Preview' }));
    await waitFor(() => expect(api.preview).toHaveBeenCalledWith('product', { staging_table: 'stg_product', load_run_id: 'l1', idempotency_key: undefined }));
    await waitFor(() => expect(squash(d)).toBe(HAND_BUILT.runPreviewed));
    expect(chips(d)).toEqual(HAND_BUILT.previewChips);

    fireEvent.click(within(d).getByRole('button', { name: 'Run' }));
    await waitFor(() => expect(api.runToEnd).toHaveBeenCalled());
    expect(await within(d).findByText('Running: matching')).toBeTruthy();
    finish({ run: { id: 'r1', status: 'COMPLETED', counts: { records: 120, published: 117 }, started_at: iso, trigger: 'manual', stage: 'DONE' } });
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(await screen.findByText(/Published 117/)).toBeTruthy();
  }, 30000);
});
