import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Card, CardContent, Typography, Grid, Chip, Button, Stack,
  CircularProgress, Alert, MenuItem, Select, FormControl, InputLabel,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper
} from '@mui/material';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';
import TrendingFlatIcon from '@mui/icons-material/TrendingFlat';
import RefreshIcon from '@mui/icons-material/Refresh';
import StorageIcon from '@mui/icons-material/Storage';
import TimelineIcon from '@mui/icons-material/Timeline';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import { mdmScoringApi, type TrendAnalysisReport } from '../api';
import { QualityTrendChart } from './QualityTrendChart';

const DIMENSIONS = [
  { key: 'COMPOSITE', label: '6-Pillar Composite Quality' },
  { key: 'SUFFICIENCY', label: 'Legacy Sufficiency Rate' },
  { key: 'COVERAGE', label: 'Universe Attribute Coverage' },
  { key: 'DELIVERY_TIMELINESS', label: 'SLA Delivery Timeliness' },
  { key: 'STABILITY', label: 'Restatement Stability' },
  { key: 'FRICTION', label: 'Operational Friction Score' },
  { key: 'LICENSING', label: 'Commercial Rights Score' },
];

const PRESETS = [
  { label: '30 Days (Hot StarRocks)', days: 30 },
  { label: '90 Days (Hot + Warm)', days: 90 },
  { label: '1 Year (Hot + Warm)', days: 365 },
  { label: '3 Years (All 3 Tiers)', days: 1095 },
];

const VENDOR_COLORS: Record<string, string> = {
  BBG: '#ff8c00', // Bloomberg orange
  RFT: '#0088cc', // Refinitiv blue
  FDS: '#00b050', // FactSet green
  ICE: '#7030a0', // ICE purple
  SPG: '#c00000', // S&P red
};

export interface HistoricalTrendsViewerProps {
  entityDomain?: string;
}

