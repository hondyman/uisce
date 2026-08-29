import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Chip,
  TextField,
  InputAdornment,
  Tooltip,
} from '@mui/material';
import {
  TableChart as TableIcon,
  PivotTableChart as PivotIcon,
  BarChart as BarIcon,
  ShowChart as LineIcon,
  PieChart as PieIcon,
  Speed as GaugeIcon,
  ViewQuilt as TreemapIcon,
  FilterList as FunnelIcon,
  Radar as RadarIcon,
  Search as SearchIcon,
  DragIndicator as DragIcon,
} from '@mui/icons-material';
import type { ViewMode } from '../types/dataExplorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

export interface VisualObjectItem {
  mode: ViewMode;
  name: string;
  category: 'Grid & Matrix' | 'Trends & Comparison' | 'Part to Whole' | 'Metrics & Funnels';
  icon: React.ReactNode;
  description: string;
  badge?: string;
}

export const VISUAL_OBJECTS: VisualObjectItem[] = [
  { mode: 'bar', name: 'Bar Chart', category: 'Trends & Comparison', icon: <BarIcon sx={{ color: '#3b82f6' }} />, description: 'Discrete categorical comparison with vertical bars', badge: 'ECharts' },
  { mode: 'stacked_bar', name: 'Stacked Bar', category: 'Trends & Comparison', icon: <BarIcon sx={{ color: '#6366f1' }} />, description: 'Multi-series cumulative categorical comparison', badge: 'ECharts' },
  { mode: 'line', name: 'Line Chart', category: 'Trends & Comparison', icon: <LineIcon sx={{ color: '#06b6d4' }} />, description: 'Continuous time-series or trend analysis with splines', badge: 'ECharts' },
  { mode: 'area', name: 'Area Chart', category: 'Trends & Comparison', icon: <LineIcon sx={{ color: '#10b981' }} />, description: 'Volume progression and filled gradient trends over time', badge: 'ECharts' },
  { mode: 'dual_axis', name: 'Dual Axis Combo', category: 'Trends & Comparison', icon: <BarIcon sx={{ color: '#ec4899' }} />, description: 'Bar volume + trend line on independent left/right scales', badge: 'ECharts' },
  { mode: 'pie', name: 'Pie & Donut', category: 'Part to Whole', icon: <PieIcon sx={{ color: '#f59e0b' }} />, description: 'Proportional share: Solid, Donut, Nightingale Rose, or Semi-Circle', badge: 'ECharts' },
  { mode: 'treemap', name: 'Treemap Matrix', category: 'Part to Whole', icon: <TreemapIcon sx={{ color: '#14b8a6' }} />, description: 'Hierarchical nested rectangles showing relative metric weight', badge: 'ECharts' },
  { mode: 'funnel', name: 'Pipeline Funnel', category: 'Metrics & Funnels', icon: <FunnelIcon sx={{ color: '#f97316' }} />, description: 'Conversion pipeline stages and sequential drop-off', badge: 'ECharts' },
  { mode: 'radar', name: 'Radar / Spider', category: 'Metrics & Funnels', icon: <RadarIcon sx={{ color: '#8b5cf6' }} />, description: 'Multi-variable polygon assessment across performance metrics', badge: 'ECharts' },
  { mode: 'gauge', name: 'KPI Speedometer', category: 'Metrics & Funnels', icon: <GaugeIcon sx={{ color: '#ef4444' }} />, description: 'Goal attainment & speedometer gauge with min/max thresholds', badge: 'ECharts' },
  { mode: 'table', name: 'Relational Table', category: 'Grid & Matrix', icon: <TableIcon sx={{ color: '#00C9C8' }} />, description: 'Structured tabular data grid with column sorting & pagination', badge: 'Native Grid' },
  { mode: 'pivot', name: 'Pivot Matrix', category: 'Grid & Matrix', icon: <PivotIcon sx={{ color: '#3b82f6' }} />, description: 'Multi-dimensional pivot with aggregated row and column totals', badge: 'Native Pivot' },
];

