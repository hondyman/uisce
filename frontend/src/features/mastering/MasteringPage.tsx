import React, { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Alert, Badge, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, FormControlLabel, InputAdornment, LinearProgress,
  MenuItem, Paper, Radio, RadioGroup, Stack, Tab, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Tabs, TextField,
  Tooltip, Typography,
} from '@mui/material';
import HubIcon from '@mui/icons-material/Hub';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import SearchIcon from '@mui/icons-material/Search';
import { CatalogErrorAlert } from '../message-catalog/parts';
import { fmt } from '../schedules/api';
import { DecisionResult, ExceptionRow, masteringApi, MatchCandidate, pct, Profile, Run } from './api';
import GoldenDrawer from './GoldenDrawer';
import { GoldenStatusChip, RunStatusChip } from './parts';
import RunDialog, { CountChips } from './RunDialog';

type TabKey = 'golden' | 'runs' | 'exceptions' | 'review';

function useDebounced<T>(v: T, ms = 300): T {
  const [d, setD] = useState(v);
  useEffect(() => { const h = setTimeout(() => setD(v), ms); return () => clearTimeout(h); }, [v, ms]);
  return d;
}

function Empty({ cols, text }: { cols: number; text: string }) {
  return <TableRow><TableCell colSpan={cols}><Typography color="text.secondary" sx={{ py: 3, textAlign: 'center' }}>{text}</Typography></TableCell></TableRow>;
}

function GoldenTab({ entity, onOpen }: { entity: string; onOpen: (id: string) => void }) {
  const { t, i18n } = useTranslation();
  const [q, setQ] = useState('');
  const [status, setStatus] = useState('');
  const dq = useDebounced(q);
  const list = useQuery({ queryKey: ['mastering', 'golden', entity, dq, status], queryFn: () => masteringApi.golden(entity, { q: dq, status }) });
  const rows = list.data?.golden ?? [];
  return (
    <>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mb: 2 }}>
        <TextField size="small" sx={{ flex: 1, maxWidth: 420 }} placeholder={t('mastering.golden.search')} value={q} onChange={(e) => setQ(e.target.value)}
          InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }} />
        <TextField select size="small" sx={{ minWidth: 180 }} label={t('mastering.golden.status')} value={status} onChange={(e) => setStatus(e.target.value)}
          SelectProps={{ displayEmpty: true }} InputLabelProps={{ shrink: true }}>
          <MenuItem value="">{t('mastering.all')}</MenuItem>
          {(['PUBLISHED', 'REVIEW'] as const).map((s) => <MenuItem key={s} value={s}>{t(`mastering.goldenStatus.${s}`)}</MenuItem>)}
        </TextField>
      </Stack>
      {list.isLoading && <LinearProgress />}
      {list.error && <CatalogErrorAlert error={list.error} />}
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>{t('mastering.golden.code')}</TableCell>
              <TableCell>{t('mastering.golden.name')}</TableCell>
              <TableCell>{t('mastering.golden.status')}</TableCell>
              <TableCell align="right">{t('mastering.golden.dqShort')}</TableCell>
              <TableCell align="right">{t('mastering.golden.identityShort')}</TableCell>
              <TableCell>{t('mastering.golden.sources')}</TableCell>
              <TableCell>{t('mastering.golden.updated')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((g) => (
              <TableRow key={g.id} hover sx={{ cursor: 'pointer' }} onClick={() => onOpen(g.id)}>
                <TableCell sx={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>{g.code}</TableCell>
                <TableCell>{g.name ?? '—'}</TableCell>
                <TableCell><GoldenStatusChip status={g.status} /> <Typography component="span" variant="caption" color="text.secondary">v{g.version}</Typography></TableCell>
                <TableCell align="right">{g.dq_score ?? '—'}</TableCell>
                <TableCell align="right">{pct(g.identity_confidence)}</TableCell>
                <TableCell>
                  <Tooltip title={Object.entries(g.winning_sources ?? {}).map(([a, s]) => `${a}: ${s}`).join('\n')} componentsProps={{ tooltip: { sx: { whiteSpace: 'pre-line' } } }}>
                    <Chip size="small" label={Array.from(new Set(Object.values(g.winning_sources ?? {}))).join(', ') || g.sources} />
                  </Tooltip>
                </TableCell>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>{fmt(g.updated_at, i18n.language)}</TableCell>
              </TableRow>
            ))}
            {!list.isLoading && !list.error && rows.length === 0 && <Empty cols={7} text={t('mastering.golden.empty')} />}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
}

