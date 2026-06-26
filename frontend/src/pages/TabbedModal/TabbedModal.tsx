// 1. Complete TabbedModal.tsx - Fixed with all necessary code
import { useState, useEffect, useMemo, useCallback, lazy, Suspense } from 'react';
import { useNotification } from '../../hooks/useNotification';
import { devLog, devWarn, devError, devDebug } from '../../utils/devLogger';
import { useQuery } from '@apollo/client';
import { Node as FlowNode, Edge, ReactFlowInstance } from 'reactflow';

// Services and Types
import { exportData } from '../../services/exportService';
import { ExportOptions } from '../../types/ExportTypes';
import { ColumnData, EnhancedSelectedAsset } from '../../types/SemanticTypes';
import { enrichNodesWithTypes } from '../../utils/nodeTypeMapping';

// Queries
import { 
  GET_COMBINED_CHART, 
  GET_ALL_SEMANTIC_DATA,
  GET_TECHNICAL_LINEAGE_CHART,
  GET_SEMANTIC_LINEAGE_CHART,
  transformChartData
} from '../../graphql/queries/semantic';

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
  
  // GraphQL Queries
  const { loading: chartLoading, error: chartError, data: chartData } = useQuery(GET_COMBINED_CHART, {
    variables: { datasourceId },
  });

  console.log('📈 TabbedModal - Query states:', {
    chartLoading,
    hasChartError: !!chartError,
    hasChartData: !!chartData,
    chartDataKeys: chartData ? Object.keys(chartData) : []
  });

  const { loading: semanticLoading, error: semanticError, data: semanticData, refetch: refetchSemanticData } = useQuery(GET_ALL_SEMANTIC_DATA, {
    variables: { datasourceId },
  });

  const { loading: technicalLineageLoading, data: technicalLineageData } = useQuery(GET_TECHNICAL_LINEAGE_CHART, {
    variables: { datasourceId },
  });

  const { loading: semanticLineageLoading, data: semanticLineageData } = useQuery(GET_SEMANTIC_LINEAGE_CHART, {
    variables: { datasourceId },
  });

  // Log chart data when it arrives
  useEffect(() => {
    devDebug('🔍 GET_COMBINED_CHART completed:', {
      hasData: !!chartData,
      hasTenantChart: !!chartData?.tenant_chart,
      chartCount: chartData?.tenant_chart?.length || 0,
      datasourceId
    });
    console.log('📊 TabbedModal - Chart Data Received:', chartData);
    if (chartData?.tenant_chart?.length > 0) {
      console.log('✅ Chart data details:', {
        chart_name: chartData.tenant_chart[0].chart_name,
        chart_length: chartData.tenant_chart[0].chart?.length,
        chart_type: typeof chartData.tenant_chart[0].chart
      });
    } else if (chartData) {
      console.warn('⚠️ No chart data found in response');
    }
  }, [chartData]);

  // Log chart errors
  useEffect(() => {
    if (chartError) {
      devError('❌ GET_COMBINED_CHART error:', chartError.message);
      console.error('Full chart query error:', chartError);
    }
  }, [chartError]);

  // Log semantic data when it arrives
  useEffect(() => {
    console.log('📋 [TabbedModal] GET_ALL_SEMANTIC_DATA returned:', {
      hasData: !!semanticData,
      loading: semanticLoading,
      hasError: !!semanticError,
      keys: semanticData ? Object.keys(semanticData) : [],
      businessTermsCount: semanticData?.business_terms?.length || 0,
      semanticTermsCount: semanticData?.semantic_terms?.length || 0,
      semanticColumnsCount: semanticData?.semantic_columns?.length || 0,
      databaseColumnsCount: semanticData?.databaseColumns?.length || 0,
      semanticEdgesCount: semanticData?.semantic_edges?.length || 0
    });
    
    // Log the actual edges array to see what's in it
    if (semanticData?.semantic_edges) {
      console.log('🔗 [TabbedModal] semantic_edges array:', semanticData.semantic_edges);
    } else {
      console.warn('⚠️ [TabbedModal] semantic_edges is missing or undefined in semanticData');
    }
  }, [semanticData, semanticLoading, semanticError]);

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
  const shouldShowMissingChartWarning = Boolean(chartError && isTenantChartMissingError(chartError));

  // Process Chart Data (ERD/Lineage)
  useEffect(() => {
    if (shouldShowMissingChartWarning) {
      devWarn('tenant_chart missing in GraphQL schema — skipping chart parsing');
      setNodes([]);
      setEdges([]);
      setAvailableCharts([]);
      return;
    }

    if (chartData && chartData.tenant_chart && chartData.tenant_chart.length > 0) {
      const charts = chartData.tenant_chart;
      setAvailableCharts(charts);
      
      // Determine which chart to display
      // 1. If user selected one, try to find it
      // 2. If not, default to the first one
      let activeChart = charts[0];
      if (selectedChartId) {
        const found = charts.find((c: any) => c.id === selectedChartId);
        if (found) {
          activeChart = found;
        } else {
          // If selected ID not found (e.g. stale), fallback to first
          setSelectedChartId(charts[0].id);
        }
      } else {
        // Initial load: select first
        setSelectedChartId(charts[0].id);
      }

      const compressedChartAsHex = activeChart.chart;
      devDebug('Processing Chart:', activeChart.chart_name, 'ID:', activeChart.id);
      
      // Ensure we have valid hex string
      if (!compressedChartAsHex || typeof compressedChartAsHex !== 'string') {
        devError('Invalid chart data - not a string:', typeof compressedChartAsHex);
        return;
      }
      
      const parsedChartData = transformChartData(compressedChartAsHex) as TechnicalLineageChart;
      
      if (parsedChartData && parsedChartData.nodes) {
        const nodesToSet = Array.isArray(parsedChartData.nodes) ? parsedChartData.nodes : [];
        const edgesToSet = Array.isArray(parsedChartData.edges) ? parsedChartData.edges : [];
        
        setNodes(nodesToSet);
        setEdges(edgesToSet);
        
        // Only select edge if none selected or if previously selected is gone
        // (Optional refinement: keep selection if valid)
        if (edgesToSet.length > 0 && !selectedEdge) {
          setSelectedEdge(edgesToSet[0]);
        }
        devDebug('Chart State Updated:', nodesToSet.length, 'nodes', edgesToSet.length, 'edges');
      } else {
        devError('Failed to parse chart data');
      }
    } else {
      devWarn('No chart data found');
      setAvailableCharts([]);
      setNodes([]);
      setEdges([]);
    }
  }, [chartData, selectedChartId]);

  // Compute semantic edges from GraphQL data (for semantic tab only)
  // Chart data should only be used for ERD diagram
  // Keep the original GraphQL format since SemanticCatalogView expects source_node_id/target_node_id
  const semanticEdgesFromGraphQL = useMemo(() => {
    if (!semanticData?.semantic_edges) return [];
    
    // Return edges in their original GraphQL format
    // SemanticCatalogView expects: source_node_id, target_node_id, edge_type_id, etc.
    return semanticData.semantic_edges;
  }, [semanticData?.semantic_edges]);

  // Render a helpful warning banner at the top of the modal when tenant_chart is not available
  if (shouldShowMissingChartWarning) {
    return (
      <div className="tabbed-modal-missing-chart">
        <div style={{ padding: 24 }}>
          <h3>Lineage/ERD data not available</h3>
          <p>
            The GraphQL field <code>tenant_chart</code> is not present in your Hasura schema for the connected
            database. This means the Schema Explorer cannot load ERD or lineage charts.
          </p>
          <p>
            Possible fixes:
          </p>
          <ul>
            <li>Ensure the <code>tenant_chart</code> table exists in the database (public.tenant_chart).</li>
            <li>Register the <code>tenant_chart</code> table in Hasura metadata (expose select permissions).</li>
            <li>Run platform migrations that add tenant_chart / apply Hasura metadata.</li>
          </ul>
        </div>
      </div>
    );
  }

  // Process semantic data
  useEffect(() => {
    if (semanticData) {
      const assets = [
        ...(semanticData.business_terms || []),
        ...(semanticData.semantic_terms || []),
        ...(semanticData.semantic_columns || []),
      ];
      setSemanticAssets(assets);
    }
  }, [semanticData]);

  // Process technical lineage data
  useEffect(() => {
    if (technicalLineageData && technicalLineageData.tenant_chart && technicalLineageData.tenant_chart.length > 0) {
      const compressedChartAsHex = technicalLineageData.tenant_chart[0].chart;
      const parsedData = transformChartData(compressedChartAsHex) as TechnicalLineageChart;
      if (parsedData) {
        setProcessedTechnicalData(parsedData);
      }
    } else if (nodes.length > 0 && edges.length > 0) {
      setProcessedTechnicalData({
        nodes,
        edges,
        viewport: { x: 0, y: 0, zoom: 1 },
        metadata: {
          chartType: 'technical_lineage',
          databaseNodeCount: nodes.length,
          databaseEdgeCount: edges.length,
        }
      });
    }
  }, [technicalLineageData, nodes, edges]);

  // Process semantic lineage data from catalog nodes and edges
  useEffect(() => {
    if (semanticLineageData && semanticLineageData.tenant_chart && semanticLineageData.tenant_chart.length > 0) {
      const compressedChartAsHex = semanticLineageData.tenant_chart[0].chart;
      const parsedData = transformChartData(compressedChartAsHex) as SemanticLineageChart;
      if (parsedData) {
        setProcessedSemanticData(parsedData);
      }
    } else if (semanticData) {
      // Build semantic lineage from catalog data when tenant_chart is not available
      const businessTerms = enrichNodesWithTypes(semanticData.business_terms || []);
      const semanticTerms = enrichNodesWithTypes(semanticData.semantic_terms || []);
      const semanticColumns = enrichNodesWithTypes(semanticData.semantic_columns || []);
      const databaseColumns = enrichNodesWithTypes(semanticData.databaseColumns || []);
      const edges = semanticData.semantic_edges || [];
      
      console.log('📊 [TabbedModal] Processing semantic data:', {
        businessTermsCount: businessTerms.length,
        semanticTermsCount: semanticTerms.length,
        semanticColumnsCount: semanticColumns.length,
        databaseColumnsCount: databaseColumns.length,
        edgesCount: edges.length,
        hasSemanticData: !!semanticData,
        semanticDataKeys: semanticData ? Object.keys(semanticData) : []
      });
      
      if (semanticTerms.length > 0 || edges.length > 0) {
        const lineageData: SemanticLineageChart = {
          businessTerms,
          semanticTerms,
          semanticColumns,
          databaseColumns,
          edges,
          viewport: { x: 0, y: 0, zoom: 1 },
          metadata: { nodeCount: semanticTerms.length, edgeCount: edges.length }
        };
        console.log('✅ [TabbedModal] Setting processedSemanticData:', {
          nodeCount: lineageData.metadata.nodeCount,
          edgeCount: lineageData.metadata.edgeCount
        });
        setProcessedSemanticData(lineageData);
      } else {
        console.log('⚠️ [TabbedModal] Skipping semantic data - no terms or edges');
      }
    }
  }, [semanticLineageData, semanticData]);

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
  if (chartLoading && !chartData) {
    console.log('⏳ TabbedModal: Showing loading state (chartLoading && !chartData)');
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
  if (chartError && !shouldShowMissingChartWarning) {
    console.log('❌ TabbedModal: Showing error state', { chartError: chartError.message });
    return (
      <div className="tabbed-modal-error" style={{ padding: '24px' }}>
        <h3>Error loading schema data</h3>
        <p>{chartError.message}</p>
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
            📊 Database ({filteredNodes.length}{filteredNodes.length !== nodes.length ? ` / ${nodes.length}` : ''})
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