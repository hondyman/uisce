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
  CallSplit as ActionIcon,
  CheckCircle as ValidIcon,
  Shield as SealIcon,
  Publish as PropagateIcon
} from '@mui/icons-material';

interface CorporateActionItem {
  actionId: string;
  securityName: string;
  isin: string;
  actionType: string;
  effectiveDate: string;
  announcementSource: string;
  termsDescription: string;
  status: 'PROPAGATED' | 'PENDING_PROPAGATION';
  merkleSeal: string;
}

export const CorporateActionStudio: React.FC<{ tenantId?: string }> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999'
}) => {
  const [actions] = useState<CorporateActionItem[]>([
    {
      actionId: 'ca-0924-01',
      securityName: 'NVIDIA Corporation (NVDA)',
      isin: 'US67066G1040',
      actionType: 'SPLIT',
      effectiveDate: '2026-09-01',
      announcementSource: 'DTCC',
      termsDescription: '10-for-1 Stock Split (Multiplier: 10.0)',
      status: 'PROPAGATED',
      merkleSeal: '4f92b1a87c0e...'
    }
  ]);

  const [notice, setNotice] = useState<string | null>(null);

  const handleSimulatePropagation = (_actionId: string) => {
    setNotice('Corporate Action successfully propagated across IBOR positions with atomic cost-basis adjustments.');
  };

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <ActionIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Corporate Action & Event Propagation Engine (CA-PE)
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Atomic graph-driven position adjustments & Merkle-sealed audit ledger
            </Typography>
          </Box>
        </Stack>
        <Chip icon={<SealIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />} label="SEC 17a-4 Sealed" size="small" sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }} />
      </Box>

      {notice && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          {notice}
        </Alert>
      )}

      {/* Corporate Action Table */}
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Security / ISIN</TableCell>
              <TableCell>Action Type</TableCell>
              <TableCell>Effective Date</TableCell>
              <TableCell>Source</TableCell>
              <TableCell>Terms & Multiplier</TableCell>
              <TableCell align="center">Status</TableCell>
              <TableCell align="center">Action</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {actions.map(a => (
              <TableRow key={a.actionId} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>{a.securityName}</Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#94A3B8', fontSize: 10 }}>{a.isin}</Typography>
                </TableCell>
                <TableCell><Chip label={a.actionType} size="small" sx={{ bgcolor: '#1E293B', color: '#38BDF8', fontSize: 10, fontWeight: 700 }} /></TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 11 }}>{a.effectiveDate}</TableCell>
                <TableCell>{a.announcementSource}</TableCell>
                <TableCell sx={{ fontSize: 11, color: '#CBD5E1' }}>{a.termsDescription}</TableCell>
                <TableCell align="center">
                  <Chip icon={<ValidIcon sx={{ fontSize: 12, color: '#10B981 !important' }} />} label={a.status} size="small" sx={{ bgcolor: '#064E3B', color: '#34D399', fontSize: 10, fontWeight: 700 }} />
                </TableCell>
                <TableCell align="center">
                  <Button
                    variant="contained"
                    size="small"
                    startIcon={<PropagateIcon sx={{ fontSize: 12 }} />}
                    onClick={() => handleSimulatePropagation(a.actionId)}
                    sx={{ bgcolor: '#0284C7', textTransform: 'none', fontSize: 10, py: 0.2, '&:hover': { bgcolor: '#0369A1' } }}
                  >
                    Re-Propagate
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default CorporateActionStudio;
