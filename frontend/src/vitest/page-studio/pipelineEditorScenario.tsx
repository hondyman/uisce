import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { expect } from 'vitest';

/**
 * The pipeline editor scenario shared by the parity recording and test: one
 * pipeline with a step of every kind, and what the page shows and does -
 * toolbar, canvas, each step's settings, problems, preview, runs, save.
 */

const iso = '2026-09-01T10:00:00Z';

export const SPEC = {
  version: 1, error_policy: 'skip_and_log',
  nodes: [
    { id: 'file_1', type: 'file_source', label: 'Vendor file', position: { x: 0, y: 0 },
      config: { uri: 'uploads/prices.csv', format: 'csv', has_header: true, columns: [{ name: 'isin', type: 'string', nullable: false }, { name: 'px', type: 'decimal' }] } },
    { id: 'map_2', type: 'map', label: 'Map fields', position: { x: 280, y: 0 },
      config: { fields: [{ from: 'isin', to: 'instrument' }, { from: 'px', to: 'price', transform: 'lookup', lookup: { A: 'Alpha' } }] } },
    { id: 'staging_sink_3', type: 'staging_sink', label: 'Load staging', position: { x: 560, y: 0 },
      config: { table: 'staging.ff_price', source_cd: 'FACTSET', domain: 'PRICE', columns: { instrument: 'isin' } } },
    { id: 'master_4', type: 'master', label: 'Master', position: { x: 840, y: 0 }, config: { entity: 'price' } },
    { id: 'bo_source_5', type: 'bo_source', label: 'Funds', position: { x: 0, y: 200 },
      config: { bo_key: 'fund', filters: [{ field: 'aum', operator: 'greater_than', value: '1000' }, { field: 'region', operator: 'in', value: ['EU', 'US'] }, { field: 'launch', operator: 'between', value: ['2020', '2024'] }] } },
    { id: 'validate_6', type: 'validate', label: 'Check', position: { x: 280, y: 200 }, config: { required: ['aum'], unique: [] } },
    { id: 'rule_check_7', type: 'rule_check', label: 'Rules', position: { x: 560, y: 200 }, config: { bo_key: 'fund', rule_ids: ['r1'] } },
    { id: 'bo_sink_8', type: 'bo_sink', label: 'Write funds', position: { x: 840, y: 200 }, config: { bo_key: 'fund', mode: 'upsert', key_fields: ['code'] } },
    { id: 'file_sink_9', type: 'file_sink', label: 'Export', position: { x: 0, y: 400 }, config: { uri: 'exports/funds.csv', format: 'csv' } },
    { id: 'iceberg_sink_10', type: 'iceberg_sink', label: 'Archive', position: { x: 280, y: 400 }, config: { namespace: 'default', table: 'raw', partition_by: [], format: 'parquet' } },
  ],
  edges: [
    { from: 'file_1', to: 'map_2' }, { from: 'map_2', to: 'staging_sink_3' }, { from: 'staging_sink_3', to: 'master_4' },
    { from: 'bo_source_5', to: 'validate_6' }, { from: 'validate_6', to: 'rule_check_7' }, { from: 'rule_check_7', to: 'bo_sink_8' },
  ],
};

export const KINDS = [
  { type: 'file_source', label: 'Read a file', category: 'source', description: 'CSV, JSON or Parquet', available: true },
  { type: 'bo_source', label: 'Read records', category: 'source', description: 'From a business object', available: true },
  { type: 'validate', label: 'Check values', category: 'step', description: 'Required and unique', available: true },
  { type: 'rule_check', label: 'Apply rules', category: 'step', description: 'Validation rules', available: true },
  { type: 'map', label: 'Map fields', category: 'step', description: 'Rename and transform', available: true },
  { type: 'bo_sink', label: 'Write records', category: 'destination', description: 'To a business object', available: true },
  { type: 'staging_sink', label: 'Load staging table', category: 'destination', description: 'Bulk load', available: true },
  { type: 'file_sink', label: 'Export a file', category: 'destination', description: 'CSV and friends', available: true },
  { type: 'iceberg_sink', label: 'Archive to lakehouse', category: 'destination', description: 'Iceberg', available: false, unavailable_reason: 'no catalog' },
  { type: 'master', label: 'Master', category: 'destination', description: 'Golden records', available: true },
];

