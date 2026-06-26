import { useMemo } from 'react';
import { Edge, Node } from 'reactflow';

const _LAYOUT_DIRECTION = 'TB'; // Top-to-Bottom (unused placeholder)
// reference to avoid unused var warning
void _LAYOUT_DIRECTION;

export const useHierarchicalLayout = (nodes: Node[], edges: Edge[], layout: string) => {
  const isHierarchical = layout === 'hierarchical';

  const processedElements = useMemo(() => {
    if (!isHierarchical) {
      // Basic auto-layout for flat view (can be improved)
      const newNodes = nodes.map((node, index) => ({
        ...node,
        position: { x: (index % 5) * 200, y: Math.floor(index / 5) * 150 },
      }));
      return { nodes: newNodes, edges };
    }

    // The backend should have already provided positions for the hierarchical layout
    return { nodes, edges };

  }, [nodes, edges, isHierarchical]);

  return processedElements;
};
