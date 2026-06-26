import React, { useState, useCallback } from 'react';
import ReactFlow, {
  addEdge,
  useNodesState,
  useEdgesState,
  Node,
} from '../shims/reactflow';
import '../shims/reactflow.css';
import { useABAC } from '../hooks/useABAC';
import { useMutation } from '@tanstack/react-query';

const initialNodes: Node[] = [
  {
    id: '1',
    type: 'input',
    position: { x: 250, y: 25 },
    data: { label: 'Start Workflow' },
  },
];

const initialEdges: any[] = [];

export const WorkflowOrchestrator: React.FC = () => {
  const { evaluate } = useABAC();
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [workflowDescription, setWorkflowDescription] = useState('');
  const [isAISuggesting, setIsAISuggesting] = useState(false);

  // AI suggestion mutation
  const aiSuggestMutation = useMutation({
    mutationFn: async (description: string) => {
      // Check ABAC permission
      const canSuggest = await evaluate('suggest', 'workflow');
      if (!canSuggest) {
        throw new Error('ABAC denied: insufficient permissions to generate workflow suggestions');
      }

      const response = await fetch('http://localhost:8081/workflows/suggest', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          description,
          context: {
            industry: 'wealth-management',
            compliance: 'FINRA',
            user: 'advisor'
          }
        })
      });

      if (!response.ok) {
        throw new Error('Failed to get AI suggestions');
      }

      return response.json();
    },
    onSuccess: (data) => {
      if (data.suggestion?.elements) {
        // Convert AI suggestion elements to ReactFlow nodes
        const aiNodes: Node[] = data.suggestion.elements.map((element: any, index: number) => ({
          id: element.id || `ai-node-${index}`,
          type: element.type || 'default',
          position: element.position || { x: Math.random() * 400, y: Math.random() * 300 },
          data: { label: element.data?.label || element.label || element.type },
        }));

        setNodes(aiNodes);
        setEdges([]); // Clear edges for new workflow
      }
    },
    onError: (error) => {
      console.error('AI suggestion failed:', error);
      alert(`AI suggestion failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  });

  const onConnect = useCallback(
    (params: any) => setEdges((eds: any[]) => addEdge(params, eds)),
    [setEdges]
  );

  const handleAISuggest = () => {
    if (!workflowDescription.trim()) {
      alert('Please enter a workflow description');
      return;
    }
    setIsAISuggesting(true);
    aiSuggestMutation.mutate(workflowDescription, {
      onSettled: () => setIsAISuggesting(false)
    });
  };

  const addNode = (type: string) => {
    const newNode: Node = {
      id: `${nodes.length + 1}`,
      type,
      position: {
        x: Math.random() * 300 + 100,
        y: Math.random() * 200 + 100
      },
      data: { label: `${type} Node` },
    };
    setNodes((nds: Node[]) => [...nds, newNode]);
  };

  const clearWorkflow = () => {
    setNodes(initialNodes);
    setEdges([]);
    setWorkflowDescription('');
  };

  return (
    <div className="h-screen w-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-white p-4 shadow-sm border-b">
        <h2 className="text-xl font-semibold mb-3">AI Workflow Orchestrator</h2>

        <div className="flex gap-4 items-end">
          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Workflow Description
            </label>
            <textarea
              value={workflowDescription}
              onChange={(e) => setWorkflowDescription(e.target.value)}
              placeholder="Describe the workflow you want to create (e.g., 'Rebalance portfolio for high-net-worth client with ESG compliance')"
              className="w-full px-3 py-2 border rounded-md text-sm"
              rows={2}
            />
          </div>

          <div className="flex gap-2">
            <button
              onClick={handleAISuggest}
              disabled={isAISuggesting || !workflowDescription.trim()}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-gray-400 text-sm font-medium"
            >
              {isAISuggesting ? 'Generating...' : '🤖 AI Suggest'}
            </button>
            <button
              onClick={clearWorkflow}
              className="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700 text-sm"
            >
              Clear
            </button>
          </div>
        </div>
      </div>

      {/* Node Palette */}
      <div className="bg-white p-3 border-b">
        <div className="flex gap-2 flex-wrap">
          <button onClick={() => addNode('start')} className="px-3 py-1 bg-green-600 text-white rounded text-sm hover:bg-green-700">
            + Start
          </button>
          <button onClick={() => addNode('decision')} className="px-3 py-1 bg-yellow-600 text-white rounded text-sm hover:bg-yellow-700">
            + Decision
          </button>
          <button onClick={() => addNode('action')} className="px-3 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">
            + Action
          </button>
          <button onClick={() => addNode('condition')} className="px-3 py-1 bg-purple-600 text-white rounded text-sm hover:bg-purple-700">
            + Condition
          </button>
          <button onClick={() => addNode('end')} className="px-3 py-1 bg-red-600 text-white rounded text-sm hover:bg-red-700">
            + End
          </button>
        </div>
      </div>

      {/* ReactFlow Canvas */}
      <div className="flex-1 p-4">
        <div className="bg-white rounded-lg shadow-sm h-full">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
          />
        </div>
      </div>

      {/* Status Bar */}
      <div className="bg-white p-3 border-t text-sm text-gray-600">
        Nodes: {nodes.length} | Edges: {edges.length}
        {aiSuggestMutation.isSuccess && (
          <span className="ml-4 text-green-600">✓ AI suggestion applied</span>
        )}
        {aiSuggestMutation.isError && (
          <span className="ml-4 text-red-600">
            ✗ Error: {aiSuggestMutation.error instanceof Error ? aiSuggestMutation.error.message : 'Unknown error'}
          </span>
        )}
      </div>
    </div>
  );
};