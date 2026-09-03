import dagre from 'dagre';
import { Node as FlowNode, Edge } from 'reactflow';

export interface NodeDimensions {
  width: number;
  height: number;
}

export type GetNodeDimensions = (node: FlowNode) => NodeDimensions;

const DEFAULT_DIMENSIONS: NodeDimensions = { width: 220, height: 60 };

export interface DagreLayoutOptions {
  direction?: 'TB' | 'LR' | 'BT' | 'RL';
  nodeSeparation?: number;
  rankSeparation?: number;
  getNodeDimensions?: GetNodeDimensions;
}

/**
 * Lays out a graph with dagre's hierarchical algorithm. Returns new node
 * objects (positions only change) — edges are returned unmodified.
 */
export function dagreLayout(
  nodes: FlowNode[],
  edges: Edge[],
  options: DagreLayoutOptions = {}
): FlowNode[] {
  if (!nodes || nodes.length === 0) return [];

  const {
    direction = 'TB',
    nodeSeparation = 60,
    rankSeparation = 100,
    getNodeDimensions,
  } = options;

  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: direction, nodesep: nodeSeparation, ranksep: rankSeparation });

  const dims = new Map<string, NodeDimensions>();
  for (const node of nodes) {
    const size = getNodeDimensions ? getNodeDimensions(node) : DEFAULT_DIMENSIONS;
    dims.set(node.id, size);
    g.setNode(node.id, { width: size.width, height: size.height });
  }

  const nodeIds = new Set(nodes.map((n) => n.id));
  for (const edge of edges) {
    if (nodeIds.has(edge.source) && nodeIds.has(edge.target)) {
      g.setEdge(edge.source, edge.target);
    }
  }

  dagre.layout(g);

  return nodes.map((node) => {
    const pos = g.node(node.id);
    const size = dims.get(node.id) || DEFAULT_DIMENSIONS;
    if (!pos) return node;
    return {
      ...node,
      position: {
        x: pos.x - size.width / 2,
        y: pos.y - size.height / 2,
      },
    };
  });
}