function RunsTab({ entity }: { entity: string }) {
  const { t, i18n } = useTranslation();
  const runs = useQuery({ queryKey: ['mastering', 'runs', entity], queryFn: () => masteringApi.runs(entity) });
  const rows = runs.data?.runs ?? [];
  return (
    <>
      {runs.isLoading && <LinearProgress />}
      {runs.error && <CatalogErrorAlert error={runs.error} />}
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>{t('mastering.runs.started')}</TableCell>
              <TableCell>{t('mastering.runs.status')}</TableCell>
              <TableCell>{t('mastering.runs.trigger')}</TableCell>
              <TableCell>{t('mastering.runs.outcome')}</TableCell>
              <TableCell>{t('mastering.runs.by')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((r) => (
              <TableRow key={r.id} hover>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>{fmt(r.started_at, i18n.language)}</TableCell>
                <TableCell>
                  <RunStatusChip status={r.status} />
                  {r.error_detail && <Typography variant="caption" color="error" component="p">{r.error_detail}</Typography>}
                </TableCell>
                <TableCell>{t(`mastering.trigger.${r.trigger}`, r.trigger)}</TableCell>
                <TableCell><CountChips counts={r.counts} /></TableCell>
                <TableCell>{r.started_by ?? '—'}</TableCell>
              </TableRow>
            ))}
            {!runs.isLoading && !runs.error && rows.length === 0 && <Empty cols={5} text={t('mastering.runs.empty')} />}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
}

function ExceptionsTab({ entity, onOpen }: { entity: string; onOpen: (id: string) => void }) {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const [status, setStatus] = useState('');
  const list = useQuery({ queryKey: ['mastering', 'exceptions', entity, status], queryFn: () => masteringApi.exceptions(entity, status) });
  const resolve = useMutation({
    mutationFn: ({ x, s }: { x: ExceptionRow; s: 'RESOLVED' | 'WAIVED' }) => masteringApi.resolve(entity, x.id, s),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['mastering'] }),
  });
  const rows = list.data?.exceptions ?? [];
  const open = status === '' || status === 'OPEN' || status === 'IN_REVIEW';
  return (
    <>
      <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
        <TextField select size="small" sx={{ minWidth: 200 }} label={t('mastering.exceptions.status')} value={status} onChange={(e) => setStatus(e.target.value)}
          SelectProps={{ displayEmpty: true }} InputLabelProps={{ shrink: true }}>
          <MenuItem value="">{t('mastering.exceptions.open')}</MenuItem>
          {(['RESOLVED', 'WAIVED'] as const).map((s) => <MenuItem key={s} value={s}>{t(`mastering.exceptionStatus.${s}`)}</MenuItem>)}
        </TextField>
      </Stack>
      {list.isLoading && <LinearProgress />}
      {list.error && <CatalogErrorAlert error={list.error} />}
      {resolve.error && <Box sx={{ mb: 2 }}><CatalogErrorAlert error={resolve.error} /></Box>}
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>{t('mastering.exceptions.type')}</TableCell>
              <TableCell>{t('mastering.exceptions.description')}</TableCell>
              <TableCell>{t('mastering.exceptions.record')}</TableCell>
              <TableCell>{t('mastering.exceptions.detected')}</TableCell>
              {open && <TableCell align="right" />}
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((x) => (
              <TableRow key={x.id} hover>
                <TableCell><Chip size="small" color={x.severity === 'ERROR' ? 'error' : 'warning'} label={t(`mastering.exceptionType.${x.type}`, x.type)} /></TableCell>
                <TableCell>{x.description}</TableCell>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>
                  {x.golden_id
                    ? <Button size="small" onClick={() => onOpen(x.golden_id!)}>{t('mastering.exceptions.openGolden')}</Button>
                    : <Typography variant="body2" fontFamily="monospace">{[x.source, x.source_key].filter(Boolean).join(' / ')}</Typography>}
                </TableCell>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>{fmt(x.detected_at, i18n.language)}</TableCell>
                {open && (
                  <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                    <Button size="small" disabled={resolve.isPending} onClick={() => resolve.mutate({ x, s: 'RESOLVED' })}>{t('mastering.exceptions.resolve')}</Button>
                    <Button size="small" color="inherit" disabled={resolve.isPending} onClick={() => resolve.mutate({ x, s: 'WAIVED' })}>{t('mastering.exceptions.waive')}</Button>
                  </TableCell>
                )}
              </TableRow>
            ))}
            {!list.isLoading && !list.error && rows.length === 0 && <Empty cols={open ? 5 : 4} text={t('mastering.exceptions.empty')} />}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
}

