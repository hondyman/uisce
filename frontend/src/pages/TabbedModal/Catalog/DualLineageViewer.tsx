import React, { useMemo, useState, useEffect, Suspense } from 'react';
import { Node as FlowNode, Edge } from 'reactflow';
import { devLog, devDebug } from "../../../utils/devLogger";
import { createPortal } from 'react-dom';
import './DualLineageViewer.css';
import { EnhancedSelectedAsset, TechnicalLineageData, SemanticLineageData } from '../../../types/SemanticTypes';
// Lazy-load heavy subcomponents to move ReactFlow and related code into separate chunks
const LineageFlow = React.lazy(() => import('./LineageFlow')) as unknown as React.ComponentType<Record<string, unknown>>;
const LineageTypeSelector = React.lazy(() => import('./LineageTypeSelector')) as unknown as React.ComponentType<Record<string, unknown>>;
const DualLineageDetailsPanel = React.lazy(() => import('./DetailsPane')) as unknown as React.ComponentType<Record<string, unknown>>;
import HoverableNode from '../../../components/HoverableNode';
import SelfReferenceEdge from '../../../components/SelfReferenceEdge';
import HierarchicalLineageNode from './HierarchicalLineageNode';
import { hierarchicalNodeTypes } from '../tabs/ContainerNodes';
import { useHierarchicalLayout, calculateHierarchicalPositions, updateNodeExpansion, HierarchicalData } from '../tabs/useHierarchicalLayout';
import { buildTechnicalLineageLayout } from './technicalLineageLayout';
import { buildSemanticLineageLayout } from './semanticLayoutBuilder';

// Combined node types supporting both regular and hierarchical nodes
const nodeTypes = {
  hoverableNode: HoverableNode,
  hierarchicalLineageNode: HierarchicalLineageNode,
  ...hierarchicalNodeTypes,
};

const edgeTypes = {
  selfReferenceEdge: SelfReferenceEdge,
};

interface DualLineageViewerProps {
  selectedAsset: EnhancedSelectedAsset | null;
  technicalData?: TechnicalLineageData | null;
  semanticData?: SemanticLineageData | null;
    hierarchicalData?: HierarchicalData | null; // New prop for hierarchical data
  onAssetClick?: (asset: EnhancedSelectedAsset) => void;
  onRelationshipClick?: (edge: any) => void;
  onToggleFullScreen?: () => void;
  isFullScreen?: boolean;
  forceLineageType?: 'technical' | 'semantic';
  preferHierarchical?: boolean; // New prop to prefer hierarchical view
  highlightedNodes?: string[];
  directionMode?: 'upstream' | 'downstream' | 'both';
}

