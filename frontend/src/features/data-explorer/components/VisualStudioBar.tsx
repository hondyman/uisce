import React, { useState, useMemo } from 'react';
import {
  Box,
  Button,
  Select,
  MenuItem,
  Typography,
  Tooltip,
  Drawer,
  Stack,
  Divider,
  TextField,
  Switch,
  FormControlLabel,
  Chip,
  Slider,
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
  Tune as TuneIcon,
  AutoAwesome as SparklesIcon,
  TableRows as TableSplitIcon,
  FilterAlt as FilterCrossIcon,
} from '@mui/icons-material';
import type {
  ViewMode,
  ChartConfig,
  ExplorerResult,
  ExplorerSource,
  ExplorerQueryState,
} from '../types/dataExplorerTypes';
import { DEFAULT_CHART_CONFIG } from '../types/dataExplorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

export interface VisualStudioProps {
  source: ExplorerSource | null;
  state: ExplorerQueryState;
  result: ExplorerResult | null;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  chartConfig?: ChartConfig;
  onChartConfigChange?: (config: ChartConfig) => void;
  activeCrossFilter?: { fieldId: string; value: string } | null;
  onClearCrossFilter?: () => void;
}

interface ChartTypeOption {
  mode: ViewMode;
  label: string;
  category: 'Grid & Matrix' | 'Trends & Comparison' | 'Part to Whole' | 'Metrics & Funnels';
  icon: React.ReactNode;
  description: string;
}

const ALL_CHARTS: ChartTypeOption[] = [
  { mode: 'table', label: 'Data Table', category: 'Grid & Matrix', icon: <TableIcon fontSize="small" />, description: 'Raw relational records with sort & pagination' },
  { mode: 'pivot', label: 'Pivot Matrix', category: 'Grid & Matrix', icon: <PivotIcon fontSize="small" />, description: 'Multi-dimensional pivot with row & column totals' },
  { mode: 'bar', label: 'Bar Chart', category: 'Trends & Comparison', icon: <BarIcon fontSize="small" />, description: 'Discrete categorical comparison' },
  { mode: 'stacked_bar', label: 'Stacked Bar', category: 'Trends & Comparison', icon: <BarIcon fontSize="small" />, description: 'Cumulative part-to-whole categorical distribution' },
  { mode: 'line', label: 'Line Chart', category: 'Trends & Comparison', icon: <LineIcon fontSize="small" />, description: 'Continuous time-series or trend analysis' },
  { mode: 'area', label: 'Area Chart', category: 'Trends & Comparison', icon: <LineIcon fontSize="small" />, description: 'Volume progression over time' },
  { mode: 'dual_axis', label: 'Dual Axis', category: 'Trends & Comparison', icon: <BarIcon fontSize="small" />, description: 'Volume bars + Trend lines with independent scales' },
  { mode: 'pie', label: 'Donut / Pie', category: 'Part to Whole', icon: <PieIcon fontSize="small" />, description: 'Proportional share of total' },
  { mode: 'treemap', label: 'Treemap', category: 'Part to Whole', icon: <TreemapIcon fontSize="small" />, description: 'Hierarchical nested rectangle density' },
  { mode: 'funnel', label: 'Funnel Stage', category: 'Metrics & Funnels', icon: <FunnelIcon fontSize="small" />, description: 'Conversion pipeline and drop-off stages' },
  { mode: 'radar', label: 'Radar / Spider', category: 'Metrics & Funnels', icon: <RadarIcon fontSize="small" />, description: 'Multi-variable polygon assessment' },
  { mode: 'gauge', label: 'KPI Gauge', category: 'Metrics & Funnels', icon: <GaugeIcon fontSize="small" />, description: 'Goal attainment & speedometer metrics' },
];