export const PREVIEW = {
  summary: {
    RecordsIn: 3, RecordsOut: 2, Errors: 1,
    Nodes: [
      { NodeID: 'file_1', Label: 'Vendor file', Type: 'file_source', In: 3, Out: 3, Errors: 0, Warnings: 0, Duration: 1, Status: 'ok', Err: '' },
      { NodeID: 'map_2', Label: 'Map fields', Type: 'map', In: 3, Out: 2, Errors: 1, Warnings: 0, Duration: 1, Status: 'ok', Err: '' },
    ],
    samples: {
      map_2: [{ Num: 1, Data: { instrument: 'US1', price: 10 } }, { Num: 2, Data: { instrument: 'US2', price: null } }],
      file_1: [{ Num: 1, Data: { isin: 'US1', px: '10' } }],
    },
  },
  rejects: [{ node_id: 'map_2', row: 3, reason: 'px is not a number', kind: 'error' }],
};

export const RUNS = [
  { id: 'run1', pipeline_id: 'p1', status: 'completed_with_errors', start_time: iso, records_in: 3, records_out: 2, errors: 1, errors_sample: [] },
];
export const RUN = {
  ...RUNS[0],
  steps: [{ NodeID: 'file_1', Label: 'Vendor file', Type: 'file_source', In: 3, Out: 3, Errors: 0, Warnings: 0, Duration: 1.5e9, Status: 'ok', Err: '' }],
  outputs: { mastering: [{ node_id: 'master_4', entity: 'price', load_run_id: 'l1', run_id: 'mr1', status: 'COMPLETED', records: 2, published: 2, held_for_review: 0, exceptions: 0 }] },
  errors_sample: [{ kind: 'error', node_id: 'map_2', row: 3, reason: 'px is not a number' }],
};

export interface Mocks {
  dp: Record<string, ReturnType<typeof import('vitest').vi.fn>>;
  pf: Record<string, ReturnType<typeof import('vitest').vi.fn>>;
  sched: Record<string, ReturnType<typeof import('vitest').vi.fn>>;
  mast: Record<string, ReturnType<typeof import('vitest').vi.fn>>;
}

export function arrange(m: Mocks) {
  m.dp.get.mockResolvedValue({ id: 'p1', name: 'Funds load', description: '', spec: structuredClone(SPEC), created_by: 'u', created_at: iso, last_modified_at: iso });
  m.dp.nodeTypes.mockResolvedValue(KINDS);
  m.dp.validate.mockResolvedValue({ valid: false, issues: [{ node_id: 'validate_6', message: 'Pick at least one unique field' }, { message: 'Two sources feed nothing' }] });
  m.dp.stagingTables.mockResolvedValue([{ table: 'staging.ff_price', columns: [{ name: 'isin', required: true }, { name: 'px_last' }] }]);
  m.dp.files.mockResolvedValue([{ path: 'uploads/prices.csv', bytes: 10 }, { path: 'uploads/other.json', bytes: 5 }]);
  m.dp.preview.mockResolvedValue(structuredClone(PREVIEW));
  m.dp.runs.mockResolvedValue(structuredClone(RUNS));
  m.dp.run.mockResolvedValue(structuredClone(RUN));
  m.dp.update.mockImplementation(async (_id: string, d: unknown) => ({ id: 'p1', ...(d as object) }));
  m.dp.suggestMapping.mockResolvedValue([]);
  m.pf.businessObjects.mockResolvedValue([{ id: 'b1', name: 'fund', display_name: 'Fund', description: 'Investment funds' }, { id: 'b2', name: 'price', display_name: 'Price' }]);
  m.pf.boSchema.mockImplementation(async () => ({ fields: [
    { name: 'code', displayName: 'Code', type: 'string', required: true }, { name: 'aum', displayName: 'AUM', type: 'decimal' },
    { name: 'region', displayName: 'Region', type: 'string' }, { name: 'launch', displayName: 'Launch', type: 'date' },
  ] }));
  m.pf.rules.mockResolvedValue([
    { id: 'r1', name: 'AUM positive', description: 'AUM must be above zero', severity: 'BLOCK', is_active: true, origin: 'core' },
    { id: 'r2', name: 'Region known', severity: 'WARN', is_active: true },
    { id: 'r3', name: 'Old rule', severity: 'WARN', is_active: false },
  ]);
  m.sched.list.mockResolvedValue({ schedules: [] });
  m.mast.profiles.mockResolvedValue({ profiles: [{ id: 'q', entity_cd: 'PRICE', display_name: 'Price', kind: 'TIMESERIES' }] });
}

const DT = /[A-Z][a-z]{2}\d{1,2},\d{4},\d{1,2}:\d{2}(:\d{2})?(AM|PM)|\d{1,2}\/\d{1,2}\/\d{4},\d{1,2}:\d{2}:\d{2}(AM|PM)?/g;
const sq = (x: string | null | undefined) => (x ?? '').replace(/[\s​]+/g, '').replace(DT, '<dt>');

