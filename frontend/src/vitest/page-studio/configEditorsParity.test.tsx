import React from 'react';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';

/**
 * The mastering configuration row editor and the staging binding editor,
 * built in Page Studio: a Dialog holding a Form whose fields come from data
 * (the table's columns) and whose field -> column map is a `map` field with
 * suggestions and added identifier / price rows. For the same API they show
 * what the hand-built ConfigRowEditor / BindingEditor showed - recorded
 * below (squashed text and inputs) before those were retired - and propose
 * exactly the same changes.
 */

const hierarchyColumns = [
  { name: 'field_group', type: 'text', required: true, key: true },
  { name: 'product_type_cd', type: 'text', required: false, key: true },
  { name: 'source', type: 'source', required: true, key: true },
  { name: 'priority', type: 'integer', required: true, key: false },
  { name: 'is_fallback', type: 'boolean', required: false, key: false },
];
const ruleColumns = [
  { name: 'rule_cd', type: 'text', required: true, key: true },
  { name: 'rule_name', type: 'text', required: true, key: false },
  { name: 'deterministic_keys', type: 'list', required: false, key: false },
  { name: 'fuzzy_keys', type: 'json', required: false, key: false },
  { name: 'threshold_auto_match', type: 'number', required: false, key: false },
  { name: 'metadata', type: 'json', required: false, key: false },
];
const goldRow = { id: 'g1', origin: 'core', inherited: true, overridden: false, pending: false, values: { field_group: 'NAME', product_type_cd: 'ALL', source: 'REFINITIV', priority: 10, is_fallback: false } };
const ownRow = { id: 'o1', origin: 'tenant', inherited: false, overridden: false, pending: false, values: { field_group: 'NAME', product_type_cd: null, source: 'FACTSET', priority: 1, is_fallback: true } };
const ruleRow = { id: 'm1', origin: 'tenant', inherited: false, overridden: false, pending: false, values: {
  rule_cd: 'NAME_FUZZY', rule_name: 'Name', deterministic_keys: ['isin', 'cusip'], fuzzy_keys: [{ field: 'name', method: 'trigram', weight: 0.7 }],
  threshold_auto_match: 0.95, metadata: { description: 'x' } } };
const sources = { can_edit: true, table: { kind: 'source_system', columns: [], rows: [
  { id: 's1', values: { code: 'FACTSET' } }, { id: 's2', values: { code: 'BLOOMBERG' } }, { id: 's3', values: { code: 'REFINITIV' } }] } };
const binding = { id: 'b9', bo_key: 'price', bo_name: 'Price', staging_table: 'staging.ff_price', fields: { instrument: 'isin', '@source_key': 'isin', 'id:ISIN': 'isin' }, origin: 'tenant', version: 1, updated_at: '2026-09-01T10:00:00Z' };

const cfg = vi.hoisted(() => ({ table: vi.fn(), changes: vi.fn(), propose: vi.fn(), decide: vi.fn() }));
vi.mock('../../features/mastering/api', async (orig) => {
  const real = await orig<typeof import('../../features/mastering/api')>();
  return {
    ...real, masteringConfigApi: cfg,
    masteringApi: { ...real.masteringApi, profiles: async () => ({ profiles: [{ id: 'p', entity_cd: 'PRODUCT', display_name: 'Product', kind: 'RECORD' }] }) },
  };
});
const sb = vi.hoisted(() => ({ list: vi.fn(), changes: vi.fn(), propose: vi.fn() }));
vi.mock('../../features/staging-bindings/api', async (orig) => {
  const real = await orig<typeof import('../../features/staging-bindings/api')>();
  return { ...real, stagingBindingsApi: { ...real.stagingBindingsApi, ...sb } };
});
const dp = vi.hoisted(() => ({ stagingTables: vi.fn(), suggestMapping: vi.fn() }));
const pf = vi.hoisted(() => ({ businessObjects: vi.fn(), boSchema: vi.fn() }));
vi.mock('../../features/data-pipelines/api', async (orig) => {
  const real = await orig<typeof import('../../features/data-pipelines/api')>();
  return { ...real, pipelinesApi: { ...real.pipelinesApi, ...dp }, platformApi: { ...real.platformApi, ...pf } };
});
const me = vi.hoisted(() => vi.fn());
vi.mock('../../features/message-catalog/api', async (orig) => {
  const real = await orig<typeof import('../../features/message-catalog/api')>();
  return { ...real, msgcatApi: { ...real.msgcatApi, me } };
});

import '../../studio-core/registerDomains';
import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { matchRulesBlueprint, sourceHierarchyBlueprint } from '../../pages/page-studio/app/blueprints/mdmConfig';
import { stagingBindingsBlueprint } from '../../pages/page-studio/app/blueprints/stagingBindings';
import type { CorePageDefinition } from '../../types/pageStudio';

