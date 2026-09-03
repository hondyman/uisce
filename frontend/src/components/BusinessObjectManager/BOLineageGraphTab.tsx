import React, { useEffect, useState } from 'react';
import { Box, CircularProgress, Alert, Typography } from '@mui/material';

import { CatalogGraph } from '../graph/CatalogGraph';
import { CatalogResponseGraph, CatalogResponseNode } from '../../api/viewDefinitions';
import { BONode } from './CustomNodes/BONode';
import { TermNode } from './CustomNodes/TermNode';
import { CalculationNode } from './CustomNodes/CalculationNode';
import { TableNode } from './CustomNodes/TableNode';
import { ColumnNode } from './CustomNodes/ColumnNode';
import { NodeDetailDrawer } from './NodeDetailDrawer';
import { GraphLegend } from './GraphLegend';

const nodeTypeOverrides = {
  bo: BONode,
  term: TermNode,
  calculation: CalculationNode,
  table: TableNode,
  column: ColumnNode,
  related_bo: BONode, // Reuse BO node with different styling
};

const nodeDimensions: Record<string, { width: number; height: number }> = {
  bo: { width: 320, height: 160 },
  related_bo: { width: 320, height: 160 },
  calculation: { width: 220, height: 100 },
  term: { width: 200, height: 90 },
  table: { width: 180, height: 80 },
  column: { width: 160, height: 70 },
};

function getNodeDimensions(type: string) {
  return nodeDimensions[type] || { width: 250, height: 120 };
}

function getEdgeColor(type: string) {
  switch (type) {
    case 'contains':
      return '#1976d2'; // Primary blue
    case 'maps_to':
      return '#2e7d32'; // Green
    case 'belongs_to':
      return '#757575'; // Grey
    case 'relates_to':
      return '#ed6c02'; // Orange
    case 'uses':
      return '#9c27b0'; // Purple
    case 'joins_via':
      return '#d32f2f'; // Red
    default:
      return '#666';
  }
}

interface BOLineageGraphTabProps {
  boId: string;
}

// Attaches this BO's semantic terms as data on the 'bo' node so BONode can
// group and render them, and condenses the raw /api/bo/{boId}/graph response
// to just BO <-> related-BO edges — matching the original BOLineageGraphTab
// behavior before this was ported onto the shared CatalogGraph renderer.
function condenseBoGraph(raw: { nodes: any[]; edges: any[] }): CatalogResponseGraph {
  const boTerms: any[] = [];
  (raw.nodes || []).forEach((n) => {
    if (n.type === 'term') {
      boTerms.push({
        id: n.id,
        nodeName: n.data?.termName || n.label,
        termType: n.data?.termType,
        dataType: n.data?.dataType,
        isKey: n.data?.isKey,
        subtypeId: n.data?.subtypeId,
        subtypeName: n.data?.subtypeName,
      });
    }
  });

  const condensedNodes = (raw.nodes || [])
    .filter((n) => n.type === 'bo' || n.type === 'related_bo')
    .map((n) => ({
      id: n.id,
      type: n.type,
      label: n.label,
      properties: {
        ...n.data,
        terms: n.type === 'bo' ? boTerms : [],
        termCount: n.type === 'bo' ? boTerms.length : n.data?.termCount || 0,
      },
    }));

  const condensedEdges = (raw.edges || [])
    .filter((e: any) => e.source.startsWith('BO:') && e.target.startsWith('BO:'))
    .map((e: any) => ({ id: e.id, source: e.source, target: e.target, type: e.type }));

  if (condensedNodes.length > 0) {
    return { nodes: condensedNodes, edges: condensedEdges };
  }

  // Fallback: no condensable BO nodes found, render the raw graph as-is.
  return {
    nodes: (raw.nodes || []).map((n: any) => ({ id: n.id, type: n.type, label: n.label, properties: n.data || {} })),
    edges: (raw.edges || []).map((e: any) => ({ id: e.id, source: e.source, target: e.target, type: e.type })),
  };
}

export const BOLineageGraphTab: React.FC<BOLineageGraphTabProps> = ({ boId }) => {
  const [graphData, setGraphData] = useState<CatalogResponseGraph | null>(null);
  const [selectedNode, setSelectedNode] = useState<CatalogResponseNode | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const fetchGraph = async () => {
      setLoading(true);
      setError(null);
      try {
        const response = await fetch(`/api/bo/${boId}/graph`);
        if (!response.ok) {
          throw new Error(`Failed to fetch graph: ${response.statusText}`);
        }
        const data = await response.json();
        if (!cancelled) {
          setGraphData(condenseBoGraph(data));
        }
      } catch (err) {
        console.error('Failed to fetch graph:', err);
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Unknown error');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    fetchGraph();
    return () => {
      cancelled = true;
    };
  }, [boId]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '600px' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">
          <Typography variant="h6">Failed to load graph</Typography>
          <Typography variant="body2">{error}</Typography>
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ width: '100%', height: '800px', position: 'relative' }}>
      <CatalogGraph
        graphData={graphData || { nodes: [], edges: [] }}
        layout={{ algorithm: 'dagre', direction: 'TB' }}
        nodeTypeOverrides={nodeTypeOverrides}
        getNodeDimensions={getNodeDimensions}
        edgeColor={(edge) => getEdgeColor(edge.type)}
        showMiniMap
        legend={<GraphLegend />}
        onNodeSelect={setSelectedNode}
      />

      <NodeDetailDrawer
        // NodeDetailDrawer reads node.data.*; CatalogResponseNode carries the
        // same fields under .properties, so adapt the shape at the boundary.
        node={
          selectedNode
            ? ({ id: selectedNode.id, type: selectedNode.type, data: selectedNode.properties, position: { x: 0, y: 0 } } as any)
            : null
        }
        open={!!selectedNode}
        onClose={() => setSelectedNode(null)}
        boId={boId}
      />
    </Box>
  );
};
