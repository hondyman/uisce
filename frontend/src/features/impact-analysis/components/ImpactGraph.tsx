import React, { useEffect, useState, useCallback, useMemo } from 'react';
import { Box, CircularProgress, IconButton, Tooltip } from '@mui/material';
import { createPortal } from 'react-dom';
import FullscreenIcon from '@mui/icons-material/Fullscreen';
import FullscreenExitIcon from '@mui/icons-material/FullscreenExit';
import MapIcon from '@mui/icons-material/Map';

import { impactApi } from '../api/impactApi';
import { NodeType, ImpactGraphData, ImpactNode } from '../types';
import { CatalogGraph } from '../../../components/graph/CatalogGraph';
import { ColorCodedNode, getNodeTypeColor } from '../../../components/graph/ColorCodedNode';
import { CatalogResponseGraph, CatalogResponseEdge } from '../../../api/viewDefinitions';

type DirectionMode = 'upstream' | 'downstream' | 'both';

interface ImpactGraphProps {
  nodeType: NodeType;
  nodeId: string;
  highlightedNodeIds?: string[];
  directionMode?: DirectionMode;
  onStatsUpdate?: (stats: { upstreamCount: number; downstreamCount: number; totalCount: number }) => void;
  useLineageAPI?: boolean; // If true, use /lineage/node/{id}/graph instead of /impact/graph
}

const COLOR_CODED_TYPE = 'colorCoded';
const nodeTypeOverrides = { [COLOR_CODED_TYPE]: ColorCodedNode };

function extractDirection(properties: Record<string, any> | undefined): string {
  if (!properties?.metadata) return 'unknown';
  try {
    const metadata = typeof properties.metadata === 'string' ? JSON.parse(properties.metadata) : properties.metadata;
    return metadata.direction || 'unknown';
  } catch {
    return 'unknown';
  }
}

// Ported onto the shared CatalogGraph renderer. Node types here are
// open-ended (any catalog type name, not a fixed enum), so rather than a
// per-type nodeRegistry entry this viewer tags every node with one synthetic
// 'colorCoded' type and lets ColorCodedNode derive its look from
// properties.realType — matching the original design, which used ReactFlow's
// bare default node with fully inline styling rather than per-type dispatch.
export const ImpactGraph: React.FC<ImpactGraphProps> = ({
  nodeType,
  nodeId,
  highlightedNodeIds = [],
  directionMode = 'both',
  onStatsUpdate,
  useLineageAPI = false,
}) => {
  const [rawGraph, setRawGraph] = useState<ImpactGraphData | null>(null);
  const [loading, setLoading] = useState(false);
  const [isFullScreen, setIsFullScreen] = useState(false);
  const [showMiniMap, setShowMiniMap] = useState(true);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const data: ImpactGraphData = useLineageAPI
        ? await impactApi.getLineageGraph(nodeId)
        : await impactApi.getGraph(nodeType, nodeId);
      setRawGraph(data);
    } catch (error) {
      console.error('Failed to fetch impact graph:', error);
    } finally {
      setLoading(false);
    }
  }, [nodeType, nodeId, useLineageAPI]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Direction-filtered view + stats, mirroring the original: stats are
  // computed over the FULL fetched graph regardless of directionMode, while
  // the rendered subset is filtered by it.
  const { graphData, stats } = useMemo(() => {
    const nodes: ImpactNode[] = rawGraph?.nodes || [];
    const edges: CatalogResponseEdge[] = rawGraph?.edges || [];

    const nodeDirection = (n: ImpactNode) => extractDirection(n.properties);

    const upstreamCount = nodes.filter((n) => n.id !== nodeId && ['upstream', 'both'].includes(nodeDirection(n))).length;
    const downstreamCount = nodes.filter((n) => n.id !== nodeId && ['downstream', 'both'].includes(nodeDirection(n))).length;

    let filteredNodes = nodes;
    let filteredEdges = edges;
    if (directionMode !== 'both' && nodes.length > 0) {
      filteredNodes = nodes.filter((n) => {
        if (n.id === nodeId) return true;
        const dir = nodeDirection(n);
        return dir === directionMode || dir === 'both' || dir === 'unknown';
      });
      const keptIds = new Set(filteredNodes.map((n) => n.id));
      filteredEdges = edges.filter((e) => keptIds.has(e.source) && keptIds.has(e.target));
    }

    const highlightSet = new Set(highlightedNodeIds);

    const graph: CatalogResponseGraph = {
      nodes: filteredNodes.map((n) => ({
        id: n.id,
        type: COLOR_CODED_TYPE,
        label: n.label,
        properties: {
          ...n.properties,
          realType: n.type,
          isRoot: n.id === nodeId,
          highlighted: highlightSet.has(n.id),
        },
      })),
      edges: filteredEdges,
    };

    return {
      graphData: graph,
      stats: { upstreamCount, downstreamCount, totalCount: Math.max(nodes.length - 1, 0) },
    };
  }, [rawGraph, directionMode, nodeId, highlightedNodeIds]);

  useEffect(() => {
    if (rawGraph && onStatsUpdate) {
      onStatsUpdate(stats);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stats, rawGraph]);

  const edgeColorByTargetType = useCallback(
    (edge: CatalogResponseEdge) => {
      const targetNode = (rawGraph?.nodes || []).find((n) => n.id === edge.target);
      return targetNode ? getNodeTypeColorBorder(targetNode.type) : '#b1b1b7';
    },
    [rawGraph]
  );

  const toggleFullScreen = () => setIsFullScreen(!isFullScreen);

  const graphContent = (
    <CatalogGraph
      graphData={graphData}
      layout={{ algorithm: 'dagre', direction: 'LR' }}
      nodeTypeOverrides={nodeTypeOverrides}
      getNodeDimensions={() => ({ width: 200, height: 50 })}
      edgeColor={edgeColorByTargetType}
      showMiniMap={showMiniMap}
      legend={
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Tooltip title={showMiniMap ? 'Hide Map' : 'Show Map'}>
            <IconButton onClick={() => setShowMiniMap(!showMiniMap)} size="small" sx={{ bgcolor: 'white', '&:hover': { bgcolor: '#f0f0f0' }, boxShadow: 2 }}>
              <MapIcon fontSize="small" color={showMiniMap ? 'primary' : 'inherit'} />
            </IconButton>
          </Tooltip>
          <Tooltip title={isFullScreen ? 'Exit Fullscreen' : 'Fullscreen'}>
            <IconButton onClick={toggleFullScreen} size="small" sx={{ bgcolor: 'white', '&:hover': { bgcolor: '#f0f0f0' }, boxShadow: 2 }}>
              {isFullScreen ? <FullscreenExitIcon fontSize="small" /> : <FullscreenIcon fontSize="small" />}
            </IconButton>
          </Tooltip>
        </Box>
      }
    />
  );

  if (loading && !rawGraph) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', minHeight: 400 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (isFullScreen) {
    return createPortal(<div className="impact-fullscreen-overlay">{graphContent}</div>, document.body);
  }

  return <Box sx={{ width: '100%', height: '100%', minHeight: 600 }}>{graphContent}</Box>;
};

// Re-derives just the border color for edge coloring (matches
// ColorCodedNode's own color table so edges visually agree with their
// target node) without duplicating the whole map inline here.
function getNodeTypeColorBorder(nodeType: string): string {
  return getNodeTypeColor(nodeType).border;
}
