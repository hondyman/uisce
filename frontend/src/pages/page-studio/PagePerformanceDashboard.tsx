import React from 'react';
import { Box, Paper, Typography, Grid, Card, CardContent } from '@mui/material';

export const PagePerformanceDashboard: React.FC = () => {
  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="h6" sx={{ mb: 2 }}>Performance Dashboard</Typography>
      <Grid container spacing={2}>
        <Grid item xs={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="h4">0ms</Typography>
              <Typography variant="body2" color="text.secondary">Avg. Load Time</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="h4">0</Typography>
              <Typography variant="body2" color="text.secondary">Total Views</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="h4">100%</Typography>
              <Typography variant="body2" color="text.secondary">Success Rate</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="h4">0</Typography>
              <Typography variant="body2" color="text.secondary">Errors</Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};
