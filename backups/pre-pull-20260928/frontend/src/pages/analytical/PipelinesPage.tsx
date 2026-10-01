import React, { useState, useMemo, useCallback } from 'react';
import {
  Box, Typography, Button, IconButton, Tooltip, Chip, Paper, Tabs, Tab, TextField, InputBase,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Divider, Snackbar, Alert,
  CircularProgress,
} from '@mui/material';
import {
  Search as SearchIcon, AccountTree as AccountTreeIcon, PlayArrow as PlayArrowIcon,
  Pause as PauseIcon, Stop as StopIcon, Schedule as ScheduleIcon, History as HistoryIcon,
  Refresh as RefreshIcon, Add as AddIcon, Storage as StorageIcon, CheckCircle as CheckCircleIcon,
  ErrorOutline as ErrorOutlineIcon, HourglassEmpty as HourglassEmptyIcon, Bolt as BoltIcon,
  CloudUpload as CloudUploadIcon, CloudDownload as CloudDownloadIcon,
  Transform as TransformIcon, Storage as StorageOutIcon,
  CheckCircleOutline as CheckCircleOutlineIcon, Loop as LoopIcon, ArrowForward as ArrowForwardIcon,
  BarChart as BarChartIcon, Timeline as TimelineIcon, MoreVert as MoreVertIcon,
} from '@mui/icons-material';
import AnalyticalShell from '../../components/analytical/AnalyticalShell';
import apiClient from '../../utils/apiClient';
import { devLog } from '../../utils/devLogger';

type NodeKind = 'source' | 'transform' | 'sink' | 'loop';

interface PipelineNode {
  id: string;
  label: string;
  kind: NodeKind;
  status: 'ok' | 'running' | 'pending' | 'error';
  meta?: string;
}

interface Pipeline {
  id: string;
  name: string;
  description: string;
  status: 'running' | 'paused' | 'failed' | 'success' | 'queued';
  owner: string;
  cadence: string;
  lastRunAt: string;
  lastDurationMs: number;
  successRate: number;
  rowsProcessed24h: string;
  schedule: string;
  nodes: PipelineNode[];
}

interface RunRow {
  id: string;
  pipelineId: string;
  startedAt: string;
  durationMs: number;
  rowsIn: number;
  rowsOut: number;
  status: 'success' | 'failed' | 'running' | 'queued';
  triggeredBy: string;
}

