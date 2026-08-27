import React from 'react';
import { Box, TextField, FormControl, InputLabel, Select, MenuItem, Typography, Paper } from '@mui/material';
import { ColumnConfig, FormatType } from '../tableColumnModel';
import { parseFormatMask } from '../evaluateDynamicProperty';

interface FormatMaskInputProps {
  formatType: FormatType;
  formatMask: string;
  prefix: string;
  suffix: string;
  onChange: (partial: { formatType?: FormatType; formatMask?: string; formatPrefix?: string; formatSuffix?: string }) => void;
}

const FORMAT_TYPES: FormatType[] = ['Auto', 'Currency', 'Percent', 'Decimal', 'Integer', 'Date', 'Text', 'Custom'];

export const FormatMaskInput: React.FC<FormatMaskInputProps> = ({
  formatType, formatMask, prefix, suffix, onChange,
}) => {
  const result = formatMask ? parseFormatMask(formatMask) : null;

  const sampleValues = [1234.5, -567.89, 0.25, 100];

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <FormControl size="small" fullWidth>
        <InputLabel sx={{ fontSize: '0.7rem' }}>Format Type</InputLabel>
        <Select value={formatType} label="Format Type"
          onChange={e => onChange({ formatType: e.target.value as FormatType })}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}>
          {FORMAT_TYPES.map(t => <MenuItem key={t} value={t} sx={{ fontSize: '0.72rem' }}>{t}</MenuItem>)}
        </Select>
      </FormControl>

      {formatType === 'Custom' && (
        <TextField
          size="small"
          label="Format Mask"
          value={formatMask}
          onChange={e => onChange({ formatMask: e.target.value })}
          placeholder="#,##0.00;[Red]-#,##0.00"
          fullWidth
          sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem', fontFamily: 'monospace' } }}
          helperText={
            result && !result.isSupported
              ? `Unsupported tokens: ${result.unsupportedTokens.join(', ')}`
              : 'Excel-style: #,##0.00  0%  [Red]-#,##0'
          }
          error={result !== null && !result.isSupported}
        />
      )}

      <Box sx={{ display: 'flex', gap: 1 }}>
        <TextField size="small" label="Prefix" value={prefix}
          onChange={e => onChange({ formatPrefix: e.target.value })} placeholder="$"
          sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' }, flex: 1 }} />
        <TextField size="small" label="Suffix" value={suffix}
          onChange={e => onChange({ formatSuffix: e.target.value })} placeholder="%"
          sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' }, flex: 1 }} />
      </Box>

      {result && result.isSupported && (
        <Paper sx={{ p: 0.75, bgcolor: 'rgba(0,0,0,0.15)', borderRadius: 1, border: '1px solid rgba(255,255,255,0.06)' }}>
          <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.6rem', display: 'block', mb: 0.5 }}>Preview:</Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}>
            {sampleValues.map((v, i) => (
              <Typography key={i} variant="caption" sx={{ fontSize: '0.68rem', fontFamily: 'monospace', color: v < 0 ? '#EF4444' : 'primary.main' }}>
                {v} → {prefix}{result.format(v)}{suffix}
              </Typography>
            ))}
          </Box>
        </Paper>
      )}
    </Box>
  );
};

export default FormatMaskInput;
