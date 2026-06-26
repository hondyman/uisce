import React, { useCallback, useMemo, useRef } from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  Node,
  Edge,
  OnNodesChange,
  OnEdgesChange,
  Connection,
  NodeTypes,
  ReactFlowInstance,
  useReactFlow,
} from 'reactflow';
import { ActivityNode, ApprovalNode, DecisionNode, EventNode, StartNode, EndNode } from './CustomNodes';

interface DesignerCanvasProps {
  nodes: Node[];
  edges: Edge[];
  onNodesChange: OnNodesChange;
  onEdgesChange: OnEdgesChange;
  onConnect: (params: Connection) => void;
  onNodeSelect: (id: string | null) => void;
  setNodes: React.Dispatch<React.SetStateAction<Node[]>>;
}

const nodeTypes: NodeTypes = {
  activity: ActivityNode,
  approval: ApprovalNode,
  decision: DecisionNode,
  event: EventNode,
  start: StartNode,
  end: EndNode,
};

export const DesignerCanvas: React.FC<DesignerCanvasProps> = ({
  nodes,
  edges,
  onNodesChange,
  onEdgesChange,
  onConnect,
  onNodeSelect,
  setNodes,
}) => {
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const { project } = useReactFlow();

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const type = event.dataTransfer.getData('application/reactflow');

      // check if the dropped element is valid
      if (typeof type === 'undefined' || !type) {
        return;
      }

      const reactFlowBounds = reactFlowWrapper.current?.getBoundingClientRect();
      
      if (!reactFlowBounds) return;

      const position = project({
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      });

      const newNode: Node = {
        id: `node-${nodes.length + 1}`,
        type,
        position,
        data: { label: `${type.charAt(0).toUpperCase() + type.slice(1)}` },
      };

      setNodes((nds) => nds.concat(newNode));
    },
    [project, nodes.length, setNodes]
  );

  const onDragStart = (event: React.DragEvent, nodeType: string) => {
    event.dataTransfer.setData('application/reactflow', nodeType);
    event.dataTransfer.effectAllowed = 'move';
  };

  return (
    <div className="flex-1 flex h-full w-full">
       {/* Floating Toolbar (Sidebar) */}
       <div className="absolute top-4 left-4 flex flex-col gap-3 rounded-lg bg-white dark:bg-[#18232f] border border-gray-200 dark:border-gray-700 p-2 shadow-lg z-10">
        <div 
            className="flex h-12 w-12 items-center justify-center rounded-md bg-background-light dark:bg-background-dark text-gray-500 dark:text-gray-400 hover:bg-primary/10 hover:text-primary cursor-grab" 
            title="Activity"
            draggable
            onDragStart={(event) => onDragStart(event, 'activity')}
        >
          <span className="material-symbols-outlined text-3xl">settings</span>
        </div>
        <div 
            className="flex h-12 w-12 items-center justify-center rounded-md bg-background-light dark:bg-background-dark text-gray-500 dark:text-gray-400 hover:bg-primary/10 hover:text-primary cursor-grab" 
            title="Approval"
            draggable
            onDragStart={(event) => onDragStart(event, 'approval')}
        >
          <span className="material-symbols-outlined text-3xl">person</span>
        </div>
        <div 
            className="flex h-12 w-12 items-center justify-center rounded-md bg-background-light dark:bg-background-dark text-gray-500 dark:text-gray-400 hover:bg-primary/10 hover:text-primary cursor-grab" 
            title="Event"
            draggable
            onDragStart={(event) => onDragStart(event, 'event')}
        >
          <span className="material-symbols-outlined text-3xl">notifications</span>
        </div>
        <div 
            className="flex h-12 w-12 items-center justify-center rounded-md bg-background-light dark:bg-background-dark text-gray-500 dark:text-gray-400 hover:bg-primary/10 hover:text-primary cursor-grab" 
            title="Decision"
            draggable
            onDragStart={(event) => onDragStart(event, 'decision')}
        >
          <span className="material-symbols-outlined text-3xl">call_split</span>
        </div>
      </div>

      <div className="flex-1 h-full w-full" ref={reactFlowWrapper}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onNodeClick={(_, node) => onNodeSelect(node.id)}
          onPaneClick={() => onNodeSelect(null)}
          nodeTypes={nodeTypes}
          onDrop={onDrop}
          onDragOver={onDragOver}
          fitView
          className="bg-dots"
        >
          <Background color="#9dabb9" gap={20} size={1} />
          <Controls />
          <MiniMap />
        </ReactFlow>
      </div>

      <style>{`
        .bg-dots {
            background-color: var(--bg-background-light);
        }
        .dark .bg-dots {
            background-color: var(--bg-background-dark);
        }
        .react-flow__node {
            border: none;
            background: transparent;
        }
      `}</style>
    </div>
  );
};
