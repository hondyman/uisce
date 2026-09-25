import apiClient from '../../utils/apiClient';

// Types mirror backend/internal/schedule (model.go, store.go, service.go).

export type CalendarRule = 'none' | 'skip' | 'next_business_day' | 'business_day_of_month';

export interface Timing {
  cron: string;
  time_zone: string;
  calendar?: string;
  calendar_rule?: CalendarRule;
  business_day?: number;
  start_at?: string;
  end_at?: string;
}

export interface Target {
  kind: string;
  ref: string;
  params?: Record<string, unknown>;
}

export interface Schedule {
  id: string;
  name: string;
  description?: string;
  target: Target;
  timing: Timing;
  enabled: boolean;
  owner_id: string;
  version: number;
  created_by: string;
  created_at: string;
  updated_by?: string;
  updated_at: string;
}

export interface ScheduleInput {
  name: string;
  description?: string;
  target: Target;
  timing: Timing;
  enabled?: boolean;
}

export interface KindInfo {
  kind: string;
  label: string;
}

export interface TargetInfo {
  ref: string;
  name: string;
  description?: string;
}

export interface CalendarInfo {
  code: string;
  name: string;
  owner: 'core' | 'tenant';
  first_date?: string;
  last_date?: string;
  has_tenant_layer: boolean;
}

export interface Upcoming {
  at: string;
  action: 'run' | 'skip' | 'wait';
  runs_at?: string;
  reason?: string;
  half_day?: boolean;
}

export interface Run {
  id: string;
  schedule_id: string;
  schedule_name: string;
  target_kind: string;
  target_ref: string;
  trigger: 'schedule' | 'manual';
  triggered_by?: string;
  scheduled_for: string;
  started_at?: string;
  finished_at?: string;
  status: 'running' | 'succeeded' | 'failed' | 'skipped';
  skip_reason?: string;
  outcome?: { rows?: number; summary?: string; refs?: Record<string, unknown> } | null;
  error_code?: string;
  has_output: boolean;
}

export interface RenderedError {
  code: string;
  severity: string;
  message: string;
  user_action?: string;
}

const BASE = '/api/schedules';

export const schedulesApi = {
  list: () => apiClient<{ schedules: Schedule[] }>(`${BASE}/`),
  get: (id: string) => apiClient<Schedule>(`${BASE}/${id}`),
  create: (input: ScheduleInput) => apiClient<Schedule>(`${BASE}/`, { method: 'POST', body: JSON.stringify(input) }),
  update: (id: string, input: ScheduleInput) =>
    apiClient<Schedule>(`${BASE}/${id}`, { method: 'PUT', body: JSON.stringify(input) }),
  remove: (id: string) => apiClient<{ deleted: boolean }>(`${BASE}/${id}`, { method: 'DELETE' }),
  pause: (id: string) => apiClient<Schedule>(`${BASE}/${id}/pause`, { method: 'POST', body: '{}' }),
  resume: (id: string) => apiClient<Schedule>(`${BASE}/${id}/resume`, { method: 'POST', body: '{}' }),
  runNow: (id: string) => apiClient<{ started: boolean }>(`${BASE}/${id}/run`, { method: 'POST', body: '{}' }),
  kinds: () => apiClient<{ kinds: KindInfo[] }>(`${BASE}/kinds`),
  targets: (kind: string) => apiClient<{ targets: TargetInfo[] }>(`${BASE}/targets?kind=${encodeURIComponent(kind)}`),
  calendars: () => apiClient<{ calendars: CalendarInfo[] }>(`${BASE}/calendars`),
  preview: (timing: Timing, count = 8) =>
    apiClient<{ upcoming: Upcoming[] }>(`${BASE}/preview`, { method: 'POST', body: JSON.stringify({ timing, count }) }),
  runs: (scheduleId?: string, status?: string) => {
    const qs = new URLSearchParams();
    if (status) qs.set('status', status);
    const path = scheduleId ? `${BASE}/${scheduleId}/runs` : `${BASE}/runs`;
    return apiClient<{ runs: Run[] }>(`${path}?${qs}`);
  },
  run: (runId: string) => apiClient<{ run: Run; error: RenderedError | null }>(`${BASE}/runs/${runId}`),
  outputUrl: (runId: string) => `${BASE}/runs/${runId}/output`,
};

// --- timing presets -----------------------------------------------------------

export type Preset = 'weekdays' | 'daily' | 'weekly' | 'monthly_bd' | 'custom';

export interface PresetState {
  preset: Preset;
  time: string; // HH:MM
  weekday: number; // 1..7 (Mon..Sun) for weekly
  cron: string; // custom
}

/** The cron a preset produces. monthly_bd fires on weekdays; the calendar rule picks the day. */
export function presetCron(p: PresetState): string {
  const [h, m] = p.time.split(':').map((x) => Number(x) || 0);
  switch (p.preset) {
    case 'weekdays':
    case 'monthly_bd':
      return `${m} ${h} * * 1-5`;
    case 'daily':
      return `${m} ${h} * * *`;
    case 'weekly':
      return `${m} ${h} * * ${p.weekday % 7}`;
    default:
      return p.cron;
  }
}

/** Best-effort reading of a cron back into a preset, for editing. */
export function presetFrom(cron: string, rule?: CalendarRule): PresetState {
  const base: PresetState = { preset: 'custom', time: '18:00', weekday: 1, cron };
  const f = cron.trim().split(/\s+/);
  if (f.length !== 5 || !/^\d+$/.test(f[0]) || !/^\d+$/.test(f[1]) || f[2] !== '*' || f[3] !== '*') return base;
  const time = `${f[1].padStart(2, '0')}:${f[0].padStart(2, '0')}`;
  if (f[4] === '1-5') return { ...base, preset: rule === 'business_day_of_month' ? 'monthly_bd' : 'weekdays', time };
  if (f[4] === '*') return { ...base, preset: 'daily', time };
  if (/^[0-6]$/.test(f[4])) return { ...base, preset: 'weekly', time, weekday: Number(f[4]) || 7 };
  return base;
}

export function fmt(iso: string | undefined, lang: string, timeZone?: string): string {
  if (!iso) return '—';
  try {
    return new Intl.DateTimeFormat(lang, { dateStyle: 'medium', timeStyle: 'short', timeZone }).format(new Date(iso));
  } catch {
    return iso;
  }
}
