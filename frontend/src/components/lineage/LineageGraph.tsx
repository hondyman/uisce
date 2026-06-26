
import React, { useCallback, useEffect, useState } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
    NodeTypes,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { Box, Typography, Chip, Tooltip, IconButton, Paper, CircularProgress } from '@mui/material';
import {
  KeyboardArrowDown,
  KeyboardArrowUp,
  TableChart as TableIcon,
  Business as BOIcon,
  Storage as StorageIcon,
  Functions as AlgoIcon,
   Visibility as ViewIcon
} from '@mui/icons-material';

// --- Custom Node Components ---

const CustomLineageNode = ({ data }: { data: any }) => {
  return (
    <Paper
      elevation={3}
      sx={{
        p: 1,
        minWidth: 180,
        border: '1px solid',
        borderColor: data.selected ? 'primary.main' : 'divider',
        borderRadius: 1,
        bgcolor: 'background.paper',
         transition: 'all 0.2s',
          ...(data.selected && { boxShadow: '0 0 0 2px #2196f3' })
      }}
    >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
            {data.type === 'BO' && <BOIcon color="primary" fontSize="small" />}
            {data.type === 'Table' && <TableIcon color="action" fontSize="small" />}
            {data.type === 'View' && <ViewIcon color="info" fontSize="small" />}
             {/* Fallback icon */}
             {!['BO', 'Table', 'View'].includes(data.type) && <StorageIcon color="disabled" fontSize="small" />}

            <Typography variant="subtitle2" sx={{ fontWeight: 600, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {data.label}
            </Typography>
        </Box>

        {data.metadata && (
             <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                 {data.metadata.env && <Chip label={data.metadata.env} size="small" variant="outlined" sx={{ fontSize: '0.65rem', height: 20 }} />}
             </Box>
        )}

      {/* Expand/Collapse Handle for Columns (Future) */}
      {data.expandable && (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 0.5, borderTop: '1px solid', borderColor: 'divider', pt: 0.5 }}>
            <IconButton size="small" sx={{ p: 0.5 }}>
                 <KeyboardArrowDown fontSize="inherit" />
            </IconButton>
        </Box>
      )}
    </Paper>
  );
};

const nodeTypes: NodeTypes = {
  lineageNode: CustomLineageNode,
};


interface LineageGraphProps {
  nodeId: string;
  depth?: number;
  onNodeClick?: (nodeId: string) => void;
}

export const LineageGraph: React.FC<LineageGraphProps> = ({ nodeId, depth = 3, onNodeClick }) => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetchGraph = async () => {
      setLoading(true);
      try {
        console.log('[LineageGraph] Fetching graph for:', nodeId, 'depth:', depth);
        const response = await fetch(`/api/lineage/node/${encodeURIComponent(nodeId)}/graph?depth=${depth}`);
        console.log('[LineageGraph] Response status:', response.status);
        if (!response.ok) return;

        const data = await response.json();
        console.log('[LineageGraph] Graph data:', data);
        
        // Transform to React Flow
        // Simple auto-layout logic (simplified horizontal layout for now, ideally use dagre)
        // For this iteration we will just list them or use a very basic positioner if dagre isn't available in client
        // But since we removed dagre dependency, we'll implement a simple rank-based layout
        
        const rfNodes: Node[] = [];
        const rfEdges: Edge[] = [];

        // Simple level-based layout
        const levels: Record<string, number> = {};
        const visited = new Set<string>();
        const queue: {id: string, level: number}[] = [{id: nodeId, level: 0}];
        
        // Build adjacency for traversal
        const adj: Record<string, string[]> = {};
         if (data.edges) {
            data.edges.forEach((e: any) => {
                if (!adj[e.from_id]) adj[e.from_id] = [];
                adj[e.from_id].push(e.to_id);
            });
         }

         // BFS for levels
         while (queue.length > 0) {
             const {id, level} = queue.shift()!;
             if (visited.has(id)) continue;
             visited.add(id);
             levels[id] = level;
             
             if (adj[id]) {
                 adj[id].forEach(next => queue.push({id: next, level: level + 1}));
             }
         }

        // Group nodes by level
        const nodesByLevel: Record<number, any[]> = {};
        
        if (data.nodes) {
            data.nodes.forEach((n: any) => {
                // If disconnected/upstream, might not be in bfs from root if direction is mixed.
                // Fallback level 0
                const lvl = levels[n.id] !== undefined ? levels[n.id] : 0;
                 if (!nodesByLevel[lvl]) nodesByLevel[lvl] = [];
                 nodesByLevel[lvl].push(n);
            });

             Object.entries(nodesByLevel).forEach(([lvlStr, levelNodes]) => {
                 const lvl = parseInt(lvlStr);
                 levelNodes.forEach((n, idx) => {
                      rfNodes.push({
                          id: n.id,
                          type: 'lineageNode',
                          position: { x: lvl * 250, y: idx * 100 },
                          data: { 
                              label: n.name || n.id, 
                              type: n.type,
                              metadata: n.metadata,
                              selected: n.id === nodeId
                          },
                      });
                 });
             });
        }

        if (data.edges) {
            data.edges.forEach((e: any) => {
                rfEdges.push({
                    id: `${e.from_id}-${e.to_id}`,
                    source: e.from_id,
                    target: e.to_id,
                    type: 'smoothstep',
                    markerEnd: {
                        type: MarkerType.ArrowClosed,
                    },
                    animated: true,
                    style: { stroke: '#b1b1b7' },
                });
            });
        }

        setNodes(rfNodes);
        setEdges(rfEdges);

      } catch (err) {
        console.error("Failed to fetch lineage:", err);
      } finally {
          setLoading(false);
      }
    };

    fetchGraph();
  }, [nodeId, depth]);

    if (loading) {
        return <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}><CircularProgress /></Box>
    }

  return (
    <Box sx={{ width: '100%', height: '600px', border: '1px solid', borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        fitView
        attributionPosition="bottom-right"
      >
        <Controls />
        <MiniMap />
        <Background gap={12} size={1} />
      </ReactFlow>
    </Box>
  );
};
