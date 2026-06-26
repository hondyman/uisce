import React from 'react';
import { Box, Typography, Paper, Grid } from '@mui/material';

interface Props {
  stats: any;
}

export const ConflictStats: React.FC<Props> = ({ stats }) => {
  if (!stats) return null;

  return (
    <Paper sx={{ p: 2, mb: 3 }}>
      <Typography variant="h6" gutterBottom>Conflict Overview</Typography>
      <Grid container spacing={2}>
        <Grid item xs={12} sm={3}>
          <Box p={2} bgcolor="error.light" borderRadius={1}>
            <Typography variant="subtitle2">Pending</Typography>
            <Typography variant="h4">{stats.pending || 0}</Typography>
          </Box>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Box p={2} bgcolor="success.light" borderRadius={1}>
            <Typography variant="subtitle2">Resolved</Typography>
            <Typography variant="h4">{stats.resolved || 0}</Typography>
          </Box>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Box p={2} bgcolor="info.light" borderRadius={1}>
            <Typography variant="subtitle2">Time Overlap</Typography>
            <Typography variant="h4">{stats.by_type?.time_overlap || 0}</Typography>
          </Box>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Box p={2} bgcolor="warning.light" borderRadius={1}>
            <Typography variant="subtitle2">Title Mismatch</Typography>
            <Typography variant="h4">{stats.by_type?.title_mismatch || 0}</Typography>
          </Box>
        </Grid>
      </Grid>
    </Paper>
  );
};
