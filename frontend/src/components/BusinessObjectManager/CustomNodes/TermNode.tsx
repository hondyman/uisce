import React from 'react';
import { Handle, Position } from 'reactflow';
import { Box, Typography, Chip } from '@mui/material';
import {
  Calculate as CalcIcon,
  DataObject as DimensionIcon,
  Functions as MeasureIcon,
  Key as KeyIcon,
} from '@mui/icons-material';

export const TermNode: React.FC<{ data: any }> = ({ data }) => {
  const getIcon = () => {
    if (data.termType === 'calculation') return <CalcIcon color="secondary" />;
    if (data.termType === 'measure') return <MeasureIcon color="primary" />;
    return <DimensionIcon color="action" />;
  };

  const getBorderColor = () => {
    if (data.termType === 'calculation') return 'secondary.main';
    if (data.termType === 'measure') return 'primary.main';
    return 'grey.400';
  };

  return (
    <Box
      sx={{
        padding: 1.5,
        border: '2px solid',
        borderColor: getBorderColor(),
        borderRadius: 2,
        background: 'white',
        minWidth: 180,
        boxShadow: 2,
        '&:hover': {
          boxShadow: 4,
          transform: 'translateY(-2px)',
          transition: 'all 0.2s',
        },
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: '#666' }} />

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
        {getIcon()}
        <Typography variant="subtitle2" fontWeight="bold" noWrap>
          {data.termName || data.label}
        </Typography>
      </Box>

      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
        {data.dataType || 'unknown'}
      </Typography>

      <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
        {data.isKey && (
          <Chip
            icon={<KeyIcon fontSize="small" />}
            label="PK"
            size="small"
            color="primary"
            sx={{ height: 20, fontSize: '0.65rem' }}
          />
        )}
        {data.isForeignKey && (
          <Chip
            label="FK"
            size="small"
            color="secondary"
            sx={{ height: 20, fontSize: '0.65rem' }}
          />
        )}
        {data.aggregation && (
          <Chip
            label={data.aggregation}
            size="small"
            variant="outlined"
            sx={{ height: 20, fontSize: '0.65rem' }}
          />
        )}
      </Box>

      <Handle type="source" position={Position.Bottom} style={{ background: '#666' }} />
    </Box>
  );
};
