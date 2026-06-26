import React, { useEffect, useState, useMemo, useCallback } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
  Panel,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { Box, CircularProgress, IconButton, Tooltip } from '@mui/material';
import { createPortal } from 'react-dom';
import { impactApi } from '../api/impactApi';
import { NodeType, ImpactGraphData } from '../types';
import dagre from 'dagre';
import FullscreenIcon from '@mui/icons-material/Fullscreen';
import FullscreenExitIcon from '@mui/icons-material/FullscreenExit';
import MapIcon from '@mui/icons-material/Map';

type DirectionMode = 'upstream' | 'downstream' | 'both';

interface ImpactGraphProps {
  nodeType: NodeType;
  nodeId: string;
  highlightedNodeIds?: string[];
  directionMode?: DirectionMode;
  onStatsUpdate?: (stats: { upstreamCount: number; downstreamCount: number; totalCount: number }) => void;
  useLineageAPI?: boolean; // If true, use /lineage/node/{id}/graph instead of /impact/graph
}

// Dynamic color mapping based on node type (consistent with other parts of the app)
const getNodeTypeColor = (nodeType: string): { bg: string; border: string; text: string } => {
  // Normalize type - handle both uppercase AGE format and lowercase underscore format
  const normalizedType = nodeType.toLowerCase().replace(/\s+/g, '_');
  
  const colorMap: Record<string, { bg: string; border: string; text: string }> = {
    'business_object': { bg: '#DBEAFE', border: '#1E40AF', text: '#001F3F' },
    'business_term': { bg: '#DBEAFE', border: '#1E40AF', text: '#001F3F' },
    'semantic_term': { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' },
    'semantic_model': { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' },
    'semantic_view': { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' },
    'semantic_column': { bg: '#FED7AA', border: '#92400E', text: '#3F2305' },
    'database_column': { bg: '#DCFCE7', border: '#15803D', text: '#052E16' },
    'db_column': { bg: '#DCFCE7', border: '#15803D', text: '#052E16' },
    'column': { bg: '#DCFCE7', border: '#15803D', text: '#052E16' },
    'table': { bg: '#F3E8FF', border: '#7E22CE', text: '#3F0F5C' },
    'schema': { bg: '#FCE7F3', border: '#BE185D', text: '#500724' },
    'database': { bg: '#FEE2E2', border: '#DC2626', text: '#4C0519' },
    'bo_field': { bg: '#DBEAFE', border: '#0284C7', text: '#001F3F' },
    'api_endpoint': { bg: '#FED7AA', border: '#D97706', text: '#3F2305' },
    'bi_artifact': { bg: '#FEE2E2', border: '#DC2626', text: '#4C0519' },
    'ai_artifact': { bg: '#FCE7F3', border: '#EC4899', text: '#500724' },
    'access_rule': { bg: '#E2E8F0', border: '#475569', text: '#0F172A' },
  };
  
  return colorMap[normalizedType] || { bg: '#F3F4F6', border: '#9CA3AF', text: '#374151' };
};

const getLayoutedElements = (nodes: Node[], edges: Edge[], direction = 'LR') => {
  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));

  const nodeWidth = 200;
  const nodeHeight = 50;

  dagreGraph.setGraph({ rankdir: direction });

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
  });

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  dagre.layout(dagreGraph);

  nodes.forEach((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);
    node.position = {
      x: nodeWithPosition.x - nodeWidth / 2,
      y: nodeWithPosition.y - nodeHeight / 2,
    };
  });

  return { nodes, edges };
};

