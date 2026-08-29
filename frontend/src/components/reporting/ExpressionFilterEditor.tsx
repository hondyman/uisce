import React from 'react';
import { Box, Typography, TextField, Select, MenuItem, FormControl, InputLabel } from '@mui/material';

export const CategorySelector: React.FC<{ value: string; onChange: (v: string) => void }> = ({ value, onChange }) => (
  <FormControl size="small" sx={{ minWidth: 150 }}>
    <InputLabel>Category</InputLabel>
    <Select value={value} onChange={(e) => onChange(e.target.value)} label="Category">
      <MenuItem value="all">All</MenuItem>
      <MenuItem value="personal">Personal</MenuItem>
      <MenuItem value="business">Business</MenuItem>
    </Select>
  </FormControl>
);

export const ExpressionFilterEditor: React.FC<{ condition: unknown; onChange: (c: unknown) => void }> = ({ condition, onChange }) => (
  <Box sx={{ p: 2, border: '1px dashed', borderColor: 'divider', borderRadius: 1 }}>
    <Typography variant="caption" color="text.secondary">Expression Filter Editor</Typography>
    <TextField size="small" fullWidth placeholder="Filter expression" sx={{ mt: 1 }} />
  </Box>
);

export const HavingBuilder: React.FC<{ condition: unknown; onChange: (c: unknown) => void }> = ({ condition, onChange }) => (
  <Box sx={{ p: 2, border: '1px dashed', borderColor: 'divider', borderRadius: 1 }}>
    <Typography variant="caption" color="text.secondary">HAVING Clause Builder</Typography>
  </Box>
);

export const QualifyBuilder: React.FC<{ condition: unknown; onChange: (c: unknown) => void }> = ({ condition, onChange }) => (
  <Box sx={{ p: 2, border: '1px dashed', borderColor: 'divider', borderRadius: 1 }}>
    <Typography variant="caption" color="text.secondary">QUALIFY Clause Builder</Typography>
  </Box>
);

export const BitemporalBuilder: React.FC<{ condition: unknown; onChange: (c: unknown) => void }> = ({ condition, onChange }) => (
  <Box sx={{ p: 2, border: '1px dashed', borderColor: 'divider', borderRadius: 1 }}>
    <Typography variant="caption" color="text.secondary">Bitemporal Filter Builder</Typography>
  </Box>
);
