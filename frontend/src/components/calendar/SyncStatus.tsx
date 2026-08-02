import React from 'react';
import { Box, Typography, LinearProgress, Paper, Alert, Chip } from '@mui/material';
import type { SyncStatus as SyncStatusType } from '../../types/calendar';

interface Props {
  status: SyncStatusType | null;
}

export const SyncStatusCard: React.FC<Props> = ({ status }) => {
  if (!status) return null;

  const isRunning = status.status === 'running' || status.status === 'pending';
  const isFailed = status.status === 'failed';

  return (
    <Paper sx={{ p: 2, mb: 3 }}>
      <Box display="flex" alignItems="center" gap={2} mb={1}>
        <Typography variant="h6" gutterBottom sx={{ mb: 0 }}>
          Sync Status: {status.status.toUpperCase()}
        </Typography>
        <Chip size="small" label={status.provider} variant="outlined" />
      </Box>

      {isRunning && (
        <Box mb={2}>
          <LinearProgress variant="determinate" value={status.progress || 0} />
          <Typography variant="caption" color="textSecondary" sx={{ mt: 0.5, display: 'block' }}>
            {status.progress || 0}% complete — {status.processed_events || 0} / {status.total_events || 0} events
          </Typography>
        </Box>
      )}

      {!isRunning && (
        <Box display="flex" gap={3}>
          <Typography variant="body2">Processed: {status.processed_events || 0}</Typography>
          <Typography variant="body2" color="success.main">Completed: {status.status === 'completed' ? 'Yes' : 'No'}</Typography>
        </Box>
      )}

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
