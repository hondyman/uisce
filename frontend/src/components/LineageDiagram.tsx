import { useState, useEffect } from 'react';
import { devError } from './utils/devLogger';
import LazyReactFlow, { ReactFlowFallback, LazyReactFlowSubcomponents } from './ReactFlowLoader';
import './LineageDiagram.css';
import ParentNode from './ParentNode';
import DefaultNode from './DefaultNode';
import CustomEdge from './CustomEdge'; // Import CustomEdge

const nodeTypes = {
  parentNode: ParentNode,
  default: DefaultNode,
};

const edgeTypes = {
  customEdge: CustomEdge, // Add the custom edge type
};

interface LineageDiagramProps {
  subjectIds?: string[];
  nodes?: any[];
  edges?: any[];
}

const LineageDiagram: React.FC<LineageDiagramProps> = ({ subjectIds, nodes: initialNodes, edges: initialEdges }) => {
  const [nodes, setNodes] = useState(initialNodes || []);
  const [edges, setEdges] = useState(initialEdges || []);

  useEffect(() => {
    if (initialNodes) {
      setNodes(initialNodes);
    }
    if (initialEdges) {
      setEdges(initialEdges);
    }
  }, [initialNodes, initialEdges]);

  useEffect(() => {
    // Fetch data from the backend
    const fetchData = async () => {
      try {
  const apiBase = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8000';
  const response = await fetch(`${apiBase.replace(/\/$/, '')}/api/lineage`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            input: {
              subject_ids: subjectIds,
            },
          }),
        });

        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        // Update state with the fetched data
        setNodes(data.nodes || []);
        setEdges(data.edges || []);
      } catch (error) {
        devError('Error fetching lineage data:', error);
      }
    };

    if (subjectIds && subjectIds.length > 0) {
      fetchData();
    }
  }, [subjectIds]);

  return (
  <>
      <div className="lineage-diagram-root">
        <LazyReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          fitView
          className="lineage-flow"
        />

        <ReactFlowFallback>
          <LazyReactFlowSubcomponents>
            {({ MiniMap, Controls, Background }: any) => (
              <>
                <MiniMap position="top-left" />
                <Controls position="top-left" />
                <Background />
              </>
            )}
          </LazyReactFlowSubcomponents>
        </ReactFlowFallback>
      </div>
  </>
  );
};

export default LineageDiagram;
