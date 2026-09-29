import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { expect, vi } from 'vitest';

/**
 * The Schedules console scenario shared by the parity recording and test:
 * schedules of three kinds (a timetable with a calendar, an externally
 * triggered one, business day N), filtering, pause / run now / delete, the
 * run history with its filters and a failed run's reason, and the schedule
 * editor creating one schedule and editing another.
 */

const iso = (d: string) => `2026-09-${d}T10:00:00Z`;

export const SCHEDULES = [
  { id: 's1', name: 'Daily NAV report', target: { kind: 'report', ref: 'r1' }, enabled: true, owner_id: 'u', version: 1, created_by: 'u', created_at: iso('01'), updated_at: iso('20'),
    timing: { mode: 'timetable', cron: '0 18 * * 1-5', time_zone: 'Europe/London', calendar: 'XLON', calendar_rule: 'skip' } },
  { id: 's2', name: 'FactSet load', target: { kind: 'data_pipeline', ref: 'p1' }, enabled: false, owner_id: 'u', version: 1, created_by: 'u', created_at: iso('01'), updated_at: iso('21'),
    timing: { mode: 'external', cron: '', time_zone: 'UTC', calendar: 'XLON', calendar_rule: 'skip' } },
  { id: 's3', name: 'Month-end pack', target: { kind: 'saved_query', ref: 'q1' }, enabled: true, owner_id: 'u', version: 1, created_by: 'u', created_at: iso('01'), updated_at: iso('22'),
    timing: { mode: 'timetable', cron: '30 6 * * 1-5', time_zone: 'UTC', calendar: 'XLON', calendar_rule: 'business_day_of_month', business_day: -1 } },
];
export const RUNS = [
  { id: 'run1', schedule_id: 's1', schedule_name: 'Daily NAV report', target_kind: 'report', target_ref: 'r1', trigger: 'schedule', scheduled_for: iso('24'), status: 'succeeded', outcome: { rows: 12 }, has_output: true },
  { id: 'run2', schedule_id: 's3', schedule_name: 'Month-end pack', target_kind: 'saved_query', target_ref: 'q1', trigger: 'manual', scheduled_for: iso('25'), status: 'failed', error_code: 'SCH-9001', has_output: false },
  { id: 'run3', schedule_id: 's1', schedule_name: 'Daily NAV report', target_kind: 'report', target_ref: 'r1', trigger: 'schedule', scheduled_for: iso('26'), status: 'skipped', skip_reason: 'XLON is closed', has_output: false },
  { id: 'run4', schedule_id: 's2', schedule_name: 'FactSet load', target_kind: 'data_pipeline', target_ref: 'p1', trigger: 'external', external_system: 'tidal', external_ref: 'JOB42', idempotency_key: 'k-1', scheduled_for: iso('27'), status: 'succeeded', outcome: { summary: '26 read · 23 written' }, has_output: false },
];

export type Api = Record<string, ReturnType<typeof vi.fn>>;

export function arrange(api: Api) {
  api.list.mockResolvedValue({ schedules: structuredClone(SCHEDULES) });
  api.kinds.mockResolvedValue({ kinds: [{ kind: 'report', label: 'Report' }, { kind: 'saved_query', label: 'Saved query' }, { kind: 'data_pipeline', label: 'Data pipeline' }] });
  api.targets.mockImplementation(async (kind: string) => ({ targets: kind === 'report' ? [{ ref: 'r1', name: 'NAV report' }, { ref: 'r2', name: 'Risk report' }] : [] }));
  api.calendars.mockResolvedValue({ calendars: [
    { code: 'XLON', name: 'London Stock Exchange', owner: 'core', has_tenant_layer: true },
    { code: 'XNYS', name: 'New York Stock Exchange', owner: 'core', has_tenant_layer: false },
  ] });
  api.preview.mockResolvedValue({ upcoming: [
    { at: '2026-10-01T17:00:00Z', action: 'run' },
    { at: '2026-10-02T17:00:00Z', action: 'skip', reason: 'XLON closed' },
    { at: '2026-10-05T17:00:00Z', action: 'wait', runs_at: '2026-10-06T17:00:00Z' },
    { at: '2026-10-07T17:00:00Z', action: 'run', half_day: true },
  ] });
  api.runs.mockResolvedValue({ runs: structuredClone(RUNS) });
  api.run.mockResolvedValue({ run: RUNS[1], error: { code: 'SCH-9001', severity: 'error', message: 'The saved query no longer exists.', user_action: 'Pick another query.' } });
  api.pause.mockResolvedValue({});
  api.resume.mockResolvedValue({});
  api.runNow.mockResolvedValue({ started: true });
  api.remove.mockResolvedValue({ deleted: true });
  api.create.mockImplementation(async (input: object) => ({ id: 'new', ...input }));
  api.update.mockImplementation(async (id: string, input: object) => ({ id, ...input }));
}

