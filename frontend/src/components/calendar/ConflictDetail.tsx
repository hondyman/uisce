import React from 'react';
import { Box, Typography, Paper, Grid, Divider, Chip } from '@mui/material';
import { SyncConflict } from '../../hooks/useConflictResolution';
import dayjs from 'dayjs';

interface Props {
  conflict: SyncConflict;
}

const providerLabel: Record<string, string> = {
  google: 'Google Calendar',
  microsoft: 'Microsoft Calendar',
  apple: 'Apple Calendar',
};

export const ConflictDetail: React.FC<Props> = ({ conflict }) => {
  if (!conflict) return null;

  const intData = conflict.internal_event_data || {};
  const extData = conflict.external_event_data || {};

  return (
    <Box sx={{ p: 2, border: '1px solid #ddd', borderRadius: 1, mb: 2 }}>
      <Box display="flex" alignItems="center" gap={2} mb={2}>
        <Typography variant="h6" gutterBottom sx={{ mb: 0 }}>
          Type: {conflict.conflict_type.replace('_', ' ')}
        </Typography>
        <Chip size="small" label={conflict.provider} color="primary" variant="outlined" />
      </Box>

      <Grid container spacing={3}>
        <Grid size={{ 'xs': 12, 'md': 6 }}>
          <Paper elevation={0} sx={{ p: 2, bgcolor: '#f5f5f5' }}>
            <Typography variant="subtitle1" fontWeight="bold">Internal Event</Typography>
            <Divider sx={{ my: 1 }} />
            <Typography variant="body2"><strong>Title:</strong> {intData.title || 'N/A'}</Typography>
            <Typography variant="body2"><strong>Start:</strong> {intData.start_time ? dayjs(intData.start_time).format('lll') : 'N/A'}</Typography>
            <Typography variant="body2"><strong>End:</strong> {intData.end_time ? dayjs(intData.end_time).format('lll') : 'N/A'}</Typography>
          </Paper>
        </Grid>

        <Grid size={{ 'xs': 12, 'md': 6 }}>
          <Paper elevation={0} sx={{ p: 2, bgcolor: '#e3f2fd' }}>
            <Typography variant="subtitle1" fontWeight="bold">
              {providerLabel[conflict.provider] || 'External Calendar'} Event
            </Typography>
            <Divider sx={{ my: 1 }} />
            <Typography variant="body2"><strong>Title:</strong> {extData.title || extData.summary || 'N/A'}</Typography>
            <Typography variant="body2"><strong>Start:</strong> {extData.startTime ? dayjs(extData.startTime).format('lll') : extData.start?.dateTime ? dayjs(extData.start.dateTime).format('lll') : 'N/A'}</Typography>
            <Typography variant="body2"><strong>End:</strong> {extData.endTime ? dayjs(extData.endTime).format('lll') : extData.end?.dateTime ? dayjs(extData.end.dateTime).format('lll') : 'N/A'}</Typography>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};
