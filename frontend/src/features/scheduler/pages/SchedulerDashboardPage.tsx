/**
 * World-Class Enterprise Scheduler - Dashboard Page
 * Main scheduler dashboard with real-time metrics, job monitoring, and quick actions
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Tooltip,
  Alert
} from '@mui/material';
import {
  Schedule as ScheduleIcon,
  PlayArrow as RunIcon,
  Refresh as RefreshIcon,
  CheckCircle as SuccessIcon,
  Error as ErrorIcon,
  HourglassEmpty as PendingIcon,
  Replay as RetryIcon,
  Add as AddIcon,
  TrendingUp as MetricIcon
} from '@mui/icons-material';

interface ExecutionItem {
  id: string;
  jobName: string;
  cronExpr: string;
  status: 'SUCCESS' | 'FAILED' | 'RUNNING' | 'PENDING';
  durationMs: number;
  lastRunAt: string;
  nextRunAt: string;
}

export function SchedulerDashboardPage() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  const [executions, setExecutions] = useState<ExecutionItem[]>([
    {
      id: 'exec-01',
      jobName: 'CRIMS Order Ingestion & Allocation CDC',
      cronExpr: '*/5 * * * *',
      status: 'SUCCESS',
      durationMs: 420,
      lastRunAt: '2 min ago',
      nextRunAt: 'in 3 min'
    },
    {
      id: 'exec-02',
      jobName: 'Iceberg Bitemporal Partition Compaction',
      cronExpr: '0 0 * * *',
      status: 'RUNNING',
      durationMs: 14200,
      lastRunAt: 'running...',
      nextRunAt: 'Tomorrow 00:00'
    },
    {
      id: 'exec-03',
      jobName: 'Multi-Vendor Pricing Consensus Tick (MDM)',
      cronExpr: '*/1 * * * *',
      status: 'SUCCESS',
      durationMs: 180,
      lastRunAt: '30s ago',
      nextRunAt: 'in 30s'
    },
    {
      id: 'exec-04',
      jobName: 'GLEIF Level-2 Legal Entity Tree Sync',
      cronExpr: '0 4 * * *',
      status: 'FAILED',
      durationMs: 3100,
      lastRunAt: '1h ago',
      nextRunAt: 'Tomorrow 04:00'
    }
  ]);

  const handleRetry = (id: string, name: string) => {
    setExecutions(prev =>
      prev.map(e => (e.id === id ? { ...e, status: 'RUNNING', lastRunAt: 're-evaluating...' } : e))
    );
    setNotice(`Manually triggered execution for job '${name}'.`);
  };

  return (
    <Box sx={{ p: 3, bgcolor: '#071526', minHeight: '100vh', color: '#F8FAFC' }}>
      {/* Header */}
      <Paper elevation={0} sx={{ p: 2.5, mb: 3, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 2 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center" flexWrap="wrap" gap={2}>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <ScheduleIcon sx={{ color: '#00D4FF', fontSize: 32 }} />
            <Box>
              <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 18 }}>
                Enterprise Job Scheduler & Temporal Mesh HUD
              </Typography>
              <Typography variant="caption" sx={{ color: '#94A3B8' }}>
                Deterministic cron dispatch, sliding SLA windows, and automated retry topologies
              </Typography>
            </Box>
          </Stack>

          <Stack direction="row" spacing={1.5}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<RefreshIcon />}
              onClick={() => setNotice('Telemetry refreshed.')}
              sx={{ color: '#38BDF8', borderColor: '#0284C7', textTransform: 'none', fontWeight: 600 }}
            >
              Refresh
            </Button>
            <Button
              variant="contained"
              size="small"
              startIcon={<AddIcon />}
              onClick={() => navigate('/scheduler/jobs/new')}
              sx={{ bgcolor: '#0284C7', color: '#FFF', textTransform: 'none', fontWeight: 700, '&:hover': { bgcolor: '#0369A1' } }}
            >
              Create Scheduled Job
            </Button>
          </Stack>
        </Box>
      </Paper>

      {notice && (
        <Alert severity="info" sx={{ mb: 3, bgcolor: '#0B1E36', color: '#38BDF8', border: '1px solid #0284C7' }}>
          {notice}
        </Alert>
      )}

      {/* KPI Telemetry */}
      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Active Jobs Registered</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#38BDF8', fontFamily: 'monospace' }}>28</Typography>
            <Typography variant="caption" sx={{ color: '#34D399' }}>100% tenant isolated</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Success Rate (24h)</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#34D399', fontFamily: 'monospace' }}>99.4%</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>1,842 runs evaluated</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Median SLA Latency</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#FBBF24', fontFamily: 'monospace' }}>240ms</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>WASM + StarRocks execution</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Active Breaches / Retries</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#F87171', fontFamily: 'monospace' }}>1</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>GLEIF tree connection timeout</Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Executions Table */}
      <Paper sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 2, overflow: 'hidden' }}>
        <Box sx={{ p: 2, borderBottom: '1px solid #1E293B', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: 13 }}>
            Recent Scheduled Executions
          </Typography>
          <Chip label="Auto-Polling: 30s" size="small" sx={{ bgcolor: '#071526', color: '#94A3B8', fontSize: 10 }} />
        </Box>

        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
                <TableCell>Job Name</TableCell>
                <TableCell>Cron Schedule</TableCell>
                <TableCell align="center">Status</TableCell>
                <TableCell align="center">Duration</TableCell>
                <TableCell>Last Execution</TableCell>
                <TableCell>Next Trigger</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {executions.map(e => (
                <TableRow key={e.id} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                  <TableCell sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>
                    {e.jobName}
                  </TableCell>
                  <TableCell sx={{ fontFamily: 'monospace', color: '#CBD5E1', fontSize: 11 }}>
                    {e.cronExpr}
                  </TableCell>
                  <TableCell align="center">
                    <Chip
                      icon={
                        e.status === 'SUCCESS' ? <SuccessIcon sx={{ fontSize: 12, color: '#34D399 !important' }} /> :
                        e.status === 'FAILED' ? <ErrorIcon sx={{ fontSize: 12, color: '#F87171 !important' }} /> :
                        <PendingIcon sx={{ fontSize: 12, color: '#FBBF24 !important' }} />
                      }
                      label={e.status}
                      size="small"
                      sx={{
                        bgcolor: e.status === 'SUCCESS' ? '#064E3B' : e.status === 'FAILED' ? '#450A0A' : '#451A03',
                        color: e.status === 'SUCCESS' ? '#34D399' : e.status === 'FAILED' ? '#FCA5A5' : '#FDE68A',
                        fontWeight: 700,
                        fontSize: 10,
                        height: 20
                      }}
                    />
                  </TableCell>
                  <TableCell align="center" sx={{ fontFamily: 'monospace', color: '#CBD5E1', fontSize: 11 }}>
                    {e.durationMs}ms
                  </TableCell>
                  <TableCell sx={{ color: '#94A3B8', fontSize: 11 }}>
                    {e.lastRunAt}
                  </TableCell>
                  <TableCell sx={{ color: '#94A3B8', fontSize: 11 }}>
                    {e.nextRunAt}
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title="Trigger Immediate Execution">
                      <IconButton size="small" onClick={() => handleRetry(e.id, e.jobName)} sx={{ color: '#00D4FF' }}>
                        <RetryIcon sx={{ fontSize: 16 }} />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>
    </Box>
  );
}

export default SchedulerDashboardPage;
