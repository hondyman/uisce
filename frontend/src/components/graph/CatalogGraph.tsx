import { ComponentType, ReactNode, useMemo, useState, useEffect } from 'react';
import ReactFlow, {
  Background,
  BackgroundVariant,
  MiniMap,
  Panel,
  Node as FlowNode,
  Edge,
  NodeProps,
  NodeTypes,
  ReactFlowInstance,
  Viewport,
  FitViewOptions,
} from 'reactflow';
import 'reactflow/dist/style.css';

import { getNodeTypes } from './nodeRegistry';
import { dagreLayout, GetNodeDimensions, DagreLayoutOptions } from './layout/dagreLayout';
import { bfsLevelLayout, BfsLevelLayoutOptions } from './layout/bfsLevelLayout';

export interface ResponseGraph {
  nodes: FlowNode[];
  edges: Edge[];
}

export type LayoutMode =
  | 'none'
  | 'dagre'
  | 'bfs'
  | { type: 'dagre'; options?: DagreLayoutOptions }
  | { type: 'bfs'; options?: BfsLevelLayoutOptions }
  | ((nodes: FlowNode[], edges: Edge[]) => FlowNode[]);

export interface GroupingRule {
  /** Groups nodes sharing the same key when the group exceeds `threshold`. */
  key: (node: FlowNode) => string;
  threshold: number;
  label?: (key: string, count: number) => string;
}

interface BaseCatalogGraphProps {
  nodeTypeOverrides?: Record<string, ComponentType<NodeProps>>;
  getNodeDimensions?: GetNodeDimensions;
  edgeColor?: (edge: Edge) => string | undefined;
  showMiniMap?: boolean;
  legend?: ReactNode;
  children?: ReactNode;
  layout?: LayoutMode;
  /** Node-grouping rules; pass `[]` (default) to disable clustering entirely. */
  grouping?: GroupingRule[];
  onInit?: (instance: ReactFlowInstance) => void;
  onNodeClick?: (event: React.MouseEvent, node: FlowNode) => void;
  onEdgeClick?: (event: React.MouseEvent, edge: Edge) => void;
  onPaneClick?: () => void;
  onMoveEnd?: (event: unknown, viewport: Viewport) => void;
  defaultViewport?: Viewport;
  fitView?: boolean;
  fitViewOptions?: FitViewOptions;
  minZoom?: number;
  maxZoom?: number;
  nodesDraggable?: boolean;
  className?: string;
}

interface PreFetchedProps extends BaseCatalogGraphProps {
  mode: 'pre-fetched';
  graphData: ResponseGraph;
}

interface ViewFetchProps extends BaseCatalogGraphProps {
  mode: 'view-fetch';
  viewDefinitionId: string;
  rootNodeId: string;
}

export type CatalogGraphProps = PreFetchedProps | ViewFetchProps;

function applyGrouping(nodes: FlowNode[], edges: Edge[], rules: GroupingRule[]): ResponseGraph {
  if (!rules || rules.length === 0) return { nodes, edges };

  let workingNodes = nodes;
  let workingEdges = edges;

  for (const rule of rules) {
    const buckets = new Map<string, FlowNode[]>();
    for (const node of workingNodes) {
      const key = rule.key(node);
      if (!buckets.has(key)) buckets.set(key, []);
      buckets.get(key)!.push(node);
    }

    const keep: FlowNode[] = [];
    const collapsedOf = new Map<string, string>();

    for (const [key, group] of buckets.entries()) {
      if (group.length <= rule.threshold) {
        keep.push(...group);
        continue;
      }
      const clusterId = `cluster-${key}`;
      const label = rule.label ? rule.label(key, group.length) : `${key} (${group.length})`;
      const avgX = group.reduce((s, n) => s + (n.position?.x || 0), 0) / group.length;
      const avgY = group.reduce((s, n) => s + (n.position?.y || 0), 0) / group.length;
      keep.push({
        id: clusterId,
        type: 'cluster',
        position: { x: avgX, y: avgY },
        data: { label, count: group.length, memberIds: group.map((n) => n.id) },
      });
      for (const n of group) collapsedOf.set(n.id, clusterId);
    }

    const remappedEdges = workingEdges
      .map((edge) => ({
        ...edge,
        source: collapsedOf.get(edge.source) || edge.source,
        target: collapsedOf.get(edge.target) || edge.target,
      }))
      .filter((edge) => edge.source !== edge.target);

    workingNodes = keep;
    workingEdges = remappedEdges;
  }

  return { nodes: workingNodes, edges: workingEdges };
}

