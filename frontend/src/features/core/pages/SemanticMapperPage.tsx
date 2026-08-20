import React from 'react';
import { Box } from '@mui/material';
import IntelligentSemanticMapper from '../../mapper/IntelligentSemanticMapper';

export default function SemanticMapperPage() {
  return (
    <Box sx={{ height: 'calc(100vh - 64px)', display: 'flex', flexDirection: 'column' }}>
      <IntelligentSemanticMapper />
    </Box>
  );
}
