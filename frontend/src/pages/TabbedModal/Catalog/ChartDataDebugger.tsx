// ChartDataDebugger.tsx - Debug component to validate chart data loading
import { useState, useEffect, useMemo } from 'react';
import './ChartDataDebugger.css';
import { devLog, devDebug, devError } from '../../../api';
import DataCatalogTree from '../tabs/DataCatalogTree';
import { EnhancedSelectedAsset } from '../../../types/SemanticTypes';
import LineageDiagram from '../../../components/LineageDiagram';

interface ChartDataDebuggerProps {
  datasourceId: string;
  onChartDataLoaded?: (data: ChartData | Record<string, unknown>) => void;
}

interface DebugInfoEntry {
  success: boolean;
  dataStructure?: Record<string, unknown> & { format?: string; nodeCount?: number; edgeCount?: number; sampleNode?: unknown };
  metadata?: Record<string, unknown>;
  debug?: Record<string, unknown>;
  responseSize?: number;
  timestamp?: string;
  error?: string;
}

interface DebugInfo {
  [chartType: string]: DebugInfoEntry;
}

interface FlowNode {
  id: string;
  type: string;
  data: Record<string, unknown>;
  position: { x: number; y: number };
}

interface TechnicalChart {
  nodes: FlowNode[];
  edges: Array<Record<string, unknown>>;
}

interface SemanticNode {
  id: string;
  node_name: string;
  node_type: string;
  description?: string;
  qualified_path?: string;
  properties?: Record<string, unknown>;
}

interface SemanticChart {
  businessTerms: SemanticNode[];
  semanticTerms: SemanticNode[];
  semanticColumns: SemanticNode[];
  databaseColumns: SemanticNode[];
  edges: Array<Record<string, unknown>>;
}

type ChartData = TechnicalChart | SemanticChart | null;

// Type guards
function isTechnicalChart(d: unknown): d is TechnicalChart {
  if (typeof d !== 'object' || d === null) return false;
  const obj = d as Record<string, unknown>;
  return Array.isArray(obj.nodes);
}

function isSemanticChart(d: unknown): d is SemanticChart {
  if (typeof d !== 'object' || d === null) return false;
  const obj = d as Record<string, unknown>;
  return Array.isArray(obj.businessTerms) || Array.isArray(obj.semanticTerms);
}

// Normalize unknown payloads into a ChartData-like shape or generic record
function normalizeChartData(d: unknown): ChartData | Record<string, unknown> | null {
  if (isTechnicalChart(d) || isSemanticChart(d)) return d;
  if (typeof d === 'object' && d !== null) return d as Record<string, unknown>;
  return null;
}

function edgeHasRelationshipType(edge: unknown, rel: string): boolean {
  if (typeof edge !== 'object' || edge === null) return false;
  const e = edge as Record<string, unknown>;
  if (!e.data || typeof e.data !== 'object') return false;
  const data = e.data as Record<string, unknown>;
  return typeof data.relationship_type === 'string' && data.relationship_type === rel;
}

function getIdFromAsset(asset: unknown): string | undefined {
  if (!asset || typeof asset !== 'object') return undefined;
  const a = asset as Record<string, unknown>;
  const maybe = a['id'] ?? a['Id'] ?? a['identifier'];
  return maybe == null ? undefined : String(maybe);
}