function resolveLayout(
  layout: LayoutMode | undefined,
  nodes: FlowNode[],
  edges: Edge[],
  getNodeDimensions?: GetNodeDimensions
): FlowNode[] {
  if (!layout || layout === 'none') return nodes;
  if (typeof layout === 'function') return layout(nodes, edges);
  if (layout === 'dagre') return dagreLayout(nodes, edges, { getNodeDimensions });
  if (layout === 'bfs') return bfsLevelLayout(nodes, edges, { getNodeDimensions });
  if (layout.type === 'dagre') return dagreLayout(nodes, edges, { getNodeDimensions, ...layout.options });
  if (layout.type === 'bfs') return bfsLevelLayout(nodes, edges, { getNodeDimensions, ...layout.options });
  return nodes;
}

/**
 * Shared ReactFlow renderer for catalog/lineage-style graphs.
 *
 * Two data modes:
 *  - "pre-fetched": caller supplies an already-normalized `graphData` (and
 *    typically its own layout, since traversal-specific viewers often need
 *    layout logic CatalogGraph doesn't know about).
 *  - "view-fetch": caller supplies a `viewDefinitionId` + `rootNodeId` and
 *    CatalogGraph fetches/normalizes the graph itself. Not yet wired to a
 *    backend endpoint — reserved for a future ViewDefinition-backed viewer.
 */
export default function CatalogGraph(props: CatalogGraphProps) {
  const {
    nodeTypeOverrides,
    getNodeDimensions,
    edgeColor,
    showMiniMap = false,
    legend,
    children,
    layout = 'none',
    grouping = [],
    onInit,
    onNodeClick,
    onEdgeClick,
    onPaneClick,
    onMoveEnd,
    defaultViewport,
    fitView = true,
    fitViewOptions,
    minZoom = 0.1,
    maxZoom = 2,
    nodesDraggable = true,
    className,
  } = props;

  const [fetchedGraph, setFetchedGraph] = useState<ResponseGraph | null>(null);
  const [fetchError, setFetchError] = useState<string | null>(null);

  useEffect(() => {
    if (props.mode !== 'view-fetch') return;
    setFetchError(
      'CatalogGraph view-fetch mode has no backend endpoint wired up yet; pass mode="pre-fetched" with graphData instead.'
    );
  }, [props.mode]);

  const rawGraph: ResponseGraph = useMemo(() => {
    if (props.mode === 'pre-fetched') return props.graphData;
    return fetchedGraph || { nodes: [], edges: [] };
  }, [props, fetchedGraph]);

  const nodeTypes: NodeTypes = useMemo(() => getNodeTypes(nodeTypeOverrides), [nodeTypeOverrides]);

  const grouped = useMemo(
    () => applyGrouping(rawGraph.nodes, rawGraph.edges, grouping),
    [rawGraph, grouping]
  );

  const layoutedNodes = useMemo(
    () => resolveLayout(layout, grouped.nodes, grouped.edges, getNodeDimensions),
    [layout, grouped, getNodeDimensions]
  );

  const styledEdges: Edge[] = useMemo(() => {
    if (!edgeColor) return grouped.edges;
    return grouped.edges.map((edge) => {
      const color = edgeColor(edge);
      if (!color) return edge;
      return {
        ...edge,
        style: { ...edge.style, stroke: color },
      };
    });
  }, [grouped.edges, edgeColor]);

  if (fetchError) {
    return <div style={{ padding: 24, color: '#b91c1c' }}>{fetchError}</div>;
  }

  return (
    <ReactFlow
      className={className}
      nodes={layoutedNodes}
      edges={styledEdges}
      nodeTypes={nodeTypes}
      onInit={onInit}
      onNodeClick={onNodeClick}
      onEdgeClick={onEdgeClick}
      onPaneClick={onPaneClick}
      onMoveEnd={onMoveEnd}
      nodesDraggable={nodesDraggable}
      nodesConnectable={false}
      elementsSelectable
      panOnDrag
      fitView={fitView && layoutedNodes.length > 0}
      fitViewOptions={fitViewOptions}
      defaultViewport={defaultViewport}
      minZoom={minZoom}
      maxZoom={maxZoom}
      attributionPosition="bottom-left"
      proOptions={{ hideAttribution: true }}
    >
      <Background color="#cbd5e1" gap={24} size={2} variant={BackgroundVariant.Dots} />
      {showMiniMap && <MiniMap pannable zoomable />}
      {legend && (
        <Panel position="bottom-left" className="catalog-graph-legend">
          {legend}
        </Panel>
      )}
      {children}
    </ReactFlow>
  );
}