interface VisualObjectsPaletteProps {
  onSelectVisual: (mode: ViewMode) => void;
  activeMode?: ViewMode;
}

export const VisualObjectsPalette: React.FC<VisualObjectsPaletteProps> = ({
  onSelectVisual,
  activeMode,
}) => {
  const theme = useExplorerTheme();
  const [searchTerm, setSearchTerm] = useState('');

  const filteredVisuals = VISUAL_OBJECTS.filter(
    (v) =>
      v.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      v.category.toLowerCase().includes(searchTerm.toLowerCase()) ||
      v.description.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const categories = Array.from(new Set(filteredVisuals.map((v) => v.category)));

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', bgcolor: theme.backgroundElevated }}>
      {/* Search Header */}
      <Box sx={{ p: 1.5, borderBottom: `1px solid ${theme.border}` }}>
        <TextField
          size="small"
          placeholder="Filter visual objects..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          fullWidth
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ fontSize: 16, color: theme.textMuted }} />
              </InputAdornment>
            ),
          }}
          sx={{
            '& .MuiOutlinedInput-root': {
              fontSize: '0.78rem',
              borderRadius: 2,
              bgcolor: theme.background,
            },
          }}
        />
      </Box>

      {/* Visuals List Grouped by Category */}
      <Box sx={{ flex: 1, overflowY: 'auto', p: 1.5 }}>
        {categories.map((cat) => {
          const items = filteredVisuals.filter((v) => v.category === cat);
          return (
            <Box key={cat} sx={{ mb: 2 }}>
              <Typography
                variant="caption"
                sx={{
                  color: theme.textMuted,
                  fontWeight: 700,
                  fontSize: '0.7rem',
                  letterSpacing: 0.8,
                  textTransform: 'uppercase',
                  display: 'block',
                  mb: 1,
                  px: 0.5,
                }}
              >
                {cat} ({items.length})
              </Typography>

              <Stack spacing={1}>
                {items.map((item) => {
                  const isActive = activeMode === item.mode;
                  return (
                    <Paper
                      key={item.mode}
                      elevation={0}
                      draggable
                      onDragStart={(e) => {
                        const payload = {
                          type: 'visual_object',
                          mode: item.mode,
                          name: item.name,
                        };
                        e.dataTransfer.setData('application/json', JSON.stringify(payload));
                        e.dataTransfer.setData('text/plain', `visual:${item.mode}`);
                        e.dataTransfer.effectAllowed = 'copy';
                      }}
                      onClick={() => onSelectVisual(item.mode)}
                      sx={{
                        p: 1.25,
                        borderRadius: 2,
                        border: '1px solid',
                        borderColor: isActive ? theme.accent : theme.border,
                        bgcolor: isActive ? (theme.background) : theme.backgroundElevated,
                        cursor: 'grab',
                        transition: 'all 0.15s ease-in-out',
                        '&:hover': {
                          borderColor: theme.accent,
                          bgcolor: theme.background,
                          transform: 'translateY(-1px)',
                          boxShadow: '0 2px 8px rgba(0,0,0,0.06)',
                        },
                        '&:active': { cursor: 'grabbing' },
                      }}
                    >
                      <Box sx={{ display: 'flex', alignItems: 'center', mb: 0.5 }}>
                        <Stack direction="row" spacing={1} alignItems="center">
                          <DragIcon sx={{ fontSize: 14, color: theme.textMuted, cursor: 'grab' }} />
                          {item.icon}
                          <Typography variant="body2" sx={{ fontWeight: 700, fontSize: '0.82rem', color: theme.text }}>
                            {item.name}
                          </Typography>
                        </Stack>
                      </Box>
                      <Typography
                        variant="caption"
                        sx={{
                          color: theme.textMuted,
                          fontSize: '0.72rem',
                          lineHeight: 1.3,
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden',
                          pl: 2.75,
                        }}
                      >
                        {item.description}
                      </Typography>
                    </Paper>
                  );
                })}
              </Stack>
            </Box>
          );
        })}
      </Box>
    </Box>
  );
};

export default VisualObjectsPalette;
