import React from 'react';
import { Box, Typography, LinearProgress, Paper, Alert } from '@mui/material';
import { SyncStatus as SyncStatusType } from '../../types/calendar';

interface Props {
  status: SyncStatusType | null;
}

export const SyncStatus: React.FC<Props> = ({ status }) => {
  if (!status) return null;

  const isRunning = status.status === 'running' || status.status === 'pending';
  const isFailed = status.status === 'failed';

  return (
    <Paper sx={{ p: 2, mb: 3 }}>
      <Typography variant="h6" gutterBottom>
        Sync Status: {status.status.toUpperCase()}
      </Typography>
      
      {isRunning && <LinearProgress sx={{ mb: 2 }} />}

      <Box display="flex" gap={3} mb={isFailed ? 2 : 0}>
        <Typography variant="body2">Processed: {status.events_processed}</Typography>
        <Typography variant="body2" color="success.main">Created: {status.events_created}</Typography>
        <Typography variant="body2" color="info.main">Updated: {status.events_updated}</Typography>
        <Typography variant="body2" color="error.main">Deleted: {status.events_deleted}</Typography>
      </Box>

      {isFailed && status.errors && status.errors.length > 0 && (
        <Alert severity="error">
          <Typography variant="subtitle2">Errors occurred during sync:</Typography>
          <ul>
            {status.errors.map((err, i) => (
              <li key={i}>{err}</li>
            ))}
          </ul>
        </Alert>
      )}
    </Paper>
  );
};
