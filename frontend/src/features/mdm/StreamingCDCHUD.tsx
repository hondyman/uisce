import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  Grid,
  Divider,
  LinearProgress,
  Card,
  CardContent,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow
} from '@mui/material';
import {
  Sensors as StreamingIcon,
  Speed as LatencyIcon,
  CheckCircle as ValidIcon,
  Hub as FanoutIcon,
  TrendingUp as RateIcon
} from '@mui/icons-material';

interface VendorStreamMetrics {
  vendor: string;
  ticksPerSec: number;
  avgLatencyMs: number;
  lastTickTime: string;
  status: 'ACTIVE' | 'IDLE';
}

interface DownstreamFanoutTarget {
  targetSystem: string;
  channel: string;
  throughputPerSec: number;
  syncState: 'SYNCHRONIZED' | 'IN_FLIGHT';
}

export const StreamingCDCHUD: React.FC<{ tenantId?: string }> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999'
}) => {
  const [vendorStreams, setVendorStreams] = useState<VendorStreamMetrics[]>([
    { vendor: 'BLOOMBERG', ticksPerSec: 1420, avgLatencyMs: 1.8, lastTickTime: '2ms ago', status: 'ACTIVE' },
    { vendor: 'REFINITIV', ticksPerSec: 890, avgLatencyMs: 2.3, lastTickTime: '15ms ago', status: 'ACTIVE' },
    { vendor: 'IDC', ticksPerSec: 410, avgLatencyMs: 3.1, lastTickTime: '80ms ago', status: 'ACTIVE' },
    { vendor: 'DTCC', ticksPerSec: 120, avgLatencyMs: 4.0, lastTickTime: '200ms ago', status: 'ACTIVE' }
  ]);

  const [downstreamTargets] = useState<DownstreamFanoutTarget[]>([
    { targetSystem: 'IBOR Shadow Settlement', channel: 'In-Memory Seam', throughputPerSec: 2840, syncState: 'SYNCHRONIZED' },
    { targetSystem: 'ABOR General Ledger', channel: 'Temporal Saga', throughputPerSec: 1420, syncState: 'SYNCHRONIZED' },
    { targetSystem: 'Real-Time Risk Limits', channel: 'FastRecord VM', throughputPerSec: 2840, syncState: 'SYNCHRONIZED' },
    { targetSystem: 'Client Reporting Mesh', channel: 'Apache Iceberg', throughputPerSec: 850, syncState: 'SYNCHRONIZED' }
  ]);

  useEffect(() => {
    const interval = setInterval(() => {
      setVendorStreams(prev =>
        prev.map(v => ({
          ...v,
          ticksPerSec: Math.floor(v.ticksPerSec + (Math.random() * 80 - 40))
        }))
      );
    }, 1500);
    return () => clearInterval(interval);
  }, []);

  const totalTicksPerSec = vendorStreams.reduce((acc, v) => acc + v.ticksPerSec, 0);

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      {/* HUD Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <StreamingIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Live Streaming CDC Ingress & Temporal Fan-Out HUD
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Redpanda streaming feeds, in-memory universal mastering, and real-time downstream propagation
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            icon={<RateIcon sx={{ fontSize: 14, color: '#00D4FF !important' }} />}
            label={`${totalTicksPerSec.toLocaleString()} ticks/sec`}
            size="small"
            sx={{ bgcolor: '#0B1E36', color: '#00D4FF', fontWeight: 700, fontSize: 11, fontFamily: 'monospace' }}
          />
          <Chip
            icon={<LatencyIcon sx={{ fontSize: 14, color: '#34D399 !important' }} />}
            label="E2E Latency: < 2.5ms"
            size="small"
            sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11, fontFamily: 'monospace' }}
          />
        </Stack>
      </Box>

      {/* KPI Stream Matrix Cards */}
      <Grid container spacing={2} mb={3}>
        {vendorStreams.map(s => (
          <Grid   key={s.vendor} size={{ xs: 12, sm: 3 }}>
            <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center" mb={1}>
                <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700 }}>
                  {s.vendor}
                </Typography>
                <Chip label={s.status} size="small" sx={{ bgcolor: '#064E3B', color: '#34D399', fontSize: 9, height: 16 }} />
              </Stack>
              <Typography variant="h5" sx={{ fontWeight: 700, color: '#F8FAFC', fontFamily: 'monospace' }}>
                {s.ticksPerSec.toLocaleString()} <span style={{ fontSize: 11, color: '#64748B' }}>tps</span>
              </Typography>
              <Stack direction="row" justifyContent="space-between" alignItems="center" mt={0.5}>
                <Typography variant="caption" sx={{ color: '#00D4FF', fontFamily: 'monospace', fontSize: 10 }}>
                  {s.avgLatencyMs.toFixed(1)} ms latency
                </Typography>
                <Typography variant="caption" sx={{ color: '#64748B', fontSize: 10 }}>
                  {s.lastTickTime}
                </Typography>
              </Stack>
            </Paper>
          </Grid>
        ))}
      </Grid>

      {/* Downstream Temporal Fan-out Progress */}
      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', mb: 1.5, display: 'block' }}>
        Concurrently Synchronized Downstream Engines
      </Typography>

      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Downstream Target</TableCell>
              <TableCell>Delivery Channel</TableCell>
              <TableCell align="right">Throughput</TableCell>
              <TableCell align="center">Temporal Sync State</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {downstreamTargets.map(t => (
              <TableRow key={t.targetSystem} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Stack direction="row" spacing={1.5} alignItems="center">
                    <FanoutIcon sx={{ color: '#00D4FF', fontSize: 16 }} />
                    <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>
                      {t.targetSystem}
                    </Typography>
                  </Stack>
                </TableCell>
                <TableCell sx={{ fontSize: 11, color: '#CBD5E1' }}>{t.channel}</TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', color: '#34D399', fontWeight: 700 }}>
                  {t.throughputPerSec.toLocaleString()} events/sec
                </TableCell>
                <TableCell align="center">
                  <Chip
                    icon={<ValidIcon sx={{ fontSize: 12, color: '#10B981 !important' }} />}
                    label={t.syncState}
                    size="small"
                    sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 10 }}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default StreamingCDCHUD;
