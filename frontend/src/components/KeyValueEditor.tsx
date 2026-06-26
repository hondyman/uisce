import type { FC } from 'react';
import { Box, TextField, IconButton, Button } from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';

interface Props {
  value: Record<string, any>;
  onChange: (v: Record<string, any>) => void;
}

const KeyValueEditor: FC<Props> = ({ value, onChange }) => {
  const entries = Object.entries(value || {});
  const setAt = (idx: number, k: string, v: any) => {
    const next: any = { ...(value || {}) };
    const key = entries[idx] ? entries[idx][0] : k;
    if (key !== k && entries[idx]) {
      // rename key
      delete next[key];
    }
    next[k] = v;
    onChange(next);
  };

  const addRow = () => {
    const next = { ...(value || {}) };
    let i = 1;
    while (next[`key_${i}`]) i++;
    next[`key_${i}`] = '';
    onChange(next);
  };

  const removeRow = (k: string) => {
    const next = { ...(value || {}) };
    delete next[k];
    onChange(next);
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      {entries.map(([k, v], idx) => (
        <Box key={k} sx={{ display: 'flex', gap: 1 }}>
          <TextField label="Key" value={k} onChange={(e) => setAt(idx, e.target.value, v)} />
          <TextField label="Value" value={typeof v === 'object' ? JSON.stringify(v) : String(v)} onChange={(e) => setAt(idx, k, e.target.value)} />
          <IconButton onClick={() => removeRow(k)}><DeleteIcon /></IconButton>
        </Box>
      ))}
      <Button onClick={addRow}>Add field</Button>
    </Box>
  );
};

export default KeyValueEditor;
