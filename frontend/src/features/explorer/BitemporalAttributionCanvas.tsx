import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Slider,
  Chip,
  Grid,
  Divider,
  Button,
  Tooltip,
} from '@mui/material';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import HistoryEduIcon from '@mui/icons-material/HistoryEdu';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import ShieldIcon from '@mui/icons-material/Shield';

export interface AttributionSlice {
  factorName: string;
  allocationEffectPct: number;
  selectionEffectPct: number;
  totalImpactBps: number;
}

export const BitemporalAttributionCanvas: React.FC<{ tenantId: string }> = () => {
  const [effectiveDateIdx, setEffectiveDateIdx] = useState<number>(3);
  const [knowledgeDateIdx, setKnowledgeDateIdx] = useState<number>(4);

  const dates = ['2026-05-01', '2026-06-01', '2026-07-01', '2026-08-01', '2026-08-25'];

  const [attributionFactors] = useState<AttributionSlice[]>([
    { factorName: 'Tech Equities (Overweight)', allocationEffectPct: -1.2, selectionEffectPct: -0.6, totalImpactBps: -180 },
    { factorName: 'Fixed Income (Duration Hedge)', allocationEffectPct: 0.8, selectionEffectPct: 0.4, totalImpactBps: 120 },
    { factorName: 'FX Currency (EUR Drag)', allocationEffectPct: -0.3, selectionEffectPct: -0.1, totalImpactBps: -40 },
  ]);

  return (
    <Paper
      elevation={0}
      sx={{
        p: 3,
        bgcolor: '#071526',
        color: '#F8FAFC',
        border: '1px solid #1E293B',
        borderRadius: 2,
        fontFamily: 'sans-serif',
      }}
    >
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <HistoryEduIcon sx={{ color: '#00D4FF', fontSize: 26 }} />
          <Box>
            <Typography variant="subtitle1" sx={{ fontWeight: 700, fontSize: 16 }}>
              Bitemporal ($T_e \perp T_k$) & Brinson-Fachler Multi-Factor Attribution
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Dual-axis point-in-time reconstruction with zero-copy shadow sandboxing
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={1}>
          <Chip
            icon={<ShieldIcon sx={{ fontSize: '14px !important', color: '#10B981' }} />}
            label="Merkle Sealed (SEC 17a-4)"
            size="small"
            sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 10 }}
          />
        </Stack>
      </Box>

      {/* Dual Bitemporal Slider Controls */}
      <Grid container spacing={3} mb={3}>
        <Grid   size={{ xs: 12, md: 6 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center" mb={1}>
              <Typography variant="caption" sx={{ color: '#38BDF8', fontWeight: 700 }}>
                Effective Event Time ($T_e$ - Economic Reality)
              </Typography>
              <Chip label={dates[effectiveDateIdx]} size="small" sx={{ bgcolor: '#0284C7', color: '#fff', fontSize: 10, fontWeight: 700 }} />
            </Stack>
            <Slider
              value={effectiveDateIdx}
              min={0}
              max={dates.length - 1}
              step={1}
              marks
              onChange={(_, val) => setEffectiveDateIdx(val as number)}
              sx={{ color: '#38BDF8' }}
            />
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, md: 6 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center" mb={1}>
              <Typography variant="caption" sx={{ color: '#F59E0B', fontWeight: 700 }}>
                Knowledge Time ($T_k$ - System As-Of Time)
              </Typography>
              <Chip label={dates[knowledgeDateIdx]} size="small" sx={{ bgcolor: '#D97706', color: '#fff', fontSize: 10, fontWeight: 700 }} />
            </Stack>
            <Slider
              value={knowledgeDateIdx}
              min={0}
              max={dates.length - 1}
              step={1}
              marks
              onChange={(_, val) => setKnowledgeDateIdx(val as number)}
              sx={{ color: '#F59E0B' }}
            />
          </Paper>
        </Grid>
      </Grid>

      {/* Attribution Factor Table */}
      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Brinson-Fachler Multi-Factor Performance Drivers (SpotIQ++ Engine)
      </Typography>

      <Stack spacing={1.5}>
        {attributionFactors.map((f, i) => (
          <Paper
            key={i}
            sx={{
              p: 1.5,
              bgcolor: '#0B1E36',
              border: '1px solid #1E293B',
              borderRadius: 1.5,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <Box>
              <Typography variant="body2" sx={{ fontWeight: 600, color: '#F8FAFC' }}>
                {f.factorName}
              </Typography>
              <Typography variant="caption" sx={{ color: '#94A3B8' }}>
                Allocation: {f.allocationEffectPct > 0 ? `+${f.allocationEffectPct}` : f.allocationEffectPct}% | Selection: {f.selectionEffectPct > 0 ? `+${f.selectionEffectPct}` : f.selectionEffectPct}%
              </Typography>
            </Box>

            <Chip
              label={`${f.totalImpactBps > 0 ? `+${f.totalImpactBps}` : f.totalImpactBps} bps`}
              size="small"
              sx={{
                bgcolor: f.totalImpactBps > 0 ? 'rgba(16, 185, 129, 0.15)' : 'rgba(239, 68, 68, 0.15)',
                color: f.totalImpactBps > 0 ? '#34D399' : '#FCA5A5',
                border: `1px solid ${f.totalImpactBps > 0 ? '#059669' : '#DC2626'}`,
                fontWeight: 700,
                fontFamily: 'monospace',
              }}
            />
          </Paper>
        ))}
      </Stack>
    </Paper>
  );
};
