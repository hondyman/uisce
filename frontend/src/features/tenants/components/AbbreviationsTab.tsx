import React from 'react';
import { Box } from '@mui/material';
import { AbbreviationManager } from '../../../components/AbbreviationManagerV2';

interface AbbreviationsTabProps {
  tenantId: string;
}

export default function AbbreviationsTab({ tenantId }: AbbreviationsTabProps): JSX.Element {
  return (
    <Box sx={{ p: 0 }}>
      <AbbreviationManager className="" />
    </Box>
  );
}
