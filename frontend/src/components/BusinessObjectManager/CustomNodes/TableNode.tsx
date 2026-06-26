import React from 'react';
import { Handle, Position } from 'reactflow';
import { Box, Typography } from '@mui/material';
import { Storage as TableIcon } from '@mui/icons-material';

export const TableNode: React.FC<{ data: any }> = ({ data }) => {
  return (
    <Box
      sx={{
        padding: 1.25,
        border: '2px solid',
        borderColor: 'grey.500',
        borderRadius: 1.5,
        background: '#fafafa',
        minWidth: 160,
        boxShadow: 1,
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: '#757575' }} />

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <TableIcon sx={{ fontSize: 18, color: 'grey.600' }} />
        <Box>
          {data.schema && (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
              {data.schema}
            </Typography>
          )}
          <Typography variant="body2" fontWeight="medium" color="grey.800">
            {data.tableName || data.label}
          </Typography>
        </Box>
      </Box>

      <Handle type="source" position={Position.Bottom} style={{ background: '#757575' }} />
    </Box>
  );
};
