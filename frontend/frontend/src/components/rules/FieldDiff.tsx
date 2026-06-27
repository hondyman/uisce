import React from 'react';
import { Box, Typography } from '@mui/material';

export const FieldDiff: React.FC<{ coreFields: string[]; overrideFields: string[] }> = ({ coreFields, overrideFields }) => {
  const coreSet = new Set(coreFields);
  const overrideSet = new Set(overrideFields);
  const added = overrideFields.filter(f => !coreSet.has(f));
  const removed = coreFields.filter(f => !overrideSet.has(f));

  return (
    <Box>
      <Typography variant="h6">Field Changes</Typography>
      <Box component="ul">
        {added.map(f => (
          <li key={`+${f}`} style={{ color: '#1b5e20' }}>+ {f}</li>
        ))}
        {removed.map(f => (
          <li key={`-${f}`} style={{ color: '#b71c1c' }}>- {f}</li>
        ))}
      </Box>
    </Box>
  );
};

export default FieldDiff;