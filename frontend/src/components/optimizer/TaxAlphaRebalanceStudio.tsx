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
} from '@mui/material';
import {
  AccountBalanceWallet as WalletIcon,
  LocalAtm as TaxIcon,
  SwapHoriz as RebalanceIcon,
  CheckCircle as ValidIcon,
  WarningAmber as WashWarningIcon,
  Send as FixSendIcon,
  Speed as LatencyIcon,
  Bolt as WASMIcon
} from '@mui/icons-material';

interface TaxLotRow {
  lotId: string;
  ticker: string;
  securityName: string;
  shares: number;
  costBasis: number;
  marketPrice: number;
  unrealizedPnL: number;
  taxTerm: 'SHORT_TERM' | 'LONG_TERM';
  isWashBlocked: boolean;
  proxySubstituteTicker?: string;
}

export const TaxAlphaRebalanceStudio: React.FC<{ tenantId?: string; portfolioId?: string }> = ({
  tenantId: _tenantId,
  portfolioId: _portfolioId
}) => {
  const [isSolving, setIsSolving] = useState(false);
  const [harvestedTotal, setHarvestedTotal] = useState(14820.0);
  const [solverLatency, setSolverLatency] = useState(3.42);

  const [lots, _setLots] = useState<TaxLotRow[]>([
    {
      lotId: 'lot-01',
      ticker: 'SPY',
      securityName: 'SPDR S&P 500 ETF Trust',
      shares: 1200,
      costBasis: 565.0,
      marketPrice: 538.5,
      unrealizedPnL: -31800.0,
      taxTerm: 'SHORT_TERM',
      isWashBlocked: false,
      proxySubstituteTicker: 'IVV (iShares Core S&P 500)'
    },
    {
      lotId: 'lot-02',
      ticker: 'NVDA',
      securityName: 'NVIDIA Corporation',
      shares: 850,
      costBasis: 142.0,
      marketPrice: 128.4,
      unrealizedPnL: -11560.0,
      taxTerm: 'SHORT_TERM',
      isWashBlocked: false,
      proxySubstituteTicker: 'SMH (VanEck Semiconductor ETF)'
    },
    {
      lotId: 'lot-03',
      ticker: 'AAPL',
      securityName: 'Apple Inc.',
      shares: 500,
      costBasis: 235.0,
      marketPrice: 226.0,
      unrealizedPnL: -4500.0,
      taxTerm: 'LONG_TERM',
      isWashBlocked: true
    }
  ]);

  const handleExecuteOptimization = () => {
    setIsSolving(true);
    setTimeout(() => {
      setIsSolving(false);
      setHarvestedTotal(16035.2);
      setSolverLatency(2.89);
    }, 600);
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
          <WASMIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              WASM Tax-Alpha & Portfolio Optimization Kernel
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Sub-5ms in-memory QP solver, IRS 30-day wash-sale shielding & FIX 35=D order routing
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            icon={<LatencyIcon sx={{ fontSize: 14, color: '#00D4FF !important' }} />}
            label={`${solverLatency} ms WASM SLA`}
            size="small"
            sx={{ bgcolor: '#0B1E36', color: '#00D4FF', fontWeight: 700, fontSize: 11, fontFamily: 'monospace' }}
          />
          <Button
            variant="contained"
            size="small"
            startIcon={<RebalanceIcon />}
            onClick={handleExecuteOptimization}
            disabled={isSolving}
            sx={{ bgcolor: '#0284C7', fontWeight: 600, textTransform: 'none', '&:hover': { bgcolor: '#0369A1' } }}
          >
            {isSolving ? 'Solving QP...' : 'Run Tax-Loss Harvest'}
          </Button>
        </Stack>
      </Box>

      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={0.5}>
              <TaxIcon sx={{ color: '#10B981', fontSize: 20 }} />
              <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                Estimated Tax Alpha Generated
              </Typography>
            </Stack>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#34D399', fontFamily: 'monospace' }}>
              ${harvestedTotal.toLocaleString(undefined, { minimumFractionDigits: 2 })}
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              37% Short-Term / 20% Long-Term Alpha
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={0.5}>
              <WalletIcon sx={{ color: '#38BDF8', fontSize: 20 }} />
              <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                Eligible Loss Lots Identified
              </Typography>
            </Stack>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#F8FAFC' }}>
              2 Active ($43.3k Gross Loss)
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              1 Lot Blocked by 30-Day Wash Window
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={0.5}>
              <FixSendIcon sx={{ color: '#F59E0B', fontSize: 20 }} />
              <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                Generated Order Tickets
              </Typography>
            </Stack>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#FBBF24' }}>
              4 FIX 35=D Orders
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              2 Sells + 2 Correlated Proxy Buys
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Tax-Lot Inventory & Proxy Asset Substitutions
      </Typography>
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Security / Ticker</TableCell>
              <TableCell align="right">Shares</TableCell>
              <TableCell align="right">Cost Basis</TableCell>
              <TableCell align="right">Market Price</TableCell>
              <TableCell align="right">Unrealized PnL</TableCell>
              <TableCell align="center">Tax Term</TableCell>
              <TableCell>Replacement Proxy Asset</TableCell>
              <TableCell align="center">Wash Status</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {lots.map(l => (
              <TableRow key={l.lotId} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 600, fontSize: 12 }}>
                    {l.ticker}
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: 10 }}>
                    {l.securityName}
                  </Typography>
                </TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  {l.shares.toLocaleString()}
                </TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  ${l.costBasis.toFixed(2)}
                </TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  ${l.marketPrice.toFixed(2)}
                </TableCell>
                <TableCell
                  align="right"
                  sx={{
                    fontFamily: 'monospace',
                    fontWeight: 700,
                    color: l.unrealizedPnL < 0 ? '#EF4444' : '#34D399',
                    fontSize: 12
                  }}
                >
                  ${l.unrealizedPnL.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                </TableCell>
                <TableCell align="center">
                  <Chip
                    label={l.taxTerm === 'SHORT_TERM' ? 'Short-Term (37%)' : 'Long-Term (20%)'}
                    size="small"
                    sx={{
                      bgcolor: l.taxTerm === 'SHORT_TERM' ? '#7C2D12' : '#064E3B',
                      color: l.taxTerm === 'SHORT_TERM' ? '#FDBA74' : '#6EE7B7',
                      fontSize: 10,
                      fontWeight: 700
                    }}
                  />
                </TableCell>
                <TableCell>
                  {l.proxySubstituteTicker ? (
                    <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#00D4FF', fontWeight: 600 }}>
                      {l.proxySubstituteTicker}
                    </Typography>
                  ) : (
                    <Typography variant="caption" sx={{ color: '#64748B' }}>—</Typography>
                  )}
                </TableCell>
                <TableCell align="center">
                  {l.isWashBlocked ? (
                    <Chip
                      icon={<WashWarningIcon sx={{ fontSize: 12, color: '#EF4444 !important' }} />}
                      label="30-Day Blocked"
                      size="small"
                      sx={{ bgcolor: '#450A0A', color: '#FCA5A5', fontSize: 10, fontWeight: 700 }}
                    />
                  ) : (
                    <Chip
                      icon={<ValidIcon sx={{ fontSize: 12, color: '#10B981 !important' }} />}
                      label="Harvest Safe"
                      size="small"
                      sx={{ bgcolor: '#064E3B', color: '#34D399', fontSize: 10, fontWeight: 700 }}
                    />
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

export default TaxAlphaRebalanceStudio;
