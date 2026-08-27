import React from 'react';
import { Box, Typography, FormControlLabel, Switch, TextField, Divider, Button } from '@mui/material';
import { BandingConfig, GridlinesConfig } from '../tableColumnModel';

interface BandingEditorProps {
  banding: BandingConfig;
  onChange: (banding: BandingConfig) => void;
}

export const BandingEditor: React.FC<BandingEditorProps> = ({ banding, onChange }) => {
  const updateGrid = (partial: Partial<GridlinesConfig>) =>
    onChange({ ...banding, gridlines: { ...banding.gridlines, ...partial } });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem', textTransform: 'uppercase' }}>
        Row Banding
      </Typography>
      <FormControlLabel
        control={<Switch size="small" checked={banding.bandedRows}
          onChange={(_, v) => onChange({ ...banding, bandedRows: v })} />}
        label={<Typography variant="caption" sx={{ fontSize: '0.72rem' }}>Alternate row shading</Typography>}
      />
      {banding.bandedRows && (
        <TextField size="small" label="Band color" type="color"
          value={banding.bandColor}
          onChange={e => onChange({ ...banding, bandColor: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, width: '100%' }}
          inputProps={{ style: { padding: '3px 6px' } }}
        />
      )}

      <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

      <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem', textTransform: 'uppercase' }}>
        Column Banding
      </Typography>
      <FormControlLabel
        control={<Switch size="small" checked={banding.bandedColumns}
          onChange={(_, v) => onChange({ ...banding, bandedColumns: v })} />}
        label={<Typography variant="caption" sx={{ fontSize: '0.72rem' }}>Alternate column shading</Typography>}
      />

      <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

      <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem', textTransform: 'uppercase' }}>
        Gridlines
      </Typography>
      <Box sx={{ display: 'flex', gap: 1 }}>
        <FormControlLabel control={<Switch size="small" checked={banding.gridlines.horizontal}
          onChange={(_, v) => updateGrid({ horizontal: v })} />}
          label={<Typography variant="caption" sx={{ fontSize: '0.72rem' }}>Horizontal</Typography>} />
        <FormControlLabel control={<Switch size="small" checked={banding.gridlines.vertical}
          onChange={(_, v) => updateGrid({ vertical: v })} />}
          label={<Typography variant="caption" sx={{ fontSize: '0.72rem' }}>Vertical</Typography>} />
      </Box>
      <TextField size="small" label="Line color" type="color"
        value={banding.gridlines.color}
        onChange={e => updateGrid({ color: e.target.value })}
        sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, width: '100%' }}
        inputProps={{ style: { padding: '3px 6px' } }}
      />
      <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
        <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.65rem', width: 40 }}>Style</Typography>
        <Box sx={{ display: 'flex', gap: 0.5, flex: 1 }}>
          {(['solid', 'dashed', 'dotted', 'double'] as const).map(s => (
            <Button key={s} size="small" variant={banding.gridlines.style === s ? 'contained' : 'outlined'}
              onClick={() => updateGrid({ style: s })}
              sx={{ flex: 1, textTransform: 'none', fontSize: '0.65rem', py: 0.25, minWidth: 0 }}>
              {s}
            </Button>
          ))}
        </Box>
      </Box>
      <TextField size="small" label="Width (px)" type="number"
        value={banding.gridlines.width}
        onChange={e => updateGrid({ width: Number(e.target.value) || 1 })}
        sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem', py: 0.5 }, width: 80 }}
        inputProps={{ min: 1, max: 4 }}
      />

      <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

      <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem', textTransform: 'uppercase' }}>
        Header Row
      </Typography>
      <Box sx={{ display: 'flex', gap: 1 }}>
        <TextField size="small" label="Fill" type="color" value={banding.headerFill}
          onChange={e => onChange({ ...banding, headerFill: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, flex: 1 }}
          inputProps={{ style: { padding: '3px 6px' } }} />
        <TextField size="small" label="Text" type="color" value={banding.headerTextColor}
          onChange={e => onChange({ ...banding, headerTextColor: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, flex: 1 }}
          inputProps={{ style: { padding: '3px 6px' } }} />
      </Box>

      <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem', textTransform: 'uppercase' }}>
        Totals Row
      </Typography>
      <Box sx={{ display: 'flex', gap: 1 }}>
        <TextField size="small" label="Fill" type="color" value={banding.totalsFill}
          onChange={e => onChange({ ...banding, totalsFill: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, flex: 1 }}
          inputProps={{ style: { padding: '3px 6px' } }} />
        <TextField size="small" label="Text" type="color" value={banding.totalsTextColor}
          onChange={e => onChange({ ...banding, totalsTextColor: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, flex: 1 }}
          inputProps={{ style: { padding: '3px 6px' } }} />
      </Box>
    </Box>
  );
};

export default BandingEditor;
