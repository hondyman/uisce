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
  CircularProgress,
} from '@mui/material';
import {
  CallSplit as SplitIcon,
  CheckCircle as ValidIcon,
  PlayArrow as ExecuteIcon,
} from '@mui/icons-material';

interface CorporateActionSummary {
  eventId: string;
  eventKey: string;
  eventType: 'FORWARD_SPLIT' | 'SPIN_OFF' | 'REVERSE_SPLIT' | 'MERGER';
  targetSecurity: string;
  ticker: string;
  ratio: string;
  exDate: string;
  recordDate: string;
  payableDate: string;
  fractionalTreatment: string;
  status: 'PENDING_EX_DATE' | 'SHADOW_PROJECTED' | 'EXECUTED';
  totalEligibleShares: number;
  projectedCILUSD: number;
}

export const CorporateActionsStudio: React.FC<{ tenantId?: string }> = ({ tenantId: _tenantId }) => {
  const [isExecuting, setIsExecuting] = useState(false);
  const [selectedEvent, setSelectedEvent] = useState<CorporateActionSummary>({
    eventId: 'ca-0911-split',
    eventKey: 'ca.event.nvda_forward_split_10to1',
    eventType: 'FORWARD_SPLIT',
    targetSecurity: 'NVIDIA Corporation',
    ticker: 'NVDA',
    ratio: '10 : 1',
    exDate: '2026-06-10',
    recordDate: '2026-06-08',
    payableDate: '2026-06-11',
    fractionalTreatment: 'CASH_IN_LIEU',
    status: 'PENDING_EX_DATE',
    totalEligibleShares: 45000,
    projectedCILUSD: 1420.50
  });

  const [executionAlert, setExecutionAlert] = useState<string | null>(null);

  const handleExecuteEvent = () => {
    setIsExecuting(true);
    setTimeout(() => {
      setSelectedEvent((prev) => ({ ...prev, status: 'EXECUTED' }));
      setIsExecuting(false);
      setExecutionAlert('Ex-Date execution complete: 450,000 post-split shares active, $1,420.50 CIL credited.');
    }, 850);
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
          <SplitIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Autonomous Corporate Actions Lifecycle Studio
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Multi-vendor survivorship, shadow position projections & lot-level cost basis re-allocation
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            icon={<ValidIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />}
            label="DTCC / Bloomberg Matched"
            size="small"
            sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
          />
          <Button
            variant="contained"
            size="small"
            startIcon={isExecuting ? <CircularProgress size={14} color="inherit" /> : <ExecuteIcon />}
            onClick={handleExecuteEvent}
            disabled={selectedEvent.status === 'EXECUTED' || isExecuting}
            sx={{
              bgcolor: selectedEvent.status === 'EXECUTED' ? '#064E3B' : '#0284C7',
              color: '#F8FAFC',
              fontWeight: 600,
              textTransform: 'none',
              '&:hover': { bgcolor: selectedEvent.status === 'EXECUTED' ? '#064E3B' : '#0369A1' }
            }}
          >
            {selectedEvent.status === 'EXECUTED' ? 'Executed & Merkle-Sealed' : 'Execute Ex-Date Processing'}
          </Button>
        </Stack>
      </Box>

      {executionAlert && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          {executionAlert}
        </Alert>
      )}

      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Event Type & Multiplier
            </Typography>
            <Typography variant="h6" sx={{ fontWeight: 700, color: '#38BDF8', fontFamily: 'monospace' }}>
              {selectedEvent.eventType} ({selectedEvent.ratio})
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              {selectedEvent.targetSecurity} ({selectedEvent.ticker})
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Ex-Date / Record-Date
            </Typography>
            <Typography variant="h6" sx={{ fontWeight: 700, color: '#FBBF24', fontFamily: 'monospace' }}>
              {selectedEvent.exDate}
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Record: {selectedEvent.recordDate} | Pay: {selectedEvent.payableDate}
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Entitled Shares Projected
            </Typography>
            <Typography variant="h6" sx={{ fontWeight: 700, color: '#34D399', fontFamily: 'monospace' }}>
              {selectedEvent.status === 'EXECUTED' ? '450,000' : '45,000 → 450,000'}
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Cost Basis: $1,284.00 → $128.40/sh
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Cash-in-Lieu (CIL) Credit
            </Typography>
            <Typography variant="h6" sx={{ fontWeight: 700, color: '#F8FAFC', fontFamily: 'monospace' }}>
              ${selectedEvent.projectedCILUSD.toLocaleString(undefined, { minimumFractionDigits: 2 })}
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Auto-Credited to Projected IBOR Cash
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Tax-Lot Cost Basis Re-Allocation Ledger (Pre vs. Post Ex-Date)
      </Typography>
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Account / Portfolio</TableCell>
              <TableCell>Original Acquisition Date</TableCell>
              <TableCell align="right">Pre-Event Shares</TableCell>
              <TableCell align="right">Pre Cost Basis</TableCell>
              <TableCell align="right">Post-Event Shares</TableCell>
              <TableCell align="right">Post Cost Basis</TableCell>
              <TableCell align="center">Tax Status</TableCell>
              <TableCell align="center">Action Status</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            <TableRow sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
              <TableCell sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>Global Growth Fund (Acct #104)</TableCell>
              <TableCell sx={{ fontFamily: 'monospace', fontSize: 11 }}>2024-03-15</TableCell>
              <TableCell align="right" sx={{ fontFamily: 'monospace', fontSize: 12 }}>25,000</TableCell>
              <TableCell align="right" sx={{ fontFamily: 'monospace', fontSize: 12 }}>$1,200.00</TableCell>
              <TableCell align="right" sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#34D399', fontSize: 12 }}>250,000</TableCell>
              <TableCell align="right" sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#34D399', fontSize: 12 }}>$120.00</TableCell>
              <TableCell align="center">
                <Chip label="Long-Term Preserved" size="small" sx={{ bgcolor: '#064E3B', color: '#6EE7B7', fontSize: 10, fontWeight: 700 }} />
              </TableCell>
              <TableCell align="center">
                <Chip label={selectedEvent.status} size="small" sx={{ bgcolor: selectedEvent.status === 'EXECUTED' ? '#064E3B' : '#7C2D12', color: selectedEvent.status === 'EXECUTED' ? '#34D399' : '#FDBA74', fontSize: 10, fontWeight: 700 }} />
              </TableCell>
            </TableRow>
            <TableRow sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
              <TableCell sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>Institutional Alpha (Acct #209)</TableCell>
              <TableCell sx={{ fontFamily: 'monospace', fontSize: 11 }}>2026-01-20</TableCell>
              <TableCell align="right" sx={{ fontFamily: 'monospace', fontSize: 12 }}>20,000</TableCell>
              <TableCell align="right" sx={{ fontFamily: 'monospace', fontSize: 12 }}>$1,350.00</TableCell>
              <TableCell align="right" sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#34D399', fontSize: 12 }}>200,000</TableCell>
              <TableCell align="right" sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#34D399', fontSize: 12 }}>$135.00</TableCell>
              <TableCell align="center">
                <Chip label="Short-Term Preserved" size="small" sx={{ bgcolor: '#7C2D12', color: '#FDBA74', fontSize: 10, fontWeight: 700 }} />
              </TableCell>
              <TableCell align="center">
                <Chip label={selectedEvent.status} size="small" sx={{ bgcolor: selectedEvent.status === 'EXECUTED' ? '#064E3B' : '#7C2D12', color: selectedEvent.status === 'EXECUTED' ? '#34D399' : '#FDBA74', fontSize: 10, fontWeight: 700 }} />
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default CorporateActionsStudio;
