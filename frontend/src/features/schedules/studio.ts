import i18n from '../../i18n';
import apiClient from '../../utils/apiClient';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import {
  type CalendarRule, type Preset, type PresetState, type Run, type Schedule, type Timing, type TriggerMode, fmt, presetCron, presetFrom, schedulesApi,
} from './api';

/**
 * The scheduler's Page Studio surface: the Schedules console (a core page)
 * and the schedule editor dialog every studio page with a schedulable
 * target places (blueprints/schedules.ts). The editor edits a flat draft;
 * how a draft becomes a timing - the preset's cron, the calendar rule a
 * preset or an external trigger allows - is decided here, once, for the
 * preview and the save alike.
 */

const t = (k: string, o?: Record<string, unknown>) => i18n.t(k, o) as string;
const s = (p: Record<string, unknown>, k: string) => (p[k] === undefined || p[k] === null ? '' : String(p[k]));
const lang = () => i18n.language || 'en';
const kindLabel = (kind: string) => t(`schedules.kinds.${kind}`, { defaultValue: kind });
/** A weekday's name (1 = Monday ... 7 = Sunday), in UTC so the name matches the day the cron fires. */
const dayName = (d: number) => new Intl.DateTimeFormat(lang(), { weekday: 'long', timeZone: 'UTC' }).format(new Date(Date.UTC(2026, 0, 4 + d)));

/** "Every weekday at 18:00 (Europe/London) · XLON: skip closed days". */
export function whenText(sc: Schedule): string {
  if (sc.timing.mode === 'external') {
    let ext = `${t('schedules.when.external')} (${sc.timing.time_zone})`;
    if (sc.timing.calendar && sc.timing.calendar_rule === 'skip') ext += ` · ${sc.timing.calendar}: ${t('schedules.rulesShort.skip')}`;
    return ext;
  }
  const p = presetFrom(sc.timing.cron, sc.timing.calendar_rule);
  let when = p.preset === 'custom'
    ? t('schedules.when.custom', { cron: sc.timing.cron })
    : p.preset === 'monthly_bd'
      ? t('schedules.when.monthly_bd', { n: sc.timing.business_day, time: p.time })
      : t(`schedules.when.${p.preset}`, { time: p.time, weekday: dayName(p.weekday) });
  when += ` (${sc.timing.time_zone})`;
  if (sc.timing.calendar && sc.timing.calendar_rule && sc.timing.calendar_rule !== 'none') {
    when += ` · ${sc.timing.calendar}: ${t(`schedules.rulesShort.${sc.timing.calendar_rule}`)}`;
  }
  return when;
}

// --- the editor's draft ----------------------------------------------------------------

/** What the editor's form holds. */
export interface Draft {
  id: string; name: string; kind: string; ref: string; fixed: boolean;
  mode: TriggerMode; preset: Preset; time: string; weekday: string; cron: string; bd: number; zone: string;
  calendar: string; rule: CalendarRule; rule_ext: CalendarRule; enabled: boolean; command: string;
  params?: Record<string, unknown>;
}

const presetOf = (d: Draft): PresetState => ({ preset: d.preset, time: d.time || '18:00', weekday: Number(d.weekday) || 1, cron: d.cron });

/** The rule a draft applies: business day N brings its own; an external trigger can only skip a closed day. */
function effectiveRule(d: Draft): CalendarRule {
  if (d.mode === 'external') return d.rule_ext === 'skip' ? 'skip' : 'none';
  if (d.preset === 'monthly_bd') return 'business_day_of_month';
  return d.rule === 'business_day_of_month' ? 'none' : d.rule;
}

export function timingOf(d: Draft): Timing {
  const rule = effectiveRule(d);
  return {
    mode: d.mode,
    cron: d.mode === 'external' ? '' : presetCron(presetOf(d)),
    time_zone: d.zone,
    // Kept with rule 'none' (not applied) so switching the rule back does not lose it.
    calendar: d.calendar || undefined,
    calendar_rule: rule,
    business_day: rule === 'business_day_of_month' ? Number(d.bd) : undefined,
  };
}
const needsCalendar = (d: Draft) => effectiveRule(d) !== 'none' && !d.calendar;

