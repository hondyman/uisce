import React from 'react';
import { Box, Typography, Paper } from '@mui/material';
import { QueryExecutionResponse, ChartViewMode } from '../types/explorerTypes';
import { EnhancedFinancialGrid } from './EnhancedFinancialGrid';
import { DynamicChart } from './DynamicChart';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface SmartVisualizerPaneProps {
  viewMode: ChartViewMode;
  results: QueryExecutionResponse | null;
  onDrillDown?: (params: { dimensionKey: string; dimensionValue: string }) => void;
}

export const SmartVisualizerPane: React.FC<SmartVisualizerPaneProps> = ({ viewMode, results, onDrillDown }) => {
  const theme = useExplorerTheme();

  if (!results || results.rows.length === 0) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', minHeight: 300 }}>
        <Typography variant="body2" sx={{ color: theme.textMuted }}>
          No data returned. Adjust your prompt or filters.
        </Typography>
      </Box>
    );
  }

  if (viewMode === 'kpi' || results.columns.filter((c) => c.type !== 'number' && c.category !== 'measure').length === 0) {
    return (
      <Box sx={{ p: 2, display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr', md: '1fr 1fr 1fr' }, gap: 2 }}>
        {results.columns.map((col) => (
          <Paper
            key={col.key}
            elevation={0}
            sx={{
              p: 3,
              bgcolor: theme.backgroundElevated,
              border: `1px solid ${theme.border}`,
              borderRadius: 2.5,
              display: 'flex',
              flexDirection: 'column',
              gap: 1,
            }}
          >
            <Typography variant="caption" sx={{ color: theme.textMuted, fontWeight: 800, textTransform: 'uppercase', letterSpacing: 0.5 }}>
              {col.label}
            </Typography>
            <Typography variant="h4" sx={{ color: theme.text, fontWeight: 800, fontFamily: 'monospace' }}>
              {col.format === 'currency' ? '$' : ''}
              {Number(results.rows[0]?.[col.key] || 0).toLocaleString('en-US', { maximumFractionDigits: 2 })}
              {col.format === 'percent' ? '%' : ''}
            </Typography>
          </Paper>
        ))}
      </Box>
    );
  }

  if (viewMode === 'table') {
    return (
      <EnhancedFinancialGrid
        results={results}
        onRowClick={(row) => {
          if (onDrillDown) {
            const firstDim = results.columns.find((c) => c.category === 'dimension' || c.category === 'time');
            if (firstDim && row[firstDim.key]) {
              onDrillDown({ dimensionKey: firstDim.key, dimensionValue: String(row[firstDim.key]) });
            }
          }
        }}
      />
    );
  }

  return (
    <Paper
      elevation={0}
      sx={{
        height: '100%',
        minHeight: 450,
        display: 'flex',
        flexDirection: 'column',
        bgcolor: theme.backgroundElevated,
        border: `1px solid ${theme.border}`,
        borderRadius: 2.5,
        overflow: 'hidden',
      }}
    >
      <DynamicChart results={results} viewMode={viewMode} onPointClick={onDrillDown} />
    </Paper>
  );
};

export default SmartVisualizerPane;
