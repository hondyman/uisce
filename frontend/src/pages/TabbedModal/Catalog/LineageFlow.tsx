// React import removed (automatic JSX runtime in use)
import ReactFlow, { Background, Controls, MiniMap, ConnectionLineType } from 'reactflow';
import 'reactflow/dist/style.css'; // Import React Flow styles
import './LineageFlow.css';
import React from 'react';

interface LineageFlowProps {
  nodes: any[];
  edges: any[];
  nodeTypes?: any;
  edgeTypes?: any;
  onNodeClick?: (event: React.MouseEvent, node: any) => void;
  onEdgeClick?: (event: React.MouseEvent, edge: any) => void;
  showMiniMap?: boolean;
  highlightedNodes?: string[];
}

const LineageFlow: React.FC<LineageFlowProps> = ({ 
  nodes, 
  edges, 
  nodeTypes,
  edgeTypes,
  onNodeClick, 
  onEdgeClick,
  showMiniMap = true,
  highlightedNodes = []
}) => {
  const processedNodes = React.useMemo(() => {
    if (!highlightedNodes || highlightedNodes.length === 0) return nodes;
    return nodes.map(node => ({
      ...node,
      data: {
        ...node.data,
        isHighlighted: highlightedNodes.includes(node.id)
      }
    }));
  }, [nodes, highlightedNodes]);

  // React Flow will handle all the rendering and edge connections automatically.
  return (
    <div className="lineage-flow-root">
      <ReactFlow
        nodes={processedNodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodeClick={onNodeClick}
        onEdgeClick={onEdgeClick}
        connectionLineType={ConnectionLineType.SmoothStep}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        nodesDraggable={false}
        nodesConnectable={false}
        edgesUpdatable={false}
        edgesFocusable={true}
        defaultEdgeOptions={{
          animated: false,
          style: { strokeWidth: 2 },
        }}
        proOptions={{ hideAttribution: true }}
      >
        <Background gap={16} color="#f1f5f9" />
        <Controls position="top-left" />
        {showMiniMap && (
          <MiniMap 
            nodeStrokeWidth={3} 
            zoomable 
            pannable 
            position="bottom-left" 
          />
        )}
      </ReactFlow>
    </div>
  );
};

export default LineageFlow;