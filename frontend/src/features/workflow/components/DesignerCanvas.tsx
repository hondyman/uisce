import React, { useCallback, useRef } from 'react';
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
  useReactFlow,
} from 'reactflow';
import { DndContext, useDraggable, useDroppable, PointerSensor, useSensor, useSensors, type DragEndEvent } from '@dnd-kit/core';
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

const CANVAS_DROPPABLE_ID = 'workflow-designer-canvas';

interface DraggableToolbarItemProps {
  nodeType: string;
  title: string;
  icon: string;
}

const DraggableToolbarItem: React.FC<DraggableToolbarItemProps> = ({ nodeType, title, icon }) => {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `designer-toolbar-${nodeType}`,
    data: { nodeType },
  });

  return (
    <div
      ref={setNodeRef}
      className="flex h-12 w-12 items-center justify-center rounded-md bg-background-light dark:bg-background-dark text-gray-500 dark:text-gray-400 hover:bg-primary/10 hover:text-primary cursor-grab"
      title={title}
      style={{ opacity: isDragging ? 0.4 : 1 }}
      {...attributes}
      {...listeners}
    >
      <span className="material-symbols-outlined text-3xl">{icon}</span>
    </div>
  );
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
  const { setNodeRef: setCanvasDroppableRef } = useDroppable({ id: CANVAS_DROPPABLE_ID });
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } })
  );

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over, delta } = event;
      if (!over || over.id !== CANVAS_DROPPABLE_ID || !reactFlowWrapper.current) return;

      const type = (active.data.current as { nodeType?: string } | undefined)?.nodeType;
      if (!type) return;

      const reactFlowBounds = reactFlowWrapper.current.getBoundingClientRect();
      const activatorEvent = event.activatorEvent as PointerEvent | MouseEvent | undefined;
      const startX = activatorEvent && 'clientX' in activatorEvent ? activatorEvent.clientX : reactFlowBounds.left;
      const startY = activatorEvent && 'clientY' in activatorEvent ? activatorEvent.clientY : reactFlowBounds.top;

      const position = project({
        x: startX + delta.x - reactFlowBounds.left,
        y: startY + delta.y - reactFlowBounds.top,
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

  return (
    <DndContext sensors={sensors} onDragEnd={handleDragEnd}>
      <div className="flex-1 flex h-full w-full">
        {/* Floating Toolbar (Sidebar) */}
        <div className="absolute top-4 left-4 flex flex-col gap-3 rounded-lg bg-white dark:bg-[#18232f] border border-gray-200 dark:border-gray-700 p-2 shadow-lg z-10">
          <DraggableToolbarItem nodeType="activity" title="Activity" icon="settings" />
          <DraggableToolbarItem nodeType="approval" title="Approval" icon="person" />
          <DraggableToolbarItem nodeType="event" title="Event" icon="notifications" />
          <DraggableToolbarItem nodeType="decision" title="Decision" icon="call_split" />
        </div>

        <div
          className="flex-1 h-full w-full"
          ref={(node) => {
            (reactFlowWrapper as React.MutableRefObject<HTMLDivElement | null>).current = node;
            setCanvasDroppableRef(node);
          }}
        >
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={(_, node) => onNodeSelect(node.id)}
            onPaneClick={() => onNodeSelect(null)}
            nodeTypes={nodeTypes}
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
    </DndContext>
  );
};
