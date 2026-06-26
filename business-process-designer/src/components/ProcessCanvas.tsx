import React, { useCallback, useEffect } from 'react';
import {
  ReactFlow,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
  Edge,
  Node,
  NodeTypes,
  OnConnect,
} from 'reactflow';
import { StepType } from './StepPalette';
import StepNode from './nodes/StepNode';

import 'reactflow/dist/style.css';

const nodeTypes: NodeTypes = {
  stepNode: StepNode,
};

const initialNodes: Node[] = [
  {
    id: '1',
    type: 'stepNode',
    position: { x: 250, y: 25 },
    data: {
      label: 'Initiate Request',
      stepType: 'initiate',
      rules: [],
      eventId: '',
      status: 'active'
    },
  },
];

const initialEdges: Edge[] = [];

interface ProcessCanvasProps {
  onNodeSelect?: (node: Node | null) => void;
  processData?: any;
  onProcessChange?: (process: any) => void;
}

export default function ProcessCanvas({ onNodeSelect, processData, onProcessChange }: ProcessCanvasProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  // Update nodes when process data changes
  useEffect(() => {
    if (processData?.steps) {
      const processNodes = processData.steps.map((step: any, index: number) => ({
        id: `step-${step.id || index + 1}`,
        type: 'stepNode',
        position: { x: 250, y: 25 + (index * 150) },
        data: {
          label: step.stepName || `Step ${index + 1}`,
          stepType: step.stepType,
          rules: step.validationRules || [],
          eventId: step.eventId || '',
          status: step.status || 'draft'
        },
      }));
      setNodes(processNodes.length > 0 ? processNodes : initialNodes);
    }
  }, [processData, setNodes]);

  // Notify parent of process changes
  useEffect(() => {
    if (onProcessChange) {
      const processSteps = nodes.map((node, index) => ({
        id: node.id,
        stepOrder: index + 1,
        stepType: node.data.stepType,
        stepName: node.data.label,
        validationRules: node.data.rules || [],
        eventId: node.data.eventId || '',
        status: node.data.status || 'draft'
      }));
      onProcessChange({ ...processData, steps: processSteps });
    }
  }, [nodes, onProcessChange, processData]);

  const onConnect: OnConnect = useCallback(
    (params) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  const onNodeClick = useCallback((_event: React.MouseEvent, node: Node) => {
    onNodeSelect?.(node);
  }, [onNodeSelect]);

  const onPaneClick = useCallback(() => {
    onNodeSelect?.(null);
  }, [onNodeSelect]);

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const reactFlowBounds = event.currentTarget.getBoundingClientRect();
      const stepTypeData = event.dataTransfer.getData('application/json');

      if (stepTypeData) {
        const stepType: StepType = JSON.parse(stepTypeData);

        const position = {
          x: event.clientX - reactFlowBounds.left,
          y: event.clientY - reactFlowBounds.top,
        };

        const newNode: Node = {
          id: `${nodes.length + 1}`,
          type: 'stepNode',
          position,
          data: {
            label: stepType.label,
            stepType: stepType.id,
            rules: [],
            eventId: '',
            status: 'draft'
          },
        };

        setNodes((nds) => nds.concat(newNode));
      }
    },
    [nodes.length, setNodes]
  );

  return (
    <div className="canvas-wrapper">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        onDragOver={onDragOver}
        onDrop={onDrop}
        nodeTypes={nodeTypes}
        fitView
        attributionPosition="top-right"
      >
        <Controls />
        <Background color="#aaa" gap={16} />
      </ReactFlow>
    </div>
  );
}