import { useCallback, useMemo } from 'react';
import ReactFlow, {
  Node,
  Edge,
  addEdge,
  Connection,
  useNodesState,
  useEdgesState,
  Controls,
  Background,
  MiniMap,
} from 'reactflow';
import 'reactflow/dist/style.css';

interface APIEndpoint {
  id: string;
  path: string;
  method: string;
  description: string;
  category: string;
  service: string;
  businessTerms: string[];
  dependencies: string[];
}

interface BusinessTerm {
  id: string;
  name: string;
  description: string;
  category: string;
  relatedAPIs: string[];
}

interface LineageGraphProps {
  apis: APIEndpoint[];
  businessTerms: BusinessTerm[];
  onNodeClick: (node: { id: string; type: 'api' | 'business_term' }) => void;
}

const LineageGraph: React.FC<LineageGraphProps> = ({ apis, businessTerms, onNodeClick }) => {
  const initialNodes: Node[] = useMemo(() => {
    const nodes: Node[] = [];

    // Add API nodes
    apis.forEach((api, index) => {
      nodes.push({
        id: api.id,
        type: 'default',
        position: { x: index * 300, y: 100 },
        data: {
          label: `${api.method} ${api.path}`,
          type: 'api',
          api
        },
        style: {
          background: '#e3f2fd',
          border: '2px solid #2196f3',
          borderRadius: '8px',
          padding: '10px',
        },
      });
    });

    // Add business term nodes
    businessTerms.forEach((term, index) => {
      nodes.push({
        id: term.id,
        type: 'default',
        position: { x: index * 300, y: 400 },
        data: {
          label: term.name,
          type: 'business_term',
          term
        },
        style: {
          background: '#f3e5f5',
          border: '2px solid #9c27b0',
          borderRadius: '8px',
          padding: '10px',
        },
      });
    });

    return nodes;
  }, [apis, businessTerms]);

  const initialEdges: Edge[] = useMemo(() => {
    const edges: Edge[] = [];

    // Create edges between APIs and business terms
    apis.forEach((api) => {
      api.businessTerms.forEach((termId) => {
        const term = businessTerms.find(t => t.id === termId);
        if (term) {
          edges.push({
            id: `${api.id}-${termId}`,
            source: api.id,
            target: termId,
            type: 'smoothstep',
            animated: true,
            style: { stroke: '#2196f3', strokeWidth: 2 },
            label: 'uses',
          });
        }
      });
    });

    // Create edges between APIs and their dependencies
    apis.forEach((api) => {
      api.dependencies.forEach((dep, index) => {
        const depNodeId = `dep-${dep}`;
        // Add dependency node if it doesn't exist
        if (!initialNodes.find(n => n.id === depNodeId)) {
          initialNodes.push({
            id: depNodeId,
            type: 'default',
            position: { x: (apis.indexOf(api) * 300) + (index * 100), y: 250 },
            data: {
              label: dep,
              type: 'dependency'
            },
            style: {
              background: '#fff3e0',
              border: '2px solid #ff9800',
              borderRadius: '8px',
              padding: '10px',
            },
          });
        }

        edges.push({
          id: `${api.id}-dep-${dep}`,
          source: api.id,
          target: depNodeId,
          type: 'smoothstep',
          style: { stroke: '#ff9800', strokeWidth: 2 },
          label: 'depends on',
        });
      });
    });

    return edges;
  }, [apis, businessTerms, initialNodes]);

  const [nodes, _setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  const handleNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node) => {
      if (node.data.type === 'api' || node.data.type === 'business_term') {
        onNodeClick({ id: node.id, type: node.data.type });
      }
    },
    [onNodeClick]
  );

  return (
    <div className="w-full h-full border border-gray-300 rounded-lg">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={handleNodeClick}
        fitView
        attributionPosition="top-right"
      >
        <Controls />
        <Background />
        <MiniMap />
      </ReactFlow>
    </div>
  );
};

export { LineageGraph };