import React, { useState, useMemo, useCallback } from 'react';
import {
  Box, Typography, Button, IconButton, Tooltip, Chip, Paper, Tabs, Tab, InputBase,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Divider, Snackbar, Alert,
  Menu, MenuItem, Switch, FormControlLabel,
} from '@mui/material';
import {
  Assessment as AssessmentIcon, Description as DescriptionIcon, PlayArrow as PlayArrowIcon,
  Schedule as ScheduleIcon, History as HistoryIcon, Refresh as RefreshIcon, Add as AddIcon,
  Edit as EditIcon, Delete as DeleteIcon, MoreVert as MoreVertIcon, Search as SearchIcon,
  FilterList as FilterListIcon, GetApp as GetAppIcon, Email as EmailIcon, PictureAsPdf as PdfIcon,
  Description as DescriptionIconAlt, TableChart as TableChartIcon, BarChart as BarChartIcon,
  PieChart as PieChartIcon, ShowChart as ShowChartIcon, Star as StarIcon, StarBorder as StarBorderIcon,
  CheckCircle as CheckCircleIcon, ErrorOutline as ErrorOutlineIcon,
  Send as SendIcon, Edit as EditIconAlt, FileCopy as FileCopyIcon, ScheduleSend as ScheduleSendIcon,
} from '@mui/icons-material';
import AnalyticalShell from '../../components/analytical/AnalyticalShell';
import apiClient from '../../utils/apiClient';
import { devLog } from '../../utils/devLogger';

type OutputFormat = 'PDF' | 'CSV' | 'XLSX' | 'JSON' | 'HTML';

interface ReportTemplate {
  id: string;
  name: string;
  category: string;
  description: string;
  icon: React.ReactNode;
  defaultFormat: OutputFormat;
  estimatedRows: string;
}

interface SavedReport {
  id: string;
  name: string;
  templateId: string;
  format: OutputFormat;
  cadence: 'Manual' | 'Daily' | 'Weekly' | 'Monthly' | 'Quarterly';
  nextRun: string;
  lastRun: string;
  status: 'success' | 'failed' | 'running' | 'queued' | 'paused';
  recipients: number;
  starred?: boolean;
}

interface ScheduleSlot {
  id: string;
  reportId: string;
  reportName: string;
  nextRunAt: string;
  format: OutputFormat;
  recipients: number;
  cadence: string;
}

const TEMPLATES: ReportTemplate[] = [
  { id: 't-1', name: 'Portfolio Holdings',         category: 'Holdings',   description: 'Current positions, weights, and market valuations.',                  icon: <TableChartIcon sx={{ fontSize: 18, color: 'var(--mui-primary-light)' }} />, defaultFormat: 'PDF',  estimatedRows: '~500 rows' },
  { id: 't-2', name: 'Daily P&L Summary',          category: 'P&L',        description: 'Realized vs unrealized gains, intraday attribution.',              icon: <BarChartIcon sx={{ fontSize: 18, color: 'var(--mui-warning-main)' }} />,  defaultFormat: 'PDF',  estimatedRows: '~120 rows' },
  { id: 't-3', name: 'Risk Compliance Snapshot',   category: 'Risk',       description: 'VaR, exposure limits, sector concentration, breach summary.',        icon: <ErrorOutlineIcon sx={{ fontSize: 18, color: 'var(--mui-error)' }} />,          defaultFormat: 'PDF',  estimatedRows: '~80 rows' },
  { id: 't-4', name: 'Trade Settlement Audit',    category: 'Operations', description: 'Failed settlements, fails, late confirms.',                          icon: <ScheduleIcon sx={{ fontSize: 18, color: 'var(--mui-secondary-main)' }} />,  defaultFormat: 'CSV',  estimatedRows: '~340 rows' },
  { id: 't-5', name: 'Counterparty Exposure',     category: 'Risk',       description: 'Top counterparties by gross/net exposure and CSA utilization.',     icon: <PieChartIcon sx={{ fontSize: 18, color: 'var(--mui-tertiary)' }} />,        defaultFormat: 'XLSX', estimatedRows: '~210 rows' },
  { id: 't-6', name: 'Regulatory Filing Pack',     category: 'Compliance', description: 'Pre-formatted SEC / FINRA filing pack with disclosure tables.',      icon: <DescriptionIcon sx={{ fontSize: 18, color: 'var(--mui-secondary-main)' }} />, defaultFormat: 'PDF',  estimatedRows: '~40 pages' },
  { id: 't-7', name: 'CUSIP Concentration Limit', category: 'Risk',       description: 'Tier-1 limits, breach alerts, waiver approvals.',                    icon: <ShowChartIcon sx={{ fontSize: 18, color: 'var(--mui-warning-main)' }} />,   defaultFormat: 'PDF',  estimatedRows: '~150 rows' },
  { id: 't-8', name: 'Liquidity Stress Test',     category: 'Risk',       description: 'Bond cashflow projection under three rate scenarios.',              icon: <ShowChartIcon sx={{ fontSize: 18, color: 'var(--mui-tertiary)' }} />,     defaultFormat: 'XLSX', estimatedRows: '~640 rows' },
];