export const ImpactGraph: React.FC<ImpactGraphProps> = ({ 
  nodeType, 
  nodeId, 
  highlightedNodeIds = [],
  directionMode = 'both',
  onStatsUpdate,
  useLineageAPI = false
}) => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [allNodes, setAllNodes] = useState<Node[]>([]);
  const [allEdges, setAllEdges] = useState<Edge[]>([]);
  const [loading, setLoading] = useState(false);
  const [isFullScreen, setIsFullScreen] = useState(false);
  const [showMiniMap, setShowMiniMap] = useState(true);

  // Update node styling when highlighting changes
  useEffect(() => {
    setNodes((nds) =>
      nds.map((node) => {
        const isHighlighted = highlightedNodeIds.includes(node.id);
        const type = node.data?.type || '';
        const colors = getNodeTypeColor(type);
        
        return {
          ...node,
          style: {
            ...node.style,
            backgroundColor: node.id === nodeId ? colors.bg : '#fff',
            border: isHighlighted ? `3px solid ${colors.border}` : (node.id === nodeId ? `2px solid ${colors.border}` : `1px solid ${colors.border}`),
            boxShadow: isHighlighted ? `0 0 15px ${colors.border}` : (node.id === nodeId ? `0 0 10px ${colors.border}44` : 'none'),
            transform: isHighlighted ? 'scale(1.05)' : 'scale(1)',
            transition: 'all 0.3s ease',
            color: colors.text,
          },
        };
      })
    );
  }, [highlightedNodeIds, nodeId, setNodes]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      // Use lineage API if specified, otherwise use impact API
      console.log('[ImpactGraph] useLineageAPI:', useLineageAPI, 'nodeId:', nodeId);
      const data: ImpactGraphData = useLineageAPI 
        ? await impactApi.getLineageGraph(nodeId)
        : await impactApi.getGraph(nodeType, nodeId);
      
      console.log('[ImpactGraph] Data received:', data);
      console.log('[ImpactGraph] Nodes count:', data.nodes?.length, 'Edges count:', data.edges?.length);
      
      if (!data.nodes || data.nodes.length === 0) {
        console.warn('[ImpactGraph] No nodes returned from API');
      }
      
      const initialNodes: Node[] = data.nodes.map((n) => {
        const isHighlighted = highlightedNodeIds.includes(n.id);
        const colors = getNodeTypeColor(n.type);
        
        // Extract direction from metadata if available
        let direction = 'unknown';
        if (n.properties && typeof n.properties === 'object') {
          const props = n.properties as Record<string, any>;
          if (props.metadata) {
            try {
              const metadata = typeof props.metadata === 'string' 
                ? JSON.parse(props.metadata) 
                : props.metadata;
              direction = metadata.direction || 'unknown';
            } catch {
              // Ignore parse errors
            }
          }
        }
        
        return {
          id: n.id,
          data: { 
            label: n.label, 
            type: n.type, 
            direction,
            ...n.properties 
          },
          position: { x: 0, y: 0 },
          style: { 
            border: isHighlighted ? `3px solid ${colors.border}` : (n.id === nodeId ? `2px solid ${colors.border}` : `1px solid ${colors.border}`),
            background: n.id === nodeId ? colors.bg : '#fff',
            padding: '10px',
            borderRadius: '8px',
            fontSize: '12px',
            fontWeight: n.id === nodeId ? 'bold' : 'normal',
            width: 200,
            textAlign: 'center',
            boxShadow: isHighlighted ? `0 0 15px ${colors.border}` : (n.id === nodeId ? `0 0 10px ${colors.border}44` : 'none'),
            transition: 'all 0.3s ease',
            color: colors.text,
          }
        };
      });

      const initialEdges: Edge[] = data.edges.map(e => {
        const targetNode = data.nodes.find(n => n.id === e.target);
        const targetColors = targetNode ? getNodeTypeColor(targetNode.type) : { border: '#b1b1b7' };
        
        // Extract direction from edge metadata if available
        let edgeDirection = 'unknown';
        if (e.properties && typeof e.properties === 'object') {
          const props = e.properties as Record<string, any>;
          if (props.metadata) {
            try {
              const metadata = typeof props.metadata === 'string' 
                ? JSON.parse(props.metadata) 
                : props.metadata;
              edgeDirection = metadata.direction || 'unknown';
            } catch {
              // Ignore parse errors
            }
          }
        }
        
        return {
          id: e.id || `${e.source}-${e.target}`,
          source: e.source,
          target: e.target,
          label: e.type,
          type: 'smoothstep',
          animated: false,
          data: { direction: edgeDirection },
          markerEnd: { 
            type: MarkerType.ArrowClosed,
            color: targetColors.border
          },
          style: { 
            stroke: targetColors.border,
            strokeWidth: 2,
          },
        };
      });

      setAllNodes(initialNodes);
      setAllEdges(initialEdges);
    } catch (error) {
      console.error("Failed to fetch impact graph:", error);
    } finally {
      setLoading(false);
    }
  }, [nodeType, nodeId, highlightedNodeIds, useLineageAPI]);

  // Filter nodes and edges based on direction mode
  useEffect(() => {
    if (allNodes.length === 0) return;

    let filteredNodes = allNodes;
    let filteredEdges = allEdges;

    if (directionMode !== 'both') {
      // Filter nodes by direction
      filteredNodes = allNodes.filter(node => {
        if (node.id === nodeId) return true; // Always include root node
        const nodeDirection = node.data?.direction || 'unknown';
        return nodeDirection === directionMode || nodeDirection === 'both' || nodeDirection === 'unknown';
      });

      const filteredNodeIds = new Set(filteredNodes.map(n => n.id));

      // Filter edges to only include those where both source and target are in filtered nodes
      filteredEdges = allEdges.filter(edge => 
        filteredNodeIds.has(edge.source) && filteredNodeIds.has(edge.target)
      );
    }

    // Calculate stats
    const upstreamNodes = allNodes.filter(n => {
      const dir = n.data?.direction || 'unknown';
      return n.id !== nodeId && (dir === 'upstream' || dir === 'both');
    });
    const downstreamNodes = allNodes.filter(n => {
      const dir = n.data?.direction || 'unknown';
      return n.id !== nodeId && (dir === 'downstream' || dir === 'both');
    });

    if (onStatsUpdate) {
      onStatsUpdate({
        upstreamCount: upstreamNodes.length,
        downstreamCount: downstreamNodes.length,
        totalCount: allNodes.length - 1 // Exclude root
      });
    }

    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
      filteredNodes,
      filteredEdges
    );

    setNodes([...layoutedNodes]);
    setEdges([...layoutedEdges]);
  }, [allNodes, allEdges, directionMode, nodeId, setNodes, setEdges, onStatsUpdate]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const toggleFullScreen = () => setIsFullScreen(!isFullScreen);

  const graphContent = (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      fitView
      maxZoom={2}
      minZoom={0.1}
    >
      <Background color="#aaa" gap={20} />
      <Controls />
      {showMiniMap && <MiniMap nodeStrokeWidth={3} zoomable pannable />}
      
      <Panel position="top-right" sx={{ display: 'flex', gap: 1 }}>
        <Tooltip title={showMiniMap ? "Hide Map" : "Show Map"}>
          <IconButton onClick={() => setShowMiniMap(!showMiniMap)} size="small" sx={{ bgcolor: 'white', '&:hover': { bgcolor: '#f0f0f0' }, boxShadow: 2 }}>
            <MapIcon fontSize="small" color={showMiniMap ? "primary" : "inherit"} />
          </IconButton>
        </Tooltip>
        <Tooltip title={isFullScreen ? "Exit Fullscreen" : "Fullscreen"}>
          <IconButton onClick={toggleFullScreen} size="small" sx={{ bgcolor: 'white', '&:hover': { bgcolor: '#f0f0f0' }, boxShadow: 2 }}>
            {isFullScreen ? <FullscreenExitIcon fontSize="small" /> : <FullscreenIcon fontSize="small" />}
          </IconButton>
        </Tooltip>
      </Panel>
    </ReactFlow>
  );

  if (loading) return <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', minHeight: 400 }}><CircularProgress /></Box>;

  if (isFullScreen) {
    return createPortal(
      <div className="impact-fullscreen-overlay">
        {graphContent}
      </div>,
      document.body
    );
  }

  return (
    <Box sx={{ width: '100%', height: '100%', minHeight: 600 }}>
      {graphContent}
    </Box>
  );
};
