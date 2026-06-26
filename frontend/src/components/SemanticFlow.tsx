import React, { useMemo, useCallback } from 'react';
import ReactFlow, { Background, Controls, Node, Edge, NodeProps, MarkerType, Handle, Position } from 'reactflow';
import { Tooltip } from '@mui/material';
import dagre from 'dagre';
import 'reactflow/dist/style.css';
import './SemanticFlow.css';

// Dynamic color mapping based on node type
const getNodeTypeColor = (nodeType: string): { bg: string; border: string; text: string } => {
  const colorMap: Record<string, { bg: string; border: string; text: string }> = {
    'business_object': { bg: '#DBEAFE', border: '#1E40AF', text: '#001F3F' }, // Blue
    'business_term': { bg: '#DBEAFE', border: '#1E40AF', text: '#001F3F' }, // Blue
    'semantic_term': { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' }, // Purple
    'semantic_model': { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' }, // Purple
    'semantic_view': { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' }, // Purple
    'semantic_column': { bg: '#FED7AA', border: '#92400E', text: '#3F2305' }, // Orange
    'database_column': { bg: '#DCFCE7', border: '#15803D', text: '#052E16' }, // Green
    'db_column': { bg: '#DCFCE7', border: '#15803D', text: '#052E16' }, // Green
    'column': { bg: '#DCFCE7', border: '#15803D', text: '#052E16' }, // Green
    'table': { bg: '#F3E8FF', border: '#7E22CE', text: '#3F0F5C' }, // Purple-pink
    'schema': { bg: '#FCE7F3', border: '#BE185D', text: '#500724' }, // Pink
    'database': { bg: '#FEE2E2', border: '#DC2626', text: '#4C0519' }, // Red
    'bo_field': { bg: '#DBEAFE', border: '#0284C7', text: '#001F3F' }, // Light Blue
  };
  return colorMap[nodeType] || { bg: '#F3F4F6', border: '#9CA3AF', text: '#374151' }; // Default gray
};

// Single dynamic node component that applies colors based on node type
const DynamicNode = ({ data }: NodeProps) => {
  console.log('[DynamicNode] nodeType:', data.nodeType, 'colors:', getNodeTypeColor(data.nodeType || 'business_term'));
  const nodeColors = getNodeTypeColor(data.nodeType || 'business_term');
  
  return (
    <Tooltip 
      title={
        <div>
          <div><strong>Name:</strong> {data.name}</div>
          <div><strong>Type:</strong> {data.nodeType}</div>
          <div><strong>Description:</strong> {data.description}</div>
        </div>
      }
      arrow
      placement="top"
    >
      <div 
        className="custom-node"
        style={{
          backgroundColor: nodeColors.bg,
          borderColor: nodeColors.border,
          color: nodeColors.text,
          border: `2px solid ${nodeColors.border}`,
        }}
      >
        <Handle type="target" position={Position.Top} />
        <div className="node-label">{data.label}</div>
        <Handle type="source" position={Position.Bottom} />
      </div>
    </Tooltip>
  );
};

// Node type definitions - all use the same dynamic component
const nodeTypes = {
  businessTerm: DynamicNode,
  semanticTerm: DynamicNode,
  semanticColumn: DynamicNode,
  databaseTable: DynamicNode,
  databaseColumn: DynamicNode,
  dynamic: DynamicNode,
};

// Node type ID to node type string mapping (for color assignment)
const getNodeTypeString = (nodeTypeId: string): string => {
  const mapped = (() => {
    switch (nodeTypeId) {
      case '21645d21-de5f-4feb-af99-99273ea75626':
        return 'business_term'; // Business Terms
      case '820b942a-9c9e-4abc-acdc-84616db33098':
        return 'semantic_term'; // Semantic Terms
      case '1439f761-606a-44cb-b4f8-7aa6b27a9bf5':
        return 'semantic_column'; // Semantic Columns
      case '49a50271-ae58-4d3e-ae1c-2f5b89d89192':
        return 'table'; // Tables
      case 'a64c1011-16e8-4ddf-b447-363bf8e15c9a':
        return 'column'; // Columns
      default:
        return 'semantic_term'; // Default fallback
    }
  })();
  console.log('[getNodeTypeString]', nodeTypeId, '->', mapped);
  return mapped;
};

// Node type ID to human-readable name mapping
const getNodeTypeName = (nodeTypeId: string): string => {
  switch (nodeTypeId) {
    case '21645d21-de5f-4feb-af99-99273ea75626':
      return 'Business Term';
    case '820b942a-9c9e-4abc-acdc-84616db33098':
      return 'Semantic Term';
    case '1439f761-606a-44cb-b4f8-7aa6b27a9bf5':
      return 'Semantic Column';
    case '49a50271-ae58-4d3e-ae1c-2f5b89d89192':
      return 'Database Table';
    case 'a64c1011-16e8-4ddf-b447-363bf8e15c9a':
      return 'Database Column';
    default:
      return 'Unknown Type';
  }
};

type NodeType = Node<any>;
type EdgeType = Edge<any>;

const SemanticFlow: React.FC<{
  centerNode: any;
  allNodes: any[];
  semanticEdges: any[];
  onNodeClick: (node: any) => void;
}> = ({ centerNode, allNodes, semanticEdges, onNodeClick }) => {
  // compute linked ids (terms directly connected to the center node)
  const linkedIds = useMemo(() => {
    const ids = new Set<string>();
    
    // 1. Add connections from explicit edges
    semanticEdges.forEach((e: any) => {
      if (e.source_node_id === centerNode.id) ids.add(e.target_node_id);
      if (e.target_node_id === centerNode.id) ids.add(e.source_node_id);
    });

    // 2. Add connections from parent_id (implicit edges)
    // If centerNode has a parent, add the parent
    if (centerNode.parent_id) {
      ids.add(centerNode.parent_id);
    }

    // If any node has centerNode as parent, add that child
    allNodes.forEach((node: any) => {
      if (node.parent_id === centerNode.id) {
        ids.add(node.id);
      }
    });

    return Array.from(ids);
  }, [centerNode, semanticEdges, allNodes]);

  const linkedNodes = useMemo(() => {
    return allNodes.filter(node => linkedIds.includes(node.id));
  }, [allNodes, linkedIds]);

  // create React Flow nodes from linked nodes - positions will be computed by dagre layout
  const nodes: NodeType[] = useMemo(() => {
    console.log('[SemanticFlow] linkedNodes sample:', linkedNodes[0]);
    console.log('[SemanticFlow] linkedNodes has node_type_id?', linkedNodes.map(n => ({ id: n.id, node_type_id: n.node_type_id, node_name: n.node_name })));
    return linkedNodes.map((node: any) => ({ 
      id: node.id, 
      data: { 
        label: node.node_name, 
        raw: node,
        name: node.node_name,
        nodeType: getNodeTypeString(node.node_type_id),
        description: node.description || 'No description available'
      }, 
      position: { x: 0, y: 0 }, 
      type: 'dynamic'
    }));
  }, [linkedNodes]);

  const edges: EdgeType[] = useMemo(() => {
    const linkedSet = new Set(linkedNodes.map((node: any) => node.id));
    const result: EdgeType[] = [];
    semanticEdges.forEach((e: any, idx: number) => {
      if (linkedSet.has(e.source_node_id) && linkedSet.has(e.target_node_id)) {
        result.push({ id: `e${idx}`, source: e.source_node_id, target: e.target_node_id, animated: true, label: e.relationship_type || e.label || '' });
      }
    });
    return result;
  }, [semanticEdges, linkedNodes]);

  // create central node
  const centerFlowNode: NodeType = useMemo(() => ({
    id: `center-${centerNode.id}`,
    data: { 
      label: centerNode.node_name || centerNode.name || 'Center Node', 
      raw: centerNode,
      name: centerNode.node_name || centerNode.name || 'Center Node',
      nodeType: getNodeTypeString(centerNode.node_type_id),
      description: centerNode.description || 'No description available'
    },
    position: { x: 0, y: 0 },
    type: 'dynamic',
  }), [centerNode]);

  // edges connecting center node to linked nodes
  const centerEdges: EdgeType[] = useMemo(() => {
    return linkedNodes.map((node: any) => {
      // 1. Check for explicit edge
      const edge = semanticEdges.find((e: any) => 
        (e.source_node_id === centerNode.id && e.target_node_id === node.id) ||
        (e.target_node_id === centerNode.id && e.source_node_id === node.id)
      );

      let label = 'related';
      let isExplicit = false;

      if (edge) {
        isExplicit = true;
        if (edge.relationship_type === 'business_term_to_semantic_term') {
           if (edge.source_node_id === centerNode.id) {
             label = 'parent of';
           } else {
             label = 'child of';
           }
        } else {
           label = edge.relationship_type || 'related';
        }
      }

      // 2. If no explicit edge, check parent_id relationship
      if (!isExplicit) {
        if (centerNode.parent_id === node.id) {
          label = 'child of'; // centerNode is child of node
        } else if (node.parent_id === centerNode.id) {
          label = 'parent of'; // centerNode is parent of node
        }
      }

      return { 
        id: `c-${node.id}`, 
        source: `center-${centerNode.id}`, 
        target: node.id, 
        animated: false, 
        label: label,
        type: 'default',
        markerEnd: {
          type: MarkerType.ArrowClosed,
        },
      };
    });
  }, [centerNode, linkedNodes, semanticEdges]);

  const flowNodes = [centerFlowNode, ...nodes];
  const flowEdges = [...edges, ...centerEdges];

  // dagre layout to compute nice positions
  const getLayoutedElements = useCallback((ns: NodeType[], es: EdgeType[]) => {
    const g = new dagre.graphlib.Graph();
    g.setDefaultEdgeLabel(() => ({}));
    g.setGraph({ rankdir: 'TB', nodesep: 40, ranksep: 60 });

    // Separate center node from other nodes for special positioning
    const centerNodeData = ns.find(n => n.id.startsWith('center-'));
    const otherNodes = ns.filter(n => !n.id.startsWith('center-'));

    otherNodes.forEach((n) => {
      // estimate width/height
      g.setNode(n.id, { width: 160, height: 40 });
    });

    // Only add edges between non-center nodes to dagre
    es.filter(e => !e.source.startsWith('center-') && !e.target.startsWith('center-')).forEach((e) => {
      g.setEdge(e.source, e.target);
    });

    dagre.layout(g);

    const positionedNodes = otherNodes.map((n) => {
      const nodeWithPosition = g.node(n.id);
      return { ...n, position: { x: nodeWithPosition.x - 80, y: nodeWithPosition.y - 20 } } as NodeType;
    });

    // Position center node in the middle
    let centerPosition = { x: 0, y: 0 };
    if (positionedNodes.length > 0) {
      const avgX = positionedNodes.reduce((sum, n) => sum + n.position.x, 0) / positionedNodes.length;
      const avgY = positionedNodes.reduce((sum, n) => sum + n.position.y, 0) / positionedNodes.length;
      centerPosition = { x: avgX, y: avgY - 100 }; // Position above the average
    }

    if (centerNodeData) {
      positionedNodes.unshift({ ...centerNodeData, position: centerPosition });
    }

    return { nodes: positionedNodes, edges: es };
  }, []);

  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(() => getLayoutedElements(flowNodes, flowEdges), [flowNodes, flowEdges, getLayoutedElements]);

  return (
    <div className="semantic-flow-container">
      <ReactFlow
        nodes={layoutedNodes}
        edges={layoutedEdges}
        nodeTypes={nodeTypes}
        fitView
        attributionPosition="bottom-left"
        onNodeClick={(_evt, node) => {
          const clicked = allNodes.find(n => n.id === node.id) || (node.id === `center-${centerNode.id}` ? centerNode : null);
          if (clicked) onNodeClick(clicked);
        }}
        defaultViewport={{ x: 0, y: 0, zoom: 0.8 }}
      >
        <Background gap={16} />
        <Controls />
      </ReactFlow>
    </div>
  );
};

export default SemanticFlow;
