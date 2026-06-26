// /Users/eganpj/GitHub/semlayer/frontend/src/pages/TabbedModal/Catalog/technicalLineageLayout.ts
import { Edge } from 'reactflow';
import { EnhancedSelectedAsset, TechnicalLineageData } from '../../../types/SemanticTypes';
import { LINEAGE_LAYOUT, createLineagePosition, getLineageNodeStyle } from './lineageUtils';
import { devDebug } from '../../../utils/devLogger';

export const buildTechnicalLineageLayout = (
  selectedAsset: EnhancedSelectedAsset | null,
  technicalData: TechnicalLineageData | null | undefined
) => {
  if (!selectedAsset || !technicalData ||
    (selectedAsset.type !== 'table' && selectedAsset.type !== 'column')) {
    return { nodes: [], edges: [] };
  }

  const selectedTable = technicalData.nodes.find((n: { id: string; data?: { tableName?: string; label?: string } }) =>
    n.id === selectedAsset.nodeId ||
    (selectedAsset.name && (n.data?.tableName === selectedAsset.name || n.data?.label === selectedAsset.name))
  );

  if (!selectedTable) return { nodes: [], edges: [] };

  let relatedEdges = technicalData.edges.filter((edge: Edge) =>
    (edge.source === selectedTable.id || edge.target === selectedTable.id)
  );

  const selfReferencingEdges = relatedEdges.filter((edge: Edge) =>
    edge.source === selectedTable.id && edge.target === selectedTable.id
  );

  // Exclude self-referencing edges from normal up/downstream processing
  relatedEdges = relatedEdges.filter((edge: Edge) =>
    !(edge.source === selectedTable.id && edge.target === selectedTable.id)
  );

  if (relatedEdges.length === 0 && selfReferencingEdges.length === 0) {
    return { nodes: [], edges: [] };
  }

  const layoutNodes: any[] = [];
  const layoutEdges: any[] = [];
  const centerPosition = createLineagePosition(1, 0, 1);

  // Add center node
  layoutNodes.push({
    id: selectedTable.id,
    type: 'hoverableNode',
    position: centerPosition,
    data: {
      label: selectedTable.data?.label || selectedAsset.name,
      nodeType: 'table',
      isCenter: true,
      style: getLineageNodeStyle('table', true),
      width: LINEAGE_LAYOUT.nodeWidth,
      height: LINEAGE_LAYOUT.nodeHeight,
    },
  });

  // Add self-referencing edges with labels
  selfReferencingEdges.forEach((edge: Edge, index: number) => {
    const edgeLabel = edge.data?.relationship_type || edge.label || 'Self Reference';
    layoutEdges.push({
      id: `self-${edge.id}`,
      type: 'selfReferenceEdge',
      source: selectedTable.id,
      target: selectedTable.id,
      label: edgeLabel,
      labelStyle: { fill: '#6b7280', fontWeight: 500, fontSize: 12 },
      labelBgStyle: { fill: '#ffffff', fillOpacity: 0.9 },
      labelBgPadding: [8, 4],
      labelBgBorderRadius: 4,
      data: {
        ...edge.data,
        label: edgeLabel,
        offset: 60 + index * 20,
      },
    });
  });

  // Add upstream and downstream nodes
  const upstreamEdges = relatedEdges.filter((e: Edge) => e.target === selectedTable.id);
  upstreamEdges.forEach((edge: Edge, index: number) => {
    const sourceTable = technicalData.nodes.find((n: { id: string; data?: { label?: string } }) => n.id === edge.source);
    if (sourceTable) {
      const position = createLineagePosition(0, index, upstreamEdges.length);
      const nodeId = `upstream-${sourceTable.id}`;
      layoutNodes.push({
        id: nodeId,
        type: 'hoverableNode',
        position,
        data: {
          label: sourceTable.data?.label,
          nodeType: 'table',
          direction: 'upstream',
          style: getLineageNodeStyle('table', false),
          width: LINEAGE_LAYOUT.nodeWidth,
          height: LINEAGE_LAYOUT.nodeHeight,
        },
      });

      const edgeLabel = edge.data?.relationship_type || edge.label || 'references';
      layoutEdges.push({
        id: `upstream-${edge.id}`,
        source: nodeId,
        target: selectedTable.id,
        type: 'smoothstep',
        label: edgeLabel,
        labelStyle: { fill: '#059669', fontWeight: 600, fontSize: 13 },
        labelBgStyle: { fill: '#ffffff', fillOpacity: 0.95 },
        labelBgPadding: [8, 4],
        labelBgBorderRadius: 4,
        animated: false,
        style: { stroke: '#059669', strokeWidth: 2 },
        markerEnd: {
          type: 'arrowclosed',
          color: '#059669',
          width: 20,
          height: 20,
        },
        data: edge.data
      });
    }
  });

  const downstreamEdges = relatedEdges.filter((e: Edge) => e.source === selectedTable.id);
  downstreamEdges.forEach((edge: Edge, index: number) => {
    const targetTable = technicalData.nodes.find((n: { id: string; data?: { label?: string } }) => n.id === edge.target);
    if (targetTable) {
      const position = createLineagePosition(2, index, downstreamEdges.length);
      const nodeId = `downstream-${targetTable.id}`;
      layoutNodes.push({
        id: nodeId,
        type: 'hoverableNode',
        position,
        data: {
          label: targetTable.data?.label,
          nodeType: 'table',
          direction: 'downstream',
          style: getLineageNodeStyle('table', false),
          width: LINEAGE_LAYOUT.nodeWidth,
          height: LINEAGE_LAYOUT.nodeHeight,
        },
      });

      const edgeLabel = edge.data?.relationship_type || edge.label || 'referenced by';
      layoutEdges.push({
        id: `downstream-${edge.id}`,
        source: selectedTable.id,
        target: nodeId,
        type: 'smoothstep',
        label: edgeLabel,
        labelStyle: { fill: '#2563eb', fontWeight: 600, fontSize: 13 },
        labelBgStyle: { fill: '#ffffff', fillOpacity: 0.95 },
        labelBgPadding: [8, 4],
        labelBgBorderRadius: 4,
        animated: false,
        style: { stroke: '#2563eb', strokeWidth: 2 },
        markerEnd: {
          type: 'arrowclosed',
          color: '#2563eb',
          width: 20,
          height: 20,
        },
        data: edge.data
      });
    }
  });

  devDebug('Layout Nodes:', layoutNodes.length);
  devDebug('Layout Edges:', layoutEdges.length);

  return { nodes: layoutNodes, edges: layoutEdges };
};