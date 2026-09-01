import React from 'react';
import { Box, Typography, TextField, Button, FormControl, InputLabel, Select, MenuItem, Divider } from '@mui/material';
import { CellStyle, BorderSide } from '../tableColumnModel';

interface CellStyleEditorProps {
  label: string;
  style: CellStyle;
  onChange: (style: CellStyle) => void;
}

const FONT_FAMILIES = ['Calibri', 'Arial', 'Segoe UI', 'Roboto', 'Times New Roman', 'Georgia', 'Courier New', 'Verdana'];
const FONT_SIZES = [8, 9, 10, 11, 12, 14, 16, 18, 20, 24, 28, 32, 36, 48, 72];

export const CellStyleEditor: React.FC<CellStyleEditorProps> = ({ label, style, onChange }) => {
  const update = (partial: Partial<CellStyle>) => onChange({ ...style, ...partial });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
        {label}
      </Typography>

      <Box sx={{ display: 'flex', gap: 1 }}>
        <FormControl size="small" fullWidth>
          <InputLabel sx={{ fontSize: '0.65rem' }}>Font</InputLabel>
          <Select value={style.fontFamily || ''} label="Font"
            onChange={e => update({ fontFamily: String(e.target.value) })}
            sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' } }}>
            {FONT_FAMILIES.map(f => <MenuItem key={f} value={f} sx={{ fontSize: '0.72rem' }}>{f}</MenuItem>)}
          </Select>
        </FormControl>
        <FormControl size="small" sx={{ width: 65 }}>
          <InputLabel sx={{ fontSize: '0.65rem' }}>Size</InputLabel>
          <Select value={style.fontSize || 11} label="Size"
            onChange={e => update({ fontSize: Number(e.target.value) })}
            sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' } }}>
            {FONT_SIZES.map(s => <MenuItem key={s} value={s} sx={{ fontSize: '0.72rem' }}>{s}</MenuItem>)}
          </Select>
        </FormControl>
      </Box>

      <Box sx={{ display: 'flex', gap: 0.5 }}>
        <Button size="small" variant={style.fontWeight === 700 ? 'contained' : 'outlined'}
          onClick={() => update({ fontWeight: style.fontWeight === 700 ? 400 : 700 })}
          sx={{ flex: 1, py: 0.3, fontSize: '0.7rem', fontWeight: 700, minWidth: 28 }}>
          B
        </Button>
        <Button size="small" variant={style.fontStyle === 'italic' ? 'contained' : 'outlined'}
          onClick={() => update({ fontStyle: style.fontStyle === 'italic' ? 'normal' : 'italic' })}
          sx={{ flex: 1, py: 0.3, fontSize: '0.7rem', fontStyle: 'italic', minWidth: 28 }}>
          I
        </Button>
        <Button size="small" variant={style.textDecoration === 'underline' ? 'contained' : 'outlined'}
          onClick={() => update({ textDecoration: style.textDecoration === 'underline' ? 'none' : 'underline' })}
          sx={{ flex: 1, py: 0.3, fontSize: '0.7rem', textDecoration: 'underline', minWidth: 28 }}>
          U
        </Button>
        <Button size="small" variant={style.textDecoration === 'line-through' || style.textDecoration === 'underline line-through' ? 'contained' : 'outlined'}
          onClick={() => update({ textDecoration: style.textDecoration?.includes('line-through') ? 'underline' : 'line-through' })}
          sx={{ flex: 1, py: 0.3, fontSize: '0.7rem', textDecoration: 'line-through', minWidth: 28 }}>
          S
        </Button>
      </Box>

      <FormControl size="small" fullWidth>
        <InputLabel sx={{ fontSize: '0.65rem' }}>Transform</InputLabel>
        <Select value={style.textTransform || 'none'} label="Transform"
          onChange={e => update({ textTransform: e.target.value as CellStyle['textTransform'] })}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' } }}>
          <MenuItem value="none" sx={{ fontSize: '0.72rem' }}>None</MenuItem>
          <MenuItem value="uppercase" sx={{ fontSize: '0.72rem' }}>UPPERCASE</MenuItem>
          <MenuItem value="lowercase" sx={{ fontSize: '0.72rem' }}>lowercase</MenuItem>
          <MenuItem value="capitalize" sx={{ fontSize: '0.72rem' }}>Capitalize</MenuItem>
        </Select>
      </FormControl>

      <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

      <Box sx={{ display: 'flex', gap: 1 }}>
        <TextField size="small" label="Text Color" type="color"
          value={style.color || '#E2E8F0'}
          onChange={e => update({ color: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, flex: 1 }}
          inputProps={{ style: { padding: '3px 6px' } }} />
        <TextField size="small" label="Fill Color" type="color"
          value={style.backgroundColor || 'transparent'}
          onChange={e => update({ backgroundColor: e.target.value })}
          sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, flex: 1 }}
          inputProps={{ style: { padding: '3px 6px' } }} />
      </Box>

      <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

      <Typography variant="caption" fontWeight={600} color="text.secondary" sx={{ fontSize: '0.63rem' }}>
        Border
      </Typography>
      {(['borderTop', 'borderRight', 'borderBottom', 'borderLeft'] as const).map(side => (
        <BorderSideEditor
          key={side}
          label={side.replace('border', '')}
          value={style[side]}
          onChange={val => update({ [side]: val })}
        />
      ))}

      <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

      <Typography variant="caption" fontWeight={600} color="text.secondary" sx={{ fontSize: '0.63rem' }}>
        Padding
      </Typography>
      <Box sx={{ display: 'flex', gap: 0.5 }}>
        {(['Top', 'Right', 'Bottom', 'Left'] as const).map((dir, i) => {
          const keys: ('paddingTop' | 'paddingRight' | 'paddingBottom' | 'paddingLeft')[] = ['paddingTop', 'paddingRight', 'paddingBottom', 'paddingLeft'];
          const key = keys[i];
          return (
            <TextField
              key={key}
              size="small"
              label={dir}
              type="number"
              value={style[key] ?? 4}
              onChange={e => update({ [key]: Number(e.target.value) || 0 })}
              sx={{ '& .MuiInputBase-input': { fontSize: '0.7rem', py: 0.5 }, flex: 1 }}
              inputProps={{ min: 0, max: 32 }}
            />
          );
        })}
      </Box>
    </Box>
  );
};