export const VisualStudioBar: React.FC<VisualStudioProps> = ({
  source,
  state,
  result,
  viewMode,
  onViewModeChange,
  chartConfig = DEFAULT_CHART_CONFIG,
  onChartConfigChange,
  activeCrossFilter,
  onClearCrossFilter,
}) => {
  const theme = useExplorerTheme();
  const [propsDrawerOpen, setPropsDrawerOpen] = useState(false);

  // AI Smart Suggestion based on returned columns & rows
  const suggestions = useMemo(() => {
    const recs: ViewMode[] = [];
    const numDimensions = state.dimensions.length;
    const numTime = state.timeDimensions.length;
    const numMeasures = state.measures.length;

    if (numDimensions === 0 && numTime === 0 && numMeasures > 0) {
      recs.push('gauge', 'table');
    } else if (numTime > 0 && numMeasures > 0) {
      recs.push('line', 'area', 'dual_axis');
    } else if (numDimensions >= 1 && numMeasures >= 2) {
      recs.push('dual_axis', 'stacked_bar', 'pivot', 'radar');
    } else if (numDimensions >= 1 && numMeasures === 1) {
      recs.push('bar', 'pie', 'treemap', 'funnel');
    } else if (numDimensions >= 2) {
      recs.push('pivot', 'treemap', 'table');
    }
    return recs;
  }, [state.dimensions.length, state.timeDimensions.length, state.measures.length]);

  const currentChart = ALL_CHARTS.find((c) => c.mode === viewMode) || ALL_CHARTS[0];

  return (
    <Box
      sx={{
        px: 2,
        py: 1,
        borderBottom: `1px solid ${theme.border}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        bgcolor: theme.backgroundElevated,
        gap: 1.5,
        flexWrap: 'wrap',
      }}
    >
      {/* Visual Type Selector Dropdown / Pills */}
      <Stack direction="row" alignItems="center" spacing={1} sx={{ flexWrap: 'wrap', gap: 0.5 }}>
        {/* Quick common modes */}
        {['table', 'pivot', 'bar', 'line', 'pie'].map((m) => {
          const isSelected = viewMode === m;
          const opt = ALL_CHARTS.find((c) => c.mode === m)!;
          return (
            <Button
              key={m}
              size="small"
              variant={isSelected ? 'contained' : 'outlined'}
              startIcon={opt.icon}
              onClick={() => onViewModeChange(m as ViewMode)}
              sx={{
                textTransform: 'none',
                fontWeight: 600,
                fontSize: 12,
                borderRadius: 2,
                bgcolor: isSelected ? theme.accent : 'transparent',
                color: isSelected ? theme.text : theme.textMuted,
                borderColor: theme.border,
                '&:hover': {
                  bgcolor: isSelected ? theme.accentDark : 'rgba(255,255,255,0.05)',
                  borderColor: theme.accent,
                },
              }}
            >
              {opt.label}
            </Button>
          );
        })}

        {/* Extended Gallery Menu Select */}
        <Select
          size="small"
          value={['table', 'pivot', 'bar', 'line', 'pie'].includes(viewMode) ? '' : viewMode}
          displayEmpty
          onChange={(e) => e.target.value && onViewModeChange(e.target.value as ViewMode)}
          sx={{
            fontSize: 12,
            fontWeight: 600,
            borderRadius: 2,
            height: 30,
            borderColor: theme.border,
            bgcolor: !['table', 'pivot', 'bar', 'line', 'pie'].includes(viewMode) ? theme.accent : 'transparent',
            color: !['table', 'pivot', 'bar', 'line', 'pie'].includes(viewMode) ? theme.text : theme.textMuted,
            '& .MuiSelect-select': { py: 0.5, px: 1.5 },
          }}
        >
          <MenuItem value="" disabled sx={{ fontSize: 12, fontWeight: 700 }}>
            More Visuals ({ALL_CHARTS.length - 5})...
          </MenuItem>
          {ALL_CHARTS.filter((c) => !['table', 'pivot', 'bar', 'line', 'pie'].includes(c.mode)).map((c) => (
            <MenuItem key={c.mode} value={c.mode} sx={{ fontSize: 12 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                {c.icon}
                <Typography variant="body2" sx={{ fontSize: 12, fontWeight: 600 }}>
                  {c.label}
                </Typography>
                <Typography variant="caption" sx={{ color: theme.textMuted, ml: 1 }}>
                  ({c.category})
                </Typography>
              </Box>
            </MenuItem>
          ))}
        </Select>

        {/* Smart Recommendations */}
        {suggestions.length > 0 && (
          <Stack direction="row" alignItems="center" spacing={0.5} sx={{ ml: 1 }}>
            <Tooltip title="AI Recommended Visualizations based on your selected fields and measures">
              <Chip
                icon={<SparklesIcon sx={{ fontSize: 13, color: '#f59e0b !important' }} />}
                label="Suggested:"
                size="small"
                variant="outlined"
                sx={{
                  height: 22,
                  fontSize: 11,
                  fontWeight: 700,
                  borderColor: 'rgba(245, 158, 11, 0.4)',
                  bgcolor: 'rgba(245, 158, 11, 0.08)',
                  color: '#f59e0b',
                }}
              />
            </Tooltip>
            {suggestions.slice(0, 3).map((rec) => {
              const opt = ALL_CHARTS.find((c) => c.mode === rec);
              if (!opt || opt.mode === viewMode) return null;
              return (
                <Chip
                  key={rec}
                  label={opt.label}
                  size="small"
                  onClick={() => onViewModeChange(rec)}
                  sx={{
                    height: 22,
                    fontSize: 11,
                    fontWeight: 600,
                    cursor: 'pointer',
                    bgcolor: 'rgba(99, 102, 241, 0.12)',
                    color: '#818cf8',
                    border: '1px solid rgba(99, 102, 241, 0.3)',
                    '&:hover': { bgcolor: 'rgba(99, 102, 241, 0.25)' },
                  }}
                />
              );
            })}
          </Stack>
        )}

        {/* Active Cross Filter Indicator */}
        {activeCrossFilter && (
          <Chip
            icon={<FilterCrossIcon sx={{ fontSize: 13, color: '#ec4899 !important' }} />}
            label={`Cross Filter: ${activeCrossFilter.fieldId} = "${activeCrossFilter.value}"`}
            size="small"
            onDelete={onClearCrossFilter}
            sx={{
              height: 24,
              fontSize: 11,
              fontWeight: 700,
              bgcolor: 'rgba(236, 72, 153, 0.12)',
              color: '#ec4899',
              border: '1px solid rgba(236, 72, 153, 0.3)',
            }}
          />
        )}
      </Stack>

      {/* Right Controls: Show Table Below Toggle, Properties Drawer Trigger & Row Count */}
      <Stack direction="row" alignItems="center" spacing={1.5}>
        {viewMode !== 'table' && viewMode !== 'pivot' && (
          <Tooltip title="Show / Hide the Data Table below the Visual to inspect cross-filtered rows">
            <Button
              size="small"
              variant={chartConfig.showTableBelow ? 'contained' : 'outlined'}
              startIcon={<TableSplitIcon />}
              onClick={() => onChartConfigChange?.({ ...chartConfig, showTableBelow: !chartConfig.showTableBelow })}
              sx={{
                textTransform: 'none',
                fontSize: 11,
                fontWeight: 600,
                borderRadius: 2,
                bgcolor: chartConfig.showTableBelow ? 'rgba(0, 201, 200, 0.15)' : 'transparent',
                color: chartConfig.showTableBelow ? theme.accent : theme.textMuted,
                borderColor: chartConfig.showTableBelow ? theme.accent : theme.border,
              }}
            >
              {chartConfig.showTableBelow ? 'Table Visible' : 'Show Table Below'}
            </Button>
          </Tooltip>
        )}

        {result && (
          <Typography variant="caption" sx={{ color: theme.textMuted, fontWeight: 600 }}>
            {result.rowCount.toLocaleString()} rows returned ({result.executionTimeMs}ms)
          </Typography>
        )}

        {viewMode !== 'table' && viewMode !== 'pivot' && (
          <Tooltip title="Configure Visual Properties (Colors, Fonts, Pie Styles, Labels, Interactivity)">
            <Button
              size="small"
              variant="outlined"
              startIcon={<TuneIcon />}
              onClick={() => setPropsDrawerOpen(true)}
              sx={{
                textTransform: 'none',
                fontSize: 12,
                fontWeight: 600,
                borderRadius: 2,
                borderColor: theme.border,
                color: theme.text,
              }}
            >
              Properties
            </Button>
          </Tooltip>
        )}
      </Stack>

      {/* Chart Properties Drawer */}
      <Drawer
        anchor="right"
        open={propsDrawerOpen}
        onClose={() => setPropsDrawerOpen(false)}
        PaperProps={{
          sx: {
            width: 340,
            bgcolor: theme.backgroundElevated,
            borderLeft: `1px solid ${theme.border}`,
            p: 2.5,
          },
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
          <Stack direction="row" alignItems="center" spacing={1}>
            <TuneIcon sx={{ color: theme.accent }} />
            <Typography variant="subtitle1" fontWeight={700} sx={{ color: theme.text }}>
              Chart & ECharts Styling
            </Typography>
          </Stack>
          <Button size="small" onClick={() => setPropsDrawerOpen(false)} sx={{ textTransform: 'none' }}>
            Done
          </Button>
        </Box>

        <Divider sx={{ mb: 2, borderColor: theme.border }} />

        <Stack spacing={2.5}>
          {/* Title & Subtitle */}
          <TextField
            label="Chart Title"
            size="small"
            value={chartConfig.title || ''}
            onChange={(e) => onChartConfigChange?.({ ...chartConfig, title: e.target.value })}
            placeholder={currentChart.label}
            fullWidth
          />

          {/* Typography & Fonts */}
          <Box>
            <Typography variant="caption" fontWeight={700} sx={{ color: theme.textMuted, mb: 1, display: 'block' }}>
              Font Family
            </Typography>
            <Select
              size="small"
              value={chartConfig.fontFamily || 'Inter'}
              onChange={(e) => onChartConfigChange?.({ ...chartConfig, fontFamily: e.target.value as any })}
              fullWidth
            >
              <MenuItem value="Inter">Inter Modern</MenuItem>
              <MenuItem value="Roboto">Roboto Standard</MenuItem>
              <MenuItem value="Monospace">Monospace Code</MenuItem>
              <MenuItem value="Georgia">Georgia Serif</MenuItem>
              <MenuItem value="System">System Native</MenuItem>
            </Select>
          </Box>

          {/* Color Palette Theme */}
          <Box>
            <Typography variant="caption" fontWeight={700} sx={{ color: theme.textMuted, mb: 1, display: 'block' }}>
              Color Palette Theme
            </Typography>
            <Select
              size="small"
              value={chartConfig.paletteTheme}
              onChange={(e) => onChartConfigChange?.({ ...chartConfig, paletteTheme: e.target.value as any })}
              fullWidth
            >
              <MenuItem value="ocean">Ocean Breeze (Cyan / Blue / Navy)</MenuItem>
              <MenuItem value="sunset">Sunset Glow (Orange / Magenta / Gold)</MenuItem>
              <MenuItem value="emerald">Emerald Forest (Mint / Green / Teal)</MenuItem>
              <MenuItem value="cyberpunk">Cyberpunk (Neon Yellow / Pink / Purple)</MenuItem>
              <MenuItem value="monokai">Monokai Dark (High-Contrast Vivid)</MenuItem>
            </Select>
          </Box>

          {/* Pie Chart Specific Styles */}
          {viewMode === 'pie' && (
            <Box>
              <Typography variant="caption" fontWeight={700} sx={{ color: theme.textMuted, mb: 1, display: 'block' }}>
                Pie / Donut ECharts Style
              </Typography>
              <Select
                size="small"
                value={chartConfig.pieStyle || 'donut'}
                onChange={(e) => onChartConfigChange?.({ ...chartConfig, pieStyle: e.target.value as any })}
                fullWidth
              >
                <MenuItem value="donut">Modern Donut (40% / 70% Radius)</MenuItem>
                <MenuItem value="rose">Nightingale Rose (Radius Variation)</MenuItem>
                <MenuItem value="solid">Classic Solid Pie</MenuItem>
                <MenuItem value="semicircle">Semi-Circle Gauge Donut</MenuItem>
              </Select>
            </Box>
          )}

          {/* Bar Chart Rounded Corner */}
          {(viewMode === 'bar' || viewMode === 'stacked_bar' || viewMode === 'dual_axis') && (
            <Box>
              <Typography variant="caption" fontWeight={700} sx={{ color: theme.textMuted, mb: 0.5, display: 'block' }}>
                Bar Corner Radius ({chartConfig.barCornerRadius ?? 4}px)
              </Typography>
              <Slider
                size="small"
                min={0}
                max={16}
                value={chartConfig.barCornerRadius ?? 4}
                onChange={(_, v) => onChartConfigChange?.({ ...chartConfig, barCornerRadius: v as number })}
              />
            </Box>
          )}

          {/* Interactivity & Cross-Filtering */}
          <Box sx={{ p: 1.5, borderRadius: 2, bgcolor: theme.background, border: `1px solid ${theme.border}` }}>
            <FormControlLabel
              control={
                <Switch
                  checked={chartConfig.enableCrossFilter !== false}
                  onChange={(e) => onChartConfigChange?.({ ...chartConfig, enableCrossFilter: e.target.checked })}
                />
              }
              label={<Typography variant="body2" fontWeight={600}>Interactive Cross-Filtering</Typography>}
            />
            <Typography variant="caption" color="text.secondary" display="block">
              Clicking a chart slice, bar, or node instantly filters the dataset.
            </Typography>
          </Box>

          {/* Legend Config */}
          <Box>
            <FormControlLabel
              control={
                <Switch
                  checked={chartConfig.showLegend}
                  onChange={(e) => onChartConfigChange?.({ ...chartConfig, showLegend: e.target.checked })}
                />
              }
              label={<Typography variant="body2">Show Legend</Typography>}
            />
            {chartConfig.showLegend && (
              <Select
                size="small"
                value={chartConfig.legendPosition}
                onChange={(e) => onChartConfigChange?.({ ...chartConfig, legendPosition: e.target.value as any })}
                fullWidth
                sx={{ mt: 1 }}
              >
                <MenuItem value="top">Top Header</MenuItem>
                <MenuItem value="bottom">Bottom Footer</MenuItem>
                <MenuItem value="left">Left Sidebar</MenuItem>
                <MenuItem value="right">Right Sidebar</MenuItem>
              </Select>
            )}
          </Box>

          {/* Data Labels & Smooth curves */}
          <FormControlLabel
            control={
              <Switch
                checked={chartConfig.showDataLabels}
                onChange={(e) => onChartConfigChange?.({ ...chartConfig, showDataLabels: e.target.checked })}
              />
            }
            label={<Typography variant="body2">Display Data Value Labels</Typography>}
          />

          {(viewMode === 'line' || viewMode === 'area') && (
            <FormControlLabel
              control={
                <Switch
                  checked={chartConfig.smoothLines}
                  onChange={(e) => onChartConfigChange?.({ ...chartConfig, smoothLines: e.target.checked })}
                />
              }
              label={<Typography variant="body2">Smooth Spline Curves</Typography>}
            />
          )}

          {/* Gauge min/max */}
          {viewMode === 'gauge' && (
            <Stack direction="row" spacing={1.5}>
              <TextField
                label="Min Threshold"
                size="small"
                type="number"
                value={chartConfig.gaugeMin ?? 0}
                onChange={(e) => onChartConfigChange?.({ ...chartConfig, gaugeMin: Number(e.target.value) })}
              />
              <TextField
                label="Max Threshold"
                size="small"
                type="number"
                value={chartConfig.gaugeMax ?? 100}
                onChange={(e) => onChartConfigChange?.({ ...chartConfig, gaugeMax: Number(e.target.value) })}
              />
            </Stack>
          )}
        </Stack>
      </Drawer>
    </Box>
  );
};

export default VisualStudioBar;