/** Labels, input values, alerts, chips and buttons currently on the page. */
function inventory() {
  const q = (sel: string) => Array.from(document.querySelectorAll(sel)) as HTMLElement[];
  return {
    labels: q('label').map((l) => sq(l.textContent).replace(/\*$/, '')).filter(Boolean),
    inputs: q('input:not([type=file]),textarea').map((i) => {
      const x = i as HTMLInputElement;
      return x.type === 'checkbox' ? `check:${x.checked}` : sq(x.value);
    }),
    alerts: q('.MuiAlert-message').map((a) => sq(a.textContent)),
    chips: q('.MuiChip-label').map((c) => sq(c.textContent)).filter(Boolean),
    buttons: q('button').map((b) => sq(b.textContent)).filter(Boolean),
  };
}
type Inv = ReturnType<typeof inventory>;
/** What appeared: every list minus what was there before (as multisets). */
function added(before: Inv, after: Inv) {
  const minus = (a: string[], b: string[]) => {
    const left = [...b];
    return a.filter((x) => { const i = left.indexOf(x); if (i >= 0) { left.splice(i, 1); return false; } return true; });
  };
  return Object.fromEntries(Object.keys(after).map((k) => [k, minus(after[k as keyof Inv], before[k as keyof Inv]).sort()])) as Inv;
}

const node = (id: string) => document.querySelector(`[data-id="${id}"]`) as HTMLElement;
const tick = (ms = 150) => new Promise((r) => setTimeout(r, ms));

/** Runs the scenario, returning what was seen; the caller mounts the page at /data/pipelines/p1. */
export async function scenario(m: Mocks) {
  const out: Record<string, unknown> = {};
  await screen.findByText('Vendor file');
  await waitFor(() => expect(m.dp.validate).toHaveBeenCalled());
  await tick(700);
  out.nodes = Object.fromEntries(SPEC.nodes.map((n) => [n.id, sq(node(n.id)?.textContent)]));
  out.palette = ['Read a file', 'Map fields', 'Archive to lakehouse'].map((l) => {
    const b = screen.getAllByRole('button', { name: l }).find((x: HTMLElement) => x.closest('.MuiList-root'))!;
    return `${l}:${b.getAttribute('aria-disabled') === 'true' ? 'off' : 'on'}`;
  });
  out.problems = sq((await screen.findByText('Pick at least one unique field')).closest('ul, table')?.textContent);
  const base = inventory();

  // Each step's settings, as they appear when it is selected.
  const steps: Record<string, Inv> = {};
  for (const n of SPEC.nodes) {
    fireEvent.click(node(n.id));
    await screen.findByDisplayValue(n.label);
    await tick(400);
    steps[n.id] = added(base, inventory());
  }
  out.steps = steps;

  // Edit the map step: a third mapping, then save - what is sent. (The edit fixes the problems.)
  m.dp.validate.mockResolvedValue({ valid: true, issues: [] });
  fireEvent.click(node('map_2'));
  await screen.findByDisplayValue('Map fields');
  await tick(300);
  fireEvent.click(screen.getByRole('button', { name: 'Add mapping' }));
  await tick(300);
  const step = screen.getByDisplayValue('Map fields');
  fireEvent.change(step, { target: { value: 'Map it' } });
  await tick(500);
  const save = screen.getByRole('button', { name: /Save/ });
  await waitFor(() => expect((save as HTMLButtonElement).disabled).toBe(false));
  fireEvent.click(save);
  await waitFor(() => expect(m.dp.update).toHaveBeenCalled());
  const sent = m.dp.update.mock.calls[0][1] as { name: string; spec: typeof SPEC };
  out.saved = { name: sent.name, map: sent.spec.nodes.find((x) => x.id === 'map_2') };

  // Preview.
  const beforePreview = inventory();
  fireEvent.click(screen.getByRole('button', { name: /Preview/ }));
  await waitFor(() => expect(m.dp.preview).toHaveBeenCalled());
  await screen.findByText('px is not a number');
  await tick(300);
  out.preview = { ...added(beforePreview, inventory()), rejects: sq(screen.getByText('px is not a number').closest('tr')?.textContent),
    sample: sq(screen.getByText('US2').closest('tr')?.textContent) };
  const beforeRuns = inventory();

  // Runs.
  fireEvent.click(screen.getByRole('tab', { name: /Runs/ }));
  await screen.findByText(/3 read · 2 written · 1 rejected/);
  fireEvent.click(screen.getByText(/3 read · 2 written · 1 rejected/));
  await screen.findByText(/2 records · 2 published/);
  await tick(300);
  out.run = added(beforeRuns, inventory());
  return out;
}

export { within };