beforeAll(loadRuleEngine, 30000);
beforeEach(() => {
  [...Object.values(cfg), ...Object.values(sb), ...Object.values(dp), ...Object.values(pf), me].forEach((f) => f.mockReset());
  cfg.table.mockImplementation(async (kind: string) => (kind === 'source_system' ? sources
    : kind === 'match_rule' ? { can_edit: true, table: { kind, columns: ruleColumns, rows: [ruleRow] } }
      : { can_edit: true, table: { kind, columns: hierarchyColumns, rows: [ownRow, goldRow] } }));
  cfg.changes.mockResolvedValue({ changes: [], can_decide: true });
  cfg.propose.mockResolvedValue({ change: {} });
  sb.list.mockResolvedValue({ bindings: [binding] });
  sb.changes.mockResolvedValue({ changes: [] });
  sb.propose.mockResolvedValue({ change: {} });
  me.mockResolvedValue({ user_id: 'u1' });
  dp.stagingTables.mockResolvedValue([
    { table: 'staging.ff_price', columns: [{ name: 'isin', type: 'string' }, { name: 'px_last', type: 'number' }, { name: 'px_date', type: 'date' }, { name: 'ticker', type: 'string' }] },
    { table: 'staging.bbg_price', columns: [{ name: 'id', type: 'string' }] },
  ]);
  dp.suggestMapping.mockResolvedValue([
    { from: 'isin', to: 'instrument', confidence: 0.9, reason: 'name match' },
    { from: 'px_date', to: 'price_date', confidence: 0.6, reason: 'type match' },
  ]);
  pf.businessObjects.mockResolvedValue([{ id: 'b1', name: 'price', display_name: 'Price' }, { id: 'b2', name: 'product', display_name: 'Product' }]);
  pf.boSchema.mockResolvedValue({ fields: [{ name: 'instrument', displayName: 'Instrument', type: 'string' }, { name: 'price_date', displayName: 'Price date', type: 'date' }] });
});

function mount(bp: Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'>) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}><MemoryRouter>
      <RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} filterBar={bp.filterBar} components={bp.components} dataSources={[]} tenantId="t1" app={bp.app} padded />
    </MemoryRouter></QueryClientProvider>,
  );
}
const squash = (el: HTMLElement) => (el.textContent ?? '').replace(/[\s​]+/g, '');
const inputs = (el: HTMLElement) => Array.from(el.querySelectorAll('input,textarea')).map((i) => {
  const x = i as HTMLInputElement;
  return `${x.type}:${x.value}${x.disabled ? '!' : ''}${x.checked ? '*' : ''}`;
});
const snap = (d: HTMLElement) => ({ text: squash(d), inputs: inputs(d) });

