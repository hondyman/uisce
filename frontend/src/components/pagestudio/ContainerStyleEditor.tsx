import React from 'react';
import { Box, Typography, TextField, Divider, Stack } from '@mui/material';
import type { ContainerStyle } from './pageStudioTypes';

export type { ContainerStyle };

interface ContainerStyleEditorProps {
  label?: string;
  style: ContainerStyle;
  onChange: (style: ContainerStyle) => void;
}

export const ContainerStyleEditor: React.FC<ContainerStyleEditorProps> = ({
  label = 'Container Style',
  style,
  onChange,
}) => {
  const update = (partial: Partial<ContainerStyle>) => onChange({ ...style, ...partial });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
        {label}
      </Typography>

      <Box>
        <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.5 }}>Background Color</Typography>
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
          <input
            type="color"
            value={style.backgroundColor || '#071526'}
            onChange={(e) => update({ backgroundColor: e.target.value })}
            style={{ width: 40, height: 28, background: '#0B1E36', border: '1px solid #1E293B', borderRadius: 4, cursor: 'pointer' }}
          />
          <TextField
            size="small"
            value={style.backgroundColor || ''}
            onChange={(e) => update({ backgroundColor: e.target.value })}
            placeholder="#071526"
            inputProps={{ style: { color: '#F8FAFC', fontSize: 12, fontFamily: 'monospace' } }}
            sx={{ flex: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' } }}
          />
        </Box>
      </Box>

      <Divider sx={{ borderColor: '#1E293B' }} />

      <Box>
        <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.5 }}>Padding (px)</Typography>
        <Stack direction="row" spacing={0.5}>
          {[
            { key: 'paddingTop', label: 'T' },
            { key: 'paddingRight', label: 'R' },
            { key: 'paddingBottom', label: 'B' },
            { key: 'paddingLeft', label: 'L' },
          ].map(({ key, label: l }) => (
            <TextField
              key={key}
              size="small"
              label={l}
              type="number"
              value={(style as any)[key] ?? 8}
              onChange={(e) => update({ [key]: Number(e.target.value) })}
              inputProps={{ style: { color: '#F8FAFC', fontSize: 11, textAlign: 'center' } }}
              sx={{ width: 52, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' }, '& .MuiInputLabel-root': { fontSize: 10 } }}
            />
          ))}
        </Stack>
      </Box>

      <Divider sx={{ borderColor: '#1E293B' }} />

      <Box>
        <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.5 }}>Border</Typography>
        <Stack direction="row" spacing={0.5} alignItems="center">
          <TextField
            size="small"
            label="Width"
            type="number"
            value={style.borderWidth ?? 0}
            onChange={(e) => update({ borderWidth: Number(e.target.value) })}
            inputProps={{ style: { color: '#F8FAFC', fontSize: 11 } }}
            sx={{ width: 52, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' }, '& .MuiInputLabel-root': { fontSize: 10 } }}
          />
          <TextField
            size="small"
            label="Style"
            value={style.borderStyle ?? 'none'}
            onChange={(e) => update({ borderStyle: e.target.value as ContainerStyle['borderStyle'] })}
            select
            sx={{ flex: 1, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' }, '& .MuiInputLabel-root': { fontSize: 10 }, '& .MuiSelect-select': { color: '#F8FAFC', fontSize: 11 } }}
          >
            {['none', 'solid', 'dashed', 'dotted'].map((s) => (
              <Box key={s} component="option" value={s} sx={{ fontSize: 11 }}>{s}</Box>
            ))}
          </TextField>
          <input
            type="color"
            value={style.borderColor || '#1E293B'}
            onChange={(e) => update({ borderColor: e.target.value })}
            style={{ width: 36, height: 28, background: '#0B1E36', border: '1px solid #1E293B', borderRadius: 4, cursor: 'pointer', flexShrink: 0 }}
          />
        </Stack>
      </Box>
    </Box>
  );
};

export default ContainerStyleEditor;