const EnhancedDualLineageViewer: React.FC<DualLineageViewerProps> = ({
  selectedAsset,
  technicalData,
  semanticData,
  hierarchicalData,
  onAssetClick,
  onRelationshipClick,
  onToggleFullScreen,
  isFullScreen = false,
  forceLineageType,
  preferHierarchical = false,
  highlightedNodes = [],
  directionMode = 'both',
}) => {
  const [lineageType, setLineageType] = useState<'technical' | 'semantic'>(
    forceLineageType || 'technical'
  );
  const [selectedEdge, setSelectedEdge] = useState<Edge | null>(null);
  const [selectedNode, setSelectedNode] = useState<FlowNode<Record<string, unknown>> | null>(null);
  const [lineageNodes, setLineageNodes] = useState<FlowNode<Record<string, unknown>>[]>([]);
  const [lineageEdges, setLineageEdges] = useState<Edge[]>([]);
  const [useHierarchical, setUseHierarchical] = useState(preferHierarchical);
  const [overlayOpen, setOverlayOpen] = useState<boolean>(false);
  const [showMiniMap, setShowMiniMap] = useState<boolean>(true);

  useEffect(() => {
    if (isFullScreen) {
      const t = setTimeout(() => setOverlayOpen(true), 20);
      return () => clearTimeout(t);
    }
    setOverlayOpen(false);
  }, [isFullScreen]);

  // Determine if hierarchical view is appropriate
  const shouldUseHierarchical = useMemo(() => {
    if (!selectedAsset || !hierarchicalData) return false;
    
    // Use hierarchical for database assets with qualified paths
    const hasQualifiedPath = typeof selectedAsset?.qualifiedPath === 'string' && selectedAsset.qualifiedPath.includes('.');
    
    const isDatabaseAsset = ['column', 'table', 'schema'].includes(selectedAsset.type);
    
    return hasQualifiedPath && isDatabaseAsset;
  }, [selectedAsset, hierarchicalData]);

  // Auto-switch between flat and hierarchical views
  useEffect(() => {
    if (preferHierarchical && shouldUseHierarchical) {
      setUseHierarchical(true);
    } else if (!shouldUseHierarchical) {
      setUseHierarchical(false);
    }
  }, [shouldUseHierarchical, preferHierarchical]);

  // Auto-switch lineage type based on selected asset type or force prop
  useEffect(() => {
    if (forceLineageType) {
      if (lineageType !== forceLineageType) {
        setLineageType(forceLineageType);
      }
      return;
    }

    if (!selectedAsset) return;
    
    devLog('=== AUTO-SWITCHING LINEAGE TYPE ===');
    devDebug('Selected asset type:', selectedAsset.type);
    
    const businessTypes = ['business_term', 'semantic_term', 'semantic_model', 'semantic_column', 'database_column'];
    const technicalTypes = ['table', 'column', 'schema'];
    
    if (businessTypes.includes(selectedAsset.type)) {
  devLog('Switching to semantic lineage for business asset');
      setLineageType('semantic');
    } else if (technicalTypes.includes(selectedAsset.type)) {
  devLog('Switching to technical lineage for technical asset');
      setLineageType('technical');
    }
  }, [selectedAsset, forceLineageType, lineageType]);

  // Use hierarchical layout hook
  const hierarchicalLayout = useHierarchicalLayout(
    selectedAsset,
  hierarchicalData || null,
    lineageType === 'technical' ? technicalData : null
  );

  // Traditional flat layouts
  const technicalLineageLayout = useMemo(() => {
    if (lineageType !== 'technical' || useHierarchical) return { nodes: [], edges: [] };
    return buildTechnicalLineageLayout(selectedAsset, technicalData);
  }, [selectedAsset, technicalData, lineageType, useHierarchical]);

  const semanticLineageLayout = useMemo(() => {
    if (lineageType !== 'semantic' || useHierarchical) return { nodes: [], edges: [] };
    
  devLog('=== BUILDING SEMANTIC LINEAGE LAYOUT ===');
    
    if (!selectedAsset || !semanticData) {
  devDebug('Early return - missing data', { selectedAsset, semanticData });
      return { nodes: [], edges: [] };
    }

    // Log the structure of semanticData
    devDebug('SemanticData structure:', {
      hasNodes: 'nodes' in semanticData,
      nodesCount: ('nodes' in semanticData) ? (semanticData as any).nodes?.length : 'N/A',
      hasEdges: 'edges' in semanticData,
      edgesCount: ('edges' in semanticData) ? (semanticData as any).edges?.length : 'N/A',
      hasBusinessTerms: 'businessTerms' in semanticData,
      hasSemanticTerms: 'semanticTerms' in semanticData,
      businessTermsCount: ('businessTerms' in semanticData) ? (semanticData as any).businessTerms?.length : 'N/A',
      semanticTermsCount: ('semanticTerms' in semanticData) ? (semanticData as any).semanticTerms?.length : 'N/A',
      allKeys: Object.keys(semanticData || {})
    });

    // Your existing semantic layout logic...
    const isReactFlowFormat = 'nodes' in semanticData && Array.isArray(semanticData.nodes);
    const isOriginalFormat = 'businessTerms' in semanticData && Array.isArray(semanticData.businessTerms);
    
    devDebug('Format detection:', { isReactFlowFormat, isOriginalFormat });

  let centerNode: any;
    const assetIdentifier = selectedAsset.type === 'business_term' ? selectedAsset.name : selectedAsset.id;
    const assetName = selectedAsset.name;

  if (isReactFlowFormat) {
        const reactFlowNode = semanticData.nodes.find(n => 
            n.id === assetIdentifier || 
            n.data?.label === assetName ||
            (n.data?.id && n.data.id === assetIdentifier)
        );
        
        if (reactFlowNode) {
            centerNode = {
        id: String(reactFlowNode.id),
        node_name: reactFlowNode.data?.label || assetName,
        node_type: reactFlowNode.data?.nodeType || reactFlowNode.type || selectedAsset.type,
        description: reactFlowNode.data?.description || '',
        qualified_path: reactFlowNode.data?.qualifiedPath || '',
        properties: reactFlowNode.data?.properties || {}
            };
        }
    } else if (isOriginalFormat) {
    const rawData = semanticData as unknown as { businessTerms?: any[]; semanticTerms?: any[]; semanticColumns?: any[]; databaseColumns?: any[] };
    const allSemanticNodes: any[] = [
      ...(rawData.businessTerms || []),
      ...(rawData.semanticTerms || []),
      ...(rawData.semanticColumns || []),
      ...(rawData.databaseColumns || [])
    ];
        centerNode = allSemanticNodes.find(node =>
            node.id.toString() === assetIdentifier ||
            node.node_name.toLowerCase().trim() === assetName.toLowerCase().trim()
        );
    }

    if (!centerNode) {
  devDebug('No center node found!');
      return { nodes: [], edges: [] };
    }

    devDebug('Found centerNode:', { id: centerNode.id, name: centerNode.node_name, type: centerNode.node_type });
    
    // If data is already in ReactFlow format, return it directly
    if (isReactFlowFormat) {
      devDebug('Data is already in ReactFlow format, returning directly');
      return {
        nodes: semanticData.nodes,
        edges: semanticData.edges
      };
    }
    
    const layout = buildSemanticLineageLayout(centerNode, semanticData as SemanticLineageData);
    
    devDebug('Semantic layout result:', { 
      nodesCount: layout.nodes?.length || 0,
      edgesCount: layout.edges?.length || 0,
      firstNodeId: layout.nodes?.[0]?.id,
      firstNodeLabel: layout.nodes?.[0]?.data?.label,
      firstEdgeId: layout.edges?.[0]?.id,
      firstEdgeSource: layout.edges?.[0]?.source,
      firstEdgeTarget: layout.edges?.[0]?.target
    });
    
    return layout;
  }, [selectedAsset, semanticData, lineageType, useHierarchical]);

  // Current layout selection
  const currentLayout = useMemo(() => {
    if (useHierarchical && hierarchicalLayout.isHierarchical) {
      // Use hierarchical layout with proper positioning
      const positionedNodes = calculateHierarchicalPositions(
        hierarchicalLayout.nodes,
        hierarchicalData?.hierarchy || {}
      );
      return {
        nodes: positionedNodes,
        edges: hierarchicalLayout.edges
      };
    }
    
    return lineageType === 'technical' ? technicalLineageLayout : semanticLineageLayout;
  }, [useHierarchical, hierarchicalLayout, lineageType, technicalLineageLayout, semanticLineageLayout, hierarchicalData]);

  // Update state when the underlying layout changes
  // Update state when the underlying layout changes and apply highlighting/filtering
  useEffect(() => {
    let nodes = currentLayout.nodes;
    let edges = currentLayout.edges;

    // Apply directional filtering if in semantic mode
    if (lineageType === 'semantic' && directionMode !== 'both') {
      nodes = nodes.filter(n => {
        if (n.data?.isCenter) return true;
        // In semantic layout, x < 0 is upstream, x > 0 is downstream
        if (directionMode === 'upstream') return n.position.x < 0;
        if (directionMode === 'downstream') return n.position.x > 0;
        return true;
      });
      
      const nodeIds = new Set(nodes.map(n => n.id));
      edges = edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target));
    }

    // Apply highlighting
    if (highlightedNodes.length > 0) {
      nodes = nodes.map(n => ({
        ...n,
        data: {
          ...n.data,
          isHighlighted: highlightedNodes.includes(n.id)
        }
      }));
    }

    setLineageNodes(nodes);
    setLineageEdges(edges);
  }, [currentLayout, highlightedNodes, directionMode, lineageType]);

  devDebug('Final current layout:', currentLayout);
  devDebug('Using hierarchical:', useHierarchical);

  // Auto-select first edge
  useEffect(() => {
    if (currentLayout.edges.length > 0 && !selectedEdge) {
      setSelectedEdge(currentLayout.edges[0]);
    } else if (currentLayout.edges.length === 0) {
      setSelectedEdge(null);
    }
  }, [currentLayout.edges, selectedEdge]);

  // Event handlers
  const handleEdgeClick = (event: React.MouseEvent, edge: Edge) => {
    event.stopPropagation();
    setSelectedEdge(edge);
    setSelectedNode(null); // Clear node selection when edge is clicked
    if (onRelationshipClick) {
      onRelationshipClick(edge);
    }
  };

  const handleNodeClick = (event: React.MouseEvent, node: FlowNode<Record<string, unknown>>) => {
    event.stopPropagation();
    
    // Handle expansion/collapse for container nodes
    if (node.data?.isContainer) {
      setLineageNodes(nds => updateNodeExpansion(nds, node.id, !node.data.expanded));
      devDebug('Toggle container:', node.id, 'expanded:', !node.data.expanded);
      return;
    }
    
    // Set selected node for property display
    setSelectedNode(node);
    setSelectedEdge(null); // Clear edge selection when node is clicked
    
    // Navigate to asset if not center node
    if (onAssetClick && !node.data?.isCenter) {
      const asset = {
        type: node.data?.nodeType || 'table',
        id: node.id,
        nodeId: node.id,
        name: node.data?.label || 'Unknown',
        isCore: node.data?.isCore,
        qualifiedPath: node.data?.qualifiedPath,
        schema: node.data?.schema,
        table: node.data?.table,
        column: node.data?.column,
      };
      onAssetClick(asset as unknown as EnhancedSelectedAsset);
    }
  };

  // No asset selected state
  if (!selectedAsset) {
    return (
      <div className="dual-lineage-empty">
        <div className="dual-lineage-empty-inner">
          <div className="dual-lineage-empty-icon">📊</div>
          <h3 className="dual-lineage-empty-title">Select an Asset</h3>
          <p className="dual-lineage-empty-desc">Choose a table, column, or business term to view its lineage relationships</p>
        </div>
      </div>
    );
  }

  // Calculate counts for type selector
  const technicalCount = technicalData?.metadata?.databaseEdgeCount || 0;
  let semanticCount = 0;
  if (semanticData) {
    const getArrayLength = (obj: unknown, key: string) => {
      if (typeof obj !== 'object' || obj === null) return 0;
      const v = (obj as Record<string, unknown>)[key];
      return Array.isArray(v) ? v.length : 0;
    };

    semanticCount = getArrayLength(semanticData, 'edges') || getArrayLength(semanticData, 'businessTerms');
  }

  // No relationships found state
  if (lineageNodes.length <= 1) {
    // Render empty state but include the lineage controls (so fullscreen button is available
    // and tests can find it). When `isFullScreen` is true we still render the overlay below.
    const content = (
      <div className="dual-lineage-empty-wrapper">
        <div className="dual-lineage-empty-card">
          <div className="dual-lineage-empty-inner">
            <div className="dual-lineage-empty-icon">{lineageType === 'technical' ? '🔗' : '💫'}</div>
            <h3 className="dual-lineage-empty-title">No {useHierarchical ? 'Hierarchical ' : ''}{lineageType} Relationships</h3>
            <p className="dual-lineage-empty-desc">No {lineageType === 'technical' ? 'foreign key' : 'semantic'} relationships found for this asset.</p>
          </div>
        </div>
      </div>
    );

    if (isFullScreen) {
      // Render overlay containing the empty content so tests that mount with isFullScreen=true
      // can assert overlay presence and open animation class.
      const overlay = (
        <div className={`dlv-overlay ${overlayOpen ? 'open' : ''}`}>
          <div className="dlv-content">
            {/* include controls so fullscreen button is present */}
            <div className="dual-lineage-container">
              <div className="dual-lineage-controls">
                <div className="dual-lineage-controls-inner">
                  <div className="dual-lineage-fullscreen-container">
                    <button
                      className="dual-lineage-fullscreen-btn"
                      onClick={() => {
                        // mirror the inline behavior used elsewhere: request parent toggle
                        onToggleFullScreen && onToggleFullScreen();
                      }}
                    >⤢ Fullscreen</button>
                  </div>
                </div>
              </div>
              {content}
            </div>
          </div>
        </div>
      );

      if (typeof document !== 'undefined' && document.body) {
        return createPortal(overlay, document.body);
      }
      return overlay;
    }

    return (
      <div>
        <div className="dual-lineage-container">
          <div className="dual-lineage-controls">
            <div className="dual-lineage-controls-inner">
                <div className="dual-lineage-fullscreen-container">
                    <button
                      className="dual-lineage-fullscreen-btn"
                      onClick={() => {
                        // closing behavior: mimic inline close which notifies parent
                        onToggleFullScreen && onToggleFullScreen();
                      }}
                    >⤢ Fullscreen</button>
                  </div>
            </div>
          </div>
          {content}
        </div>
      </div>
    );
  }

  // Main render with lineage visualization
  const renderMain = () => (
    <div className="dual-lineage-container" style={{ position: 'relative' }}>
      {/* Sleek sidebar with icon controls */}
      <div style={{
        position: 'fixed',
        left: '16px',
        top: '16px',
        zIndex: 2500,
        background: 'rgba(255, 255, 255, 0.95)',
        backdropFilter: 'blur(10px)',
        borderRadius: '12px',
        padding: '12px 8px',
        boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)',
        display: 'flex',
        flexDirection: 'column',
        gap: '8px',
        border: '1px solid rgba(0, 0, 0, 0.1)'
      }}>
        {!forceLineageType && (
          <>
            {/* Lineage Type Selector - only show if not forced */}
            <Suspense fallback={<div style={{ width: '40px', height: '40px' }} />}>
              <LineageTypeSelector
                lineageType={lineageType}
                onLineageTypeChange={setLineageType}
                technicalCount={technicalCount}
                semanticCount={semanticCount}
              />
            </Suspense>
            
            {/* Hierarchical/Flat toggle */}
            {shouldUseHierarchical && (
              <button
                onClick={() => setUseHierarchical(!useHierarchical)}
                title={useHierarchical ? 'Switch to Flat View' : 'Switch to Hierarchical View'}
                style={{
                  width: '40px',
                  height: '40px',
                  borderRadius: '8px',
                  border: 'none',
                  background: useHierarchical ? '#6366f1' : '#e5e7eb',
                  color: useHierarchical ? '#ffffff' : '#6b7280',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '20px',
                  transition: 'all 0.2s ease',
                  boxShadow: useHierarchical ? '0 2px 8px rgba(99, 102, 241, 0.3)' : 'none'
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.transform = 'scale(1.05)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.transform = 'scale(1)';
                }}
              >
                🗂️
              </button>
            )}
          </>
        )}
        
        {/* MiniMap toggle - always visible */}
        <button
          onClick={() => setShowMiniMap(!showMiniMap)}
          title={showMiniMap ? 'Hide MiniMap' : 'Show MiniMap'}
          style={{
            width: '40px',
            height: '40px',
            borderRadius: '8px',
            border: 'none',
            background: showMiniMap ? '#10b981' : '#e5e7eb',
            color: showMiniMap ? '#ffffff' : '#6b7280',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '20px',
            transition: 'all 0.2s ease',
            boxShadow: showMiniMap ? '0 2px 8px rgba(16, 185, 129, 0.3)' : 'none'
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.transform = 'scale(1.05)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.transform = 'scale(1)';
          }}
        >
          🗺️
        </button>

        {/* Divider */}
        <div style={{
          height: '1px',
          background: 'rgba(0, 0, 0, 0.1)',
          margin: '4px 0'
        }} />

        {/* Fullscreen toggle - always visible */}
        <button
          onClick={() => {
            if (!isFullScreen) {
              onToggleFullScreen && onToggleFullScreen();
              return;
            }
            setOverlayOpen(false);
            setTimeout(() => {
              onToggleFullScreen && onToggleFullScreen();
            }, 180);
          }}
          title={isFullScreen ? 'Exit Fullscreen' : 'Enter Fullscreen'}
          style={{
            width: '40px',
            height: '40px',
            borderRadius: '8px',
            border: 'none',
            background: isFullScreen ? '#f59e0b' : '#e5e7eb',
            color: isFullScreen ? '#ffffff' : '#6b7280',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '20px',
            transition: 'all 0.2s ease',
            boxShadow: isFullScreen ? '0 2px 8px rgba(245, 158, 11, 0.3)' : 'none'
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.transform = 'scale(1.05)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.transform = 'scale(1)';
          }}
        >
          {isFullScreen ? '⤓' : '⤢'}
        </button>
      </div>

      {/* Main content area */}
  <div className="dual-lineage-main">
        {/* Lineage flow visualization */}
          <div className="dual-lineage-flow">
            <Suspense fallback={<div className="suspense-fallback-full">Loading graph...</div>}>
              <LineageFlow
                nodes={lineageNodes}
                edges={lineageEdges}
                nodeTypes={nodeTypes}
                edgeTypes={edgeTypes}
                onNodeClick={handleNodeClick}
                onEdgeClick={handleEdgeClick}
                showMiniMap={showMiniMap}
                highlightedNodes={highlightedNodes}
              />
            </Suspense>
          </div>

        {/* Details panel */}
        <div className="dual-lineage-details">
          <Suspense fallback={<div className="suspense-fallback-16">Loading details...</div>}>
            <DualLineageDetailsPanel
              edge={selectedEdge}
              nodes={lineageNodes}
              lineageType={lineageType}
              selectedNode={selectedNode}
            />
          </Suspense>
        </div>
      </div>
    </div>
  );

  if (isFullScreen) {
    const overlay = (
      <div className={`dlv-overlay ${overlayOpen ? 'open' : ''}`}>
        <div className="dlv-content">
          {renderMain()}
        </div>
      </div>
    );

    if (typeof document !== 'undefined' && document.body) {
      return createPortal(overlay, document.body);
    }

    return overlay;
  }

  return renderMain();
};

export default EnhancedDualLineageViewer;
