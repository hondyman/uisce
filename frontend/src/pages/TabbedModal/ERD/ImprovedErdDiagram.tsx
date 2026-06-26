/* eslint-disable @typescript-eslint/no-unused-vars */
import { useCallback, useState, useMemo, useEffect } from 'react';
import ReactFlow, {
  Background,
  Node as FlowNode,
  Edge,
  ReactFlowInstance,
  Viewport,
  MarkerType,
  BackgroundVariant,
  Handle,
  Position,
  NodeProps,
} from 'reactflow';
import BusinessEntitySemanticService from '../../../services/businessEntitySemanticService';
import ErdSidebar from './ErdSidebar';
import ErdMinimap from './ErdMinimap';
import ErdInfoPanel from './ErdInfoPanel';

// CRITICAL: Import ReactFlow styles first
import 'reactflow/dist/style.css';

// Import the custom CSS file
import './ImprovedErdDiagram.css';
import { devDebug } from '../../../utils/devLogger';

interface Column {
  name: string;
  type: string;
  isPrimaryKey?: boolean;
  isForeignKey?: boolean;
  nullable?: boolean;
}

interface TableNodeData {
  label: string;
  schema?: string;
  columns?: Column[];
  showColumns?: boolean;
  highlightedItem?: string | null;
}

interface ImprovedErdDiagramProps {
  nodes: FlowNode[];
  edges: Edge[];
  nodeTypes: any;
  showColumns: boolean;
  showMiniMap: boolean;
  highlightedItem: string | null;
  zoomLevel: number;
  onInit: (instance: ReactFlowInstance) => void;
  onNodeClick: (event: React.MouseEvent, node: FlowNode) => void;
  onEdgeClick: (event: React.MouseEvent, edge: Edge) => void;
  onPaneClick: () => void;
  onMoveEnd: (event: any, viewport: Viewport) => void;
  onZoomChange: (zoom: number) => void;
  onToggleColumns: () => void;
  onToggleMiniMap: () => void;
  onFitView: () => void;
}

// Helper to calculate node dimensions based on content
const calculateNodeSize = (nodeData: TableNodeData): { width: number; height: number } => {
  const { label, columns = [], showColumns = true } = nodeData;

  const width = (() => {
    if (!showColumns || !columns || columns.length === 0) {
      return Math.max(240, label.length * 10 + 80);
    }
    
    const longestColumnName = Math.max(0, ...columns.map(col => col.name.length));
    const longestColumnType = Math.max(0, ...columns.map(col => col.type.length));
    const contentWidth = Math.max(300, (longestColumnName * 8) + (longestColumnType * 7) + 130);
    const titleWidth = label.length * 10 + 80;
    
    return Math.max(contentWidth, titleWidth, 300);
  })();

  const headerHeight = 68; // Approx height of the table header
  const columnHeight = 42; // Height of each column row
  const height = showColumns && columns && columns.length > 0 ? 
    headerHeight + (columns.length * columnHeight) : headerHeight;

  return { width, height };
};

