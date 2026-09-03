import React, { useEffect, useMemo, useState } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  ReactFlowProvider,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { Box, CircularProgress, Alert } from '@mui/material';

import { useViewDefinition, useCatalogGraph, CatalogResponseNode, CatalogResponseEdge } from '../../api/viewDefinitions';
import { getNodeTypes, CatalogNodeComponent } from './nodeRegistry';
import { applyDagreLayout, NodeDimensionsLookup } from './layout/dagreLayout';
import { applyBfsLevelLayout } from './layout/bfsLevelLayout';
import { CatalogNodeProps } from './CatalogNodeProps';

export interface CatalogGraphProps {
  viewDefinitionId: string;
  rootNodeId: string;
  depth?: number;
  /** Per-view renderer overrides, e.g. ERD's table-with-inline-columns node for 'table'. */
  nodeTypeOverrides?: Record<string, CatalogNodeComponent>;
  getNodeDimensions?: NodeDimensionsLookup;
  onNodeSelect?: (node: CatalogResponseNode) => void;
}

function toFlowNodes(
  nodes: CatalogResponseNode[],
  onExpandCluster: (clusterId: string) => void
): Node<CatalogNodeProps>[] {
  return nodes.map((n) => ({
    id: n.id,
    type: n.isCluster ? 'cluster' : n.type,
    position: { x: 0, y: 0 },
    data: {
      id: n.id,
      type: n.type,
      label: n.label,
      properties: n.properties || {},
      isCluster: n.isCluster,
      memberIds: n.memberIds,
      onExpandCluster,
    },
  }));
}

function toFlowEdges(edges: CatalogResponseEdge[]): Edge[] {
  return edges.map((e, idx) => ({
    id: e.id || `${e.source}-${e.target}-${idx}`,
    source: e.source,
    target: e.target,
    type: 'smoothstep',
    label: e.type,
    labelStyle: { fill: '#555', fontWeight: 600, fontSize: 10 },
    style: { stroke: '#999', strokeWidth: 2 },
  }));
}

// Generic, ViewDefinition-driven graph renderer. Fetches the normalized graph
// from GET /api/view-definitions/{id}/graph/{rootNodeId}, dispatches node
// rendering through nodeRegistry, and lays out via the algorithm the view
// config picks (dagre by default; bfs-level as an explicit per-view choice).
function CatalogGraphInner({
  viewDefinitionId,
  rootNodeId,
  depth,
  nodeTypeOverrides,
  getNodeDimensions,
  onNodeSelect,
}: CatalogGraphProps) {
  const [expandedClusters, setExpandedClusters] = useState<string[]>([]);
  const { data: viewDefinition } = useViewDefinition(viewDefinitionId);
  const { data: graphData, isLoading, error } = useCatalogGraph(
    viewDefinitionId,
    rootNodeId,
    depth,
    expandedClusters
  );

  const [nodes, setNodes, onNodesChange] = useNodesState<CatalogNodeProps>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  const handleExpandCluster = (clusterId: string) => {
    setExpandedClusters((prev) => (prev.includes(clusterId) ? prev : [...prev, clusterId]));
  };

  useEffect(() => {
    if (!graphData) return;

    const flowNodes = toFlowNodes(graphData.nodes, handleExpandCluster);
    const flowEdges = toFlowEdges(graphData.edges);

    const algorithm = viewDefinition?.config?.layout?.algorithm || 'dagre';
    const direction = viewDefinition?.config?.layout?.direction;

    const layoutedNodes =
      algorithm === 'bfs-level'
        ? applyBfsLevelLayout(flowNodes, flowEdges, { rootId: rootNodeId })
        : applyDagreLayout(flowNodes, flowEdges, {
            direction: direction === 'TB' ? 'TB' : 'LR',
            getDimensions: getNodeDimensions,
          });

    setNodes(layoutedNodes);
    setEdges(flowEdges);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [graphData, viewDefinition]);

  const nodeTypes = useMemo(() => getNodeTypes(nodeTypeOverrides), [nodeTypeOverrides]);

  if (isLoading && !graphData) {
    return (
      <Box display="flex" alignItems="center" justifyContent="center" height="100%">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box p={2}>
        <Alert severity="error">{error instanceof Error ? error.message : 'Failed to load graph'}</Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ width: '100%', height: '100%' }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        onNodeClick={(_, node) => {
          if (!onNodeSelect || !graphData) return;
          const original = graphData.nodes.find((n) => n.id === node.id);
          if (original) onNodeSelect(original);
        }}
        fitView
      >
        <Controls />
        <Background variant={BackgroundVariant.Dots} />
      </ReactFlow>
    </Box>
  );
}

export const CatalogGraph: React.FC<CatalogGraphProps> = (props) => (
  <ReactFlowProvider>
    <CatalogGraphInner {...props} />
  </ReactFlowProvider>
);
