import React from 'react';
import { Box, Paper, Typography, Stack, Chip } from '@mui/material';
import {
  BusinessCenter,
  DataObject,
  Calculate,
  Storage,
  ViewColumn,
} from '@mui/icons-material';

export const GraphLegend: React.FC = () => {
  const legendItems = [
    { icon: <BusinessCenter fontSize="small" />, label: 'Business Object', color: '#1976d2' },
    { icon: <DataObject fontSize="small" />, label: 'Term', color: '#666' },
    { icon: <Calculate fontSize="small" />, label: 'Calculation', color: '#9c27b0' },
    { icon: <Storage fontSize="small" />, label: 'Table', color: '#757575' },
    { icon: <ViewColumn fontSize="small" />, label: 'Column', color: '#999' },
  ];

  const edgeTypes = [
    { label: 'Contains', color: '#1976d2', style: 'solid' },
    { label: 'Maps To', color: '#2e7d32', style: 'solid' },
    { label: 'Uses', color: '#9c27b0', style: 'dashed' },
    { label: 'Relates To', color: '#ed6c02', style: 'bold' },
  ];

  return (
    <Paper
      elevation={3}
      sx={{
        p: 2,
        bgcolor: 'white',
        minWidth: 200,
        maxWidth: 250,
      }}
    >
      <Typography variant="subtitle2" fontWeight="bold" gutterBottom>
        Graph Legend
      </Typography>

      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
        Nodes
      </Typography>
      <Stack spacing={0.75} sx={{ mb: 2 }}>
        {legendItems.map((item) => (
          <Box key={item.label} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ color: item.color }}>{item.icon}</Box>
            <Typography variant="caption">{item.label}</Typography>
          </Box>
        ))}
      </Stack>

      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
        Edges
      </Typography>
      <Stack spacing={0.75}>
        {edgeTypes.map((edge) => (
          <Box key={edge.label} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box
              sx={{
                width: 24,
                height: 2,
                bgcolor: edge.color,
                borderStyle: edge.style === 'dashed' ? 'dashed' : 'solid',
                borderWidth: edge.style === 'bold' ? 3 : 2,
                borderColor: edge.color,
              }}
            />
            <Typography variant="caption">{edge.label}</Typography>
          </Box>
        ))}
      </Stack>
    </Paper>
  );
};
