import React from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Alert
} from '@mui/material';
import {
  Insights as InsightIcon,
  Verified as GoldenIcon,
  TrendingDown as DownIcon,
  TrendingUp as UpIcon,
  WarningAmber as AnomalyIcon,
  PushPin as PinIcon,
  WaterfallChart as WaterfallIcon
} from '@mui/icons-material';

interface DriverContribution {
  dimensionName: string;
  segmentValue: string;
  impactDelta: number;
  percentageOfChange: number;
  direction: 'POSITIVE' | 'NEGATIVE' | 'NEUTRAL';
  narrative: string;
}

interface AnomalyFlag {
  metricName: string;
  zScore: number;
  observedValue: number;
  expectedValue: number;
  severity: 'WARNING' | 'CRITICAL' | 'INFO';
  message: string;
}

export interface DriverAnalysisData {
  metricKey: string;
  baselinePeriod: string;
  comparisonPeriod: string;
  totalDelta: number;
  topContributors: DriverContribution[];
  anomaliesDetected: AnomalyFlag[];
  certifiedGoldenMatch?: {
    assetId: string;
    title: string;
    certifiedBy: string;
    certifiedAt: string;
    trustScore: number;
  };
}

interface AutonomousDriverAnalysisPanelProps {
  data?: DriverAnalysisData;
  onPinInsight?: () => void;
  onRunWaterfall?: () => void;
}

