import React from 'react';
import { Box, Paper, Typography, Divider } from '@mui/material';

interface AIDocumentationViewerProps {
  documentation: string;
}

export const AIDocumentationViewer: React.FC<AIDocumentationViewerProps> = ({ documentation }) => {
  return (
    <Paper elevation={0} sx={{ p: 2, height: '100%', bgcolor: 'background.paper', overflow: 'auto' }}>
      <Typography variant="subtitle2" sx={{ mb: 2 }}>AI Documentation</Typography>
      <Divider sx={{ mb: 2 }} />
      <Typography variant="body2" component="pre" sx={{ whiteSpace: 'pre-wrap' }}>
        {documentation || 'No documentation available'}
      </Typography>
    </Paper>
  );
};
