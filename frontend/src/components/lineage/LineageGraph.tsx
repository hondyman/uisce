import React, { useEffect, useState } from 'react';
import { Handle, Position } from 'reactflow';
import { Box, Typography, Chip, Paper, IconButton, CircularProgress } from '@mui/material';
import {
  KeyboardArrowDown,
  TableChart as TableIcon,
  Business as BOIcon,
  Storage as StorageIcon,
  Visibility as ViewIcon,
} from '@mui/icons-material';

import { CatalogGraph } from '../graph/CatalogGraph';
import { CatalogResponseGraph } from '../../api/viewDefinitions';
import { CatalogNodeProps } from '../graph/CatalogNodeProps';

// --- Custom Node Component ---
// One component handling every lineage node type via internal branching on
// data.type, matching the original design (rather than per-type dispatch).

const CustomLineageNode: React.FC<{ data: CatalogNodeProps }> = ({ data }) => {
  return (
    <Paper
      elevation={3}
      sx={{
        p: 1.5,
        minWidth: 200,
        border: '1px solid',
        borderColor: data.properties?.selected ? 'primary.main' : 'divider',
        borderRadius: 2,
        bgcolor: 'background.paper',
        position: 'relative',
        transition: 'all 0.2s',
        ...(data.properties?.selected && { boxShadow: '0 0 0 2px #2196f3' }),
      }}
    >
      <Handle type="target" position={Position.Left} style={{ background: '#777', width: 8, height: 8 }} />

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
        {data.type === 'BO' && <BOIcon color="primary" fontSize="small" />}
        {data.type === 'Table' && <TableIcon color="action" fontSize="small" />}
        {data.type === 'View' && <ViewIcon color="info" fontSize="small" />}
        {!['BO', 'Table', 'View'].includes(data.type) && <StorageIcon color="disabled" fontSize="small" />}

        <Typography variant="subtitle2" sx={{ fontWeight: 700, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {data.label}
        </Typography>
      </Box>

      <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', mt: 1 }}>
        <Chip
          label={data.type || 'UNKNOWN'}
          size="small"
          color={data.type === 'BO' ? 'primary' : data.type === 'Table' ? 'default' : 'secondary'}
          sx={{ fontSize: '0.65rem', height: 18, fontWeight: 'bold' }}
        />
        {data.properties?.env && (
          <Chip label={data.properties.env} size="small" variant="outlined" sx={{ fontSize: '0.65rem', height: 18 }} />
        )}
      </Box>

      {data.properties?.expandable && (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 0.5, borderTop: '1px solid', borderColor: 'divider', pt: 0.5 }}>
          <IconButton size="small" sx={{ p: 0.5 }}>
            <KeyboardArrowDown fontSize="inherit" />
          </IconButton>
        </Box>
      )}

      <Handle type="source" position={Position.Right} style={{ background: '#777', width: 8, height: 8 }} />
    </Paper>
  );
};

// LineageNodeType values from backend/internal/lineage/lineage_types.go, all
// dispatched to the one CustomLineageNode component above.
const LINEAGE_NODE_TYPES = [
  'bo', 'bo_field', 'preagg', 'table', 'column', 'entitlement', 'aso_opt', 'tenant', 'changeset', 'page', 'api_endpoint',
];
const nodeTypeOverrides = Object.fromEntries(LINEAGE_NODE_TYPES.map((t) => [t, CustomLineageNode]));

interface LineageGraphProps {
  nodeId: string;
  depth?: number;
  onNodeClick?: (nodeId: string) => void;
}

// Ported onto the shared CatalogGraph renderer. This viewer hits the legacy
// GET /api/lineage/node/{id}/graph endpoint directly (not a ViewDefinition),
// whose response is already the same normalized graphview.ResponseGraph
// shape CatalogGraph expects — so pre-fetched mode applies with only field
// pass-through, no reshaping. Layout stays bfs-level, exactly as before
// (that choice predates this port and is preserved, not re-litigated here).
export const LineageGraph: React.FC<LineageGraphProps> = ({ nodeId, depth = 3 }) => {
  const [graphData, setGraphData] = useState<CatalogResponseGraph | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const fetchGraph = async () => {
      setLoading(true);
      try {
        const response = await fetch(`/api/lineage/node/${encodeURIComponent(nodeId)}/graph?depth=${depth}`);
        if (!response.ok) return;

        const data: CatalogResponseGraph = await response.json();
        if (cancelled) return;

        setGraphData({
          nodes: (data.nodes || []).map((n) => ({
            ...n,
            properties: { ...n.properties, selected: n.id === nodeId },
          })),
          edges: data.edges || [],
        });
      } catch (err) {
        console.error('Failed to fetch lineage:', err);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    fetchGraph();
    return () => {
      cancelled = true;
    };
  }, [nodeId, depth]);

  if (loading && !graphData) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ width: '100%', height: '600px', border: '1px solid', borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
      <CatalogGraph
        graphData={graphData || { nodes: [], edges: [] }}
        layout={{ algorithm: 'bfs-level' }}
        rootNodeId={nodeId}
        nodeTypeOverrides={nodeTypeOverrides}
        showMiniMap
      />
    </Box>
  );
};
