// src/hooks/useUnifiedSemanticBuilder.tsx
import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { useQuery } from '@apollo/client';
import { Node as FlowNode } from 'reactflow';
import { GET_ALL_BUSINESS_DATA } from '../graphql/queries/datasourceQueries.ts';
import { 
  ColumnMapping, 
  SemanticElement, 
  SemanticModel, 
  BusinessTerm, 
  CollapsedTables as _CollapsedTables, 
  ShowCode, 
  SelectedColumn,
  SemanticTerm as SemanticTermType, // Rename import to avoid conflict
  SemanticView,
  BusinessEdge
} from './../components/UnifiedSemanticBuilder/types.ts';
import { devLog } from '../utils/devLogger';
import { useAuthFetch } from '../utils/authFetch';

const resolveApiBase = () => {
  const raw = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? '';
  if (!raw) return '/api';
  const trimmed = raw.replace(/\/$/, '');
  return trimmed.endsWith('/api') ? trimmed : `${trimmed}/api`;
};

const normalizeCatalogNode = (node: any, index: number): FlowNode => {
  const rawData = node?.data && typeof node.data === 'object' ? node.data : {};
  const fallbackId = rawData?.nodeId ?? rawData?.tableName ?? rawData?.qualifiedPath ?? rawData?.qualified_path ?? `table_${index}`;
  const id = String(node?.id ?? fallbackId ?? `table_${index}`);

  const positionCandidate = node?.position;
  const position =
    positionCandidate &&
    typeof positionCandidate === 'object' &&
    typeof positionCandidate.x === 'number' &&
    typeof positionCandidate.y === 'number'
      ? positionCandidate
      : { x: 0, y: index * 80 };

  const type = typeof node?.type === 'string' ? node.type : 'table';

  const columnsSource = Array.isArray(rawData?.columns)
    ? rawData.columns
    : Array.isArray(node?.columns)
    ? node.columns
    : [];

  const columns = columnsSource.map((col: any, colIndex: number) => {
    const colId = String(col?.id ?? `${id}_col_${colIndex}`);
    const name =
      col?.name ??
      col?.column ??
      col?.node_name ??
      (typeof colId === 'string' ? colId.split('.').pop() : `column_${colIndex}`);
    const colType = col?.type ?? col?.data_type ?? col?.column_type ?? 'string';
    return {
      id: colId,
      name,
      type: colType,
      ...col,
    };
  });

  const label =
    rawData?.label ??
    node?.label ??
    (typeof fallbackId === 'string' ? fallbackId : `Table ${index + 1}`);

  const tableName =
    rawData?.tableName ??
    rawData?.qualifiedPath ??
    rawData?.qualified_path ??
    (typeof label === 'string' ? label : '');

  const qualifiedPath =
    rawData?.qualifiedPath ??
    rawData?.qualified_path ??
    (tableName && tableName.includes('.') ? tableName : tableName ? `public.${tableName}` : tableName);

  return {
    id,
    type,
    position,
    data: {
      ...rawData,
      label,
      tableName,
      qualifiedPath,
      columns,
    },
  } as FlowNode;
};

