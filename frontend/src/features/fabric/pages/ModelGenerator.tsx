import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useAuthFetch } from '../../../utils/authFetch';
import { devLog, devDebug, devError } from '../../../utils/devLogger';
import { Box, Typography, Alert, Button, Paper, Divider, CircularProgress, TextField, InputAdornment, Tooltip } from '@mui/material';
import DataCatalogTree from '../../../pages/TabbedModal/tabs/DataCatalogTree';
import { Node as FlowNode, Edge } from 'reactflow';
import yaml from 'js-yaml';
import Fuse from 'fuse.js';
import MonacoCodeEditor from '../../../components/UnifiedSemanticBuilder/MonacoCodeEditor.lazy';
import '../../../components/UnifiedSemanticBuilder/CodePanel.css';
import { useQuery } from '@apollo/client';
import { GET_TECHNICAL_LINEAGE_CHART, transformChartData } from '../../../graphql/queries/semantic';
import { GET_ALL_BUSINESS_DATA } from '../../../graphql/queries/datasourceQueries';
// import GET_ALL_BUSINESS_DATA from '../../../pages/Fabric/UnifiedSemanticBuilder';
import getErrorMessage from '../../../utils/errors';
import * as TablerIcons from '@tabler/icons-react';
import { toast } from '../../../components/ui/sonner';
import * as Icons from '../../../components/UnifiedSemanticBuilder/icons';

import { useTenant } from '../../../contexts/TenantContext';
import { useConfirm } from '../../../components/ConfirmProvider';
import { useNotification } from '../../../hooks/useNotification';
import { useModelCatalog } from '../../../hooks/useModelCatalog';
import { BusinessTerm, BusinessEdge, SemanticTerm } from '../../../components/UnifiedSemanticBuilder/types';
import { Search as SearchIcon } from '@mui/icons-material';
import ProfessionalSearchInput, { SearchSuggestion } from '../../../components/common/ProfessionalSearchInput';