function ReviewTab({ entity, onOpen }: { entity: string; onOpen: (id: string) => void }) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const list = useQuery({ queryKey: ['mastering', 'candidates', entity], queryFn: () => masteringApi.candidates(entity) });
  const [deciding, setDeciding] = useState<{ c: MatchCandidate; merge: boolean } | null>(null);
  const [keep, setKeep] = useState<'a' | 'b'>('a');
  const [note, setNote] = useState('');
  const [done, setDone] = useState<DecisionResult | null>(null);
  const decide = useMutation({
    mutationFn: () => masteringApi.decide(entity, deciding!.c.id, { merge: deciding!.merge, keep, note: note.trim() || undefined }),
    onSuccess: (r) => {
      setDone(r.decision);
      setDeciding(null);
      qc.invalidateQueries({ queryKey: ['mastering'] });
    },
  });
  const open = (c: MatchCandidate, merge: boolean) => { setDeciding({ c, merge }); setKeep('a'); setNote(''); decide.reset(); };
  const rows = list.data?.candidates ?? [];
  return (
    <>
      <Alert severity="info" sx={{ mb: 2 }}>{t('mastering.review.help')}</Alert>
      {done && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setDone(null)}>
          {done.status === 'APPROVED'
            ? t('mastering.review.merged', { sources: done.moved.sources, identifiers: done.moved.identifiers })
            : t('mastering.review.rejected')}
        </Alert>
      )}
      {list.isLoading && <LinearProgress />}
      {list.error && <CatalogErrorAlert error={list.error} />}
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>{t('mastering.review.existing')}</TableCell>
              <TableCell>{t('mastering.review.new')}</TableCell>
              <TableCell align="right">{t('mastering.review.score')}</TableCell>
              <TableCell>{t('mastering.review.rule')}</TableCell>
              <TableCell align="right" />
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((c) => (
              <TableRow key={c.id} hover>
                <TableCell><Button size="small" onClick={() => onOpen(c.a)}>{c.a_code}</Button> {c.a_name}</TableCell>
                <TableCell><Button size="small" onClick={() => onOpen(c.b)}>{c.b_code}</Button> {c.b_name}</TableCell>
                <TableCell align="right">{pct(c.score)}</TableCell>
                <TableCell>{c.rule}</TableCell>
                <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                  <Button size="small" variant="outlined" onClick={() => open(c, true)}>{t('mastering.review.merge')}</Button>
                  <Button size="small" color="inherit" sx={{ ml: 1 }} onClick={() => open(c, false)}>{t('mastering.review.notSame')}</Button>
                </TableCell>
              </TableRow>
            ))}
            {!list.isLoading && !list.error && rows.length === 0 && <Empty cols={5} text={t('mastering.review.empty')} />}
          </TableBody>
        </Table>
      </TableContainer>
      <Dialog open={!!deciding} onClose={() => setDeciding(null)} fullWidth maxWidth="sm">
        <DialogTitle>{deciding?.merge ? t('mastering.review.mergeTitle') : t('mastering.review.notSameTitle')}</DialogTitle>
        <DialogContent dividers>
          {deciding?.merge ? (
            <>
              <Typography variant="body2" sx={{ mb: 2 }}>{t('mastering.review.mergeHelp')}</Typography>
              <RadioGroup value={keep} onChange={(e) => setKeep(e.target.value as 'a' | 'b')}>
                <FormControlLabel value="a" control={<Radio size="small" />}
                  label={t('mastering.review.keep', { code: deciding.c.a_code, name: deciding.c.a_name ?? '' })} />
                <FormControlLabel value="b" control={<Radio size="small" />}
                  label={t('mastering.review.keep', { code: deciding.c.b_code, name: deciding.c.b_name ?? '' })} />
              </RadioGroup>
            </>
          ) : (
            <Typography variant="body2" sx={{ mb: 2 }}>{t('mastering.review.notSameHelp')}</Typography>
          )}
          <TextField fullWidth multiline minRows={2} sx={{ mt: 2 }} label={t('mastering.review.note')} value={note} onChange={(e) => setNote(e.target.value)}
            helperText={t('mastering.review.noteHelp')} />
          {decide.isPending && <LinearProgress sx={{ mt: 2 }} />}
          {decide.error && <Box sx={{ mt: 2 }}><CatalogErrorAlert error={decide.error} /></Box>}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeciding(null)}>{t('mastering.cancel')}</Button>
          <Button variant="contained" color={deciding?.merge ? 'primary' : 'inherit'} disabled={decide.isPending} onClick={() => decide.mutate()}>
            {deciding?.merge ? t('mastering.review.confirmMerge') : t('mastering.review.confirmNotSame')}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

/**
 * The mastering console: golden records and where every value came from,
 * runs, exceptions for stewards, and possible duplicates to review. One page
 * for every mastered entity (Product, Security, Benchmark, Price ...).
 */
