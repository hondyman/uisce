import React, { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Alert, Box, Button, Chip, IconButton, InputAdornment, LinearProgress, MenuItem, Paper, Stack, Switch, Tab, Table,
  TableBody, TableCell, TableContainer, TableHead, TableRow, Tabs, TextField, Tooltip, Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import EventRepeatIcon from '@mui/icons-material/EventRepeat';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import DownloadIcon from '@mui/icons-material/Download';
import apiClient from '../../utils/apiClient';
import { CatalogErrorAlert } from '../message-catalog/parts';
import { fmt, presetFrom, RenderedError, Run, RunFilter, Schedule, schedulesApi } from './api';
import ScheduleEditor from './ScheduleEditor';

const STATUS_COLOR: Record<Run['status'], 'default' | 'success' | 'error' | 'info'> = {
  running: 'info',
  succeeded: 'success',
  failed: 'error',
  skipped: 'default',
};

/** "Every weekday at 18:00 (Europe/London), on XLON: skip closed days". */
function useWhen() {
  const { t, i18n } = useTranslation();
  const dayName = (d: number) => new Intl.DateTimeFormat(i18n.language, { weekday: 'long' }).format(new Date(Date.UTC(2026, 0, 4 + d)));
  return (s: Schedule) => {
    if (s.timing.mode === 'external') {
      let ext = `${t('schedules.when.external')} (${s.timing.time_zone})`;
      if (s.timing.calendar && s.timing.calendar_rule === 'skip') ext += ` · ${s.timing.calendar}: ${t('schedules.rulesShort.skip')}`;
      return ext;
    }
    const p = presetFrom(s.timing.cron, s.timing.calendar_rule);
    let when = p.preset === 'custom'
      ? t('schedules.when.custom', { cron: s.timing.cron })
      : p.preset === 'monthly_bd'
        ? t('schedules.when.monthly_bd', { n: s.timing.business_day, time: p.time })
        : t(`schedules.when.${p.preset}`, { time: p.time, weekday: dayName(p.weekday) });
    when += ` (${s.timing.time_zone})`;
    if (s.timing.calendar && s.timing.calendar_rule && s.timing.calendar_rule !== 'none') {
      when += ` · ${s.timing.calendar}: ${t(`schedules.rulesShort.${s.timing.calendar_rule}`)}`;
    }
    return when;
  };
}

async function download(run: Run) {
  // Through apiClient so the request carries the user's credentials.
  const res = await apiClient<Response>(schedulesApi.outputUrl(run.id));
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = (res.headers.get('content-disposition')?.match(/filename="([^"]+)"/)?.[1]) ?? `${run.id}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setV(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return v;
}

function SearchField({ value, onChange, placeholder }: { value: string; onChange: (v: string) => void; placeholder: string }) {
  const { t } = useTranslation();
  return (
    <TextField size="small" value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
      inputProps={{ 'aria-label': t('schedules.filters.search') }} sx={{ flex: 1, minWidth: 220 }}
      InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }} />
  );
}

