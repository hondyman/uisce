import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  Grid,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import {
  SyncAlt as SyncIcon,
  AccountBalance as AborIcon,
  ShowChart as PborIcon,
  FlashOn as IborIcon,
  CheckCircle as ValidIcon,
  AutoFixHigh as AutoHealIcon
} from '@mui/icons-material';

interface TriBookBreakItem {
  breakId: string;
  securityName: string;
  isin: string;
  iborQuantity: number;
  aborQuantity: number;
  pborNavImpactUSD: number;
  variance: number;
  status: 'OPEN' | 'AUTO_RESOLVED' | 'ESCALATED';
}

export const MultiBookReconciliationHUD: React.FC<{ tenantId?: string }> = ({ tenantId: _tenantId }) => {
  const [breaks, setBreaks] = useState<TriBookBreakItem[]>([
    {
      breakId: 'brk-001',
      securityName: 'Microsoft Corporation',
      isin: 'US5949181045',
      iborQuantity: 25000,
      aborQuantity: 24500,
      pborNavImpactUSD: 204500.0,
      variance: 500,
      status: 'OPEN'
    },
    {
      breakId: 'brk-002',
      securityName: 'US Treasury Bill 3M',
      isin: 'US912797HC34',
      iborQuantity: 1000000,
      aborQuantity: 1000000,
      pborNavImpactUSD: 0.0,
      variance: 0,
      status: 'AUTO_RESOLVED'
    }
  ]);

  const [isHealing, setIsHealing] = useState(false);

  const handleAutoHeal = (breakId: string) => {
    setIsHealing(true);
    setTimeout(() => {
      setBreaks(prev =>
        prev.map(b => (b.breakId === breakId ? { ...b, status: 'AUTO_RESOLVED', variance: 0 } : b))
      );
      setIsHealing(false);
    }, 800);
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
          <SyncIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Unified Multi-Book Synchronization Engine (IBOR ↔ ABOR ↔ PBOR)
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Continuous tri-party shadow settlement & bitemporal ledger alignment
            </Typography>
          </Box>
        </Stack>
        <Chip
          icon={<ValidIcon sx={{ fontSize: 16, color: '#10B981 !important' }} />}
          label="Bitemporal Seam: Live (Tk ≥ Wt)"
          size="small"
          sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
        />
      </Box>

      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={1}>
              <IborIcon sx={{ color: '#38BDF8', fontSize: 20 }} />
              <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>
                Front-Office IBOR
              </Typography>
            </Stack>
            <Typography variant="h6" sx={{ fontWeight: 700, color: '#F8FAFC' }}>
              $142,850,210.50
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Projected Intraday Net Value
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={1}>
              <AborIcon sx={{ color: '#34D399', fontSize: 20 }} />
              <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#34D399', fontSize: 12 }}>
                Back-Office ABOR
              </Typography>
            </Stack>
            <Typography variant="h6" sx={{ fontWeight: 700, color: '#F8FAFC' }}>
              $142,645,710.50
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Settled GL Balance (Variance: -$204.5k)
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={1}>
              <PborIcon sx={{ color: '#F59E0B', fontSize: 20 }} />
              <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#F59E0B', fontSize: 12 }}>
                Performance PBOR
              </Typography>
            </Stack>
            <Typography variant="h6" sx={{ fontWeight: 700, color: '#F8FAFC' }}>
              +14.28% TWR
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Daily Compounded True Economic Return
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Continuous Position Break Ledger (SWIFT MT535 Matcher)
      </Typography>
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Security / ISIN</TableCell>
              <TableCell align="right">IBOR Quantity</TableCell>
              <TableCell align="right">Custodian ABOR</TableCell>
              <TableCell align="right">Variance</TableCell>
              <TableCell align="right">NAV Impact</TableCell>
              <TableCell align="center">Status</TableCell>
              <TableCell align="center">Action</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {breaks.map(b => (
              <TableRow key={b.breakId} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 600, fontSize: 12 }}>
                    {b.securityName}
                  </Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#38BDF8', fontSize: 10 }}>
                    {b.isin}
                  </Typography>
                </TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  {b.iborQuantity.toLocaleString()}
                </TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  {b.aborQuantity.toLocaleString()}
                </TableCell>
                <TableCell
                  align="right"
                  sx={{
                    fontFamily: 'monospace',
                    fontWeight: 700,
                    color: b.variance > 0 ? '#EF4444' : '#34D399',
                    fontSize: 12
                  }}
                >
                  {b.variance > 0 ? `+${b.variance.toLocaleString()}` : '0'}
                </TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', color: '#FBBF24', fontSize: 12 }}>
                  ${b.pborNavImpactUSD.toLocaleString()}
                </TableCell>
                <TableCell align="center">
                  <Chip
                    label={b.status}
                    size="small"
                    sx={{
                      bgcolor: b.status === 'AUTO_RESOLVED' ? '#064E3B' : '#7C2D12',
                      color: b.status === 'AUTO_RESOLVED' ? '#34D399' : '#FCA5A5',
                      fontWeight: 700,
                      fontSize: 10
                    }}
                  />
                </TableCell>
                <TableCell align="center">
                  {b.status === 'OPEN' ? (
                    <Button
                      variant="outlined"
                      size="small"
                      startIcon={<AutoHealIcon sx={{ fontSize: 12 }} />}
                      onClick={() => handleAutoHeal(b.breakId)}
                      disabled={isHealing}
                      sx={{
                        borderColor: '#0284C7',
                        color: '#38BDF8',
                        textTransform: 'none',
                        fontSize: 10,
                        py: 0.2,
                        '&:hover': { borderColor: '#38BDF8', bgcolor: 'rgba(56, 189, 248, 0.08)' }
                      }}
                    >
                      Post Synthetic GL Entry
                    </Button>
                  ) : (
                    <Typography variant="caption" sx={{ color: '#64748B' }}>
                      Aligned
                    </Typography>
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

export default MultiBookReconciliationHUD;
