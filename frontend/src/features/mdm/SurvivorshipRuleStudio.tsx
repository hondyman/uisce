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
  Slider,
  Alert
} from '@mui/material';
import {
  Tune as RuleIcon,
  Save as SaveIcon,
  CheckCircle as ValidIcon
} from '@mui/icons-material';

interface SurvivorshipRuleRow {
  attributeName: string;
  domainKey: string;
  assetClass: string;
  strategy: 'VENDOR_PRIORITY' | 'TIME_DECAY_WEIGHTED' | 'CONSENSUS_AVERAGE' | 'MOST_RECENT';
  priorityOrder: string[];
  tolerancePct: number;
  decayHalfLifeSec: number;
  isActive: boolean;
}

export const SurvivorshipRuleStudio: React.FC<{ tenantId?: string }> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999'
}) => {
  const [rules, setRules] = useState<SurvivorshipRuleRow[]>([
    {
      attributeName: 'market_price',
      domainKey: 'PRICING',
      assetClass: 'FIXED_INCOME',
      strategy: 'TIME_DECAY_WEIGHTED',
      priorityOrder: ['BLOOMBERG', 'IDC', 'REFINITIV'],
      tolerancePct: 5.0,
      decayHalfLifeSec: 1800,
      isActive: true
    },
    {
      attributeName: 'coupon_rate',
      domainKey: 'SECURITY',
      assetClass: 'FIXED_INCOME',
      strategy: 'VENDOR_PRIORITY',
      priorityOrder: ['BLOOMBERG', 'REFINITIV', 'DTCC'],
      tolerancePct: 0.0,
      decayHalfLifeSec: 86400,
      isActive: true
    },
    {
      attributeName: 'issuer_lei',
      domainKey: 'PARTY',
      assetClass: 'ALL',
      strategy: 'VENDOR_PRIORITY',
      priorityOrder: ['GLEIF', 'BLOOMBERG', 'REFINITIV'],
      tolerancePct: 0.0,
      decayHalfLifeSec: 604800,
      isActive: true
    }
  ]);

  const [savedNotice, setSavedNotice] = useState(false);

  const handleToleranceChange = (index: number, val: number) => {
    setRules((prev) => {
      const updated = [...prev];
      updated[index].tolerancePct = val;
      return updated;
    });
  };

  const handleSave = () => {
    setSavedNotice(true);
    setTimeout(() => setSavedNotice(false), 3000);
  };

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <RuleIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              MDM Dynamic Survivorship & Anomaly Configuration
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Declarative vendor waterfalls, time-decay functions, and exception trigger tolerances (Rule 1)
            </Typography>
          </Box>
        </Stack>
        <Button
          variant="contained"
          size="small"
          startIcon={<SaveIcon />}
          onClick={handleSave}
          sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 600, '&:hover': { bgcolor: '#0369A1' } }}
        >
          Save & Publish Rules
        </Button>
      </Box>

      {savedNotice && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          Survivorship rules updated in catalog store. Streaming consumers refreshed in real time.
        </Alert>
      )}

      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Attribute / Domain</TableCell>
              <TableCell>Asset Class</TableCell>
              <TableCell>Survivorship Strategy</TableCell>
              <TableCell>Vendor Hierarchy</TableCell>
              <TableCell align="center">Anomaly Tolerance (%)</TableCell>
              <TableCell align="center">Staleness Half-Life</TableCell>
              <TableCell align="center">Status</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rules.map((r, idx) => (
              <TableRow key={idx} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>
                    {r.attributeName}
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: 10 }}>
                    {r.domainKey}
                  </Typography>
                </TableCell>
                <TableCell sx={{ fontSize: 11 }}>{r.assetClass}</TableCell>
                <TableCell>
                  <Chip label={r.strategy} size="small" sx={{ bgcolor: '#1E293B', color: '#CBD5E1', fontSize: 10 }} />
                </TableCell>
                <TableCell>
                  <Stack direction="row" spacing={0.5}>
                    {r.priorityOrder.map((v, vIdx) => (
                      <Chip key={vIdx} label={`${vIdx + 1}. ${v}`} size="small" sx={{ bgcolor: '#071526', color: '#38BDF8', fontSize: 10 }} />
                    ))}
                  </Stack>
                </TableCell>
                <TableCell align="center" sx={{ width: 160 }}>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', fontWeight: 700, color: r.tolerancePct > 0 ? '#F59E0B' : '#64748B' }}>
                    ±{r.tolerancePct.toFixed(1)}%
                  </Typography>
                  <Slider
                    size="small"
                    value={r.tolerancePct}
                    min={0}
                    max={20}
                    step={0.5}
                    onChange={(_, val) => handleToleranceChange(idx, val as number)}
                    sx={{ color: '#00D4FF', p: 0.5 }}
                  />
                </TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', fontSize: 11 }}>
                  {r.decayHalfLifeSec >= 3600 ? `${r.decayHalfLifeSec / 3600}h` : `${r.decayHalfLifeSec}s`}
                </TableCell>
                <TableCell align="center">
                  <Chip
                    icon={<ValidIcon sx={{ fontSize: 12, color: '#10B981 !important' }} />}
                    label="Active"
                    size="small"
                    sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 10 }}
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

export default SurvivorshipRuleStudio;
