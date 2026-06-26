import type { FC } from 'react';
import { Select, MenuItem } from '@mui/material';

const operators = [
  { value: '=', label: '=' },
  { value: '!=', label: '≠' },
  { value: '>', label: '>' },
  { value: '<', label: '<' },
  { value: '>=', label: '≥' },
  { value: '<=', label: '≤' },
  { value: 'contains', label: 'Contains' },
  { value: 'starts', label: 'Starts' },
  { value: 'ends', label: 'Ends' }
];

type Props = { value: string; onChange: (v: string) => void };

const OperatorSelector: FC<Props> = ({ value, onChange }) => (
  <Select 
    value={value} 
    onChange={(e) => onChange(e.target.value)}
    sx={{ width: 100 }}
    size="small"
  >
    {operators.map(op => (
      <MenuItem key={op.value} value={op.value}>
        {op.label}
      </MenuItem>
    ))}
  </Select>
);

export default OperatorSelector;