export const useUnifiedSemanticBuilder = (datasourceId: string) => {
  const apiBase = useMemo(resolveApiBase, []);
  const { authFetch } = useAuthFetch();

  const [nodes, setNodes] = useState<FlowNode[]>([]);
  const [chartLoading, setChartLoading] = useState<boolean>(true);
  const [chartError, setChartError] = useState<Error | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedColumn, setSelectedColumn] = useState<SelectedColumn | null>(null);
  const [modelName, setModelName] = useState('semantic_model');
  // showCode controls the code panel format or hidden state (null).
  // Use a raw state setter and expose a normalized setter below so callers
  // that accidentally toggle a boolean (prev => !prev) or pass a boolean
  // still behave sensibly.
  const [showCode, rawSetShowCode] = useState<ShowCode>('yaml'); // Default to yaml
  // Track last non-null format so reopening preserves YAML if user selected it.
  const lastFormatRef = useRef<'json' | 'yaml'>('yaml');
  useEffect(() => {
    if (showCode === 'json' || showCode === 'yaml') {
      lastFormatRef.current = showCode;
    }
  }, [showCode]);

  // Normalized setter: accept ShowCode | boolean | updater and map to 'json' | 'yaml' | null
  const setShowCode = useCallback((value: ShowCode | boolean | ((prev: ShowCode) => ShowCode | boolean)) => {
    try {
      if (typeof value === 'function') {
        const res = (value as (prev: ShowCode) => ShowCode | boolean)(showCode);
        if (res === 'json' || res === 'yaml' || res === null) {
          rawSetShowCode(res as ShowCode);
        } else if (typeof res === 'boolean') {
          // boolean true => show (restore previous or default to yaml), false => hide (null)
          if (res) {
            rawSetShowCode(prev => (prev === null ? lastFormatRef.current : prev) as ShowCode);
          } else {
            rawSetShowCode(null);
          }
        } else {
          rawSetShowCode('yaml');
        }
      } else {
        const res = value;
        if (res === 'json' || res === 'yaml' || res === null) {
          rawSetShowCode(res as ShowCode);
        } else if (typeof res === 'boolean') {
          if (res) {
            rawSetShowCode(prev => (prev === null ? lastFormatRef.current : prev) as ShowCode);
          } else {
            rawSetShowCode(null);
          }
        } else {
          rawSetShowCode('yaml');
        }
      }
    } catch (e) {
      // ensure we never leave showCode in an unexpected state
      rawSetShowCode('yaml');
    }
  }, [showCode]);
  
  
  // Business terms state
  const [businessTerms, setBusinessTerms] = useState<BusinessTerm[]>([]);
  const [semanticTerms, setSemanticTerms] = useState<SemanticTermType[]>([]);
  const [semanticViews, setSemanticViews] = useState<SemanticView[]>([]);
  const [columnToBusinessTerm, setColumnToBusinessTerm] = useState<Map<string, BusinessTerm>>(new Map());
  
  // Semantic model state
  const [semanticModel, setSemanticModel] = useState<SemanticModel>({
    name: 'semantic_model',
    dimensions: [],
    measures: [],
    filters: [],
    joins: []
  });

  // Debug log for semantic model changes
  useEffect(() => {
    devLog('Semantic model updated:', semanticModel);
  }, [semanticModel]);

  // Column mappings for visual indicators
  const [columnMappings, setColumnMappings] = useState<Map<string, ColumnMapping>>(new Map());

  useEffect(() => {
    let cancelled = false;

    const fetchCatalogTables = async () => {
      if (!datasourceId) {
        setNodes([]);
        setChartLoading(false);
        setChartError(null);
        return;
      }

      setChartLoading(true);
      setChartError(null);

      try {
        const url = `${apiBase}/catalog/tables?tenant_instance_id=${encodeURIComponent(datasourceId)}`;
        const resp = await authFetch<{ tables?: any[]; nodes?: any[] }>(url);
        if (!resp.ok) {
          throw new Error(resp.error || `Failed to load catalog tables (${resp.status})`);
        }

        const rawNodes = Array.isArray(resp.data?.tables)
          ? resp.data?.tables
          : Array.isArray(resp.data?.nodes)
          ? resp.data?.nodes
          : [];

        const normalized = rawNodes.map((node, index) => normalizeCatalogNode(node, index));
        if (!cancelled) {
          setNodes(normalized);
        }
      } catch (err) {
        if (!cancelled) {
          setChartError(err instanceof Error ? err : new Error('Failed to load catalog tables'));
          setNodes([]);
        }
      } finally {
        if (!cancelled) {
          setChartLoading(false);
        }
      }
    };

    fetchCatalogTables();

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [datasourceId, apiBase]);

  // Query for business terms data
  const { loading: businessLoading, error: businessError, data: businessData } = useQuery(GET_ALL_BUSINESS_DATA, {
    variables: { datasourceId },
  });

  // Process business data
  useEffect(() => {
    if (businessData) {
      const { business_terms, semantic_terms, semantic_views, business_edges } = businessData;
      setBusinessTerms(Array.isArray(business_terms) ? business_terms : []);
      setSemanticTerms(Array.isArray(semantic_terms) ? semantic_terms : []);
      setSemanticViews(Array.isArray(semantic_views) ? semantic_views : []);

      const newColumnToBusinessTerm = new Map<string, BusinessTerm>();
      if (Array.isArray(business_edges) && Array.isArray(business_terms) && Array.isArray(semantic_terms) && Array.isArray(semantic_views)) {
        const semanticTermMap = new Map<string, SemanticTermType>(semantic_terms.map((st: SemanticTermType) => [st.id, st]));
        const businessTermMap = new Map<string, BusinessTerm>(business_terms.map((bt: BusinessTerm) => [bt.id, bt]));

        business_edges.forEach((edge: BusinessEdge) => {
          if (edge.relationship_type === 'SemanticViewToColumn') {
            const semanticViewId = edge.source_node_id;

            const semanticToViewEdge = (business_edges as BusinessEdge[]).find(
              (e: BusinessEdge) => e.target_node_id === semanticViewId && e.relationship_type === 'SemanticToView'
            );

            if (semanticToViewEdge) {
              const semanticTermId = semanticToViewEdge.source_node_id;
              const semanticTerm = semanticTermMap.get(semanticTermId);

              if (semanticTerm && semanticTerm.parent_id) {
                const businessTerm = businessTermMap.get(semanticTerm.parent_id);
                if (businessTerm && edge.properties && edge.properties.schema && edge.properties.table && edge.properties.column) {
                  const columnKey = `${edge.properties.schema}.${edge.properties.table}.${edge.properties.column}`;
                  newColumnToBusinessTerm.set(columnKey, businessTerm);
                }
              }
            }
          }
        });
      }
      setColumnToBusinessTerm(newColumnToBusinessTerm);
    }
  }, [businessData]);

  // Update model name in semantic model when changed
  useEffect(() => {
    setSemanticModel(prev => ({ ...prev, name: modelName }));
  }, [modelName]);

  const getColumnKey = (nodeId: string, columnName: string) => `${nodeId}:${columnName}`;

  const getColumnMapping = (nodeId: string, columnName: string): ColumnMapping | null => {
    return columnMappings.get(getColumnKey(nodeId, columnName)) || null;
  };

  const getBusinessTermForColumn = (nodeId: string, columnName: string): BusinessTerm | undefined => {
    const node = nodes.find(n => n.id === nodeId);
    if (!node || !node.data || !node.data.label) return undefined;

    const tableName = node.data.label; // This is likely schema.table
    const key = `${tableName}.${columnName}`;
    return columnToBusinessTerm.get(key);
  };

  const getMappingColor = (mappingType: 'dimension' | 'measure' | 'filter' | null): string => {
    if (!mappingType) return 'transparent';
    const colors = {
      dimension: '#3b82f6', // blue-500
      measure: '#10b981',   // emerald-500
      filter: '#f97316',    // orange-500
    };
    return colors[mappingType] || 'transparent';
  };

  const getColumnSemanticType = (dbType: string) => {
    if (dbType.includes('int') || dbType.includes('serial')) return 'number';
    if (dbType.includes('float') || dbType.includes('decimal') || dbType.includes('numeric')) return 'number';
    if (dbType.includes('date') || dbType.includes('timestamp')) return 'time';
    if (dbType.includes('bool')) return 'boolean';
    return 'string';
  };

  const formatTitle = (text: string) => {
    return text.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
  };

  // Business term fetching remains powered by the Hasura GraphQL endpoint for now.

  // Helper functions
  const isNumericType = (type: string) => {
    return type.includes('int') || type.includes('float') || type.includes('decimal') || type.includes('numeric');
  };

  const showNotification = (_message: string, _type: 'success' | 'error' = 'success') => {
    // This function is now correctly handled by the toast in the component.
  };

  // Add semantic element functions
  const addDimension = (tableName: string, column: any) => {
    const businessTerm = getBusinessTermForColumn(selectedColumn!.nodeId, column.name);
    const semanticTerm = businessTerm ? semanticTerms.find(st => st.parent_id === businessTerm.id) : undefined;
    const semanticView = semanticTerm ? semanticViews.find(sv => sv.parent_id === semanticTerm.id) : undefined;

    const newDimension: SemanticElement = {
      id: `dim_${tableName}_${column.name}_${Date.now()}`,
      name: semanticTerm?.node_name || businessTerm?.node_name || `${tableName}_${column.name}`,
      type: getColumnSemanticType(column.type),
      sql: `{CUBE}.${column.name}`,
      title: formatTitle(semanticView?.node_name || semanticTerm?.node_name || businessTerm?.node_name || `${tableName} ${column.name}`),
      description: semanticView?.description || semanticTerm?.description || businessTerm?.description || column.description || `Dimension for ${column.name}`,
      sourceTable: tableName,
      sourceColumn: column.name,
    };

    setSemanticModel(prev => ({
      ...prev,
      dimensions: [...(prev.dimensions || []), newDimension]
    }));

    // Update column mapping
    const key = getColumnKey(selectedColumn!.nodeId, column.name);
    setColumnMappings(prev => new Map(prev.set(key, {
      nodeId: selectedColumn!.nodeId,
      tableName,
      columnName: column.name,
      columnType: column.type,
      mappingType: 'dimension',
      mappingId: newDimension.id
    })));

    showNotification(`Added dimension: ${newDimension.title}`);
  };

  const addMeasure = (tableName: string, column: any) => {
    if (!isNumericType(column.type)) {
      showNotification('Measures can only be created from numeric columns', 'error');
      return;
    }

    const businessTerm = getBusinessTermForColumn(selectedColumn!.nodeId, column.name);
    const semanticTerm = businessTerm ? semanticTerms.find(st => st.parent_id === businessTerm.id) : undefined;
    const semanticView = semanticTerm ? semanticViews.find(sv => sv.parent_id === semanticTerm.id) : undefined;

    const newMeasure: SemanticElement = {
      id: `measure_${tableName}_${column.name}_${Date.now()}`,
      name: `${semanticTerm?.node_name || businessTerm?.node_name || `${tableName}_${column.name}`}_sum`,
      type: 'sum',
      sql: `{CUBE}.${column.name}`,
      title: formatTitle(`Total ${semanticView?.node_name || semanticTerm?.node_name || businessTerm?.node_name || `${tableName} ${column.name}`}`),
      description: semanticView?.description || semanticTerm?.description || businessTerm?.description || column.description || `Sum of ${column.name}`,
      sourceTable: tableName,
      sourceColumn: column.name,
    };

    setSemanticModel(prev => ({
      ...prev,
      measures: [...(prev.measures || []), newMeasure]
    }));

    // Update column mapping
    const key = getColumnKey(selectedColumn!.nodeId, column.name);
    setColumnMappings(prev => new Map(prev.set(key, {
      nodeId: selectedColumn!.nodeId,
      tableName,
      columnName: column.name,
      columnType: column.type,
      mappingType: 'measure',
      mappingId: newMeasure.id
    })));

    showNotification(`Added measure: ${newMeasure.title}`);
  };

  const addFilter = (tableName: string, column: any) => {
    const businessTerm = getBusinessTermForColumn(selectedColumn!.nodeId, column.name);
    const semanticTerm = businessTerm ? semanticTerms.find(st => st.parent_id === businessTerm.id) : undefined;
    const semanticView = semanticTerm ? semanticViews.find(sv => sv.parent_id === semanticTerm.id) : undefined;

    const newFilter: SemanticElement = {
      id: `filter_${tableName}_${column.name}_${Date.now()}`,
      name: `${semanticTerm?.node_name || businessTerm?.node_name || `${tableName}_${column.name}`}_filter`,
      type: 'string',
      sql: `{CUBE}.${column.name}`,
      title: formatTitle(`${semanticView?.node_name || semanticTerm?.node_name || businessTerm?.node_name || `${tableName} ${column.name}`} Filter`),
      description: semanticView?.description || semanticTerm?.description || businessTerm?.description || column.description || `Filter for ${column.name}`,
      sourceTable: tableName,
      sourceColumn: column.name,
    };

    setSemanticModel(prev => ({
      ...prev,
      filters: [...(prev.filters || []), newFilter]
    }));

    // Update column mapping
    const key = getColumnKey(selectedColumn!.nodeId, column.name);
    setColumnMappings(prev => new Map(prev.set(key, {
      nodeId: selectedColumn!.nodeId,
      tableName,
      columnName: column.name,
      columnType: column.type,
      mappingType: 'filter',
      mappingId: newFilter.id
    })));

    showNotification(`Added filter: ${newFilter.title}`);
  };

  const removeSemanticElement = (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string) => {
    setSemanticModel(prev => ({
      ...prev,
      [type]: (prev[type] || []).filter(item => item.id !== id)
    }));

    // Remove column mapping (only for dimensions/measures/filters, not joins)
    if (type !== 'joins') {
      const mappingToRemove = Array.from(columnMappings.entries())
        .find(([_, mapping]) => mapping.mappingId === id);
      
      if (mappingToRemove) {
        setColumnMappings(prev => {
          const newMap = new Map(prev);
          newMap.delete(mappingToRemove[0]);
          return newMap;
        });
      }
    }

    showNotification('Removed semantic element');
  };

  const toggleElementEdit = (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string) => {
    setSemanticModel(prev => ({
      ...prev,
      [type]: (prev[type] || []).map(item => 
        item.id === id ? { ...item, isEditing: !item.isEditing } : item
      )
    }));
  };

  const updateSemanticElement = (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string, updates: any) => {
    setSemanticModel(prev => ({
      ...prev,
      [type]: (prev[type] || []).map(item => 
        item.id === id ? { ...item, ...updates } : item // Removed isEditing: false to keep edit open
      )
    }));
    // Do not close edit mode automatically
  };

  // Generate JSON/YAML
  const generateJSON = useCallback(() => {
    return JSON.stringify(semanticModel, null, 2);
  }, [semanticModel]);

  const generateYAML = useCallback(() => {
    const yamlContent = `cube: ${semanticModel.name}
sql: |
  SELECT * FROM public.${semanticModel.name.toLowerCase()}

dimensions:
${semanticModel.dimensions.map(dim => `  ${dim.name}:
    sql: ${dim.sql}
    type: ${dim.type}
    title: "${dim.title}"
    description: "${dim.description}"${dim.format ? `
    format: "${dim.format}"` : ''}
`).join('\n')}

measures:
${semanticModel.measures.map(meas => `  ${meas.name}:
    sql: ${meas.sql}
    type: ${meas.type}
    title: "${meas.title}"
    description: "${meas.description}"
`).join('\n')}

segments:
${semanticModel.filters.map(fil => `  ${fil.name}:
    sql: ${fil.sql}
`).join('\n')}
`;
    return yamlContent;
  }, [semanticModel]);

  // Filter nodes based on search
  const filteredNodes = useMemo(() => {
    if (!searchTerm.trim()) return nodes;
    return nodes.filter(node =>
      node.data?.label?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      node.data?.columns?.some((col: any) =>
        col.name.toLowerCase().includes(searchTerm.toLowerCase())
      )
    );
  }, [nodes, searchTerm]);

  return {
    nodes,
    searchTerm,
    setSearchTerm,
    selectedColumn,
    setSelectedColumn,
    modelName,
    setModelName,
    showCode,
    setShowCode,
    // collapsedTables,
    // toggleTableCollapse,
    businessTerms,
    semanticTerms,
    semanticViews,
    semanticModel,
    setSemanticModel,
    columnMappings,
    setColumnMappings,
    chartLoading,
    chartError,
    businessLoading,
    businessError,
    isNumericType,
    getColumnMapping,
    getMappingColor,
    getBusinessTermForColumn,
    addDimension,
    addMeasure,
    addFilter,
    removeSemanticElement,
    toggleElementEdit,
    updateSemanticElement,
    generateJSON,
    generateYAML,
    filteredNodes,
    showNotification
  };
};