const DT = /[A-Z][a-z]{2}\d{1,2},\d{4},\d{1,2}:\d{2}(AM|PM)/g;
const sq = (x: string | null | undefined) => (x ?? '').replace(/[\s​]+/g, '').replace(DT, '<dt>');
const rows = () => Array.from(document.querySelectorAll('tbody tr')).map((r) => sq(r.textContent)).filter(Boolean);
const tick = (ms = 400) => new Promise((r) => setTimeout(r, ms));
const inputs = (el: HTMLElement) => Array.from(el.querySelectorAll('input:not([type=hidden]),textarea')).map((i) => {
  const x = i as HTMLInputElement;
  return x.type === 'checkbox' || x.type === 'radio' ? `${x.type}:${x.value}:${x.checked}` : sq(x.value);
});
/** A select by its visible label (some hand-built selects carry no accessible name). */
function selectBy(root: HTMLElement, label: RegExp | string) {
  const match = (t: string) => (typeof label === 'string' ? t.replace(/\s*\*$/, '').trim() === label : label.test(t));
  const l = Array.from(root.querySelectorAll('label')).find((x) => match(x.textContent ?? ''));
  const box = l?.closest('.MuiFormControl-root, .MuiTextField-root');
  // Or a filter select showing its current choice (e.g. "All statuses").
  const combo = (box?.querySelector('[role=combobox]')
    ?? Array.from(root.querySelectorAll('[role=combobox]')).find((c) => match(c.textContent ?? ''))) as HTMLElement | null;
  if (!combo) throw new Error(`no select labelled ${String(label)}`);
  return combo;
}
async function pick(root: HTMLElement, label: RegExp | string, option: RegExp | string) {
  fireEvent.mouseDown(selectBy(root, label));
  fireEvent.click(await screen.findByRole('option', { name: option }));
  await tick(200);
}

/** A switch or checkbox by its name (MUI switches carry role switch). */
const toggles = (root: HTMLElement, name: string) => [
  ...within(root).queryAllByRole('switch', { name }), ...within(root).queryAllByRole('checkbox', { name }),
];

/** Answers a delete confirmation, whichever way the page asks. */
async function confirmDelete() {
  await tick(200);
  const dialog = screen.queryByRole('dialog');
  if (dialog) fireEvent.click(within(dialog).getByRole('button', { name: /delete|ok|confirm/i }));
}