const PIPELINES: Pipeline[] = [
  {
    id: 'pipe-1',
    name: 'CRIMS Position Rollup (Hourly)',
    description: 'Ingests trade-execution events from Kafka, joins with security master, emits hourly valuation snapshots to StarRocks.',
    status: 'running',
    owner: '@quant-infra',
    cadence: 'Hourly',
    lastRunAt: '12m ago',
    lastDurationMs: 18420,
    successRate: 0.998,
    rowsProcessed24h: '4.1M rows',
    schedule: 'cron(0 * * * *)',
    nodes: [
      { id: 'n1', label: 'Kafka trades.events',  kind: 'source',    status: 'ok',      meta: 'avro · 14 partitions' },
      { id: 'n2', label: 'Normalize + Validate',   kind: 'transform', status: 'ok' },
      { id: 'n3', label: 'Join Security Master',   kind: 'transform', status: 'ok' },
      { id: 'n4', label: 'Aggregate (Hourly)',     kind: 'transform', status: 'ok' },
      { id: 'n5', label: 'StarRocks OLAP',        kind: 'sink',      status: 'ok',      meta: 'fact_position_rollup' },
      { id: 'n6', label: 'CDC Retries',            kind: 'loop',      status: 'pending', meta: 'max 3' },
    ],
  },
  {
    id: 'pipe-2',
    name: 'SWIFT Message Classifier',
    description: 'Consumes MT103/MT202 messages, classifies via semantic model, persists to audit_reconciliation_log.',
    status: 'running',
    owner: '@treasury-data',
    cadence: 'Realtime',
    lastRunAt: '3s ago',
    lastDurationMs: 920,
    successRate: 0.9992,
    rowsProcessed24h: '892K rows',
    schedule: 'streaming',
    nodes: [
      { id: 'n1', label: 'SWIFT Gateway',          kind: 'source',    status: 'ok' },
      { id: 'n2', label: 'MT103/MT202 Parser',     kind: 'transform', status: 'ok' },
      { id: 'n3', label: 'Semantic Classifier',    kind: 'transform', status: 'running' },
      { id: 'n4', label: 'Postgres audit',         kind: 'sink',      status: 'ok' },
    ],
  },
  {
    id: 'pipe-3',
    name: 'Daily P&L Reconciliation',
    description: 'Runs overnight to reconcile realized/unrealized P&L across trading books.',
    status: 'failed',
    owner: '@risk-eng',
    cadence: 'Daily 02:00',
    lastRunAt: '4h ago',
    lastDurationMs: 312000,
    successRate: 0.92,
    rowsProcessed24h: '128K rows',
    schedule: 'cron(0 2 * * *)',
    nodes: [
      { id: 'n1', label: 'StarRocks fact_pnl',     kind: 'source',    status: 'ok' },
      { id: 'n2', label: 'Book Alignment',         kind: 'transform', status: 'ok' },
      { id: 'n3', label: 'Variance Detection',     kind: 'transform', status: 'error', meta: 'threshold breach' },
      { id: 'n4', label: 'Ops Alert',              kind: 'sink',      status: 'pending' },
    ],
  },
  {
    id: 'pipe-4',
    name: 'Reference Data Sync (Bloomberg → StarRocks)',
    description: 'Daily sync of instrument reference data from Bloomberg B-PIPE feed.',
    status: 'success',
    owner: '@quant-infra',
    cadence: 'Daily 04:00',
    lastRunAt: '9h ago',
    lastDurationMs: 482000,
    successRate: 1.0,
    rowsProcessed24h: '18.4M rows',
    schedule: 'cron(0 4 * * *)',
    nodes: [
      { id: 'n1', label: 'Bloomberg B-PIPE',       kind: 'source',    status: 'ok' },
      { id: 'n2', label: 'CUSIP / ISIN Lookup',    kind: 'transform', status: 'ok' },
      { id: 'n3', label: 'StarRocks dim_cusip',    kind: 'sink',      status: 'ok' },
    ],
  },
];

const RUN_HISTORY: RunRow[] = [
  { id: 'r-1024', pipelineId: 'pipe-1', startedAt: '12m ago',     durationMs: 18420,  rowsIn: 412880, rowsOut: 412870, status: 'success', triggeredBy: 'cron · hourly' },
  { id: 'r-1023', pipelineId: 'pipe-1', startedAt: '1h 12m ago', durationMs: 17920,  rowsIn: 388110, rowsOut: 388090, status: 'success', triggeredBy: 'cron · hourly' },
  { id: 'r-1022', pipelineId: 'pipe-2', startedAt: '3s ago',      durationMs: 0,      rowsIn: 1240,   rowsOut: 0,      status: 'running', triggeredBy: 'streaming' },
  { id: 'r-1021', pipelineId: 'pipe-3', startedAt: '4h ago',      durationMs: 312000, rowsIn: 128400, rowsOut: 0,      status: 'failed',  triggeredBy: 'cron · daily' },
  { id: 'r-1020', pipelineId: 'pipe-4', startedAt: '9h ago',      durationMs: 482000, rowsIn: 18400000, rowsOut: 18399400, status: 'success', triggeredBy: 'cron · daily' },
  { id: 'r-1019', pipelineId: 'pipe-1', startedAt: '2h 12m ago', durationMs: 18100,  rowsIn: 401200, rowsOut: 401190, status: 'success', triggeredBy: 'cron · hourly' },
  { id: 'r-1018', pipelineId: 'pipe-1', startedAt: '3h 12m ago', durationMs: 18410,  rowsIn: 419000, rowsOut: 418990, status: 'success', triggeredBy: 'cron · hourly' },
];

