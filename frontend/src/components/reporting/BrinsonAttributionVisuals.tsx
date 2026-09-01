import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Chip,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material';
import ShowChartIcon from '@mui/icons-material/ShowChart';
import WaterfallChartIcon from '@mui/icons-material/WaterfallChart';

export interface SectorAttributionItem {
  sectorKey: string;
  sectorName: string;
  portfolioWeight: number;
  benchmarkWeight: number;
  portfolioReturn: number;
  benchmarkReturn: number;
  allocationEffect: number;
  selectionEffect: number;
  interactionEffect: number;
  totalContribution: number;
}

export interface DrawdownPointItem {
  asOfDate: string;
  portfolioNav: number;
  benchmarkNav: number;
  portfolioDrawdown: number;
  benchmarkDrawdown: number;
}

interface BrinsonAttributionVisualsProps {
  portfolioName?: string;
  benchmarkName?: string;
  totalExcessReturn?: number;
  totalAllocation?: number;
  totalSelection?: number;
  totalInteraction?: number;
  maxDrawdown?: number;
  sectors?: SectorAttributionItem[];
  drawdownSeries?: DrawdownPointItem[];
}

export const BrinsonAttributionVisuals: React.FC<BrinsonAttributionVisualsProps> = ({
  portfolioName = 'Global Opportunities Fund',
  benchmarkName = 'MSCI World Index',
  totalExcessReturn = 0.0580,
  totalAllocation = 0.0096,
  totalSelection = 0.0320,
  totalInteraction = 0.0164,
  maxDrawdown = -0.1500,
  sectors = [
    {
      sectorKey: 'tech',
      sectorName: 'Technology',
      portfolioWeight: 0.60,
      benchmarkWeight: 0.40,
      portfolioReturn: 0.15,
      benchmarkReturn: 0.10,
      allocationEffect: 0.0096,
      selectionEffect: 0.0200,
      interactionEffect: 0.0100,
      totalContribution: 0.0396,
    },
    {
      sectorKey: 'energy',
      sectorName: 'Energy',
      portfolioWeight: 0.40,
      benchmarkWeight: 0.60,
      portfolioReturn: 0.05,
      benchmarkReturn: 0.02,
      allocationEffect: 0.0064,
      selectionEffect: 0.0180,
      interactionEffect: -0.0060,
      totalContribution: 0.0184,
    },
  ],
  drawdownSeries = [],
}) => {
  const [metricView, setMetricView] = useState<'TOTAL' | 'ALLOC' | 'SELECT'>('TOTAL');

  return (
    <Box sx={{ width: '100%', bgcolor: '#050D1A', color: '#fff', borderRadius: 2, border: '1px solid #1E293B', overflow: 'hidden', fontFamily: 'sans-serif' }}>
      
      {/* Top Header */}
      <Box sx={{ p: 2, bgcolor: '#071526', borderBottom: '1px solid #1E293B', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <WaterfallChartIcon sx={{ color: '#00D4FF', fontSize: 22 }} />
          <Typography variant="subtitle2" fontWeight="700" sx={{ letterSpacing: 0.5 }}>
            Brinson-Fachler Performance Attribution & Drawdown Analysis ({portfolioName})
          </Typography>
        </Box>
        <Chip
          size="small"
          label={`Benchmark: ${benchmarkName}`}
          sx={{ bgcolor: 'rgba(0, 212, 255, 0.1)', color: '#00D4FF', fontSize: '11px', border: '1px solid rgba(0, 212, 255, 0.3)' }}
        />
      </Box>

      {/* Summary Stat Tiles */}
      <Box sx={{ p: 2, bgcolor: '#0B1E36', display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: 2, borderBottom: '1px solid #1E293B' }}>
        <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.05)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8' }}>Total Excess Return (Alpha)</Typography>
          <Typography variant="h6" fontWeight="700" sx={{ color: totalExcessReturn >= 0 ? '#10B981' : '#EF4444' }}>
            {(totalExcessReturn * 100).toFixed(2)}%
          </Typography>
        </Paper>

        <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.05)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8' }}>Allocation Effect</Typography>
          <Typography variant="h6" fontWeight="700" sx={{ color: totalAllocation >= 0 ? '#00D4FF' : '#EF4444' }}>
            {(totalAllocation * 100).toFixed(2)}%
          </Typography>
        </Paper>

        <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.05)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8' }}>Selection Effect</Typography>
          <Typography variant="h6" fontWeight="700" sx={{ color: totalSelection >= 0 ? '#A855F7' : '#EF4444' }}>
            {(totalSelection * 100).toFixed(2)}%
          </Typography>
        </Paper>

        <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.05)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8' }}>Interaction Effect</Typography>
          <Typography variant="h6" fontWeight="700" sx={{ color: totalInteraction >= 0 ? '#F59E0B' : '#EF4444' }}>
            {(totalInteraction * 100).toFixed(2)}%
          </Typography>
        </Paper>

        <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.05)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8' }}>Max Portfolio Drawdown</Typography>
          <Typography variant="h6" fontWeight="700" sx={{ color: '#EF4444' }}>
            {(maxDrawdown * 100).toFixed(2)}%
          </Typography>
        </Paper>
      </Box>

      {/* Sector Attribution Bar Breakdown */}
      <Box sx={{ p: 2.5, borderBottom: '1px solid #1E293B' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
          <Typography variant="caption" fontWeight="700" sx={{ color: '#00D4FF', textTransform: 'uppercase', letterSpacing: 0.5 }}>
            Sector Attribution Breakdown ({metricView})
          </Typography>
          <ToggleButtonGroup
            size="small"
            value={metricView}
            exclusive
            onChange={(_, val) => val && setMetricView(val)}
            sx={{ bgcolor: '#050D1A', border: '1px solid #1E293B', height: 26 }}
          >
            <ToggleButton value="TOTAL" sx={{ color: '#94A3B8', fontSize: '10px', px: 1.5 }}>Total Alpha</ToggleButton>
            <ToggleButton value="ALLOC" sx={{ color: '#94A3B8', fontSize: '10px', px: 1.5 }}>Allocation</ToggleButton>
            <ToggleButton value="SELECT" sx={{ color: '#94A3B8', fontSize: '10px', px: 1.5 }}>Selection</ToggleButton>
          </ToggleButtonGroup>
        </Box>

        {/* CSS Waterfall Simulation Grid */}
        <Stack spacing={1.5}>
          {sectors.map((sec) => {
            const val = metricView === 'TOTAL' ? sec.totalContribution : metricView === 'ALLOC' ? sec.allocationEffect : sec.selectionEffect;
            const barWidthPct = Math.min(100, Math.abs(val) * 1500);
            const isPositive = val >= 0;

            return (
              <Box key={sec.sectorKey} sx={{ display: 'grid', gridTemplateColumns: '160px 1fr 80px', alignItems: 'center', gap: 2, fontSize: '11px', fontFamily: 'monospace' }}>
                <Typography variant="caption" noWrap sx={{ color: '#E2E8F0', fontFamily: 'sans-serif' }}>{sec.sectorName}</Typography>
                
                {/* Dual-Direction Center Bar */}
                <Box sx={{ display: 'flex', width: '100%', height: 16, bgcolor: '#071526', borderRadius: 0.5, position: 'relative', overflow: 'hidden' }}>
                  <Box sx={{ position: 'absolute', left: '50%', top: 0, bottom: 0, width: '1px', bgcolor: '#334155', zIndex: 1 }} />
                  {isPositive ? (
                    <Box sx={{ position: 'absolute', left: '50%', width: `${barWidthPct / 2}%`, top: 2, bottom: 2, bgcolor: '#10B981', borderRadius: '0 2px 2px 0' }} />
                  ) : (
                    <Box sx={{ position: 'absolute', right: '50%', width: `${barWidthPct / 2}%`, top: 2, bottom: 2, bgcolor: '#EF4444', borderRadius: '2px 0 0 2px' }} />
                  )}
                </Box>

                <Typography variant="caption" sx={{ textAlign: 'right', fontWeight: 700, color: isPositive ? '#10B981' : '#EF4444' }}>
                  {(val * 100).toFixed(2)}%
                </Typography>
              </Box>
            );
          })}
        </Stack>
      </Box>

      {/* Underwater Drawdown Ribbon Visualizer */}
      <Box sx={{ p: 2.5, bgcolor: '#071526' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
          <ShowChartIcon sx={{ color: '#EF4444', fontSize: 18 }} />
          <Typography variant="caption" fontWeight="700" sx={{ color: '#EF4444', textTransform: 'uppercase', letterSpacing: 0.5 }}>
            Underwater Drawdown Ribbon (% Below High-Water Mark)
          </Typography>
        </Box>

        {/* SVG Ribbon Chart */}
        <Box sx={{ width: '100%', height: 80, bgcolor: '#050D1A', borderRadius: 1, border: '1px solid #1E293B', p: 1, position: 'relative' }}>
          <svg width="100%" height="100%" viewBox="0 0 500 60" preserveAspectRatio="none">
            <path
              d="M 0,0 L 50,15 L 100,5 L 150,30 L 200,45 L 250,20 L 300,0 L 350,10 L 400,25 L 450,40 L 500,0 L 500,0 L 0,0 Z"
              fill="rgba(239, 68, 68, 0.25)"
              stroke="#EF4444"
              strokeWidth="1.5"
            />
            <line x1="0" y1="0" x2="500" y2="0" stroke="#10B981" strokeWidth="1.5" strokeDasharray="3,3" />
          </svg>
        </Box>
      </Box>

    </Box>
  );
};

export default BrinsonAttributionVisuals;
