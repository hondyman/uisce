import React, { useEffect, useState } from 'react';
import ReactFlow, {
  ReactFlowProvider,
  Background,
  Controls,
  MiniMap,
  Node,
  Edge,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { getClaimAwareLineage } from '../../../api';
import { ClaimAwareLineageGraphData, ClaimAwareLineageNode } from '../../../types';
import dagre from 'dagre';

// Custom node component
const CustomNode = ({ data }: { data: any }) => {
  const injected = `
    .claim-node { padding: 10px 20px; border-radius: 8px; border: 2px solid; text-align: center; width: 180px; }
    .claim-node .claim-node-type { font-size: 12px; color: #6b7280; }
    .claim-node.full { border-color: #16a34a; background-color: #f0fdf4; }
    .claim-node.partial { border-color: #f59e0b; background-color: #fefce8; }
    .claim-node.none { border-color: #dc2626; background-color: #fef2f2; opacity: 0.6; }
  `;

  return (
    <>
      <style dangerouslySetInnerHTML={{ __html: injected }} />
      <div className={`claim-node ${data.visibility || ''}`} title={data.reason}>
        <strong>{data.label}</strong>
        <div className="claim-node-type">{data.type}</div>
        {data.data?.certified && <span title="Certified">✅</span>}
      </div>
    </>
  );
};

const nodeTypes = {
  custom: CustomNode,
};

// Layouting function
const getLayoutedElements = (nodes: Node[], edges: Edge[]) => {
  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));
  dagreGraph.setGraph({ rankdir: 'TB', nodesep: 100, ranksep: 100 });

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: 180, height: 60 });
  });

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  dagre.layout(dagreGraph);

  const layoutedNodes = nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - 180 / 2,
        y: nodeWithPosition.y - 60 / 2,
      },
    };
  });

  return { nodes: layoutedNodes, edges };
};

interface ClaimAwareLineageViewerProps {
  assetId: string;
  userId: string;
}

const ClaimAwareLineageViewer: React.FC<ClaimAwareLineageViewerProps> = ({ assetId, userId }) => {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const data: ClaimAwareLineageGraphData = await getClaimAwareLineage(assetId, userId);
        
        const flowNodes: Node[] = data.nodes.map((n: ClaimAwareLineageNode) => ({
          id: n.id,
          type: 'custom',
          data: { label: n.label, type: n.type, visibility: n.visibility, reason: n.reason, data: n.data },
          position: { x: 0, y: 0 },
        }));

        const flowEdges: Edge[] = data.edges.map((e, i) => ({
          id: `e${i}`,
          source: e.source,
          target: e.target,
          animated: true,
          style: { stroke: '#6b7280' },
        }));

        const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(flowNodes, flowEdges);
        setNodes(layoutedNodes);
        setEdges(layoutedEdges);

      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load lineage data');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [assetId, userId]);

  if (loading) return <div className="lineage-loading">Loading lineage...</div>;
  if (error) return <div className="lineage-error">Error: {error}</div>;

  return (
    <div className="lineage-root">
      <ReactFlowProvider>
        <ReactFlow nodes={nodes} edges={edges} nodeTypes={nodeTypes} fitView>
          <Controls className="lineage-controls" />
          <MiniMap className="lineage-minimap" />
          <Background className="lineage-background" />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
};

export default ClaimAwareLineageViewer;