const REPORTS: SavedReport[] = [
  { id: 'r-1', name: 'Daily Holdings — EQ Book',       templateId: 't-1', format: 'PDF',  cadence: 'Daily',     nextRun: 'Today 16:30 ET',  lastRun: 'Yesterday 16:32',  status: 'success', recipients: 12, starred: true },
  { id: 'r-2', name: 'P&L Summary — Macro Desk',         templateId: 't-2', format: 'PDF',  cadence: 'Daily',     nextRun: 'Today 17:00 ET',  lastRun: 'Yesterday 17:01',  status: 'success', recipients: 8 },
  { id: 'r-3', name: 'Risk VaR — End of Day',            templateId: 't-3', format: 'PDF',  cadence: 'Daily',     nextRun: 'Today 18:00 ET',  lastRun: 'Yesterday 18:01',  status: 'success', recipients: 14, starred: true },
  { id: 'r-4', name: 'Settlement Fails — T+1',           templateId: 't-4', format: 'CSV',  cadence: 'Daily',     nextRun: 'Tomorrow 09:00', lastRun: 'Today 09:00',      status: 'success', recipients: 5 },
  { id: 'r-5', name: 'Top Counterparties — Weekly',      templateId: 't-5', format: 'XLSX', cadence: 'Weekly',    nextRun: 'Friday 09:00',    lastRun: 'Last Friday 09:01', status: 'success', recipients: 11 },
  { id: 'r-6', name: 'Quarterly Filing Pack — Q3',       templateId: 't-6', format: 'PDF',  cadence: 'Quarterly', nextRun: 'Oct 15 09:00',    lastRun: 'Jul 15 09:00',    status: 'success', recipients: 22 },
  { id: 'r-7', name: 'CUSIP Tier-1 Breach Watch',       templateId: 't-7', format: 'PDF',  cadence: 'Daily',     nextRun: 'Today 17:30 ET',  lastRun: 'Yesterday 17:30',  status: 'failed',  recipients: 6 },
  { id: 'r-8', name: 'Liquidity Stress — Q4 Run',        templateId: 't-8', format: 'XLSX', cadence: 'Monthly',   nextRun: 'Oct 01 06:00',    lastRun: 'Sep 01 06:00',    status: 'success', recipients: 9 },
  { id: 'r-9', name: 'On-Demand — Holdings Adhoc',       templateId: 't-1', format: 'CSV',  cadence: 'Manual',    nextRun: '—',               lastRun: '4h ago',          status: 'success', recipients: 3 },
  { id: 'r-10', name: 'Exposure by Desk — Weekly',       templateId: 't-5', format: 'PDF',  cadence: 'Weekly',    nextRun: 'Monday 09:00',    lastRun: 'Last Monday 09:01', status: 'paused',  recipients: 7 },
];

