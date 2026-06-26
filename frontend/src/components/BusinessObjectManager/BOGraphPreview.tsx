import React, { useEffect, useState, useCallback, useMemo } from 'react';
import {
  Box,
  Typography,
  CircularProgress,
  Alert,
  Button,
  Chip,
  Paper,
  FormControl,
  Select,
  MenuItem,
  Stack,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  ZoomIn as ZoomInIcon,
  ZoomOut as ZoomOutIcon,
  FitScreen as FitScreenIcon,
  Download as DownloadIcon,
} from '@mui/icons-material';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
  Position,
  Node,
  Edge,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useTenant } from '../../contexts/TenantContext';

// ============================================================================
// Types
// ============================================================================

interface GraphNode {
  id: string;
  type: 'bo' | 'term' | 'table' | 'column' | 'calculation';
  label: string;
  group?: string;
  metadata?: Record<string, any>;
}

interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: 'HAS_ATTRIBUTE' | 'RELATES_TO' | 'MAPS_TO' | 'DEPENDS_ON';
  label?: string;
}

interface BOGraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
  bo_id: string;
  bo_name: string;
}

interface BOGraphPreviewProps {
  boId: string;
  boName?: string;
}

// ============================================================================
// Node Colors
// ============================================================================

const nodeColors: Record<string, { bg: string; border: string; text: string }> = {
  bo: { bg: '#e3f2fd', border: '#1976d2', text: '#0d47a1' },
  term: { bg: '#e8f5e9', border: '#388e3c', text: '#1b5e20' },
  table: { bg: '#fff3e0', border: '#f57c00', text: '#e65100' },
  column: { bg: '#fce4ec', border: '#c2185b', text: '#880e4f' },
  calculation: { bg: '#f3e5f5', border: '#7b1fa2', text: '#4a148c' },
};

// ============================================================================
// Custom Node Component
// ============================================================================

const CustomNode = ({ data }: { data: any }) => {
  const colors = nodeColors[data.nodeType] || nodeColors.term;
  
  return (
    <Box
      sx={{
        px: 2,
        py: 1,
        borderRadius: 1,
        border: `2px solid ${colors.border}`,
        bgcolor: colors.bg,
        minWidth: 120,
        textAlign: 'center',
      }}
    >
      <Typography variant="caption" sx={{ color: colors.text, fontWeight: 600 }}>
        {data.label}
      </Typography>
      {data.sublabel && (
        <Typography variant="caption" sx={{ display: 'block', color: 'text.secondary', fontSize: 10 }}>
          {data.sublabel}
        </Typography>
      )}
    </Box>
  );
};

const nodeTypes = {
  custom: CustomNode,
};

// ============================================================================
// Component
// ============================================================================

export const BOGraphPreview: React.FC<BOGraphPreviewProps> = ({ boId, boName }) => {
  const { tenant } = useTenant();
  const tenantId = tenant?.id || '';

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [graphData, setGraphData] = useState<BOGraphResponse | null>(null);

  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  const [viewMode, setViewMode] = useState<'full' | 'terms' | 'relationships'>('full');

  // Load graph data
  const loadGraph = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/bo/${boId}/graph?mode=${viewMode}`, {
        headers: { 'X-Tenant-ID': tenantId },
      });

      if (!response.ok) {
        throw new Error('Failed to load graph');
      }

      const data: BOGraphResponse = await response.json();
      setGraphData(data);

      // Convert to React Flow format
      const rfNodes: Node[] = data.nodes.map((n, idx) => ({
        id: n.id,
        type: 'custom',
        position: calculateNodePosition(idx, data.nodes.length, n.type),
        data: {
          label: n.label,
          sublabel: n.group,
          nodeType: n.type,
        },
      }));

      const rfEdges: Edge[] = data.edges.map((e) => ({
        id: e.id,
        source: e.source,
        target: e.target,
        label: e.label || e.type,
        type: 'smoothstep',
        animated: e.type === 'DEPENDS_ON',
        markerEnd: { type: MarkerType.ArrowClosed },
        style: { stroke: getEdgeColor(e.type) },
      }));

      setNodes(rfNodes);
      setEdges(rfEdges);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [boId, tenantId, viewMode]);

  useEffect(() => {
    if (boId && tenantId) {
      loadGraph();
    }
  }, [boId, tenantId, loadGraph]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" action={<Button onClick={loadGraph}>Retry</Button>}>
        {error}
      </Alert>
    );
  }

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Toolbar */}
      <Paper elevation={0} sx={{ p: 1, mb: 1, display: 'flex', alignItems: 'center', gap: 2 }}>
        <Typography variant="subtitle2">
          {boName || graphData?.bo_name || 'Business Object'} Graph
        </Typography>

        <FormControl size="small" sx={{ minWidth: 150 }}>
          <Select
            value={viewMode}
            onChange={(e) => setViewMode(e.target.value as any)}
          >
            <MenuItem value="full">Full Graph</MenuItem>
            <MenuItem value="terms">Terms Only</MenuItem>
            <MenuItem value="relationships">Relationships</MenuItem>
          </Select>
        </FormControl>

        <Box sx={{ flex: 1 }} />

        <Stack direction="row" spacing={0.5}>
          <Chip label={`${nodes.length} nodes`} size="small" variant="outlined" />
          <Chip label={`${edges.length} edges`} size="small" variant="outlined" />
        </Stack>

        <Tooltip title="Refresh">
          <IconButton size="small" onClick={loadGraph}>
            <RefreshIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Paper>

      {/* Graph Canvas */}
      <Box sx={{ flex: 1, border: 1, borderColor: 'divider', borderRadius: 1 }}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodeTypes={nodeTypes}
          fitView
          minZoom={0.1}
          maxZoom={2}
        >
          <Background color="#f0f0f0" gap={16} />
          <Controls />
          <MiniMap
            nodeColor={(node) => nodeColors[node.data?.nodeType]?.border || '#888'}
            maskColor="rgba(0,0,0,0.1)"
          />
        </ReactFlow>
      </Box>

      {/* Legend */}
      <Paper elevation={0} sx={{ p: 1, mt: 1, display: 'flex', gap: 2, flexWrap: 'wrap' }}>
        {Object.entries(nodeColors).map(([type, colors]) => (
          <Stack key={type} direction="row" alignItems="center" spacing={0.5}>
            <Box
              sx={{
                width: 12,
                height: 12,
                borderRadius: 0.5,
                bgcolor: colors.bg,
                border: `2px solid ${colors.border}`,
              }}
            />
            <Typography variant="caption" sx={{ textTransform: 'capitalize' }}>
              {type}
            </Typography>
          </Stack>
        ))}
      </Paper>
    </Box>
  );
};

// ============================================================================
// Helpers
// ============================================================================

function calculateNodePosition(index: number, total: number, type: string): { x: number; y: number } {
  // Simple radial layout
  const radius = 200 + (total > 10 ? total * 10 : 0);
  const angle = (index / total) * 2 * Math.PI;
  
  // Offset by type for grouping
  const typeOffset = { bo: 0, term: 1, table: 2, column: 3, calculation: 4 }[type] || 0;
  
  return {
    x: 400 + Math.cos(angle) * (radius + typeOffset * 50),
    y: 300 + Math.sin(angle) * (radius + typeOffset * 50),
  };
}

function getEdgeColor(type: string): string {
  switch (type) {
    case 'HAS_ATTRIBUTE': return '#388e3c';
    case 'RELATES_TO': return '#1976d2';
    case 'MAPS_TO': return '#f57c00';
    case 'DEPENDS_ON': return '#7b1fa2';
    default: return '#888';
  }
}

export default BOGraphPreview;
