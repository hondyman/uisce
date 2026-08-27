import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Grid,
  Alert
} from '@mui/material';
import {
  History as HistoryIcon,
  Shield as ShieldIcon,
  CheckCircle as ValidIcon,
  WarningAmber as WarningIcon,
  Tune as TuneIcon,
  Download as ExportIcon
} from '@mui/icons-material';

interface BacktestSummary {
  ruleName: string;
  dynamicDenominator: string;
  asOfPeriod: string;
  daysEvaluated: number;
  breachesCount: number;
  exceptionsCount: number;
  merkleRootHash: string;
}

export const BitemporalRuleReplayStudio: React.FC<{ tenantId?: string }> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999'
}) => {
  const [summary] = useState<BacktestSummary>({
    ruleName: 'At least 80% of Debt > BBB+',
    dynamicDenominator: 'ST1 Debt Market Value',
    asOfPeriod: '2024-01-01 to 2026-08-01',
    daysEvaluated: 640,
    breachesCount: 14,
    exceptionsCount: 3,
    merkleRootHash: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
  });

  const [notice, setNotice] = useState<string | null>(null);

  const handleExportMerkle = () => {
    setNotice(`Exported SEC Rule 17a-4 Merkle Proof [${summary.merkleRootHash.substring(0, 16)}...]`);
  };

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <HistoryIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Bitemporal Rule Replay & Regulatory Backtester
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Iceberg as-of time travel compliance simulation with automated dynamic denominators
            </Typography>
          </Box>
        </Stack>
        <Chip icon={<ValidIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />} label="Iceberg As-Of Replay: Ready" size="small" sx={{ bgcolor: '#0B1E36', color: '#34D399', fontWeight: 700, fontSize: 11, border: '1px solid #1E293B' }} />
      </Box>

      {notice && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          {notice}
        </Alert>
      )}

      {/* Configuration & Rule Details */}
      <Paper sx={{ p: 2, mb: 3, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', display: 'block', mb: 1 }}>
          Rule Specification & Dynamic Binding
        </Typography>
        <Typography variant="body1" sx={{ fontWeight: 700, color: '#38BDF8', mb: 0.5 }}>
          {summary.ruleName}
        </Typography>
        <Typography variant="body2" sx={{ color: '#CBD5E1', fontSize: 12 }}>
          Dynamic Denominator Bound: <strong>{summary.dynamicDenominator}</strong> (via graph traversal)
        </Typography>
      </Paper>

      {/* KPI Cards */}
      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Portfolio Days Evaluated</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#38BDF8', fontFamily: 'monospace' }}>{summary.daysEvaluated}</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>{summary.asOfPeriod}</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Active Breaches Identified</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#F87171', fontFamily: 'monospace' }}>{summary.breachesCount}</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Prior to Remediation</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Data Exceptions Flagged</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#FBBF24', fontFamily: 'monospace' }}>{summary.exceptionsCount}</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Unrated Foreign Debt</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>SEC 17a-4 Merkle Seal</Typography>
            <Typography variant="body2" sx={{ fontWeight: 700, color: '#34D399', fontFamily: 'monospace', fontSize: 11, wordBreak: 'break-all' }}>
              {summary.merkleRootHash.substring(0, 16)}...
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Immutable WORM Proof</Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Action Buttons */}
      <Stack direction="row" spacing={2} justifyContent="flex-end">
        <Button
          variant="outlined"
          startIcon={<TuneIcon sx={{ fontSize: 16 }} />}
          sx={{ color: '#38BDF8', borderColor: '#0284C7', textTransform: 'none', fontWeight: 600, fontSize: 12 }}
        >
          Auto-Tune Dynamic Denominator
        </Button>
        <Button
          variant="contained"
          startIcon={<ExportIcon sx={{ fontSize: 16 }} />}
          onClick={handleExportMerkle}
          sx={{ bgcolor: '#0284C7', color: '#FFF', textTransform: 'none', fontWeight: 700, fontSize: 12, '&:hover': { bgcolor: '#0369A1' } }}
        >
          Export SEC Rule 17a-4 Merkle Proof
        </Button>
      </Stack>
    </Paper>
  );
};

export default BitemporalRuleReplayStudio;