const SCHEDULE: ScheduleSlot[] = [
  { id: 's-1', reportId: 'r-3', reportName: 'Risk VaR — End of Day',    nextRunAt: 'Today 18:00 ET',   format: 'PDF',  recipients: 14, cadence: 'Daily 18:00' },
  { id: 's-2', reportId: 'r-1', reportName: 'Daily Holdings — EQ Book', nextRunAt: 'Today 16:30 ET',   format: 'PDF',  recipients: 12, cadence: 'Daily 16:30' },
  { id: 's-3', reportId: 'r-4', reportName: 'Settlement Fails — T+1',   nextRunAt: 'Tomorrow 09:00',  format: 'CSV',  recipients: 5,  cadence: 'Daily 09:00' },
  { id: 's-4', reportId: 'r-5', reportName: 'Top Counterparties — Wk',  nextRunAt: 'Friday 09:00',    format: 'XLSX', recipients: 11, cadence: 'Weekly Fri 09:00' },
  { id: 's-5', reportId: 'r-6', reportName: 'Quarterly Filing Pack',    nextRunAt: 'Oct 15 09:00',    format: 'PDF',  recipients: 22, cadence: 'Quarterly Q+14' },
];

const ReportsPage: React.FC = () => {
  const [tab, setTab] = useState(0);
  const [reports, setReports] = useState<SavedReport[]>(REPORTS);
  const [search, setSearch] = useState('');
  const [snack, setSnack] = useState<{ open: boolean; msg: string; severity?: 'success' | 'info' | 'error' }>({ open: false, msg: '' });
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
  const [activeReportId, setActiveReportId] = useState<string | null>(null);

  const showSnack = useCallback((msg: string, severity: 'success' | 'info' | 'error' = 'info') => setSnack({ open: true, msg, severity }), []);

  const visibleReports = useMemo(() => {
    const q = search.toLowerCase();
    if (!q) return reports;
    return reports.filter((r) => r.name.toLowerCase().includes(q) || r.cadence.toLowerCase().includes(q));
  }, [reports, search]);

  const totals = useMemo(() => ({
    total: reports.length,
    success: reports.filter((r) => r.status === 'success').length,
    failed:  reports.filter((r) => r.status === 'failed').length,
    running: reports.filter((r) => r.status === 'running').length,
    starred: reports.filter((r) => r.starred).length,
    recipients: reports.reduce((acc, r) => acc + r.recipients, 0),
  }), [reports]);

  const handleRun = useCallback(async (id: string) => {
    const r = reports.find((x) => x.id === id);
    setReports((prev) => prev.map((x) => (x.id === id ? { ...x, status: 'running' } : x)));
    showSnack(`Running ${r?.name}…`, 'info');
    try {
      await apiClient<unknown>(`/api/reports/${id}/run`, { method: 'POST' });
    } catch {
      // fallback local simulation
    }
    setTimeout(() => {
      setReports((prev) => prev.map((x) => (x.id === id ? { ...x, status: 'success', lastRun: 'just now' } : x)));
      showSnack(`${r?.name} completed`, 'success');
    }, 700);
  }, [reports, showSnack]);

  const handleToggleStar = useCallback((id: string) => {
    setReports((prev) => prev.map((r) => (r.id === id ? { ...r, starred: !r.starred } : r)));
  }, []);

  const handleDelete = useCallback((id: string) => {
    const r = reports.find((x) => x.id === id);
    setReports((prev) => prev.filter((x) => x.id !== id));
    showSnack(`Deleted ${r?.name}`, 'info');
  }, [reports, showSnack]);

  const handleMenu = useCallback((e: React.MouseEvent<HTMLElement>, id: string) => {
    setAnchorEl(e.currentTarget);
    setActiveReportId(id);
  }, []);

  const handleMenuClose = useCallback(() => {
    setAnchorEl(null);
    setActiveReportId(null);
  }, []);

  return (
    <AnalyticalShell>
      <Box sx={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, bgcolor: 'var(--mui-bg-default)' }}>
        {/* Header */}
        <Paper sx={{ p: 2, display: 'flex', flexDirection: { xs: 'column', md: 'row' }, alignItems: { md: 'center' }, justifyContent: 'space-between', gap: 1.5, mx: 2, mt: 2, borderRadius: '4px' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Box sx={{ p: 1, borderRadius: '4px', bgcolor: 'var(--mui-bg-subtle)', color: 'var(--mui-warning-main)', display: 'flex', alignItems: 'center' }}>
              <AssessmentIcon sx={{ fontSize: 22 }} />
            </Box>
            <Box>
              <Typography sx={{ fontFamily: 'var(--font-headline-md)', fontSize: '1.25rem', color: 'var(--mui-text-primary)', fontWeight: 600, letterSpacing: '-0.01em' }}>
                Reports
              </Typography>
              <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                Scheduled, ad-hoc, and template-driven reports with multi-format export and email distribution.
              </Typography>
            </Box>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
            <Box sx={{ bgcolor: 'var(--mui-bg-paper)', px: 1.5, py: 0.75, borderRadius: '4px', display: 'flex', alignItems: 'center', gap: 1 }}>
              <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: 'var(--mui-primary-light)' }} />
              <Typography className="font-mono" sx={{ fontSize: '0.6875rem', color: 'var(--mui-text-primary)' }}>
                {totals.success}/{totals.total} OK
              </Typography>
            </Box>
            {totals.failed > 0 && (
              <Box sx={{ bgcolor: 'var(--mui-bg-paper)', px: 1.5, py: 0.75, borderRadius: '4px', display: 'flex', alignItems: 'center', gap: 1 }}>
                <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: 'var(--mui-error)' }} />
                <Typography className="font-mono" sx={{ fontSize: '0.6875rem', color: 'var(--mui-text-primary)' }}>
                  {totals.failed} failed
                </Typography>
              </Box>
            )}
            <Box sx={{ bgcolor: 'var(--mui-bg-paper)', px: 1.5, py: 0.75, borderRadius: '4px', display: 'flex', alignItems: 'center', gap: 1 }}>
              <EmailIcon sx={{ fontSize: 14, color: 'var(--mui-secondary-main)' }} />
              <Typography className="font-mono" sx={{ fontSize: '0.6875rem', color: 'var(--mui-text-primary)' }}>
                {totals.recipients} subscribers
              </Typography>
            </Box>
            <Button
              startIcon={<AddIcon sx={{ fontSize: 18 }} />}
              onClick={() => { setTab(1); showSnack('Pick a template to start', 'info'); }}
              sx={{ bgcolor: 'var(--mui-primary-main)', color: 'var(--mui-primary-contrastText)', fontFamily: 'var(--font-headline-sm)', fontSize: '0.8125rem', px: 1.5, py: 1, textTransform: 'none', boxShadow: 'none', '&:hover': { bgcolor: 'var(--mui-primary-light)', boxShadow: 'none' } }}
            >
              New Report
            </Button>
          </Box>
        </Paper>

        <Box sx={{ mx: 2, mt: 1.5, display: 'flex', flexDirection: 'column', gap: 1.5, flex: 1, minHeight: 0 }}>
          <Paper sx={{ px: 1.5, py: 0.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ minHeight: 36, '& .MuiTab-root': { minHeight: 36, py: 0.5, px: 1.5, fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', textTransform: 'none' } }}>
              <Tab label={<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}><DescriptionIcon sx={{ fontSize: 16 }} /><span>Saved Reports</span></Box>} />
              <Tab label={<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}><FileCopyIcon sx={{ fontSize: 16 }} /><span>Templates</span></Box>} />
              <Tab label={<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}><ScheduleIcon sx={{ fontSize: 16 }} /><span>Schedule</span></Box>} />
              <Tab label={<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}><HistoryIcon sx={{ fontSize: 16 }} /><span>History</span></Box>} />
            </Tabs>
            <Box sx={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
              <SearchIcon sx={{ position: 'absolute', left: 8, color: 'var(--mui-text-secondary)', fontSize: 16, pointerEvents: 'none' }} />
              <InputBase
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search reports…"
                sx={{ bgcolor: 'var(--mui-bg-subtle)', color: 'var(--mui-text-primary)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', pl: 4, pr: 1, py: 0.25, borderRadius: '3px', width: 220, '& input': { padding: 0 } }}
              />
            </Box>
          </Paper>

          {/* SAVED REPORTS */}
          {tab === 0 && (
            <Paper sx={{ overflow: 'hidden' }}>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      {['Report', 'Format', 'Cadence', 'Next Run', 'Last Run', 'Status', 'Subs', ''].map((h) => (
                        <TableCell key={h} sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: '1px solid var(--mui-border)' }}>
                          {h}
                        </TableCell>
                      ))}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {visibleReports.map((r) => (
                      <TableRow key={r.id} hover>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', py: 1 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
                            <IconButton size="small" onClick={() => handleToggleStar(r.id)} sx={{ p: 0.25, color: r.starred ? 'var(--mui-warning-main)' : 'var(--mui-text-secondary)' }}>
                              {r.starred ? <StarIcon sx={{ fontSize: 14 }} /> : <StarBorderIcon sx={{ fontSize: 14 }} />}
                            </IconButton>
                            <Box>
                              <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', color: 'var(--mui-text-primary)', fontWeight: 600 }}>
                                {r.name}
                              </Typography>
                              <Typography sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)' }}>
                                {r.id}
                              </Typography>
                            </Box>
                          </Box>
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)' }}>
                          <FormatChip format={r.format} />
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)' }}>
                          {r.cadence}
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)' }}>
                          {r.nextRun}
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                          {r.lastRun}
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)' }}>
                          <StatusChip status={r.status} />
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)', textAlign: 'center' }}>
                          {r.recipients}
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', textAlign: 'right' }}>
                          <Tooltip title="Run now">
                            <IconButton size="small" onClick={() => handleRun(r.id)} disabled={r.status === 'running'} sx={{ color: 'var(--mui-primary-light)' }}>
                              <PlayArrowIcon sx={{ fontSize: 14 }} />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title="Download latest">
                            <IconButton size="small" onClick={() => showSnack('Download started', 'success')} sx={{ color: 'var(--mui-text-secondary)' }}>
                              <GetAppIcon sx={{ fontSize: 14 }} />
                            </IconButton>
                          </Tooltip>
                          <IconButton size="small" onClick={(e) => handleMenu(e, r.id)} sx={{ color: 'var(--mui-text-secondary)' }}>
                            <MoreVertIcon sx={{ fontSize: 14 }} />
                          </IconButton>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Paper>
          )}

          {/* TEMPLATES */}
          {tab === 1 && (
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(3, 1fr)', lg: 'repeat(4, 1fr)' }, gap: 1.5 }}>
              {TEMPLATES.map((t) => (
                <Paper
                  key={t.id}
                  sx={{
                    p: 1.5,
                    cursor: 'pointer',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 1,
                    transition: 'all 0.15s',
                    '&:hover': { borderColor: 'var(--mui-primary-main)', transform: 'translateY(-1px)' },
                  }}
                  onClick={() => {
                    const newReport: SavedReport = {
                      id: `r-${Date.now()}`,
                      name: `${t.name} (draft)`,
                      templateId: t.id,
                      format: t.defaultFormat,
                      cadence: 'Manual',
                      nextRun: '—',
                      lastRun: '—',
                      status: 'queued',
                      recipients: 0,
                    };
                    setReports((prev) => [newReport, ...prev]);
                    setTab(0);
                    showSnack(`Created report from ${t.name}`, 'success');
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Box sx={{ width: 36, height: 36, borderRadius: '4px', bgcolor: 'var(--mui-bg-subtle)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                      {t.icon}
                    </Box>
                    <FormatChip format={t.defaultFormat} />
                  </Box>
                  <Box>
                    <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.875rem', color: 'var(--mui-text-primary)', fontWeight: 600 }}>
                      {t.name}
                    </Typography>
                    <Typography sx={{ fontFamily: 'var(--font-data-label)', fontSize: '0.5625rem', color: 'var(--mui-secondary-main)', textTransform: 'uppercase', letterSpacing: '0.05em', mt: 0.25 }}>
                      {t.category}
                    </Typography>
                  </Box>
                  <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                    {t.description}
                  </Typography>
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mt: 'auto' }}>
                    <Typography className="font-mono" sx={{ fontSize: '0.625rem', color: 'var(--mui-text-secondary)' }}>
                      {t.estimatedRows}
                    </Typography>
                    <Button
                      size="small"
                      startIcon={<AddIcon sx={{ fontSize: 12 }} />}
                      sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: 'var(--mui-primary-light)', textTransform: 'uppercase', letterSpacing: '0.05em', minWidth: 0, px: 0.75, py: 0.25 }}
                    >
                      Use
                    </Button>
                  </Box>
                </Paper>
              ))}
            </Box>
          )}

          {/* SCHEDULE */}
          {tab === 2 && (
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', lg: '2fr 1fr' }, gap: 1.5 }}>
              <Paper sx={{ overflow: 'hidden' }}>
                <Box sx={{ p: 1.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--mui-border)' }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <ScheduleSendIcon sx={{ fontSize: 18, color: 'var(--mui-primary-light)' }} />
                    <Typography sx={{ fontFamily: 'var(--font-headline-sm)', fontSize: '0.95rem', color: 'var(--mui-text-primary)', fontWeight: 600 }}>
                      Upcoming Runs (next 24h)
                    </Typography>
                  </Box>
                  <Box className="font-mono" sx={{ fontSize: '0.6875rem', color: 'var(--mui-text-secondary)' }}>
                    America/New_York
                  </Box>
                </Box>
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        {['Report', 'Cadence', 'Next Run', 'Format', 'Recipients', ''].map((h) => (
                          <TableCell key={h} sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: '1px solid var(--mui-border)' }}>
                            {h}
                          </TableCell>
                        ))}
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {SCHEDULE.map((s) => (
                        <TableRow key={s.id} hover>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', color: 'var(--mui-text-primary)', fontWeight: 600 }}>
                            {s.reportName}
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)' }}>
                            {s.cadence}
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-secondary-main)' }}>
                            {s.nextRunAt}
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)' }}>
                            <FormatChip format={s.format} />
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)', textAlign: 'center' }}>
                            {s.recipients}
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', textAlign: 'right' }}>
                            <Tooltip title="Run now">
                              <IconButton size="small" onClick={() => handleRun(s.reportId)} sx={{ color: 'var(--mui-primary-light)' }}>
                                <PlayArrowIcon sx={{ fontSize: 14 }} />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Reschedule">
                              <IconButton size="small" sx={{ color: 'var(--mui-text-secondary)' }}>
                                <ScheduleIcon sx={{ fontSize: 14 }} />
                              </IconButton>
                            </Tooltip>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Paper>

              <Paper sx={{ p: 1.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
                  <EmailIcon sx={{ fontSize: 18, color: 'var(--mui-secondary-main)' }} />
                  <Typography sx={{ fontFamily: 'var(--font-headline-sm)', fontSize: '0.95rem', color: 'var(--mui-text-primary)', fontWeight: 600 }}>
                    Distribution Channels
                  </Typography>
                </Box>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  <ChannelRow icon={<EmailIcon sx={{ fontSize: 16 }} />} title="Email digest" subtitle="Daily digest at 18:00 ET" enabled count={totals.recipients} onToggle={() => showSnack('Email digest toggled', 'info')} />
                  <ChannelRow icon={<SendIcon sx={{ fontSize: 16 }} />} title="Slack #risk-reports" subtitle="Push to channel on completion" enabled count={0} onToggle={() => showSnack('Slack channel toggled', 'info')} />
                  <ChannelRow icon={<FileCopyIcon sx={{ fontSize: 16 }} />} title="S3 archive" subtitle="s3://uisce-reports/{yyyy}/{mm}" enabled count={0} onToggle={() => showSnack('S3 archive toggled', 'info')} />
                  <ChannelRow icon={<ScheduleSendIcon sx={{ fontSize: 16 }} />} title="API webhook" subtitle="POST to /hooks/reports" enabled={false} count={0} onToggle={() => showSnack('Webhook toggled', 'info')} />
                </Box>
                <Divider sx={{ my: 1.5, borderColor: 'var(--mui-border)' }} />
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Typography sx={{ fontFamily: 'var(--font-data-label)', fontSize: '0.6875rem', color: 'var(--mui-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                    Failure retries
                  </Typography>
                  <Typography className="font-mono" sx={{ fontSize: '0.75rem', color: 'var(--mui-text-primary)' }}>
                    3 attempts • exponential backoff
                  </Typography>
                </Box>
              </Paper>
            </Box>
          )}

          {/* HISTORY */}
          {tab === 3 && (
            <Paper sx={{ overflow: 'hidden' }}>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      {['Timestamp', 'Report', 'Status', 'Duration', 'Format', 'Subs', 'Triggered By'].map((h) => (
                        <TableCell key={h} sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: '1px solid var(--mui-border)' }}>
                          {h}
                        </TableCell>
                      ))}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {reports.flatMap((r) => [
                      { ts: r.lastRun, name: r.name, status: r.status, duration: '4.2s', format: r.format, subs: r.recipients, by: r.cadence === 'Manual' ? 'manual' : r.cadence.toLowerCase() },
                      { ts: '2 days ago', name: r.name, status: 'success', duration: '4.1s', format: r.format, subs: r.recipients, by: r.cadence === 'Manual' ? 'manual' : r.cadence.toLowerCase() },
                      { ts: '3 days ago', name: r.name, status: 'success', duration: '3.9s', format: r.format, subs: r.recipients, by: r.cadence === 'Manual' ? 'manual' : r.cadence.toLowerCase() },
                    ]).map((row, i) => (
                      <TableRow key={i} hover>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)' }}>
                          {row.ts}
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', color: 'var(--mui-text-primary)' }}>
                          {row.name}
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)' }}>
                          <StatusChip status={row.status as SavedReport['status']} />
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                          {row.duration}
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)' }}>
                          <FormatChip format={row.format as OutputFormat} />
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)', textAlign: 'center' }}>
                          {row.subs}
                        </TableCell>
                        <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                          {row.by}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Paper>
          )}
        </Box>

        <Menu anchorEl={anchorEl} open={!!anchorEl} onClose={handleMenuClose}>
          <MenuItem onClick={() => { if (activeReportId) handleRun(activeReportId); handleMenuClose(); }}>
            <PlayArrowIcon sx={{ fontSize: 16, mr: 1 }} /> Run now
          </MenuItem>
          <MenuItem onClick={() => { showSnack('Recipients dialog opened', 'info'); handleMenuClose(); }}>
            <EmailIcon sx={{ fontSize: 16, mr: 1 }} /> Manage recipients
          </MenuItem>
          <MenuItem onClick={() => { showSnack('Edit form opened', 'info'); handleMenuClose(); }}>
            <EditIconAlt sx={{ fontSize: 16, mr: 1 }} /> Edit
          </MenuItem>
          <MenuItem onClick={() => { if (activeReportId) handleDelete(activeReportId); handleMenuClose(); }}>
            <DeleteIcon sx={{ fontSize: 16, mr: 1, color: 'var(--mui-error)' }} /> <Box component="span" sx={{ color: 'var(--mui-error)' }}>Delete</Box>
          </MenuItem>
        </Menu>

        <Snackbar open={snack.open} autoHideDuration={3500} onClose={() => setSnack({ ...snack, open: false })} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
          <Alert severity={snack.severity ?? 'info'} sx={{ width: '100%' }}>{snack.msg}</Alert>
        </Snackbar>
      </Box>
    </AnalyticalShell>
  );
};

