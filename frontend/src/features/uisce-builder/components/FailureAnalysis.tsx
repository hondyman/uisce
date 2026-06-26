import React from 'react';
import { Box, Typography, Paper, Grid } from '@mui/material';

// Types will eventually come from the backend model
export interface DebugStep {
  validationRule?: string;
  actualValue?: any;
  errorDetails?: string;
  status: 'PASS' | 'FAIL';
}

interface FailureAnalysisProps {
  step: DebugStep;
}

const FailureAnalysis = ({ step }: FailureAnalysisProps) => {
  return (
    <Box 
      sx={{ 
        mt: 2, 
        p: 2, 
        bgcolor: '#fef2f2', 
        border: 1, 
        borderColor: 'error.light', 
        borderRadius: 1 
      }}
    >
      <Typography variant="subtitle2" color="error.dark" sx={{ fontWeight: 'bold', mb: 1 }}>
        ❌ Validation Failed
      </Typography>
      
      <Grid container spacing={2}>
        {/* Column 1: The Rule */}
        <Grid item xs={6}>
          <Paper variant="outlined" sx={{ p: 1.5, height: '100%' }}>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', textTransform: 'uppercase', mb: 0.5 }}>
              The Rule Required
            </Typography>
            <Typography variant="body2" component="code" sx={{ fontFamily: 'monospace', color: 'primary.main', fontWeight: 600 }}>
               {step.validationRule || 'amount < 1,000,000'}
            </Typography>
          </Paper>
        </Grid>

        {/* Column 2: The Actual Data */}
        <Grid item xs={6}>
          <Paper variant="outlined" sx={{ p: 1.5, height: '100%' }}>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', textTransform: 'uppercase', mb: 0.5 }}>
              The Trade Contained
            </Typography>
            <Typography variant="body2" component="code" sx={{ fontFamily: 'monospace', color: 'error.main', fontWeight: 600 }}>
               {step.actualValue || '5,000,000'}
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      <Box sx={{ mt: 2 }}>
        <Typography variant="body2" color="text.primary">
          <strong>Error Message:</strong> {step.errorDetails || 'Value exceeds configured limit.'}
        </Typography>
      </Box>
    </Box>
  );
};

export default FailureAnalysis;
