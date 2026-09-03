import { Node, Edge } from 'reactflow';

export interface BfsLevelLayoutOptions {
  rootId: string;
  columnWidth?: number;
  rowHeight?: number;
}

// Non-dagre alternative, ported from components/lineage/LineageGraph.tsx. That
// viewer deliberately avoided dagre because a layered layout doesn't preserve a
// clean upstream/downstream split visually — BFS-from-root levels do, trivially:
// each column is a hop distance from the root, so direction reads left-to-right
// (or right-to-left for upstream) regardless of how dagre would rank the DAG.
export function applyBfsLevelLayout(
  nodes: Node[],
  edges: Edge[],
  { rootId, columnWidth = 250, rowHeight = 100 }: BfsLevelLayoutOptions
): Node[] {
  const adjacency: Record<string, string[]> = {};
  edges.forEach((e) => {
    if (!adjacency[e.source]) adjacency[e.source] = [];
    adjacency[e.source].push(e.target);
  });

  const levels: Record<string, number> = {};
  const visited = new Set<string>();
  const queue: { id: string; level: number }[] = [{ id: rootId, level: 0 }];

  while (queue.length > 0) {
    const { id, level } = queue.shift()!;
    if (visited.has(id)) continue;
    visited.add(id);
    levels[id] = level;
    (adjacency[id] || []).forEach((next) => queue.push({ id: next, level: level + 1 }));
  }

  const nodesByLevel: Record<number, Node[]> = {};
  nodes.forEach((n) => {
    const level = levels[n.id] ?? 0;
    if (!nodesByLevel[level]) nodesByLevel[level] = [];
    nodesByLevel[level].push(n);
  });

  return nodes.map((n) => {
    const level = levels[n.id] ?? 0;
    const idxInLevel = nodesByLevel[level].findIndex((x) => x.id === n.id);
    return {
      ...n,
      position: { x: level * columnWidth, y: idxInLevel * rowHeight },
    };
  });
}
