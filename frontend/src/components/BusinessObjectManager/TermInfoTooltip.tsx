import React from 'react';
import { Box, Typography, Stack, Tooltip, IconButton } from '@mui/material';
import { Info as InfoIcon } from '@mui/icons-material';

interface TermInfoTooltipProps {
  term: {
    node_name: string;
    description?: string;
    qualified_path?: string;
    properties?: {
      sql?: string;
      [key: string]: any;
    };
    source_column?: string; // Fallback if properties.sql isn't available
  };
}

export const TermInfoTooltip: React.FC<TermInfoTooltipProps> = ({ term }) => {
  return (
    <Tooltip
      title={
        <Box sx={{ p: 0.5 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>
            {term.node_name}
          </Typography>
          <Typography variant="body2" sx={{ fontSize: '0.75rem', mb: 1, opacity: 0.9 }}>
            {term.description || "No description available."}
          </Typography>
          <Stack spacing={0.5}>
            {term.qualified_path && (
            <Typography variant="caption" display="block" sx={{ opacity: 0.8 }}>
              <strong>Path:</strong> {term.qualified_path}
            </Typography>
            )}
            {(term.properties?.sql || term.source_column) && (
              <Typography variant="caption" display="block" sx={{ opacity: 0.8 }}>
                <strong>Source:</strong> {term.properties?.sql || term.source_column}
              </Typography>
            )}
          </Stack>
        </Box>
      }
      arrow
      placement="top"
    >
      <IconButton 
        size="small" 
        sx={{ 
          p: 0.5, 
          color: 'text.secondary', 
          cursor: 'help',
          '&:hover': { color: 'primary.main' }
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <InfoIcon fontSize="small" sx={{ fontSize: '1rem' }} />
      </IconButton>
    </Tooltip>
  );
};