const NODE_ICON: Record<NodeKind, React.ReactNode> = {
  source:    <CloudUploadIcon sx={{ fontSize: 16 }} />,
  transform: <TransformIcon sx={{ fontSize: 16 }} />,
  sink:      <CloudDownloadIcon sx={{ fontSize: 16 }} />,
  loop:      <LoopIcon sx={{ fontSize: 16 }} />,
};

const STATUS_COLOR: Record<Pipeline['status'], string> = {
  running: 'var(--mui-primary-light)',
  paused:  'var(--mui-text-secondary)',
  failed:  'var(--mui-error)',
  success: 'var(--mui-primary-light)',
  queued:  'var(--mui-tertiary)',
};

const NODE_STATUS_COLOR: Record<PipelineNode['status'], string> = {
  ok:      'var(--mui-primary-light)',
  running: 'var(--mui-tertiary)',
  pending: 'var(--mui-warning-main)',
  error:   'var(--mui-error)',
};

const PipelinesPage: React.FC = () => {
  const [pipelines, setPipelines] = useState<Pipeline[]>(PIPELINES);
  const [runs] = useState<RunRow[]>(RUN_HISTORY);
  const [selectedId, setSelectedId] = useState<string>('pipe-1');
  const [search, setSearch] = useState('');
  const [tab, setTab] = useState(0);
  const [snack, setSnack] = useState<{ open: boolean; msg: string; severity?: 'success' | 'info' | 'error' }>({ open: false, msg: '' });

  const showSnack = useCallback((msg: string, severity: 'success' | 'info' | 'error' = 'info') => setSnack({ open: true, msg, severity }), []);

  const visible = useMemo(() => {
    const q = search.toLowerCase();
    if (!q) return pipelines;
    return pipelines.filter((p) => p.name.toLowerCase().includes(q) || p.owner.toLowerCase().includes(q));
  }, [pipelines, search]);

  const selected = pipelines.find((p) => p.id === selectedId) ?? pipelines[0];

  const handleRun = useCallback(async (id: string) => {
    const pipe = pipelines.find((p) => p.id === id);
    if (!pipe) return;
    setPipelines((prev) => prev.map((p) => (p.id === id ? { ...p, status: 'queued' } : p)));
    showSnack(`Triggering ${pipe.name}…`, 'info');
    try {
      await apiClient<unknown>('/api/pipelines/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pipelineId: id }),
      });
    } catch {
      // Local fallback simulation
    }
    setTimeout(() => {
      setPipelines((prev) => prev.map((p) => (p.id === id ? { ...p, status: 'running', lastRunAt: 'just now' } : p)));
      showSnack(`${pipe.name} started`, 'success');
    }, 600);
  }, [pipelines, showSnack]);

  const handlePause = useCallback((id: string) => {
    const pipe = pipelines.find((p) => p.id === id);
    setPipelines((prev) => prev.map((p) => (p.id === id ? { ...p, status: 'paused' } : p)));
    showSnack(`${pipe?.name} paused`, 'info');
  }, [pipelines, showSnack]);

  const handleResume = useCallback((id: string) => {
    const pipe = pipelines.find((p) => p.id === id);
    setPipelines((prev) => prev.map((p) => (p.id === id ? { ...p, status: 'running' } : p)));
    showSnack(`${pipe?.name} resumed`, 'success');
  }, [pipelines, showSnack]);

  return (
    <AnalyticalShell>
      <Box sx={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, bgcolor: 'var(--mui-bg-default)' }}>
        {/* Header */}
        <Paper sx={{ p: 2, display: 'flex', flexDirection: { xs: 'column', md: 'row' }, alignItems: { md: 'center' }, justifyContent: 'space-between', gap: 1.5, mx: 2, mt: 2, borderRadius: '4px' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Box sx={{ p: 1, borderRadius: '4px', bgcolor: 'var(--mui-bg-subtle)', color: 'var(--mui-tertiary)', display: 'flex', alignItems: 'center' }}>
              <AccountTreeIcon sx={{ fontSize: 22 }} />
            </Box>
            <Box>
              <Typography sx={{ fontFamily: 'var(--font-headline-md)', fontSize: '1.25rem', color: 'var(--mui-text-primary)', fontWeight: 600, letterSpacing: '-0.01em' }}>
                Data Pipelines
              </Typography>
              <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                Streaming and batch DAGs orchestrating ingest, transform, and delivery to OLTP/OLAP sinks.
              </Typography>
            </Box>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
            <Box sx={{ bgcolor: 'var(--mui-bg-paper)', px: 1.5, py: 0.75, borderRadius: '4px', display: 'flex', alignItems: 'center', gap: 1 }}>
              <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: 'var(--mui-primary-light)' }} />
              <Typography className="font-mono" sx={{ fontSize: '0.6875rem', color: 'var(--mui-text-primary)' }}>
                {pipelines.filter((p) => p.status === 'running').length} running
              </Typography>
            </Box>
            <Box sx={{ bgcolor: 'var(--mui-bg-paper)', px: 1.5, py: 0.75, borderRadius: '4px', display: 'flex', alignItems: 'center', gap: 1 }}>
              <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: 'var(--mui-error)' }} />
              <Typography className="font-mono" sx={{ fontSize: '0.6875rem', color: 'var(--mui-text-primary)' }}>
                {pipelines.filter((p) => p.status === 'failed').length} failed
              </Typography>
            </Box>
            <Box sx={{ bgcolor: 'var(--mui-bg-paper)', px: 1.5, py: 0.75, borderRadius: '4px', display: 'flex', alignItems: 'center', gap: 1 }}>
              <ScheduleIcon sx={{ fontSize: 14, color: 'var(--mui-secondary-main)' }} />
              <Typography className="font-mono" sx={{ fontSize: '0.6875rem', color: 'var(--mui-text-primary)' }}>
                4 cron / 1 streaming
              </Typography>
            </Box>
            <Button
              startIcon={<AddIcon sx={{ fontSize: 18 }} />}
              onClick={() => showSnack('Pipeline draft created', 'success')}
              sx={{ bgcolor: 'var(--mui-primary-main)', color: 'var(--mui-primary-contrastText)', fontFamily: 'var(--font-headline-sm)', fontSize: '0.8125rem', px: 1.5, py: 1, textTransform: 'none', boxShadow: 'none', '&:hover': { bgcolor: 'var(--mui-primary-light)', boxShadow: 'none' } }}
            >
              New Pipeline
            </Button>
          </Box>
        </Paper>

        {/* Search + list */}
        <Box sx={{ mx: 2, mt: 1.5, display: 'flex', flexDirection: { xs: 'column', md: 'row' }, gap: 1.5, flex: 1, minHeight: 0 }}>
          {/* Pipeline list */}
          <Paper sx={{ p: 1.5, display: 'flex', flexDirection: 'column', gap: 1, width: { xs: '100%', md: 340 }, flexShrink: 0, minHeight: 0, overflow: 'hidden' }}>
            <Box sx={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
              <SearchIcon sx={{ position: 'absolute', left: 8, color: 'var(--mui-text-secondary)', fontSize: 16, pointerEvents: 'none' }} />
              <InputBase
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Filter pipelines…"
                fullWidth
                sx={{
                  bgcolor: 'var(--mui-bg-subtle)',
                  color: 'var(--mui-text-primary)',
                  fontFamily: 'var(--font-mono-label)',
                  fontSize: '0.75rem',
                  pl: 4,
                  pr: 1,
                  py: 0.5,
                  borderRadius: '3px',
                  '& input': { padding: 0 },
                }}
              />
            </Box>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, overflowY: 'auto' }}>
              {visible.map((p) => (
                <Box
                  key={p.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => setSelectedId(p.id)}
                  onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') setSelectedId(p.id); }}
                  sx={{
                    p: 1.25,
                    borderRadius: '4px',
                    cursor: 'pointer',
                    bgcolor: p.id === selectedId ? 'var(--mui-bg-elevated)' : 'var(--mui-bg-subtle)',
                    border: '1px solid',
                    borderColor: p.id === selectedId ? 'var(--mui-primary-main)' : 'var(--mui-border)',
                    transition: 'all 0.15s',
                    '&:hover': { borderColor: 'var(--mui-primary-main)' },
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, minWidth: 0 }}>
                      <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: STATUS_COLOR[p.status], flexShrink: 0 }} />
                      <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', color: 'var(--mui-text-primary)', fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {p.name}
                      </Typography>
                    </Box>
                    <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: STATUS_COLOR[p.status], textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                      {p.status}
                    </Box>
                  </Box>
                  <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.6875rem', color: 'var(--mui-text-secondary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', mb: 0.5 }}>
                    {p.description}
                  </Typography>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)' }}>
                    <Box>{p.cadence}</Box>
                    <Box>·</Box>
                    <Box>{p.lastRunAt}</Box>
                    <Box>·</Box>
                    <Box>{(p.successRate * 100).toFixed(2)}%</Box>
                  </Box>
                </Box>
              ))}
            </Box>
          </Paper>

          {/* Detail */}
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 1.5, minWidth: 0 }}>
            <Paper sx={{ p: 2 }}>
              <Box sx={{ display: 'flex', alignItems: { xs: 'flex-start', md: 'center' }, justifyContent: 'space-between', gap: 1, flexWrap: 'wrap' }}>
                <Box>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
                    <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: STATUS_COLOR[selected.status], animation: selected.status === 'running' ? 'pulse 2s infinite' : 'none' }} />
                    <Typography sx={{ fontFamily: 'var(--font-headline-md)', fontSize: '1.25rem', color: 'var(--mui-text-primary)', fontWeight: 600 }}>
                      {selected.name}
                    </Typography>
                    <Chip size="small" label={selected.status.toUpperCase()} sx={{ bgcolor: 'var(--mui-bg-elevated)', color: STATUS_COLOR[selected.status], fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', height: 18 }} />
                  </Box>
                  <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', color: 'var(--mui-text-secondary)', mt: 0.5 }}>
                    {selected.description}
                  </Typography>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  {selected.status === 'running' ? (
                    <Button
                      onClick={() => handlePause(selected.id)}
                      startIcon={<PauseIcon sx={{ fontSize: 16 }} />}
                      sx={{ bgcolor: 'var(--mui-bg-subtle)', color: 'var(--mui-text-primary)', fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', px: 1.5, py: 0.75, textTransform: 'none', '&:hover': { bgcolor: 'var(--mui-bg-elevated)' } }}
                    >
                      Pause
                    </Button>
                  ) : selected.status === 'paused' ? (
                    <Button
                      onClick={() => handleResume(selected.id)}
                      startIcon={<PlayArrowIcon sx={{ fontSize: 16 }} />}
                      sx={{ bgcolor: 'var(--mui-bg-subtle)', color: 'var(--mui-primary-light)', fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', px: 1.5, py: 0.75, textTransform: 'none', '&:hover': { bgcolor: 'var(--mui-bg-elevated)' } }}
                    >
                      Resume
                    </Button>
                  ) : null}
                  <Button
                    onClick={() => handleRun(selected.id)}
                    startIcon={<PlayArrowIcon sx={{ fontSize: 16 }} />}
                    sx={{ bgcolor: 'var(--mui-primary-main)', color: 'var(--mui-primary-contrastText)', fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', px: 1.5, py: 0.75, textTransform: 'none', boxShadow: 'none', '&:hover': { bgcolor: 'var(--mui-primary-light)', boxShadow: 'none' } }}
                  >
                    Run Now
                  </Button>
                </Box>
              </Box>

              <Box sx={{ display: 'grid', gridTemplateColumns: { xs: 'repeat(2, 1fr)', md: 'repeat(4, 1fr)' }, gap: 1, mt: 1.5 }}>
                <DetailStat label="Owner"          value={selected.owner} />
                <DetailStat label="Schedule"       value={selected.schedule} />
                <DetailStat label="Last Run"       value={selected.lastRunAt} />
                <DetailStat label="Last Duration"  value={`${(selected.lastDurationMs / 1000).toFixed(1)}s`} />
                <DetailStat label="Success Rate"   value={`${(selected.successRate * 100).toFixed(2)}%`} />
                <DetailStat label="Rows 24h"       value={selected.rowsProcessed24h} />
              </Box>
            </Paper>

            <Paper sx={{ p: 1.5, display: 'flex', flexDirection: 'column', gap: 1 }}>
              <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ minHeight: 36, '& .MuiTab-root': { minHeight: 36, py: 0.5, px: 1.5, fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', textTransform: 'none' } }}>
                <Tab label={<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}><AccountTreeIcon sx={{ fontSize: 16 }} /><span>DAG</span></Box>} />
                <Tab label={<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}><HistoryIcon sx={{ fontSize: 16 }} /><span>Run History</span></Box>} />
                <Tab label={<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}><BarChartIcon sx={{ fontSize: 16 }} /><span>Metrics</span></Box>} />
              </Tabs>
              <Divider sx={{ borderColor: 'var(--mui-border)' }} />

              {tab === 0 && <DagView nodes={selected.nodes} />}

              {tab === 1 && (
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        {['Run ID', 'Started', 'Duration', 'Rows In', 'Rows Out', 'Status', 'Triggered By'].map((h) => (
                          <TableCell key={h} sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: '1px solid var(--mui-border)' }}>
                            {h}
                          </TableCell>
                        ))}
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {runs.filter((r) => r.pipelineId === selected.id || true).map((r) => (
                        <TableRow key={r.id} hover>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)' }}>
                            {r.id}
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                            {r.startedAt}
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                            {r.status === 'running' ? <CircularProgress size={10} sx={{ color: 'var(--mui-tertiary)', mr: 0.5 }} /> : null}
                            {r.durationMs ? `${(r.durationMs / 1000).toFixed(1)}s` : '—'}
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-tertiary)' }}>
                            {r.rowsIn.toLocaleString()}
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-primary-light)' }}>
                            {r.rowsOut.toLocaleString()}
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)' }}>
                            <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, px: 0.75, py: 0.25, borderRadius: '2px', fontSize: '0.625rem', fontFamily: 'var(--font-mono-label)', fontWeight: 600,
                              bgcolor: r.status === 'success' ? 'rgba(107, 216, 203, 0.15)' : r.status === 'failed' ? 'rgba(255, 180, 171, 0.15)' : 'rgba(147, 204, 255, 0.15)',
                              color: r.status === 'success' ? 'var(--mui-primary-light)' : r.status === 'failed' ? 'var(--mui-error)' : 'var(--mui-tertiary)',
                            }}>
                              {r.status === 'success' ? <CheckCircleOutlineIcon sx={{ fontSize: 12 }} /> :
                               r.status === 'failed' ? <ErrorOutlineIcon sx={{ fontSize: 12 }} /> :
                               r.status === 'running' ? <CircularProgress size={10} sx={{ color: 'inherit' }} /> :
                               <HourglassEmptyIcon sx={{ fontSize: 12 }} />}
                              {r.status.toUpperCase()}
                            </Box>
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                            {r.triggeredBy}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}

              {tab === 2 && (
                <Box sx={{ display: 'grid', gridTemplateColumns: { xs: 'repeat(2, 1fr)', md: 'repeat(4, 1fr)' }, gap: 1.5 }}>
                  <DetailStat label="p50 Duration" value={`${Math.round(selected.lastDurationMs * 0.85) / 1000}s`} />
                  <DetailStat label="p95 Duration" value={`${Math.round(selected.lastDurationMs * 1.4) / 1000}s`} />
                  <DetailStat label="p99 Duration" value={`${Math.round(selected.lastDurationMs * 2.1) / 1000}s`} />
                  <DetailStat label="Total Runs 30d" value="720" />
                  <DetailStat label="Failed 30d" value={String(720 - Math.round(720 * selected.successRate))} />
                  <DetailStat label="Bytes 24h" value="12.4 GB" />
                  <DetailStat label="Throughput" value="48K rec/s" />
                  <DetailStat label="Backpressure" value="0%" />
                </Box>
              )}
            </Paper>
          </Box>
        </Box>

        <Snackbar open={snack.open} autoHideDuration={3500} onClose={() => setSnack({ ...snack, open: false })} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
          <Alert severity={snack.severity ?? 'info'} sx={{ width: '100%' }}>{snack.msg}</Alert>
        </Snackbar>
      </Box>
    </AnalyticalShell>
  );
};

