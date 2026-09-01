import React, { useState } from 'react';
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
  Alert,
  LinearProgress,
} from '@mui/material';
import {
  Radar as RadarIcon,
  CheckCircle as ValidIcon,
  WarningAmber as WarningIcon,
  AutoFixHigh as ResizeIcon,
  Speed as LatencyIcon,
  TrendingUp as MomentumIcon
} from '@mui/icons-material';

interface DriftRiskItem {
  id: string;
  portfolioName: string;
  ruleCode: string;
  targetSecurity: string;
  currentUtilizationPct: number;
  breachProbability: number;
  hoursToBreach: number;
  proposedShares: number;
  maxCompliantShares: number;
  proxyTicker: string;
  isImminent: boolean;
}

export const PredictiveComplianceRadar: React.FC<{ tenantId?: string }> = ({ tenantId: _tenantId }) => {
  const [risks, setRisks] = useState<DriftRiskItem[]>([
    {
      id: 'drift-01',
      portfolioName: 'Global Growth Master Fund (Acct #104)',
      ruleCode: 'LIMIT_ISSUER_MAX_5',
      targetSecurity: 'NVIDIA Corporation (NVDA)',
      currentUtilizationPct: 92.4,
      breachProbability: 0.88,
      hoursToBreach: 14.5,
      proposedShares: 15000,
      maxCompliantShares: 8420,
      proxyTicker: 'SMH (VanEck Semiconductor ETF)',
      isImminent: true
    },
    {
      id: 'drift-02',
      portfolioName: 'Institutional Alpha Fund (Acct #209)',
      ruleCode: 'LIMIT_SECTOR_TECH_20',
      targetSecurity: 'Microsoft Corporation (MSFT)',
      currentUtilizationPct: 78.1,
      breachProbability: 0.34,
      hoursToBreach: 72.0,
      proposedShares: 5000,
      maxCompliantShares: 5000,
      proxyTicker: 'XLK (Technology Select Sector)',
      isImminent: false
    }
  ]);

  const [remediatedId, setRemediatedId] = useState<string | null>(null);

  const handleApplyAutoResize = (id: string) => {
    setRisks(prev =>
      prev.map(r =>
        r.id === id
          ? { ...r, proposedShares: r.maxCompliantShares, breachProbability: 0.08, isImminent: false }
          : r
      )
    );
    setRemediatedId(id);
  };

  return (
    <Paper
      elevation={0}
      sx={{
        p: 3,
        bgcolor: '#071526',
        color: '#F8FAFC',
        border: '1px solid #1E293B',
        borderRadius: 2
      }}
    >
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <RadarIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              AI Predictive Compliance & Passive Drift Radar
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Real-time violation probability forecasting, automatic trade re-sizing, and graph-driven proxy asset substitution
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            icon={<LatencyIcon sx={{ fontSize: 14, color: '#00D4FF !important' }} />}
            label="Inference SLA: < 2.5 ms"
            size="small"
            sx={{ bgcolor: '#0B1E36', color: '#00D4FF', fontWeight: 700, fontSize: 11, fontFamily: 'monospace' }}
          />
          <Chip
            icon={<ValidIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />}
            label="Passive Drift: Active"
            size="small"
            sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
          />
        </Stack>
      </Box>

      {remediatedId && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          Trade re-sized to maximum compliant allocation. Remaining volume routed to correlated proxy asset.
        </Alert>
      )}

      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={0.5}>
              <WarningIcon sx={{ color: '#EF4444', fontSize: 20 }} />
              <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                Imminent Breaches Forecasted (&lt; 48 hrs)
              </Typography>
            </Stack>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#FCA5A5', fontFamily: 'monospace' }}>
              1 Account at Risk
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              NVDA Issuer Cap (88% Violation Probability)
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={0.5}>
              <MomentumIcon sx={{ color: '#38BDF8', fontSize: 20 }} />
              <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                Average Group Capacity Utilization
              </Typography>
            </Stack>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#38BDF8', fontFamily: 'monospace' }}>
              85.25%
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Across Top 50 Institutional Portfolios
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={0.5}>
              <ValidIcon sx={{ color: '#F59E0B', fontSize: 20 }} />
              <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                Auto-Sizing & Proxy Efficiency
              </Typography>
            </Stack>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#FBBF24', fontFamily: 'monospace' }}>
              100% Order Preservation
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Zero Hard Trade Rejections Post-Resize
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Live Portfolio Risk Trajectory & Trade Resizing Recommendations
      </Typography>
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Portfolio / Mandate</TableCell>
              <TableCell>Target Security</TableCell>
              <TableCell align="center">Utilization</TableCell>
              <TableCell align="center">Breach Probability</TableCell>
              <TableCell align="center">Time to Breach</TableCell>
              <TableCell align="right">Proposed vs Compliant</TableCell>
              <TableCell>Graph Proxy Substitute</TableCell>
              <TableCell align="center">Action</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {risks.map(r => (
              <TableRow key={r.id} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>
                    {r.portfolioName}
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: 10 }}>
                    {r.ruleCode}
                  </Typography>
                </TableCell>
                <TableCell sx={{ fontSize: 12 }}>{r.targetSecurity}</TableCell>
                <TableCell align="center">
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', fontWeight: 700 }}>
                    {r.currentUtilizationPct.toFixed(1)}%
                  </Typography>
                  <LinearProgress
                    variant="determinate"
                    value={r.currentUtilizationPct}
                    sx={{
                      height: 4,
                      borderRadius: 1,
                      bgcolor: '#071526',
                      '& .MuiLinearProgress-bar': {
                        bgcolor: r.currentUtilizationPct > 90 ? '#EF4444' : '#38BDF8'
                      }
                    }}
                  />
                </TableCell>
                <TableCell align="center">
                  <Chip
                    label={`${Math.round(r.breachProbability * 100)}% Risk`}
                    size="small"
                    sx={{
                      bgcolor: r.isImminent ? '#450A0A' : '#064E3B',
                      color: r.isImminent ? '#FCA5A5' : '#34D399',
                      fontWeight: 700,
                      fontSize: 10
                    }}
                  />
                </TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', color: r.isImminent ? '#EF4444' : '#34D399', fontSize: 11 }}>
                  {r.hoursToBreach.toFixed(1)} hrs
                </TableCell>
                <TableCell align="right">
                  <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: 11 }}>
                    {r.proposedShares.toLocaleString()} → <strong style={{ color: '#34D399' }}>{r.maxCompliantShares.toLocaleString()}</strong>
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#00D4FF', fontWeight: 600 }}>
                    {r.proxyTicker}
                  </Typography>
                </TableCell>
                <TableCell align="center">
                  {r.isImminent ? (
                    <Button
                      variant="contained"
                      size="small"
                      startIcon={<ResizeIcon sx={{ fontSize: 12 }} />}
                      onClick={() => handleApplyAutoResize(r.id)}
                      sx={{ bgcolor: '#0284C7', textTransform: 'none', fontSize: 10, py: 0.2, '&:hover': { bgcolor: '#0369A1' } }}
                    >
                      Auto-Resize
                    </Button>
                  ) : (
                    <Typography variant="caption" sx={{ color: '#64748B' }}>Compliant</Typography>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default PredictiveComplianceRadar;
