import React, { useEffect, useMemo, useRef, useState } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  BackgroundVariant,
  MiniMap,
  Panel,
  useNodesState,
  useEdgesState,
  ReactFlowProvider,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { Box, CircularProgress, Alert } from '@mui/material';

import {
  useViewDefinition,
  useCatalogGraph,
  CatalogResponseNode,
  CatalogResponseEdge,
  CatalogResponseGraph,
  ViewLayoutConfig,
} from '../../api/viewDefinitions';
import { getNodeTypes, CatalogNodeComponent } from './nodeRegistry';
import { applyDagreLayout, NodeDimensionsLookup } from './layout/dagreLayout';
import { applyBfsLevelLayout } from './layout/bfsLevelLayout';
import { CatalogNodeProps } from './CatalogNodeProps';

export interface CatalogGraphProps {
  /**
   * View-fetch mode: pass viewDefinitionId + rootNodeId to have CatalogGraph
   * fetch and normalize the graph itself via GET
   * /api/view-definitions/{id}/graph/{rootNodeId}.
   */
  viewDefinitionId?: string;
  rootNodeId?: string;
  depth?: number;
  /**
   * Pre-fetched mode: pass an already-normalized graph directly (e.g. from
   * GET /api/bo/{boId}/graph converted to this shape) when a viewer's data
   * comes from a source other than the generic lineage traversal. `layout`
   * substitutes for the ViewDefinition's layout config in this mode.
   */
  graphData?: CatalogResponseGraph;
  layout?: ViewLayoutConfig;
  /** Per-view renderer overrides, e.g. ERD's table-with-inline-columns node for 'table'. */
  nodeTypeOverrides?: Record<string, CatalogNodeComponent>;
  getNodeDimensions?: NodeDimensionsLookup;
  onNodeSelect?: (node: CatalogResponseNode) => void;
  edgeColor?: (edge: CatalogResponseEdge) => string;
  showMiniMap?: boolean;
  legend?: React.ReactNode;
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
      // Properties are spread onto the top level too (not just nested under
      // `properties`) so pre-existing node components ported in from older
      // viewers — e.g. BusinessObjectManager/CustomNodes, which read fields
      // straight off `data.X` — keep working unmodified. New components
      // should read from `data.properties.X` per CatalogNodeProps.
      ...n.properties,
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

function toFlowEdges(edges: CatalogResponseEdge[], edgeColor?: (edge: CatalogResponseEdge) => string): Edge[] {
  return edges.map((e, idx) => ({
    id: e.id || `${e.source}-${e.target}-${idx}`,
    source: e.source,
    target: e.target,
    type: 'smoothstep',
    label: e.type,
    labelStyle: { fill: '#555', fontWeight: 600, fontSize: 10 },
    style: { stroke: edgeColor ? edgeColor(e) : '#999', strokeWidth: 2 },
  }));
}

// Hashes full node/edge content, not just ids — a viewer like ImpactGraph
// re-renders the *same* graph shape with different per-node properties (e.g.
// highlightedNodeIds changing which nodes are emphasized), and that must
// still trigger a relayout/re-render. Hashing only ids would silently ignore
// such updates since the id set never changes.
function graphSignatureOf(graphData: CatalogResponseGraph | undefined | null): string | null {
  if (!graphData) return null;
  try {
    return JSON.stringify(graphData);
  } catch {
    return `${graphData.nodes.map((n) => n.id).join(',')}|${graphData.edges
      .map((e) => e.id || `${e.source}-${e.target}`)
      .join(',')}`;
  }
}

// Generic graph renderer shared by every catalog graph viewer. Either fetches
// and normalizes a graph itself (view-fetch mode, driven by a ViewDefinition)
// or renders a graph a caller already fetched from elsewhere (pre-fetched
// mode, e.g. a viewer built on /api/bo/{boId}/graph). Either way, node
// rendering dispatches through nodeRegistry and layout runs through the
// shared dagre/bfs-level strategies — no viewer hand-rolls either again.
function CatalogGraphInner({
  viewDefinitionId,
  rootNodeId,
  depth,
  graphData: providedGraphData,
  layout: providedLayout,
  nodeTypeOverrides,
  getNodeDimensions,
  onNodeSelect,
  edgeColor,
  showMiniMap,
  legend,
}: CatalogGraphProps) {
  const [expandedClusters, setExpandedClusters] = useState<string[]>([]);
  const fetchMode = !providedGraphData;

  const { data: viewDefinition } = useViewDefinition(fetchMode ? viewDefinitionId : undefined);
  const { data: fetchedGraphData, isLoading, error } = useCatalogGraph(
    fetchMode ? viewDefinitionId : undefined,
    fetchMode ? rootNodeId : undefined,
    depth,
    expandedClusters
  );

  const graphData = providedGraphData || fetchedGraphData;
  const layoutConfig = providedLayout || viewDefinition?.config?.layout;

  const [nodes, setNodes, onNodesChange] = useNodesState<CatalogNodeProps>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const lastAppliedSignatureRef = useRef<string | null>(null);

  const handleExpandCluster = (clusterId: string) => {
    setExpandedClusters((prev) => (prev.includes(clusterId) ? prev : [...prev, clusterId]));
  };

  // graphData/layoutConfig come from data-fetching hooks (or a caller prop)
  // whose reference stability isn't guaranteed (react-query keeps it stable in
  // practice, but a badly memoized selector — or a naive test mock — can
  // return a fresh object every render). Re-laying-out on every render would
  // then set state, trigger a re-render, see "new" deps, and loop forever.
  // Gate on a content signature instead of relying on reference identity.
  const graphSignature = graphSignatureOf(graphData);
  const layoutSignature = `${layoutConfig?.algorithm || ''}:${layoutConfig?.direction || ''}`;

  useEffect(() => {
    if (!graphData || graphSignature === null) return;

    const signature = `${graphSignature}::${layoutSignature}`;
    if (lastAppliedSignatureRef.current === signature) return;
    lastAppliedSignatureRef.current = signature;

    const flowNodes = toFlowNodes(graphData.nodes, handleExpandCluster);
    const flowEdges = toFlowEdges(graphData.edges, edgeColor);

    const algorithm = layoutConfig?.algorithm || 'dagre';
    const direction = layoutConfig?.direction;

    const layoutedNodes =
      algorithm === 'bfs-level' && rootNodeId
        ? applyBfsLevelLayout(flowNodes, flowEdges, { rootId: rootNodeId })
        : applyDagreLayout(flowNodes, flowEdges, {
            direction: direction === 'TB' ? 'TB' : 'LR',
            getDimensions: getNodeDimensions,
          });

    setNodes(layoutedNodes);
    setEdges(flowEdges);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [graphSignature, layoutSignature]);

  const nodeTypes = useMemo(() => getNodeTypes(nodeTypeOverrides), [nodeTypeOverrides]);

  if (fetchMode && isLoading && !graphData) {
    return (
      <Box display="flex" alignItems="center" justifyContent="center" height="100%">
        <CircularProgress />
      </Box>
    );
  }

  if (fetchMode && error) {
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
        {showMiniMap && <MiniMap nodeStrokeWidth={3} zoomable pannable style={{ backgroundColor: '#f5f5f5' }} />}
        {legend && <Panel position="top-right">{legend}</Panel>}
      </ReactFlow>
    </Box>
  );
}

export const CatalogGraph: React.FC<CatalogGraphProps> = (props) => (
  <ReactFlowProvider>
    <CatalogGraphInner {...props} />
  </ReactFlowProvider>
);