export async function scenario(api: Api) {
  const out: Record<string, unknown> = {};
  await screen.findByText('Daily NAV report');
  await tick();
  out.list = rows();

  // Filters over what each row shows.
  const search = screen.getByPlaceholderText('Search by name, target, calendar or timing');
  fireEvent.change(search, { target: { value: 'business day' } });
  await tick(700);
  out.searched = rows();
  fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }));
  await tick(700);
  out.cleared = rows().length;

  // Pause, run now, delete.
  // Each row's Active switch (the hand-built one has no accessible name).
  const active = Array.from(document.querySelectorAll('tbody input[type=checkbox]')) as HTMLElement[];
  fireEvent.click(active[0]);
  await waitFor(() => expect(api.pause).toHaveBeenCalled());
  out.paused = api.pause.mock.calls[0][0];
  fireEvent.click(screen.getAllByRole('button', { name: 'Delete' })[1]);
  await confirmDelete();
  await waitFor(() => expect(api.remove).toHaveBeenCalled());
  out.removed = api.remove.mock.calls[0][0];
  fireEvent.click(screen.getAllByRole('button', { name: 'Run now' })[2]);
  await waitFor(() => expect(api.runNow).toHaveBeenCalled());
  out.ranNow = api.runNow.mock.calls[0][0];

  // Run history (run now shows it).
  await screen.findByText('Scheduled for');
  await tick();
  out.runs = rows();
  out.downloads = screen.queryAllByRole('button', { name: 'Download the result' }).length;
  fireEvent.click(screen.getByText(/Failed \(SCH-9001\)/));
  await tick(300);
  // Shown on clicking the row, or with the row's detail toggle.
  const failedRow = screen.getByText(/Failed \(SCH-9001\)/).closest('tr') as HTMLElement;
  const show = within(failedRow).queryByRole('button', { name: 'Show detail' });
  if (show) fireEvent.click(show);
  const reason = await screen.findByText(/The saved query no longer exists\./);
  out.failure = sq(reason.closest('.MuiAlert-message, td')?.textContent);
  await pick(document.body, 'All statuses', 'Failed');
  await waitFor(() => expect(api.runs.mock.calls.some((c) => (c[1] as { status?: string })?.status === 'failed')).toBe(true));
  out.runFilter = api.runs.mock.calls.find((c) => (c[1] as { status?: string })?.status === 'failed')?.[1];

  // The editor: a new schedule.
  fireEvent.click(screen.getByRole('tab', { name: 'Schedules' }));
  fireEvent.click(await screen.findByRole('button', { name: 'New schedule' }));
  let d = await screen.findByRole('dialog');
  await tick();
  out.newForm = { labels: (Array.from(d.querySelectorAll('label')) as HTMLElement[]).map((l) => sq(l.textContent)).filter(Boolean), inputs: inputs(d) };
  fireEvent.change(within(d).getByRole('textbox', { name: 'Name' }), { target: { value: 'Weekly risk' } });
  await pick(d, 'What to run', 'Report');
  await pick(d, 'Which one', 'Risk report');
  await pick(d, 'Repeat', 'Once a week');
  await pick(d, 'Day', /Friday/);
  fireEvent.change(within(d).getByLabelText('Time'), { target: { value: '07:15' } });
  await pick(d, 'Time zone', 'Europe/London');
  await pick(d, 'Business calendar', /XLON/);
  fireEvent.click(within(d).getByRole('radio', { name: 'On a closed day, run on the next business day' }));
  await tick(900);
  out.preview = (Array.from(d.querySelectorAll('.MuiChip-label')) as HTMLElement[]).map((c) => sq(c.textContent)).filter(Boolean);
  const saveNew = within(d).getByRole('button', { name: 'Save' });
  await waitFor(() => expect((saveNew as HTMLButtonElement).disabled).toBe(false));
  fireEvent.click(saveNew);
  await waitFor(() => expect(api.create).toHaveBeenCalled());
  out.created = api.create.mock.calls[0][0];
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());

  // Editing the externally triggered one: its trigger command shows.
  fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[1]);
  d = await screen.findByRole('dialog');
  await tick();
  out.editForm = { labels: (Array.from(d.querySelectorAll('label')) as HTMLElement[]).map((l) => sq(l.textContent)).filter(Boolean), inputs: inputs(d), command: /uisce-job run --schedule s2/.test(d.textContent ?? '') };
  fireEvent.click(toggles(d, 'Active')[0]);
  fireEvent.click(within(d).getByRole('button', { name: 'Save' }));
  await waitFor(() => expect(api.update).toHaveBeenCalled());
  out.updated = api.update.mock.calls[0];
  return out;
}
