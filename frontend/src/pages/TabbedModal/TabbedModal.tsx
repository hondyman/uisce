// 1. Complete TabbedModal.tsx - Fixed with all necessary code
import { useState, useEffect, useMemo, useCallback, lazy, Suspense } from 'react';
import { useNotification } from '../../hooks/useNotification';
import { devLog, devWarn, devError, devDebug } from '../../utils/devLogger';
import { useApiQuery } from '../../hooks/useApiQuery';
import { Node as FlowNode, Edge, ReactFlowInstance } from 'reactflow';

// Services and Types
import { exportData } from '../../services/exportService';
import { ExportOptions } from '../../types/ExportTypes';
import { ColumnData, EnhancedSelectedAsset } from '../../types/SemanticTypes';
import { enrichNodesWithTypes } from '../../utils/nodeTypeMapping';

// Queries

// Components
import TableNode from '../../components/TableNode';
import ProfessionalSearchInput, { SearchSuggestion } from '../../components/common/ProfessionalSearchInput';
import ColumnDetailsModal from '../../components/ColumnDetailsModal';
// Lazy-load heavy subcomponents so TabbedModal doesn't pull them into the main chunk
const ErdDiagram = lazy(() => import('./ERD/ImprovedErdDiagram')) as unknown as React.ComponentType<any>;
const ErdControls = lazy(() => import('./ErdControls')) as unknown as React.ComponentType<any>;
const EnhancedExportOverlay = lazy(async () => {
  const mod = await import('../../components/ExportModal/EnhancedExportOverlay');
  const m = mod as unknown as { EnhancedExportOverlay?: React.ComponentType<any>; default?: React.ComponentType<any> };
  return { default: (m.EnhancedExportOverlay || m.default || (() => null)) as React.ComponentType<any> };
}) as unknown as React.ComponentType<any>;
const ExportButton = lazy(async () => {
  const mod = await import('./ExportButton');
  const m = mod as unknown as { ExportButton?: React.ComponentType<any>; default?: React.ComponentType<any> };
  return { default: (m.ExportButton || m.default || (() => null)) as React.ComponentType<any> };
}) as unknown as React.ComponentType<any>;
const DualLineageViewer = lazy(() => import('./Catalog/DualLineageViewer')) as unknown as React.ComponentType<any>;
const DatabaseCatalogView = lazy(() => import('./tabs/DatabaseCatalogView')) as unknown as React.ComponentType<any>;
const SemanticCatalogView = lazy(() => import('./tabs/SemanticCatalogView')) as unknown as React.ComponentType<any>;

// Styles
import './TabbedModal.css';

// Interfaces
interface TabbedModalProps {
  datasourceId: string;
  tenantId?: string;
  onClose: () => void;
  isModal?: boolean;
}

interface TableNodeData {
  schemaName?: string;
  tableName?: string;
  label?: string;
  isCore?: boolean;
  columns?: ColumnData[];
}

interface TechnicalLineageChart {
  nodes: FlowNode<TableNodeData>[];
  edges: Edge[];
  viewport: Record<string, unknown>;
  metadata: Record<string, any>;
}

interface SemanticLineageChart {
  businessTerms: any[];
  semanticTerms: any[];
  semanticColumns: any[];
  databaseColumns: any[];
  edges: any[];
  viewport: Record<string, unknown>;
  metadata: Record<string, any>;
}

interface SearchResult {
  id: string;
  kind: 'table' | 'column';
  label: string;
  nodeId: string;
  tableName: string;
  columnIndex?: number;
  isCore: boolean;
  hasModel: boolean;
}

interface CatalogSummary {
  id: string;
  label: string;
  tableName: string;
  schemaName: string;
  qualifiedPath?: string;
  hasModel: boolean;
  isCore: boolean;
  columnCount: number;
  modelTitle?: string;
  modelStatus?: string;
}

const detectIsCore = (data: TableNodeData & Record<string, any>, modelInfo?: Record<string, any>): boolean => {
  if (!data) return false;
  if (data.isCore === true || data.is_core === true) return true;
  if (typeof data.core_id === 'string' && data.core_id.length > 0) return true;
  if (data.core && typeof data.core === 'object') return true;
  if (Array.isArray(data.tags) && data.tags.some((tag: string) => tag?.toLowerCase() === 'core')) return true;
  if (modelInfo) {
    if (modelInfo.isCore === true || modelInfo.is_core === true) return true;
    if (typeof modelInfo.core_id === 'string' && modelInfo.core_id.length > 0) return true;
    if (modelInfo.core && typeof modelInfo.core === 'object') return true;
  }
  return false;
};

const deriveCatalogSummary = (node: FlowNode<TableNodeData>): CatalogSummary => {
  const rawData = node.data || {};
  const rawAny = rawData as Record<string, any>;
  const modelInfo = rawAny.modelInfo ?? {};
  const label = rawAny.label || rawAny.tableName || rawAny.qualifiedPath || node.id;
  const qualifiedPath = rawAny.qualifiedPath || rawAny.tableName || label;
  const tableName = rawAny.tableName || qualifiedPath || label;
  const schemaName = rawData.schemaName || (typeof tableName === 'string' && tableName.includes('.') ? tableName.split('.')[0] : 'default');
  const hasModel = Boolean(modelInfo?.exists ?? modelInfo?.status ?? modelInfo?.version ?? modelInfo?.title);
  const isCore = detectIsCore(rawData as TableNodeData & Record<string, any>, modelInfo);
  const columnCount = Array.isArray(rawAny.columns) ? rawAny.columns.length : 0;

  return {
    id: node.id,
    label: String(label),
    tableName: String(tableName),
    schemaName: String(schemaName),
    qualifiedPath: typeof qualifiedPath === 'string' ? qualifiedPath : undefined,
    hasModel,
    isCore,
    columnCount,
    modelTitle: modelInfo?.title,
    modelStatus: modelInfo?.status,
  };
};

