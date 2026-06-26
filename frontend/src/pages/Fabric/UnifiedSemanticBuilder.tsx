import React, { useEffect, useMemo, useState, useCallback } from 'react';
import { devLog } from '../../utils/devLogger';
import usePaletteDrop from '../../hooks/usePaletteDrop';
import CatalogSidebarWrapper from '../../components/UnifiedSemanticBuilder/CatalogSidebarWrapper';
import LoadingView, { ErrorView } from '../../components/UnifiedSemanticBuilder/BuilderStatus';
import { useCoreModelBuilder } from '../../hooks/useCoreModelBuilder';
import useShowCodeSync from '../../hooks/useShowCodeSync';
import { useUnifiedSemanticBuilder } from '../../hooks/useUnifiedSemanticBuilder';
import { useModelCatalog } from '../../hooks/useModelCatalog';
import '../../components/UnifiedSemanticBuilder/ModelCatalogSidebar.css';
import '../../components/UnifiedSemanticBuilder/DataCatalogSidebar.css';
import './UnifiedSemanticBuilder.css';
import BuilderHeader from '../../components/UnifiedSemanticBuilder/BuilderHeader';
import useModelSaver from '../../hooks/useModelSaver';
import useEnsureCustomAndAdd from '../../hooks/useEnsureCustomAndAdd';
import useClipboard from '../../hooks/useClipboard';

import type { ModelCatalogNode } from '../../types/model';
import type { CoreOption } from '../../components/UnifiedSemanticBuilder/financialCalculations';
import { libraryOptions } from '../../components/UnifiedSemanticBuilder/financialCalculations';
import '../../components/UnifiedSemanticBuilder/SemanticPalette.css';
import { useBuilderGenerators, computeCustomModelDiff } from '../../hooks/useBuilderGenerators';
import useCompatibility from '../../hooks/useCompatibility';
import useModelCreator from '../../hooks/useModelCreator';
import useElementCreator from '../../hooks/useElementCreator';
import yaml from 'js-yaml';
import type { SemanticModel } from '../../components/UnifiedSemanticBuilder/types';
import { useAuthFetch } from '../../utils/authFetch';
import resolveApiUrl from '../../utils/resolveApiUrl';
import WorkspaceMain from '../../components/UnifiedSemanticBuilder/WorkspaceMain';
import { GlobalSearchProvider } from '../../contexts/GlobalSearchContext';
import ModelModals from '../../components/UnifiedSemanticBuilder/ModelModals';
import type { ElementKind } from '../../components/UnifiedSemanticBuilder/AddElementModal';
import ConfirmDeleteModal from '../../components/UnifiedSemanticBuilder/ConfirmDeleteModal';

interface UnifiedSemanticBuilderProps {
  tenantId: string;
  datasourceId: string;
  alphaDatasourceId?: string;
  onClose: () => void;
}

