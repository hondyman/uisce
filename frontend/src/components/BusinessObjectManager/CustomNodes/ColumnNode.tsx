import React from 'react';
import { Handle, Position } from 'reactflow';
import { Box, Typography } from '@mui/material';
import { ViewColumn as ColumnIcon } from '@mui/icons-material';

export const ColumnNode: React.FC<{ data: any }> = ({ data }) => {
  return (
    <Box
      sx={{
        padding: 1,
        border: '1px solid',
        borderColor: 'grey.400',
        borderRadius: 1,
        background: 'white',
        minWidth: 140,
        boxShadow: 0.5,
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: '#999' }} />

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
        <ColumnIcon sx={{ fontSize: 16, color: 'grey.500' }} />
        <Box>
          <Typography variant="caption" fontWeight="medium" color="grey.700">
            {data.columnName || data.label}
          </Typography>
          {data.dataType && (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', fontSize: '0.65rem' }}>
              {data.dataType}
            </Typography>
          )}
        </Box>
      </Box>

      <Handle type="source" position={Position.Bottom} style={{ background: '#999' }} />
    </Box>
  );
};