export const HistoricalTrendsViewer: React.FC<HistoricalTrendsViewerProps> = ({ entityDomain }) => {
  const [dimension, setDimension] = useState<string>('COMPOSITE');
  const [selectedPreset, setSelectedPreset] = useState<number>(90);
  const [report, setReport] = useState<TrendAnalysisReport | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchTrends = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const toDate = new Date();
      const fromDate = new Date();
      fromDate.setDate(fromDate.getDate() - selectedPreset);

      const dateToStr = toDate.toISOString().slice(0, 10);
      const dateFromStr = fromDate.toISOString().slice(0, 10);

      const res = await mdmScoringApi.trends(
        dimension,
        ['BBG', 'RFT', 'FDS', 'ICE', 'SPG'],
        dateFromStr,
        dateToStr,
        entityDomain
      );
      setReport(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to load historical trend analytics');
    } finally {
      setLoading(false);
    }
  }, [dimension, selectedPreset, entityDomain]);

  useEffect(() => {
    fetchTrends();
  }, [fetchTrends]);

  const renderTrendDirectionChip = (dir: string, slope: number) => {
    const slopeStr = slope !== undefined ? ` (${slope >= 0 ? '+' : ''}${slope.toFixed(2)} pts/30d)` : '';
    if (dir === 'IMPROVING') {
      return (
        <Chip
          size="small"
          color="success"
          icon={<TrendingUpIcon fontSize="small" />}
          label={`IMPROVING${slopeStr}`}
          sx={{ fontWeight: 700 }}
        />
      );
    }
    if (dir === 'DECLINING') {
      return (
        <Chip
          size="small"
          color="error"
          icon={<TrendingDownIcon fontSize="small" />}
          label={`DECLINING${slopeStr}`}
          sx={{ fontWeight: 700 }}
        />
      );
    }
    if (dir === 'INSUFFICIENT_DATA') {
      return (
        <Chip
          size="small"
          variant="outlined"
          icon={<HelpOutlineIcon fontSize="small" />}
          label="INSUFFICIENT DATA"
          sx={{ fontWeight: 600, color: 'text.secondary' }}
        />
      );
    }
    return (
      <Chip
        size="small"
        color="info"
        variant="outlined"
        icon={<TrendingFlatIcon fontSize="small" />}
        label={`STABLE${slopeStr}`}
        sx={{ fontWeight: 700 }}
      />
    );
  };

  const renderTierTag = (tier: string) => {
    switch (tier) {
      case 'HOT':
        return <Chip size="small" label="HOT (StarRocks)" color="warning" sx={{ fontWeight: 700 }} />;
      case 'WARM':
        return <Chip size="small" label="WARM (PostgreSQL)" color="primary" sx={{ fontWeight: 700 }} />;
      case 'COLD':
        return <Chip size="small" label="COLD (Iceberg)" color="secondary" sx={{ fontWeight: 700 }} />;
      default:
        return <Chip size="small" label={tier} variant="outlined" />;
    }
  };

  return (
    <Box sx={{ p: 2 }}>
      {/* Header and Controls */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3, flexWrap: 'wrap', gap: 2 }}>
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <TimelineIcon color="primary" sx={{ fontSize: 30 }} />
            <Typography variant="h5" sx={{ fontWeight: 700 }}>
              Three-Tier Historical Trend Analytics
            </Typography>
            {report?.tiers_hit && (
              <Stack direction="row" spacing={1}>
                {report.tiers_hit.map((t) => (
                  <React.Fragment key={t}>{renderTierTag(t)}</React.Fragment>
                ))}
                {report.empty_tiers?.map((t) => (
                  <Chip key={t} size="small" variant="outlined" color="default" label={`EMPTY: ${t}`} sx={{ fontStyle: 'italic' }} />
                ))}
              </Stack>
            )}
          </Box>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            Deterministic watermark routing querying StarRocks Hot (0-30d), PostgreSQL Warm (31-365d), and Lakekeeper Iceberg Cold (366+d) with interval-weighted density regression.
          </Typography>
        </Box>

        <Stack direction="row" spacing={1.5} alignItems="center">
          <FormControl size="small" sx={{ minWidth: 220 }}>
            <InputLabel>Scoring Dimension</InputLabel>
            <Select
              value={dimension}
              label="Scoring Dimension"
              onChange={(e) => setDimension(e.target.value)}
            >
              {DIMENSIONS.map((d) => (
                <MenuItem key={d.key} value={d.key}>
                  {d.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={fetchTrends}
            disabled={loading}
          >
            Refresh
          </Button>
        </Stack>
      </Box>

      {/* Date Range Presets */}
      <Stack direction="row" spacing={1} sx={{ mb: 3 }}>
        {PRESETS.map((p) => (
          <Button
            key={p.days}
            variant={selectedPreset === p.days ? 'contained' : 'outlined'}
            size="small"
            onClick={() => setSelectedPreset(p.days)}
            startIcon={<StorageIcon fontSize="small" />}
          >
            {p.label}
          </Button>
        ))}
      </Stack>

      {/* Loading & Error States */}
      {loading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 6 }}>
          <CircularProgress size={32} />
          <Typography variant="body2" sx={{ ml: 2 }}>Routing multi-tier trend query...</Typography>
        </Box>
      )}

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>{error}</Alert>
      )}

      {/* Boundary Conflict Warning Alert */}
      {report?.boundary_conflicts && report.boundary_conflicts.length > 0 && (
        <Alert
          severity="warning"
          icon={<WarningAmberIcon />}
          sx={{ mb: 3, borderRadius: 2 }}
        >
          <strong>Tier Synchronization Discrepancy:</strong> {report.boundary_conflict_count} boundary conflict(s) detected across storage tiers. High-precedence tier (HOT &gt; WARM &gt; COLD) retained.
        </Alert>
      )}

      {report && !loading && (
        <Box>
          {/* Vendor Summary Cards */}
          <Grid container spacing={2} sx={{ mb: 3 }}>
            {report.series.map((s) => (
              <Grid item xs={12} sm={6} md={2.4} key={s.vendor_id}>
                <Card
                  variant="outlined"
                  sx={{
                    borderRadius: 2,
                    borderTop: `4px solid ${VENDOR_COLORS[s.vendor_id] || '#888'}`,
                  }}
                >
                  <CardContent sx={{ p: 2 }}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
                        {s.vendor_id}
                      </Typography>
                      {renderTrendDirectionChip(s.trend_direction, s.slope_per_30d)}
                    </Box>
                    <Typography variant="caption" color="text.secondary" noWrap>
                      {s.vendor_name}
                    </Typography>

                    <Box sx={{ mt: 1.5 }}>
                      <Typography variant="h5" sx={{ fontWeight: 700, color: 'primary.main' }}>
                        {s.avg_score_weighted?.toFixed(1) || s.avg_score.toFixed(1)}%
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Weighted avg ({s.points.length} pts)
                      </Typography>
                      <Box sx={{ mt: 0.5 }}>
                        <Typography variant="caption" color="text.secondary" display="block">
                          Range: {s.min_score.toFixed(1)}% — {s.max_score.toFixed(1)}%
                        </Typography>
                        <Typography variant="caption" color="text.secondary" display="block">
                          Arithmetic avg: {s.avg_score.toFixed(1)}%
                        </Typography>
                      </Box>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>

          {/* Interactive Visual Trendline Chart with Watermarks & Zone Shading */}
          <QualityTrendChart
            entityDomain={entityDomain}
            initialDimension={dimension}
            initialPresetDays={selectedPreset}
          />

          {/* Interactive Multi-Vendor Time-Series Matrix */}
          <Card variant="outlined" sx={{ borderRadius: 2, mb: 3 }}>
            <CardContent sx={{ p: 2.5 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  Historical Trend Trajectory: {DIMENSIONS.find((d) => d.key === report.dimension)?.label}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Query Span: {new Date(report.date_from).toLocaleDateString()} to {new Date(report.date_to).toLocaleDateString()}
                </Typography>
              </Box>

              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'action.hover' }}>
                      <TableCell sx={{ fontWeight: 700 }}>Observation Date</TableCell>
                      <TableCell align="center" sx={{ fontWeight: 700 }}>Storage Tier</TableCell>
                      {report.series.map((s) => (
                        <TableCell key={s.vendor_id} align="right" sx={{ fontWeight: 700, color: VENDOR_COLORS[s.vendor_id] }}>
                          {s.vendor_id} ({s.vendor_name.split(' ')[0]})
                        </TableCell>
                      ))}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {/* Render up to recent 30 samples */}
                    {report.series[0]?.points.slice(-30).reverse().map((pt, idx) => {
                      const dStr = new Date(pt.as_of_date).toLocaleDateString();
                      return (
                        <TableRow key={idx} hover>
                          <TableCell><strong>{dStr}</strong></TableCell>
                          <TableCell align="center">
                            {renderTierTag(pt.storage_tier)}
                          </TableCell>
                          {report.series.map((s) => {
                            const match = s.points.find(
                              (p) => p.as_of_date.slice(0, 10) === pt.as_of_date.slice(0, 10)
                            );
                            return (
                              <TableCell key={s.vendor_id} align="right" sx={{ fontWeight: 600 }}>
                                {match ? `${match.score_value.toFixed(1)}%` : '—'}
                              </TableCell>
                            );
                          })}
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>
        </Box>
      )}
    </Box>
  );
};
