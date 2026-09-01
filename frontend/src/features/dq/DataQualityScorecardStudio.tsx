import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow
} from '@mui/material';
import {
  Assessment as ScorecardIcon,
  CheckCircle as ValidIcon,
  WarningAmber as WarningIcon,
  AutoAwesome as AiIcon
} from '@mui/icons-material';

interface DomainHealthItem {
  domainKey: string;
  assetClass: string;
  vendorSource: string;
  completeness: number;
  accuracy: number;
  timeliness: number;
  consistency: number;
  compositeScore: number;
  status: 'HEALTHY' | 'WARNING' | 'CRITICAL';
}

export const DataQualityScorecardStudio: React.FC<{ tenantId?: string }> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999'
}) => {
  const [healthData] = useState<DomainHealthItem[]>([
    {
      domainKey: 'PRICING',
      assetClass: 'FIXED_INCOME',
      vendorSource: 'BLOOMBERG',
      completeness: 99.8,
      accuracy: 94.2,
      timeliness: 99.9,
      consistency: 100.0,
      compositeScore: 98.4,
      status: 'HEALTHY'
    },
    {
      domainKey: 'SECURITY',
      assetClass: 'EQUITY',
      vendorSource: 'REFINITIV',
      completeness: 88.5,
      accuracy: 99.1,
      timeliness: 91.0,
      consistency: 95.0,
      compositeScore: 93.4,
      status: 'WARNING'
    }
  ]);

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <ScorecardIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Data Quality & Health Scoring Scorecards (DQ-HSC)
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Continuous real-time scoring across Completeness, Accuracy, Timeliness, and Consistency
            </Typography>
          </Box>
        </Stack>
        <Chip
          icon={<AiIcon sx={{ fontSize: 14, color: '#00D4FF !important' }} />}
          label="Active DQ Streaming Mesh"
          size="small"
          sx={{ bgcolor: '#0B1E36', color: '#00D4FF', fontWeight: 700, fontSize: 11, border: '1px solid #1E293B' }}
        />
      </Box>

      {/* KPI Cards */}
      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Enterprise Health Index</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#34D399', fontFamily: 'monospace' }}>95.9%</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Across All MDM Domains</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Completeness Average</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#38BDF8', fontFamily: 'monospace' }}>98.1%</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Mandatory Attribute Checks</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Accuracy Violations</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#FBBF24', fontFamily: 'monospace' }}>1 Active Break</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>&gt; 5% Vendor Delta</Typography>
          </Paper>
        </Grid>
        <Grid   size={{ xs: 12, sm: 3 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>Scoring Latency</Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#34D399', fontFamily: 'monospace' }}>&lt; 12ms</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Continuous In-Memory Eval</Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Heatmap Table */}
      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Domain Health & Dimension Scorecards
      </Typography>
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Domain / Asset Class</TableCell>
              <TableCell>Vendor Source</TableCell>
              <TableCell align="center">Completeness</TableCell>
              <TableCell align="center">Accuracy</TableCell>
              <TableCell align="center">Timeliness</TableCell>
              <TableCell align="center">Consistency</TableCell>
              <TableCell align="center">Composite Health</TableCell>
              <TableCell align="center">Status</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {healthData.map((h, idx) => (
              <TableRow key={idx} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>{h.domainKey}</Typography>
                  <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: 10 }}>{h.assetClass}</Typography>
                </TableCell>
                <TableCell><Chip label={h.vendorSource} size="small" sx={{ bgcolor: '#1E293B', color: '#CBD5E1', fontSize: 10 }} /></TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', color: '#34D399' }}>{h.completeness}%</TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', color: h.accuracy < 95 ? '#FBBF24' : '#34D399' }}>{h.accuracy}%</TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', color: '#34D399' }}>{h.timeliness}%</TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', color: '#34D399' }}>{h.consistency}%</TableCell>
                <TableCell align="center">
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontFamily: 'monospace' }}>
                    {h.compositeScore}%
                  </Typography>
                </TableCell>
                <TableCell align="center">
                  <Chip
                    icon={h.status === 'HEALTHY' ? <ValidIcon sx={{ fontSize: 12, color: '#10B981 !important' }} /> : <WarningIcon sx={{ fontSize: 12, color: '#F59E0B !important' }} />}
                    label={h.status}
                    size="small"
                    sx={{
                      bgcolor: h.status === 'HEALTHY' ? '#064E3B' : '#451A03',
                      color: h.status === 'HEALTHY' ? '#34D399' : '#FDBA74',
                      fontWeight: 700,
                      fontSize: 10
                    }}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default DataQualityScorecardStudio;
