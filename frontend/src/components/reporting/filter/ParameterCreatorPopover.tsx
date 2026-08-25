import React, { useState } from 'react';
import {
  Box, Typography, TextField, Select, MenuItem, Button,
  Popover, FormControl, InputLabel,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';

export type ParameterType = 'string' | 'number' | 'date' | 'boolean' | 'list';

export interface DerivedFieldMeta {
  fieldLabel: string;
  fieldName: string;
  fieldType: string;
  fieldExpr?: string;
}

function humanize(str: string): string {
  return str
    .replace(/_/g, ' ')
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .split(' ')
    .map(w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join(' ');
}

function toCamelCase(str: string): string {
  return str
    .replace(/_/g, ' ')
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .split(' ')
    .map((w, i) => i === 0 ? w.charAt(0).toUpperCase() + w.slice(1).toLowerCase() : w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join('')
    .replace(/[^a-zA-Z0-9]/g, '');
}

function deriveParameterType(fieldType: string, fieldExpr?: string): ParameterType {
  if (fieldExpr) {
    const upper = fieldExpr.toUpperCase();
    if (upper.includes('YEAR') || upper.includes('MONTH') || upper.includes('DAY') ||
        upper.includes('ROUND') || upper.includes('FLOOR') || upper.includes('CEIL') ||
        upper.includes('ABS') || upper.includes('LENGTH') || upper.includes('MOD') ||
        upper.includes('EXTRACT')) {
      return 'number';
    }
    if (upper.includes('DATE') || upper.includes('TODAY') || upper.includes('YESTERDAY') ||
        upper.includes('MTD') || upper.includes('QTD') || upper.includes('YTD') ||
        upper.includes('BUSINESS') || upper.includes('DATESINPERIOD')) {
      return 'date';
    }
    if (upper.includes('UPPER') || upper.includes('LOWER') || upper.includes('TRIM') ||
        upper.includes('SUBSTR') || upper.includes('REPLACE') || upper.includes('CONCAT') ||
        upper.includes('REGEXP') || upper.includes('JSON') || upper.includes('COALESCE') ||
        upper.includes('CAST') || upper.includes('TO_')) {
      return 'string';
    }
  }
  const t = (fieldType || 'string').toLowerCase();
  if (t.includes('int') || t.includes('float') || t.includes('decimal') ||
      t.includes('numeric') || t.includes('currency') || t.includes('money') ||
      t.includes('double') || t.includes('number')) return 'number';
  if (t.includes('date') || t.includes('time') || t.includes('timestamp')) return 'date';
  if (t.includes('bool')) return 'boolean';
  return 'string';
}

function deriveName(fieldLabel: string, fieldExpr?: string): string {
  let base = fieldLabel || '';
  if (fieldExpr) {
    const match = /^([A-Z_]+)\s*\(\s*([^)]+)\s*\)/i.exec(fieldExpr);
    if (match) {
      const fn = match[1].toLowerCase();
      if (['year', 'month', 'day'].includes(fn)) base = fieldLabel;
      else base = `${fn.charAt(0).toUpperCase() + fn.slice(1)}${fieldLabel}`;
    }
  }
  return toCamelCase(base) + 'Param';
}

interface Props {
  anchorEl: HTMLElement | null;
  onClose: () => void;
  field: DerivedFieldMeta;
  onSave: (param: { name: string; type: ParameterType; prompt: string; sourceType: 'manual'; defaultValue?: string }) => void;
}

const ParameterCreatorPopover: React.FC<Props> = ({ anchorEl, onClose, field, onSave }) => {
  const derivedName = deriveName(field.fieldLabel, field.fieldExpr);
  const derivedType = deriveParameterType(field.fieldType, field.fieldExpr);
  const derivedPrompt = humanize(field.fieldLabel);

  const [name, setName] = useState(derivedName);
  const [type, setType] = useState<ParameterType>(derivedType);
  const [prompt, setPrompt] = useState(derivedPrompt);
  const [defaultValue, setDefaultValue] = useState('');
  const [error, setError] = useState('');

  const open = Boolean(anchorEl);

  const handleSave = () => {
    if (!name.trim()) { setError('Name is required'); return; }
    if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(name.trim())) {
      setError('Name must start with a letter and contain only letters, numbers, underscores');
      return;
    }
    setError('');
    onSave({
      name: name.trim(),
      type,
      prompt: prompt.trim() || name.trim(),
      sourceType: 'manual',
      defaultValue: defaultValue.trim() || undefined,
    });
    onClose();
  };

  return (
    <Popover
      open={open}
      anchorEl={anchorEl}
      onClose={onClose}
      anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
      transformOrigin={{ vertical: 'top', horizontal: 'left' }}
      slotProps={{
        paper: {
          sx: {
            bgcolor: '#071526',
            border: '1px solid #1E293B',
            borderRadius: 2,
            width: 360,
            overflow: 'hidden',
          },
        },
      }}
    >
      <Box sx={{
        px: 2, py: 1.5,
        borderBottom: '1px solid #1E293B',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        bgcolor: '#0B1E36',
      }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <AddIcon sx={{ fontSize: 16, color: '#00D4FF' }} />
          <Typography sx={{ fontSize: '0.78rem', fontWeight: 700, color: '#E2E8F0' }}>
            New Parameter
          </Typography>
        </Box>
        <Typography sx={{ fontSize: '0.65rem', color: '#64748B' }}>
          Derived from: <span style={{ color: '#00D4FF', fontFamily: 'monospace' }}>{field.fieldExpr || field.fieldName}</span>
        </Typography>
      </Box>

      <Box sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
        <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 1.5 }}>
          <TextField
            label="Name"
            size="small"
            value={name}
            onChange={e => { setName(e.target.value); setError(''); }}
            error={Boolean(error)}
            helperText={error || (
              <span style={{ fontSize: '0.6rem', color: '#64748B' }}>
                Auto-derived from field
              </span>
            )}
            inputProps={{ style: { fontSize: '0.75rem', fontFamily: 'monospace' } }}
            InputLabelProps={{ style: { fontSize: '0.72rem' } }}
            sx={{ '& .MuiOutlinedInput-root': { bgcolor: '#0B1E36', '& fieldset': { borderColor: '#1E293B' }, '&:hover fieldset': { borderColor: '#334155' }, '&.Mui-focused fieldset': { borderColor: '#00D4FF' } } }}
          />
          <FormControl size="small" sx={{ '& .MuiOutlinedInput-root': { bgcolor: '#0B1E36', '& fieldset': { borderColor: '#1E293B' }, '&:hover fieldset': { borderColor: '#334155' }, '&.Mui-focused fieldset': { borderColor: '#00D4FF' } } }}>
            <InputLabel sx={{ fontSize: '0.72rem' }}>Type</InputLabel>
            <Select
              value={type}
              label="Type"
              onChange={e => setType(e.target.value as ParameterType)}
              sx={{ fontSize: '0.75rem' }}
            >
              <MenuItem value="string" sx={{ fontSize: '0.75rem' }}>string</MenuItem>
              <MenuItem value="number" sx={{ fontSize: '0.75rem' }}>number</MenuItem>
              <MenuItem value="date" sx={{ fontSize: '0.75rem' }}>date</MenuItem>
              <MenuItem value="boolean" sx={{ fontSize: '0.75rem' }}>boolean</MenuItem>
            </Select>
          </FormControl>
        </Box>

        <TextField
          label="Prompt"
          size="small"
          value={prompt}
          onChange={e => setPrompt(e.target.value)}
          placeholder="User-facing label shown at runtime"
          inputProps={{ style: { fontSize: '0.75rem' } }}
          InputLabelProps={{ style: { fontSize: '0.72rem' } }}
          sx={{ '& .MuiOutlinedInput-root': { bgcolor: '#0B1E36', '& fieldset': { borderColor: '#1E293B' }, '&:hover fieldset': { borderColor: '#334155' }, '&.Mui-focused fieldset': { borderColor: '#00D4FF' } } }}
        />

        <TextField
          label="Default Value"
          size="small"
          value={defaultValue}
          onChange={e => setDefaultValue(e.target.value)}
          placeholder="Optional"
          inputProps={{ style: { fontSize: '0.75rem' } }}
          InputLabelProps={{ style: { fontSize: '0.72rem' } }}
          sx={{ '& .MuiOutlinedInput-root': { bgcolor: '#0B1E36', '& fieldset': { borderColor: '#1E293B' }, '&:hover fieldset': { borderColor: '#334155' }, '&.Mui-focused fieldset': { borderColor: '#00D4FF' } } }}
        />

        <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
          <Button
            size="small"
            onClick={onClose}
            sx={{
              textTransform: 'none', fontSize: '0.72rem',
              color: '#64748B', border: '1px solid #1E293B',
              '&:hover': { bgcolor: '#0B1E36', borderColor: '#334155' },
            }}
          >
            Cancel
          </Button>
          <Button
            size="small"
            variant="contained"
            onClick={handleSave}
            startIcon={<AddIcon sx={{ fontSize: 14 }} />}
            sx={{
              textTransform: 'none', fontSize: '0.72rem', fontWeight: 700,
              bgcolor: '#00D4FF', color: '#050D1A',
              '&:hover': { bgcolor: '#00B8DB' },
            }}
          >
            Create Parameter
          </Button>
        </Box>
      </Box>
    </Popover>
  );
};

export default ParameterCreatorPopover;