/** What the hand-built editors rendered for these fixtures. */
const TAIL = 'Reason(showntotheapprover)Reason(showntotheapprover)Nothingchangesuntilanotheradministratorapproves.CancelSendforapproval';
const HIERARCHY = (title: string, source = '') => `${title}Fieldgroup*Fieldgroup*ProducttypeProducttypeSource*${source}Source*Priority*Priority*Isfallback${TAIL}`;
const SB_TOP = 'ProposeastagingbindingNothingchangesuntilanotheradministratorapprovesyourproposal.';
const SB_KEYS = 'MasteringkeysWhatmasteringreadsfromthissourcebesidesfields:itsownrecordkey(linksare-deliveredrecordtothesamegoldenrecord),itsas-oftime,anditsidentifiers(matcheddeterministically).Sourcerecordkey@source_keySourceas-oftime@as_of';
const SB_ADD = 'IdentifiertypeIdentifiertypeAddidentifierPricetypePricetypeAcolumnholdingpricesofthistypeAddpricecolumn';
const SB_TAIL = 'ReasonReasonWhythischange-showntotheapproverandkeptintheaudittrail.CancelPropose';
const SB_PICKED = 'BusinessobjectPrice(price)BusinessobjectStagingtablestaging.ff_priceStagingtable';
const HAND_BUILT = {
  new: { text: HIERARCHY('Proposeanewrow'), inputs: ['text:', 'text:', 'text:', 'number:', 'checkbox:on', 'textarea:', 'textarea:x'] },
  edit: { text: HIERARCHY('Proposeachange', 'FACTSET'), inputs: ['text:NAME', 'text:', 'text:FACTSET', 'number:1', 'checkbox:on*', 'textarea:', 'textarea:x'] },
  override: {
    text: HIERARCHY('OverrideforyourenvironmentThisrowcomesfromthegoldcopy.Yourversionreplacesitinyourenvironmentonly;thegoldcopyisunchanged.', 'REFINITIV'),
    inputs: ['text:NAME!', 'text:ALL!', 'text:REFINITIV!', 'number:10', 'checkbox:on', 'textarea:', 'textarea:x'],
  },
  rule: {
    text: `ProposeachangeRule*Rule*Rulename*Rulename*DeterministickeysisincusipDeterministickeysTypeavalueandpressEnterFuzzykeysFieldFieldMethodMethodWeightWeightAddThresholdautomatchThresholdautomatchMetadata{"description":"x"}Metadata${TAIL}`,
    inputs: ['text:NAME_FUZZY', 'text:Name', 'text:', 'text:name', 'text:trigram', 'text:0.7', 'number:0.95', 'textarea:{\n  "description": "x"\n}', 'textarea:x', 'textarea:', 'textarea:x'],
  },
  sources: ['BLOOMBERG', 'FACTSET', 'REFINITIV'],
  bindingOpen: { text: `${SB_TOP}BusinessobjectBusinessobjectStagingtableStagingtable${SB_TAIL}`, inputs: ['text:', 'text:', 'textarea:', 'textarea:x'] },
  bindingPicked: {
    text: `${SB_TOP}${SB_PICKED}Fieldmapping(0of2bound)SuggestBusinessobjectfieldStagingcolumnInstrumentinstrument·stringPricedateprice_date·date${SB_KEYS}${SB_ADD}${SB_TAIL}`,
    inputs: ['text:price', 'text:staging.ff_price', 'text:', 'text:', 'text:', 'text:', 'text:', 'text:', 'textarea:', 'textarea:x'],
  },
  bindingSuggested: {
    text: `${SB_TOP}${SB_PICKED}Fieldmapping(2of2bound)SuggestBusinessobjectfieldStagingcolumnInstrumentinstrument·stringSuggested90%Pricedateprice_date·dateSuggested60%${SB_KEYS}${SB_ADD}${SB_TAIL}`,
    inputs: ['text:price', 'text:staging.ff_price', 'text:isin', 'text:px_date', 'text:', 'text:', 'text:', 'text:', 'textarea:', 'textarea:x'],
  },
  bindingIdentifier: `${SB_TOP}${SB_PICKED}Fieldmapping(2of2bound)SuggestBusinessobjectfieldStagingcolumnInstrumentinstrument·stringSuggested90%Pricedateprice_date·dateSuggested60%${SB_KEYS}Identifier:TICKERid:TICKERRemove${SB_ADD}${SB_TAIL}`,
  bindingEdit: {
    text: `ProposeachangetothebindingNothingchangesuntilanotheradministratorapprovesyourproposal.${SB_PICKED}Fieldmapping(1of2bound)SuggestBusinessobjectfieldStagingcolumnInstrumentinstrument·stringPricedateprice_date·date${SB_KEYS}Identifier:ISINid:ISINRemove${SB_ADD}${SB_TAIL}`,
    inputs: ['text:price!', 'text:staging.ff_price!', 'text:isin', 'text:', 'text:isin', 'text:', 'text:isin', 'text:', 'text:', 'textarea:', 'textarea:x'],
  },
};

async function openConfigEditor(bp: Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'>, seen: string, button: string) {
  mount(bp);
  await screen.findAllByText(seen);
  const b = (await screen.findAllByRole('button', { name: button }))[0];
  fireEvent.click(b);
  return screen.findByRole('dialog');
}