export default function MasteringPage() {
  const { t } = useTranslation();
  const profiles = useQuery({ queryKey: ['mastering', 'profiles'], queryFn: masteringApi.profiles });
  const [entity, setEntity] = useState('');
  const [tab, setTab] = useState<TabKey>('golden');
  const [goldenId, setGoldenId] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const [lastRun, setLastRun] = useState<Run | null>(null);

  const list = profiles.data?.profiles ?? [];
  useEffect(() => { if (!entity && list.length) setEntity(list[0].entity_cd.toLowerCase()); }, [list, entity]);
  const profile: Profile | undefined = list.find((p) => p.entity_cd.toLowerCase() === entity);

  const exceptions = useQuery({ queryKey: ['mastering', 'exceptions', entity, ''], queryFn: () => masteringApi.exceptions(entity), enabled: !!entity });
  const candidates = useQuery({ queryKey: ['mastering', 'candidates', entity], queryFn: () => masteringApi.candidates(entity), enabled: !!entity });
  const openCount = exceptions.data?.exceptions.length ?? 0;
  const reviewCount = candidates.data?.candidates.length ?? 0;

  const badge = (label: string, n: number) => (
    <Badge color="warning" badgeContent={n} sx={{ pr: n ? 1.5 : 0 }}>{label}</Badge>
  );

  return (
    // Own themed surface: the shell's canvas is dark whatever the MUI mode.
    <Box sx={{ bgcolor: 'background.default', color: 'text.primary', minHeight: '100%' }}>
      <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1400, mx: 'auto' }}>
        <Stack direction={{ xs: 'column', md: 'row' }} alignItems={{ md: 'center' }} spacing={2} sx={{ mb: 2 }}>
          <Stack direction="row" spacing={2} alignItems="center" sx={{ flex: 1 }}>
            <HubIcon color="primary" fontSize="large" />
            <Box>
              <Typography variant="h5" fontWeight={700}>{t('mastering.title')}</Typography>
              <Typography color="text.secondary">{t('mastering.subtitle')}</Typography>
            </Box>
          </Stack>
          <TextField select size="small" sx={{ minWidth: 200 }} label={t('mastering.entity')} value={entity} onChange={(e) => { setEntity(e.target.value); setGoldenId(null); }}>
            {list.map((p) => <MenuItem key={p.id} value={p.entity_cd.toLowerCase()}>{p.display_name}</MenuItem>)}
          </TextField>
          <Button variant="contained" startIcon={<PlayArrowIcon />} disabled={!profile} onClick={() => setRunning(true)}>{t('mastering.runLoad')}</Button>
        </Stack>
        {profiles.isLoading && <LinearProgress />}
        {profiles.error && <CatalogErrorAlert error={profiles.error} />}
        {!profiles.isLoading && !profiles.error && list.length === 0 && <Alert severity="info">{t('mastering.noProfiles')}</Alert>}

        {lastRun && (
          <Alert severity={lastRun.status === 'FAILED' ? 'error' : lastRun.status === 'PARTIAL' ? 'warning' : 'success'} sx={{ mb: 2 }} onClose={() => setLastRun(null)}>
            <Typography variant="body2" sx={{ mb: 1 }}>
              {lastRun.replayed ? t('mastering.replayed') : t(`mastering.finished.${lastRun.status}`, lastRun.status)}
            </Typography>
            <CountChips counts={lastRun.counts} />
            {lastRun.error_detail && <Typography variant="caption" component="p" sx={{ mt: 1 }}>{lastRun.error_detail}</Typography>}
          </Alert>
        )}

        {entity && (
          <>
            <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }} variant="scrollable" allowScrollButtonsMobile>
              <Tab value="golden" label={t('mastering.tabs.golden')} />
              <Tab value="runs" label={t('mastering.tabs.runs')} />
              <Tab value="exceptions" label={badge(t('mastering.tabs.exceptions'), openCount)} />
              <Tab value="review" label={badge(t('mastering.tabs.review'), reviewCount)} />
            </Tabs>
            {tab === 'golden' && <GoldenTab entity={entity} onOpen={setGoldenId} />}
            {tab === 'runs' && <RunsTab entity={entity} />}
            {tab === 'exceptions' && <ExceptionsTab entity={entity} onOpen={setGoldenId} />}
            {tab === 'review' && <ReviewTab entity={entity} onOpen={setGoldenId} />}
          </>
        )}
      </Box>
      {entity && <GoldenDrawer entity={entity} id={goldenId} onClose={() => setGoldenId(null)} />}
      {profile && running && (
        <RunDialog profile={profile} open={running} onClose={() => setRunning(false)}
          onDone={(r) => { setRunning(false); setLastRun(r); setTab(r.counts.exceptions ? 'exceptions' : 'golden'); }} />
      )}
    </Box>
  );
}
