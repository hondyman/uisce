import React from 'react';
import { Box, Button, Typography } from '@mui/material';

interface Props {
  conflictId: string;
  onResolve: (strategy: string) => void;
  isResolving: boolean;
}

export const ConflictResolution: React.FC<Props> = ({ conflictId, onResolve, isResolving }) => {
  return (
    <Box display="flex" gap={2} mt={2}>
      <Typography variant="subtitle2" sx={{ alignSelf: 'center' }}>Resolve using:</Typography>
      <Button 
        variant="contained" 
        color="primary" 
        size="small"
        disabled={isResolving}
        onClick={() => onResolve('keep_internal')}
      >
        Keep Internal
      </Button>
      <Button 
        variant="contained" 
        color="info" 
        size="small"
        disabled={isResolving}
        onClick={() => onResolve('keep_google')}
      >
        Keep Google
      </Button>
      <Button 
        variant="outlined" 
        color="warning" 
        size="small"
        disabled={isResolving}
        onClick={() => onResolve('merge')}
      >
        Merge
      </Button>
      <Button 
        variant="text" 
        color="error" 
        size="small"
        disabled={isResolving}
        onClick={() => onResolve('skip')}
      >
        Skip Sync
      </Button>
    </Box>
  );
};