describe('configuration row editor built in Page Studio', () => {
  it('a new row: the form from the table\'s columns, sources from the registry, only filled values sent', async () => {
    const d = await openConfigEditor(sourceHierarchyBlueprint(), 'FACTSET', 'Propose a ranking');
    await waitFor(() => expect(snap(d)).toEqual(HAND_BUILT.new));
    fireEvent.change(within(d).getByLabelText(/Field group/), { target: { value: 'PRICE' } });
    fireEvent.mouseDown(within(d).getByLabelText(/Source/));
    const list = await screen.findByRole('listbox');
    expect(within(list).getAllByRole('option').map((o: HTMLElement) => o.textContent)).toEqual(HAND_BUILT.sources);
    fireEvent.click(within(list).getByRole('option', { name: 'BLOOMBERG' }));
    fireEvent.change(within(d).getByLabelText(/Priority/), { target: { value: '5' } });
    fireEvent.click(within(d).getByRole('button', { name: 'Send for approval' }));
    await waitFor(() => expect(cfg.propose).toHaveBeenCalledWith(
      { kind: 'source_priority', entity: 'product', action: 'upsert', values: { field_group: 'PRICE', source: 'BLOOMBERG', priority: 5 }, reason: '' }));
  }, 30000);

  it('a change to an own row sends only what changed, with the reason', async () => {
    const d = await openConfigEditor(sourceHierarchyBlueprint(), 'FACTSET', 'Propose change');
    await waitFor(() => expect(snap(d)).toEqual(HAND_BUILT.edit));
    fireEvent.change(within(d).getByLabelText(/Priority/), { target: { value: '3' } });
    fireEvent.change(within(d).getByLabelText(/Reason/), { target: { value: 'why' } });
    fireEvent.click(within(d).getByRole('button', { name: 'Send for approval' }));
    await waitFor(() => expect(cfg.propose).toHaveBeenCalledWith(
      { kind: 'source_priority', entity: 'product', action: 'upsert', target_id: 'o1', values: { priority: 3 }, reason: 'why' }));
  }, 30000);

  it('an override of a gold-copy row keeps its key read-only', async () => {
    const d = await openConfigEditor(sourceHierarchyBlueprint(), 'FACTSET', 'Override');
    await waitFor(() => expect(snap(d)).toEqual(HAND_BUILT.override));
  }, 30000);

  it('a match rule: key lists as chips, fuzzy keys as rows (numbers kept numbers), other JSON as text', async () => {
    const d = await openConfigEditor(matchRulesBlueprint(), 'NAME_FUZZY', 'Propose change');
    await waitFor(() => expect(snap(d)).toEqual(HAND_BUILT.rule));
    fireEvent.change(within(d).getAllByLabelText(/Weight/)[0], { target: { value: '0.8' } });
    fireEvent.click(within(d).getByRole('button', { name: 'Send for approval' }));
    await waitFor(() => expect(cfg.propose).toHaveBeenCalledWith(
      { kind: 'match_rule', entity: 'product', action: 'upsert', target_id: 'm1', values: { fuzzy_keys: [{ field: 'name', method: 'trigram', weight: 0.8 }] }, reason: '' }));
  }, 30000);
});

describe('staging binding editor built in Page Studio', () => {
  it('a new binding: pick object and table, suggest, add an identifier, propose the bound fields', async () => {
    mount(stagingBindingsBlueprint());
    await screen.findAllByText('staging.ff_price');
    fireEvent.click(screen.getByRole('button', { name: 'Propose binding' }));
    const d = await screen.findByRole('dialog');
    await waitFor(() => expect(snap(d)).toEqual(HAND_BUILT.bindingOpen));

    fireEvent.mouseDown(within(d).getAllByRole('combobox')[0]);
    fireEvent.click(await screen.findByRole('option', { name: 'Price (price)' }));
    fireEvent.mouseDown(within(d).getAllByRole('combobox')[1]);
    fireEvent.click(await screen.findByRole('option', { name: 'staging.ff_price' }));
    await waitFor(() => expect(snap(d)).toEqual(HAND_BUILT.bindingPicked));

    fireEvent.click(within(d).getByRole('button', { name: 'Suggest' }));
    await waitFor(() => expect(snap(d)).toEqual(HAND_BUILT.bindingSuggested));
    expect(dp.suggestMapping).toHaveBeenCalledWith(
      [{ name: 'isin', type: 'string' }, { name: 'px_last', type: 'number' }, { name: 'px_date', type: 'date' }, { name: 'ticker', type: 'string' }],
      [{ name: 'instrument', label: 'Instrument', type: 'string' }, { name: 'price_date', label: 'Price date', type: 'date' }],
    );

    fireEvent.change(within(d).getAllByLabelText(/Identifier type/)[0], { target: { value: 'ticker' } });
    fireEvent.click(within(d).getByRole('button', { name: 'Add identifier' }));
    await waitFor(() => expect(squash(d)).toBe(HAND_BUILT.bindingIdentifier));

    fireEvent.change(within(d).getByLabelText(/Reason/), { target: { value: 'new feed' } });
    fireEvent.click(within(d).getByRole('button', { name: 'Propose' }));
    // Unbound rows (the new identifier) are left out.
    await waitFor(() => expect(sb.propose).toHaveBeenCalledWith(
      { bo_key: 'price', staging_table: 'staging.ff_price', action: 'upsert', reason: 'new feed', fields: { instrument: 'isin', price_date: 'px_date' } }));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  }, 30000);

  it('a change to a binding keeps object and table, shows its mapping and identifiers, and can remove one', async () => {
    mount(stagingBindingsBlueprint());
    await screen.findAllByText('staging.ff_price');
    fireEvent.click(screen.getByRole('button', { name: 'Propose a change' }));
    const d = await screen.findByRole('dialog');
    await waitFor(() => expect(snap(d)).toEqual(HAND_BUILT.bindingEdit));
    fireEvent.click(within(d).getByRole('button', { name: 'Remove Identifier: ISIN' }));
    fireEvent.click(within(d).getByRole('button', { name: 'Propose' }));
    await waitFor(() => expect(sb.propose).toHaveBeenCalledWith(
      { bo_key: 'price', staging_table: 'staging.ff_price', action: 'upsert', reason: undefined, fields: { instrument: 'isin', '@source_key': 'isin' } }));
  }, 30000);
});
