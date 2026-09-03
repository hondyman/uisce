import dagre from 'dagre';
import { Node, Edge } from 'reactflow';

const DEFAULT_DIMENSIONS = { width: 220, height: 90 };

export interface NodeDimensionsLookup {
  (nodeType: string): { width: number; height: number };
}

export interface DagreLayoutOptions {
  direction?: 'TB' | 'LR';
  rankSep?: number;
  nodeSep?: number;
  getDimensions?: NodeDimensionsLookup;
}

// Standard dagre layered layout, proven in BOLineageGraphTab.tsx / ImpactGraph.tsx.
// Positions each node at the center dagre computed, adjusted for its own size.
export function applyDagreLayout(
  nodes: Node[],
  edges: Edge[],
  options: DagreLayoutOptions = {}
): Node[] {
  const { direction = 'LR', rankSep = 120, nodeSep = 80, getDimensions } = options;
  const dimensionsFor = getDimensions || (() => DEFAULT_DIMENSIONS);

  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));
  dagreGraph.setGraph({ rankdir: direction, ranksep: rankSep, nodesep: nodeSep });

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, dimensionsFor(node.type || 'default'));
  });

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  dagre.layout(dagreGraph);

  return nodes.map((node) => {
    const dims = dimensionsFor(node.type || 'default');
    const withPosition = dagreGraph.node(node.id);
    return {
      ...node,
      position: {
        x: (withPosition?.x ?? 0) - dims.width / 2,
        y: (withPosition?.y ?? 0) - dims.height / 2,
      },
    };
  });
}