interface BorderSideEditorProps {
  label: string;
  value?: BorderSide;
  onChange: (val: BorderSide | undefined) => void;
}

const BORDER_STYLES: BorderSide['style'][] = ['none', 'solid', 'dashed', 'dotted', 'double'];

const BorderSideEditor: React.FC<BorderSideEditorProps> = ({ label, value, onChange }) => {
  const isEnabled = value && value.style !== 'none';

  return (
    <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
      <TextField
        size="small" label={label} type="number"
        value={isEnabled ? value!.width : ''}
        placeholder="0"
        onChange={e => {
          const w = Number(e.target.value) || 0;
          onChange(w > 0 ? { ...value!, width: w } : undefined);
        }}
        sx={{ '& .MuiInputBase-input': { fontSize: '0.7rem', py: 0.4 }, width: 50 }}
        inputProps={{ min: 0, max: 8 }}
      />
      <FormControl size="small" sx={{ flex: 1 }}>
        <Select
          value={value?.style || 'none'}
          onChange={e => {
            const s = e.target.value as BorderSide['style'];
            onChange(s === 'none' ? undefined : { width: value?.width || 1, style: s, color: value?.color || '#ffffff' });
          }}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.7rem', py: 0.4 } }}
        >
          {BORDER_STYLES.map(s => <MenuItem key={s} value={s} sx={{ fontSize: '0.7rem' }}>{s}</MenuItem>)}
        </Select>
      </FormControl>
      <TextField
        size="small" type="color"
        value={isEnabled ? (value!.color || '#ffffff') : '#ffffff'}
        onChange={e => isEnabled && onChange({ ...value!, color: e.target.value })}
        disabled={!isEnabled}
        sx={{ '& .MuiInputBase-input': { p: 0.25, height: 26, minWidth: 28 }, width: 36 }}
        inputProps={{ style: { padding: '2px 4px' } }}
      />
    </Box>
  );
};

export default CellStyleEditor;
