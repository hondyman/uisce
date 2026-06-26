import React from 'react';
import { CircularProgress, Box, Typography, Alert } from '@mui/material';

export const LoadingSpinner: React.FC<{ message?: string }> = ({ message = 'Loading...' }) => (
  <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" p={3}>
    <CircularProgress size={40} sx={{ mb: 2 }} />
    <Typography color="textSecondary">{message}</Typography>
  </Box>
);

export const ErrorAlert: React.FC<{ error: Error | unknown }> = ({ error }) => {
  const message = error instanceof Error ? error.message : String(error);
  return <Alert severity="error" sx={{ my: 2 }}>{message}</Alert>;
};

export const LoadingOverlay: React.FC<{ active: boolean; message?: string }> = ({ active, message = 'Processing...' }) => {
  if (!active) return null;
  return (
    <Box
      sx={{
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(255, 255, 255, 0.7)',
        zIndex: 1000,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <CircularProgress />
      <Typography variant="h6" sx={{ mt: 2 }}>{message}</Typography>
    </Box>
  );
};