const UnifiedSemanticBuilder: React.FC<UnifiedSemanticBuilderProps> = ({ 
  tenantId,
  datasourceId, 
  onClose 
}) => {
  // Track selected element by id (dimension/measure/filter/join)
  const [selectedElementId, setSelectedElementId] = useState<string | null>(null);
  // Centralized authenticated fetch, declared at component top-level per Rules of Hooks
  const { authFetch } = useAuthFetch();
  const {
    nodes,
    searchTerm: _searchTerm,
    setSearchTerm: _setSearchTerm,
    selectedColumn,
    setSelectedColumn: _setSelectedColumn,
    modelName,
    setModelName,
    showCode,
    setShowCode,
    businessTerms,
    semanticTerms,
    semanticViews: _semanticViews,
    semanticModel,
    setSemanticModel,
    columnMappings: _columnMappings,
    setColumnMappings: _setColumnMappings,
    chartLoading,
    chartError,
    businessLoading,
    businessError,
    isNumericType: _isNumericType,
    getColumnMapping: _getColumnMapping,
    getMappingColor: _getMappingColor,
    getBusinessTermForColumn,
    addDimension: rawAddDimension,
    addMeasure: rawAddMeasure,
    addFilter: rawAddFilter,
    removeSemanticElement,
    toggleElementEdit,
    updateSemanticElement,
    generateJSON: rawGenerateJSON,
    generateYAML: rawGenerateYAML,
    filteredNodes,
    showNotification,
  } = useUnifiedSemanticBuilder(datasourceId);
  

  const {
    models: catalogModels,
    selectedModel,
    setSelectedModel,
    searchTerm: _modelSearchTerm,
    setSearchTerm: _setModelSearchTerm,
    loading: modelsLoading,
    error: modelsError,
    createCustomModel,
    cloneModel,
    updateModel,
  refreshModels,
    deleteModel,
  } = useModelCatalog(tenantId, datasourceId);
  
  const coreOptions: CoreOption[] = useMemo(() => {
    const coreBusinessTerms = (businessTerms || [])
      .filter(term => Boolean(asRecord(term)?.isCore))
      .map(term => {
        const t = asRecord(term) || {};
        const p = (t.properties as Record<string, unknown>) || {};
        return {
          name: getString(t, 'node_name'),
          title: getString(t, 'node_name'),
          type: (p.type as string) || 'string',
          sql: (t.qualified_path as string) || getString(t, 'node_name'),
          description: getString(t, 'description') || '',
          sourceTable: '',
          sourceColumn: getString(t, 'node_name'),
          format: p.format as string | undefined,
          aggregationType: p.aggregationType as string | undefined,
          defaultValue: p.defaultValue as unknown,
        } as CoreOption;
      });

    const coreSemanticTerms = (semanticTerms || [])
      .map(term => {
        const t = asRecord(term) || {};
        const p = (t.properties as Record<string, unknown>) || {};
        return {
          name: getString(t, 'node_name'),
          title: getString(t, 'node_name'),
          type: (p.type as string) || 'string',
          sql: (t.qualified_path as string) || getString(t, 'node_name'),
          description: getString(t, 'description') || '',
          sourceTable: '',
          sourceColumn: getString(t, 'node_name'),
          format: p.format as string | undefined,
          aggregationType: p.aggregationType as string | undefined,
          defaultValue: p.defaultValue as unknown,
        } as CoreOption;
      });

    const sampleCoreOptions: CoreOption[] = [
      { name: 'customer_id', title: 'Customer ID', type: 'dimension', sql: 'customers.id', description: 'Unique customer identifier', sourceTable: 'customers', sourceColumn: 'id' },
      { name: 'order_date', title: 'Order Date', type: 'dimension', sql: 'orders.order_date', description: 'Date of the order', sourceTable: 'orders', sourceColumn: 'order_date' },
      { name: 'product_name', title: 'Product Name', type: 'dimension', sql: 'products.name', description: 'Name of the product', sourceTable: 'products', sourceColumn: 'name' },
    ];

    return [...coreBusinessTerms, ...coreSemanticTerms, ...sampleCoreOptions];
  }, [businessTerms, semanticTerms]);

  const { ensureCustomAndApply, wrapAdd, enhancedRemove } = useEnsureCustomAndAdd({ selectedModel, createCustomModel, showNotification });

  const enhancedRemoveSemanticElement = enhancedRemove(removeSemanticElement);
  const addDimension = wrapAdd(rawAddDimension);
  const addMeasure = wrapAdd(rawAddMeasure);
  const addFilter = wrapAdd(rawAddFilter);

  const {
    generateCustomModelObject: _generateCustomModelObject,
    generateMergedModelObject,
    generateJSON: _generateJSON,
    generateYAML: _generateYAML,
    generateCoreJSON,
    generateCoreYAML,
    generateCustomJSON,
    generateCustomYAML,
  } = useBuilderGenerators({ selectedModel, catalogModels, semanticModel, rawGenerateJSON, rawGenerateYAML });

  // isSaving is provided by useModelSaver
  const { isSaving, handleSave } = useModelSaver({ selectedModel, modelName, semanticModel, createCustomModel, updateModel, setSemanticModel, setModelName, showNotification });
  const [issueLevelFilter, setIssueLevelFilter] = useState<'all' | 'error' | 'warning'>('all');
  const [issueCodeFilter, setIssueCodeFilter] = useState<string>('');
  const [expandChanges, setExpandChanges] = useState<Record<string, boolean>>({});
  const [expandIssues, setExpandIssues] = useState<Record<string, boolean>>({});
  const [isCodeDirty, setIsCodeDirty] = useState(false);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  // Global search term shared across catalog, editor, palette and code panels
  const [globalSearchTerm, setGlobalSearchTerm] = useState('');
  const { compat: _compat, compatErr, compatLoading, refreshCompatibility, filteredCompat } = useCompatibility({ datasourceId, issueLevelFilter, issueCodeFilter });
  const [activeWorkspaceTab, setActiveWorkspaceTab] = useState<'canvas' | 'custom' | 'calculations'>('canvas');
  const [_activeEditorTab, setActiveEditorTab] = useState<'core' | 'custom' | 'merged'>('custom');
  const [editMode, setEditMode] = useState(false);
  const { formatType: _formatType, setFormatType: _setFormatType } = useShowCodeSync(showCode, setShowCode);
  const { isOver, drop } = usePaletteDrop('model');

  // Track the active catalog tab for auto-selection logic
  const [activeCatalogTab, setActiveCatalogTab] = useState<'core' | 'custom'>('core');
  // Remember last selected model per tab so we can restore focus when switching back
  const [lastSelectedByTab, setLastSelectedByTab] = useState<{ core: string | null; custom: string | null }>({ core: null, custom: null });

  // Modal state for adding elements
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [pendingKind, setPendingKind] = useState<ElementKind | null>(null);
  const [pendingTargetTable, setPendingTargetTable] = useState<string | { id: string; qualified_path: string } | null>(null);
  const [createCustomModelModalOpen, setCreateCustomModelModalOpen] = useState(false);
  const [pendingModel, setPendingModel] = useState<any | null>(null);
  const [pendingTargetTab, setPendingTargetTab] = useState<'core' | 'custom' | null>(null);
  const [showUnsavedConfirm, setShowUnsavedConfirm] = useState(false);

  const openAddModal = (kind: ElementKind, targetTable?: string | { id: string; qualified_path: string } | null) => {
    setPendingKind(kind);
    setPendingTargetTable(targetTable || null);
    setAddModalOpen(true);
  };

  const { handleCreateCustomModel } = useModelCreator({ createCustomModel, setSemanticModel, setModelName, showNotification });

  // Small local narrowers to avoid widespread `as any` casts in render logic
  const asRecord = (o: unknown): Record<string, unknown> | undefined => (typeof o === 'object' && o !== null) ? (o as Record<string, unknown>) : undefined;
  const getString = (o: unknown, k: string): string => {
    const r = asRecord(o);
    const v = r?.[k];
    return v == null ? '' : String(v);
  };
  const _getProp = <T,>(o: unknown, k: string): T | undefined => {
    const r = asRecord(o);
    return r ? (r[k] as T | undefined) : undefined;
  };

  // Helper: parse incoming JSON/YAML code into the "custom-only" semanticModel shape
  const parseCodeToCustomModel = (text: string, format: 'json' | 'yaml' | 'jsonc' | null) => {
    try {
      const obj = (format === 'yaml') ? yaml.load(text) : JSON.parse(text);
      // crude normalization: pick first cube if present, or top-level measures/dimensions
      const pickCube = (raw: any) => {
        if (!raw) return null;
        if (Array.isArray(raw.cubes) && raw.cubes.length) return raw.cubes[0];
        if (raw.cube) return raw.cube;
        return raw;
      };
      const chosen = pickCube(obj) || {};
      const normEntries = (items: unknown) => {
        if (!items) return [];
        if (Array.isArray(items)) return items.map((it: unknown, idx: number) => ({ id: `imported_${idx}_${Date.now()}`, is_custom: true, ...(asRecord(it) || {}) }));
        if (typeof items === 'object') return Object.entries(items as Record<string, unknown>).map(([k, v]) => ({ id: `imported_${k}_${Date.now()}`, is_custom: true, name: getString(v, 'name') || k, ...(asRecord(v) || {}) }));
        return [];
      };
  const customModel = {
        name: chosen.name || obj.name || modelName || 'semantic_model',
        dimensions: normEntries(chosen.dimensions || obj.dimensions),
        measures: normEntries(chosen.measures || obj.measures),
        filters: normEntries(chosen.filters || obj.filters || obj.segments),
        joins: normEntries(chosen.joins || obj.joins),
        is_custom: true,
  } as unknown;
      return customModel;
    } catch (e) {
      return null;
    }
  };

  // Handler invoked by CodePanel when user imports/applies code (live or explicit)
  const handleImportCode = async (text: string, format: 'json' | 'yaml' | 'jsonc' | null) => {
    const parsed = parseCodeToCustomModel(text, format);
    if (!parsed) return;
    try {
      // update semantic model state so tiles re-render
  setSemanticModel(parsed as unknown as SemanticModel);
      setHasUnsavedChanges(true);
      setIsCodeDirty(true);
    } catch (e) {
      // ignore
    }
  };

  const handleCloneModel = useCallback(async (baseModelKey: string) => {
    try {
      const clonedModel = await cloneModel(baseModelKey);
      showNotification(`Model "${clonedModel.display_name || clonedModel.model_key}" cloned successfully!`, 'success');
    } catch (error) {
      showNotification(`Failed to clone model: ${error instanceof Error ? error.message : 'Unknown error'}`, 'error');
    }
  }, [cloneModel, showNotification]);

  const handleArchiveModel = useCallback(async (modelId: string, _isCore?: boolean, _modelKey?: string) => {
    try {
  await updateModel(modelId, { status: 'archived' } as Record<string, unknown>);
      showNotification('Model archived', 'success');
    } catch (error) {
      showNotification(`Failed to archive: ${error instanceof Error ? error.message : 'Unknown error'}`, 'error');
    }
  }, [updateModel, showNotification]);

  const handlePublishModel = useCallback(async (modelId: string) => {
    try {
  await updateModel(modelId, { status: 'published' } as Record<string, unknown>);
      showNotification('Model published', 'success');
    } catch (error) {
      showNotification(`Failed to publish: ${error instanceof Error ? error.message : 'Unknown error'}`, 'error');
    }
  }, [updateModel, showNotification]);

  const handleDraftModel = useCallback(async (modelId: string) => {
    try {
  await updateModel(modelId, { status: 'draft' } as Record<string, unknown>);
      showNotification('Model moved to draft', 'success');
    } catch (error) {
      showNotification(`Failed to set draft: ${error instanceof Error ? error.message : 'Unknown error'}`, 'error');
    }
  }, [updateModel, showNotification]);

  const handleDeleteModel = useCallback(async (modelId: string, isCore?: boolean, modelKey?: string) => {
    const result = await deleteModel(modelId, isCore, modelKey);
    if (result.success) {
      const modelType = isCore ? 'core model' : 'custom model';
      showNotification(`${modelType} "${modelKey || 'Unknown'}" deleted successfully!`, 'success');
    } else {
      const modelType = isCore ? 'core model' : 'custom model';
      showNotification(`Failed to delete ${modelType}: ${result.error?.message || 'Unknown error'}`, 'error');
    }
  }, [deleteModel, showNotification]);

  const { handleCreateElement } = useElementCreator({ setSemanticModel, selectedColumn, ensureCustomAndApply });

  const { copyToClipboard: _copyToClipboard } = useClipboard();

  // compatibility is handled by useCompatibility hook which auto-refreshes on datasourceId


  // Track the last loaded model to prevent overwriting user changes
  const [_lastLoadedModelKey, setLastLoadedModelKey] = useState<string | null>(null); // intentionally kept for future use

  // When a model is selected from the catalog, load its data into the builder.
  // Core model building via custom hook
  useCoreModelBuilder({
    selectedModel,
    datasourceId,
    setSemanticModel,
    setLastLoadedModelKey,
    showNotification,
  });

  // When a custom model is selected, populate the editor with ONLY custom tiles
  // Prefer source_config if provided (custom-only). Otherwise, compute a diff
  // vs. the parent resolved_config and normalize it for the canvas.
  useEffect(() => {
    const toArray = (val: any) => Array.isArray(val)
      ? val
      : val && typeof val === 'object'
      ? Object.entries(val).map(([name, v]: any) => ({ name, ...(v || {}) }))
      : [];
      const pickCube = (raw: unknown) => {
        if (!raw) return null;
        const r = asRecord(raw) || {};
        if (r.cubes && Array.isArray(r.cubes)) {
          const matchName = String(r.name ?? selectedModel?.model_key ?? selectedModel?.display_name);
          const cubes = Array.isArray(r.cubes) ? (r.cubes as unknown[]) : [];
          const found = cubes.find(c => {
            if (!c || typeof c !== 'object') return false;
            return String((c as Record<string, unknown>)['name'] ?? '') === matchName;
          });
          return found ?? cubes[0] ?? raw;
        }
        if (r.cube && typeof r.cube === 'object') return raw;
        return raw;
      };
    const normEntries = (val: any, kind: 'dimension'|'measure'|'filter'|'join'|'pre_aggregation') => {
      const arr: any[] = toArray(val);
      const now = Date.now();
      return arr.map((e: any, idx: number) => ({
        ...e,
        id: e?.id || `${kind}_${e?.name || idx}_${now}`,
        title: e?.title || e?.name || `${kind} ${idx}`,
        sourceTable: e?.sourceTable || e?.table || '',
        sourceColumn: e?.sourceColumn || e?.column || '',
        is_custom: true,
        isEditing: false,
      }));
    };

    const buildCustomOnlyFromConfig = (cfg: any) => {
      const chosen: any = pickCube(cfg) || {};
      const getField = (key: string) => {
        const chosenRec = asRecord(chosen) || {};
        const cfgRec = asRecord(cfg) || {};
        const chosenCube = asRecord(chosenRec.cube) || {};
        // If chosen has cube and the direct key is undefined, prefer cube[key]
        if (chosenRec && chosenRec.cube && chosenRec[key] === undefined) return chosenRec[key] ?? chosenCube[key] ?? cfgRec[key];
        return chosenRec[key] ?? cfgRec[key];
      };
      const rawName = (asRecord(cfg)?.name as string) || chosen?.name || selectedModel?.display_name || selectedModel?.model_key || 'semantic_model';
      const rawFilters = getField('filters') ?? getField('segments');
      const customModel = {
        name: rawName,
        dimensions: normEntries(getField('dimensions'), 'dimension'),
        measures: normEntries(getField('measures'), 'measure'),
        filters: normEntries(rawFilters, 'filter'),
        joins: normEntries(getField('joins'), 'join'),
  } as Record<string, unknown>;
      const preAggs = normEntries(getField('pre_aggregations'), 'pre_aggregation');
      if (preAggs.length) customModel.pre_aggregations = preAggs;
      return customModel;
    };

    if (selectedModel && selectedModel.is_custom) {
      try {
        // 1) If source_config exists, it should already be custom-only
        if (selectedModel.source_config) {
          const customOnly = buildCustomOnlyFromConfig(selectedModel.source_config);
          setSemanticModel({ ...customOnly, is_custom: true } as unknown as SemanticModel);
          return;
        }

        // 2) If only resolved_config is present, compute diff vs parent and fill details
        if (selectedModel.resolved_config && selectedModel.parent_model_key) {
          const parent = catalogModels.find((m: any) => m.model_key === selectedModel.parent_model_key);
          if (parent && parent.resolved_config) {
            // Normalize current (selected custom) and parent to array-shapes for lookup
            const currAll = buildCustomOnlyFromConfig(selectedModel.resolved_config);
            const parentCfg = parent.resolved_config;
            const diff = computeCustomModelDiff({ selectedModel, parentResolvedConfig: parentCfg, currentConfig: currAll });

            const fillFromCurr = (items: any[] | undefined, kind: 'dimension'|'measure'|'filter'|'join'|'pre_aggregation', currList: any[]) => {
              const now = Date.now();
              return (items || []).map((it: any, idx: number) => {
                const name = it?.name;
                const curr = (currList || []).find((c: any) => c?.name === name) || {};
                return {
                  id: `custom_${kind}_${name || idx}_${now}`,
                  title: it?.title || curr?.title || name || `${kind} ${idx}`,
                  sourceTable: it?.sourceTable || curr?.sourceTable || curr?.table || '',
                  sourceColumn: it?.sourceColumn || curr?.sourceColumn || curr?.column || '',
                  is_custom: true,
                  isEditing: false,
                  ...curr,
                  ...it,
                };
              });
            };

                  const currAllRec = asRecord(currAll) || {};
                  const diffDims = _getProp<unknown[]>(diff, 'dimensions') || [];
                  const diffMeasures = _getProp<unknown[]>(diff, 'measures') || [];
                  const diffFilters = _getProp<unknown[]>(diff, 'filters') || [];
                  const diffJoins = _getProp<unknown[]>(diff, 'joins') || [];
                  const diffPreAggs = _getProp<unknown[]>(diff, 'pre_aggregations') || [];
                  const currAllDims = Array.isArray(currAllRec.dimensions) ? (currAllRec.dimensions as unknown[]) : [];
                  const currAllMeasures = Array.isArray(currAllRec.measures) ? (currAllRec.measures as unknown[]) : [];
                  const currAllFilters = Array.isArray(currAllRec.filters) ? (currAllRec.filters as unknown[]) : [];
                  const currAllJoins = Array.isArray(currAllRec.joins) ? (currAllRec.joins as unknown[]) : [];
                  const currAllPreAggs = Array.isArray(currAllRec.pre_aggregations) ? (currAllRec.pre_aggregations as unknown[]) : [];
                  const customOnly = {
                    name: currAll.name,
                    dimensions: fillFromCurr(diffDims || [], 'dimension', currAllDims || []),
                    measures: fillFromCurr(diffMeasures || [], 'measure', currAllMeasures || []),
                    filters: fillFromCurr(diffFilters || [], 'filter', currAllFilters || []),
                    joins: fillFromCurr(diffJoins || [], 'join', currAllJoins || []),
                  } as Record<string, unknown>;
                  const preAggs = fillFromCurr(diffPreAggs || [], 'pre_aggregation', currAllPreAggs || []);
            if (preAggs.length) (customOnly as Record<string, unknown>).pre_aggregations = preAggs;
            setSemanticModel({ ...(customOnly as Record<string, unknown>), is_custom: true } as unknown as SemanticModel);
            return;
          }
        }
      } catch (e) {
        // Fall through to no-op; we won't override current editor content on error
      }
    }
  }, [selectedModel?.id, selectedModel?.is_custom, selectedModel?.source_config, selectedModel?.resolved_config, selectedModel?.parent_model_key, catalogModels, setSemanticModel]);

  // When the save action completes, reset the dirty flag for the code editor.
  const wasSaving = React.useRef(false);
  useEffect(() => {
    if (wasSaving.current && !isSaving) {
      setIsCodeDirty(false);
      setHasUnsavedChanges(false);
    }
    wasSaving.current = isSaving;
  }, [isSaving]);

  // Listen for a global mark-dirty event (e.g., from code edits)
  useEffect(() => {
    const handler = () => setHasUnsavedChanges(true);
    window.addEventListener('semlayer.markDirty', handler);
    return () => window.removeEventListener('semlayer.markDirty', handler);
  }, []);

  // Listen for requests from child components to open the code view
  useEffect(() => {
    const onOpenCode = () => {
      setActiveWorkspaceTab('custom');
      // Ensure the code panel shows JSON format
      try { setShowCode('json'); } catch {}
    };
    window.addEventListener('semlayer.openCode', onOpenCode);
    return () => window.removeEventListener('semlayer.openCode', onOpenCode);
  }, [setShowCode]);

  // Warn when navigating away with unsaved changes
  useEffect(() => {
    const beforeUnload = (e: BeforeUnloadEvent) => {
      if (!hasUnsavedChanges) return;
      e.preventDefault();
      e.returnValue = '';
      return '';
    };
    if (hasUnsavedChanges) {
      window.addEventListener('beforeunload', beforeUnload);
    } else {
      window.removeEventListener('beforeunload', beforeUnload);
    }
    return () => window.removeEventListener('beforeunload', beforeUnload);
  }, [hasUnsavedChanges]);

  // Handle rename events from sidebar
  useEffect(()=>{
    const handler = (e: Event) => {
      const custom = (e as CustomEvent | Event);
      const detail = (custom as CustomEvent).detail as { id?: string; title?: string } | undefined;
      if (!detail || !detail.id) return;
      // optimistic local update if selected
      if (selectedModel && selectedModel.id === detail.id) {
        setModelName(String(detail.title ?? ''));
      }
      const payload: Record<string, unknown> = { title: String(detail.title ?? '') };
      updateModel(detail.id, payload).catch(err => {
        showNotification(`Rename failed: ${err instanceof Error ? err.message : 'Unknown error'}`, 'error');
      });
    };
    // Use a properly typed EventListener for add/remove
    const _handler: EventListener = handler as EventListener;
    document.addEventListener('model.rename', _handler);
    return () => document.removeEventListener('model.rename', _handler);
  },[updateModel, showNotification, selectedModel, setModelName]);

  // Update modelName when selectedModel changes
  useEffect(() => {
    if (selectedModel) {
      const name = selectedModel.display_name || selectedModel.title || selectedModel.model_key || 'semantic_model';
      setModelName(name);
    }
  }, [selectedModel?.id, selectedModel?.display_name, selectedModel?.title, selectedModel?.model_key, setModelName]);

  // Avoid early returns that would change hook call order; render conditionally in JSX instead

  // Explicitly cast catalogModels to unknown first, then to ModelCatalogNode[]
  const processedCatalogModels = (catalogModels || []).map((model) => ({
    ...model,
    id: model.id || '',
    model_key: model.model_key || '',
    status: (model.status as 'draft' | 'published' | 'archived') || 'draft', // Default to 'draft'
    version: typeof model.version === 'number' ? model.version : parseFloat(String(model.version) || '1'), // Convert to number safely
    can_edit: model.can_edit || false,
    core_model_exists: model.core_model_exists || false,
  }));

  // Safely compute existing element names for modals (handle undefined semanticModel fields)
  const existingNames = useMemo(() => {
    try {
      const mRec = asRecord(semanticModel) || {};
      const dims = Array.isArray(mRec.dimensions) ? (mRec.dimensions as unknown[]) : [];
      const meas = Array.isArray(mRec.measures) ? (mRec.measures as unknown[]) : [];
      const filts = Array.isArray(mRec.filters) ? (mRec.filters as unknown[]) : [];
      return [...dims, ...meas, ...filts].map((e: unknown) => String(asRecord(e)?.name || '')).filter(Boolean) as string[];
    } catch {
      return [] as string[];
    }
  }, [semanticModel]);

  // Options for selecting a base model to extend from
  const baseModelOptions = useMemo(() => {
    try {
      return processedCatalogModels
        .map((m: any) => ({ key: m.model_key, label: m.display_name || m.title || m.model_key, kind: (m.is_custom ? 'custom' : 'core') as 'core' | 'custom' }));
    } catch {
      return [] as Array<{ key: string; label: string; kind: 'core' | 'custom' }>;
    }
  }, [processedCatalogModels, selectedModel]);

  // Change the base model (extends) for the currently selected custom model
  const onChangeExtends = useCallback(async (newBaseKey: string) => {
    let prevSelected: any = null;
    try {
      // DEBUG: trace extends change attempts (tests may remove this later)
  // eslint-disable-next-line no-console
  try { devLog('[UnifiedSemanticBuilder] onChangeExtends called', { newBaseKey, selectedModelKey: getString(selectedModel, 'model_key'), parent: getString(selectedModel, 'parent_model_key') }); } catch {}
      const notifyLocal = (msg: string, level?: 'success' | 'error') => {
        try { devLog('[UnifiedSemanticBuilder] notifyLocal', { msg, level }); } catch {}
        try { showNotification?.(msg, level); } catch {}
      };
      if (!selectedModel || !selectedModel.is_custom) return;
      if (!newBaseKey) {
        notifyLocal('Clearing base model is not supported by the API yet.', 'error');
        return;
      }
      // Prevent invalid or idempotent selections
      // Immediate guard with normalization (covers dot vs slash)
      const normalizeKey = (s: any) => {
        let v = (s || '').toString().trim().toLowerCase();
        if (!v) return '';
        if (v.includes('.')) v = v.replace(/\.+/g, '/');
        if (!v.startsWith('/')) v = '/' + v;
        v = v.replace(/\/+/, '/');
        return v;
      };
      const newNorm = normalizeKey(newBaseKey);
  const selfNorm = normalizeKey(getString(selectedModel, 'model_key'));
  const parentNorm = normalizeKey(getString(selectedModel, 'parent_model_key'));
      if (newNorm === selfNorm) { notifyLocal('Invalid extends: a model cannot extend itself.', 'error'); return; }
      if (newNorm === parentNorm) { notifyLocal(`Model already extends "${newBaseKey}"`, 'success'); return; }
      try {
        const { analyzeExtendsSelection } = await import('./extendsUtils');
  const res = analyzeExtendsSelection(newBaseKey, getString(selectedModel, 'model_key'), getString(selectedModel, 'parent_model_key'));
        if (!res.valid) {
          if (res.reason === 'self') { notifyLocal('Invalid extends: a model cannot extend itself.', 'error'); }
          else if (res.reason === 'idempotent') { notifyLocal(`Model already extends "${newBaseKey}"`, 'success'); }
          else { notifyLocal('Invalid extends selection', 'error'); }
          return;
        }
      } catch (e) {
        // fallback: previous inline validation
        const norm = (s: any) => (s || '').toString().trim().toLowerCase();
  if (norm(newBaseKey) === norm(selectedModel.model_key)) { notifyLocal('Invalid extends: a model cannot extend itself.', 'error'); return; }
  if (norm(newBaseKey) === norm(getString(selectedModel, 'parent_model_key'))) { notifyLocal(`Model already extends "${newBaseKey}"`, 'success'); return; }
      }
  // Optimistically update selectedModel so the UI (extends tile + code) reflects the change immediately
  prevSelected = selectedModel;
  // Build optimistic object safely from selectedModel record
  const optimistic = { ...(asRecord(selectedModel) || {}), parent_model_key: newBaseKey };
  try { setSelectedModel(optimistic as unknown as ModelCatalogNode); } catch {}
      // Build extension payload using the current custom-only model object
    let modelObj: any = null;
      try {
        // Prefer generator hook if available
        const maybeObj = (generateMergedModelObject ? generateMergedModelObject() : null);
        // Fallback to custom JSON parse
        modelObj = maybeObj || (generateCustomJSON ? JSON.parse(generateCustomJSON()) : JSON.parse(rawGenerateJSON()));
      } catch {}
      const payload: any = {
        base_model_key: newBaseKey,
        model_key: selectedModel.model_key,
        title: selectedModel.title || selectedModel.display_name || selectedModel.model_key,
        description: selectedModel.description || '',
        status: selectedModel.status || 'draft',
        model_object: modelObj || {},
      };
  const resp = await authFetch<any>(resolveApiUrl(`/api/fabric/extensions?tenant_instance_id=${encodeURIComponent(datasourceId)}`), { method: 'POST', json: payload });
  if (!resp.ok) throw new Error(resp.error || `Request failed: ${resp.status}`);
  const data = resp.data;
      // Refresh models and optimistically update selected
      try { await refreshModels(); } catch {}
  const updated = data && (data.model || data);
  const merged = { ...(asRecord(prevSelected) || {}), parent_model_key: newBaseKey, ...(updated || {}) };
  try { setSelectedModel(merged as unknown as ModelCatalogNode); } catch {}
  showNotification?.(`Updated base model to "${newBaseKey}"`, 'success');
    } catch (err: any) {
  // Revert optimistic change
  try { setSelectedModel(prevSelected); } catch {}
  try {
    const eid = `extends__${newBaseKey}`;
    const issues = [{ level: 'error', code: 'extend_update_failed', message: err?.message || 'Failed to update extends', element_id: eid }];
    // Broadcast so SemanticModelOverview can highlight tiles
    window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } }));
  } catch {}
  showNotification?.(`Failed to update extends: ${err?.message || 'Unknown error'}`, 'error');
    }
  }, [selectedModel, datasourceId, refreshModels, showNotification, generateMergedModelObject, generateCustomJSON, rawGenerateJSON]);

  // Handle catalog tab changes with auto-selection and per-tab memory
  const handleCatalogTabChange = useCallback((tab: 'core' | 'custom') => {
    setActiveCatalogTab(tab);

    const modelsForTab = processedCatalogModels.filter(model =>
      tab === 'core' ? (model.is_core && !model.is_custom) : model.is_custom
    );

    // If we have a remembered selection for this tab, restore it
    const rememberedId = tab === 'core' ? lastSelectedByTab.core : lastSelectedByTab.custom;
    const rememberedModel = rememberedId ? modelsForTab.find(m => m.id === rememberedId) : undefined;

    if (rememberedModel) {
      setSelectedModel(rememberedModel);
      try { setGlobalSearchTerm(''); } catch {}
      // Exit edit mode for core so canvas renders read-only tiles
      if (!rememberedModel.is_custom) setEditMode(false);
      // Force canvas view and hide code to reflect the selected model immediately
      setActiveWorkspaceTab('canvas');
      setShowCode(null);
      // nudge canvas to focus for immediate interaction
      try { window.dispatchEvent(new CustomEvent('semlayer.focusCanvas')); } catch {}
      return;
    }

    // Otherwise, if no model selected yet for this tab, select the first one
    if (modelsForTab.length > 0) {
      const firstModel = modelsForTab[0];
      setSelectedModel(firstModel);
  setLastSelectedByTab(prev => ({ ...prev, [tab]: firstModel.id }));
      try { setGlobalSearchTerm(''); } catch {}
      if (!firstModel.is_custom) setEditMode(false);
      // Force canvas view and hide code, so tiles update immediately
      setActiveWorkspaceTab('canvas');
      setShowCode(null);
      // Reset editor dirty flags when switching context
      setIsCodeDirty(false);
      setHasUnsavedChanges(false);
      // nudge canvas focus
      try { window.dispatchEvent(new CustomEvent('semlayer.focusCanvas')); } catch {}
    }
  }, [processedCatalogModels, lastSelectedByTab, setSelectedModel, setGlobalSearchTerm, setEditMode, setActiveWorkspaceTab, setShowCode, setIsCodeDirty, setHasUnsavedChanges]);

  // Helper: derive selected element from semanticModel by id
  const selectedElement = useMemo(() => {
    if (!selectedElementId || !semanticModel) return null;
    const mRec = asRecord(semanticModel) || {};
    const dims = Array.isArray(mRec.dimensions) ? (mRec.dimensions as unknown[]) : [];
    const meas = Array.isArray(mRec.measures) ? (mRec.measures as unknown[]) : [];
    const filts = Array.isArray(mRec.filters) ? (mRec.filters as unknown[]) : [];
    const joins = Array.isArray(mRec.joins) ? (mRec.joins as unknown[]) : [];
    const preAggs = Array.isArray(mRec.pre_aggregations) ? (mRec.pre_aggregations as unknown[]) : [];
    const all = [...dims, ...meas, ...filts, ...joins, ...preAggs];
    return all.find(el => asRecord(el)?.id === selectedElementId) || null;
  }, [selectedElementId, semanticModel]);

  // Pass setSelectedElementId to SemanticModelOverview for tile click
  // Pass selectedElement to EnhancedTileForm (via WorkspaceMain or direct)

  return (
    <GlobalSearchProvider value={{ searchTerm: globalSearchTerm, setSearchTerm: setGlobalSearchTerm }}>
      {(chartLoading || businessLoading) ? (
        <LoadingView />
      ) : (chartError || businessError) ? (
        <ErrorView error={chartError?.message || businessError?.message} onClose={onClose} />
      ) : (
        <div className="builder-container">
          <BuilderHeader
            modelName={modelName}
            setModelName={setModelName}
            selectedModel={selectedModel as ModelCatalogNode | null}
            handleSave={handleSave}
            isSaving={isSaving}
          />
          {/* Main Content Area */}
          <div className="builder-content">
            <CatalogSidebarWrapper
              processedCatalogModels={processedCatalogModels}
              modelSearchTerm={globalSearchTerm}
              setModelSearchTerm={setGlobalSearchTerm}
              selectedModel={selectedModel}
              setSelectedModel={setSelectedModel}
              setCreateCustomModelModalOpen={setCreateCustomModelModalOpen}
              onCloneModel={handleCloneModel}
              onModelSelect={(model: ModelCatalogNode, targetTab: 'core' | 'custom') => {
                if (hasUnsavedChanges) {
                  setPendingModel(model);
                  setPendingTargetTab(targetTab);
                  setShowUnsavedConfirm(true);
                  return;
                }
                setSelectedModel(model);
                setLastSelectedByTab(prev => ({ ...prev, [targetTab]: model.id }));
                if (targetTab === 'core') setActiveEditorTab('core');
                else setActiveEditorTab('custom');
                if (!model.is_custom) setEditMode(false);
                setIsCodeDirty(false);
                setHasUnsavedChanges(false);
                setSelectedElementId(null); // clear selection on model change
              }}
              modelsLoading={modelsLoading}
              modelsError={modelsError?.message}
              onDeleteModel={handleDeleteModel}
              onArchiveModel={handleArchiveModel}
              onPublishModel={handlePublishModel}
              onDraftModel={handleDraftModel}
              onEnterEditMode={() => {
                setEditMode(true);
                setActiveWorkspaceTab('canvas');
                setShowCode(null);
                window.dispatchEvent(new CustomEvent('semlayer.focusCanvas'));
              }}
              activeTab={activeCatalogTab}
              onTabChange={handleCatalogTabChange}
            />
            <WorkspaceMain
              isOver={isOver}
              drop={drop}
              activeWorkspaceTab={activeWorkspaceTab}
              setActiveWorkspaceTab={setActiveWorkspaceTab}
              editMode={editMode}
              setEditMode={setEditMode}
              selectedColumn={selectedColumn}
              addDimension={(...args) => { setHasUnsavedChanges(true); addDimension(...args); }}
              addMeasure={(...args) => { setHasUnsavedChanges(true); addMeasure(...args); }}
              addFilter={(...args) => { setHasUnsavedChanges(true); addFilter(...args); }}
              getBusinessTermForColumn={getBusinessTermForColumn}
              semanticModel={semanticModel}
              setSemanticModel={setSemanticModel}
              modelName={modelName}
              showCode={showCode}
              setShowCode={setShowCode}
              rawGenerateJSON={rawGenerateJSON}
              rawGenerateYAML={rawGenerateYAML}
              generateCoreJSON={generateCoreJSON}
              generateCoreYAML={generateCoreYAML}
              generateCustomJSON={generateCustomJSON}
              generateCustomYAML={generateCustomYAML}
              generateMergedModelObject={generateMergedModelObject}
              selectedModel={selectedModel}
              openAddModal={openAddModal}
              enhancedRemoveSemanticElement={(type, id) => { setHasUnsavedChanges(true); return enhancedRemoveSemanticElement(type, id); }}
              toggleElementEdit={toggleElementEdit}
              updateSemanticElement={(type, id, updates) => { setHasUnsavedChanges(true); return updateSemanticElement(type, id, updates); }}
              coreOptions={coreOptions}
              refreshCompatibility={refreshCompatibility}
              compatLoading={compatLoading}
              issueLevelFilter={issueLevelFilter}
              setIssueLevelFilter={setIssueLevelFilter}
              issueCodeFilter={issueCodeFilter}
              setIssueCodeFilter={setIssueCodeFilter}
              compatErr={compatErr}
              filteredCompat={filteredCompat}
              filteredNodes={filteredNodes}
              expandIssues={expandIssues}
              setExpandIssues={setExpandIssues}
              expandChanges={expandChanges}
              setExpandChanges={setExpandChanges}
              isCodeDirty={isCodeDirty}
              setIsCodeDirty={setIsCodeDirty}
              availableBaseModels={baseModelOptions}
              onChangeExtends={onChangeExtends}
              onImportCode={handleImportCode}
              selectedElement={selectedElement}
              setSelectedElementId={setSelectedElementId}
            />
          </div>
        </div>
      )}
      <ModelModals
        addModalOpen={addModalOpen}
        pendingKind={pendingKind}
        pendingTargetTable={pendingTargetTable}
        setAddModalOpen={setAddModalOpen}
        onCreateElement={handleCreateElement}
        coreOptions={coreOptions}
        libraryOptions={libraryOptions}
        existingNames={existingNames}
        nodes={nodes}
        semanticModel={semanticModel}
        createCustomModelModalOpen={createCustomModelModalOpen}
        setCreateCustomModelModalOpen={setCreateCustomModelModalOpen}
        handleCreateCustomModel={handleCreateCustomModel}
      />
      <ConfirmDeleteModal
        open={showUnsavedConfirm}
        title="Unsaved changes"
        message="You have unsaved changes. Discard and switch models?"
        onCancel={() => {
          setShowUnsavedConfirm(false);
          setPendingModel(null);
          setPendingTargetTab(null);
        }}
          onConfirm={() => {
            if (pendingModel) {
              setSelectedModel(pendingModel as unknown as ModelCatalogNode);
              if (pendingTargetTab === 'core') setActiveEditorTab('core');
              if (pendingTargetTab === 'custom') setActiveEditorTab('custom');
              if (!asRecord(pendingModel)?.is_custom) setEditMode(false);
              setSelectedElementId(null);
            }
            setIsCodeDirty(false);
            setHasUnsavedChanges(false);
            setShowUnsavedConfirm(false);
            setPendingModel(null);
            setPendingTargetTab(null);
          }}
      />
    </GlobalSearchProvider>
  );
};

export default UnifiedSemanticBuilder;