import React, { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert, Autocomplete, Button, Dialog, DialogActions, DialogContent, DialogTitle, FormControlLabel, MenuItem,
  Stack, Switch, TextField, ToggleButton, ToggleButtonGroup, Typography,
} from '@mui/material';
import { pipelinesApi, Schedule } from './api';

type Preset = 'weekdays' | 'daily' | 'hourly' | 'custom';
const DAYS = 'Mon Tue Wed Thu Fri Sat Sun'.split(' ');

/** Builds a 5-field cron from a preset (the server validates it). */
export function cronFor(preset: Preset, time: string, custom: string): string {
  const [h, m] = (time || '06:00').split(':').map(n => String(Number(n)));
  switch (preset) {
    case 'weekdays': return `${m} ${h} * * 1-5`;
    case 'daily': return `${m} ${h} * * *`;
    case 'hourly': return `${m} * * * *`;
    default: return custom.trim();
  }
}

/** Recognises a cron the presets can show, so an existing schedule opens on its preset. */
export function presetOf(cron: string): { preset: Preset; time: string } {
  const p = cron.trim().split(/\s+/);
  const hhmm = (h: string, m: string) => `${h.padStart(2, '0')}:${m.padStart(2, '0')}`;
  if (p.length === 5 && /^\d+$/.test(p[0])) {
    if (/^\d+$/.test(p[1]) && p[2] === '*' && p[3] === '*' && p[4] === '1-5') return { preset: 'weekdays', time: hhmm(p[1], p[0]) };
    if (/^\d+$/.test(p[1]) && p[2] === '*' && p[3] === '*' && p[4] === '*') return { preset: 'daily', time: hhmm(p[1], p[0]) };
    if (p[1] === '*' && p[2] === '*' && p[3] === '*' && p[4] === '*') return { preset: 'hourly', time: hhmm('0', p[0]) };
  }
  return { preset: 'custom', time: '06:00' };
}

const ZONES: string[] = (() => {
  try { return (Intl as unknown as { supportedValuesOf(k: string): string[] }).supportedValuesOf('timeZone'); } catch { return ['UTC']; }
})();

export function ScheduleDialog({ pipelineId, open, onClose }: { pipelineId: string; open: boolean; onClose: () => void }) {
  const qc = useQueryClient();
  const current = useQuery({ queryKey: ['dp-schedule', pipelineId], queryFn: () => pipelinesApi.schedule(pipelineId), enabled: open });
  const [enabled, setEnabled] = useState(true);
  const [preset, setPreset] = useState<Preset>('weekdays');
  const [time, setTime] = useState('06:00');
  const [custom, setCustom] = useState('0 6 * * 1-5');
  const [tz, setTz] = useState(Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC');

  useEffect(() => {
    const s = current.data?.schedule;
    if (!s) return;
    const p = presetOf(s.cron);
    setEnabled(s.enabled); setPreset(p.preset); setTime(p.time); setCustom(s.cron); setTz(s.timezone || 'UTC');
  }, [current.data]);

  const cron = useMemo(() => cronFor(preset, time, custom), [preset, time, custom]);
  const save = useMutation({
    mutationFn: (s: Schedule) => pipelinesApi.setSchedule(pipelineId, s),
    onSuccess: v => { qc.setQueryData(['dp-schedule', pipelineId], v); },
  });
  const next = save.data?.next_runs ?? (save.isIdle ? current.data?.next_runs : undefined);
  const err = save.error ? String((save.error as Error).message).replace(/^API Error: \d+ [^-]*- /, '') : null;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Run on a schedule</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <FormControlLabel control={<Switch checked={enabled} onChange={e => setEnabled(e.target.checked)} />}
            label={enabled ? 'Scheduled' : 'Not scheduled (runs only when you press Run)'} />
          {enabled && (
            <>
              <ToggleButtonGroup exclusive size="small" color="primary" value={preset} onChange={(_, v) => v && setPreset(v)}
                sx={{ '& .MuiToggleButton-root': { color: 'text.primary', textTransform: 'none' } }}>
                <ToggleButton value="weekdays">Weekdays</ToggleButton>
                <ToggleButton value="daily">Every day</ToggleButton>
                <ToggleButton value="hourly">Every hour</ToggleButton>
                <ToggleButton value="custom">Custom</ToggleButton>
              </ToggleButtonGroup>
              {preset === 'hourly' && (
                <TextField select size="small" label="At minute" value={time.split(':')[1]} onChange={e => setTime(`00:${e.target.value}`)} sx={{ width: 160 }}>
                  {['00', '05', '10', '15', '20', '30', '45'].map(m => <MenuItem key={m} value={m}>:{m}</MenuItem>)}
                </TextField>
              )}
              {(preset === 'weekdays' || preset === 'daily') && (
                <TextField size="small" type="time" label="At" value={time} onChange={e => setTime(e.target.value)} sx={{ width: 160 }} />
              )}
              {preset === 'custom' && (
                <TextField size="small" label="Cron expression" value={custom} onChange={e => setCustom(e.target.value)}
                  helperText="minute hour day-of-month month day-of-week, e.g. 0 6 1 * * = 06:00 on the 1st of each month" />
              )}
              <Autocomplete size="small" options={ZONES} value={tz} onChange={(_, v) => v && setTz(v)}
                renderInput={p => <TextField {...p} label="Time zone" />} />
              <Typography variant="caption" color="text.secondary">
                {preset !== 'custom' && `${preset === 'weekdays' ? `${DAYS.slice(0, 5).join(', ')}` : preset === 'daily' ? 'Every day' : 'Every hour'}`}
                {' '}· cron <code>{cron}</code>. If a run is still going when the next is due, that one is skipped.
              </Typography>
            </>
          )}
          {err && <Alert severity="error">{err}</Alert>}
          {save.isSuccess && <Alert severity="success">{save.data.schedule?.enabled ? 'Schedule saved.' : 'Schedule turned off.'}</Alert>}
          {next && next.length > 0 && (
            <Stack>
              <Typography variant="subtitle2">Next runs</Typography>
              {next.map(t => <Typography key={t} variant="body2">{new Date(t).toLocaleString(undefined, { timeZone: tz, dateStyle: 'full', timeStyle: 'short' })} ({tz})</Typography>)}
            </Stack>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
        <Button variant="contained" disabled={save.isPending} onClick={() => save.mutate({ cron, timezone: tz, enabled })}>Save schedule</Button>
      </DialogActions>
    </Dialog>
  );
}
