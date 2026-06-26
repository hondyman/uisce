import React from 'react';
import { Box, Typography, Paper, Grid, Divider } from '@mui/material';
import { SyncConflict } from '../../hooks/useConflictResolution';
import dayjs from 'dayjs';

interface Props {
  conflict: SyncConflict;
}

export const ConflictDetail: React.FC<Props> = ({ conflict }) => {
  if (!conflict) return null;

  const intData = conflict.internal_event_data || {};
  const gData = conflict.google_event_data || {};

  return (
    <Box sx={{ p: 2, border: '1px solid #ddd', borderRadius: 1, mb: 2 }}>
      <Typography variant="h6" gutterBottom>
        Type: {conflict.conflict_type.replace('_', ' ')}
      </Typography>
      
      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <Paper elevation={0} sx={{ p: 2, bgcolor: '#f5f5f5' }}>
            <Typography variant="subtitle1" fontWeight="bold">Internal Event</Typography>
            <Divider sx={{ my: 1 }} />
            <Typography variant="body2"><strong>Title:</strong> {intData.title || 'N/A'}</Typography>
            <Typography variant="body2"><strong>Start:</strong> {intData.start_time ? dayjs(intData.start_time).format('lll') : 'N/A'}</Typography>
            <Typography variant="body2"><strong>End:</strong> {intData.end_time ? dayjs(intData.end_time).format('lll') : 'N/A'}</Typography>
          </Paper>
        </Grid>
        
        <Grid item xs={12} md={6}>
          <Paper elevation={0} sx={{ p: 2, bgcolor: '#e3f2fd' }}>
            <Typography variant="subtitle1" fontWeight="bold">Google Calendar Event</Typography>
            <Divider sx={{ my: 1 }} />
            <Typography variant="body2"><strong>Title:</strong> {gData.summary || 'N/A'}</Typography>
            <Typography variant="body2"><strong>Start:</strong> {gData.start?.dateTime ? dayjs(gData.start.dateTime).format('lll') : 'N/A'}</Typography>
            <Typography variant="body2"><strong>End:</strong> {gData.end?.dateTime ? dayjs(gData.end.dateTime).format('lll') : 'N/A'}</Typography>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};
