import React from 'react';
import { Handle, Position } from 'reactflow';
import { Box, Typography, Chip } from '@mui/material';
import { BusinessCenter as BOIcon } from '@mui/icons-material';

export const BONode: React.FC<{ data: any }> = ({ data }) => {
  const isRelated = data.relationshipType !== undefined;

  return (
    <Box
      sx={{
        padding: 2.5,
        border: '3px solid',
        borderColor: isRelated ? 'secondary.main' : 'primary.main',
        borderRadius: 3,
        background: isRelated
          ? 'linear-gradient(135deg, #f5f5f5 0%, #e3f2fd 100%)'
          : 'linear-gradient(135deg, #ffffff 0%, #e3f2fd 100%)',
        minWidth: 220,
        boxShadow: 4,
        position: 'relative',
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: '#1976d2' }} />

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1.5 }}>
        <BOIcon color="primary" sx={{ fontSize: 28 }} />
        <Box>
          <Typography variant="subtitle1" fontWeight="bold" color="primary.dark">
            {data.name || data.label}
          </Typography>
          {isRelated && (
            <Chip
              label={data.relationshipType}
              size="small"
              color="secondary"
              sx={{ mt: 0.5, fontSize: '0.7rem' }}
            />
          )}
        </Box>
      </Box>

      {data.description && (
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
          {data.description.length > 60
            ? `${data.description.substring(0, 60)}...`
            : data.description}
        </Typography>
      )}

      {data.termCount !== undefined && (
        <Box sx={{ display: 'flex', gap: 1, mt: 1 }}>
          <Chip label={`${data.termCount} terms`} size="small" variant="outlined" />
          {data.relatedBOCount > 0 && (
            <Chip label={`${data.relatedBOCount} related`} size="small" variant="outlined" />
          )}
        </Box>
      )}

      <Handle type="source" position={Position.Bottom} style={{ background: '#1976d2' }} />
    </Box>
  );
};
