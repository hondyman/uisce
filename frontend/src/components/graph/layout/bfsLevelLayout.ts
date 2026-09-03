import { Node as FlowNode, Edge } from 'reactflow';
import { GetNodeDimensions, NodeDimensions } from './dagreLayout';

const DEFAULT_DIMENSIONS: NodeDimensions = { width: 220, height: 60 };

export interface BfsLevelLayoutOptions {
  rootId?: string;
  levelSeparation?: number;
  nodeSeparation?: number;
  getNodeDimensions?: GetNodeDimensions;
}

/**
 * Arranges nodes into horizontal levels by BFS distance from a root node
 * (or from all in-degree-0 nodes when no root is given). Nodes unreachable
 * from any root are appended as an extra trailing level so nothing is lost.
 */
export function bfsLevelLayout(
  nodes: FlowNode[],
  edges: Edge[],
  options: BfsLevelLayoutOptions = {}
): FlowNode[] {
  if (!nodes || nodes.length === 0) return [];

  const { rootId, levelSeparation = 160, nodeSeparation = 60, getNodeDimensions } = options;

  const adjacency = new Map<string, string[]>();
  const inDegree = new Map<string, number>();
  for (const node of nodes) {
    adjacency.set(node.id, []);
    inDegree.set(node.id, 0);
  }
  for (const edge of edges) {
    if (adjacency.has(edge.source) && adjacency.has(edge.target)) {
      adjacency.get(edge.source)!.push(edge.target);
      inDegree.set(edge.target, (inDegree.get(edge.target) || 0) + 1);
    }
  }

  const roots = rootId && adjacency.has(rootId)
    ? [rootId]
    : nodes.filter((n) => (inDegree.get(n.id) || 0) === 0).map((n) => n.id);
  const startIds = roots.length > 0 ? roots : [nodes[0].id];

  const level = new Map<string, number>();
  const queue: string[] = [];
  for (const id of startIds) {
    level.set(id, 0);
    queue.push(id);
  }
  while (queue.length > 0) {
    const id = queue.shift()!;
    const depth = level.get(id)!;
    for (const next of adjacency.get(id) || []) {
      if (!level.has(next)) {
        level.set(next, depth + 1);
        queue.push(next);
      }
    }
  }

  const maxLevel = level.size > 0 ? Math.max(...level.values()) : -1;
  const unreached = nodes.filter((n) => !level.has(n.id));
  for (const node of unreached) {
    level.set(node.id, maxLevel + 1);
  }

  const byLevel = new Map<number, FlowNode[]>();
  for (const node of nodes) {
    const lvl = level.get(node.id) ?? 0;
    if (!byLevel.has(lvl)) byLevel.set(lvl, []);
    byLevel.get(lvl)!.push(node);
  }

  const positioned: FlowNode[] = [];
  const sortedLevels = Array.from(byLevel.keys()).sort((a, b) => a - b);
  let y = 0;
  for (const lvl of sortedLevels) {
    const rowNodes = byLevel.get(lvl)!;
    let x = 0;
    let rowHeight = 0;
    for (const node of rowNodes) {
      const size = getNodeDimensions ? getNodeDimensions(node) : DEFAULT_DIMENSIONS;
      positioned.push({ ...node, position: { x, y } });
      x += size.width + nodeSeparation;
      rowHeight = Math.max(rowHeight, size.height);
    }
    y += rowHeight + levelSeparation;
  }

  return positioned;
}
