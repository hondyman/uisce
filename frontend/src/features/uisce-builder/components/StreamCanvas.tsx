import React, { useCallback, useMemo, useRef, DragEvent } from 'react';
import ReactFlow, { 
  ReactFlowProvider, 
  Controls, 
  Background, 
  NodeTypes,
  MiniMap,
  ConnectionLineType,
} from 'reactflow';
import 'reactflow/dist/style.css';
import useUisceStore, { UisceNode } from '../hooks/useUisceStore';
import CustomNode from './CustomNode';
import { Box, useTheme } from '@mui/material';

const StreamCanvas = () => {
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const theme = useTheme();
  const { 
    nodes, edges, 
    onNodesChange, onEdgesChange, onConnect, 
    addNode, selectNode 
  } = useUisceStore();

  const nodeTypes = useMemo<NodeTypes>(() => ({
    default: CustomNode,
    input: CustomNode 
  }), []);

  const onDrop = useCallback(
    (event: DragEvent<HTMLDivElement>) => {
      event.preventDefault();
      
      const type = event.dataTransfer.getData('application/reactflow');
      if (!type || !reactFlowWrapper.current) return;

      const reactFlowBounds = reactFlowWrapper.current.getBoundingClientRect();
      const position = {
        x: event.clientX - reactFlowBounds.left - 100,
        y: event.clientY - reactFlowBounds.top,
      };

      const newNode: UisceNode = {
        id: `filter_${Date.now()}`,
        type: 'default',
        position,
        data: { 
            label: event.dataTransfer.getData('application/reactflow-label') || `${type} Filter`, 
            filterType: type,
            config: {} 
        },
      };

      addNode(newNode);
    },
    [addNode]
  );

  const onDragOver = useCallback((event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  return (
    <Box sx={{ width: '100%', height: '100%', flexGrow: 1, bgcolor: '#f8fafc' }} ref={reactFlowWrapper}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={(_e, node) => selectNode(node.id)}
        onPaneClick={() => selectNode(null)}
        nodeTypes={nodeTypes}
        onDrop={onDrop}
        onDragOver={onDragOver}
        fitView
        connectionLineType={ConnectionLineType.SmoothStep}
        defaultEdgeOptions={{
            type: 'smoothstep',
            animated: true,
            style: { stroke: '#64748b', strokeWidth: 2 }
        }}
      >
        <Background gap={24} color="#e2e8f0" />
        <Controls style={{ boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)', border: 'none', borderRadius: 8, overflow: 'hidden' }} />
        <MiniMap 
            style={{ height: 120, borderRadius: 8, boxShadow: '0 10px 15px -3px rgb(0 0 0 / 0.1)' }} 
            zoomable 
            pannable 
            nodeColor={(n) => {
                if (n.data.label.includes('Sanctions')) return '#fca5a5';
                if (n.data.label.includes('Limit')) return '#86efac';
                if (n.data.label.includes('AI')) return '#d8b4fe';
                return '#e2e8f0';
            }}
        />
      </ReactFlow>
    </Box>
  );
};

export default StreamCanvas;
