import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Box,
  Typography,
  Button,
  Paper,
  IconButton,
  Stack,
  Avatar,
  Alert,
  CircularProgress,
  TextField,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Tooltip,
  ToggleButtonGroup,
  ToggleButton,
  Menu,
  MenuItem,
  Drawer,
  useTheme,
} from '@mui/material';
import {
  PlayArrow as PlayIcon,
  Save as SaveIcon,
  Download as DownloadIcon,
  Edit as EditIcon,
  Storage as StorageIcon,
  Code as CodeIcon,
  AccountTree as PlanIcon,
  DataObject as JsonIcon,
  TableChart as TableIcon,
  BarChart as BarChartIcon,
  ShowChart as LineChartIcon,
  PieChart as PieChartIcon,
  ScatterPlot as ScatterIcon,
  Insights as KpiIcon,
  PostAdd as PromoteIcon,
  FileDownload as FileDownloadIcon,
} from '@mui/icons-material';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  createNewQueryTab,
} from '../features/data-explorer/types/dataExplorerTypes';
import { useExplorerTheme } from '../features/data-explorer/hooks/useExplorerTheme';
import type {
  BusinessObjectSummary,
  ExplorerQueryState,
  ExplorerSource,
  FilterSelection,
  SavedExplorerQuery,
  SortSelection,
  ViewMode,
  QueryTabState,
  CalculationDefinition,
  QueryParameter,
  ScheduleConfig,
  ShareConfig,
} from '../features/data-explorer/types/dataExplorerTypes';
import { emptyExplorerState } from '../features/data-explorer/types/dataExplorerTypes';
import {
  loadExplorerSource,
  fetchBusinessObjects,
} from '../features/data-explorer/services/dataExplorerApi';
import { useExplorerConversations } from '../features/data-explorer/hooks/useExplorerConversations';
import { useDataExplorerQuery } from '../features/data-explorer/hooks/useDataExplorerQuery';
import { useSavedExplorerQueries } from '../features/data-explorer/hooks/useSavedExplorerQueries';
import { SemanticPalette } from '../features/data-explorer/components/SemanticPalette';
import { QueryDefinitionBar } from '../features/data-explorer/components/QueryDefinitionBar';
import { FilterModal } from '../features/data-explorer/components/FilterModal';
import { FilterPillBar } from '../features/data-explorer/components/FilterPillBar';
import { ResultsTablePane } from '../features/data-explorer/components/ResultsTablePane';
import { ModelPickerDialog } from '../features/data-explorer/components/ModelPickerDialog';
import { UnifiedBOPickerModal } from '../components/common/UnifiedBOPickerModal';
import { VisualizationPane } from '../features/data-explorer/components/VisualizationPane';
import { SqlPreviewDialog } from '../features/data-explorer/components/SqlPreviewDialog';
import { ExplainPlanPane } from '../features/data-explorer/components/ExplainPlanPane';
import { JsonPreviewPane } from '../features/data-explorer/components/JsonPreviewPane';
import { AIPromptBar } from '../features/data-explorer/components/AIPromptBar';
import { AIInsightBadge } from '../features/data-explorer/components/AIInsightBadge';
import { DisambiguationBanner } from '../features/data-explorer/components/DisambiguationBanner';
import { ConversationRail } from '../features/data-explorer/components/ConversationRail';
import { ExplorerLandingHero } from '../features/data-explorer/components/ExplorerLandingHero';
import { QueryTabManager } from '../features/data-explorer/components/QueryTabManager';
import { CalculationModal } from '../features/data-explorer/components/CalculationModal';
import { ParametersToolbar } from '../features/data-explorer/components/ParametersToolbar';
import { ScheduleQueryModal } from '../features/data-explorer/components/ScheduleQueryModal';
import { ShareQueryModal } from '../features/data-explorer/components/ShareQueryModal';
import { useExplorerAI } from '../features/data-explorer/hooks/useExplorerAI';
import { useQueryTelemetry } from '../features/data-explorer/hooks/useQueryTelemetry';
import { SmartVisualizerPane } from '../features/data-explorer/components/SmartVisualizerPane';
import { DrilldownBreadcrumbs } from '../features/data-explorer/components/DrilldownBreadcrumbs';
import { SemanticReviewQueue } from '../features/data-explorer/components/SemanticReviewQueue';
import { RCAInsightPanel } from '../features/data-explorer/components/RCAInsightPanel';
import { QueryMutationTimeline } from '../features/data-explorer/components/QueryMutationTimeline';
import { ForecastAuditPanel } from '../features/data-explorer/components/ForecastAuditPanel';
import { Schedule as ScheduleIcon, Share as ShareIcon } from '@mui/icons-material';
import type {
  ExplorerQueryDefinition,
  SemanticField,
  ChatMessage as ExplorerChatMessage,
} from '../features/data-explorer/types/explorerTypes';
import type { ConversationMessage } from '../features/data-explorer/types/conversationTypes';
import { convertExplorerToReportBuilder } from '../features/data-explorer/types/explorerTypes';
import { devError } from '../utils/devLogger';

const QUERY_NAME_DEFAULT = 'Untitled Query';

function buildAssistantSQLHint(query: ExplorerQueryDefinition): string {
  const select = [...query.dimensions];
  for (const m of query.measures) select.push(`${m.agg}(${m.fieldId})`);
  if (select.length === 0) return '';
  return `SELECT ${select.join(', ')} FROM ${query.title || 'data'}`;
}

function inferConfidence(message: ExplorerChatMessage): number | undefined {
  const text = (message.content || '').toLowerCase();
  if (text.includes('confidence') || text.includes('not sure')) return 0.4;
  if (text.includes('synthesized') || text.includes('fallback')) return 0.5;
  return 0.85;
}

