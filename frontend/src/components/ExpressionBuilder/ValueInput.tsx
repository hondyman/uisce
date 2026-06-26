import type { FC } from 'react';
import { TextField, Select, MenuItem } from '@mui/material';

type Props = { value: any; field?: string | null; onChange: (v: any) => void };

const ValueInput: FC<Props> = ({ value, field, onChange }) => {
  const fieldType = field === 'age' || field === 'total' ? 'number' : field === 'is_vip' ? 'boolean' : 'string';

  if (fieldType === 'number') {
    return (
      <TextField
        type="number"
        value={value || ''}
        onChange={(e) => onChange(e.target.value ? Number(e.target.value) : null)}
        sx={{ width: 120 }}
        size="small"
        aria-label="Number value"
      />
    );
  }

  if (fieldType === 'boolean') {
    return (
      <Select
        value={String(value || '')}
        onChange={(e) => onChange(e.target.value === 'true')}
        size="small"
        sx={{ width: 120 }}
        aria-label={`Boolean value for ${field || 'field'}`}
      >
        <MenuItem value="">(choose)</MenuItem>
        <MenuItem value="true">true</MenuItem>
        <MenuItem value="false">false</MenuItem>
      </Select>
    );
  }

  return (
    <TextField
      value={value || ''}
      onChange={(e) => onChange(e.target.value)}
      sx={{ width: 140 }}
      size="small"
      placeholder="Value"
      aria-label="Value"
    />
  );
};

export default ValueInput;