function draftOf(sc: Schedule | undefined, fixed?: { kind: string; ref: string; name?: string }): Draft {
  const p = sc ? presetFrom(sc.timing.cron, sc.timing.calendar_rule) : { preset: 'weekdays' as Preset, time: '18:00', weekday: 1, cron: '0 18 * * 1-5' };
  const rule = sc?.timing.calendar_rule ?? 'none';
  return {
    id: sc?.id ?? '', name: sc?.name ?? fixed?.name ?? '', kind: sc?.target.kind ?? fixed?.kind ?? '', ref: sc?.target.ref ?? fixed?.ref ?? '', fixed: !!fixed,
    mode: sc?.timing.mode ?? 'timetable', preset: p.preset, time: p.time, weekday: String(p.weekday), cron: p.cron, bd: sc?.timing.business_day ?? 1,
    zone: sc?.timing.time_zone ?? Intl.DateTimeFormat().resolvedOptions().timeZone ?? 'UTC',
    calendar: sc?.timing.calendar ?? '', rule, rule_ext: rule === 'skip' ? 'skip' : 'none', enabled: sc?.enabled ?? true,
    command: sc ? `uisce-job run --schedule ${sc.id} --key "$JOB_RUN_ID" --system tidal --wait` : '',
    params: sc?.target.params,
  };
}

const ZONES = [
  'UTC', 'America/New_York', 'America/Chicago', 'America/Los_Angeles', 'America/Toronto', 'Europe/London',
  'Europe/Dublin', 'Europe/Paris', 'Europe/Frankfurt', 'Europe/Zurich', 'Asia/Hong_Kong', 'Asia/Singapore',
  'Asia/Tokyo', 'Australia/Sydney',
];

// --- runs ------------------------------------------------------------------------------

const STATUS_COLOR: Record<Run['status'], string> = { running: 'info', succeeded: 'success', failed: 'error', skipped: 'default' };
// A failed run's reason never changes: fetched once.
const reasons = new Map<string, Promise<string>>();
function reasonOf(r: Run): Promise<string> {
  if (!reasons.has(r.id)) {
    reasons.set(r.id, schedulesApi.run(r.id).then(({ error }) => (error
      ? [error.message, error.user_action, t('schedules.runs.reference', { code: error.code, ref: r.id })].filter(Boolean).join('\n')
      : '')).catch(() => ''));
  }
  return reasons.get(r.id)!;
}

async function download(runId: string) {
  // Through apiClient so the request carries the user's credentials.
  const res = await apiClient<Response>(schedulesApi.outputUrl(runId));
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = (res.headers.get('content-disposition')?.match(/filename="([^"]+)"/)?.[1]) ?? `${runId}.csv`;
  a.click();
  URL.revokeObjectURL(url);
  return { downloaded: true };
}

const ID = { name: 'id', type: 'string' as const, required: true };