const DetailStat: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <Box sx={{ bgcolor: 'var(--mui-bg-subtle)', px: 1.25, py: 1, borderRadius: '4px', display: 'flex', flexDirection: 'column' }}>
    <Typography sx={{ fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
      {label}
    </Typography>
    <Typography sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.8125rem', color: 'var(--mui-text-primary)', fontWeight: 600, mt: 0.5, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
      {value}
    </Typography>
  </Box>
);

const DagView: React.FC<{ nodes: PipelineNode[] }> = ({ nodes }) => (
  <Box sx={{ overflowX: 'auto', py: 2 }}>
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, minWidth: 'fit-content' }}>
      {nodes.map((n, i) => (
        <React.Fragment key={n.id}>
          <Box
            sx={{
              bgcolor: 'var(--mui-bg-subtle)',
              border: '1.5px solid',
              borderColor: n.status === 'error' ? 'var(--mui-error)' : n.status === 'running' ? 'var(--mui-tertiary)' : 'var(--mui-border)',
              borderRadius: '4px',
              p: 1.25,
              minWidth: 180,
              display: 'flex',
              flexDirection: 'column',
              gap: 0.5,
              position: 'relative',
              animation: n.status === 'running' ? 'pulse 2s infinite' : 'none',
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, color: NODE_STATUS_COLOR[n.status] }}>
                {NODE_ICON[n.kind]}
              </Box>
              <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.5625rem', color: NODE_STATUS_COLOR[n.status], textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                {n.kind}
              </Box>
            </Box>
            <Typography sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)', fontWeight: 600 }}>
              {n.label}
            </Typography>
            {n.meta && (
              <Typography sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)' }}>
                {n.meta}
              </Typography>
            )}
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mt: 0.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: NODE_STATUS_COLOR[n.status] }} />
                <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: NODE_STATUS_COLOR[n.status], textTransform: 'uppercase' }}>
                  {n.status}
                </Box>
              </Box>
              <IconButton size="small" sx={{ color: 'var(--mui-text-secondary)', p: 0 }}>
                <MoreVertIcon sx={{ fontSize: 14 }} />
              </IconButton>
            </Box>
          </Box>
          {i < nodes.length - 1 && (
            <Box sx={{ display: 'flex', alignItems: 'center', color: 'var(--mui-text-secondary)' }}>
              {n.kind === 'loop' || nodes[i + 1].kind === 'loop' ? <LoopIcon sx={{ fontSize: 18 }} /> : <ArrowForwardIcon sx={{ fontSize: 18 }} />}
            </Box>
          )}
        </React.Fragment>
      ))}
    </Box>
  </Box>
);

export default PipelinesPage;
