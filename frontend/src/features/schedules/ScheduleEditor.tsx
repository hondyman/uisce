import React, { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Alert, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, FormControl, FormControlLabel,
  InputLabel, LinearProgress, MenuItem, Radio, RadioGroup, Select, Stack, Switch, TextField, Typography,
} from '@mui/material';
import { CatalogErrorAlert } from '../message-catalog/parts';
import {
  CalendarRule, fmt, presetCron, presetFrom, PresetState, Schedule, ScheduleInput, schedulesApi, Timing, TriggerMode,
} from './api';

const ZONES = [
  'UTC', 'America/New_York', 'America/Chicago', 'America/Los_Angeles', 'America/Toronto', 'Europe/London',
  'Europe/Dublin', 'Europe/Paris', 'Europe/Frankfurt', 'Europe/Zurich', 'Asia/Hong_Kong', 'Asia/Singapore',
  'Asia/Tokyo', 'Australia/Sydney',
];

interface Props {
  open: boolean;
  onClose: () => void;
  /** Edit this schedule; omit to create. */
  schedule?: Schedule;
  /** Fix the target (when embedded in a report or query editor). */
  fixedTarget?: { kind: string; ref: string; name?: string };
}

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}

export default function ScheduleEditor({ open, onClose, schedule, fixedTarget }: Props) {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const [name, setName] = useState(schedule?.name ?? fixedTarget?.name ?? '');
  const [kind, setKind] = useState(schedule?.target.kind ?? fixedTarget?.kind ?? '');
  const [ref, setRef] = useState(schedule?.target.ref ?? fixedTarget?.ref ?? '');
  const [preset, setPreset] = useState<PresetState>(
    schedule ? presetFrom(schedule.timing.cron, schedule.timing.calendar_rule) : { preset: 'weekdays', time: '18:00', weekday: 1, cron: '0 18 * * 1-5' },
  );
  const [zone, setZone] = useState(schedule?.timing.time_zone ?? Intl.DateTimeFormat().resolvedOptions().timeZone ?? 'UTC');
  const [calendar, setCalendar] = useState(schedule?.timing.calendar ?? '');
  const [rule, setRule] = useState<CalendarRule>(schedule?.timing.calendar_rule ?? 'none');
  const [bd, setBd] = useState<number>(schedule?.timing.business_day ?? 1);
  const [enabled, setEnabled] = useState(schedule?.enabled ?? true);
  const [mode, setMode] = useState<TriggerMode>(schedule?.timing.mode ?? 'timetable');
  const external = mode === 'external';

  const kinds = useQuery({ queryKey: ['sched-kinds'], queryFn: schedulesApi.kinds, enabled: open });
  const targets = useQuery({
    queryKey: ['sched-targets', kind],
    queryFn: () => schedulesApi.targets(kind),
    enabled: open && !!kind && !fixedTarget,
  });
  const calendars = useQuery({ queryKey: ['sched-calendars'], queryFn: schedulesApi.calendars, enabled: open });

  // Monthly business day N needs a calendar and its rule.
  // An external schedule can only skip a closed day (the caller waits for an answer).
  const effectiveRule: CalendarRule = external
    ? (rule === 'skip' ? 'skip' : 'none')
    : preset.preset === 'monthly_bd' ? 'business_day_of_month' : rule === 'business_day_of_month' ? 'none' : rule;
  const timing: Timing = useMemo(() => ({
    mode,
    cron: external ? '' : presetCron(preset),
    time_zone: zone,
    // Kept with rule 'none' (not applied) so switching the rule back doesn't lose it.
    calendar: calendar || undefined,
    calendar_rule: effectiveRule,
    business_day: effectiveRule === 'business_day_of_month' ? bd : undefined,
  }), [mode, external, preset, zone, calendar, effectiveRule, bd]);
  const debounced = useDebounced(timing, 350);
  const needsCalendar = effectiveRule !== 'none' && !calendar;
  const preview = useQuery({
    queryKey: ['sched-preview', debounced],
    queryFn: () => schedulesApi.preview(debounced, 8),
    enabled: open && !external && !needsCalendar && !!debounced.cron,
    retry: false,
  });

  const save = useMutation({
    mutationFn: (input: ScheduleInput) => (schedule ? schedulesApi.update(schedule.id, input) : schedulesApi.create(input)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['sched-list'] });
      onClose();
    },
  });

  const submit = () => save.mutate({ name, target: { kind, ref }, timing, enabled });
  const lang = i18n.language;
  const canSave = !!name.trim() && !!kind && !!ref && !needsCalendar && !save.isPending;

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>{schedule ? t('schedules.editor.editTitle') : t('schedules.editor.newTitle')}</DialogTitle>
      <DialogContent dividers>
        <Stack spacing={2.5}>
          <TextField label={t('schedules.editor.name')} value={name} onChange={(e) => setName(e.target.value)} />

          {!fixedTarget && (
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
              <FormControl sx={{ minWidth: 200 }}>
                <InputLabel>{t('schedules.editor.kind')}</InputLabel>
                <Select label={t('schedules.editor.kind')} value={kind} onChange={(e) => { setKind(e.target.value); setRef(''); }}>
                  {(kinds.data?.kinds ?? []).map((k) => <MenuItem key={k.kind} value={k.kind}>{t(`schedules.kinds.${k.kind}`, k.label)}</MenuItem>)}
                </Select>
              </FormControl>
              <FormControl sx={{ flex: 1 }} disabled={!kind}>
                <InputLabel>{t('schedules.editor.target')}</InputLabel>
                <Select label={t('schedules.editor.target')} value={ref} onChange={(e) => setRef(e.target.value)}>
                  {(targets.data?.targets ?? []).map((x) => <MenuItem key={x.ref} value={x.ref}>{x.name}</MenuItem>)}
                </Select>
              </FormControl>
            </Stack>
          )}
          {targets.error && <CatalogErrorAlert error={targets.error} />}

          <Box>
            <Typography variant="subtitle2" gutterBottom>{t('schedules.editor.mode')}</Typography>
            <RadioGroup row value={mode} onChange={(e) => setMode(e.target.value as TriggerMode)}>
              <FormControlLabel value="timetable" control={<Radio size="small" />} label={t('schedules.modes.timetable')} />
              <FormControlLabel value="external" control={<Radio size="small" />} label={t('schedules.modes.external')} />
            </RadioGroup>
            {external && (
              <Alert severity="info" sx={{ mt: 1 }}>
                <Typography variant="body2">{t('schedules.editor.externalHelp')}</Typography>
                {schedule && (
                  <Box component="pre" sx={{ mt: 1, mb: 0, p: 1, fontSize: 12, bgcolor: 'action.hover', borderRadius: 1, overflowX: 'auto' }}>
                    {`uisce-job run --schedule ${schedule.id} --key "$JOB_RUN_ID" --system tidal --wait`}
                  </Box>
                )}
              </Alert>
            )}
          </Box>

          <Box>
            <Typography variant="subtitle2" gutterBottom>{t('schedules.editor.when')}</Typography>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems={{ sm: 'center' }}>
              {!external && (<>
              <FormControl sx={{ minWidth: 240 }}>
                <InputLabel>{t('schedules.editor.repeat')}</InputLabel>
                <Select label={t('schedules.editor.repeat')} value={preset.preset}
                  onChange={(e) => setPreset({ ...preset, preset: e.target.value as PresetState['preset'], cron: presetCron(preset) })}>
                  {(['weekdays', 'daily', 'weekly', 'monthly_bd', 'custom'] as const).map((p) => (
                    <MenuItem key={p} value={p}>{t(`schedules.presets.${p}`)}</MenuItem>
                  ))}
                </Select>
              </FormControl>
              {preset.preset === 'custom' ? (
                <TextField sx={{ flex: 1 }} label={t('schedules.editor.cron')} value={preset.cron}
                  onChange={(e) => setPreset({ ...preset, cron: e.target.value })} helperText={t('schedules.editor.cronHelp')}
                  inputProps={{ style: { fontFamily: 'monospace' } }} />
              ) : (
                <TextField type="time" label={t('schedules.editor.time')} value={preset.time}
                  onChange={(e) => setPreset({ ...preset, time: e.target.value })} InputLabelProps={{ shrink: true }} sx={{ width: 150 }} />
              )}
              {preset.preset === 'weekly' && (
                <FormControl sx={{ minWidth: 160 }}>
                  <InputLabel>{t('schedules.editor.weekday')}</InputLabel>
                  <Select label={t('schedules.editor.weekday')} value={preset.weekday}
                    onChange={(e) => setPreset({ ...preset, weekday: Number(e.target.value) })}>
                    {[1, 2, 3, 4, 5, 6, 7].map((d) => (
                      <MenuItem key={d} value={d}>
                        {new Intl.DateTimeFormat(lang, { weekday: 'long' }).format(new Date(Date.UTC(2026, 0, 4 + d)))}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              )}
              {preset.preset === 'monthly_bd' && (
                <TextField type="number" label={t('schedules.editor.businessDay')} value={bd} sx={{ width: 160 }}
                  onChange={(e) => setBd(Number(e.target.value))} inputProps={{ min: -23, max: 23 }}
                  helperText={t('schedules.editor.businessDayHelp')} />
              )}
              </>)}
              <FormControl sx={{ minWidth: 200 }}>
                <InputLabel>{t('schedules.editor.timeZone')}</InputLabel>
                <Select label={t('schedules.editor.timeZone')} value={zone} onChange={(e) => setZone(e.target.value)}>
                  {Array.from(new Set([zone, ...ZONES])).map((z) => <MenuItem key={z} value={z}>{z}</MenuItem>)}
                </Select>
              </FormControl>
            </Stack>
          </Box>

          <Box>
            <Typography variant="subtitle2" gutterBottom>{t('schedules.editor.calendar')}</Typography>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems={{ sm: 'flex-start' }}>
              <FormControl sx={{ minWidth: 260 }}>
                <InputLabel>{t('schedules.editor.calendar')}</InputLabel>
                <Select label={t('schedules.editor.calendar')} value={calendar} onChange={(e) => setCalendar(e.target.value)}>
                  <MenuItem value="">{t('schedules.editor.noCalendar')}</MenuItem>
                  {(calendars.data?.calendars ?? []).map((c) => (
                    <MenuItem key={c.code} value={c.code}>
                      {c.code} · {c.name}{c.has_tenant_layer ? ` (${t('schedules.editor.withYourDays')})` : ''}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              {(external || preset.preset !== 'monthly_bd') && (
                <RadioGroup value={calendar ? effectiveRule : 'none'} onChange={(e) => setRule(e.target.value as CalendarRule)}>
                  {(external ? (['none', 'skip'] as const) : (['none', 'skip', 'next_business_day'] as const)).map((r) => (
                    <FormControlLabel key={r} value={r} disabled={!calendar && r !== 'none'} control={<Radio size="small" />}
                      label={t(`schedules.rules.${r}`)} />
                  ))}
                </RadioGroup>
              )}
            </Stack>
            {needsCalendar && <Alert severity="info" sx={{ mt: 1 }}>{t('schedules.editor.pickCalendar')}</Alert>}
          </Box>

          {!external && <Box>
            <Typography variant="subtitle2" gutterBottom>{t('schedules.editor.nextRuns')}</Typography>
            {preview.isFetching && <LinearProgress />}
            {preview.error && <CatalogErrorAlert error={preview.error} />}
            <Stack spacing={0.5}>
              {(preview.data?.upcoming ?? []).map((u) => (
                <Stack key={u.at} direction="row" spacing={1} alignItems="center">
                  <Chip size="small" label={t(`schedules.actions.${u.action}`)}
                    color={u.action === 'run' ? 'success' : u.action === 'wait' ? 'warning' : 'default'} variant="outlined" />
                  <Typography variant="body2" sx={{ textDecoration: u.action === 'skip' ? 'line-through' : 'none' }}>
                    {fmt(u.at, lang, zone)}
                  </Typography>
                  {u.action === 'wait' && u.runs_at && (
                    <Typography variant="body2" color="warning.main">→ {fmt(u.runs_at, lang, zone)}</Typography>
                  )}
                  {u.half_day && <Chip size="small" label={t('schedules.editor.halfDay')} />}
                  {u.reason && <Typography variant="caption" color="text.secondary">{u.reason}</Typography>}
                </Stack>
              ))}
            </Stack>
          </Box>}

          <FormControlLabel control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />}
            label={t('schedules.editor.enabled')} />
          {save.error && <CatalogErrorAlert error={save.error} />}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t('schedules.cancel')}</Button>
        <Button variant="contained" onClick={submit} disabled={!canSave}>{t('schedules.save')}</Button>
      </DialogActions>
    </Dialog>
  );
}