const operations: OperationDef[] = [
  {
    id: 'schedules.list', domain: 'sched-list', kind: 'query', label: 'Schedules',
    description: 'Every schedule, filtered in the browser over what each row shows (name, target, calendar, timing - in the viewer\'s language). rows: name, updated_text, kind_label, when_text, enabled.',
    params: [{ name: 'q', type: 'string' }, { name: 'kind', type: 'string' }, { name: 'state', type: 'string', description: 'active or paused' }],
    fields: [{ name: 'rows', type: 'array' }, { name: 'filtered', type: 'boolean' }, { name: 'empty_text' }],
    rowsPath: 'rows',
    rowFields: [
      { name: 'id' }, { name: 'name', label: 'Name' }, { name: 'kind_label', label: 'Runs' }, { name: 'when_text', label: 'When' },
      { name: 'enabled', label: 'Enabled', type: 'boolean' }, { name: 'updated_text', label: 'Updated' },
    ],
    run: async (p) => {
      const all = (await schedulesApi.list()).schedules;
      const q = s(p, 'q'); const kind = s(p, 'kind'); const state = s(p, 'state');
      const words = q.trim().toLocaleLowerCase(lang()).split(/\s+/).filter(Boolean);
      const rows = all.filter((sc) => {
        if (kind && sc.target.kind !== kind) return false;
        if (state && (state === 'active') !== sc.enabled) return false;
        if (!words.length) return true;
        const hay = [sc.name, sc.description, sc.target.kind, kindLabel(sc.target.kind), sc.timing.calendar, sc.timing.cron, whenText(sc)]
          .filter(Boolean).join(' ').toLocaleLowerCase(lang());
        return words.every((w) => hay.includes(w));
      }).map((sc) => ({
        ...sc, kind_label: kindLabel(sc.target.kind), when_text: whenText(sc), updated_text: t('schedules.updated', { when: fmt(sc.updated_at, lang()) }),
      }));
      const filtered = !!(q.trim() || kind || state);
      return { rows, filtered, empty_text: filtered ? t('schedules.filters.noMatches') : t('schedules.empty') };
    },
  },
  {
    id: 'schedules.kinds', domain: 'sched-list', kind: 'query', label: 'What can be scheduled',
    params: [], fields: [{ name: 'value' }, { name: 'label' }],
    run: async () => (await schedulesApi.kinds()).kinds.map((k) => ({ value: k.kind, label: t(`schedules.kinds.${k.kind}`, { defaultValue: k.label }) })),
  },
  {
    id: 'schedules.setActive', domain: 'sched-list', kind: 'mutation', label: 'Pause or resume a schedule',
    params: [ID, { name: 'active', type: 'boolean', required: true }],
    invalidates: [['sched-list'], ['sched-runs']],
    run: (p) => (p.active === true || p.active === 'true' ? schedulesApi.resume(s(p, 'id')) : schedulesApi.pause(s(p, 'id'))),
  },
  {
    id: 'schedules.runNow', domain: 'sched-list', kind: 'mutation', label: 'Run a schedule now',
    params: [ID], invalidates: [['sched-list'], ['sched-runs']],
    run: (p) => schedulesApi.runNow(s(p, 'id')),
  },
  {
    id: 'schedules.remove', domain: 'sched-list', kind: 'mutation', label: 'Delete a schedule',
    description: 'Its run history is kept.', params: [ID], invalidates: [['sched-list'], ['sched-runs']],
    run: (p) => schedulesApi.remove(s(p, 'id')),
  },
  {
    id: 'schedules.runs', domain: 'sched-runs', kind: 'query', label: 'Run history',
    description: 'Searched on the server (history is paged to the latest runs). rows: when_text, schedule_name, trigger chips, status, result_text (tone error when failed), reason (a failed run\'s explanation), has_output.',
    params: [{ name: 'q', type: 'string' }, { name: 'status', type: 'string' }, { name: 'kind', type: 'string' }, { name: 'from', type: 'string' }, { name: 'to', type: 'string' }],
    fields: [{ name: 'rows', type: 'array' }, { name: 'filtered', type: 'boolean' }, { name: 'empty_text' }],
    run: async (p) => {
      const filter = { q: s(p, 'q').trim(), status: s(p, 'status'), kind: s(p, 'kind'), from: s(p, 'from'), to: s(p, 'to') };
      const runs = (await schedulesApi.runs(undefined, filter)).runs;
      const filtered = Object.values(filter).some(Boolean);
      const rows = await Promise.all(runs.map(async (r) => ({
        ...r,
        when_text: fmt(r.scheduled_for, lang()),
        chips: [
          ...(r.trigger === 'manual' ? [{ label: t('schedules.runs.manual') }] : []),
          ...(r.trigger === 'external' ? [{ label: [r.external_system || t('schedules.runs.external'), r.external_ref].filter(Boolean).join(' · '), color: 'info', variant: 'outlined' }] : []),
        ],
        key_text: r.idempotency_key ? t('schedules.runs.key', { key: r.idempotency_key }) : '',
        status_label: t(`schedules.status.${r.status}`), status_color: STATUS_COLOR[r.status],
        result_text: r.status === 'skipped' ? r.skip_reason ?? ''
          : r.status === 'failed' ? t('schedules.runs.failedCode', { code: r.error_code })
            : r.outcome?.summary ?? (r.outcome?.rows != null ? t('schedules.runs.rows', { count: r.outcome.rows }) : ''),
        tone: r.status === 'failed' ? 'error' : '',
        reason: r.status === 'failed' ? await reasonOf(r) : '',
      })));
      return { rows, filtered, empty_text: filtered ? t('schedules.filters.noMatches') : t('schedules.runs.none') };
    },
  },
  {
    id: 'schedules.download', domain: 'sched-runs', kind: 'mutation', label: 'Download a run\'s result', invalidates: [],
    params: [ID], run: (p) => download(s(p, 'id')),
  },
  {
    id: 'schedules.editorStart', domain: 'sched-list', kind: 'query', label: 'The schedule editor\'s start',
    description: 'Edit a schedule (id), or start one - for a fixed target (kind, ref, name) or any. draft: the form; key changes when a different schedule opens; title.',
    params: [{ name: 'id', type: 'string' }, { name: 'kind', type: 'string' }, { name: 'ref', type: 'string' }, { name: 'name', type: 'string' }, { name: 'open', type: 'string' }],
    fields: [{ name: 'draft', type: 'object' }, { name: 'key' }, { name: 'title' }],
    run: async (p) => {
      let sc: Schedule | undefined;
      const fixed = s(p, 'kind') && s(p, 'ref') ? { kind: s(p, 'kind'), ref: s(p, 'ref'), name: s(p, 'name') } : undefined;
      if (s(p, 'id')) sc = await schedulesApi.get(s(p, 'id'));
      else if (fixed && fixed.ref !== 'new') sc = (await schedulesApi.list(fixed)).schedules?.[0];
      return {
        draft: draftOf(sc, fixed), key: `${sc?.id ?? 'new'}:${s(p, 'open')}`,
        title: sc ? t('schedules.editor.editTitle') : t('schedules.editor.newTitle'),
      };
    },
  },
  {
    id: 'schedules.normalize', domain: 'sched-list', kind: 'mutation', label: 'Tidy a schedule draft', invalidates: [],
    description: 'Keeps the custom timetable in step with the preset, so switching to Custom starts from the timetable shown.',
    params: [{ name: 'draft', type: 'object', required: true }],
    run: async (p) => {
      const d = p.draft as Draft;
      return d.preset === 'custom' ? d : { ...d, cron: presetCron(presetOf(d)) };
    },
  },
  {
    id: 'schedules.targets', domain: 'sched-list', kind: 'query', label: 'What a kind can run',
    params: [{ name: 'kind', type: 'string', required: true }], fields: [{ name: 'value' }, { name: 'label' }],
    run: async (p) => (await schedulesApi.targets(s(p, 'kind'))).targets.map((x) => ({ value: x.ref, label: x.name })),
  },
  {
    id: 'schedules.calendars', domain: 'sched-list', kind: 'query', label: 'Business calendars', params: [],
    fields: [{ name: 'value' }, { name: 'label' }],
    run: async () => [
      { value: '', label: t('schedules.editor.noCalendar') },
      ...(await schedulesApi.calendars()).calendars.map((c) => ({
        value: c.code, label: `${c.code} · ${c.name}${c.has_tenant_layer ? ` (${t('schedules.editor.withYourDays')})` : ''}`,
      })),
    ],
  },
  {
    id: 'schedules.choices', domain: 'sched-list', kind: 'query', label: 'The editor\'s fixed choices',
    description: 'presets, weekdays (named in the viewer\'s language), zones (with the draft\'s own), rules and external rules.',
    params: [{ name: 'zone', type: 'string' }],
    run: async (p) => ({
      presets: (['weekdays', 'daily', 'weekly', 'monthly_bd', 'custom'] as const).map((x) => ({ value: x, label: t(`schedules.presets.${x}`) })),
      weekdays: [1, 2, 3, 4, 5, 6, 7].map((d) => ({ value: String(d), label: dayName(d) })),
      zones: Array.from(new Set([s(p, 'zone'), ...ZONES].filter(Boolean))).map((z) => ({ value: z, label: z })),
    }),
  },
  {
    id: 'schedules.preview', domain: 'sched-list', kind: 'query', label: 'A draft\'s next runs',
    description: 'rows: action (Runs / Skipped / Moves), when (in the schedule\'s zone), moved_to, half_day, reason. needs_calendar: the rule needs a calendar first.',
    params: [{ name: 'draft', type: 'object' }],
    run: async (p) => {
      const d = p.draft as Draft | undefined;
      if (!d || !d.mode) return { rows: [], needs_calendar: false };
      if (needsCalendar(d)) return { rows: [], needs_calendar: true };
      if (d.mode === 'external') return { rows: [], needs_calendar: false };
      const timing = timingOf(d);
      if (!timing.cron) return { rows: [], needs_calendar: false };
      const { upcoming } = await schedulesApi.preview(timing, 8);
      return {
        needs_calendar: false,
        rows: upcoming.map((u) => ({
          id: u.at, action: u.action, action_label: t(`schedules.actions.${u.action}`),
          when: fmt(u.at, lang(), d.zone), skipped: u.action === 'skip',
          moved_to: u.action === 'wait' && u.runs_at ? `→ ${fmt(u.runs_at, lang(), d.zone)}` : '',
          half_day: !!u.half_day, reason: u.reason ?? '',
        })),
      };
    },
  },
  {
    id: 'schedules.save', domain: 'sched-list', kind: 'mutation', label: 'Save a schedule',
    description: 'Creates it, or updates the one being edited (draft.id).',
    params: [{ name: 'draft', type: 'object', required: true }],
    invalidates: [['sched-list'], ['sched-runs']],
    run: async (p) => {
      const d = p.draft as Draft;
      const params = d.params && Object.keys(d.params).length > 0 ? d.params : undefined;
      const input = { name: d.name, target: { kind: d.kind, ref: d.ref, params }, timing: timingOf(d), enabled: !!d.enabled };
      return d.id ? schedulesApi.update(d.id, input) : schedulesApi.create(input);
    },
  },
  {
    id: 'schedules.forTarget', domain: 'sched-list', kind: 'query', label: 'A target\'s schedule',
    description: 'The (at most one) schedule of a product target: scheduled (enabled), label for its button.',
    params: [{ name: 'kind', type: 'string', required: true }, { name: 'ref', type: 'string', required: true }],
    fields: [{ name: 'scheduled', type: 'boolean' }, { name: 'label' }],
    run: async (p) => {
      const ref = s(p, 'ref');
      if (!ref || ref === 'new') return { scheduled: false, label: 'Schedule' };
      const sc = (await schedulesApi.list({ kind: s(p, 'kind'), ref })).schedules?.[0];
      return { scheduled: !!sc?.enabled, label: sc?.enabled ? 'Scheduled' : 'Schedule' };
    },
  },
];

registerOperations(operations);
