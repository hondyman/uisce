import React from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Box, Typography, Stack } from '@mui/material';
import { Extension as DefaultIcon } from '@mui/icons-material';
import { CatalogNodeProps } from './CatalogNodeProps';

// Fallback renderer for any node type without a registered component —
// notably a tenant-custom node type, styled from that type's own
// catalog_node_types.config.color/icon rather than a hardcoded look.
export const DefaultCatalogNode: React.FC<NodeProps<CatalogNodeProps>> = ({ data }) => {
  const color = data.properties?.typeConfig?.color || '#607d8b';

  return (
    <Box
      sx={{
        padding: 1.5,
        border: '2px solid',
        borderColor: color,
        borderRadius: 2,
        background: '#ffffff',
        minWidth: 160,
        boxShadow: 1,
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: color }} />

      <Stack direction="row" alignItems="center" spacing={1}>
        <DefaultIcon sx={{ fontSize: 18, color }} />
        <Box>
          <Typography variant="subtitle2" fontWeight="bold">
            {data.label}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {data.type}
          </Typography>
        </Box>
      </Stack>

      <Handle type="source" position={Position.Bottom} style={{ background: color }} />
    </Box>
  );
};