/** Target kinds for the filter menus: the registered runners, labelled like the table chips. */
function KindSelect({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const { t } = useTranslation();
  const kinds = useQuery({ queryKey: ['sched-kinds'], queryFn: schedulesApi.kinds });
  return (
    <TextField select size="small" value={value} onChange={(e) => onChange(e.target.value)} sx={{ minWidth: 170 }}
      inputProps={{ 'aria-label': t('schedules.columns.runs') }}>
      <MenuItem value="">{t('schedules.filters.allKinds')}</MenuItem>
      {(kinds.data?.kinds ?? []).map((k) => (
        <MenuItem key={k.kind} value={k.kind}>{t(`schedules.kinds.${k.kind}`, k.label)}</MenuItem>
      ))}
    </TextField>
  );
}

const NO_FILTER: RunFilter = { q: '', status: '', kind: '', from: '', to: '' };

function RunsTable({ scheduleId }: { scheduleId?: string }) {
  const { t, i18n } = useTranslation();
  const [filter, setFilter] = useState<RunFilter>(NO_FILTER);
  const q = useDebounced(filter.q ?? '', 300);
  // Searched on the server: history is paged to the latest runs, so a
  // client-side filter would miss older matches.
  const applied = useMemo(() => ({ ...filter, q: q.trim() }), [filter, q]);
  const filtered = Object.values(applied).some(Boolean);
  const set = (k: keyof RunFilter) => (v: string) => setFilter((f) => ({ ...f, [k]: v }));
  const runs = useQuery({
    queryKey: ['sched-runs', scheduleId ?? 'all', applied],
    queryFn: () => schedulesApi.runs(scheduleId, applied),
    refetchInterval: 10000,
    placeholderData: (prev) => prev,
  });
  const [open, setOpen] = useState<string | null>(null);
  const detail = useQuery({ queryKey: ['sched-run', open], queryFn: () => schedulesApi.run(open!), enabled: !!open });
  const list = runs.data?.runs ?? [];
  const err: RenderedError | null | undefined = detail.data?.error;

  return (
    <Box>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} sx={{ mb: 2 }} flexWrap="wrap" useFlexGap>
        <SearchField value={filter.q ?? ''} onChange={set('q')} placeholder={t('schedules.filters.searchRuns')} />
        <TextField select size="small" value={filter.status} onChange={(e) => set('status')(e.target.value)} sx={{ minWidth: 150 }}
          inputProps={{ 'aria-label': t('schedules.runs.status') }}>
          <MenuItem value="">{t('schedules.filters.allStatuses')}</MenuItem>
          {(Object.keys(STATUS_COLOR) as Run['status'][]).map((st) => (
            <MenuItem key={st} value={st}>{t(`schedules.status.${st}`)}</MenuItem>
          ))}
        </TextField>
        <KindSelect value={filter.kind ?? ''} onChange={set('kind')} />
        <TextField type="date" size="small" label={t('schedules.filters.from')} value={filter.from} onChange={(e) => set('from')(e.target.value)}
          InputLabelProps={{ shrink: true }} inputProps={{ max: filter.to || undefined }} />
        <TextField type="date" size="small" label={t('schedules.filters.to')} value={filter.to} onChange={(e) => set('to')(e.target.value)}
          InputLabelProps={{ shrink: true }} inputProps={{ min: filter.from || undefined }} />
        {filtered && <Button onClick={() => setFilter(NO_FILTER)}>{t('schedules.filters.clear')}</Button>}
      </Stack>
      {runs.isFetching && <LinearProgress />}
      {runs.error && <CatalogErrorAlert error={runs.error} />}
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>{t('schedules.runs.when')}</TableCell>
              <TableCell>{t('schedules.runs.schedule')}</TableCell>
              <TableCell>{t('schedules.runs.status')}</TableCell>
              <TableCell>{t('schedules.runs.result')}</TableCell>
              <TableCell />
            </TableRow>
          </TableHead>
          <TableBody>
            {list.map((r) => (
              <React.Fragment key={r.id}>
                <TableRow hover sx={{ cursor: 'pointer' }} onClick={() => setOpen(open === r.id ? null : r.id)}>
                  <TableCell sx={{ whiteSpace: 'nowrap' }}>{fmt(r.scheduled_for, i18n.language)}</TableCell>
                  <TableCell>
                    {r.schedule_name}
                    {r.trigger === 'manual' && <Chip size="small" sx={{ ml: 1 }} label={t('schedules.runs.manual')} />}
                    {r.trigger === 'external' && (
                      <Tooltip title={r.idempotency_key ? t('schedules.runs.key', { key: r.idempotency_key }) : ''}>
                        <Chip size="small" sx={{ ml: 1 }} color="info" variant="outlined"
                          label={[r.external_system || t('schedules.runs.external'), r.external_ref].filter(Boolean).join(' · ')} />
                      </Tooltip>
                    )}
                  </TableCell>
                  <TableCell><Chip size="small" color={STATUS_COLOR[r.status]} label={t(`schedules.status.${r.status}`)} /></TableCell>
                  <TableCell>
                    <Typography variant="body2" color={r.status === 'failed' ? 'error.main' : 'text.primary'}>
                      {r.status === 'skipped' ? r.skip_reason
                        : r.status === 'failed' ? t('schedules.runs.failedCode', { code: r.error_code })
                          : r.outcome?.summary ?? (r.outcome?.rows != null ? t('schedules.runs.rows', { count: r.outcome.rows }) : '')}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    {r.has_output && (
                      <Tooltip title={t('schedules.runs.download')}>
                        <IconButton size="small" onClick={(e) => { e.stopPropagation(); void download(r); }}><DownloadIcon fontSize="small" /></IconButton>
                      </Tooltip>
                    )}
                  </TableCell>
                </TableRow>
                {open === r.id && err && (
                  <TableRow>
                    <TableCell colSpan={5}>
                      <Alert severity="error">
                        <Typography variant="body2">{err.message}</Typography>
                        {err.user_action && <Typography variant="body2" sx={{ mt: 0.5 }}>{err.user_action}</Typography>}
                        <Typography variant="caption" sx={{ opacity: 0.75 }}>{t('schedules.runs.reference', { code: err.code, ref: r.id })}</Typography>
                      </Alert>
                    </TableCell>
                  </TableRow>
                )}
              </React.Fragment>
            ))}
            {!runs.isLoading && list.length === 0 && (
              <TableRow><TableCell colSpan={5}><Typography color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>{t(filtered ? 'schedules.filters.noMatches' : 'schedules.runs.none')}</Typography></TableCell></TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}

export default function SchedulesPage() {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const when = useWhen();
  const [tab, setTab] = useState<'schedules' | 'runs'>('schedules');
  const [editing, setEditing] = useState<Schedule | 'new' | null>(null);
  const list = useQuery({ queryKey: ['sched-list'], queryFn: () => schedulesApi.list() });
  const refresh = () => {
    qc.invalidateQueries({ queryKey: ['sched-list'] });
    qc.invalidateQueries({ queryKey: ['sched-runs'] });
  };
  const toggle = useMutation({
    mutationFn: (s: Schedule) => (s.enabled ? schedulesApi.pause(s.id) : schedulesApi.resume(s.id)),
    onSuccess: refresh,
  });
  const runNow = useMutation({ mutationFn: (s: Schedule) => schedulesApi.runNow(s.id), onSuccess: () => { setTab('runs'); setTimeout(refresh, 1500); } });
  const remove = useMutation({ mutationFn: (s: Schedule) => schedulesApi.remove(s.id), onSuccess: refresh });
  const actionError = toggle.error ?? runNow.error ?? remove.error;
  const all = list.data?.schedules ?? [];
  const [search, setSearch] = useState('');
  const [kind, setKind] = useState('');
  const [state, setState] = useState<'' | 'active' | 'paused'>('');
  const filtered = !!(search.trim() || kind || state);
  // Every schedule is already loaded, so this tab filters in the browser -
  // over what the row shows, in the viewer's language.
  const schedules = useMemo(() => {
    const words = search.trim().toLocaleLowerCase(i18n.language).split(/\s+/).filter(Boolean);
    return all.filter((s) => {
      if (kind && s.target.kind !== kind) return false;
      if (state && (state === 'active') !== s.enabled) return false;
      if (!words.length) return true;
      const text = [s.name, s.description, s.target.kind, t(`schedules.kinds.${s.target.kind}`, s.target.kind), s.timing.calendar, s.timing.cron, when(s)]
        .filter(Boolean).join(' ').toLocaleLowerCase(i18n.language);
      return words.every((w) => text.includes(w));
    });
  }, [all, search, kind, state, when, t, i18n.language]);

  return (
    // Own themed surface: the shell's canvas is dark whatever the MUI mode.
    <Box sx={{ bgcolor: 'background.default', color: 'text.primary', minHeight: '100%' }}>
      <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1400, mx: 'auto' }}>
        <Stack direction={{ xs: 'column', md: 'row' }} alignItems={{ md: 'center' }} spacing={2} sx={{ mb: 2 }}>
          <Stack direction="row" spacing={2} alignItems="center" sx={{ flex: 1 }}>
            <EventRepeatIcon color="primary" fontSize="large" />
            <Box>
              <Typography variant="h5" fontWeight={700}>{t('schedules.title')}</Typography>
              <Typography color="text.secondary">{t('schedules.subtitle')}</Typography>
            </Box>
          </Stack>
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => setEditing('new')}>{t('schedules.new')}</Button>
        </Stack>

        <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Tab value="schedules" label={t('schedules.tabs.schedules')} />
          <Tab value="runs" label={t('schedules.tabs.runs')} />
        </Tabs>

        {actionError && <Box sx={{ mb: 2 }}><CatalogErrorAlert error={actionError} /></Box>}
        {list.error && <CatalogErrorAlert error={list.error} />}

        {tab === 'schedules' && (
          <>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} sx={{ mb: 2 }}>
              <SearchField value={search} onChange={setSearch} placeholder={t('schedules.filters.searchSchedules')} />
              <KindSelect value={kind} onChange={setKind} />
              <TextField select size="small" value={state} onChange={(e) => setState(e.target.value as typeof state)} sx={{ minWidth: 170 }}
                inputProps={{ 'aria-label': t('schedules.columns.enabled') }}>
                <MenuItem value="">{t('schedules.filters.allStates')}</MenuItem>
                <MenuItem value="active">{t('schedules.filters.active')}</MenuItem>
                <MenuItem value="paused">{t('schedules.filters.paused')}</MenuItem>
              </TextField>
              {filtered && <Button onClick={() => { setSearch(''); setKind(''); setState(''); }}>{t('schedules.filters.clear')}</Button>}
            </Stack>
            {list.isLoading && <LinearProgress />}
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>{t('schedules.columns.name')}</TableCell>
                    <TableCell>{t('schedules.columns.runs')}</TableCell>
                    <TableCell>{t('schedules.columns.when')}</TableCell>
                    <TableCell>{t('schedules.columns.enabled')}</TableCell>
                    <TableCell align="right" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {schedules.map((s) => (
                    <TableRow key={s.id} hover>
                      <TableCell>
                        <Typography variant="body2" fontWeight={600}>{s.name}</Typography>
                        <Typography variant="caption" color="text.secondary">{t('schedules.updated', { when: fmt(s.updated_at, i18n.language) })}</Typography>
                      </TableCell>
                      <TableCell><Chip size="small" variant="outlined" label={t(`schedules.kinds.${s.target.kind}`, s.target.kind)} /></TableCell>
                      <TableCell><Typography variant="body2">{when(s)}</Typography></TableCell>
                      <TableCell>
                        <Switch checked={s.enabled} onChange={() => toggle.mutate(s)} disabled={toggle.isPending}
                          inputProps={{ 'aria-label': t('schedules.columns.enabled') }} />
                      </TableCell>
                      <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                        <Tooltip title={t('schedules.runNow')}><IconButton size="small" onClick={() => runNow.mutate(s)}><PlayArrowIcon fontSize="small" /></IconButton></Tooltip>
                        <Tooltip title={t('schedules.edit')}><IconButton size="small" onClick={() => setEditing(s)}><EditIcon fontSize="small" /></IconButton></Tooltip>
                        <Tooltip title={t('schedules.delete')}>
                          <IconButton size="small" onClick={() => { if (window.confirm(t('schedules.confirmDelete', { name: s.name }))) remove.mutate(s); }}>
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  ))}
                  {!list.isLoading && schedules.length === 0 && (
                    <TableRow><TableCell colSpan={5}><Typography color="text.secondary" sx={{ py: 3, textAlign: 'center' }}>{t(filtered ? 'schedules.filters.noMatches' : 'schedules.empty')}</Typography></TableCell></TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </>
        )}

        {tab === 'runs' && <RunsTable />}

        {editing && (
          <ScheduleEditor open onClose={() => setEditing(null)} schedule={editing === 'new' ? undefined : editing} />
        )}
      </Box>
    </Box>
  );
}
