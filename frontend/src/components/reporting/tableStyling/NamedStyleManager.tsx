import React, { useState } from 'react';
import {
  Box, Typography, Button, IconButton, TextField, Paper, Divider,
} from '@mui/material';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import AddIcon from '@mui/icons-material/Add';
import { NamedStyle, CellStyle } from '../tableColumnModel';

interface NamedStyleManagerProps {
  styles: NamedStyle[];
  onChange: (styles: NamedStyle[]) => void;
  onApplyStyle?: (style: CellStyle) => void;
}

function uid() {
  return `${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
}

export const NamedStyleManager: React.FC<NamedStyleManagerProps> = ({ styles, onChange, onApplyStyle }) => {
  const [editingId, setEditingId] = useState<string | null>(null);

  const add = () => {
    const newStyle: NamedStyle = {
      id: uid(),
      name: `Style ${styles.length + 1}`,
      cellStyle: { fontFamily: 'Calibri', fontSize: 11, fontWeight: 400 },
      scope: 'body',
    };
    onChange([...styles, newStyle]);
    setEditingId(newStyle.id);
  };

  const update = (id: string, partial: Partial<NamedStyle>) =>
    onChange(styles.map(s => s.id === id ? { ...s, ...partial } : s));

  const remove = (id: string) => {
    onChange(styles.filter(s => s.id !== id));
    if (editingId === id) setEditingId(null);
  };

  const applyStyle = (style: CellStyle) => onApplyStyle?.(style);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.7rem' }}>
          {styles.length} named style(s)
        </Typography>
        <Button size="small" startIcon={<AddIcon sx={{ fontSize: 14 }} />} onClick={add}
          sx={{ textTransform: 'none', fontSize: '0.7rem', py: 0.25 }}>
          New Style
        </Button>
      </Box>

      {styles.map(style => (
        <Paper key={style.id} sx={{
          p: 1, bgcolor: 'rgba(0,0,0,0.15)',
          border: `1px solid ${editingId === style.id ? 'rgba(0,212,255,0.3)' : 'rgba(255,255,255,0.06)'}`,
          borderRadius: 1,
        }}>
          <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center', mb: 0.5 }}>
            {editingId === style.id ? (
              <TextField size="small" value={style.name}
                onChange={e => update(style.id, { name: e.target.value })}
                autoFocus
                sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.25 }, flex: 1 }}
                onBlur={() => setEditingId(null)}
                onKeyDown={e => e.key === 'Enter' && setEditingId(null)}
              />
            ) : (
              <Typography variant="caption" sx={{ flex: 1, fontSize: '0.72rem', fontWeight: 600, cursor: 'pointer' }}
                onClick={() => setEditingId(style.id)}>
                {style.name}
              </Typography>
            )}
            <Box sx={{ display: 'flex', gap: 0.25 }}>
              <Button size="small" variant="outlined" onClick={() => applyStyle(style.cellStyle)}
                sx={{ py: 0.15, px: 0.75, fontSize: '0.6rem', textTransform: 'none', minWidth: 0 }}>
                Apply
              </Button>
              <IconButton size="small" onClick={() => remove(style.id)} color="error" sx={{ p: 0.25 }}>
                <DeleteOutlineIcon sx={{ fontSize: 13 }} />
              </IconButton>
            </Box>
          </Box>

          <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center', pl: 0.5 }}>
            <Box sx={{
              width: 12, height: 12, borderRadius: '2px',
              bgcolor: style.cellStyle.backgroundColor || 'transparent',
              border: '1px solid rgba(255,255,255,0.1)',
              flexShrink: 0,
            }} />
            <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.6rem' }}>
              {style.cellStyle.fontFamily || 'Calibri'} {style.cellStyle.fontSize || 11}pt
              {style.cellStyle.fontWeight === 700 ? ' Bold' : ''}
              {style.cellStyle.fontStyle === 'italic' ? ' Italic' : ''}
            </Typography>
          </Box>
        </Paper>
      ))}

      {styles.length === 0 && (
        <Typography variant="caption" color="text.disabled" sx={{ fontStyle: 'italic', display: 'block', fontSize: '0.7rem', textAlign: 'center', py: 1 }}>
          No named styles. Create one to reuse across columns.
        </Typography>
      )}
    </Box>
  );
};

export default NamedStyleManager;