const ModelGenerator: React.FC = () => {
  const { datasource, tenant } = useTenant();
  const confirm = useConfirm();
  const notification = useNotification();
  // hook to interact with the model catalog when toggling core/custom active
  const tenantId = (tenant as any)?.id || '';
  const datasourceId = (datasource as any)?.id || '';
  const shouldInitCatalog = Boolean(tenantId && datasourceId);
  const { models: catalogModels, selectedModel: catalogSelectedModel, setSelectedModel: setCatalogSelectedModel, createCustomModel } = useModelCatalog(shouldInitCatalog ? tenantId : 'skip', shouldInitCatalog ? datasourceId : 'skip');
  const [nodes, setNodes] = useState<FlowNode[]>([]);
  const [selection, setSelection] = useState<Set<string>>(new Set());
  const [generatedJson, setGeneratedJson] = useState<string>('');
  const [generatedYaml, setGeneratedYaml] = useState<string>('');
  const [searchTokens, setSearchTokens] = useState<Array<{ label: string; section: string; key: string }>>([]);
  const quickFindRef = useRef<HTMLInputElement | null>(null);
  const [_quickFindRef] = [quickFindRef];
  // Monaco editor API for reveal/focus controls
  const monacoApiRef = useRef<any>(null);
  const [fuseObj, setFuseObj] = useState<Fuse<any> | null>(null);
  const [_fuseObj, _setFuseObj] = [fuseObj, setFuseObj];
  const [activeGeneratorTab, setActiveGeneratorTab] = useState<'json' | 'yaml'>('json');
  const [modelMetadata, setModelMetadata] = useState<Record<string, any>>({});
  const [generationLoading, setGenerationLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [editorSearch, setEditorSearch] = useState('');
  // core/custom toggles removed — not needed in this view
  const [sectionCounts, setSectionCounts] = useState<{ [k: string]: number }>({});
  const [generationError, setGenerationError] = useState<string | null>(null);
  const [columnToBusinessTerm, setColumnToBusinessTerm] = useState<Map<string, BusinessTerm>>(new Map());
  const [matchIndex, setMatchIndex] = useState(0);
  const [matchCount, setMatchCount] = useState(0);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [_semanticTermMap, setSemanticTermMap] = useState<Map<string, SemanticTerm>>(new Map());
  const [showSemanticSuggestions, setShowSemanticSuggestions] = useState<boolean>(() => {
    try {
      const v = localStorage.getItem('mg_show_semantic');
      return v === 'true';
    } catch (e) {
      return false;
    }
  });
  const [semanticSuggestions, setSemanticSuggestions] = useState<SearchSuggestion[]>([]);
  const [isSearchingSemanticTerms, setIsSearchingSemanticTerms] = useState(false);

  const singularForSection: Record<string, string> = {
    dimensions: 'dimension',
    measures: 'measure',
    joins: 'join',
    filters: 'filter',
  };

  const handleJumpToSection = (section: string) => {
    // Match fabric builder: dispatch just the section unless we have a dotted key (e.g. schema.table.column)
    const token = searchTokens.find((t) => t.section === section && t.key && typeof t.key === 'string');
    const detail: any = { section };
    const maybeKey = token ? token.key : (catalogSelectedModel?.model_key || '');
    // Only include keys that are likely matchable in the JSON/YAML content:
    // - dotted paths like schema.table.column
    // - or simple token names (alphanumeric, underscore, dot)
    if (maybeKey && typeof maybeKey === 'string') {
      const isSimpleToken = /^[a-zA-Z0-9_.]+$/.test(maybeKey);
      if (maybeKey.includes('.') || isSimpleToken) {
        detail.key = maybeKey;
      }
    }
    window.dispatchEvent(new CustomEvent('semlayer.jumpToSection', { detail }));
  };

  // Helper: update match count and reset index (removed for Prism)
  // const updateMatchCount = (term: string, candidates: Array<any>) => { ... };

  // const navigateMatch = (dir: number) => { ... };

  // const copyCurrentMatch = async () => { ... };

  // Fetch chart data using GraphQL, similar to TabbedModal
  const { loading, error: chartError, data: chartData } = useQuery(GET_TECHNICAL_LINEAGE_CHART, {
    variables: { datasourceId: datasource?.id },
    skip: !datasource,
    fetchPolicy: 'network-only', // Ensure we get fresh data
  });

  // Fetch business data
  const { data: businessData, error: businessError } = useQuery(GET_ALL_BUSINESS_DATA, {
    variables: { datasourceId: datasource?.id },
    skip: !datasource,
  });

  // Log Apollo errors via derived state, per guidance
  useEffect(() => {
    if (chartError) devError('Error fetching technical lineage chart:', chartError);
  }, [chartError]);
  useEffect(() => {
    if (businessError) devError('Error fetching business data:', businessError);
  }, [businessError]);

  // Process the fetched chart data
  useEffect(() => {
    // build search tokens whenever generated content changes
    const tokens: Array<{ label: string; section: string; key: string }> = [];
    try {
      if (generatedJson) {
        const parsed = JSON.parse(generatedJson);
        const arr = Array.isArray(parsed) ? parsed : [parsed];
        arr.forEach((m: any) => {
          // Handle backend ResolvedModelConfig structure with cubes
          if (m.cubes && Array.isArray(m.cubes)) {
            m.cubes.forEach((cube: any) => {
              if (cube.dimensions) {
                Object.keys(cube.dimensions).forEach((key) => {
                  const dim = cube.dimensions[key];
                  tokens.push({ label: `dim: ${dim.name || key}`, section: 'dimensions', key: dim.name || key });
                });
              }
              if (cube.measures) {
                Object.keys(cube.measures).forEach((key) => {
                  const meas = cube.measures[key];
                  tokens.push({ label: `meas: ${meas.name || key}`, section: 'measures', key: meas.name || key });
                });
              }
              if (cube.segments) {
                Object.keys(cube.segments).forEach((key) => {
                  const filt = cube.segments[key];
                  tokens.push({ label: `filt: ${filt.name || key}`, section: 'filters', key: filt.name || key });
                });
              }
            });
          } else {
            // Fallback to frontend SemanticModel structure
            (m.dimensions || []).forEach((d: any) => tokens.push({ label: `dim: ${d.name || d.id}`, section: 'dimensions', key: d.name || d.id }));
            (m.measures || []).forEach((d: any) => tokens.push({ label: `meas: ${d.name || d.id}`, section: 'measures', key: d.name || d.id }));
            (m.filters || []).forEach((d: any) => tokens.push({ label: `filt: ${d.name || d.id}`, section: 'filters', key: d.name || d.id }));
          }
        });
      } else if (generatedYaml) {
        const parsed = yaml.load(generatedYaml) as any;
        const arr = Array.isArray(parsed) ? parsed : [parsed];
        arr.forEach((m: any) => {
          // Handle backend ResolvedModelConfig structure with cubes
          if (m.cubes && Array.isArray(m.cubes)) {
            m.cubes.forEach((cube: any) => {
              if (cube.dimensions) {
                Object.keys(cube.dimensions).forEach((key) => {
                  const dim = cube.dimensions[key];
                  tokens.push({ label: `dim: ${dim.name || key}`, section: 'dimensions', key: dim.name || key });
                });
              }
              if (cube.measures) {
                Object.keys(cube.measures).forEach((key) => {
                  const meas = cube.measures[key];
                  tokens.push({ label: `meas: ${meas.name || key}`, section: 'measures', key: meas.name || key });
                });
              }
              if (cube.segments) {
                Object.keys(cube.segments).forEach((key) => {
                  const filt = cube.segments[key];
                  tokens.push({ label: `filt: ${filt.name || key}`, section: 'filters', key: filt.name || key });
                });
              }
            });
          } else {
            // Fallback to frontend SemanticModel structure
            (m.dimensions || []).forEach((d: any) => tokens.push({ label: `dim: ${d.name || d.id}`, section: 'dimensions', key: d.name || d.id }));
            (m.measures || []).forEach((d: any) => tokens.push({ label: `meas: ${d.name || d.id}`, section: 'measures', key: d.name || d.id }));
            (m.filters || []).forEach((d: any) => tokens.push({ label: `filt: ${d.name || d.id}`, section: 'filters', key: d.name || d.id }));
          }
        });
      }
    } catch (e) {
      // ignore parse errors
    }
    setSearchTokens(tokens);
    // Build a Fuse index for the quick find
    try {
      const fuse = new Fuse(tokens, { keys: ['label', 'key'], threshold: 0.4, ignoreLocation: true });
      setFuseObj(fuse);
    } catch (e) {
      setFuseObj(null);
    }
    // compute simple counts for sections
    const counts: { [k: string]: number } = { dimensions: 0, measures: 0, joins: 0, filters: 0 };
    try {
      const src = generatedJson && generatedJson.trim() ? generatedJson : generatedYaml;
      if (src && src.trim()) {
        let parsed: any = null;
        if (generatedJson && generatedJson.trim()) parsed = JSON.parse(generatedJson);
        else parsed = yaml.load(generatedYaml);
        // If parsed is an array, only take the first element for counting.
        const model = Array.isArray(parsed) ? parsed[0] : parsed;
        if (model && typeof model === 'object') {
          // Handle backend ResolvedModelConfig structure with cubes
          if (model.cubes && Array.isArray(model.cubes) && model.cubes.length > 0) {
            const cube = model.cubes[0];
            counts.dimensions = cube.dimensions ? Object.keys(cube.dimensions).length : 0;
            counts.measures = cube.measures ? Object.keys(cube.measures).length : 0;
            counts.joins = cube.joins ? Object.keys(cube.joins).length : 0;
            counts.filters = cube.segments ? Object.keys(cube.segments).length : 0;
          } else {
            // Fallback to frontend SemanticModel structure
            counts.dimensions = Array.isArray(model.dimensions) ? model.dimensions.length : 0;
            counts.measures = Array.isArray(model.measures) ? model.measures.length : 0;
            counts.joins = Array.isArray(model.joins) ? model.joins.length : 0;
            counts.filters = Array.isArray(model.filters) ? model.filters.length : 0;
          }
        }
      }
    } catch (e) {
      // ignore
    }
    setSectionCounts(counts);
  }, [generatedJson, generatedYaml]);

  // Removed updateDecorations for Monaco

  // Removed jump event handler for Monaco

  // Removed decorations useEffect for Monaco

  // Removed keyboard shortcuts for Monaco

  // Process the fetched chart data
  useEffect(() => {
    const fetchCatalogTables = async () => {
      if (!datasource) return null;
      try {
        // In local dev sometimes the Vite proxy can hang. When running in dev mode,
        // call the backend directly to avoid proxy issues. In production this will
        // continue to use the relative `/api` path.
        const base = import.meta.env.DEV ? 'http://localhost:9090' : '';
        const resp = await authFetch<{tables:any[]}>(`${base}/api/catalog/tables?tenant_instance_id=${datasource.id}`, { method: 'GET' });
        if (resp.ok && resp.data && Array.isArray(resp.data.tables)) {
          return resp.data.tables as any[];
        }
      } catch (e) {
        devLog('No catalog tables endpoint available or failed to fetch', e);
      }
      return null;
    };

    (async () => {
      const catalogNodes = await fetchCatalogTables();
      if (catalogNodes && catalogNodes.length > 0) {
        devLog(`✅ ModelGenerator: Loaded ${catalogNodes.length} tables from catalog endpoint`);
        setNodes(catalogNodes as any[]);
        return;
      }

      if (chartData && chartData.tenant_chart && chartData.tenant_chart.length > 0) {
      try {
        devLog('📦 ModelGenerator: Received compressed chart data via GraphQL');
        const compressedChartAsHex = chartData.tenant_chart[0].chart;
        const parsedChartData = transformChartData(compressedChartAsHex) as { nodes: FlowNode[], edges: Edge[] };
        
        if (parsedChartData && parsedChartData.nodes) {
          devLog(`✅ ModelGenerator: Successfully parsed chart. Found ${parsedChartData.nodes.length} nodes.`);
          // Filter for only table nodes to display in the tree
          const tableNodes = parsedChartData.nodes
            .filter(node => node.data?.nodeType === 'table' || node.type === 'table');
          setNodes(tableNodes);
        } else {
          devLog('⚠️ ModelGenerator: Parsed chart data is empty or invalid.');
          setNodes([]);
        }
      } catch (e) {
        devError('💥 ModelGenerator: Failed to process chart data:', getErrorMessage(e));
        setNodes([]);
      }
    } else if (chartData) {
      devLog('⚠️ ModelGenerator: GraphQL query returned no chart data.');
      setNodes([]);
    }
    })();
  }, [chartData]);

  useEffect(() => {
    const fetchMetadata = async () => {
      if (nodes.length > 0 && datasource) {
        const tableNames = nodes
          .map((n) => n.data.tableName)
          .filter((name): name is string => !!name);
        if (tableNames.length > 0) {
          try {
            const metadata = await checkModelMetadata(tableNames);
            setModelMetadata(metadata);
          } catch (e: unknown) {
            devError("Failed to fetch model metadata", getErrorMessage(e));
            setGenerationError("Could not load existing model status.");
          }
        }
      } else {
        setModelMetadata({}); // Clear metadata if nodes/datasource are gone
      }
    };

    fetchMetadata();
  }, [nodes, datasource]);

  // Process business data to map columns to business terms
  useEffect(() => {
    if (businessData) {
      const { business_terms, semantic_terms, semantic_views, business_edges } = businessData;
      const newColumnToBusinessTerm = new Map<string, BusinessTerm>();

      if (business_edges && business_terms && semantic_terms && semantic_views) {
        const semanticTermMap = new Map<string, SemanticTerm>(semantic_terms.map((st: SemanticTerm) => [st.id, st]));
        const businessTermMap = new Map<string, BusinessTerm>(business_terms.map((bt: BusinessTerm) => [bt.id, bt]));

        setSemanticTermMap(semanticTermMap);

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
      devLog(`Processed ${newColumnToBusinessTerm.size} column-to-business-term mappings.`);
    }
  }, [businessData]);

  const { authFetch } = useAuthFetch();
  const checkModelMetadata = useCallback(async (tableNames: string[]) => {
    if (!datasource || tableNames.length === 0) return {};
    try {
      const resp = await authFetch<{results:any}>(
        '/api/fabric/models/metadata',
        { method: 'POST', json: { tenant_instance_id: datasource.id, table_names: tableNames } }
      );
      if (!resp.ok) {
        devError(`Failed to fetch metadata: ${resp.status}`, resp.error);
        return {}; // Return empty object instead of throwing
      }
      return resp.data?.results || {};
    } catch (error) {
      devError("Error in checkModelMetadata:", error);
      return {}; // Return empty object on any error
    }
  }, [authFetch, datasource]);

  const searchSemanticTerms = useCallback(async (query: string): Promise<SearchSuggestion[]> => {
    // Allow empty query when we can provide scope tables to bias results.
    if (!datasource) return [];
    
    try {
      setIsSearchingSemanticTerms(true);
      // Build scope tables from selection (table-<nodeId> entries)
      const scopeTables: string[] = Array.from(selection)
        .filter(id => id.startsWith('table-'))
        .map(id => nodes.find(n => n.id === id.replace('table-', ''))?.data?.tableName)
        .filter((name): name is string => !!name);

      // Optionally include a selected column context if exactly one column is selected
      // Build column context from selection: prefer column-<schema.table.column> ids
      let columnContext: any = undefined;
      const columnSelections = Array.from(selection).filter(id => id.startsWith('column-'));
      if (columnSelections.length === 1) {
        const colKey = columnSelections[0].replace('column-', '');
        // try to parse schema.table.column or schema.table.col
        const parts = colKey.split('.');
        if (parts.length >= 3) {
          columnContext = { schema: parts[0], table: parts[1], column: parts.slice(2).join('.') };
          // If we have metadata with data type for the table, include it
          const tableName = `${parts[0]}.${parts[1]}`;
          const meta = modelMetadata[tableName];
          if (meta && meta.columns) {
            const colMeta = meta.columns.find((c: any) => c.column_name === parts.slice(2).join('.'));
            if (colMeta && colMeta.data_type) columnContext.data_type = colMeta.data_type;
          }
        }
      }

      const payload = { query, limit: 20, scope_tables: scopeTables, column: columnContext };
      // Debug: log outgoing semantic search payload (visible in browser console)
      try { devDebug('searchSemanticTerms request payload:', payload); } catch {}
      const response = await authFetch<{ results: any[] }>('/api/semantic-terms/search', {
        method: 'POST',
        json: payload
      });
      // Debug: log raw response wrapper from authFetch
      try { devDebug('searchSemanticTerms raw response:', response); } catch {}
      
      if (!response.ok) {
        devError('Failed to search semantic terms:', response.error);
        try { toast.error?.(`Semantic search failed: ${response.error || response.status}`); } catch {}
        return [];
      }

      const results = response.data?.results || [];
      if (!results || results.length === 0) {
        try { toast(`No semantic suggestions found`); } catch {}
      }
      try { devDebug('searchSemanticTerms results length:', (results && results.length) || 0, 'sample:', results.slice(0,5)); } catch {}
      return results.map((term: any) => ({
        id: term.node_id || term.id,
        title: term.term_name || term.node_name || term.id,
        subtitle: term.data_type ? `${term.data_type} • ${Math.round((term.score || 0) * 100)}%` : `${Math.round((term.score || 0) * 100)}%`,
        type: 'semantic_term',
        description: 'Semantic Term',
        // attach raw result for later actions
        result: term
      }));
    } catch (error) {
      devError('Error searching semantic terms:', error);
      return [];
    } finally {
      setIsSearchingSemanticTerms(false);
    }
  }, [authFetch, datasource]);

  // When semantic suggestion mode is opened, fetch a short list of defaults
  // so the user sees suggestions immediately (even with an empty query).
  useEffect(() => {
    let mounted = true;
    (async () => {
      if (!showSemanticSuggestions) return;
      try {
        // If user hasn't typed anything we request a short list of popular terms
        const results = await searchSemanticTerms(editorSearch && editorSearch.length >= 2 ? editorSearch : '');
        if (mounted) setSemanticSuggestions(results);
      } catch (e) {
        // ignore
      }
    })();
    return () => { mounted = false; };
  }, [showSemanticSuggestions, editorSearch, searchSemanticTerms]);

  const createCustomModelIfNotExists = useCallback(async (modelKey: string) => {
    if (!datasource) return;
    const customKey = '/u_' + modelKey.slice(1); // e.g., /public/users -> /u_public/users
    try {
      // Check if custom already exists
      const checkResp = await authFetch(`/api/fabric/models/definition?tenant_instance_id=${datasource.id}&model_key=${encodeURIComponent(customKey)}`);
      if (checkResp.ok) {
        devLog(`Custom model ${customKey} already exists, skipping creation.`);
        return;
      }
      // Create the custom model
      const createResp = await authFetch('/api/fabric/models', {
        method: 'POST',
        json: {
          tenant_instance_id: datasource.id,
          model_key: customKey,
          config: { extends: modelKey }
        }
      });
      if (createResp.ok) {
        devLog(`Successfully created custom model ${customKey} extending ${modelKey}.`);
      } else {
        devError(`Failed to create custom model ${customKey}:`, createResp.error);
      }
    } catch (error) {
      devError(`Error creating custom model for ${modelKey}:`, error);
    }
  }, [authFetch, datasource]);

  const handleGenerateForSelection = async () => {
    if (selection.size === 0 || !datasource) {
      return;
    }

    try {
      setGenerationLoading(true);
      setGenerationError(null);
      setGeneratedJson('');
      setGeneratedYaml('');

      const selectedTables = Array.from(selection)
        .filter((id) => id.startsWith('table-'))
        .map((id) => nodes.find((n) => n.id === id.replace('table-', ''))?.data.tableName)
        .filter((name): name is string => !!name);

      if (selectedTables.length === 0) {
        setGenerationError('Please select at least one table to generate a model.');
        setGenerationLoading(false);
        return;
      }

      // Use the pre-fetched metadata from state, filtered for the current selection
      const metadataMap = Object.fromEntries(
        Object.entries(modelMetadata).filter(([key]) => selectedTables.includes(key))
      );
      const existing = Object.values(metadataMap).filter((m: any) => m.exists);

      let overwrite = false;
      if (existing.length > 0) {
        const warningMsg = existing.map((m: any) =>
          `${m.table_name} (created ${new Date(m.created_at).toLocaleDateString()} by ${m.created_by || 'unknown'})`
        ).join('\n');

        overwrite = await confirm({ title: 'Existing models detected', description: `The following models already exist:\n${warningMsg}\n\nOverwrite them?` });
      }

      // The backend will handle skipping or overwriting based on the flag.
      // We always send the full list of selected tables.

      // Call generate with the overwrite flag
      const genResp = await authFetch<{generated:any[]; skipped:string[]; overwritten:string[]}>(
        '/api/fabric/models/generate',
        { method: 'POST', json: { tenant_instance_id: datasource.id, scope: { type: 'tables', names: selectedTables }, overwrite, accept_relationships: true } }
      );
      if (!genResp.ok) {
        devError(`Failed to generate models: ${genResp.status}`, genResp.error);
        setGenerationError(`Failed to generate models: ${genResp.error || `HTTP ${genResp.status}`}`);
        return;
      }
      const { generated = [], skipped = [], overwritten = [] } = genResp.data || ({} as any);

  if (skipped.length > 0) toast(`Skipped existing models for: ${skipped.join(', ')}`);
  if (overwritten.length > 0) toast.success(`Overwritten models for: ${overwritten.join(', ')}`);

      if (generated.length > 0) {
        const allResolvedConfigs = generated.map((m: any) => {
          const config = m.resolved_config || m;
          // Enrich with business term info
          if (config.cubes && Array.isArray(config.cubes)) {
            config.cubes.forEach((cube: any) => {
              const tableName = cube.sql_table; // e.g., "public.users"
              if (!tableName) return;

              const enrich = (elements: Record<string, any> | undefined) => {
                if (!elements) return;
                Object.values(elements).forEach((el: any) => {
                  const columnKey = `${tableName}.${el.source_column}`;
                  const businessTerm = columnToBusinessTerm.get(columnKey);
                  if (businessTerm) {
                    el.business_term = businessTerm.node_name;
                    el.business_term_id = businessTerm.id;
                    el.business_term_classification = businessTerm.properties?.classification || 'Unclassified';
                  }
                });
              };

              enrich(cube.dimensions);
              enrich(cube.measures);
            });
          }
          return config;
        });

        // Automatically create custom models for each generated core model
        for (const gen of generated) {
          const modelKey = gen.model_key || '/' + gen.table_name.replace('.', '/');
          await createCustomModelIfNotExists(modelKey);
        }
  // If a custom catalog model is selected, ensure there is a custom model and generate an 'extends' block
  if (catalogSelectedModel && catalogSelectedModel.is_custom === false) {
          const baseKey = (catalogSelectedModel as any).model_key || '';
          // Try to find an existing custom model for this base
          let existingCustom = catalogModels.find((m: any) => m.is_custom && ((m as any).parent_model_key === baseKey || (m as any).parent_model_key === (baseKey + '_custom')));
          if (!existingCustom) {
            // Create a custom model that inherits the core model
            try {
              const newCustom = await createCustomModel(baseKey);
              if (newCustom) {
                existingCustom = newCustom as any;
                // set the selected model in local state
                setCatalogSelectedModel(existingCustom as any);
              }
            } catch (e: any) {
              devError('Failed to create custom model during generation:', getErrorMessage(e));
              setGenerationError(getErrorMessage(e, 'Failed to create custom model'));
              setGenerationLoading(false);
              return;
            }
          } else {
            // If a custom exists, select it so UI reflects it
            setCatalogSelectedModel(existingCustom as any);
          // usedCatalogModel not needed here; selection is handled via setCatalogSelectedModel
          }

          const customConfig = {
            // Keep extends pointing to the core model so the custom inherits it
            extends: baseKey,
            // This example assumes the first generated model is the one being customized.
            dimensions: allResolvedConfigs[0]?.dimensions || [],
            measures: allResolvedConfigs[0]?.measures || [],
          };
          setGeneratedJson(JSON.stringify(customConfig, null, 2));
          setGeneratedYaml(yaml.dump(customConfig));
        } else {
          setGeneratedJson(JSON.stringify(allResolvedConfigs, null, 2));
          setGeneratedYaml(yaml.dump(allResolvedConfigs));
        }
      } else if (skipped.length === 0 && overwritten.length === 0) {
        setGenerationError("No new models were generated.");
      }

      // Update metadata for newly created models so the UI reflects their existence.
      // This ensures the model icon appears immediately without a page refresh.
      setModelMetadata(prevMetadata => {
        const newMetadata = { ...prevMetadata };
        generated.forEach((model: any) => {
          // model.table_name is the qualified name (e.g., "public.users") from our backend change
          if (model.table_name) {
            const dotForm = model.table_name; // e.g. public.users
            const slashForm = model.table_name.replace('.', '/'); // e.g. public/users
            const leadingSlash = '/' + slashForm; // e.g. /public/users

            const entries = [dotForm, slashForm, leadingSlash];
            entries.forEach((key) => {
              newMetadata[key] = { ...newMetadata[key], exists: true, title: model.model_name };
            });
          }
        });
        return newMetadata;
      });
    } catch (err: any) {
  devError('💥 Failed to generate model:', err);
      setGenerationError(err.message);
    } finally {
      setGenerationLoading(false);
    }
  };

  // Generate model for a single table (used by DataCatalogTree row action)
  const handleGenerateForTable = async (tableName: string) => {
    if (!datasource) return;
    // Temporarily select this table and call the shared generation flow
    try {
      setSelection(new Set([`table-${tableName}`]));
      // Reuse existing handler but adapt nodes mapping: our handleGenerateForSelection expects selection IDs mapped to node ids
      // Build the node id from nodes array (find node with matching tableName)
      const node = nodes.find(n => n.data?.tableName === tableName || n.data?.qualifiedPath === tableName || n.data?.label === tableName);
      if (!node) {
        setGenerationError('Could not locate table node for generation.');
        return;
      }
      setSelection(new Set([`table-${node.id}`]));
      // Now call the same flow as the bulk generator but only for this table
      setGenerationLoading(true);
      setGenerationError(null);

      const selectedTables = [tableName];

      const metadataMap = Object.fromEntries(
        Object.entries(modelMetadata).filter(([key]) => selectedTables.includes(key))
      );
      const existing = Object.values(metadataMap).filter((m: any) => m.exists);

      let overwrite = false;
      if (existing.length > 0) {
        const warningMsg = existing.map((m: any) => `${m.table_name} (created ${new Date(m.created_at).toLocaleDateString()} by ${m.created_by || 'unknown'})`).join('\n');
        overwrite = await confirm({ title: 'Existing model detected', description: `The following models already exist:\n${warningMsg}\n\nOverwrite them?` });
      }

      const genResp = await authFetch<{generated:any[]; skipped:string[]; overwritten:string[]}>(
        '/api/fabric/models/generate',
        { method: 'POST', json: { tenant_instance_id: datasource.id, scope: { type: 'tables', names: selectedTables }, overwrite, accept_relationships: true } }
      );
      if (!genResp.ok) {
        devError(`Failed to generate model: ${genResp.status}`, genResp.error);
  setGenerationError(`Failed to generate model: ${genResp.error || `HTTP ${genResp.status}`}`);
  toast.error(`Failed to generate model: ${genResp.error || `HTTP ${genResp.status}`}`);
        return;
      }
      const { generated = [], skipped = [], overwritten = [] } = genResp.data || ({} as any);
      if (generated.length > 0) {
        // Show generated model in editor
        const config = generated[0].resolved_config || generated[0];
        setGeneratedJson(JSON.stringify(config, null, 2));
        setGeneratedYaml(yaml.dump(config));
        // Update metadata
        setModelMetadata(prev => {
          const next = { ...prev };
          generated.forEach((m: any) => {
            if (m.table_name) {
              const dotForm = m.table_name;
              const slashForm = m.table_name.replace('.', '/');
              const leadingSlash = '/' + slashForm;
              [dotForm, slashForm, leadingSlash].forEach(k => { next[k] = { ...next[k], exists: true, title: m.model_name }; });
            }
          });
          return next;
        });

        // Create custom for the generated model
        const modelKey = generated[0].model_key || '/' + generated[0].table_name.replace('.', '/');
        await createCustomModelIfNotExists(modelKey);
      } else if (skipped.length === 0 && overwritten.length === 0) {
        setGenerationError('No model was generated for this table.');
        toast('No model was generated for this table.');
      }
    } catch (err: any) {
      devError('Failed to generate for table:', err);
  setGenerationError(err.message || String(err));
  toast.error(err.message || String(err));
    } finally {
      setGenerationLoading(false);
    }
  };

  const handleLoadExistingModel = async (tableName: string) => {
    if (!datasource) return;

    try {
      setGenerationLoading(true);
      setGenerationError(null);
      setGeneratedJson('');
      setGeneratedYaml('');

      // The model_key in the DB is "/public/users", but the tableName is "public.users"
      const modelKey = '/' + tableName.replace('.', '/');

      const defResp = await authFetch<any>(`/api/fabric/models/definition?tenant_instance_id=${datasource.id}&model_key=${encodeURIComponent(modelKey)}`);
      if (!defResp.ok) {
        devError(`Failed to load model definition: ${defResp.status}`, defResp.error);
        setGenerationError(`Failed to load model: ${defResp.error || `HTTP ${defResp.status}`}`);
        return;
      }
  const modelDef = defResp.data;
      
      if (modelDef) {
        // If the currently selected catalog model is custom, create an 'extends' block
        if (catalogSelectedModel && catalogSelectedModel.is_custom === true && modelDef.parent_model_key) {
          const customConfig = {
            extends: modelDef.parent_model_key,
            ...(modelDef.resolved_config || {})
          };
          setGeneratedJson(JSON.stringify(customConfig, null, 2));
          setGeneratedYaml(yaml.dump(customConfig));
        } else {
          setGeneratedJson(JSON.stringify(modelDef.resolved_config || modelDef, null, 2));
          setGeneratedYaml(yaml.dump(modelDef.resolved_config || modelDef));
        }
      } else {
        setGenerationError("Loaded model definition is empty or invalid.");
      }
    } catch (err: any) {
  devError('💥 Failed to load existing model:', err);
      setGenerationError(err.message);
    } finally {
      setGenerationLoading(false);
    }
  };

  // Add jump to section functionality for the editor
  useEffect(() => {
    const handleJumpToSection = (e: any) => {
      const { section } = e.detail || {};
      if (!section) return;

      const content = activeGeneratorTab === 'json' ? generatedJson : generatedYaml;
      if (!content) return;

      const lines = content.split('\n');
      let targetLine = -1;

      // Find the line that contains the section
      for (let i = 0; i < lines.length; i++) {
        const line = lines[i].toLowerCase();
        if (section === 'dimensions' && (line.includes('"dimensions"') || line.includes('dimensions:'))) {
          targetLine = i;
          break;
        } else if (section === 'measures' && (line.includes('"measures"') || line.includes('measures:'))) {
          targetLine = i;
          break;
        } else if (section === 'joins' && (line.includes('"joins"') || line.includes('joins:'))) {
          targetLine = i;
          break;
        } else if (section === 'filters' && (line.includes('"segments"') || line.includes('"filters"') || line.includes('segments:') || line.includes('filters:'))) {
          targetLine = i;
          break;
        }
      }

      if (targetLine >= 0) {
        // Prefer Monaco API to reveal the line (1-based)
        try {
          const api = monacoApiRef.current;
          if (api && typeof api.revealRange === 'function') {
            api.revealRange(targetLine + 1, targetLine + 1);
            api.focus?.();
            return;
          }
        } catch {}
        // Fallback: DOM scroll on editor container
        const editorElement = document.querySelector('.editor-input') as HTMLElement;
        if (editorElement) {
          const lineHeight = parseFloat(getComputedStyle(editorElement).lineHeight || '18');
          editorElement.scrollTop = Math.max(0, targetLine * lineHeight - 40);
          (editorElement as HTMLElement).focus();
        }
      }
    };

    window.addEventListener('semlayer.jumpToSection', handleJumpToSection);
    return () => window.removeEventListener('semlayer.jumpToSection', handleJumpToSection);
  }, [generatedJson, generatedYaml, activeGeneratorTab]);

  // Calculate matches when search term changes
  useEffect(() => {
    const term = editorSearch.trim();
    if (!term) {
      setMatchCount(0);
      setMatchIndex(0);
      return;
    }

    const content = activeGeneratorTab === 'json' ? generatedJson : generatedYaml;
    if (!content) {
      setMatchCount(0);
      setMatchIndex(0);
      return;
    }

    const lines = content.split('\n');
    const matches = lines.filter(line => line.toLowerCase().includes(term.toLowerCase()));
    setMatchCount(matches.length);
    setMatchIndex(0);
  }, [editorSearch, generatedJson, generatedYaml, activeGeneratorTab]);

  // Navigate to match
  const navigateMatch = (direction: number) => {
    const term = editorSearch.trim();
    if (!term || matchCount === 0) return;

    const content = activeGeneratorTab === 'json' ? generatedJson : generatedYaml;
    if (!content) return;

    const lines = content.split('\n');
    const matches: Array<{ line: number }> = [];
    lines.forEach((line, i) => {
      if (line.toLowerCase().includes(term.toLowerCase())) {
        matches.push({ line: i });
      }
    });

    const newIndex = (matchIndex + direction + matches.length) % matches.length;
    setMatchIndex(newIndex);

    // Scroll to the match
    const targetLine = matches[newIndex].line;
    // Prefer Monaco API when available
    try {
      const api = monacoApiRef.current;
      if (api && typeof api.revealRange === 'function') {
        api.revealRange(targetLine + 1, targetLine + 1);
        api.focus?.();
        return;
      }
    } catch {}
    // Fallback: DOM scroll on editor container
    const editorElement = document.querySelector('.editor-input') as HTMLElement;
    if (editorElement) {
      const lineHeight = parseFloat(getComputedStyle(editorElement).lineHeight || '18');
      editorElement.scrollTop = Math.max(0, targetLine * lineHeight - 40);
      editorElement.focus();
    }
  };

  return (
    <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <Box sx={{ flexGrow: 1, display: 'flex', overflow: 'hidden' }}>
  {/* Left Panel - Data Catalog */}
  <Paper sx={{ width: '420px', minWidth: '360px', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <Typography variant="h6" gutterBottom>
              Data Catalog
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {datasource ? 
                `${datasource.source_name} (${nodes.length} tables)` : 
                'No datasource selected'
              }
            </Typography>
            <TextField
              fullWidth
              size="small"
              placeholder="Search tables..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              sx={{ mt: 2 }}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon />
                  </InputAdornment>
                ),
              }}
            />
          </Box>
          
          <Box sx={{ flex: 1, overflow: 'auto' }}>
            {loading && (
              <Box sx={{ p: 2, textAlign: 'center' }}>
                <CircularProgress size={24} />
                <Typography variant="body2" sx={{ mt: 1 }}>
                  Loading schema...
                </Typography>
              </Box>
            )}
            
            {chartError && !loading && (
              <Alert severity="error" sx={{ m: 1 }}>
                <Typography variant="body2" component="div">
                  {chartError.message}
                </Typography>
                <Typography variant="caption" sx={{ mt: 1, display: 'block' }}>
                  Check console for more details. Try running the catalog scanner first.
                </Typography>
              </Alert>
            )}
            
            {!loading && !chartError && nodes.length > 0 && (
              <DataCatalogTree
                nodes={nodes}
                onAssetSelect={() => {}} // Single-select not used here
                searchTerm={searchTerm}
                highlightedItem={null}
                multiselect={true}
                selection={selection}
                onSelectionChange={setSelection}
                modelMetadata={modelMetadata}
                showColumns={false}
                onModelIconClick={handleLoadExistingModel}
                onGenerateModelForTable={handleGenerateForTable}
                showGoldCopyIcon={true}
              />
            )}
            
            {!loading && !chartError && nodes.length === 0 && (
              <Box sx={{ p: 2, textAlign: 'center' }}>
                <Typography color="text.secondary" sx={{ mb: 2 }}>
                  No tables found in this datasource.
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ mb: 2, display: 'block' }}>
                  This might mean:
                </Typography>
                <Typography variant="caption" component="div" sx={{ textAlign: 'left', mb: 2 }}>
                  • The catalog scanner hasn't been run yet<br/>
                  • The database is empty<br/>
                  • There's a connection issue<br/>
                  • The schema endpoint is returning empty data
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
                  Try running the catalog scanner from the Connections page first.
                </Typography>
              </Box>
            )}
          </Box>
        </Paper>

        {/* Right Panel - Generated Model */}
        <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Box>
                <Typography variant="h6" gutterBottom>
                  Generated Semantic Model
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {selection.size} item(s) selected for generation.
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Button 
                  variant="contained" 
                  size="small"
                  onClick={handleGenerateForSelection}
                  disabled={selection.size === 0 || generationLoading}
                >
                  {generationLoading ? 'Generating...' : `Generate for Selection (${selection.size})`}
                </Button>
                <Button
                  variant="outlined"
                  size="small"
                  onClick={() => setSelection(new Set())}
                  disabled={selection.size === 0 || generationLoading}
                >
                  Clear
                </Button>
              </Box>
            </Box>
            
            {generationError && (
              <Alert severity="error" sx={{ mt: 2, mx: 2 }}>
                {generationError}
              </Alert>
            )}
            
            {datasource && (
              <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                Using: {datasource.source_name} ({datasource.alpha_datasource?.datasource_type || 'unknown'})
              </Typography>
            )}
            
            <Divider />
            
            
          </Box>
          
          <Paper 
            sx={{ 
              flex: 1, 
              m: 2, 
              p: 2, 
              overflow: 'auto',
              backgroundColor: '#f5f5f5'
            }}
          >
            <Box sx={{ mb: 1, display: 'flex', gap: 1, alignItems: 'center' }}>
              <Box sx={{ minWidth: 320 }}>
                <ProfessionalSearchInput
                  value={editorSearch}
                  onChange={async (v) => { 
                    setEditorSearch(v); 
                    if (showSemanticSuggestions && v && v.length >= 2) {
                      // Search semantic terms from backend
                      setIsSearchingSemanticTerms(true);
                      const results = await searchSemanticTerms(v);
                      // Sort suggestions by returned score (desc)
                      results.sort((a: any, b: any) => ((b as any).result?.score || 0) - ((a as any).result?.score || 0));
                      setSemanticSuggestions(results);
                      setIsSearchingSemanticTerms(false);
                    } else if (showSemanticSuggestions && (!v || v.length < 2)) {
                      // Clear suggestions if search term is too short
                      setSemanticSuggestions([]);
                    } else if (showSuggestions) {
                      // Hide content search suggestions when typing
                      setShowSuggestions(false);
                    }
                  }}
                  onSuggestionSelect={(s: SearchSuggestion) => {
                    if (!s) return;
                    if (showSemanticSuggestions) {
                      // For semantic term suggestions, maybe copy to clipboard or something
                      // For now, just log it
                      devLog('Selected semantic term:', s);
                      setShowSemanticSuggestions(false);
                    } else {
                      // suggestions store key in id
                      window.dispatchEvent(new CustomEvent('semlayer.jumpToSection', { detail: { section: s.description || s.type || 'dimensions', key: s.id } }));
                      setShowSuggestions(false);
                    }
                  }}
                  suggestions={
                    showSemanticSuggestions 
                      ? semanticSuggestions.slice(0, 200)
                      : searchTokens.slice(0, 200).map(t => ({ id: t.key, title: t.label, description: t.section }))
                  }
                  placeholder={showSemanticSuggestions ? "Search semantic terms..." : "Type to search content, press Ctrl+Space for semantic terms..."}
                  // Show semantic suggestions even when editorSearch is empty so the
                  // user sees a starter list right after pressing Ctrl+Space.
                  showSuggestions={
                    showSemanticSuggestions || (showSuggestions && !!editorSearch)
                  }
                  onFocus={() => { /* no-op */ }}
                  onBlur={() => { /* no-op */ }}
                  navigationEnabled={true}
                  currentMatch={matchCount > 0 ? matchIndex + 1 : 0}
                  totalMatches={matchCount}
                  onNavigateMatch={navigateMatch}
                  onKeyDown={(e) => {
                    // Show semantic term suggestions when user presses Ctrl+Space
                    if (e.ctrlKey && e.key === ' ') {
                      e.preventDefault();
                      setShowSemanticSuggestions(true);
                      setShowSuggestions(false);
                    }
                    // Hide suggestions on Escape
                    if (e.key === 'Escape') {
                      setShowSuggestions(false);
                      setShowSemanticSuggestions(false);
                    }
                  }}
                  loading={isSearchingSemanticTerms}
                />
                {/* Temporary semantic toggle so users don't need Ctrl+Space */}
                <Button
                  size="small"
                  variant={showSemanticSuggestions ? 'contained' : 'outlined'}
                  sx={{ ml: 1, height: 36 }}
                  startIcon={<TablerIcons.IconSearch size={14} />}
                  onClick={async () => {
                    // toggle semantic suggestion mode
                    const next = !showSemanticSuggestions;
                    setShowSemanticSuggestions(next);
                    try { localStorage.setItem('mg_show_semantic', next ? 'true' : 'false'); } catch {}
                    // If enabling, fetch a short default list (empty query) and show a toast
                    if (next) {
                      try {
                        setIsSearchingSemanticTerms(true);
                        const results = await searchSemanticTerms('');
                        // Sort and set
                        results.sort((a: any, b: any) => ((b as any).result?.score || 0) - ((a as any).result?.score || 0));
                        setSemanticSuggestions(results);
                        if (results.length > 0) {
                          try { toast.success(`Loaded ${results.length} semantic suggestions`); } catch {}
                        } else {
                          try { toast(`No semantic suggestions available`); } catch {}
                        }
                      } catch (e) {
                        try { toast.error?.('Failed to load semantic suggestions'); } catch {}
                      } finally {
                        setIsSearchingSemanticTerms(false);
                      }
                    } else {
                      // clearing suggestions when disabling
                      setSemanticSuggestions([]);
                    }
                  }}
                  disabled={!datasource}
                >
                  Semantic
                </Button>
              </Box>
              <Box sx={{ display: 'flex', gap: 1, ml: 2, alignItems: 'center' }}>
                <Typography variant="body2" sx={{ mr: 1 }}>Totals:</Typography>
                <Tooltip title={`Jump to first ${singularForSection.dimensions}`}>
                  <Button
                    size="small"
                    variant="text"
                    startIcon={<Icons.IconDatabase size={16} style={{ color: 'var(--dimension-color)' }} />}
                    onClick={() => handleJumpToSection('dimensions')}
                    sx={{ color: 'var(--dimension-color)' }}
                  >
                    {sectionCounts.dimensions ?? 0}
                  </Button>
                </Tooltip>
                <Tooltip title={`Jump to first ${singularForSection.measures}`}>
                  <Button
                    size="small"
                    variant="text"
                    startIcon={<Icons.IconChartBar size={16} style={{ color: 'var(--measure-color)' }} />}
                    onClick={() => handleJumpToSection('measures')}
                    sx={{ color: 'var(--measure-color)' }}
                  >
                    {sectionCounts.measures ?? 0}
                  </Button>
                </Tooltip>
                <Tooltip title={`Jump to first ${singularForSection.joins}`}>
                  <Button
                    size="small"
                    variant="text"
                    startIcon={<TablerIcons.IconPlugConnected size={16} style={{ color: 'var(--join-color)' }} />}
                    onClick={() => handleJumpToSection('joins')}
                    sx={{ color: 'var(--join-color)' }}
                  >
                    {sectionCounts.joins ?? 0}
                  </Button>
                </Tooltip>
                <Tooltip title={`Jump to first ${singularForSection.filters}`}>
                  <Button
                    size="small"
                    variant="text"
                    startIcon={<Icons.IconFilter size={16} style={{ color: 'var(--filter-color)' }} />}
                    onClick={() => handleJumpToSection('filters')}
                    sx={{ color: 'var(--filter-color)' }}
                  >
                    {sectionCounts.filters ?? 0}
                  </Button>
                </Tooltip>
              </Box>

              <Box sx={{ ml: 'auto', display: 'flex', gap: 1, alignItems: 'center' }}>
                <div className="format-toggle-group">
                  <button
                    className={`format-toggle-btn ${activeGeneratorTab === 'json' ? 'active' : ''}`}
                    onClick={() => setActiveGeneratorTab('json')}
                    title="View JSON output"
                  >
                    <Icons.IconFileText size={14} style={{ marginRight: 8 }} />
                    JSON
                  </button>
                  <button
                    className={`format-toggle-btn ${activeGeneratorTab === 'yaml' ? 'active' : ''}`}
                    onClick={() => setActiveGeneratorTab('yaml')}
                    title="View YAML output"
                  >
                    <Icons.IconFileText size={14} style={{ marginRight: 8 }} />
                    YAML
                  </button>
                </div>
              </Box>
            </Box>

            {/* Move editor below search controls */}
            <div className="prism-generator-editor-wrapper">
              {activeGeneratorTab === 'json' ? (
                <MonacoCodeEditor
                  value={generatedJson || '// Select items from the Data Catalog and click "Generate for Selection" to see the JSON model'}
                  language="json"
                  readOnly
                  onMount={(api: any) => { monacoApiRef.current = api; }}
                />
              ) : (
                <MonacoCodeEditor
                  value={generatedYaml || '# Select items from the Data Catalog and click "Generate for Selection" to see the YAML model'}
                  language="yaml"
                  readOnly
                  onMount={(api: any) => { monacoApiRef.current = api; }}
                />
              )}
            </div>

            {/* Removed matches dialog as it's not working */}
          </Paper>
        </Box>
      </Box>
    </Box>
  );
};

export default ModelGenerator;