export const DataExplorerPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const explorerTheme = useExplorerTheme();

  // Multi-Tab Query Workspace
  const [tabs, setTabs] = useState<QueryTabState[]>([
    createNewQueryTab('tab-1', 'Query 1', null),
  ]);
  const [activeTabId, setActiveTabId] = useState<string>('tab-1');

  const activeTab = useMemo(
    () => tabs.find((t) => t.id === activeTabId) || tabs[0],
    [tabs, activeTabId]
  );

  const queryName = activeTab.name;
  const setQueryName = useCallback(
    (name: string) => {
      setTabs((prev) =>
        prev.map((t) => (t.id === activeTabId ? { ...t, name } : t))
      );
    },
    [activeTabId]
  );

  const source = activeTab.source;
  const setSource = useCallback(
    (src: ExplorerSource | null) => {
      setTabs((prev) =>
        prev.map((t) => (t.id === activeTabId ? { ...t, source: src } : t))
      );
    },
    [activeTabId]
  );

  const state = activeTab.queryState;
  const setState = useCallback(
    (updater: ExplorerQueryState | ((prev: ExplorerQueryState) => ExplorerQueryState)) => {
      setTabs((prevTabs) =>
        prevTabs.map((t) => {
          if (t.id === activeTabId) {
            const nextState = typeof updater === 'function' ? updater(t.queryState) : updater;
            return { ...t, queryState: nextState };
          }
          return t;
        })
      );
    },
    [activeTabId]
  );

  const viewMode = activeTab.viewMode;
  const setViewMode = useCallback(
    (mode: ViewMode) => {
      setTabs((prev) =>
        prev.map((t) => (t.id === activeTabId ? { ...t, viewMode: mode } : t))
      );
    },
    [activeTabId]
  );

  const [isEditingName, setIsEditingName] = useState(false);
  const [sourceLoadError, setSourceLoadError] = useState<string | null>(null);
  const [allBusinessObjects, setAllBusinessObjects] = useState<BusinessObjectSummary[]>([]);
  const [showModelPicker, setShowModelPicker] = useState(false);
  const [filterModalOpen, setFilterModalOpen] = useState(false);
  const [editingFilterIndex, setEditingFilterIndex] = useState<number | null>(null);
  const [calculationModalOpen, setCalculationModalOpen] = useState(false);
  const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
  const [shareModalOpen, setShareModalOpen] = useState(false);
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);
  const [saveName, setSaveName] = useState('');
  const [tab, setTab] = useState<'results' | 'sql' | 'plan' | 'json'>('results');
  const [sqlDialogOpen, setSqlDialogOpen] = useState(false);
  const [exportAnchor, setExportAnchor] = useState<null | HTMLElement>(null);
  const [reviewQueueOpen, setReviewQueueOpen] = useState(false);
  const [timelineDrawerOpen, setTimelineDrawerOpen] = useState(false);
  const [queryHistory, setQueryHistory] = useState<ExplorerQueryDefinition[]>([]);
  const { logInteraction } = useQueryTelemetry();

  // Multi-Tab actions
  const handleAddTab = useCallback(() => {
    const nextNum = tabs.length + 1;
    const newTab = createNewQueryTab(undefined, `Query ${nextNum}`, source);
    setTabs((prev) => [...prev, newTab]);
    setActiveTabId(newTab.id);
  }, [tabs.length, source]);

  const handleCloseTab = useCallback(
    (tabId: string) => {
      if (tabs.length <= 1) return;
      const nextTabs = tabs.filter((t) => t.id !== tabId);
      setTabs(nextTabs);
      if (activeTabId === tabId) {
        setActiveTabId(nextTabs[0].id);
      }
    },
    [tabs, activeTabId]
  );

  const handleRenameTab = useCallback((tabId: string, newName: string) => {
    setTabs((prev) =>
      prev.map((t) => (t.id === tabId ? { ...t, name: newName } : t))
    );
  }, []);

  const handleDuplicateTab = useCallback(
    (tabId: string) => {
      const target = tabs.find((t) => t.id === tabId);
      if (!target) return;
      const dup = {
        ...target,
        id: `tab-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`,
        name: `${target.name} (Copy)`,
      };
      setTabs((prev) => [...prev, dup]);
      setActiveTabId(dup.id);
    },
    [tabs]
  );

  const { records: savedRecords, isLoading: savedLoading, save: saveQuery, remove: removeQuery } =
    useSavedExplorerQueries();

  // Load catalog list for landing hero
  useEffect(() => {
    fetchBusinessObjects()
      .then((bos) => setAllBusinessObjects(bos))
      .catch((err) => devError('fetchBusinessObjects failed', err));
  }, []);

  const { result, isLoading, isPreviewing, error, run, lastRunAt } = useDataExplorerQuery({
    source,
    state,
  });

  // Map ExplorerSource fields into SemanticField[]
  const semanticCatalog: SemanticField[] = useMemo(() => {
    if (!source) return [];
    return source.fields.map((f) => ({
      id: f.id,
      name: f.name,
      label: f.displayName,
      category: f.category,
      type: f.type,
      aggregation: f.defaultAggregation,
      description: f.description,
    }));
  }, [source]);

  // Current query as ExplorerQueryDefinition
  const unifiedQueryDef: ExplorerQueryDefinition = useMemo(() => {
    return {
      title: queryName,
      dimensions: state.dimensions.map((d) => d.fieldId),
      measures: state.measures.map((m) => ({ fieldId: m.fieldId, agg: m.agg })),
      timeDimensions: state.timeDimensions.map((t) => ({ fieldId: t.fieldId, granularity: t.granularity || 'month' })),
      filters: state.filters.map((f, idx) => ({
        id: `f-${idx}`,
        fieldId: f.fieldId,
        operator: f.operator,
        value: f.values.join(', '),
      })),
      limit: state.limit,
      suggestedChart: viewMode,
    };
  }, [queryName, state, viewMode]);

  // Unified Query Definition change handler from AI
  const handleAIQueryChange = useCallback((newQuery: ExplorerQueryDefinition) => {
    setState((prev) => ({
      ...prev,
      dimensions: newQuery.dimensions.map((fieldId) => ({ fieldId })),
      measures: newQuery.measures.map((m) => ({ fieldId: m.fieldId, agg: m.agg })),
      timeDimensions: newQuery.timeDimensions.map((t) => ({ fieldId: t.fieldId, granularity: t.granularity })),
      filters: newQuery.filters.map((f) => ({
        fieldId: f.fieldId,
        operator: (f.operator as any) || 'equals',
        values: [String(f.value || '')],
      })),
    }));
    if (newQuery.title) {
      setQueryName(newQuery.title);
    }
  }, []);

  const handleAIExecute = useCallback(() => {
    run();
  }, [run]);

  // ─── Conversation management ────────────────────────────────────────────
  const {
    refresh: refreshConversations,
    getConversation: fetchConversation,
    createConversation: createConv,
    appendMessage: appendConvMessage,
    deleteConversation: _deleteConv,
    moveToFolder: _moveToFolder,
    pin: _pinConv,
    share: _shareConv,
    createFolder: _createFolder,
    renameFolder: _renameFolder,
    deleteFolder: _deleteFolder,
  } = useExplorerConversations();

  const [activeConversationId, setActiveConversationId] = useState<string | null>(null);

  // Persist chat messages from the AI hook into the active conversation.
  const handleAIMessagePersisted = useCallback(
    (message: ExplorerChatMessage) => {
      if (!activeConversationId) return;
      void (async () => {
        const convMessages = await fetchConversation(activeConversationId).catch(() => null);
        const previousMessages = convMessages?.messages ?? [];
        const nextId = message.id;
        if (previousMessages.some((m) => m.id === nextId)) return;
        await appendConvMessage(activeConversationId, {
          role: message.role,
          content: message.content,
          generatedSql: message.querySnapshot
            ? buildAssistantSQLHint(message.querySnapshot)
            : undefined,
          confidence: inferConfidence(message),
        } as Omit<ConversationMessage, 'id' | 'timestamp'>).catch((err) => {
          devError('Failed to persist message', err);
        });
      })();
    },
    [activeConversationId, appendConvMessage, fetchConversation]
  );

  const {
    messages: aiMessages,
    isLoading: aiLoading,
    error: aiError,
    insight,
    isCacheHit,
    suggestedFollowUps,
    ambiguities,
    submitPrompt: submitPromptRaw,
    clearConversation,
  } = useExplorerAI({
    catalog: semanticCatalog,
    currentQuery: unifiedQueryDef,
    onQueryChange: handleAIQueryChange,
    onViewModeChange: (m) => setViewMode(m as ViewMode),
    onExecuteQuery: handleAIExecute,
    onMessage: handleAIMessagePersisted,
  });

  /**
   * Auto-create a conversation whenever the user submits a prompt and
   * none is active. The persisted thread then gets every subsequent
   * message via handleAIMessagePersisted.
   */
  const submitPrompt = useCallback(
    async (promptText: string) => {
      try {
        if (!activeConversationId) {
          const created = await createConv({
            title: promptText.slice(0, 60),
            sourceId: source?.id,
            bindingId: source?.bindingId,
            queryState: state,
          });
          setActiveConversationId(created.id);
          setQueryName(created.title);
        }
      } catch (err) {
        devError('Failed to auto-create conversation', err);
      }
      await submitPromptRaw(promptText);
    },
    [activeConversationId, createConv, source, state, submitPromptRaw]
  );

  const handleSelectConversation = useCallback(
    (conv: import('../features/data-explorer/types/conversationTypes').Conversation) => {
      setActiveConversationId(conv.id);
      setQueryName(conv.title);
      if (conv.sourceId && conv.bindingId) {
        loadExplorerSource(conv.sourceId, conv.bindingId)
          .then((loaded) => {
            setSource(loaded);
            if (conv.queryState) {
              setState({
                ...conv.queryState,
                sourceId: loaded.id,
                bindingId: loaded.bindingId,
              });
            } else {
              setState(emptyExplorerState(loaded.id, loaded.bindingId));
            }
            setShowModelPicker(false);
            setQueryName(conv.title);
          })
          .catch(devError);
      }
    },
    []
  );

  const handleNewConversation = useCallback(async () => {
    const created = await createConv({
      title: 'New conversation',
      sourceId: source?.id,
      bindingId: source?.bindingId,
      queryState: state,
    });
    setActiveConversationId(created.id);
    setQueryName(created.title);
  }, [createConv, source, state]);

  useEffect(() => {
    if (!activeConversationId) return;
    void refreshConversations();
  }, [activeConversationId, refreshConversations]);

  // Deep links /data-explorer?bo=<id>&binding=<id>
  useEffect(() => {
    const search = new URLSearchParams(location.search);
    const boId = search.get('bo');
    const bindingId = search.get('binding');
    if (!boId) return;
    let cancelled = false;
    setSourceLoadError(null);
    loadExplorerSource(boId, bindingId || undefined)
      .then((loaded) => {
        if (cancelled) return;
        setSource(loaded);
        setState(emptyExplorerState(loaded.id, loaded.bindingId));
        setQueryName(`${loaded.displayName} Exploration`);
        setShowModelPicker(false);
      })
      .catch((err) => {
        if (cancelled) return;
        const message = err instanceof Error ? err.message : 'Failed to load BO.';
        setSourceLoadError(message);
        devError('loadExplorerSource failed', err);
      });
    return () => {
      cancelled = true;
    };
  }, [location.search]);

  const handlePickBo = useCallback(
    async (bo: BusinessObjectSummary) => {
      setSourceLoadError(null);
      try {
        const loaded = await loadExplorerSource(bo.id, bo.defaultBindingId);
        setSource(loaded);
        setState(emptyExplorerState(loaded.id, loaded.bindingId));
        setQueryName(`${bo.displayName} Exploration`);
        setShowModelPicker(false);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to load BO.';
        setSourceLoadError(message);
        devError('loadExplorerSource failed', err);
      }
    },
    []
  );

  const handleRunHeroPrompt = useCallback(
    async (promptText: string, targetBoHint?: string) => {
      let boToUse = allBusinessObjects[0];
      if (targetBoHint) {
        const found = allBusinessObjects.find(
          (b) =>
            b.name.toLowerCase().includes(targetBoHint.toLowerCase()) ||
            b.displayName.toLowerCase().includes(targetBoHint.toLowerCase())
        );
        if (found) boToUse = found;
      }

      if (boToUse) {
        try {
          const loaded = await loadExplorerSource(boToUse.id, boToUse.defaultBindingId);
          setSource(loaded);
          setState(emptyExplorerState(loaded.id, loaded.bindingId));
          setQueryName(promptText);
          setShowModelPicker(false);
          // Trigger prompt
          setTimeout(() => {
            submitPrompt(promptText);
          }, 150);
        } catch (e) {
          devError('Prompt setup failed', e);
        }
      }
    },
    [allBusinessObjects, submitPrompt]
  );

  const handleReopenModelPicker = useCallback(() => {
    setShowModelPicker(true);
  }, []);

  const handleClosePicker = useCallback(() => {
    setShowModelPicker(false);
  }, []);

  const handleOpenFilterModal = (_fieldId?: string) => {
    setEditingFilterIndex(null);
    setFilterModalOpen(true);
  };

  const handleEditFilterModal = (index: number) => {
    setEditingFilterIndex(index);
    setFilterModalOpen(true);
  };

  const handleApplyFilter = (filter: FilterSelection) => {
    setState((prev) => {
      const next = [...prev.filters];
      if (editingFilterIndex !== null && editingFilterIndex < next.length) {
        next[editingFilterIndex] = filter;
      } else {
        next.push(filter);
      }
      return { ...prev, filters: next };
    });
    setEditingFilterIndex(null);
  };

  const handleRemoveFilter = (index: number) => {
    setState((prev) => ({
      ...prev,
      filters: prev.filters.filter((_, i) => i !== index),
    }));
  };

  const handleToggleDimension = (fieldId: string) => {
    setState((prev) => {
      const inTime = prev.timeDimensions.some((t) => t.fieldId === fieldId);
      const inDim = prev.dimensions.some((d) => d.fieldId === fieldId);
      if (inTime) {
        return {
          ...prev,
          timeDimensions: prev.timeDimensions.filter((t) => t.fieldId !== fieldId),
        };
      }
      if (inDim) {
        return {
          ...prev,
          dimensions: prev.dimensions.filter((d) => d.fieldId !== fieldId),
        };
      }
      return {
        ...prev,
        dimensions: [...prev.dimensions, { fieldId }],
      };
    });
  };

  const handleAddTimeDimension = (fieldId: string) => {
    setState((prev) => {
      if (prev.timeDimensions.some((t) => t.fieldId === fieldId)) {
        return {
          ...prev,
          timeDimensions: prev.timeDimensions.filter((t) => t.fieldId !== fieldId),
        };
      }
      return {
        ...prev,
        timeDimensions: [...prev.timeDimensions, { fieldId, granularity: 'month' }],
        dimensions: prev.dimensions.filter((d) => d.fieldId !== fieldId),
      };
    });
  };

  const handleRemoveTimeDimension = (fieldId: string) => {
    setState((prev) => ({
      ...prev,
      timeDimensions: prev.timeDimensions.filter((t) => t.fieldId !== fieldId),
    }));
  };

  const handleRemoveDimension = (fieldId: string) => {
    setState((prev) => ({
      ...prev,
      dimensions: prev.dimensions.filter((d) => d.fieldId !== fieldId),
      timeDimensions: prev.timeDimensions.filter((t) => t.fieldId !== fieldId),
    }));
  };

  const handleToggleMeasure = (
    fieldId: string,
    defaultAgg?: ExplorerSource['fields'][number]['defaultAggregation']
  ) => {
    setState((prev) => {
      const exists = prev.measures.find((m) => m.fieldId === fieldId);
      if (exists) {
        return {
          ...prev,
          measures: prev.measures.filter((m) => m.fieldId !== fieldId),
        };
      }
      return {
        ...prev,
        measures: [
          ...prev.measures,
          { fieldId, agg: defaultAgg || 'SUM' },
        ],
      };
    });
  };

  const handleRemoveMeasure = (fieldId: string) => {
    setState((prev) => ({
      ...prev,
      measures: prev.measures.filter((m) => m.fieldId !== fieldId),
    }));
  };

  const handleUpdateDimensionExpression = useCallback(
    (fieldId: string, expression?: string) => {
      setState((prev) => ({
        ...prev,
        dimensions: prev.dimensions.map((d) =>
          d.fieldId === fieldId ? { ...d, expression } : d
        ),
        timeDimensions: prev.timeDimensions.map((t) =>
          t.fieldId === fieldId ? { ...t, expression } : t
        ),
      }));
    },
    [setState]
  );

  const handleUpdateMeasureExpression = useCallback(
    (fieldId: string, expression?: string) => {
      setState((prev) => ({
        ...prev,
        measures: prev.measures.map((m) =>
          m.fieldId === fieldId ? { ...m, expression } : m
        ),
      }));
    },
    [setState]
  );

  const handleUpdateMeasureAgg = (fieldId: string, agg: import('../features/data-explorer/types/dataExplorerTypes').AggFn) => {
    setState((prev) => ({
      ...prev,
      measures: prev.measures.map((m) =>
        m.fieldId === fieldId ? { ...m, agg } : m
      ),
    }));
  };

  const handleToggleSort = (fieldId: string) => {
    setState((prev) => {
      const existing = prev.sorts.find((s) => s.fieldId === fieldId);
      if (!existing) {
        const next: SortSelection = { fieldId, direction: 'asc' };
        return { ...prev, sorts: [...prev.sorts, next] };
      }
      if (existing.direction === 'asc') {
        return {
          ...prev,
          sorts: prev.sorts.map((s) =>
            s.fieldId === fieldId ? { ...s, direction: 'desc' as const } : s
          ),
        };
      }
      return {
        ...prev,
        sorts: prev.sorts.filter((s) => s.fieldId !== fieldId),
      };
    });
  };

  // Calculation Handlers
  const handleSaveCalculation = (calculation: CalculationDefinition) => {
    setState((prev) => {
      const existsIdx = prev.calculations.findIndex((c) => c.id === calculation.id);
      const next = [...prev.calculations];
      if (existsIdx >= 0) {
        next[existsIdx] = calculation;
      } else {
        next.push(calculation);
      }
      return { ...prev, calculations: next };
    });
  };

  const handleRemoveCalculation = (calcId: string) => {
    setState((prev) => ({
      ...prev,
      calculations: prev.calculations.filter((c) => c.id !== calcId),
    }));
  };

  // Parameter Handlers
  const handleAddParameter = (param: QueryParameter) => {
    setState((prev) => ({
      ...prev,
      parameters: [...(prev.parameters || []), param],
    }));
  };

  const handleUpdateParameter = (param: QueryParameter) => {
    setState((prev) => ({
      ...prev,
      parameters: (prev.parameters || []).map((p) => (p.id === param.id ? param : p)),
    }));
  };

  const handleRemoveParameter = (paramId: string) => {
    setState((prev) => ({
      ...prev,
      parameters: (prev.parameters || []).filter((p) => p.id !== paramId),
    }));
  };

  const handleChangeParamValue = (paramId: string, value: any) => {
    setState((prev) => ({
      ...prev,
      parameters: (prev.parameters || []).map((p) =>
        p.id === paramId ? { ...p, currentValue: value } : p
      ),
    }));
  };

  const handleLimitChange = (limit: number) => {
    setState((prev) => ({ ...prev, limit }));
  };

  const handleRun = useCallback(() => {
    run();
  }, [run]);

  const handleOpenSaveDialog = () => {
    setSaveName(queryName || QUERY_NAME_DEFAULT);
    setSaveDialogOpen(true);
  };

  const handleConfirmSave = async () => {
    if (!source) return;
    const name = saveName.trim() || QUERY_NAME_DEFAULT;
    try {
      await saveQuery({
        name,
        sourceKind: 'business_object',
        sourceId: source.id,
        queryState: state,
        createdBy: 'me',
      });
      setQueryName(name);
      setSaveDialogOpen(false);
      void logInteraction({
        prompt: name,
        generatedQuery: unifiedQueryDef,
        executedQuery: unifiedQueryDef,
        wasEdited: false,
        wasSaved: true,
        rating: 1,
      });
    } catch (err) {
      devError('Save query failed', err);
    }
  };

  const handleOpenSaved = useCallback(
    async (record: SavedExplorerQuery) => {
      try {
        const loaded = await loadExplorerSource(record.sourceId);
        setSource(loaded);
        setState({
          ...record.queryState,
          sourceId: loaded.id,
          bindingId: loaded.bindingId,
        });
        setQueryName(record.name);
        setShowModelPicker(false);
        navigate(`/data-explorer?bo=${encodeURIComponent(loaded.id)}&binding=${encodeURIComponent(loaded.bindingId)}`, { replace: true });
      } catch (err) {
        devError('Load saved query failed', err);
      }
    },
    [navigate]
  );

  const handleDeleteSaved = async (id: string) => {
    try {
      await removeQuery(id);
    } catch (err) {
      devError('Delete saved query failed', err);
    }
  };

  const handleExportCSV = () => {
    if (!result || !result.rows || result.rows.length === 0) return;
    const headers = result.columns.map((c) => `"${c.name}"`).join(',');
    const rows = result.rows
      .map((r) => result.columns.map((c) => `"${(r as any)[c.name] ?? ''}"`).join(','))
      .join('\n');
    const blob = new Blob([`${headers}\n${rows}`], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${queryName.replace(/[^a-zA-Z0-9_]/g, '_')}_export_${Date.now()}.csv`;
    a.click();
    setExportAnchor(null);

    void logInteraction({
      prompt: queryName,
      generatedQuery: unifiedQueryDef,
      executedQuery: unifiedQueryDef,
      wasEdited: false,
      wasExported: true,
    });
  };

  const handleExportJSON = () => {
    if (!result || !result.rows || result.rows.length === 0) return;
    const blob = new Blob([JSON.stringify(result.rows, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${queryName.replace(/[^a-zA-Z0-9_]/g, '_')}_export_${Date.now()}.json`;
    a.click();
    setExportAnchor(null);

    void logInteraction({
      prompt: queryName,
      generatedQuery: unifiedQueryDef,
      executedQuery: unifiedQueryDef,
      wasEdited: false,
      wasExported: true,
    });
  };

  const handleSaveSchedule = (config: ScheduleConfig) => {
    void logInteraction({
      prompt: `Schedule: ${config.scheduleName}`,
      generatedQuery: unifiedQueryDef,
      executedQuery: unifiedQueryDef,
      wasEdited: false,
      rating: 1,
    });
  };

  const handleShareQuery = (config: ShareConfig) => {
    void logInteraction({
      prompt: `Share: ${queryName}`,
      generatedQuery: unifiedQueryDef,
      executedQuery: unifiedQueryDef,
      wasEdited: false,
      rating: 1,
    });
  };

  const handleCloneToReport = () => {
    if (!source) return;
    const reportState = convertExplorerToReportBuilder(unifiedQueryDef, source.id, source.bindingId);
    sessionStorage.setItem('imported_report_builder_state', JSON.stringify(reportState));
    navigate('/reports/builder?import=explorer');
    setExportAnchor(null);

    void logInteraction({
      prompt: queryName,
      generatedQuery: unifiedQueryDef,
      executedQuery: unifiedQueryDef,
      wasEdited: false,
      clonedToReport: true,
      rating: 1,
    });
  };

  const initialFilterEditing =
    editingFilterIndex !== null ? state.filters[editingFilterIndex] : null;

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        height: 'calc(100vh - 64px)',
        bgcolor: explorerTheme.background,
      }}
    >
      {/* Top Header */}
      <Paper
        elevation={0}
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: `1px solid ${explorerTheme.border}`,
          px: 3,
          py: 1.2,
          bgcolor: 'white',
          borderRadius: 0,
        }}
      >
        <Stack direction="row" spacing={3} alignItems="center">
          <Stack direction="row" spacing={1} alignItems="center">
            <Box
              sx={{
                width: 32,
                height: 32,
                bgcolor: explorerTheme.accent,
                borderRadius: 1.5,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <StorageIcon sx={{ fontSize: 18, color: '#FFF' }} />
            </Box>
            <Typography variant="body1" fontWeight="800" sx={{ letterSpacing: -0.2 }}>
              Uuisce Data Explorer
            </Typography>
          </Stack>

          <Box sx={{ width: 1, height: 24, bgcolor: explorerTheme.border }} />

          {source ? (
            <Stack direction="row" spacing={2} alignItems="center">
              <Typography variant="body2" sx={{ color: explorerTheme.textMuted, fontWeight: 500 }}>
                Query:
              </Typography>
              {isEditingName ? (
                <TextField
                  value={queryName}
                  onChange={(e) => setQueryName(e.target.value)}
                  onBlur={() => setIsEditingName(false)}
                  autoFocus
                  variant="standard"
                  sx={{ width: 320 }}
                />
              ) : (
                <Stack
                  direction="row"
                  alignItems="center"
                  spacing={0.5}
                  sx={{ cursor: 'pointer' }}
                  onClick={() => setIsEditingName(true)}
                >
                   <Typography variant="body1" fontWeight={700} sx={{ color: explorerTheme.text }}>
                    {queryName}
                  </Typography>
                  <EditIcon sx={{ fontSize: 15, color: explorerTheme.textMuted }} />
                </Stack>
              )}
              <Button
                size="small"
                onClick={handleReopenModelPicker}
                sx={{
                  textTransform: 'none',
                  fontSize: '0.72rem',
                  color: explorerTheme.accent,
                  fontWeight: 700,
                  bgcolor: explorerTheme.accentMuted,
                  '&:hover': { bgcolor: explorerTheme.accentHover },
                }}
              >
                Change Model
              </Button>
            </Stack>
          ) : (
              <Typography variant="body2" sx={{ color: explorerTheme.textMuted }}>
              Select a semantic Business Object to explore
            </Typography>
          )}
        </Stack>

        <Stack direction="row" spacing={1.5} alignItems="center">
          <Tooltip title="Save query">
            <span>
              <IconButton size="small" disabled={!source} onClick={handleOpenSaveDialog}>
                <SaveIcon fontSize="small" />
              </IconButton>
            </span>
          </Tooltip>

          <Tooltip title="View step-by-step query mutation log">
            <Button
              size="small"
              variant="outlined"
              onClick={() => setTimelineDrawerOpen(true)}
              sx={{
                textTransform: 'none',
                fontSize: '0.75rem',
                fontWeight: 700,
                borderColor: explorerTheme.border,
                color: isDark ? '#A78BFA' : '#7C3AED',
                bgcolor: isDark ? 'rgba(167, 139, 250, 0.08)' : '#F5F3FF',
                '&:hover': { bgcolor: isDark ? 'rgba(167, 139, 250, 0.15)' : '#EDE9FE' },
              }}
            >
              Event Log
            </Button>
          </Tooltip>

          <Tooltip title="Review learned domain acronyms and auto-calculated metrics">
            <Button
              size="small"
              variant="outlined"
              onClick={() => setReviewQueueOpen(true)}
              sx={{
                textTransform: 'none',
                fontSize: '0.75rem',
                fontWeight: 700,
                borderColor: explorerTheme.border,
                color: isDark ? '#2DD4BF' : '#0D9488',
                bgcolor: isDark ? 'rgba(45, 212, 191, 0.08)' : '#F0FDFA',
                '&:hover': { bgcolor: 'rgba(45, 212, 191, 0.15)', borderColor: '#2DD4BF' },
              }}
            >
              Learned Terms
            </Button>
          </Tooltip>

          <Button
            size="small"
            variant="outlined"
            startIcon={<ShareIcon fontSize="small" />}
            disabled={!source}
            onClick={() => setShareModalOpen(true)}
            sx={{
              textTransform: 'none',
              fontSize: '0.75rem',
              fontWeight: 700,
              borderColor: explorerTheme.border,
                color: explorerTheme.accent,
            }}
          >
            Share
          </Button>

          <Button
            size="small"
            variant="outlined"
            startIcon={<ScheduleIcon fontSize="small" />}
            disabled={!source}
            onClick={() => setScheduleModalOpen(true)}
            sx={{
              textTransform: 'none',
              fontSize: '0.75rem',
              fontWeight: 700,
              borderColor: explorerTheme.border,
                color: explorerTheme.accent,
            }}
          >
            Schedule
          </Button>

          <Button
            size="small"
            variant="outlined"
            startIcon={<DownloadIcon fontSize="small" />}
            disabled={!result || result.rows.length === 0}
            onClick={(e) => setExportAnchor(e.currentTarget)}
            sx={{
              textTransform: 'none',
              fontSize: '0.75rem',
              fontWeight: 700,
              borderColor: explorerTheme.border,
              color: explorerTheme.text,
            }}
          >
            Export / Promote
          </Button>
          <Menu anchorEl={exportAnchor} open={Boolean(exportAnchor)} onClose={() => setExportAnchor(null)}>
            <MenuItem onClick={handleExportCSV} sx={{ fontSize: '0.78rem', gap: 1 }}>
              <FileDownloadIcon fontSize="small" /> Export as CSV
            </MenuItem>
            <MenuItem onClick={handleExportJSON} sx={{ fontSize: '0.78rem', gap: 1 }}>
              <JsonIcon fontSize="small" /> Export as JSON
            </MenuItem>
            <MenuItem onClick={handleCloneToReport} sx={{ fontSize: '0.78rem', gap: 1 }}>
              <PromoteIcon fontSize="small" sx={{ color: explorerTheme.accent }} /> Save as Report Dataset (SSRS)
            </MenuItem>
          </Menu>

          <Button
            variant="contained"
            startIcon={isLoading || aiLoading ? <CircularProgress size={14} color="inherit" /> : <PlayIcon />}
            onClick={handleRun}
            disabled={!source || isLoading || aiLoading}
            sx={{
              bgcolor: explorerTheme.accent,
              color: '#FFF',
              borderRadius: 2,
              px: 2.5,
              fontWeight: 700,
              fontSize: 13,
              textTransform: 'none',
              boxShadow: `0 2px 8px ${explorerTheme.accentMuted}`,
              '&:hover': { bgcolor: explorerTheme.accentDark },
            }}
          >
            Run Query
          </Button>
          <Avatar sx={{ width: 34, height: 34, border: `1px solid ${explorerTheme.border}`, bgcolor: explorerTheme.backgroundElevated }} />
        </Stack>
      </Paper>

      {/* Multi-Tab Workspace Bar */}
      {source && (
        <QueryTabManager
          tabs={tabs}
          activeTabId={activeTabId}
          onSelectTab={(id) => setActiveTabId(id)}
          onAddTab={handleAddTab}
          onCloseTab={handleCloseTab}
          onRenameTab={handleRenameTab}
          onDuplicateTab={handleDuplicateTab}
        />
      )}

      {sourceLoadError && (
        <Alert severity="error" sx={{ m: 2 }}>
          {sourceLoadError}
        </Alert>
      )}

      {/* Main workbench or Redesigned Landing Hero */}
      <Box sx={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {source && (
          <ConversationRail
            activeId={activeConversationId}
            onSelect={handleSelectConversation}
            onNewConversation={handleNewConversation}
          />
        )}
        {source ? (
          <>
            <SemanticPalette
              source={source}
              state={state}
              onToggleDimension={handleToggleDimension}
              onToggleMeasure={handleToggleMeasure}
              onAddTimeDimension={handleAddTimeDimension}
              onOpenFilterModal={handleOpenFilterModal}
              onOpenCalculationModal={() => setCalculationModalOpen(true)}
              onRemoveCalculation={handleRemoveCalculation}
            />

            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
              {/* Natural Language AI Prompt Bar */}
              <AIPromptBar
                onSendPrompt={submitPrompt}
                isLoading={aiLoading}
                suggestedFollowUps={suggestedFollowUps}
                error={aiError}
                onResetChat={clearConversation}
              />

              {/* Parameters Bar */}
              <ParametersToolbar
                parameters={state.parameters || []}
                onAddParameter={handleAddParameter}
                onUpdateParameter={handleUpdateParameter}
                onRemoveParameter={handleRemoveParameter}
                onChangeParamValue={handleChangeParamValue}
              />

              {/* Ambiguity Disambiguation Banner if present */}
              {ambiguities.length > 0 && (
                <Box sx={{ px: 2, pt: 1 }}>
                  <DisambiguationBanner
                    question={ambiguities[0]}
                    options={[
                      {
                        label: 'Use Default Dimension',
                        action: () => {
                          ambiguities.shift();
                        },
                      },
                    ]}
                    onDismiss={() => {
                      ambiguities.length = 0;
                    }}
                  />
                </Box>
              )}

              {/* Interactive Query Shelves */}
              <QueryDefinitionBar
                source={source}
                state={state}
                onRemoveDimension={handleRemoveDimension}
                onRemoveMeasure={handleRemoveMeasure}
                onRemoveTimeDimension={handleRemoveTimeDimension}
                onUpdateMeasureAgg={handleUpdateMeasureAgg}
                onUpdateDimensionExpression={handleUpdateDimensionExpression}
                onUpdateMeasureExpression={handleUpdateMeasureExpression}
                onToggleSort={handleToggleSort}
                onLimitChange={handleLimitChange}
              />
              <FilterPillBar
                source={source}
                filters={state.filters}
                onAddFilter={() => handleOpenFilterModal()}
                onRemoveFilter={handleRemoveFilter}
                onEditFilter={handleEditFilterModal}
              />

              {/* Status Header */}
              {lastRunAt && result && (
                <Box
                  sx={{
                    px: 3,
                    py: 0.5,
                    bgcolor: explorerTheme.backgroundElevated,
          borderBottom: `1px solid ${explorerTheme.border}`,
                    display: 'flex',
                    justifyContent: 'space-between',
                  }}
                >
                  <Typography variant="caption" sx={{ color: explorerTheme.textMuted, fontWeight: 600 }}>
                    {result.rowCount.toLocaleString()} rows · {result.executionTimeMs} ms
                  </Typography>
                  {result.warnings && result.warnings.length > 0 && (
                      <Typography variant="caption" sx={{ color: explorerTheme.warning, fontWeight: 600 }}>
                      {result.warnings[0]}
                    </Typography>
                  )}
                </Box>
              )}

              {/* Visualization toolbar + result tabs */}
              <Paper
                elevation={0}
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  px: 2,
                  py: 0.8,
                  bgcolor: explorerTheme.backgroundElevated,
                  borderBottom: `1px solid ${explorerTheme.border}`,
                  borderRadius: 0,
                }}
              >
                <Stack direction="row" spacing={0.5} alignItems="center">
                  <Tooltip title="Results table">
                    <IconButton
                      size="small"
                      onClick={() => setTab('results')}
                      sx={{
                        color: tab === 'results' ? explorerTheme.accent : explorerTheme.textMuted,
                        bgcolor: tab === 'results' ? explorerTheme.accentMuted : 'transparent',
                      }}
                    >
                      <TableIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="SQL">
                    <IconButton
                      size="small"
                      onClick={() => setTab('sql')}
                      sx={{
                        color: tab === 'sql' ? explorerTheme.accent : explorerTheme.textMuted,
                        bgcolor: tab === 'sql' ? explorerTheme.accentMuted : 'transparent',
                      }}
                    >
                      <CodeIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Explain Plan">
                    <IconButton
                      size="small"
                      onClick={() => setTab('plan')}
                      sx={{
                        color: tab === 'plan' ? explorerTheme.accent : explorerTheme.textMuted,
                        bgcolor: tab === 'plan' ? explorerTheme.accentMuted : 'transparent',
                      }}
                    >
                      <PlanIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="JSON">
                    <IconButton
                      size="small"
                      onClick={() => setTab('json')}
                      sx={{
                        color: tab === 'json' ? explorerTheme.accent : explorerTheme.textMuted,
                        bgcolor: tab === 'json' ? explorerTheme.accentMuted : 'transparent',
                      }}
                    >
                      <JsonIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </Stack>
                <Stack direction="row" spacing={1} alignItems="center">
                  {tab === 'results' && (
                    <>
                      <Typography
                        variant="caption"
                        sx={{ color: explorerTheme.textMuted, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.8 }}
                      >
                        View
                      </Typography>
                      <ToggleButtonGroup
                        value={viewMode}
                        exclusive
                        size="small"
                        onChange={(_, v) => v && setViewMode(v)}
                        sx={{
                          '& .MuiToggleButton-root': {
                            border: `1px solid ${explorerTheme.border}`,
                            color: explorerTheme.textMuted,
                            px: 1,
                            py: 0.25,
                            '&.Mui-selected': {
                              bgcolor: explorerTheme.accentMuted,
                              color: explorerTheme.accent,
                              fontWeight: 700,
                              '&:hover': { bgcolor: explorerTheme.accentHover },
                            },
                          },
                        }}
                      >
                        <ToggleButton value="table" aria-label="Table"><TableIcon fontSize="small" /></ToggleButton>
                        <ToggleButton value="bar" aria-label="Bar"><BarChartIcon fontSize="small" /></ToggleButton>
                        <ToggleButton value="line" aria-label="Line"><LineChartIcon fontSize="small" /></ToggleButton>
                        <ToggleButton value="area" aria-label="Area"><LineChartIcon fontSize="small" /></ToggleButton>
                        <ToggleButton value="pie" aria-label="Pie"><PieChartIcon fontSize="small" /></ToggleButton>
                        <ToggleButton value="scatter" aria-label="Scatter"><ScatterIcon fontSize="small" /></ToggleButton>
                        <ToggleButton value="kpi" aria-label="KPI"><KpiIcon fontSize="small" /></ToggleButton>
                      </ToggleButtonGroup>
                    </>
                  )}
                  <Button
                    size="small"
                    onClick={() => setSqlDialogOpen(true)}
                    sx={{ textTransform: 'none', color: explorerTheme.accent, fontWeight: 600, fontSize: '0.75rem' }}
                  >
                    View SQL
                  </Button>
                </Stack>
              </Paper>

              {/* Result content area */}
              <Box sx={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column', p: tab === 'results' ? 2 : 0 }}>
                {/* Automated Executive Insights Banner */}
                {tab === 'results' && insight && (
                  <AIInsightBadge
                    summaryText={insight.summaryText}
                    topDriver={insight.topDriver}
                    anomalies={insight.anomalies}
                    isCacheHit={isCacheHit}
                  />
                )}

                {/* Conversational Drilldown Breadcrumbs */}
                {tab === 'results' && (
                  <DrilldownBreadcrumbs
                    history={queryHistory}
                    currentQuery={unifiedQueryDef}
                    onNavigateBack={(historyIndex) => {
                      if (historyIndex < 0 || historyIndex >= queryHistory.length) return;
                      const historicalQuery = queryHistory[historyIndex];
                      setQueryHistory((prev) => prev.slice(0, historyIndex));
                      handleAIQueryChange(historicalQuery);
                      void submitPrompt(`I navigated back to the ${historicalQuery.title || 'previous'} view. Please summarize the data at this higher level.`);
                    }}
                  />
                )}

                {tab === 'results' && (
                  <SmartVisualizerPane
                    viewMode={viewMode as ChartViewMode}
                    results={result as any}
                    onDrillDown={(params) => {
                      // Snapshot query state into history stack before drilling
                      setQueryHistory((prev) => [...prev, unifiedQueryDef]);
                      setState((prev) => ({
                        ...prev,
                        filters: [
                          ...prev.filters,
                          {
                            fieldId: params.dimensionKey,
                            operator: 'equals' as any,
                            values: [params.dimensionValue],
                          },
                        ],
                      }));
                      void submitPrompt(
                        `I drilled down into ${params.dimensionValue} for ${params.dimensionKey}. Summarize the key drivers and suggest the next granular breakdown.`
                      );
                    }}
                  />
                )}

                {/* Root Cause Analysis (RCA) Investigation Panel */}
                {tab === 'results' && result && result.rows.length > 0 && (
                  <RCAInsightPanel
                    baseQuery={unifiedQueryDef}
                    targetMeasure={unifiedQueryDef.measures[0]?.fieldId}
                    catalog={semanticCatalog}
                  />
                )}

                {/* Quantitative Forecast Model Validation & Audit Trail */}
                {tab === 'results' && result && result.rows.length > 0 && unifiedQueryDef.timeDimensions.length > 0 && (
                  <ForecastAuditPanel
                    dimension={unifiedQueryDef.dimensions[0] || 'all'}
                    measure={unifiedQueryDef.measures[0]?.fieldId || 'valuation'}
                    forecastData={result.rows}
                  />
                )}
                {tab === 'sql' && (
                  <Box sx={{ flex: 1, overflow: 'auto', bgcolor: explorerTheme.backgroundElevated, color: explorerTheme.info, p: 3 }}>
                    <Box
                      component="pre"
                      sx={{ m: 0, fontFamily: 'monospace', fontSize: 13, whiteSpace: 'pre-wrap' }}
                    >
                      {result?.sql || '-- Select dimensions or measures to generate SQL.'}
                    </Box>
                  </Box>
                )}
                {tab === 'plan' && source && (
                  <ExplainPlanPane source={source} state={state} initialPlan={result?.plan} />
                )}
                {tab === 'json' && <JsonPreviewPane result={result} />}
              </Box>
            </Box>
          </>
        ) : (
          /* Redesigned Enterprise Landing Screen */
          <ExplorerLandingHero
            businessObjects={allBusinessObjects}
            savedQueries={savedRecords}
            onSelectBo={handlePickBo}
            onOpenSavedQuery={handleOpenSaved}
            onRunPrompt={handleRunHeroPrompt}
          />
        )}
      </Box>

      {source && (
        <FilterModal
          open={filterModalOpen}
          onClose={() => {
            setFilterModalOpen(false);
            setEditingFilterIndex(null);
          }}
          source={source}
          initialFieldId={undefined}
          initialFilter={initialFilterEditing ?? undefined}
          onApply={handleApplyFilter}
        />
      )}

      {showModelPicker && (
        <UnifiedBOPickerModal
          open
          context="query"
          onClose={handleClosePicker}
          onPick={(bo, bindingId, selectedRelatedBOs, _bindingDetails, selectedSubtypeKey) => {
            void (async () => {
              setSourceLoadError(null);
              try {
                const loaded = await loadExplorerSource(bo.id, bindingId || bo.defaultBindingId, selectedSubtypeKey);
                if (selectedRelatedBOs && selectedRelatedBOs.length > 0 && loaded.relatedBOs) {
                  loaded.relatedBOs = loaded.relatedBOs.filter((r) => selectedRelatedBOs.includes(r.boName));
                }
                loaded.selectedSubtypeKey = selectedSubtypeKey ?? null;
                const displayNameWithSubtype = selectedSubtypeKey && loaded.subtypes && loaded.subtypes[selectedSubtypeKey]
                  ? `${loaded.displayName} (${loaded.subtypes[selectedSubtypeKey].displayName})`
                  : loaded.displayName;
                setSource(loaded);
                setState(emptyExplorerState(loaded.id, loaded.bindingId));
                setQueryName(`${displayNameWithSubtype} Exploration`);
                setShowModelPicker(false);
              } catch (err) {
                const message = err instanceof Error ? err.message : 'Failed to load BO.';
                setSourceLoadError(message);
                devError('loadExplorerSource failed', err);
              }
            })();
          }}
          savedQueries={savedRecords}
          savedLoading={savedLoading}
          onOpenSaved={handleOpenSaved}
          onDeleteSaved={(id) => {
            void handleDeleteSaved(id);
          }}
        />
      )}

      <Dialog
        open={saveDialogOpen}
        onClose={() => setSaveDialogOpen(false)}
        maxWidth="xs"
        fullWidth
      >
        <DialogTitle sx={{ fontWeight: 800, fontSize: '0.92rem' }}>Save Exploration Query</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            fullWidth
            label="Query Name"
            value={saveName}
            onChange={(e) => setSaveName(e.target.value)}
            sx={{ mt: 1 }}
          />
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setSaveDialogOpen(false)} sx={{ textTransform: 'none', color: explorerTheme.textMuted }}>
            Cancel
          </Button>
          <Button
            onClick={handleConfirmSave}
            variant="contained"
            disabled={!saveName.trim()}
            sx={{
              bgcolor: explorerTheme.accent,
              color: '#FFF',
              textTransform: 'none',
              fontWeight: 700,
              '&:hover': { bgcolor: explorerTheme.accentDark },
            }}
          >
            Save
          </Button>
        </DialogActions>
      </Dialog>

      <SqlPreviewDialog
        open={sqlDialogOpen}
        onClose={() => setSqlDialogOpen(false)}
        sql={result?.sql || ''}
        warningCount={result?.warnings?.length}
      />

      <Dialog
        open={reviewQueueOpen}
        onClose={() => setReviewQueueOpen(false)}
        maxWidth="md"
        fullWidth
        PaperProps={{
          sx: { bgcolor: explorerTheme.backgroundElevated, borderRadius: 3, border: `1px solid ${explorerTheme.border}` },
        }}
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', color: explorerTheme.text, pb: 1 }}>
          <Typography variant="h6" fontWeight={800} sx={{ color: explorerTheme.text }}>
            Semantic OS Knowledge Review Desk
          </Typography>
          <Button onClick={() => setReviewQueueOpen(false)} sx={{ color: explorerTheme.textMuted, textTransform: 'none' }}>
            Close
          </Button>
        </DialogTitle>
        <DialogContent sx={{ p: 2 }}>
          <SemanticReviewQueue />
        </DialogContent>
      </Dialog>

      {source && (
        <CalculationModal
          open={calculationModalOpen}
          onClose={() => setCalculationModalOpen(false)}
          availableFields={source.fields}
          onSave={handleSaveCalculation}
        />
      )}

      <ScheduleQueryModal
        open={scheduleModalOpen}
        onClose={() => setScheduleModalOpen(false)}
        queryName={queryName}
        onSaveSchedule={handleSaveSchedule}
      />

      <ShareQueryModal
        open={shareModalOpen}
        onClose={() => setShareModalOpen(false)}
        queryName={queryName}
        onShare={handleShareQuery}
      />

      <Drawer
        anchor="right"
        open={timelineDrawerOpen}
        onClose={() => setTimelineDrawerOpen(false)}
        PaperProps={{
          sx: {
            width: 380,
            bgcolor: explorerTheme.backgroundElevated,
            borderLeft: `1px solid ${explorerTheme.border}`,
          },
        }}
      >
        <Box
          sx={{
            p: 2,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: `1px solid ${explorerTheme.border}`,
          }}
        >
          <Typography variant="subtitle2" sx={{ color: explorerTheme.text, fontWeight: 800 }}>
            Query Event Log & Mutations
          </Typography>
          <Button
            size="small"
            onClick={() => setTimelineDrawerOpen(false)}
            sx={{ color: explorerTheme.textMuted, textTransform: 'none', fontSize: '0.75rem' }}
          >
            Close
          </Button>
        </Box>
        <Box sx={{ flexGrow: 1, overflowY: 'auto' }}>
          <QueryMutationTimeline messages={aiMessages} />
        </Box>
      </Drawer>
    </Box>
  );
};

export default DataExplorerPage;
