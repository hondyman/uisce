import React from 'react';
import { Box, Typography, TextField, Paper } from '@mui/material';
import { FreezePaneConfig } from '../tableColumnModel';

interface FreezePaneEditorProps {
  freezePane: FreezePaneConfig;
  onChange: (fp: FreezePaneConfig) => void;
}

export const FreezePaneEditor: React.FC<FreezePaneEditorProps> = ({ freezePane, onChange }) => {
  const update = (partial: Partial<FreezePaneConfig>) => onChange({ ...freezePane, ...partial });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.7rem' }}>
        Freeze rows and columns so they remain visible when scrolling.
      </Typography>

      <Box sx={{ display: 'flex', gap: 1 }}>
        <TextField size="small" label="Header Rows" type="number"
          value={freezePane.frozenHeaderRows}
          onChange={e => update({ frozenHeaderRows: Math.max(0, Number(e.target.value) || 0) })}
          inputProps={{ min: 0, max: 10 }}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' }, flex: 1 }} />
        <TextField size="small" label="Header Cols" type="number"
          value={freezePane.frozenHeaderColumns}
          onChange={e => update({ frozenHeaderColumns: Math.max(0, Number(e.target.value) || 0) })}
          inputProps={{ min: 0, max: 10 }}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' }, flex: 1 }} />
      </Box>

      <Box sx={{ display: 'flex', gap: 1 }}>
        <TextField size="small" label="Trailing Rows" type="number"
          value={freezePane.frozenTrailingRows}
          onChange={e => update({ frozenTrailingRows: Math.max(0, Number(e.target.value) || 0) })}
          inputProps={{ min: 0, max: 10 }}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' }, flex: 1 }} />
        <TextField size="small" label="Trailing Cols" type="number"
          value={freezePane.frozenTrailingColumns}
          onChange={e => update({ frozenTrailingColumns: Math.max(0, Number(e.target.value) || 0) })}
          inputProps={{ min: 0, max: 10 }}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' }, flex: 1 }} />
      </Box>

      {(freezePane.frozenHeaderRows > 0 || freezePane.frozenHeaderColumns > 0) && (
        <Paper sx={{ p: 1, bgcolor: 'rgba(0,212,255,0.05)', border: '1px solid rgba(0,212,255,0.15)', borderRadius: 1 }}>
          <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.65rem', display: 'block' }}>
            Preview: first {freezePane.frozenHeaderRows} row(s) and {freezePane.frozenHeaderColumns} column(s) will stay fixed while scrolling.
          </Typography>
        </Paper>
      )}
    </Box>
  );
};

export default FreezePaneEditor;