// Professional Table Node Component with Enhanced Interactions
const ProfessionalTableNode: React.FC<NodeProps<TableNodeData> & { infoMode?: boolean; onColumnClick?: (column: Column, nodeId: string) => void }> = ({ data, id, selected, infoMode, onColumnClick }) => {
  // Safe defaults for all data properties
  const safeData = data || {};
  const { 
    label = 'Unknown Table', 
    schema, 
    columns = [], 
    showColumns = true, 
    highlightedItem 
  } = safeData;

  const isHighlighted = highlightedItem === `table-${id}` || highlightedItem === id;
  
  // Calculate optimal dimensions
  const { width: nodeWidth, height: nodeHeight } = useMemo(() => calculateNodeSize(safeData), [safeData]);

  // Calculate statistics
  const stats = useMemo(() => {
    const validColumns = Array.isArray(columns) ? columns : [];
    const pkCount = validColumns.filter(c => c.isPrimaryKey).length;
    const fkCount = validColumns.filter(c => c.isForeignKey).length;
    const nullableCount = validColumns.filter(c => c.nullable).length;
    return { pkCount, fkCount, nullableCount, total: validColumns.length };
  }, [columns]);

  useEffect(() => {
    console.log(`ProfessionalTableNode rendered: ${label} (${id})`);
  }, [label, id]);

  const styleId = `professional-node-style-${id}`;

  return (
    <div 
      id={styleId} 
      className={`professional-table-node ${selected ? 'professional-table-node--selected' : ''} ${isHighlighted ? 'professional-table-node--highlighted' : ''}`}
      title={`${label}${schema ? ` (${schema})` : ''}\n${stats.total} columns • ${stats.pkCount} PK • ${stats.fkCount} FK`}
    >
      <style>{`#${styleId} { --node-width: ${nodeWidth}px; --node-height: ${nodeHeight}px; }`}</style>
      
      {/* Enhanced Connection Handles */}
      <Handle type="target" position={Position.Top} className="connection-handle professional-handle professional-handle--top" />
      <Handle type="source" position={Position.Bottom} className="connection-handle professional-handle professional-handle--bottom" />
      <Handle type="target" position={Position.Left} className="connection-handle professional-handle professional-handle--left" />
      <Handle type="source" position={Position.Right} className="connection-handle professional-handle professional-handle--right" />
      
      {/* Professional Table Header */}
      <div className="professional-table-header">
        <div className="professional-header-pattern" />
        <div className="professional-header-left">
          <div className="professional-header-icon">📊</div>
          <div>
            <div className="professional-header-title">{label}</div>
            {showColumns && stats.total > 0 && (
              <div className="professional-header-sub">
                {stats.total} col{stats.total !== 1 ? 's' : ''} • {stats.pkCount} PK • {stats.fkCount} FK
              </div>
            )}
          </div>
        </div>
        <div className="professional-type-indicator">TABLE</div>
      </div>
      
      {/* Enhanced Columns Section */}
      {showColumns && stats.total > 0 && (
        <div className="professional-columns">
          {Array.isArray(columns) && columns.map((column, index) => {
            if (!column) return null;
            const isColumnHighlighted = highlightedItem === `column-${id}-${index}`;
            const isPK = column.isPrimaryKey;
            const isFK = column.isForeignKey;
            
            const rowClass = `professional-column-row ${
              isColumnHighlighted ? 'professional-column-row--highlighted' : 
              isPK ? 'professional-column-row--pk' : 
              isFK ? 'professional-column-row--fk' : ''
            }`;

            const tooltipText = [
              column.name,
              column.type,
              isPK ? 'Primary Key' : '',
              isFK ? 'Foreign Key' : '',
              column.nullable === false ? 'NOT NULL' : 'Nullable'
            ].filter(Boolean).join(' • ');

            return (
              <div
                key={index}
                className={rowClass}
                title={tooltipText}
                onClick={() => infoMode && onColumnClick && onColumnClick(column, id)}
                style={{ cursor: infoMode ? 'pointer' : 'default' }}
              >
                {/* Enhanced Key Icon */}
                <div className={`professional-col-key ${isPK ? 'professional-col-key--pk' : isFK ? 'professional-col-key--fk' : ''}`}>
                  {isPK && <span>🔑</span>}
                  {isFK && !isPK && <span>🔗</span>}
                </div>
                
                {/* Enhanced Column Details */}
                <div className="professional-col-details">
                  <div className="professional-col-info">
                    <span className="professional-col-name">{column.name}</span>
                    <span className="professional-col-type">{column.type}</span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

// Main Improved ERD Diagram Component
const ImprovedErdDiagram: React.FC<ImprovedErdDiagramProps> = ({
  nodes: initialNodes,
  edges: initialEdges,
  nodeTypes,
  showColumns,
  showMiniMap,
  highlightedItem,
  zoomLevel,
  onInit,
  onNodeClick,
  onEdgeClick,
  onPaneClick,
  onMoveEnd,
  onZoomChange,
  onToggleColumns,
  onToggleMiniMap,
  onFitView
}) => {
  console.log('🎨 ImprovedErdDiagram component mounted with:', {
    nodesCount: initialNodes?.length || 0,
    edgesCount: initialEdges?.length || 0,
    nodeTypesKeys: nodeTypes ? Object.keys(nodeTypes) : [],
    showColumns,
    showMiniMap
  });
  const [isMinimapVisible, setMinimapVisible] = useState(showMiniMap);
  const [infoMode, setInfoMode] = useState(false);
  const [selectedColumnInfo, setSelectedColumnInfo] = useState<{ column: Column; tableName: string } | null>(null);

  const [selectedEdgeInfo, setSelectedEdgeInfo] = useState<Edge | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);

  useEffect(() => {
    setMinimapVisible(showMiniMap);
  }, [showMiniMap]);

  const handleToggleMinimap = useCallback(() => {
    setMinimapVisible(prev => !prev);
  }, []);

  const handleToggleInfoMode = useCallback(() => {
    setInfoMode(prev => !prev);
    // Close info panel when disabling info mode
    if (infoMode) {
      setSelectedColumnInfo(null);
      setSelectedEdgeInfo(null);
    }
  }, [infoMode]);

  const handleColumnClick = useCallback((column: Column, nodeId: string) => {
    if (infoMode) {
      const node = initialNodes.find(n => n.id === nodeId);
      const tableName = node?.data?.label || nodeId;
      setSelectedColumnInfo({ column, tableName });
      setSelectedEdgeInfo(null);
    }
  }, [infoMode, initialNodes]);

  const handleEdgeClickInfo = useCallback((event: React.MouseEvent, edge: Edge) => {
    if (infoMode) {
      setSelectedEdgeInfo(edge);
      setSelectedColumnInfo(null);
    }
    onEdgeClick(event, edge);
  }, [infoMode, onEdgeClick]);

  const handleCloseInfoPanel = useCallback(() => {
    setSelectedColumnInfo(null);
    setSelectedEdgeInfo(null);
  }, []);

  const handleGenerateMappings = useCallback(async () => {
    try {
      if (!window.confirm('This will analyze all columns and generate semantic terms. Continue?')) {
        return;
      }
      
      setIsGenerating(true);
      const tenant = JSON.parse(localStorage.getItem('selected_tenant') || '{}');
      const datasource = JSON.parse(localStorage.getItem('selected_datasource') || '{}');
      
      if (!tenant.id || !datasource.id) {
        alert('Missing tenant or datasource context');
        return;
      }

      const svc = new BusinessEntitySemanticService(tenant.id, datasource.id);
      const result = await svc.generateSemanticMappings();
      
      alert(`Semantic analysis complete! Generated ${result.semantic_terms_created || 0} terms.`);
      // Ideally trigger a refresh of the diagram here if needed, but for now just notify
      
    } catch (err: any) {
      console.error('Failed to generate mappings:', err);
      alert('Failed to generate mappings: ' + (err.message || 'Unknown error'));
    } finally {
      setIsGenerating(false);
    }
  }, []);

  // Intelligent layout with adaptive spacing based on actual node sizes
  const generateOptimalLayout = (nodes: FlowNode[], edges: Edge[], showCols: boolean): FlowNode[] => {
    if (!nodes || nodes.length === 0) return [];

    // Calculate actual node sizes based on current column visibility
    const nodesWithSizes = nodes.map(node => {
      const size = calculateNodeSize({ ...node.data, showColumns: showCols });
      return {
        ...node,
        width: size.width,
        height: size.height
      };
    });

    // Adaptive spacing based on whether columns are shown
    // When columns are hidden, nodes are smaller so we use tighter spacing
    // When columns are shown, nodes are larger so we need more space
    const baseHorizontalPadding = showCols ? 180 : 120;
    const baseVerticalPadding = showCols ? 150 : 100;
    
    // Calculate average node dimensions for intelligent spacing
    const avgWidth = nodesWithSizes.reduce((sum, n) => sum + n.width, 0) / nodesWithSizes.length;
    const avgHeight = nodesWithSizes.reduce((sum, n) => sum + n.height, 0) / nodesWithSizes.length;
    
    // Adaptive padding scales with node size
    const horizontalPadding = Math.max(baseHorizontalPadding, avgWidth * 0.3);
    const verticalPadding = Math.max(baseVerticalPadding, avgHeight * 0.4);
    
    // Dynamic canvas width based on number of nodes and their sizes
    const estimatedRowWidth = avgWidth + horizontalPadding;
    const optimalNodesPerRow = Math.ceil(Math.sqrt(nodesWithSizes.length));
    const layoutWidth = estimatedRowWidth * Math.min(optimalNodesPerRow, 8); // Max 8 nodes per row

    const positionedNodes: FlowNode[] = [];
    let currentX = 100;
    let currentY = 100;
    let rowNodes: (FlowNode & {width: number, height: number})[] = [];

    const positionRow = (row: (FlowNode & {width: number, height: number})[], yOffset: number) => {
      if (row.length === 0) return 0;
      
      // Calculate total row width to center it
      const totalRowWidth = row.reduce((sum, n, i) => 
        sum + n.width + (i < row.length - 1 ? horizontalPadding : 0), 0
      );
      
      // Center the row
      let xOffset = Math.max(100, (layoutWidth - totalRowWidth) / 2);
      const rowHeight = Math.max(...row.map(n => n.height));
      
      for (const node of row) {
        positionedNodes.push({
          ...node,
          position: { x: xOffset, y: yOffset }
        });
        xOffset += node.width + horizontalPadding;
      }
      
      return rowHeight;
    };

    // Distribute nodes into rows intelligently
    for (const node of nodesWithSizes) {
      // Check if adding this node would exceed layout width
      const rowWidth = rowNodes.reduce((sum, n) => sum + n.width + horizontalPadding, 0);
      
      if (rowNodes.length > 0 && (rowWidth + node.width > layoutWidth || rowNodes.length >= 8)) {
        // Position current row
        const rowHeight = positionRow(rowNodes, currentY);
        currentY += rowHeight + verticalPadding;
        currentX = 100;
        rowNodes = [];
      }
      
      rowNodes.push(node);
      currentX += node.width + horizontalPadding;
    }

    // Position final row
    if (rowNodes.length > 0) {
      positionRow(rowNodes, currentY);
    }

    devDebug(`Layout: ${showCols ? 'WITH' : 'WITHOUT'} columns. Spacing: H=${Math.round(horizontalPadding)}px, V=${Math.round(verticalPadding)}px, Width=${Math.round(layoutWidth)}px`);

    return positionedNodes;
  };

  // Define node types locally to ensure they are available
  const finalNodeTypes = useMemo(() => {
    const TableNodeWithInfo = (props: NodeProps<TableNodeData>) => (
      <ProfessionalTableNode {...props} infoMode={infoMode} onColumnClick={handleColumnClick} />
    );
    
    return {
      professionalTable: TableNodeWithInfo,
      table: TableNodeWithInfo, // Fallback for 'table' type
      ...nodeTypes
    };
  }, [nodeTypes, infoMode, handleColumnClick]);

  // Enhanced nodes with proper data flow
  const enhancedNodes = useMemo(() => {
    if (!initialNodes || !Array.isArray(initialNodes) || initialNodes.length === 0) {
      return [];
    }

    // Pass showColumns and highlightedItem to node data
    const nodesWithData = initialNodes.map(node => ({
      ...node,
      type: 'professionalTable', // Force correct node type
      data: {
        ...node.data,
        showColumns,
        highlightedItem
      }
    }));
    
    devDebug('ImprovedErdDiagram: Processing', nodesWithData.length, 'nodes');
    devDebug('ImprovedErdDiagram: Sample node positions:', nodesWithData.slice(0, 5).map(n => ({ 
      id: n.id, 
      label: n.data?.label, 
      x: n.position?.x,
      y: n.position?.y,
      hasPosition: !!(n.position && (n.position.x !== 0 || n.position.y !== 0))
    })));

    // CRITICAL: When showColumns changes, we MUST recalculate layout
    // because node sizes change dramatically (header-only vs full columns)
    // We ignore cached positions in this case to ensure optimal spacing
    const hasPositions = nodesWithData.some(n => n.position && (n.position.x !== 0 || n.position.y !== 0));
    
    // Always regenerate layout to ensure proper spacing for current column visibility
    // This prevents overlap when columns are shown and excessive spacing when hidden
    devDebug('ImprovedErdDiagram: Forcing layout regeneration for optimal spacing (showColumns=' + showColumns + ')');
    
    const layoutedNodes = generateOptimalLayout(nodesWithData, initialEdges, showColumns);
    devDebug('ImprovedErdDiagram: Layout generated. First 3 nodes:', layoutedNodes.slice(0, 3).map(n => ({ 
      id: n.id, 
      position: n.position,
      width: n.width,
      height: n.height
    })));
    return layoutedNodes;
  }, [initialNodes, initialEdges, showColumns, highlightedItem]);


  // Enhanced edges with professional styling
  const enhancedEdges = useMemo(() => {
    if (!initialEdges || !Array.isArray(initialEdges)) return [];
    
    return initialEdges.map((edge) => ({
      ...edge,
      className: 'professional-erd-edge',
      label: edge.data?.relationship,
      labelShowBg: true,
      style: {
        strokeWidth: 3,
        strokeDasharray: edge.data?.relationship === 'optional' ? '10,5' : undefined,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 24,
        height: 24,
      },
      type: 'smoothstep',
      animated: false,
      pathOptions: {
        offset: 15,
        borderRadius: 16
      }
    }));
  }, [initialEdges]);

  // Removed unused customNodeTypes definition

  devDebug('ImprovedErdDiagram: Rendering ReactFlow with', enhancedNodes.length, 'nodes and', enhancedEdges.length, 'edges');
  devDebug('ImprovedErdDiagram: nodeTypes registered:', Object.keys(finalNodeTypes));
  devDebug('ImprovedErdDiagram: First enhanced node:', enhancedNodes[0]);

  console.log('🔧 ABOUT TO RENDER REACTFLOW', {
    enhancedNodesCount: enhancedNodes.length,
    enhancedEdgesCount: enhancedEdges.length,
    customNodeTypesKeys: Object.keys(finalNodeTypes),
    firstNodeType: enhancedNodes[0]?.type,
    firstNodeId: enhancedNodes[0]?.id
  });

  return (
    <div className="improved-erd-container">
      {(() => { console.log('✅ ReactFlow container div rendered'); return null; })()}
      <ReactFlow
        nodes={enhancedNodes}
        edges={enhancedEdges}
        nodeTypes={finalNodeTypes}
        onInit={onInit}
        onNodeClick={onNodeClick}
        onEdgeClick={handleEdgeClickInfo}
        onPaneClick={onPaneClick}
        onMoveEnd={onMoveEnd}
        nodesDraggable={true}
        nodesConnectable={false}
        elementsSelectable={true}
        selectNodesOnDrag={false}
        panOnDrag={true}
        fitView={enhancedNodes.length > 0}
        fitViewOptions={{
          padding: 0.2,
          minZoom: 0.4,
          maxZoom: 1.5,
          includeHiddenNodes: false
        }}
        defaultViewport={{ x: 0, y: 0, zoom: 0.9 }}
        minZoom={0.3}
        maxZoom={2.0}
        attributionPosition="bottom-left"
        proOptions={{ hideAttribution: true }}
      >
        {/* Sidebar with controls */}
        <ErdSidebar
          zoomLevel={zoomLevel}
          showColumns={showColumns}
          showMiniMap={showMiniMap}
          infoMode={infoMode}
          onZoomChange={onZoomChange}
          onToggleColumns={onToggleColumns}
          onToggleMiniMap={handleToggleMinimap}
          onFitView={onFitView}
          onToggleInfoMode={handleToggleInfoMode}
          onGenerateMappings={handleGenerateMappings}
        />
        
        {isMinimapVisible && <ErdMinimap />}
        
        <Background 
          color="#cbd5e1" 
          gap={[24, 24]} 
          size={2}
          offset={2}
          className="improved-erd-background"
          variant={BackgroundVariant.Dots}
        />
      </ReactFlow>

      {/* Info Panel */}
      <ErdInfoPanel
        isOpen={infoMode && (selectedColumnInfo !== null || selectedEdgeInfo !== null)}
        selectedColumn={selectedColumnInfo?.column}
        selectedEdge={selectedEdgeInfo}
        tableName={selectedColumnInfo?.tableName}
        onClose={handleCloseInfoPanel}
      />

      {/* Professional watermark */}
      <div className="erd-watermark">Professional ERD Diagram</div>
    </div>
  );
};

export default ImprovedErdDiagram;