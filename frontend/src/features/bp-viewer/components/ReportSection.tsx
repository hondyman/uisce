import React from 'react';
import { ReportInfo } from '../types';
import { Box, Typography, Button, Paper } from '@mui/material';
import PictureAsPdfIcon from '@mui/icons-material/PictureAsPdf';

interface ReportSectionProps {
  report?: ReportInfo;
}

export const ReportSection: React.FC<ReportSectionProps> = ({ report }) => {
  if (!report) {
    return (
        <Paper variant="outlined" sx={{ p: 2, textAlign: 'center' }}>
            <Typography color="text.secondary">No report generated yet.</Typography>
        </Paper>
    );
  }

  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Box display="flex" alignItems="center" justifyContent="space-between">
        <Box>
            <Typography variant="subtitle1" fontWeight="bold">
                Model Change Report
            </Typography>
            <Typography variant="body2" color="text.secondary">
                Generated: {new Date(report.generatedAt).toLocaleString()}
            </Typography>
             <Typography variant="caption" color="text.secondary">
                ID: {report.reportId}
            </Typography>
        </Box>
        <Button 
            variant="contained" 
            startIcon={<PictureAsPdfIcon />} 
            href={report.reportUrl} 
            target="_blank"
            size="small"
        >
            View Report
        </Button>
      </Box>
    </Paper>
  );
};
