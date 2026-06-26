import React from 'react';
import { Handle, Position } from 'reactflow';
import { Box, Typography, Chip } from '@mui/material';
import { Calculate as CalcIcon } from '@mui/icons-material';

export const CalculationNode: React.FC<{ data: any }> = ({ data }) => {
  return (
    <Box
      sx={{
        padding: 1.5,
        border: '2px solid',
        borderColor: 'secondary.main',
        borderRadius: 2,
        background: 'linear-gradient(135deg, #f3e5f5 0%, #ffffff 100%)',
        minWidth: 200,
        boxShadow: 3,
        '&:hover': {
          boxShadow: 5,
          borderColor: 'secondary.dark',
        },
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: '#9c27b0' }} />

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
        <CalcIcon color="secondary" />
        <Typography variant="subtitle2" fontWeight="bold" color="secondary.dark">
          {data.name || data.label}
        </Typography>
      </Box>

      {data.formula && (
        <Box
          sx={{
            p: 0.75,
            bgcolor: 'grey.100',
            borderRadius: 1,
            fontFamily: 'monospace',
            fontSize: '0.7rem',
            mb: 1,
            maxWidth: '100%',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {data.formula.length > 30 ? `${data.formula.substring(0, 30)}...` : data.formula}
        </Box>
      )}

      {data.returnType && (
        <Chip
          label={data.returnType}
          size="small"
          color="secondary"
          variant="outlined"
          sx={{ fontSize: '0.65rem' }}
        />
      )}

      <Handle type="source" position={Position.Bottom} style={{ background: '#9c27b0' }} />
    </Box>
  );
};
