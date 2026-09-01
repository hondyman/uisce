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
} from '@mui/material';
import {
  Storage as MdmIcon,
  CheckCircle as ValidIcon,
  WarningAmber as ExceptionIcon,
  AutoFixHigh as AutoResolveIcon,
} from '@mui/icons-material';

interface GoldenRecordItem {
  goldenId: string;
  primaryBK: string;
  entityName: string;
  assetClass: string;
  identifiers: { isin: string; cusip: string; sedol: string; figi: string };
  marketPrice: number;
  priceSource: string;
  couponRate?: number;
  lastMastered: string;
  hasOpenException: boolean;
  varianceDeltaPct?: number;
}

export const EnterpriseMDMStudio: React.FC<{ tenantId?: string }> = ({ tenantId: _tenantId }) => {
  const [records, setRecords] = useState<GoldenRecordItem[]>([
    {
      goldenId: 'gld-sec-001',
      primaryBK: 'US0378331005',
      entityName: 'Apple Inc. Common Stock',
      assetClass: 'EQUITY',
      identifiers: { isin: 'US0378331005', cusip: '037833100', sedol: '2046251', figi: 'BBG000B9XRY4' },
      marketPrice: 226.40,
      priceSource: 'BLOOMBERG_BPIPE',
      lastMastered: '2 mins ago',
      hasOpenException: false
    },
    {
      goldenId: 'gld-sec-002',
      primaryBK: 'US912810TL44',
      entityName: 'US Treasury N/B 4.25% 2034',
      assetClass: 'FIXED_INCOME',
      identifiers: { isin: 'US912810TL44', cusip: '912810TL4', sedol: 'BP8X912', figi: 'BBG01G6W7T44' },
      marketPrice: 98.42,
      priceSource: 'REFINITIV_DATASCOPE',
      couponRate: 4.25,
      lastMastered: 'Just now',
      hasOpenException: true,
      varianceDeltaPct: 6.15
    }
  ]);

  const [triageNotice, setTriageNotice] = useState<string | null>(null);

  const handleResolveException = (goldenId: string) => {
    setRecords(prev =>
      prev.map(r => (r.goldenId === goldenId ? { ...r, hasOpenException: false, priceSource: 'BLOOMBERG_OVERRIDE' } : r))
    );
    setTriageNotice('Pricing tolerance exception resolved: Applied Bloomberg evaluated quote with audit passport.');
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
          <MdmIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Enterprise Master Data Management (MDM) Studio
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Multi-vendor survivorship, cross-reference identifier resolution & exception triage (Security / Pricing Master)
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            icon={<ValidIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />}
            label="Survivorship Waterfall: Live"
            size="small"
            sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
          />
        </Stack>
      </Box>

      {triageNotice && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          {triageNotice}
        </Alert>
      )}

      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Mastered Securities</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#38BDF8', fontFamily: 'monospace' }}>14,250</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>100% Symbology Bound</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Active Contributors</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#34D399' }}>Bloomberg, LSEG, ICE</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Real-time Ingress Stream</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Open Price Exceptions</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#EF4444', fontFamily: 'monospace' }}>1 Active</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>&gt; 5% Inter-Vendor Delta</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Automated Match Rate</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#FBBF24' }}>99.82%</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Deterministic Graph Lookup</Typography>
          </Paper>
        </Grid>
      </Grid>

      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Master Golden Records & Cross-Vendor Identifiers
      </Typography>
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Entity / Canonical Name</TableCell>
              <TableCell>Cross-Ref Identifiers</TableCell>
              <TableCell align="right">Golden Price</TableCell>
              <TableCell>Winning Source</TableCell>
              <TableCell align="center">Status</TableCell>
              <TableCell align="center">Action</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {records.map(r => (
              <TableRow key={r.goldenId} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>
                    {r.entityName}
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: 10 }}>
                    {r.assetClass} | BK: {r.primaryBK}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#CBD5E1', display: 'block', fontSize: 10 }}>
                    ISIN: {r.identifiers.isin} | CUSIP: {r.identifiers.cusip}
                  </Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#64748B', fontSize: 10 }}>
                    FIGI: {r.identifiers.figi} | SEDOL: {r.identifiers.sedol}
                  </Typography>
                </TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#34D399', fontSize: 12 }}>
                  ${r.marketPrice.toFixed(2)}
                </TableCell>
                <TableCell>
                  <Chip label={r.priceSource} size="small" sx={{ bgcolor: '#1E293B', color: '#38BDF8', fontSize: 10 }} />
                </TableCell>
                <TableCell align="center">
                  {r.hasOpenException ? (
                    <Chip
                      icon={<ExceptionIcon sx={{ fontSize: 12, color: '#EF4444 !important' }} />}
                      label={`Delta: +${r.varianceDeltaPct}%`}
                      size="small"
                      sx={{ bgcolor: '#450A0A', color: '#FCA5A5', fontWeight: 700, fontSize: 10 }}
                    />
                  ) : (
                    <Chip
                      icon={<ValidIcon sx={{ fontSize: 12, color: '#10B981 !important' }} />}
                      label="Golden Clean"
                      size="small"
                      sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 10 }}
                    />
                  )}
                </TableCell>
                <TableCell align="center">
                  {r.hasOpenException ? (
                    <Button
                      variant="contained"
                      size="small"
                      startIcon={<AutoResolveIcon sx={{ fontSize: 12 }} />}
                      onClick={() => handleResolveException(r.goldenId)}
                      sx={{ bgcolor: '#0284C7', textTransform: 'none', fontSize: 10, py: 0.2, '&:hover': { bgcolor: '#0369A1' } }}
                    >
                      Resolve Break
                    </Button>
                  ) : (
                    <Typography variant="caption" sx={{ color: '#64748B' }}>Synchronized</Typography>
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

export default EnterpriseMDMStudio;
