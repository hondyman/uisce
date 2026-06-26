import React from 'react';
import { Box, Typography, Button, Divider, Alert, CircularProgress } from '@mui/material';
import { useConflictResolution } from '../hooks/useConflictResolution';
import { ConflictStats } from '../components/calendar/ConflictStats';
import { ConflictList } from '../components/calendar/ConflictList';

export const ConflictsPage: React.FC = () => {
  // Hardcode tenant ID for demo purposes.
  const tenantId = import.meta.env.VITE_TENANT_ID || 'tenant-1';

  const {
    conflicts,
    isLoading,
    stats,
    handleResolve,
    handleAutoResolve,
    isResolving,
    isAutoResolving,
    error,
    clearError
  } = useConflictResolution(tenantId);

  return (
    <Box sx={{ p: 4, maxWidth: 1200, margin: '0 auto' }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">Sync Resolution Center</Typography>
        <Button 
          variant="outlined" 
          color="primary"
          onClick={() => handleAutoResolve()}
          disabled={isAutoResolving || isLoading || !conflicts?.length}
        >
          {isAutoResolving ? 'Resolving...' : 'Auto-Resolve (Low Severity)'}
        </Button>
      </Box>

      {error && (
        <Alert severity="error" onClose={clearError} sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      {isLoading ? (
        <Box display="flex" justifyContent="center" p={4}>
          <CircularProgress />
        </Box>
      ) : (
        <>
          <ConflictStats stats={stats} />
          
          <Divider sx={{ my: 4 }} />
          
          <ConflictList 
            conflicts={conflicts || []} 
            onResolve={handleResolve} 
            isResolving={isResolving} 
          />
        </>
      )}
    </Box>
  );
};