// Utility Hooks
const useDebounce = (value: string, delay: number) => {
  const [debouncedValue, setDebouncedValue] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedValue(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return debouncedValue;
};

// ReactFlow Node Types
const nodeTypes = {
  databaseTable: TableNode,
};

const TabbedModal: React.FC<TabbedModalProps> = ({ datasourceId, tenantId = 'default', onClose, isModal = true }) => {
  console.log('🚀 TabbedModal mounted with:', { datasourceId, tenantId, isModal, hasOnClose: !!onClose });
  
  
  // REST API calls replacing GraphQL
  const { loading: nodesLoading, error: nodesError, data: rawNodes } = useApiQuery<any[]>(
    `api/rest/catalog-nodes?limit=10000`,
    { skip: !datasourceId }
  );

  const { loading: edgesLoading, error: edgesError, data: rawEdges } = useApiQuery<any[]>(
    `api/rest/catalog-edges?limit=10000`,
    { skip: !datasourceId }
  );

  const isLoading = nodesLoading || edgesLoading;
  const isError = !!nodesError || !!edgesError;

  // Re-map REST nodes and edges into the format TabbedModal expects
  // TabbedModal expects `nodes` state to be FlowNode objects, and `edges` to be ReactFlow edges.
  // It also expects `semanticAssets` which we can filter from `rawNodes`

  // Main tab state - elevated from catalog sub-tabs
  const [activeTab, setActiveTab] = useState<'database' | 'diagram' | 'lineage' | 'semantic'>(
    () => (localStorage.getItem('erdActiveTab') as 'database' | 'diagram' | 'lineage' | 'semantic') || 'database'
  );

  // Data state
  const [nodes, setNodes] = useState<FlowNode<TableNodeData>[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [semanticAssets, setSemanticAssets] = useState<any[]>([]);
  const [processedTechnicalData, setProcessedTechnicalData] = useState<TechnicalLineageChart | null>(null);
  const [processedSemanticData, setProcessedSemanticData] = useState<SemanticLineageChart | null>(null);
  const [hierarchicalData, setHierarchicalData] = useState<any | null>(null);
  const semanticData: any = null;
  const refetchSemanticData = () => {};
  
  // Chart selection state
  const [availableCharts, setAvailableCharts] = useState<any[]>([]);
  const [selectedChartId, setSelectedChartId] = useState<string | null>(null);

  // UI state
  const [selectedAsset, setSelectedAsset] = useState<EnhancedSelectedAsset | null>(null);
  const [selectedEdge, setSelectedEdge] = useState<Edge | null>(null);
  const [isRelationshipPanelOpen, setIsRelationshipPanelOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const debouncedSearchTerm = useDebounce(searchTerm, 300);
  const [highlightedItem, setHighlightedItem] = useState<string | null>(null);
  const [coreFilter, setCoreFilter] = useState<'all' | 'core' | 'custom'>('all');
  const [highlightedSuggestionIndex, setHighlightedSuggestionIndex] = useState<number>(-1);
  const [showSearchSuggestions, setShowSearchSuggestions] = useState(false);
  const [isSearchFocused, setIsSearchFocused] = useState(false);

  const [isExportViewVisible, setIsExportViewVisible] = useState(false);
  const [isColumnModalOpen, setIsColumnModalOpen] = useState(false);
  const [columnModalTableName, setColumnModalTableName] = useState<string | undefined>(undefined);
  const [columnModalColumns, setColumnModalColumns] = useState<ColumnData[]>([]);
  const [isLineageFullScreen, setIsLineageFullScreen] = useState(false);
  const [isErdFullScreen, setIsErdFullScreen] = useState(false);
  const [isSemanticFullScreen, setIsSemanticFullScreen] = useState(false);

  const catalogSummaries = useMemo(() => nodes.map((node) => deriveCatalogSummary(node)), [nodes]);
  const summaryById = useMemo(() => {
    const map = new Map<string, CatalogSummary>();
    catalogSummaries.forEach((summary) => map.set(summary.id, summary));
    return map;
  }, [catalogSummaries]);
  const nodesById = useMemo(() => {
    const map = new Map<string, FlowNode<TableNodeData>>();
    nodes.forEach((node) => map.set(node.id, node));
    return map;
  }, [nodes]);
  const coreStats = useMemo(() => {
    let core = 0;
    let custom = 0;
    catalogSummaries.forEach((summary) => {
      if (summary.isCore) {
        core += 1;
      } else {
        custom += 1;
      }
    });
    return {
      core,
      custom,
      total: catalogSummaries.length,
    };
  }, [catalogSummaries]);
  const allowedTableIds = useMemo(() => {
    const allowed = new Set<string>();
    catalogSummaries.forEach((summary) => {
      const coreMatches =
        coreFilter === 'all' ||
        (coreFilter === 'core' ? summary.isCore : !summary.isCore);
      if (coreMatches) {
        allowed.add(summary.id);
      }
    });
    return allowed;
  }, [catalogSummaries, coreFilter]);
  const filteredNodes = useMemo(
    () => nodes.filter((node) => allowedTableIds.has(node.id)),
    [nodes, allowedTableIds]
  );


  const { searchSuggestions, suggestionMetadata } = useMemo(() => {
    if (!searchTerm.trim()) {
      return { searchSuggestions: [] as SearchSuggestion[], suggestionMetadata: {} as Record<string, SearchResult> };
    }

    const loweredTerm = searchTerm.trim().toLowerCase();
    const metadata: Record<string, SearchResult> = {};
    const suggestions: SearchSuggestion[] = [];
    const seen = new Set<string>();

    const pushSuggestion = (result: SearchResult, suggestion: SearchSuggestion) => {
      if (seen.has(result.id)) {
        return;
      }
      seen.add(result.id);
      metadata[result.id] = result;
      suggestions.push(suggestion);
    };

    // Database tab: search tables and columns
    if (activeTab === 'database') {
      for (const node of filteredNodes) {
        const summary = summaryById.get(node.id);
        if (!summary) continue;

        const tableTokens = [summary.label, summary.tableName, summary.schemaName, summary.qualifiedPath]
          .filter(Boolean)
          .map((value) => String(value).toLowerCase());

        const matchesTable = tableTokens.some((token) => token.includes(loweredTerm));
        if (matchesTable) {
          const result: SearchResult = {
            id: `table-${node.id}`,
            kind: 'table',
            label: summary.label,
            nodeId: node.id,
            tableName: summary.tableName,
            isCore: summary.isCore,
            hasModel: summary.hasModel,
          };

          pushSuggestion(result, {
            id: result.id,
            title: summary.label,
            subtitle: `${summary.schemaName}.${summary.tableName}`,
            description: summary.hasModel ? 'Assigned model' : 'Unassigned',
            type: summary.isCore ? 'core' : 'custom',
          });
        }

        const columns = node.data?.columns ?? [];
        for (let index = 0; index < columns.length; index += 1) {
          if (suggestions.length >= 40) break;
          const column = columns[index];
          if (!column?.name) {
            continue;
          }
          const columnName = column.name.toLowerCase();
          const columnDescription = column.description?.toLowerCase();
          const matchesColumn =
            columnName.includes(loweredTerm) ||
            (typeof columnDescription === 'string' && columnDescription.includes(loweredTerm));
          if (!matchesColumn) continue;

          const result: SearchResult = {
            id: `column-${node.id}-${index}`,
            kind: 'column',
            label: column.name,
            nodeId: node.id,
            tableName: summary.tableName,
            columnIndex: index,
            isCore: summary.isCore,
            hasModel: summary.hasModel,
          };

          pushSuggestion(result, {
            id: result.id,
            title: column.name,
            subtitle: `${summary.schemaName}.${summary.tableName}`,
            description: `Column • ${summary.hasModel ? 'Assigned model' : 'Unassigned'}`,
            type: summary.isCore ? 'core' : 'custom',
          });
        }

        if (suggestions.length >= 40) {
          break;
        }
      }
    }

    // Semantic tab: search semantic assets
    if (activeTab === 'semantic') {
      for (const asset of semanticAssets) {
        if (suggestions.length >= 40) break;
        
        const assetName = (asset.name || asset.title || '').toLowerCase();
        const assetDescription = (asset.description || '').toLowerCase();
        
        if (assetName.includes(loweredTerm) || assetDescription.includes(loweredTerm)) {
          const assetType = asset.type || 'semantic_term';
          const result: SearchResult = {
            id: `semantic-${asset.id}`,
            kind: assetType as any,
            label: asset.name || asset.title || 'Unnamed',
            nodeId: asset.id,
            tableName: '',
            isCore: false,
            hasModel: false,
          };

          pushSuggestion(result, {
            id: result.id,
            title: asset.name || asset.title || 'Unnamed',
            subtitle: assetType.replace('_', ' '),
            description: asset.description || '',
            type: 'custom',
          });
        }
      }
    }

    // Diagram/ERD tab: search table names only (no columns for cleaner navigation)
    if (activeTab === 'diagram') {
      for (const node of filteredNodes) {
        if (suggestions.length >= 40) break;
        
        const summary = summaryById.get(node.id);
        if (!summary) continue;

        const tableTokens = [summary.label, summary.tableName, summary.schemaName, summary.qualifiedPath]
          .filter(Boolean)
          .map((value) => String(value).toLowerCase());

        const matchesTable = tableTokens.some((token) => token.includes(loweredTerm));
        if (matchesTable) {
          const result: SearchResult = {
            id: `erd-table-${node.id}`,
            kind: 'table',
            label: summary.label,
            nodeId: node.id,
            tableName: summary.tableName,
            isCore: summary.isCore,
            hasModel: summary.hasModel,
          };

          pushSuggestion(result, {
            id: result.id,
            title: summary.label,
            subtitle: `${summary.schemaName}.${summary.tableName}`,
            description: `${summary.columnCount} columns`,
            type: summary.isCore ? 'core' : 'custom',
          });
        }
      }
    }

    return { searchSuggestions: suggestions, suggestionMetadata: metadata };
  }, [filteredNodes, summaryById, searchTerm, activeTab, semanticAssets]);


  useEffect(() => {
    setHighlightedSuggestionIndex((current) => (current >= searchSuggestions.length ? -1 : current));
  }, [searchSuggestions.length]);

  useEffect(() => {
    if (isSearchFocused && searchTerm.trim().length > 0) {
      setShowSearchSuggestions(true);
    }
  }, [isSearchFocused, searchTerm, coreFilter, searchSuggestions.length]);

  // ERD controls state
  const [showColumns, setShowColumns] = useState<boolean>(
    () => localStorage.getItem('erdShowColumns') !== 'false'
  );
  const [showMiniMap, setShowMiniMap] = useState<boolean>(
    () => localStorage.getItem('erdShowMiniMap') !== 'false'
  );
  const [zoomLevel, setZoomLevel] = useState(1);
  const [reactFlowInstance, setReactFlowInstance] = useState<ReactFlowInstance | null>(null);
  const [isExporting, setIsExporting] = useState(false);

  // Detect GraphQL errors that indicate the tenant_chart field is missing
  const isTenantChartMissingError = (err: any) => {
    if (!err) return false;
    // Apollo error may include graphQLErrors array or a message string
    const msg = String(err.message || '').toLowerCase();
    if (msg.includes("field 'tenant_chart' not found") || msg.includes("field \"tenant_chart\" not found")) return true;
    if (Array.isArray(err.graphQLErrors)) {
      return err.graphQLErrors.some((g: any) => String(g.message || '').toLowerCase().includes("field 'tenant_chart' not found") || String(g.message || '').toLowerCase().includes('field "tenant_chart" not found'));
    }
    return false;
  };

  // When Hasura was not configured to expose tenant_chart, surface a friendly call-to-action
  const shouldShowMissingChartWarning = false;

  
  // Process raw REST data into UI format
  useEffect(() => {
    if (!rawNodes || !rawEdges) return;

    // Filter nodes by type
    const tables = rawNodes.filter((n: any) => n.node_type_id === '49a50271-ae58-4d3e-ae1c-2f5b89d89192');
    const columns = rawNodes.filter((n: any) => n.node_type_id === 'a64c1011-16e8-4ddf-b447-363bf8e15c9a');
    
    const businessTerms = rawNodes.filter((n: any) => n.node_type_id === '1b5d1a10-2f96-48c6-a734-710e20ec4221' || n.type === 'business_term');
    const semanticTerms = rawNodes.filter((n: any) => n.node_type_id === 'c5b8b9dc-3693-4fcb-8f74-729221fc3304' || n.type === 'semantic_term');

    // Build flow nodes for ERD
    const flowNodes: FlowNode<TableNodeData>[] = tables.map((t: any, index: number) => {
      const tableColumns = columns.filter((c: any) => c.parent_id === t.id);
      
      const columnData = tableColumns.map((col: any) => {
        const props = col.properties || {};
        return {
          name: col.node_name,
          type: props.data_type || 'unknown',
          nullable: props.is_nullable !== false,
          isPrimaryKey: props.is_primary_key === true,
          isForeignKey: props.is_foreign_key === true,
          isCore: props.is_core === true,
        };
      });

      return {
        id: t.id,
        type: 'databaseTable',
        position: { x: (index % 5) * 300, y: Math.floor(index / 5) * 200 },
        data: {
          label: t.node_name,
          tableName: t.node_name,
          schemaName: t.qualified_path?.split('.')[0] || 'public',
          isCore: t.properties?.is_core === true,
          columns: columnData,
        }
      };
    });

    const flowEdges: Edge[] = rawEdges.map((e: any) => ({
      id: e.id,
      source: e.source_node_id,
      target: e.target_node_id,
      type: 'smoothstep',
      animated: true,
      data: e.properties
    }));

    setNodes(flowNodes);
    setEdges(flowEdges);

    setProcessedTechnicalData({
      nodes: flowNodes,
      edges: flowEdges,
      viewport: { x: 0, y: 0, zoom: 1 },
      metadata: { chartType: 'technical_lineage' }
    });

    // Populate semantic assets
    setSemanticAssets([...businessTerms, ...semanticTerms, ...columns]);
    
    setProcessedSemanticData({
      businessTerms: enrichNodesWithTypes(businessTerms),
      semanticTerms: enrichNodesWithTypes(semanticTerms),
      semanticColumns: enrichNodesWithTypes(columns),
      databaseColumns: enrichNodesWithTypes(columns),
      edges: rawEdges,
      viewport: { x: 0, y: 0, zoom: 1 },
      metadata: { nodeCount: semanticTerms.length, edgeCount: rawEdges.length }
    });
    
  }, [rawNodes, rawEdges]);
const semanticEdgesFromGraphQL = rawEdges || [];

  // Render a helpful warning banner at the top of the modal when tenant_chart is not available
  
  // Fetch hierarchical data when selected asset changes
  useEffect(() => {
    const fetchHierarchicalData = async () => {
      if (!selectedAsset || !selectedAsset.qualifiedPath) {
        setHierarchicalData(null);
        return;
      }

      try {
        // The API endpoint for hierarchical data. Note: The exact path might need adjustment
        // to match your router configuration (e.g., adding a prefix like /api/v1).
        const response = await fetch(`/api/lineage/hierarchical/${datasourceId}`, {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ selectedAsset }),
        });

        if (!response.ok) {
          throw new Error(`Failed to fetch hierarchical data with status: ${response.status}`);
        }

        const data = await response.json();
        
        // Only set data if the layout is hierarchical
        setHierarchicalData(data.layout === 'hierarchical' ? data.data : null);
  } catch (error) {
    try { devError('Failed to fetch hierarchical data:', error); } catch {}
    setHierarchicalData(null);
  }
    };

    fetchHierarchicalData();
  }, [selectedAsset, datasourceId]);

  // Search functionality
  // Save preferences to localStorage
  useEffect(() => {
    localStorage.setItem('erdActiveTab', activeTab);
    localStorage.setItem('erdShowColumns', String(showColumns));
    localStorage.setItem('erdShowMiniMap', String(showMiniMap));
  }, [activeTab, showColumns, showMiniMap]);

  // Event Handlers
  const handleAssetSelect = useCallback(
    (asset: EnhancedSelectedAsset, options?: { preventTabSwitch?: boolean }) => {
      setSelectedAsset(asset);
      setSelectedEdge(null);
      
      // Auto-switch tab based on asset type
      // Auto-switch tab based on asset type
      if (!options?.preventTabSwitch) {
        if (asset.type === 'business_term' || asset.type === 'semantic_term' || asset.type === 'semantic_model') {
          setActiveTab('semantic');
        } else if (asset.type === 'table' || asset.type === 'column' || asset.type === 'schema') {
          setActiveTab('database');
        } else {
          // Default to database for other types or unknown
          setActiveTab('database');
        }
      }
      
      if (asset.nodeId && reactFlowInstance && (asset.type === 'table' || asset.type === 'column')) {
        const node = nodesById.get(asset.nodeId);
        if (node) {
          reactFlowInstance.setCenter(
            node.position.x + (node.width || 200) / 2,
            node.position.y + (node.height || 100) / 2,
            { zoom: 1.2 }
          );
        }
      }
      
      setHighlightedItem(asset.id);
    },
    [nodesById, reactFlowInstance]
  );

  // This effect handles auto-selection when a tab becomes active.
  useEffect(() => {
    if (activeTab === 'database') {
      const targetNodes = filteredNodes.length > 0 ? filteredNodes : nodes;
      if (targetNodes.length === 0) {
        return;
      }
      const firstNode = targetNodes[0];
      const asset: EnhancedSelectedAsset = {
        type: 'table',
        id: `table-${firstNode.id}`,
        nodeId: firstNode.id,
        tableName: firstNode.data?.label || 'Unknown Table',
        name: firstNode.data?.label || 'Unknown Table',
        node: firstNode,
        columns: firstNode.data?.columns || [],
        isCore: firstNode.data?.isCore,
      };
      handleAssetSelect(asset, { preventTabSwitch: true });
    }
  }, [activeTab, nodes, filteredNodes, handleAssetSelect]);

  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: FlowNode) => {
      const tableNode = node as FlowNode<TableNodeData>;
      const asset: EnhancedSelectedAsset = {
        type: 'table',
        id: `table-${tableNode.id}`,
        nodeId: tableNode.id,
        tableName: tableNode.data?.label || 'Unknown Table',
        name: tableNode.data?.label || 'Unknown Table',
        node: tableNode,
        columns: tableNode.data?.columns || [],
        isCore: tableNode.data?.isCore,
      };
      setSelectedAsset(asset);
      setHighlightedItem(asset.id);
      setSelectedEdge(null);
    },
    []
  );

  const handleEdgeClick = useCallback(
    (_: React.MouseEvent, edge: Edge) => {
      setSelectedEdge(edge);
      setIsRelationshipPanelOpen(true);
    },
    []
  );

  const handleCloseRelationshipPanel = useCallback(() => {
    setIsRelationshipPanelOpen(false);
  }, []);

  const handleSearchChange = (term: string) => {
    setSearchTerm(term);
    const hasValue = term.trim().length > 0;
    setShowSearchSuggestions(hasValue && isSearchFocused);
  };

  const handleSearchFocus = () => {
    setIsSearchFocused(true);
    if (searchTerm.trim()) {
      setShowSearchSuggestions(true);
    }
  };

  const handleSearchBlur = () => {
    setIsSearchFocused(false);
    setShowSearchSuggestions(false);
  };

  const handleSearchSelect = useCallback(
    (suggestion: SearchSuggestion) => {
      const item = suggestionMetadata[suggestion.id];
      if (!item) {
        devWarn('Search item not found in metadata:', suggestion.id);
        return;
      }

      const node = nodesById.get(item.nodeId);
      if (!node) {
        devWarn('Node not found for search result:', item.nodeId);
        return;
      }

      let asset: EnhancedSelectedAsset | null = null;
      if (item.kind === 'table') {
        asset = {
          type: 'table',
          id: item.id,
          nodeId: item.nodeId,
          tableName: item.tableName,
          name: item.label,
          node,
          columns: node.data?.columns || [],
          isCore: item.isCore,
        };
      } else if (typeof item.columnIndex === 'number' && node.data?.columns) {
        const column = node.data.columns[item.columnIndex];
        if (!column) {
          return;
        }
        asset = {
          type: 'column',
          id: item.id,
          nodeId: item.nodeId,
          tableName: item.tableName,
          columnName: column.name,
          name: column.name,
          column,
          isCore: Boolean(column.isCore ?? item.isCore),
        };
      }

      if (!asset) {
        return;
      }

      // Don't force tab switch - stay on current tab
      handleAssetSelect(asset, { preventTabSwitch: true });

      // If on diagram tab, highlight and center the selected node
      if (activeTab === 'diagram') {
        // Set highlighted item to make the node stand out
        setHighlightedItem(item.nodeId);
        
        // Center the node in the viewport
        if (reactFlowInstance) {
          // Get the actual node from ReactFlow instance to ensure we have the computed layout position
          // The 'node' variable from nodesById might have initial (0,0) coordinates if layout happens in the child component
          const flowNode = reactFlowInstance.getNode(item.nodeId);
          
          if (flowNode && flowNode.position) {
            const nodeWidth = flowNode.width || 200;
            const nodeHeight = flowNode.height || 100;
            
            setTimeout(() => {
              reactFlowInstance.setCenter(
                flowNode.position.x + nodeWidth / 2,
                flowNode.position.y + nodeHeight / 2,
                { zoom: 1.5, duration: 600 }
              );
            }, 100);
          } else {
             // Fallback to initial node if flowNode not found (unlikely if rendered)
             const fallbackNode = node;
             const fallbackWidth = fallbackNode.width || 200;
             const fallbackHeight = fallbackNode.height || 100;
             
             if (fallbackNode.position && (fallbackNode.position.x !== 0 || fallbackNode.position.y !== 0)) {
                setTimeout(() => {
                  reactFlowInstance.setCenter(
                    fallbackNode.position.x + fallbackWidth / 2,
                    fallbackNode.position.y + fallbackHeight / 2,
                    { zoom: 1.5, duration: 600 }
                  );
                }, 100);
             } else {
                 devWarn('Cannot center node: position is (0,0) or undefined', item.nodeId);
             }
          }
        }
      }

      setSearchTerm('');
      setShowSearchSuggestions(false);
      setIsSearchFocused(false);
      setHighlightedSuggestionIndex(-1);
    },
    [suggestionMetadata, nodesById, handleAssetSelect, activeTab, reactFlowInstance]
  );

  const handleSearchClear = () => {
    setSearchTerm('');
    setShowSearchSuggestions(false);
    setHighlightedSuggestionIndex(-1);
  };

  const handleExportClick = () => {
    if (!nodes || nodes.length === 0) {
      devLog('No nodes available for export');
      return;
    }
    setIsExportViewVisible(true);
  };

  const handleCancelExport = () => {
    setIsExportViewVisible(false);
  };

  const handleActualExport = async (options: ExportOptions) => {
    setIsExporting(true);
    try {
      await exportData(nodes, edges, options);
      setIsExportViewVisible(false);
    } catch (error) {
      devError('Export failed:', error);
      const notification = useNotification();
      notification.error('Export failed. Please check developer logs for details and try again.');
    } finally {
      setIsExporting(false);
    }
  };

  // ERD Control Handlers
  const onInit = (rfi: ReactFlowInstance) => {
    setReactFlowInstance(rfi);
    setZoomLevel(rfi.getZoom());
    const savedViewport = localStorage.getItem('erdViewport');
    if (savedViewport) {
      rfi.setViewport(JSON.parse(savedViewport));
    }
  };

  const handleZoomChange = (newZoom: number) => {
    if (reactFlowInstance) {
      reactFlowInstance.zoomTo(newZoom);
      setZoomLevel(newZoom);
      localStorage.setItem('erdViewport', JSON.stringify(reactFlowInstance.getViewport()));
    }
  };

  const handlePaneClick = () => {
    setHighlightedItem(null);
    setSelectedEdge(null);
  };
  
  const handleToggleColumns = () => setShowColumns(!showColumns);
  const handleToggleMiniMap = () => setShowMiniMap(!showMiniMap);
  const handleFitView = () => reactFlowInstance?.fitView();
  const exportToPng = async () => {
    // Implementation for PNG export
  };

  // Loading and error states
  // Only block on critical chart loading for initial render
  // Other queries can load in the background with per-tab indicators
  if (isLoading && !processedTechnicalData && !processedSemanticData) {
    console.log('⏳ TabbedModal: Showing loading state (isLoading && !processedTechnicalData && !processedSemanticData)');
    return (
      <div className="tabbed-modal-loading" style={{ 
        display: 'flex', 
        flexDirection: 'column', 
        alignItems: 'center', 
        justifyContent: 'center', 
        height: '100vh',
        gap: '16px'
      }}>
        <div>Loading schema data...</div>
        <div style={{ fontSize: '0.875rem', color: '#666' }}>
          Fetching database catalog and lineage information
        </div>
      </div>
    );
  }

  // Show error only if chart query failed (most critical)
  if (isError && !shouldShowMissingChartWarning) {
    const errorMsg = nodesError?.message || edgesError?.message || 'Unknown error';
    console.log('❌ TabbedModal: Showing error state', { errorMsg });
    return (
      <div className="tabbed-modal-error" style={{ padding: '24px' }}>
        <h3>Error loading schema data</h3>
        <p>{errorMsg}</p>
        <button onClick={() => window.location.reload()} style={{ marginTop: '16px', padding: '8px 16px', cursor: 'pointer' }}>
          Retry
        </button>
      </div>
    );
  }

  console.log('✅ TabbedModal: Rendering main UI', { 
    isModal, 
    activeTab, 
    nodesCount: nodes.length,
    chartEdgesCount: edges.length,
    semanticEdgesCount: semanticEdgesFromGraphQL.length
  });

  return (
    <div className={isModal ? "tabbed-modal-container" : "schema-explorer-container"}>
      <div className={isModal ? "tabbed-modal-header" : "schema-explorer-header"}>
        <div className="tabs">
          <button
            className={`tab ${activeTab === 'database' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('database');
              setIsExportViewVisible(false);
            }}
          >
            📊 Technical Assets ({filteredNodes.length}{filteredNodes.length !== nodes.length ? ` / ${nodes.length}` : ''})
          </button>
          <button
            className={`tab ${activeTab === 'semantic' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('semantic');
              setIsExportViewVisible(false);
            }}
          >
            🌳 Semantic
          </button>
          <button
            className={`tab ${activeTab === 'diagram' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('diagram');
              setIsExportViewVisible(false);
            }}
          >
            📈 ERD Diagram
          </button>
          <button
            className={`tab ${activeTab === 'lineage' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('lineage');
              setIsExportViewVisible(false);
            }}
          >
            🕸️ Lineage
          </button>

        </div>

        <div className="header-controls">
          {activeTab === 'diagram' && availableCharts.length > 1 && (
            <select
              id="chart-selector"
              aria-label="Select chart"
              title="Select chart"
              className="chart-selector"
              value={selectedChartId || ''}
              onChange={(e) => setSelectedChartId(e.target.value)}
              style={{ 
                marginRight: 12, 
                padding: '8px 12px', 
                borderRadius: '6px', 
                border: '1px solid #e2e8f0',
                backgroundColor: 'white',
                fontSize: '0.9rem',
                color: '#1e293b',
                cursor: 'pointer',
                outline: 'none',
                height: '36px'
              }}
            >
              {availableCharts.map((chart) => (
                <option key={chart.id} value={chart.id}>
                  {chart.chart_name === 'semantic_lineage_chart' ? 'Semantic Lineage' : 
                   chart.chart_name === 'erd_chart' ? 'ERD Diagram' : 
                   chart.chart_name === 'enhanced_erd_chart' ? 'Enhanced ERD' :
                   chart.chart_name.replace(/_/g, ' ')}
                </option>
              ))}
            </select>
          )}

          <ProfessionalSearchInput
            value={searchTerm}
            onChange={handleSearchChange}
            onClear={handleSearchClear}
            placeholder="Search tables or columns..."
            suggestions={searchSuggestions}
            showSuggestions={showSearchSuggestions}
            onSuggestionSelect={handleSearchSelect}
            onFocus={handleSearchFocus}
            onBlur={handleSearchBlur}
            highlightedIndex={highlightedSuggestionIndex}
            onHighlightChange={setHighlightedSuggestionIndex}
            variant="enhanced"
            className="header-search-input"
          />
          <Suspense fallback={<div className="suspense-fallback-40" />}>
            <ExportButton
              onClick={handleExportClick}
              disabled={!nodes || nodes.length === 0}
            />
          </Suspense>
          {isModal && (
            <button
              onClick={onClose}
              className="close-btn"
              aria-label="Close"
              title="Close"
            >
              ✕
            </button>
          )}
        </div>
      </div>

      {/* Filters toolbar - hidden on diagram tab */}
      {activeTab !== 'diagram' && (
        <div className="catalog-toolbar">
          <div className="catalog-toolbar-filters">
            <div className="filter-group">
              <span className="filter-label">Model type</span>
              <button
                type="button"
                className={`filter-chip ${coreFilter === 'all' ? 'active' : ''}`}
                onClick={() => setCoreFilter('all')}
              >
                All
                <span className="filter-count">{coreStats.total}</span>
              </button>
              <button
                type="button"
                className={`filter-chip ${coreFilter === 'core' ? 'active' : ''}`}
                onClick={() => setCoreFilter('core')}
              >
                Core
                <span className="filter-count">{coreStats.core}</span>
              </button>
              <button
                type="button"
                className={`filter-chip ${coreFilter === 'custom' ? 'active' : ''}`}
                onClick={() => setCoreFilter('custom')}
              >
                Custom
                <span className="filter-count">{coreStats.custom}</span>
              </button>
            </div>
          </div>
        </div>
      )}

      <div className={isModal ? "tabbed-modal-content" : "schema-explorer-content"}>
        {activeTab === 'database' && (
          <Suspense fallback={<div>Loading database view...</div>}>
            <DatabaseCatalogView
            nodes={filteredNodes}
            edges={edges}
            selectedAsset={selectedAsset}
            selectedEdge={selectedEdge}
            highlightedItem={highlightedItem}
            searchTerm={debouncedSearchTerm}
            showColumns={showColumns}
            onAssetSelect={handleAssetSelect}
            onEdgeClick={handleEdgeClick}
            onColumnCountClick={(node: FlowNode<TableNodeData>) => {
              const cols = (node.data?.columns ?? []) as ColumnData[];
              // Enrich columns with semantic terms from edges
              const enrichedCols = cols.map((col: ColumnData) => {
                const colQualified = `${node.data?.tableName || node.id}.${col.name}`;
                // Find semantic terms connected to this column via edges
                const connectedTerms = (semanticData?.semantic_terms || []).filter((term: any) => {
                  // Check if there's an edge connecting this term to this column
                  return (semanticData?.semantic_edges || []).some((edge: any) =>
                    (edge.source === term.id && edge.target?.includes(col.name)) ||
                    (edge.target === term.id && edge.source?.includes(col.name)) ||
                    // Also check properties for column references
                    (term.properties?.column === col.name || term.properties?.qualified_column === colQualified)
                  );
                });
                return { ...col, semanticTerms: connectedTerms };
              });
              setColumnModalTableName(node.data?.label || node.data?.tableName || node.id || 'table');
              setColumnModalColumns(enrichedCols as ColumnData[]);
              setIsColumnModalOpen(true);
            }}
            onTotalColumnsClick={(cols: any[], label?: string) => {
              setColumnModalTableName(label || 'All Tables');
              setColumnModalColumns(cols as ColumnData[]);
              setIsColumnModalOpen(true);
            }}
            onOpenColumnsModal={(label: string, cols: any[]) => {
              setColumnModalTableName(label || 'table');
              setColumnModalColumns(cols as ColumnData[]);
              setIsColumnModalOpen(true);
            }}
            isRelationshipPanelOpen={isRelationshipPanelOpen}
            onCloseRelationshipPanel={handleCloseRelationshipPanel}
            forceLineageType="technical"
            processedTechnicalData={processedTechnicalData}
            processedSemanticData={processedSemanticData}
            hierarchicalData={hierarchicalData}
            preferHierarchical={true}
            datasourceId={datasourceId}
            tenantId={tenantId}
            onRefresh={() => {
              // Refetch data after mappings are applied
              window.location.reload(); // Simple refresh for now
            }}
            />
          </Suspense>
        )}

        {activeTab === 'semantic' && (
          <Suspense fallback={<div>Loading semantic view...</div>}>
            <SemanticCatalogView
              semanticAssets={semanticData?.semantic_terms || []}
              selectedAsset={selectedAsset}
              onAssetSelect={handleAssetSelect}
              searchTerm={debouncedSearchTerm}
              highlightedItem={highlightedItem}
              semanticData={{
                nodes: semanticData?.semantic_terms || [],
                edges: semanticEdgesFromGraphQL,
                businessTerms: semanticData?.business_terms || []
              }}
              technicalData={processedTechnicalData}
              datasourceId={datasourceId}
              tenantId={tenantId}
              onToggleFullScreen={() => setIsSemanticFullScreen(true)}
              isFullScreen={false}
              onRefresh={() => {
                refetchSemanticData();
              }}
            />
          </Suspense>
        )}

        {activeTab === 'diagram' && (
          <div className="diagram-tab" style={{ height: '100%', width: '100%', minHeight: '500px' }}>
            {(() => {
              console.log('🎯 RENDERING DIAGRAM TAB - activeTab === diagram:', activeTab === 'diagram', 'Nodes count:', nodes.length, 'Edges count:', edges.length);
              devDebug('ERD: Rendering diagram tab. Total nodes:', nodes.length, 'Filtered nodes:', filteredNodes.length, 'Edges:', edges.length);
              devDebug('ERD: First 3 nodes:', nodes.slice(0, 3).map(n => ({ id: n.id, label: n.data?.label, position: n.position })));
              return null;
            })()}
            {nodes.length === 0 ? (
              <div style={{ 
                display: 'flex', 
                flexDirection: 'column', 
                alignItems: 'center', 
                justifyContent: 'center', 
                height: '100%', 
                textAlign: 'center',
                color: '#64748b',
                padding: '24px'
              }}>
                <div style={{ fontSize: '64px', marginBottom: '16px' }}>📊</div>
                <h2 style={{ margin: '0 0 8px 0', color: '#334155' }}>No ERD Chart Available</h2>
                <p style={{ margin: '0 0 16px 0', maxWidth: '400px' }}>
                  The ERD chart for this datasource has not been generated yet. 
                  Please run a catalog scan or chart generation to create the ERD visualization.
                </p>
                <p style={{ margin: '0', fontSize: '12px', color: '#94a3b8' }}>
                  Datasource ID: {datasourceId}
                </p>
              </div>
            ) : (
              <Suspense fallback={<div>Loading diagram...</div>}>
                <ErdDiagram
                nodes={nodes}
                edges={edges}
                nodeTypes={nodeTypes}
                showColumns={showColumns}
                showMiniMap={showMiniMap}
                highlightedItem={highlightedItem}
                zoomLevel={zoomLevel}
                onInit={onInit}
                onNodeClick={handleNodeClick}
                onEdgeClick={handleEdgeClick}
                onPaneClick={handlePaneClick}
                onMoveEnd={(_event: unknown, viewport: { x: number; y: number; zoom: number }) =>
                  localStorage.setItem('erdViewport', JSON.stringify(viewport))
                }
                onZoomChange={handleZoomChange}
                onToggleColumns={handleToggleColumns}
                onToggleMiniMap={handleToggleMiniMap}
                onFitView={handleFitView}
                />
              </Suspense>
            )}
          </div>
        )}

        {activeTab === 'lineage' && (
          <Suspense fallback={<div>Loading lineage viewer...</div>}>
            <DualLineageViewer
            selectedAsset={selectedAsset}
            technicalData={processedTechnicalData}
            semanticData={processedSemanticData}
            hierarchicalData={hierarchicalData}
              preferHierarchical={true}
              onAssetClick={handleAssetSelect}
              onRelationshipClick={(edge: Edge) => setSelectedEdge(edge)}
              onToggleFullScreen={() => setIsLineageFullScreen(true)}
              isFullScreen={false}
            />
          </Suspense>
        )}
      </div>

      {isExportViewVisible && (
        <Suspense fallback={<div>Preparing export...</div>}>
          <EnhancedExportOverlay
            nodes={nodes}
            edges={edges}
            onExport={handleActualExport}
            onCancel={handleCancelExport}
          />
  </Suspense>
      )}
      {isColumnModalOpen && (
        <ColumnDetailsModal
          open={isColumnModalOpen}
          onClose={() => setIsColumnModalOpen(false)}
          tableName={columnModalTableName}
          columns={columnModalColumns}
        />
      )}

      {isLineageFullScreen && (
        <div>
          <Suspense fallback={<div>Loading lineage viewer...</div>}>
            <DualLineageViewer
              selectedAsset={selectedAsset}
              technicalData={processedTechnicalData}
              semanticData={processedSemanticData}
              hierarchicalData={hierarchicalData}
              preferHierarchical={true}
              onAssetClick={handleAssetSelect}
              onRelationshipClick={(edge: Edge) => setSelectedEdge(edge)}
              onToggleFullScreen={() => setIsLineageFullScreen(false)}
              isFullScreen={true}
            />
          </Suspense>
        </div>
      )}

      {/* ERD Fullscreen Overlay */}
      {isErdFullScreen && (
        <div className="erd-fullscreen-overlay">
          <div className="erd-fullscreen-header">
            <h2>ERD Diagram - Fullscreen</h2>
            <button
              className="erd-fullscreen-close"
              onClick={() => setIsErdFullScreen(false)}
              aria-label="Exit fullscreen"
            >
              ✕ Close
            </button>
          </div>
          <div className="erd-fullscreen-content">
            <Suspense fallback={<div>Loading diagram controls...</div>}>
              <ErdControls
                zoomLevel={zoomLevel}
                showColumns={showColumns}
                showMiniMap={showMiniMap}
                isExporting={isExporting}
                isFullScreen={true}
                onZoomChange={handleZoomChange}
                onToggleColumns={handleToggleColumns}
                onToggleMiniMap={handleToggleMiniMap}
                onFitView={handleFitView}
                onExportPng={exportToPng}
                onToggleFullScreen={() => setIsErdFullScreen(false)}
              />
            </Suspense>
            <Suspense fallback={<div>Loading diagram...</div>}>
              <ErdDiagram
                nodes={filteredNodes}
                edges={edges}
                nodeTypes={nodeTypes}
                showColumns={showColumns}
                showMiniMap={showMiniMap}
                highlightedItem={highlightedItem}
                onInit={onInit}
                onNodeClick={handleNodeClick}
                onEdgeClick={handleEdgeClick}
                onPaneClick={handlePaneClick}
                onMoveEnd={(_event: unknown, viewport: { x: number; y: number; zoom: number }) =>
                  localStorage.setItem('erdViewport', JSON.stringify(viewport))
                }
              />
            </Suspense>
          </div>
        </div>
      )}

      {/* Semantic Fullscreen Overlay */}
      {isSemanticFullScreen && (
        <div className="erd-fullscreen-overlay" style={{ zIndex: 2000 }}>
          <div className="erd-fullscreen-header">
            <h2>Semantic Lineage - Fullscreen</h2>
            <button
              className="erd-fullscreen-close"
              onClick={() => setIsSemanticFullScreen(false)}
              aria-label="Exit fullscreen"
            >
              ✕ Close
            </button>
          </div>
          <div className="erd-fullscreen-content">
            <Suspense fallback={<div>Loading semantic view...</div>}>
              <SemanticCatalogView
                semanticAssets={semanticData?.semantic_terms || []}
                selectedAsset={selectedAsset}
                onAssetSelect={handleAssetSelect}
                searchTerm={debouncedSearchTerm}
                highlightedItem={highlightedItem}
                semanticData={processedSemanticData}
                technicalData={processedTechnicalData}
                datasourceId={datasourceId}
                tenantId={tenantId}
                onToggleFullScreen={() => setIsSemanticFullScreen(false)}
                isFullScreen={true}
                onRefresh={() => {
                  refetchSemanticData();
                }}
              />
            </Suspense>
          </div>
        </div>
      )}
    </div>
  );
};

export default TabbedModal;