const FormatChip: React.FC<{ format: OutputFormat }> = ({ format }) => {
  const meta: Record<OutputFormat, { color: string; bg: string }> = {
    PDF:  { color: '#ffdad6', bg: '#93000a' },
    CSV:  { color: 'var(--mui-primary-light)', bg: 'rgba(107, 216, 203, 0.15)' },
    XLSX: { color: 'var(--mui-tertiary)', bg: 'rgba(147, 204, 255, 0.15)' },
    JSON: { color: 'var(--mui-warning-main)', bg: 'rgba(251, 146, 60, 0.15)' },
    HTML: { color: 'var(--mui-secondary-main)', bg: 'rgba(192, 198, 221, 0.15)' },
  };
  const icon: Record<OutputFormat, React.ReactNode> = {
    PDF:  <PdfIcon sx={{ fontSize: 12 }} />,
    CSV:  <TableChartIcon sx={{ fontSize: 12 }} />,
    XLSX: <TableChartIcon sx={{ fontSize: 12 }} />,
    JSON: <DescriptionIcon sx={{ fontSize: 12 }} />,
    HTML: <DescriptionIcon sx={{ fontSize: 12 }} />,
  };
  return (
    <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, px: 0.75, py: 0.25, borderRadius: '2px', bgcolor: meta[format].bg, color: meta[format].color, fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', fontWeight: 700 }}>
      {icon[format]}
      {format}
    </Box>
  );
};

