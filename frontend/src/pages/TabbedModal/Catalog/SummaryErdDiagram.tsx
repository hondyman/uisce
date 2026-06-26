/* eslint-disable @typescript-eslint/no-unused-vars */
import { useCallback, useMemo } from 'react';
import type { FC } from 'react';
import ReactFlow, {
  MiniMap,
  Controls,
  Background,
  Node as FlowNode,
  Edge,
  ReactFlowInstance,
  NodeMouseHandler,
  Viewport
} from 'reactflow';
import 'reactflow/dist/style.css';

interface ErdDiagramProps {
  nodes: FlowNode[];
  edges: Edge[];
  nodeTypes: any;
  showColumns: boolean;
  highlightedItem: string | null;
  onInit: (_instance: ReactFlowInstance) => void;
  onNodeClick: (_event: React.MouseEvent, _node: FlowNode) => void;
  onEdgeClick: (_event: React.MouseEvent, _edge: Edge) => void;
  onPaneClick: () => void;
  onMoveEnd: (_event: any, _viewport: Viewport) => void;
  showMiniMap: boolean;
}

const ErdDiagram: FC<ErdDiagramProps> = ({
  nodes: initialNodes,
  edges: initialEdges,
  nodeTypes,
  showColumns,
  highlightedItem,
  onInit,
  onNodeClick,
  onEdgeClick,
  onPaneClick,
  onMoveEnd
}) => {
  // Local typed tooltip shape to avoid `as any` casts
  type TooltipWithCleanup = HTMLElement & { cleanup?: () => void };
  const { showMiniMap } = { showMiniMap: true };

  // Helper function to generate static positions for nodes
  const generateNodePositions = (nodes: FlowNode[]): FlowNode[] => {
    const gridSpacing = 300;
    const columnsPerRow = Math.ceil(Math.sqrt(nodes.length));
    
    return nodes.map((node, index) => {
      const row = Math.floor(index / columnsPerRow);
      const col = index % columnsPerRow;
      
      return {
        ...node,
        position: {
          x: col * gridSpacing + (row % 2) * (gridSpacing / 2), // Offset alternate rows
          y: row * gridSpacing
        }
      };
    });
  };

  // Helper function to get node type display name
  const getNodeTypeDisplay = (node: FlowNode): string => {
    // Check if it's a business term node (from lineage graph)
    if (node.data?.type) {
      switch (node.data.type) {
        case 'business_term':
          return 'Business Term';
        case 'semantic_term':
          return 'Semantic Term';
        case 'semantic_view':
          return 'Semantic View';
        case 'column':
          return 'Database Column';
        default:
          return String(node.data.type);
      }
    }
    
    // Check if it's a database table node
    if (node.type === 'databaseTable' || node.data?.columns) {
      return 'Database Table';
    }
    
    return node.type || 'Node';
  };

  // Static nodes with fixed positions and ABSOLUTELY NO visual effects
  const staticNodes = useMemo(() => {
    // Generate positions if nodes don't have them or if they're all at 0,0
    const needsPositioning = initialNodes.some(node => 
      !node.position || (node.position.x === 0 && node.position.y === 0)
    );
    
    const positionedNodes = needsPositioning ? 
      generateNodePositions(initialNodes) : 
      initialNodes;

    return positionedNodes.map((node) => {
      const isHighlighted = highlightedItem === `table-${node.id}` || 
                           highlightedItem === `column-${node.id}`;
      
      const nodeTypeDisplay = getNodeTypeDisplay(node);
      
      return {
        ...node,
        data: {
          ...node.data,
          showColumns,
          nodeType: nodeTypeDisplay,
        },
        style: {
          ...node.style,
          cursor: 'pointer',
          border: isHighlighted ? '2px solid #007bff' : node.style?.border,
        },
        draggable: false,
        selectable: true,
      };
    });
  }, [initialNodes, showColumns, highlightedItem]);

  // Static edges with highlighting
  const staticEdges = useMemo(() => {
    return initialEdges.map((edge) => ({
      ...edge,
      style: {
        ...edge.style,
        strokeWidth: edge.style?.strokeWidth || 2,
        stroke: edge.style?.stroke || '#b1b1b7',
      },
      animated: false
    }));
  }, [initialEdges]);

  // Handle node mouse enter - simplified tooltip without movement
  const onNodeMouseEnter: NodeMouseHandler = useCallback((event, node) => {
    
    // Create or update tooltip
    const tooltip = document.getElementById('node-tooltip');
    if (tooltip) {
      tooltip.remove();
    }
    
    const newTooltip = document.createElement('div');
    newTooltip.id = 'node-tooltip';
    newTooltip.style.cssText = `
      position: fixed;
      background: rgba(0, 0, 0, 0.9);
      color: white;
      padding: 8px 12px;
      border-radius: 6px;
      font-size: 12px;
      pointer-events: none;
      z-index: 9999;
      white-space: pre-line;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
      max-width: 200px;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    `;
    
    const nodeTypeDisplay = getNodeTypeDisplay(node);
    newTooltip.textContent = `${nodeTypeDisplay}\n${node.data?.label || 'Unknown'}\nID: ${node.id}\n\nClick to select and view details`;
    
    document.body.appendChild(newTooltip);
    
    // Position tooltip near mouse
    const updateTooltipPosition = (e: Event) => {
      const mouseEvent = e as MouseEvent;
      newTooltip.style.left = `${mouseEvent.clientX + 10}px`;
      newTooltip.style.top = `${mouseEvent.clientY - 10}px`;
    };
    
  // Position initially
  updateTooltipPosition(event as unknown as Event);
    
    // Follow mouse movement
    const mouseMoveHandler = (e: MouseEvent) => updateTooltipPosition(e);
    document.addEventListener('mousemove', mouseMoveHandler);
    
    // Store cleanup function
    (newTooltip as TooltipWithCleanup).cleanup = () => {
      document.removeEventListener('mousemove', mouseMoveHandler);
    };
  }, []);

  // Handle node mouse leave
  const onNodeMouseLeave: NodeMouseHandler = useCallback(() => {
    // Remove tooltip and cleanup event listeners
    const tooltip = document.getElementById('node-tooltip');
    if (tooltip) {
      const t = tooltip as TooltipWithCleanup;
      if (typeof t.cleanup === 'function') {
        t.cleanup();
      }
      tooltip.remove();
    }
  }, []);

  return (
    <div className="erd-diagram-root">
      <ReactFlow
        nodes={staticNodes}
        edges={staticEdges}
        nodeTypes={nodeTypes}
        onInit={onInit}
        onNodeClick={onNodeClick}
        onEdgeClick={onEdgeClick}
        onPaneClick={onPaneClick}
        onMoveEnd={onMoveEnd}
        onNodeMouseEnter={onNodeMouseEnter}
        onNodeMouseLeave={onNodeMouseLeave}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={true}
        selectNodesOnDrag={false}
        panOnDrag={[1, 2]}
        fitView
        fitViewOptions={{
          padding: 0.1,
          minZoom: 0.1,
          maxZoom: 1.5
        }}
        defaultViewport={{ x: 0, y: 0, zoom: 0.8 }}
        minZoom={0.1}
        maxZoom={2}
        attributionPosition="bottom-left"
        proOptions={{ hideAttribution: true }}
        className="erd-reactflow"
      >
        <Controls position="top-left" className="erd-controls" />
        
        {showMiniMap && (
          <MiniMap
            position="top-left"
            className="erd-minimap"
            nodeStrokeColor={(node) => {
              if (node.data?.type === 'business_term') return '#8e44ad';
              if (node.data?.type === 'semantic_term') return '#3498db';
              if (node.data?.type === 'semantic_view') return '#27ae60';
              return '#666';
            }}
            nodeColor={(node) => {
              if (node.data?.type === 'business_term') return '#f3e5f5';
              if (node.data?.type === 'semantic_term') return '#e3f2fd';
              if (node.data?.type === 'semantic_view') return '#e8f5e8';
              return '#f8f9fa';
            }}
            nodeBorderRadius={6}
      maskColor="rgba(0, 0, 0, 0.1)"
          />
        )}
        
    <Background color="#e9ecef" gap={20} size={1} className="erd-background" />
      </ReactFlow>

      {/* Ultra-aggressive CSS to prevent ANY node movement */}
      <style dangerouslySetInnerHTML={{
        __html: `
          /* Completely disable all transforms and animations */
          .react-flow__renderer {
            z-index: 1 !important;
          }
          
          .react-flow__container {
            z-index: 1 !important;
          }
          
          .react-flow__pane {
            z-index: 1 !important;
          }
          
          .react-flow__viewport {
            z-index: 1 !important;
          }
          
          .react-flow__nodes {
            z-index: 2 !important;
          }
          
          .react-flow__edges {
            z-index: 1 !important;
          }
          
          .react-flow__node {
            cursor: pointer !important;
            position: relative !important;
            z-index: 3 !important;
            /* ABSOLUTELY NO MOVEMENT OF ANY KIND */
            transform: none !important;
            transition: none !important;
            animation: none !important;
            will-change: auto !important;
            filter: none !important;
            box-shadow: none !important;
            scale: none !important;
            translate: none !important;
            rotate: none !important;
          }
          
          .react-flow__node:hover {
            /* ZERO hover effects */
            transform: none !important;
            transition: none !important;
            animation: none !important;
            filter: none !important;
            box-shadow: none !important;
            scale: none !important;
            translate: none !important;
            rotate: none !important;
            background: none !important;
            border: none !important;
          }
          
          .react-flow__node.selected {
            transform: none !important;
            box-shadow: none !important;
            filter: none !important;
            scale: none !important;
            translate: none !important;
            rotate: none !important;
          }
          
          .react-flow__node * {
            transform: none !important;
            transition: none !important;
            animation: none !important;
            will-change: auto !important;
            filter: none !important;
            box-shadow: none !important;
          }
          
          .react-flow__node *:hover {
            transform: none !important;
            transition: none !important;
            animation: none !important;
            filter: none !important;
            box-shadow: none !important;
            scale: none !important;
            translate: none !important;
            rotate: none !important;
          }
          
          /* Kill any existing CSS that might interfere */
          .react-flow__node-databaseTable,
          .react-flow__node-databaseTable:hover,
          .react-flow__node-databaseTable.selected,
          .react-flow__node-databaseTable * {
            transform: none !important;
            transition: none !important;
            animation: none !important;
            filter: none !important;
            box-shadow: none !important;
            scale: none !important;
            translate: none !important;
            rotate: none !important;
            will-change: auto !important;
          }
          
          /* Override any global hover effects */
          [class*="node"]:hover,
          [class*="Node"]:hover,
          [data-node]:hover {
            transform: none !important;
            transition: none !important;
            animation: none !important;
            filter: none !important;
            box-shadow: none !important;
            scale: none !important;
            translate: none !important;
            rotate: none !important;
          }
          
          /* Edge styling only */
          .react-flow__edge {
            cursor: pointer !important;
            z-index: 2 !important;
          }
          
          .react-flow__edge:hover .react-flow__edge-path {
            stroke: #007bff !important;
            stroke-width: 3px !important;
          }
          
          .react-flow__controls {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
            z-index: 10 !important;
          }
          
          .react-flow__controls-button {
            background: white !important;
            border: none !important;
            border-bottom: 1px solid #e9ecef !important;
          }
          
          .react-flow__controls-button:hover {
            background: #f8f9fa !important;
          }
          
          .react-flow__minimap {
            z-index: 10 !important;
          }
          
          #node-tooltip {
            animation: none !important;
            z-index: 9999 !important;
          }
        `
      }} />
    </div>
  );
};

export default ErdDiagram;