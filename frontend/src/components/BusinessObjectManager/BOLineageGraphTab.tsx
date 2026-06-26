import React, { useEffect, useState, useCallback } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  Panel,
  MiniMap,
  BackgroundVariant,
} from 'reactflow';
import dagre from 'dagre';
import 'reactflow/dist/style.css';
import { Box, CircularProgress, Alert, Typography } from '@mui/material';

import { BONode } from './CustomNodes/BONode';
import { TermNode } from './CustomNodes/TermNode';
import { CalculationNode } from './CustomNodes/CalculationNode';
import { TableNode } from './CustomNodes/TableNode';
import { ColumnNode } from './CustomNodes/ColumnNode';
import { NodeDetailDrawer } from './NodeDetailDrawer';
import { GraphLegend } from './GraphLegend';

const nodeTypes = {
  bo: BONode,
  term: TermNode,
  calculation: CalculationNode,
  table: TableNode,
  column: ColumnNode,
  related_bo: BONode, // Reuse BO node with different styling
};

interface BOLineageGraphTabProps {
  boId: string;
}

interface GraphData {
  nodes: any[];
  edges: any[];
}

export const BOLineageGraphTab: React.FC<BOLineageGraphTabProps> = ({ boId }) => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchGraph();
  }, [boId]);

  const fetchGraph = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`/api/bo/${boId}/graph`);
      if (!response.ok) {
        throw new Error(`Failed to fetch graph: ${response.statusText}`);
      }
      const data: GraphData = await response.json();

      // Apply auto-layout
      const layouted = applyDagreLayout(data.nodes, data.edges);

      setNodes(layouted.nodes);
      setEdges(layouted.edges);
    } catch (err) {
      console.error('Failed to fetch graph:', err);
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const applyDagreLayout = (rawNodes: any[], rawEdges: any[]) => {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setDefaultEdgeLabel(() => ({}));
    dagreGraph.setGraph({ rankdir: 'TB', ranksep: 120, nodesep: 100 });

    // Determine node dimensions based on type
    const getNodeDimensions = (type: string) => {
      switch (type) {
        case 'bo':
        case 'related_bo':
          return { width: 250, height: 120 };
        case 'calculation':
          return { width: 220, height: 100 };
        case 'term':
          return { width: 200, height: 90 };
        case 'table':
          return { width: 180, height: 80 };
        case 'column':
          return { width: 160, height: 70 };
        default:
          return { width: 200, height: 100 };
      }
    };

    rawNodes.forEach((node) => {
      const dims = getNodeDimensions(node.type);
      dagreGraph.setNode(node.id, dims);
    });

    rawEdges.forEach((edge) => {
      dagreGraph.setEdge(edge.source, edge.target);
    });

    dagre.layout(dagreGraph);

    const layoutedNodes = rawNodes.map((node) => {
      const nodeWithPosition = dagreGraph.node(node.id);
      const dims = getNodeDimensions(node.type);
      
      return {
        ...node,
        position: {
          x: nodeWithPosition.x - dims.width / 2,
          y: nodeWithPosition.y - dims.height / 2,
        },
        // Add styling based on type
        style: node.type === 'bo' ? { zIndex: 10 } : {},
      };
    });

    // Style edges based on type
    const layoutedEdges = rawEdges.map((edge) => ({
      ...edge,
      type: edge.type === 'uses' ? 'smoothstep' : 'default',
      animated: edge.type === 'uses',
      style: {
        stroke: getEdgeColor(edge.type),
        strokeWidth: edge.type === 'relates_to' ? 3 : 2,
        strokeDasharray: edge.type === 'joins_via' ? '5,5' : undefined,
      },
    }));

    return { nodes: layoutedNodes, edges: layoutedEdges };
  };

  const getEdgeColor = (type: string) => {
    switch (type) {
      case 'contains':
        return '#1976d2'; // Primary blue
      case 'maps_to':
        return '#2e7d32'; // Green
      case 'belongs_to':
        return '#757575'; // Grey
      case 'relates_to':
        return '#ed6c02'; // Orange
      case 'uses':
        return '#9c27b0'; // Purple
      case 'joins_via':
        return '#d32f2f'; // Red
      default:
        return '#666';
    }
  };

  const onNodeClick = useCallback((event: React.MouseEvent, node: Node) => {
    setSelectedNode(node);
  }, []);

  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
  }, []);

  if (loading) {
    return (
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: '600px',
        }}
      >
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">
          <Typography variant="h6">Failed to load graph</Typography>
          <Typography variant="body2">{error}</Typography>
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ width: '100%', height: '800px', position: 'relative' }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        nodeTypes={nodeTypes}
        fitView
        minZoom={0.1}
        maxZoom={2}
      >
        <Controls />
        <Background variant={BackgroundVariant.Dots} gap={16} size={1} />
        <MiniMap
          nodeStrokeWidth={3}
          zoomable
          pannable
          style={{
            backgroundColor: '#f5f5f5',
          }}
        />
        <Panel position="top-right">
          <GraphLegend />
        </Panel>
      </ReactFlow>

      <NodeDetailDrawer
        node={selectedNode}
        open={!!selectedNode}
        onClose={() => setSelectedNode(null)}
        boId={boId}
      />
    </Box>
  );
};