const StatusChip: React.FC<{ status: SavedReport['status'] }> = ({ status }) => {
  const meta = {
    success: { color: 'var(--mui-primary-light)', bg: 'rgba(107, 216, 203, 0.15)', label: 'OK' },
    failed:  { color: 'var(--mui-error)',          bg: 'rgba(255, 180, 171, 0.15)', label: 'FAILED' },
    running: { color: 'var(--mui-tertiary)',       bg: 'rgba(147, 204, 255, 0.15)', label: 'RUNNING' },
    queued:  { color: 'var(--mui-warning-main)',   bg: 'rgba(251, 146, 60, 0.15)',  label: 'QUEUED' },
    paused:  { color: 'var(--mui-text-secondary)', bg: 'var(--mui-bg-elevated)',     label: 'PAUSED' },
  }[status];
  return (
    <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, px: 0.75, py: 0.25, borderRadius: '2px', bgcolor: meta.bg, color: meta.color, fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', fontWeight: 700 }}>
      {status === 'success' ? <CheckCircleIcon sx={{ fontSize: 12 }} /> :
       status === 'failed' ? <ErrorOutlineIcon sx={{ fontSize: 12 }} /> : null}
      {meta.label}
    </Box>
  );
};

const ChannelRow: React.FC<{ icon: React.ReactNode; title: string; subtitle: string; enabled: boolean; count: number; onToggle: () => void }> = ({ icon, title, subtitle, enabled, count, onToggle }) => (
  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', p: 1, borderRadius: '4px', bgcolor: 'var(--mui-bg-subtle)' }}>
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <Box sx={{ color: 'var(--mui-secondary-main)' }}>{icon}</Box>
      <Box>
        <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', color: 'var(--mui-text-primary)', fontWeight: 600 }}>
          {title}
        </Typography>
        <Typography sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)' }}>
          {subtitle}{count > 0 ? ` · ${count} subscribers` : ''}
        </Typography>
      </Box>
    </Box>
    <Switch size="small" checked={enabled} onChange={onToggle} />
  </Box>
);

export default ReportsPage;
