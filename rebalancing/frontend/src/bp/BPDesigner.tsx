import React, { useCallback, useState, useRef } from "react";
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  addEdge,
  Connection,
  Edge,
  Node,
  ReactFlowProvider,
  useNodesState,
  useEdgesState,
  useReactFlow,
} from "reactflow";
import "reactflow/dist/style.css";
import { useDesignerState } from "../hooks/useDesignerState";
import { ApprovalNode, ApprovalNodeData } from './ApprovalNode'; // Verify path
import { NodePalette } from './NodePalette'; // Verify path
import { ApprovalConfigPanel } from './ApprovalConfigPanel'; // Verify path

const nodeTypes = {
  approval: ApprovalNode,
};

interface Props {
  tenantId?: string;
  bpDefId?: string;
}

const BPDesignerCanvas: React.FC<Props> = ({ tenantId, bpDefId }) => {
  const { state, load, save, setNodes: setDesignerNodes, setEdges: setDesignerEdges } = useDesignerState(tenantId);
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  
  // Local state for React Flow
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  
  const { project } = useReactFlow();

  // Sync initial load
  React.useEffect(() => {
    if (bpDefId) {
       load(bpDefId);
    }
  }, [bpDefId, load]);

  // Sync state -> local nodes
  React.useEffect(() => {
    if (state.nodes.length > 0) {
        // Transform incoming nodes if needed, or assume compatible
        // Note: state.nodes come from useDesignerState which maps from Backend Steps.
        // We might need to ensure types match. 
        // useDesignerState currently maps steps to nodes.
        const mappedNodes = state.nodes.map(n => {
           if (n.type === 'approval' && !n.data.approvalChain) {
               // Ensure default data structure if missing
               return { ...n, data: { ...n.data, approvalChain: { rules: [], fallbackRole: '' } } };
           }
           return n;
        });
        setNodes(mappedNodes);
        
        // If we had edges in state, use them. Currently useDesignerState might rely on explicit edges or seq.
        // If the backend assumes sequence, we might not have edges.
        // BUT, the user wants visual designer. We'll stick to local edges for now if backend doesn't provide them.
        if (state.edges && state.edges.length > 0) setEdges(state.edges);
    }
  }, [state.nodes, state.edges, setNodes, setEdges]);

  // Sync local -> state (for autosave capability)
  React.useEffect(() => {
      setDesignerNodes(nodes);
      setDesignerEdges(edges);
  }, [nodes, edges, setDesignerNodes, setDesignerEdges]);

  const onConnect = useCallback((params: Connection) => setEdges((eds) => addEdge(params, eds)), [setEdges]);

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const reactFlowBounds = reactFlowWrapper.current?.getBoundingClientRect();
      const data = event.dataTransfer.getData('application/reactflow');
      
      if (!data || !reactFlowBounds) return;

      const nodeSpec = JSON.parse(data);
      const position = project({
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      });

      const newNode: Node = {
        id: `node_${Date.now()}`,
        type: nodeSpec.type,
        position,
        data: { 
            label: nodeSpec.label,
            stepKey: `${nodeSpec.type}_${Date.now()}`,
            // Init data based on type
            ...(nodeSpec.type === 'approval' ? { approvalChain: { rules: [], fallbackRole: '' } } : {})
        },
      };

      setNodes((nds) => [...nds, newNode]);
      setSelectedNodeId(newNode.id);
    },
    [project, setNodes]
  );
  
  const updateNodeData = (nodeId: string, data: any) => {
    setNodes((nds) =>
      nds.map((node) => (node.id === nodeId ? { ...node, data } : node))
    );
  };

  const selectedNode = nodes.find((n) => n.id === selectedNodeId);

  return (
    <div style={{ display: 'flex', height: '100%', flexDirection: 'column' }}>
       {/* Toolbar / Header */}
       <div style={{ padding: '10px', borderBottom: '1px solid #ddd', display: 'flex', justifyContent: 'space-between', background: '#f8f9fa' }}>
          <div>
            <strong>BP Designer</strong> {state.bpDefId ? `- ${state.bpDefId}` : '(New)'}
          </div>
          <div>
             {state.lastSavedAt && <span style={{ marginRight: 10, fontSize: 12, color: '#666' }}>Saved: {new Date(state.lastSavedAt).toLocaleTimeString()}</span>}
             {state.isSaving && <span style={{ marginRight: 10, fontSize: 12, color: 'blue' }}>Saving...</span>}
             {state.error && <span style={{ marginRight: 10, fontSize: 12, color: 'red' }}>Error: {state.error}</span>}
             <button onClick={save} disabled={state.isSaving}>Save Draft</button>
          </div>
       </div>

       <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
            {/* Palette */}
            <div style={{ width: '200px', borderRight: '1px solid #cbd5e1', background: '#f8fafc', overflowY: 'auto' }}>
                <NodePalette />
            </div>

            {/* Canvas */}
            <div className="reactflow-wrapper" ref={reactFlowWrapper} style={{ flex: 1, position: 'relative' }}>
                <ReactFlow
                    nodes={nodes}
                    edges={edges}
                    onNodesChange={onNodesChange}
                    onEdgesChange={onEdgesChange}
                    onConnect={onConnect}
                    onDragOver={onDragOver}
                    onDrop={onDrop}
                    onNodeClick={(_, node) => setSelectedNodeId(node.id)}
                    nodeTypes={nodeTypes}
                    fitView
                >
                    <Background />
                    <Controls />
                    <MiniMap />
                </ReactFlow>
            </div>

            {/* Inspector Panel */}
            {selectedNode && selectedNode.type === 'approval' && (
                <div style={{ width: '300px', borderLeft: '1px solid #cbd5e1', background: '#fff', overflowY: 'auto' }}>
                    <ApprovalConfigPanel
                        node={selectedNode}
                        onUpdate={updateNodeData}
                    />
                </div>
            )}
             {selectedNode && selectedNode.type !== 'approval' && (
                <div style={{ width: '300px', borderLeft: '1px solid #cbd5e1', padding: '16px' }}>
                    <h3>{selectedNode.data.label}</h3>
                    <p>Configuration for this node type not yet implemented in this demo.</p>
                </div>
            )}
       </div>
    </div>
  );
};

// Wrap in Provider
export const BPDesigner: React.FC<Props> = (props) => (
    <ReactFlowProvider>
        <BPDesignerCanvas {...props} />
    </ReactFlowProvider>
);
