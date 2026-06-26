import React, { useCallback, useState } from 'react';
import { ReactFlowProvider, Node, Edge, useNodesState, useEdgesState, Connection, addEdge } from 'reactflow';
import { DesignerCanvas } from '../components/DesignerCanvas';
import { PropertiesPanel } from '../components/PropertiesPanel';
import 'reactflow/dist/style.css';
import { devDebug } from '../../../utils/devLogger';

const initialNodes: Node[] = [
  { id: 'start', type: 'start', position: { x: 50, y: 250 }, data: { label: 'Start' } },
  { id: 'node-1', type: 'activity', position: { x: 250, y: 100 }, data: { label: 'Generate Report' } },
  { id: 'node-2', type: 'approval', position: { x: 450, y: 250 }, data: { label: 'Manager Approval', sla: '24h' } },
  { id: 'node-3', type: 'decision', position: { x: 700, y: 100 }, data: { label: 'Check Amount' } },
  { id: 'node-4', type: 'event', position: { x: 250, y: 400 }, data: { label: 'Notify Requester' } },
  { id: 'end', type: 'end', position: { x: 900, y: 250 }, data: { label: 'End' } },
];

const initialEdges: Edge[] = [
  { id: 'e1', source: 'start', target: 'node-1' },
  { id: 'e2', source: 'node-1', target: 'node-2' },
  { id: 'e3', source: 'node-2', target: 'node-3' },
  { id: 'e4', source: 'node-3', target: 'end', label: '<= $1000' },
  { id: 'e5', source: 'node-3', target: 'node-2', label: '> $1000', type: 'step' }, // Loop back example
  { id: 'e6', source: 'node-4', target: 'end' },
];

export const WorkflowDesignerPage: React.FC = () => {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  const onConnect = useCallback((params: Connection) => setEdges((eds) => addEdge(params, eds)), [setEdges]);

  const onNodeSelect = useCallback((id: string | null) => {
    setSelectedNodeId(id);
  }, []);

  const selectedNode = nodes.find((n) => n.id === selectedNodeId) || null;

  const handleSave = useCallback(async (status: string) => {
    const template = {
      name: "New Process", // TODO: Add name input
      description: "Created via Designer",
      status,
      steps: nodes.map(n => ({
        id: n.id,
        name: n.data.label,
        type: n.type,
        // Map other fields from data
        sla: n.data.sla,
        role: n.data.role,
        activity_ref: n.data.activityRef
      })),
      transitions: edges.map(e => ({
        from: e.source,
        to: e.target
      })),
      audit: { hash_chain: true, policy_refs: [] }
    };

    devDebug("Saving Process:", template);
    // TODO: Call API
    // await fetch('/api/bp?tenant_id=...&tenant_instance_id=...', { method: 'POST', body: JSON.stringify(template) });
  }, [nodes, edges]);

  return (
    <ReactFlowProvider>
      <div className="flex h-screen w-full flex-col font-display bg-background-light dark:bg-background-dark text-text-light-primary dark:text-text-dark-primary">
        {/* Header */}
        <header className="flex h-16 shrink-0 items-center justify-between border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-[#18232f] px-4">
          <div className="flex items-center gap-4">
            <span className="material-symbols-outlined text-primary text-3xl">hub</span>
            <h1 className="text-lg font-bold">Process Designer</h1>
          </div>
          <div className="flex items-center gap-4 rounded-lg bg-background-light dark:bg-background-dark p-1">
            <button onClick={() => handleSave('Draft')} className="rounded-md px-3 py-1 text-sm font-semibold bg-primary text-white shadow-sm">Draft</button>
            <button onClick={() => handleSave('Active')} className="rounded-md px-3 py-1 text-sm font-medium text-gray-500 dark:text-gray-400 hover:bg-primary/10">Published</button>
            <button className="rounded-md px-3 py-1 text-sm font-medium text-gray-500 dark:text-gray-400 hover:bg-primary/10">Deprecated</button>
          </div>
          <div className="flex items-center gap-3">
            <button className="flex items-center gap-2 rounded-md border border-gray-200 dark:border-gray-700 px-3 py-1.5 text-sm font-medium hover:bg-background-light dark:hover:bg-background-dark">
              <span className="material-symbols-outlined text-lg">difference</span>
              <span>Diff Viewer</span>
            </button>
            <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" style={{ backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuBGsSxCVTbLmWUj9Sfs2vR9MQJ2HIP00YDX5Bs2q6XgQYacBcpI27BsBnRfoSor7u8N7_yB76HebAxkVKgHEpsu4iQ4J5X3Sn3ePaxP53m4T-aAtES0Q2P2IUmjBpOGisA9Fs08ZSSkgiHDESTIiFy_SSEzZQvUdfIhnMesLDBesw8pluEyAJSzDgdrv_XZOj8A96y9NgqxusuGszGi85CXAu6ecTKKiV-YseEoGDwCLt-NsUrHWY7YrNkU9aQ5MpOO072JUVDI8-Al")' }}></div>
          </div>
        </header>

        <main className="flex min-h-0 flex-1">
          {/* Canvas Area */}
          <div className="flex flex-1 relative">
               <DesignerCanvas 
                 nodes={nodes} 
                 edges={edges} 
                 onNodesChange={onNodesChange} 
                 onEdgesChange={onEdgesChange} 
                 onConnect={onConnect}
                 onNodeSelect={onNodeSelect}
                 setNodes={setNodes}
               />
          </div>

          {/* Properties Panel */}
          <aside className="flex w-96 shrink-0 flex-col border-l border-gray-200 dark:border-gray-800 bg-white dark:bg-[#18232f]">
             <PropertiesPanel selectedNode={selectedNode} />
          </aside>
        </main>
      </div>
    </ReactFlowProvider>
  );
};