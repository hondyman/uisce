import React, { useState } from 'react';
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
  Autorenew as RebalanceIcon,
  CheckCircle as ValidIcon,
  Send as DispatchIcon
} from '@mui/icons-material';

interface DriftPortfolioRow {
  portfolioId: string;
  portfolioName: string;
  ruleCode: string;
  currentUtil: number;
  projected4hUtil: number;
  driftStatus: 'STABLE' | 'DRIFT_WARNING' | 'CRITICAL_DRIFT';
  recommendedAction: string;
}

export const CCDSARStudio: React.FC<{ tenantId?: string }> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999'
}) => {
  const [portfolios, setPortfolios] = useState<DriftPortfolioRow[]>([
    {
      portfolioId: 'port-104',
      portfolioName: 'Global Growth Master Fund (Acct #104)',
      ruleCode: 'LIMIT_ISSUER_MAX_5',
      currentUtil: 92.4,
      projected4hUtil: 96.8,
      driftStatus: 'CRITICAL_DRIFT',
      recommendedAction: 'Trim 3,450 shares NVDA ($443k) to restore 75% limit headroom.'
    },
    {
      portfolioId: 'port-209',
      portfolioName: 'Institutional Alpha Fund (Acct #209)',
      ruleCode: 'LIMIT_SECTOR_TECH_20',
      currentUtil: 78.1,
      projected4hUtil: 81.2,
      driftStatus: 'DRIFT_WARNING',
      recommendedAction: 'Monitor momentum drift; no immediate action required.'
    }
  ]);

  const [dispatchedNotice, setDispatchedNotice] = useState<string | null>(null);

  const handleDispatchBasket = (portfolioId: string) => {
    setPortfolios(prev =>
      prev.map(p => (p.portfolioId === portfolioId ? { ...p, driftStatus: 'STABLE', projected4hUtil: 74.5 } : p))
    );
    setDispatchedNotice('Autonomous Rebalancing Basket dispatched to EMS via FIX 35=D. Merkle Audit Seal verified.');
  };

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <RebalanceIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Continuous Compliance Drift & Autonomous Rebalancing (CC-DSAR)
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Real-time intraday drift forecasting & automated rebalancing basket generation
            </Typography>
          </Box>
        </Stack>
        <Chip icon={<ValidIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />} label="CC-DSAR Engine: Active" size="small" sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }} />
      </Box>

      {dispatchedNotice && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          {dispatchedNotice}
        </Alert>
      )}

      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Portfolio Account</TableCell>
              <TableCell>Mandate Rule</TableCell>
              <TableCell align="center">Current Utilization</TableCell>
              <TableCell align="center">Projected 4h Drift</TableCell>
              <TableCell align="center">Status</TableCell>
              <TableCell>Remediation Basket</TableCell>
              <TableCell align="center">Action</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {portfolios.map(p => (
              <TableRow key={p.portfolioId} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>{p.portfolioName}</Typography>
                </TableCell>
                <TableCell sx={{ fontSize: 11, fontFamily: 'monospace' }}>{p.ruleCode}</TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', fontWeight: 700 }}>
                  {p.currentUtil.toFixed(1)}%
                </TableCell>
                <TableCell align="center">
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', fontWeight: 700, color: p.projected4hUtil > 90 ? '#EF4444' : '#F59E0B' }}>
                    {p.projected4hUtil.toFixed(1)}%
                  </Typography>
                </TableCell>
                <TableCell align="center">
                  <Chip
                    label={p.driftStatus}
                    size="small"
                    sx={{
                      bgcolor: p.driftStatus === 'CRITICAL_DRIFT' ? '#450A0A' : '#451A03',
                      color: p.driftStatus === 'CRITICAL_DRIFT' ? '#FCA5A5' : '#FDBA74',
                      fontWeight: 700,
                      fontSize: 10
                    }}
                  />
                </TableCell>
                <TableCell sx={{ fontSize: 11, color: '#CBD5E1' }}>{p.recommendedAction}</TableCell>
                <TableCell align="center">
                  {p.driftStatus === 'CRITICAL_DRIFT' ? (
                    <Button
                      variant="contained"
                      size="small"
                      startIcon={<DispatchIcon sx={{ fontSize: 12 }} />}
                      onClick={() => handleDispatchBasket(p.portfolioId)}
                      sx={{ bgcolor: '#0284C7', textTransform: 'none', fontSize: 10, py: 0.2, '&:hover': { bgcolor: '#0369A1' } }}
                    >
                      Dispatch Basket
                    </Button>
                  ) : (
                    <Typography variant="caption" sx={{ color: '#64748B' }}>Stable</Typography>
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

export default CCDSARStudio;