const ChartDataDebugger: React.FC<ChartDataDebuggerProps> = ({ datasourceId, onChartDataLoaded }) => {
  const [debugInfo, setDebugInfo] = useState<DebugInfo>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>('');

  const analyzeDataStructure = (data: unknown) => {
    if (!data) return { type: 'null', structure: 'No data' };
    const obj = (typeof data === 'object' && data !== null) ? (data as Record<string, unknown>) : {};
    const structure: Record<string, unknown> & { format?: string } = { type: Array.isArray(data) ? 'array' : typeof data, keys: Object.keys(obj || {}) };

    if (isTechnicalChart(data)) {
      structure.format = 'ReactFlow';
      structure.nodeCount = data.nodes.length || 0;
      structure.edgeCount = data.edges.length || 0;
      structure.sampleNode = data.nodes[0];
      structure.sampleEdge = data.edges[0];
      return structure;
    }

    if (isSemanticChart(data)) {
      // semantic shapes may contain arrays of different node categories
      const sd = data as SemanticChart;
      structure.format = 'Semantic';
      structure.businessTerms = (sd.businessTerms || []).length || 0;
      structure.semanticTerms = (sd.semanticTerms || []).length || 0;
      structure.semanticColumns = (sd.semanticColumns || []).length || 0;
      structure.databaseColumns = (sd.databaseColumns || []).length || 0;
      structure.edges = (sd.edges || []).length || 0;
      return structure;
    }

    structure.format = 'Unknown';
    return structure;
  };

  // Safely extract expected API response fields from an unknown payload
  const extractApiResponse = (r: unknown) => {
    if (typeof r === 'object' && r !== null) {
      const obj = r as Record<string, unknown>;
      return {
        success: typeof obj.success === 'boolean' ? obj.success : true,
        data: obj.data,
        metadata: (obj.metadata && typeof obj.metadata === 'object') ? (obj.metadata as Record<string, unknown>) : undefined,
        debug: (obj.debug && typeof obj.debug === 'object') ? (obj.debug as Record<string, unknown>) : undefined,
        timestamp: typeof obj.timestamp === 'string' ? obj.timestamp : new Date().toISOString(),
        raw: obj,
      };
    }
    return { success: true, data: r, timestamp: new Date().toISOString(), raw: r };
  };

  const loadAndDebugChartData = async (chartType: string) => {
    setLoading(true);
    setError('');
    try {
      devLog(`Loading chart data: ${chartType} for datasource: ${datasourceId}`);
  const response = await fetch(`/api/chart/${datasourceId}/${chartType}?debug=true`, { credentials: 'include' });
    const text = await response.text();
      devDebug('Raw response:', text);
      if (!response.ok) throw new Error(`HTTP ${response.status}: ${text}`);
  let data: unknown;
  try { data = JSON.parse(text); } catch (parseError: unknown) { const message = parseError instanceof Error ? parseError.message : String(parseError); throw new Error(`JSON Parse Error: ${message}`); }
      devDebug('Parsed response:', data);
  const resp = extractApiResponse(data);
  setDebugInfo(prev => ({ ...prev, [chartType]: { success: resp.success, dataStructure: analyzeDataStructure(resp.data), metadata: resp.metadata, debug: resp.debug, responseSize: text.length, timestamp: resp.timestamp } }));
  if (onChartDataLoaded) onChartDataLoaded(normalizeChartData(resp.data));
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      devError(`Error loading ${chartType}:`, message);
      setError(`${chartType}: ${message}`);
      setDebugInfo(prev => ({ ...prev, [chartType]: { success: false, error: message, timestamp: new Date().toISOString() } }));
    } finally {
      setLoading(false);
    }
  };

  const loadAllChartTypes = async () => {
    const chartTypes = ['technical', 'semantic', 'erd', 'enhanced'];
    for (const ct of chartTypes) {
      // run sequentially to avoid overwhelming the server
      // short delay between requests
      // eslint-disable-next-line no-await-in-loop
      await loadAndDebugChartData(ct);
      // small pause so the UI updates between requests
      // eslint-disable-next-line no-await-in-loop
      await new Promise(r => setTimeout(r, 100));
    }
  };

  return (
    <div className="chart-debugger-root">
      <h3 className="chart-debugger-title">Chart Data Debugger</h3>

      <div className="chart-debugger-controls">
        <button onClick={() => loadAndDebugChartData('technical')} disabled={loading} className="cd-btn cd-btn-accent">Load Technical</button>
        <button onClick={() => loadAndDebugChartData('semantic')} disabled={loading} className="cd-btn cd-btn-accent">Load Semantic</button>
        <button onClick={loadAllChartTypes} disabled={loading} className="cd-btn cd-btn-success">Load All Charts</button>
        <button onClick={async () => {
          try {
            const response = await fetch(`/api/chart/${datasourceId}/health`, { credentials: 'include' });
            const health = await response.json();
            devLog('Chart health:', health);
            setDebugInfo(prev => ({ ...prev, health: health?.data }));
          } catch (err) {
            devError('Health check failed:', err);
          }
        }} className="cd-btn cd-btn-warning">Health Check</button>
      </div>

      {loading && <div className="cd-loading">Loading chart data...</div>}
      {error && <div className="cd-error">Error: {error}</div>}

      {Object.keys(debugInfo).length > 0 && (
        <div className="cd-results">
          <strong className="cd-results-title">Debug Results:</strong>
          {Object.entries(debugInfo).map(([chartType, info]) => {
            const ds = info.dataStructure as Record<string, unknown> | undefined;
            return (
            <div key={chartType} className="cd-entry">
              <strong className={`cd-entry-title ${info.success ? 'success' : 'failure'}`}>{chartType.toUpperCase()}:</strong>
              {info.success ? (
                <div className="cd-entry-body">
                  <div>Format: {info.dataStructure?.format || 'Unknown'}</div>
                  <div>Response Size: {info.responseSize} bytes</div>
                  {info.dataStructure?.format === 'ReactFlow' && (
                    <>
                      <div>Nodes: {info.dataStructure.nodeCount}</div>
                      <div>Edges: {info.dataStructure.edgeCount}</div>
                      {info.dataStructure.sampleNode && (
                        <details>
                          <summary>Sample Node</summary>
                          <pre className="cd-pre">{JSON.stringify(info.dataStructure.sampleNode, null, 2)}</pre>
                        </details>
                      )}
                    </>
                  )}
                  {info.dataStructure?.format === 'Semantic' && (
                    <>
                      <div>Business Terms: {String(ds?.businessTerms ?? 0)}</div>
                      <div>Semantic Terms: {String(ds?.semanticTerms ?? 0)}</div>
                      <div>Semantic Columns: {String(ds?.semanticColumns ?? 0)}</div>
                      <div>Database Columns: {String(ds?.databaseColumns ?? 0)}</div>
                      <div>Edges: {String(ds?.edges ?? 0)}</div>
                    </>
                  )}
                  {info.metadata && (
                    <details>
                      <summary>Metadata</summary>
                      <pre className="cd-pre">{JSON.stringify(info.metadata, null, 2)}</pre>
                    </details>
                  )}
                </div>
              ) : (
                <div className="cd-entry-error">Error: {info.error}</div>
              )}
              <div className="cd-entry-ts">{info.timestamp}</div>
            </div>
          )})}
        </div>
      )}
    </div>
  );
};

