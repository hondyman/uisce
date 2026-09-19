import React, { useCallback, useMemo, useRef } from 'react';
import ReactFlow, {
  ReactFlowProvider,
  Controls,
  Background,
  NodeTypes,
  MiniMap,
  ConnectionLineType,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useDroppable, useDndMonitor, DragEndEvent } from '@dnd-kit/core';
import useUisceStore, { UisceNode } from '../hooks/useUisceStore';
import CustomNode from './CustomNode';
import { Box, useTheme, alpha } from '@mui/material';

export const STREAM_CANVAS_DROPZONE_ID = 'stream-canvas-dropzone';

const StreamCanvas = () => {
  const reactFlowWrapper = useRef<HTMLDivElement | null>(null);
  const theme = useTheme();
  const {
    nodes, edges,
    onNodesChange, onEdgesChange, onConnect,
    addNode, selectNode
  } = useUisceStore();

  const { setNodeRef, isOver } = useDroppable({ id: STREAM_CANVAS_DROPZONE_ID });

  const setRefs = useCallback((node: HTMLDivElement | null) => {
    reactFlowWrapper.current = node;
    setNodeRef(node);
  }, [setNodeRef]);

  const nodeTypes = useMemo<NodeTypes>(() => ({
    default: CustomNode,
    input: CustomNode
  }), []);

  useDndMonitor({
    onDragEnd(event: DragEndEvent) {
      const { active, over } = event;
      if (over?.id !== STREAM_CANVAS_DROPZONE_ID) return;
      const filterType = active.data.current?.filterType as string | undefined;
      if (!filterType || !reactFlowWrapper.current) return;

      const label = active.data.current?.label as string | undefined;
      const bounds = reactFlowWrapper.current.getBoundingClientRect();
      const activeRect = active.rect.current.translated ?? active.rect.current.initial;
      const centerX = activeRect ? activeRect.left + activeRect.width / 2 : bounds.left + bounds.width / 2;
      const centerY = activeRect ? activeRect.top + activeRect.height / 2 : bounds.top + bounds.height / 2;

      const position = {
        x: centerX - bounds.left - 100,
        y: centerY - bounds.top,
      };

      const newNode: UisceNode = {
        id: `filter_${Date.now()}`,
        type: 'default',
        position,
        data: {
          label: label || `${filterType} Filter`,
          filterType,
          config: {},
        },
      };

      addNode(newNode);
    },
  });

  return (
    <Box
      sx={{
        width: '100%',
        height: '100%',
        flexGrow: 1,
        bgcolor: isOver ? alpha('#6366f1', 0.06) : '#f8fafc',
        transition: 'background-color 0.15s ease',
      }}
      ref={setRefs}
    >
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={(_e, node) => selectNode(node.id)}
        onPaneClick={() => selectNode(null)}
        nodeTypes={nodeTypes}
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
