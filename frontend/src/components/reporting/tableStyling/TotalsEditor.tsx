import React from 'react';
import { Box, Typography, FormControlLabel, Switch, TextField, Button, Divider } from '@mui/material';
import { TotalsConfig, GrandTotalConfig, SubtotalsConfig, ColumnConfig, AggregateConfig } from '../tableColumnModel';

const NUMERIC_FORMAT_TYPES = new Set(['Auto', 'Currency', 'Percent', 'Decimal', 'Integer', 'Custom']);

function isNumericColumn(col: ColumnConfig): boolean {
  return NUMERIC_FORMAT_TYPES.has(col.formatType);
}

interface TotalsEditorProps {
  totals: TotalsConfig;
  onChange: (totals: TotalsConfig) => void;
  columns?: ColumnConfig[];
  onColumnsChange?: (columns: ColumnConfig[]) => void;
}

export const TotalsEditor: React.FC<TotalsEditorProps> = ({
  totals,
  onChange,
  columns = [],
  onColumnsChange,
}) => {
  const updateGrand = (partial: Partial<GrandTotalConfig>) => {
    const wasEnabled = totals.grandTotal.enabled;
    const willEnable = partial.enabled ?? totals.grandTotal.enabled;
    onChange({ ...totals, grandTotal: { ...totals.grandTotal, ...partial } });
    if (!wasEnabled && willEnable && onColumnsChange) {
      const updated = columns.map(col => {
        if (!isNumericColumn(col)) return col;
        if (col.aggregate?.enabled) return col;
        const newAggregate: AggregateConfig = {
          enabled: true,
          function: 'SUM',
          scope: 'column',
        };
        return { ...col, aggregate: newAggregate };
      });
      onColumnsChange(updated);
    }
  };

  const updateSub = (partial: Partial<SubtotalsConfig>) =>
    onChange({ ...totals, subtotals: { ...totals.subtotals, ...partial } });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      {/* Grand Total */}
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        <FormControlLabel
          control={<Switch size="small" checked={totals.grandTotal.enabled}
            onChange={(_, v) => updateGrand({ enabled: v })} />}
          label={<Typography variant="caption" sx={{ fontSize: '0.75rem', fontWeight: 600 }}>Grand Total</Typography>}
        />

        {totals.grandTotal.enabled && (
          <Box sx={{ pl: 2, display: 'flex', flexDirection: 'column', gap: 1 }}>
            <TextField
              size="small"
              label="Label"
              value={totals.grandTotal.label}
              onChange={e => updateGrand({ label: e.target.value })}
              fullWidth
              sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}
            />
            <Box sx={{ display: 'flex', gap: 0.5 }}>
              <Button
                size="small" variant={totals.grandTotal.position === 'bottom' ? 'contained' : 'outlined'}
                onClick={() => updateGrand({ position: 'bottom' })}
                sx={{ flex: 1, textTransform: 'none', fontSize: '0.7rem', py: 0.4 }}
              >
                Bottom
              </Button>
              <Button
                size="small" variant={totals.grandTotal.position === 'top' ? 'contained' : 'outlined'}
                onClick={() => updateGrand({ position: 'top' })}
                sx={{ flex: 1, textTransform: 'none', fontSize: '0.7rem', py: 0.4 }}
              >
                Top
              </Button>
            </Box>
          </Box>
        )}
      </Box>

      <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

      {/* Subtotals */}
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        <FormControlLabel
          control={<Switch size="small" checked={totals.subtotals.enabled}
            onChange={(_, v) => updateSub({ enabled: v })} />}
          label={<Typography variant="caption" sx={{ fontSize: '0.75rem', fontWeight: 600 }}>Group Subtotals</Typography>}
        />

        {totals.subtotals.enabled && (
          <Box sx={{ pl: 2, display: 'flex', flexDirection: 'column', gap: 1 }}>
            <TextField
              size="small"
              label="Label (use {groupValue} for group name)"
              value={totals.subtotals.label}
              onChange={e => updateSub({ label: e.target.value })}
              fullWidth
              sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}
              placeholder="Total {groupValue}"
            />
            <Box sx={{ display: 'flex', gap: 0.5 }}>
              <Button
                size="small" variant={totals.subtotals.position === 'bottom' ? 'contained' : 'outlined'}
                onClick={() => updateSub({ position: 'bottom' })}
                sx={{ flex: 1, textTransform: 'none', fontSize: '0.7rem', py: 0.4 }}
              >
                Below
              </Button>
              <Button
                size="small" variant={totals.subtotals.position === 'top' ? 'contained' : 'outlined'}
                onClick={() => updateSub({ position: 'top' })}
                sx={{ flex: 1, textTransform: 'none', fontSize: '0.7rem', py: 0.4 }}
              >
                Above
              </Button>
            </Box>
          </Box>
        )}
      </Box>
    </Box>
  );
};

export default TotalsEditor;
