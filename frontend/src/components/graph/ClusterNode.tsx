import React from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Box, Typography, Stack, Chip, IconButton } from '@mui/material';
import { UnfoldMore as ExpandClusterIcon, Layers as ClusterIcon } from '@mui/icons-material';
import { CatalogNodeProps } from './CatalogNodeProps';

// Generic collapsible cluster node, replacing/generalizing:
//   - BONode.tsx's subtype-grouped, collapsible term list
//   - glossary LineageGraph.tsx's DatasourceNode / GroupedBONode
// A cluster always renders collapsed (server already folded its members away);
// expanding asks CatalogGraph to refetch with this cluster's id in
// ?expandClusters= so the backend returns its real member nodes instead.
export const ClusterNode: React.FC<NodeProps<CatalogNodeProps>> = ({ id, data }) => {
  const memberCount = data.properties?.memberCount ?? data.memberIds?.length ?? 0;
  const clusterLabel = data.properties?.clusterLabel || data.label || 'Group';
  const childNodeType = data.properties?.childNodeType;

  return (
    <Box
      sx={{
        padding: 1.5,
        border: '2px dashed',
        borderColor: 'grey.500',
        borderRadius: 2,
        background: 'linear-gradient(135deg, #fafafa 0%, #eeeeee 100%)',
        minWidth: 180,
        boxShadow: 1,
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: '#757575' }} />

      <Stack direction="row" alignItems="center" spacing={1}>
        <ClusterIcon sx={{ fontSize: 20, color: 'text.secondary' }} />
        <Box sx={{ flexGrow: 1 }}>
          <Typography variant="subtitle2" fontWeight="bold" color="text.secondary">
            {clusterLabel}
          </Typography>
          {childNodeType && (
            <Typography variant="caption" color="text.disabled">
              {childNodeType}
            </Typography>
          )}
        </Box>
        <IconButton
          size="small"
          onClick={() => data.onExpandCluster?.(id)}
          title={`Expand ${memberCount} ${childNodeType || 'items'}`}
        >
          <ExpandClusterIcon fontSize="small" />
        </IconButton>
      </Stack>

      <Chip
        label={`${memberCount} items`}
        size="small"
        variant="outlined"
        sx={{ mt: 1, height: 20, fontSize: '0.65rem' }}
      />

      <Handle type="source" position={Position.Bottom} style={{ background: '#757575' }} />
    </Box>
  );
};