export const AutonomousDriverAnalysisPanel: React.FC<AutonomousDriverAnalysisPanelProps> = ({
  data = {
    metricKey: 'Net Fund Return',
    baselinePeriod: 'July 2026',
    comparisonPeriod: 'August 2026',
    totalDelta: -2.4,
    topContributors: [
      {
        dimensionName: 'Sector',
        segmentValue: 'Tech Equities Allocation',
        impactDelta: -1.8,
        percentageOfChange: 75.0,
        direction: 'NEGATIVE',
        narrative: 'Driven by semiconductor pullbacks and profit-taking in large-cap holdings.'
      },
      {
        dimensionName: 'Currency',
        segmentValue: 'EUR / USD FX Drag',
        impactDelta: -0.4,
        percentageOfChange: 16.7,
        direction: 'NEGATIVE',
        narrative: 'Currency drag across EUR-denominated institutional share classes.'
      }
    ],
    anomaliesDetected: [
      {
        metricName: 'CRIMS.Trade_Order Volume',
        zScore: 4.2,
        observedValue: 48200000,
        expectedValue: 11400000,
        severity: 'CRITICAL',
        message: "Trading volume in 'CRIMS.Trade_Order' spiked 4.2σ above 30-day moving average."
      }
    ],
    certifiedGoldenMatch: {
      assetId: 'gold-nav-q2',
      title: 'Official Institutional Regulatory NAV & Return',
      certifiedBy: 'Chief Risk Officer (Enterprise MDM)',
      certifiedAt: '2026-08-15',
      trustScore: 1.0
    }
  },
  onPinInsight,
  onRunWaterfall
}) => {
  return (
    <Paper
      elevation={0}
      sx={{
        p: 2.5,
        mb: 2,
        bgcolor: '#071526',
        color: '#F8FAFC',
        border: '1px solid #1E293B',
        borderRadius: 2
      }}
    >
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={1.5} mb={2} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <InsightIcon sx={{ color: '#00D4FF', fontSize: 24 }} />
          <Box>
            <Typography variant="subtitle1" sx={{ fontWeight: 700, fontSize: 14 }}>
              SpotIQ Driver Decomposition & Variance Explanation
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Multi-variable driver attribution across {data.baselinePeriod} vs {data.comparisonPeriod}
            </Typography>
          </Box>
        </Stack>

        {data.certifiedGoldenMatch && (
          <Chip
            icon={<GoldenIcon sx={{ fontSize: 14, color: '#FBBF24 !important' }} />}
            label={`Certified Golden Answer (${(data.certifiedGoldenMatch.trustScore * 100).toFixed(0)}% Trust)`}
            size="small"
            sx={{
              bgcolor: '#451A03',
              color: '#FBBF24',
              fontWeight: 700,
              fontSize: 11,
              border: '1px solid rgba(251, 191, 36, 0.3)'
            }}
          />
        )}
      </Box>

      {/* Delta Banner */}
      <Box
        sx={{
          p: 1.5,
          mb: 2,
          bgcolor: data.totalDelta < 0 ? 'rgba(239, 68, 68, 0.12)' : 'rgba(16, 185, 129, 0.12)',
          border: data.totalDelta < 0 ? '1px solid rgba(239, 68, 68, 0.3)' : '1px solid rgba(16, 185, 129, 0.3)',
          borderRadius: 1.5,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}
      >
        <Stack direction="row" spacing={1} alignItems="center">
          {data.totalDelta < 0 ? <DownIcon sx={{ color: '#F87171' }} /> : <UpIcon sx={{ color: '#34D399' }} />}
          <Typography variant="body2" sx={{ fontWeight: 700, color: data.totalDelta < 0 ? '#F87171' : '#34D399' }}>
            {data.metricKey} {data.totalDelta < 0 ? 'dropped' : 'gained'} {data.totalDelta}% ({data.baselinePeriod} → {data.comparisonPeriod})
          </Typography>
        </Stack>
        <Typography variant="caption" sx={{ color: '#94A3B8' }}>
          Automated regression decomposition
        </Typography>
      </Box>

      {/* Contributors Table */}
      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Key Variance Contributors
      </Typography>
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5, mb: 2 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Dimension Segment</TableCell>
              <TableCell align="center">Impact</TableCell>
              <TableCell align="center">% of Change</TableCell>
              <TableCell>Driver Narrative</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {data.topContributors.map((c, idx) => (
              <TableRow key={idx} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>{c.segmentValue}</Typography>
                  <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: 10 }}>{c.dimensionName}</Typography>
                </TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', fontWeight: 700, color: c.impactDelta < 0 ? '#F87171' : '#34D399' }}>
                  {c.impactDelta > 0 ? `+${c.impactDelta}%` : `${c.impactDelta}%`}
                </TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', color: '#CBD5E1' }}>
                  {c.percentageOfChange}%
                </TableCell>
                <TableCell sx={{ fontSize: 11, color: '#CBD5E1' }}>{c.narrative}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Anomaly Alerts */}
      {data.anomaliesDetected.map((a, idx) => (
        <Alert
          key={idx}
          severity="warning"
          icon={<AnomalyIcon sx={{ color: '#FBBF24' }} />}
          sx={{
            mb: 2,
            bgcolor: '#451A03',
            color: '#FDBA74',
            border: '1px solid rgba(251, 191, 36, 0.3)',
            fontSize: 12
          }}
        >
          <strong>Anomaly Detected ({a.zScore}σ):</strong> {a.message}
        </Alert>
      ))}

      {/* Action Buttons */}
      <Stack direction="row" spacing={1.5} justifyContent="flex-end">
        <Button
          size="small"
          variant="outlined"
          startIcon={<WaterfallIcon sx={{ fontSize: 14 }} />}
          onClick={onRunWaterfall}
          sx={{
            color: '#38BDF8',
            borderColor: '#0284C7',
            textTransform: 'none',
            fontSize: 11,
            fontWeight: 600,
            '&:hover': { borderColor: '#38BDF8', bgcolor: 'rgba(56, 189, 248, 0.08)' }
          }}
        >
          Run Full Waterfall Attribution
        </Button>
        <Button
          size="small"
          variant="contained"
          startIcon={<PinIcon sx={{ fontSize: 14 }} />}
          onClick={onPinInsight}
          sx={{
            bgcolor: '#0284C7',
            color: '#FFFFFF',
            textTransform: 'none',
            fontSize: 11,
            fontWeight: 700,
            '&:hover': { bgcolor: '#0369A1' }
          }}
        >
          Pin Insight to Liveboard
        </Button>
      </Stack>
    </Paper>
  );
};

export default AutonomousDriverAnalysisPanel;