export const useChartData = (datasourceId: string, chartType: string): { data: ChartData; loading: boolean; error: string; reload: () => void } => {
  const [data, setData] = useState<ChartData>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string>('');

  useEffect(() => {
    const loadData = async () => {
      if (!datasourceId || !chartType) return;

      setLoading(true);
      setError('');

      try {
        devLog(`useChartData: Loading ${chartType} for ${datasourceId}`);
        
  const response = await fetch(`/api/chart/${datasourceId}/${chartType}`, { credentials: 'include' });
        
        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(`HTTP ${response.status}: ${errorText}`);
        }

        const result = await response.json();
  devDebug(`useChartData: Received ${chartType} data:`, result);
        
        if (!result.success) {
          throw new Error(result.error || 'Unknown API error');
        }

        setData(result.data);
        
      } catch (err) {
  const errorMessage = err instanceof Error ? err.message : String(err);
  devError(`useChartData error for ${chartType}:`, errorMessage);
  setError(errorMessage);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [datasourceId, chartType]);

  return { data, loading, error, reload: () => window.location.reload() };
};

// Enhanced parent component that uses the debugger
interface EnhancedDataCatalogProps {
  datasourceId: string;
  selectedAsset: EnhancedSelectedAsset | null;
  onAssetSelect: (asset: EnhancedSelectedAsset) => void;
  searchTerm: string;
  highlightedItem: string | null;
  showDebugger?: boolean;
}

export const EnhancedDataCatalog: React.FC<EnhancedDataCatalogProps> = ({
  datasourceId,
  selectedAsset,
  onAssetSelect,
  searchTerm,
  highlightedItem,
  showDebugger = false
}) => {
  const [activeView, setActiveView] = useState<'technical' | 'semantic' | 'lineage'>('semantic');

  // Load both chart types
  const { 
    data: technicalData, 
    loading: technicalLoading, 
    error: technicalError 
  } = useChartData(datasourceId, 'technical');
  
  const { 
    data: semanticData, 
    loading: semanticLoading, 
    error: semanticError 
  } = useChartData(datasourceId, 'semantic');

  // Get current data based on active chart type
  const currentData = activeView === 'technical' ? technicalData : semanticData;
  const currentLoading = activeView === 'technical' ? technicalLoading : semanticLoading;
  const currentError = activeView === 'technical' ? technicalError : semanticError;

  // Extract nodes from current data (handle both formats)
  const nodes = useMemo(() => {
    if (!currentData) return [];
    // Handle ReactFlow / technical format
    if (isTechnicalChart(currentData)) {
      return currentData.nodes || [];
    }

    // Handle original semantic format - convert to ReactFlow-like structure
    if (activeView === 'semantic' && isSemanticChart(currentData)) {
      const allSemanticNodes: SemanticNode[] = [
        ...(currentData.businessTerms || []),
        ...(currentData.semanticTerms || []),
        ...(currentData.semanticColumns || []),
        ...(currentData.databaseColumns || [])
      ];

      return allSemanticNodes.map(node => ({
        id: String(node.id),
        position: { x: 0, y: 0 }, // Add default position
        type: node.node_type,
        data: {
          label: node.node_name,
          nodeType: node.node_type,
          description: node.description,
          qualifiedPath: node.qualified_path,
          properties: node.properties,
          layer: node.node_type?.includes?.('business') ? 'business' : 
                 node.node_type?.includes?.('semantic') ? 'semantic' : 'database'
        }
      }));
    }

    return [];
  }, [currentData, activeView]);

  const technicalNodes = useMemo(() => {
    if (!isTechnicalChart(technicalData)) return [];
    return technicalData.nodes.filter(node => node.type === 'table');
  }, [technicalData]);

  const technicalEdges = useMemo(() => {
    if (!isTechnicalChart(technicalData)) return [];
  return technicalData.edges.filter(edge => edgeHasRelationshipType(edge, 'foreign_key'));
  }, [technicalData]);


  if (currentLoading && activeView !== 'lineage') {
    return (
      <div className="edc-loading">
        <div>Loading {activeView} chart data...</div>
      </div>
    );
  }

  if (currentError && activeView !== 'lineage') {
    return (
      <div className="edc-error-wrap">
        <div className="edc-error-text">Error loading {activeView} data: {currentError}</div>
        {showDebugger && (
          <ChartDataDebugger 
            datasourceId={datasourceId}
            onChartDataLoaded={(data) => { devLog('Debugger loaded data:', data); }}
          />
        )}
      </div>
    );
  }

  return (
    <div>
      {/* Chart type selector */}
      <div className="edc-chart-type-selector">
        <button
          onClick={() => setActiveView('technical')}
          className={"edc-ct-btn " + (activeView === 'technical' ? 'active' : '')}
        >
          Technical
        </button>
        
        <button
          onClick={() => setActiveView('semantic')}
          className={"edc-ct-btn " + (activeView === 'semantic' ? 'active' : '')}
        >
          Semantic
        </button>
        <button
          onClick={() => setActiveView('lineage')}
          disabled={!selectedAsset}
          className={"edc-ct-btn " + (activeView === 'lineage' ? 'active' : '')}
        >
          Lineage
        </button>
      </div>

      {/* Debug panel */}
      {showDebugger && (
        <ChartDataDebugger 
          datasourceId={datasourceId}
          onChartDataLoaded={(data) => { devLog('Chart data loaded:', data); }}
        />
      )}

      {activeView === 'technical' && technicalData && (
        <LineageDiagram nodes={technicalNodes} edges={technicalEdges} />
      )}
      {activeView === 'semantic' && (
        <DataCatalogTree
          nodes={nodes}
          onAssetSelect={onAssetSelect}
          searchTerm={searchTerm}
          highlightedItem={highlightedItem}
          showGoldCopyIcon={true}
        />
      )}
      {activeView === 'lineage' && (
        (() => {
          // compute subjectIds safely without non-null assertions
          const id = getIdFromAsset(selectedAsset);
          const subjectIds = id ? [id] : [];
          return <LineageDiagram subjectIds={subjectIds} />;
        })()
      )}
    </div>
  );
};

export default ChartDataDebugger;
