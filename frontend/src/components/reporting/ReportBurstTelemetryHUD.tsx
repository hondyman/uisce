import React, { useState, useEffect } from 'react';
import { 
  Activity, RefreshCw, Clock, Zap, ShieldAlert
} from 'lucide-react';
import {
  Box,
  Typography,
  Grid,
  Button,
  Paper,
  Chip,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  LinearProgress,
  CircularProgress,
} from '@mui/material';

interface TelemetryData {
  batchId: string;
  status: string;
  totalClients: number;
  successfulCount: number;
  failedCount: number;
  throughputPerSec: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
  failedSlices: Array<{
    artifactId: string;
    clientId: string;
    errorReason: string;
    retryCount: number;
  }>;
}

export const ReportBurstTelemetryHUD: React.FC<{ batchId: string; tenantId?: string }> = ({
  batchId,
  tenantId = '',
}) => {
  const [telemetry, setTelemetry] = useState<TelemetryData | null>(null);
  const [isRetrying, setIsRetrying] = useState(false);

  const fetchTelemetry = async () => {
    try {
      const res = await fetch(`/api/reports/batches/${batchId}/telemetry`, {
        headers: tenantId ? { 'X-Tenant-ID': tenantId } : {},
      });
      if (res.ok) {
        const data = await res.json();
        setTelemetry(data);
      }
    } catch (e) {
      console.error('Failed fetching telemetry:', e);
    }
  };

  useEffect(() => {
    fetchTelemetry();
    const interval = setInterval(fetchTelemetry, 3000);
    return () => clearInterval(interval);
  }, [batchId]);

  const handleRetryDLQ = async () => {
    setIsRetrying(true);
    try {
      const res = await fetch(`/api/reports/batches/${batchId}/retry-dlq`, {
        method: 'POST',
        headers: tenantId ? { 'X-Tenant-ID': tenantId } : {},
      });
      if (res.ok) {
        fetchTelemetry();
      }
    } finally {
      setIsRetrying(false);
    }
  };

  if (!telemetry) {
    return (
      <Box sx={{ p: 3, display: 'flex', alignItems: 'center', gap: 1.5, color: '#94A3B8' }}>
        <CircularProgress size={16} sx={{ color: '#06B6D4' }} />
        <Typography variant="caption">Loading live burst telemetry stream...</Typography>
      </Box>
    );
  }

  const progressPct = telemetry.totalClients > 0
    ? Math.round(((telemetry.successfulCount + telemetry.failedCount) / telemetry.totalClients) * 100)
    : 0;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5, p: 2.5, bgcolor: '#0B192C', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
      {/* Header & Live Status */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.08)', pb: 1.5 }}>
        <Box>
          <Typography variant="subtitle2" fontWeight="700" sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#F8FAFC', textTransform: 'uppercase', letterSpacing: 0.5 }}>
            <Activity size={16} color="#38BDF8" /> Live Burst Execution HUD
          </Typography>
          <Typography variant="caption" sx={{ color: '#64748B', fontFamily: 'monospace' }}>
            Batch ID: {batchId}
          </Typography>
        </Box>
        <Chip
          size="small"
          label={telemetry.status}
          sx={{
            fontWeight: 800,
            fontSize: '0.68rem',
            bgcolor: telemetry.status === 'COMPLETED' ? 'rgba(16, 185, 129, 0.15)' : telemetry.status === 'PARTIAL' ? 'rgba(245, 158, 11, 0.15)' : 'rgba(56, 189, 248, 0.15)',
            color: telemetry.status === 'COMPLETED' ? '#34D399' : telemetry.status === 'PARTIAL' ? '#FBBF24' : '#38BDF8',
            border: `1px solid ${telemetry.status === 'COMPLETED' ? 'rgba(16, 185, 129, 0.3)' : telemetry.status === 'PARTIAL' ? 'rgba(245, 158, 11, 0.3)' : 'rgba(56, 189, 248, 0.3)'}`,
          }}
        />
      </Box>

      {/* Progress Bar */}
      <Box>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
          <Typography variant="caption" sx={{ color: '#CBD5E1', fontWeight: 600 }}>Burst Execution Progress</Typography>
          <Typography variant="caption" sx={{ color: '#38BDF8', fontWeight: 700 }}>
            {progressPct}% ({telemetry.successfulCount + telemetry.failedCount} / {telemetry.totalClients} Slices)
          </Typography>
        </Box>
        <LinearProgress
          variant="determinate"
          value={progressPct}
          sx={{
            height: 8,
            borderRadius: 4,
            bgcolor: 'rgba(255,255,255,0.05)',
            '& .MuiLinearProgress-bar': {
              background: 'linear-gradient(90deg, #06B6D4 0%, #10B981 100%)',
              borderRadius: 4,
            },
          }}
        />
      </Box>

      {/* Telemetry Metrics Grid */}
      <Grid container spacing={2}>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper sx={{ p: 1.5, bgcolor: 'rgba(15, 23, 42, 0.7)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'flex', alignItems: 'center', gap: 0.5, fontWeight: 700, textTransform: 'uppercase', fontSize: '0.65rem' }}>
              <Zap size={13} color="#FBBF24" /> Throughput
            </Typography>
            <Typography variant="h6" fontWeight="800" sx={{ color: '#F8FAFC', mt: 0.5 }}>
              {telemetry.throughputPerSec.toFixed(1)} <Typography component="span" variant="caption" sx={{ color: '#64748B' }}>docs/s</Typography>
            </Typography>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper sx={{ p: 1.5, bgcolor: 'rgba(15, 23, 42, 0.7)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'flex', alignItems: 'center', gap: 0.5, fontWeight: 700, textTransform: 'uppercase', fontSize: '0.65rem' }}>
              <Clock size={13} color="#38BDF8" /> p50 Latency
            </Typography>
            <Typography variant="h6" fontWeight="800" sx={{ color: '#F8FAFC', mt: 0.5 }}>
              {telemetry.p50LatencyMs} <Typography component="span" variant="caption" sx={{ color: '#64748B' }}>ms</Typography>
            </Typography>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper sx={{ p: 1.5, bgcolor: 'rgba(15, 23, 42, 0.7)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'flex', alignItems: 'center', gap: 0.5, fontWeight: 700, textTransform: 'uppercase', fontSize: '0.65rem' }}>
              <Clock size={13} color="#C084FC" /> p95 Latency
            </Typography>
            <Typography variant="h6" fontWeight="800" sx={{ color: '#F8FAFC', mt: 0.5 }}>
              {telemetry.p95LatencyMs} <Typography component="span" variant="caption" sx={{ color: '#64748B' }}>ms</Typography>
            </Typography>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper sx={{ p: 1.5, bgcolor: 'rgba(15, 23, 42, 0.7)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'flex', alignItems: 'center', gap: 0.5, fontWeight: 700, textTransform: 'uppercase', fontSize: '0.65rem' }}>
              <Clock size={13} color="#F87171" /> p99 Latency
            </Typography>
            <Typography variant="h6" fontWeight="800" sx={{ color: '#F8FAFC', mt: 0.5 }}>
              {telemetry.p99LatencyMs} <Typography component="span" variant="caption" sx={{ color: '#64748B' }}>ms</Typography>
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Dead-Letter Queue (DLQ) Card */}
      {telemetry.failedSlices && telemetry.failedSlices.length > 0 && (
        <Paper sx={{ p: 2, bgcolor: 'rgba(239, 68, 68, 0.08)', border: '1px solid rgba(239, 68, 68, 0.25)', borderRadius: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
            <Typography variant="subtitle2" fontWeight="700" sx={{ color: '#FCA5A5', display: 'flex', alignItems: 'center', gap: 1, textTransform: 'uppercase', fontSize: '0.72rem' }}>
              <ShieldAlert size={15} color="#EF4444" /> Dead-Letter Queue ({telemetry.failedSlices.length} Failed Slices)
            </Typography>
            <Button
              variant="contained"
              size="small"
              onClick={handleRetryDLQ}
              disabled={isRetrying}
              startIcon={isRetrying ? <CircularProgress size={12} color="inherit" /> : <RefreshCw size={13} />}
              sx={{ bgcolor: '#EF4444', color: '#FFF', textTransform: 'none', fontSize: '0.7rem', fontWeight: 800, '&:hover': { bgcolor: '#DC2626' } }}
            >
              {isRetrying ? 'Retrying...' : '1-Click Retry Failed Slices'}
            </Button>
          </Box>

          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell sx={{ color: '#FCA5A5', fontSize: '0.68rem', fontWeight: 700 }}>Client ID</TableCell>
                <TableCell sx={{ color: '#FCA5A5', fontSize: '0.68rem', fontWeight: 700 }}>Error Reason</TableCell>
                <TableCell sx={{ color: '#FCA5A5', fontSize: '0.68rem', fontWeight: 700 }}>Retries</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {telemetry.failedSlices.map((slice) => (
                <TableRow key={slice.artifactId}>
                  <TableCell sx={{ color: '#FECACA', fontSize: '0.68rem', fontFamily: 'monospace' }}>{slice.clientId}</TableCell>
                  <TableCell sx={{ color: '#E2E8F0', fontSize: '0.68rem' }}>{slice.errorReason || 'Render timeout / schema mismatch'}</TableCell>
                  <TableCell sx={{ color: '#94A3B8', fontSize: '0.68rem' }}>{slice.retryCount}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Paper>
      )}
    </Box>
  );
};

export default ReportBurstTelemetryHUD;
