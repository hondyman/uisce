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
  Tooltip,
  ToggleButtonGroup,
  ToggleButton,
  Menu,
  MenuItem,
  useTheme,
  Chip,
  Tabs,
  Tab,
} from '@mui/material';
import {
  PlayArrow as PlayIcon,
  Save as SaveIcon,
  Download as DownloadIcon,
  Edit as EditIcon,
  Storage as StorageIcon,
  Code as CodeIcon,
  TableChart as TableIcon,
  BarChart as BarChartIcon,
  ShowChart as LineChartIcon,
  PieChart as PieChartIcon,
  ArrowBack as BackIcon,
  Schedule as ScheduleIcon,
  Lock as LockIcon,
  DesignServices as DesignIcon,
  Assessment as ResultsIcon,
} from '@mui/icons-material';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import {
  createNewQueryTab,
  emptyExplorerState,
} from '../features/data-explorer/types/dataExplorerTypes';
import { useExplorerTheme } from '../features/data-explorer/hooks/useExplorerTheme';
import type {
  BusinessObjectSummary,
  ExplorerQueryState,
  ExplorerSource,
  FilterSelection,
  SortSelection,
  ViewMode,
  QueryTabState,
  QueryParameter,
  ScheduleConfig,
} from '../features/data-explorer/types/dataExplorerTypes';
import {
  loadExplorerSource,
  fetchBusinessObjects,
} from '../features/data-explorer/services/dataExplorerApi';
import { useDataExplorerQuery } from '../features/data-explorer/hooks/useDataExplorerQuery';
import { useSavedExplorerQueries } from '../features/data-explorer/hooks/useSavedExplorerQueries';
import BOFieldsPalette, { BOField } from '../components/reporting/BOFieldsPalette';
import FilterBuilderPanel, { FilterGroup, buildSQL } from '../components/reporting/FilterBuilderPanel';
import { ReportScheduleBurstingTab } from '../components/reporting/ReportScheduleBurstingTab';
import ParametersDialog from '../components/reporting/ParametersDialog';
import { QueryDefinitionBar } from '../features/data-explorer/components/QueryDefinitionBar';
import { FilterModal } from '../features/data-explorer/components/FilterModal';
import { FilterPillBar } from '../features/data-explorer/components/FilterPillBar';
import { ResultsTablePane } from '../features/data-explorer/components/ResultsTablePane';
import { UnifiedBOPickerModal } from '../components/common/UnifiedBOPickerModal';
import { VisualizationPane } from '../features/data-explorer/components/VisualizationPane';
import { SqlPreviewDialog } from '../features/data-explorer/components/SqlPreviewDialog';
import { QueryTabManager } from '../features/data-explorer/components/QueryTabManager';
import { ParametersToolbar } from '../features/data-explorer/components/ParametersToolbar';
import { ScheduleQueryModal } from '../features/data-explorer/components/ScheduleQueryModal';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import TuneIcon from '@mui/icons-material/Tune';
import { devError } from '../utils/devLogger';

const QUERY_NAME_DEFAULT = 'Untitled Query';

export const DataQueryBuilderPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const params = useParams<{ queryId?: string }>();
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const explorerTheme = useExplorerTheme();

  // Multi-Tab Query Workspace
  const [tabs, setTabs] = useState<QueryTabState[]>([
    createNewQueryTab('tab-1', 'Query 1', null),
  ]);
  const [activeTabId, setActiveTabId] = useState<string>('tab-1');

  // Main UI Mode: 'design' | 'results' | 'filters' | 'schedule' | 'parameters'
  const [mainTab, setMainTab] = useState<'design' | 'results' | 'filters' | 'schedule' | 'parameters'>('design');

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
  const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
  const [parametersDialogOpen, setParametersDialogOpen] = useState(false);
  const [sqlDialogOpen, setSqlDialogOpen] = useState(false);
  const [exportAnchor, setExportAnchor] = useState<null | HTMLElement>(null);
  const [reportFilterGroups, setReportFilterGroups] = useState<FilterGroup[]>([]);

  // Sync state.filters into reportFilterGroups and vice-versa
  useEffect(() => {
    if (state.filters && state.filters.length > 0 && reportFilterGroups.length === 0) {
      const conditions = state.filters.map((f, idx) => ({
        id: `cond_${idx}_${Date.now()}`,
        field: f.fieldId,
        fieldLabel: source?.fields.find((field) => field.id === f.fieldId || field.name === f.fieldId)?.displayName || f.fieldId,
        dataType: source?.fields.find((field) => field.id === f.fieldId || field.name === f.fieldId)?.type || 'string',
        operator: f.operator,
        value: f.values[0] || '',
        values: f.values,
        enabled: true,
      }));
      setReportFilterGroups([{
        id: `group_main_${Date.now()}`,
        combinator: 'AND',
        category: 'WHERE',
        conditions,
      }]);
    }
  }, [state.filters, source]);

  const handleFilterGroupsChange = useCallback((groups: FilterGroup[]) => {
    setReportFilterGroups(groups);
    const newFilters: FilterSelection[] = [];
    groups.forEach((g) => {
      g.conditions.forEach((c) => {
        if (c.enabled && c.field) {
          newFilters.push({
            fieldId: c.field,
            operator: (c.operator as any) || 'equals',
            values: c.values && c.values.length > 0 ? c.values : (c.value ? [c.value] : []),
          });
        }
      });
    });
    setState((prev) => ({ ...prev, filters: newFilters }));
  }, [setState]);

  // Multi-Tab actions
  const handleAddTab = useCallback(() => {
    setShowModelPicker(true);
  }, []);

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

  // Auto-open BO picker if no Business Object is loaded yet
  useEffect(() => {
    const search = new URLSearchParams(location.search);
    const boId = search.get('bo');
    const locState = location.state as { query?: any } | undefined;
    if (!source && !boId && !locState?.query) {
      setShowModelPicker(true);
    }
  }, [location.search, location.state, source]);

  // Load catalog list
  useEffect(() => {
    fetchBusinessObjects()
      .then((bos) => setAllBusinessObjects(bos))
      .catch((err) => devError('fetchBusinessObjects failed', err));
  }, []);

  // Check state passed from navigation / query library
  useEffect(() => {
    const locState = location.state as { query?: any } | undefined;
    if (locState?.query) {
      const q = locState.query;
      setQueryName(q.name);
      if (q.sourceId) {
        loadExplorerSource(q.sourceId, q.queryState?.bindingId)
          .then((loaded) => {
            setSource(loaded);
            if (q.queryState) {
              setState({
                ...q.queryState,
                sourceId: loaded.id,
                bindingId: loaded.bindingId,
              });
            }
          })
          .catch(devError);
      }
    }
  }, [location.state]);

  const { result, isLoading, isPreviewing, error, run, lastRunAt } = useDataExplorerQuery({
    source,
    state,
  });

  // Execute query and auto-switch to Results tab
  const handleRunQuery = useCallback(async () => {
    setMainTab('results');
    await run();
  }, [run]);

  // Deep links /data-explorer/builder?bo=<id>&binding=<id>
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
        setSourceLoadError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      cancelled = true;
    };
  }, [location.search]);

  // Dimension / Measure / Filter mutators
  const handleToggleDimension = useCallback(
    (fieldId: string) => {
      setState((prev) => {
        const exists = prev.dimensions.some((d) => d.fieldId === fieldId);
        const nextDims = exists
          ? prev.dimensions.filter((d) => d.fieldId !== fieldId)
          : [...prev.dimensions, { fieldId }];
        return { ...prev, dimensions: nextDims };
      });
    },
    [setState]
  );

  const handleToggleMeasure = useCallback(
    (fieldId: string, agg?: import('../features/data-explorer/types/dataExplorerTypes').AggFn) => {
      setState((prev) => {
        const exists = prev.measures.some((m) => m.fieldId === fieldId);
        const nextMeasures = exists
          ? prev.measures.filter((m) => m.fieldId !== fieldId)
          : [...prev.measures, { fieldId, agg: agg || 'SUM' }];
        return { ...prev, measures: nextMeasures };
      });
    },
    [setState]
  );

  const handleAddTimeDimension = useCallback(
    (fieldId: string) => {
      setState((prev) => {
        const exists = prev.timeDimensions.some((t) => t.fieldId === fieldId);
        const nextTime = exists
          ? prev.timeDimensions.filter((t) => t.fieldId !== fieldId)
          : [...prev.timeDimensions, { fieldId, granularity: 'month' }];
        return { ...prev, timeDimensions: nextTime };
      });
    },
    [setState]
  );

  const handleRemoveDimension = useCallback(
    (fieldId: string) => {
      setState((prev) => ({
        ...prev,
        dimensions: prev.dimensions.filter((d) => d.fieldId !== fieldId),
      }));
    },
    [setState]
  );

  const handleRemoveMeasure = useCallback(
    (fieldId: string) => {
      setState((prev) => ({
        ...prev,
        measures: prev.measures.filter((m) => m.fieldId !== fieldId),
      }));
    },
    [setState]
  );

  const handleRemoveTimeDimension = useCallback(
    (fieldId: string) => {
      setState((prev) => ({
        ...prev,
        timeDimensions: prev.timeDimensions.filter((t) => t.fieldId !== fieldId),
      }));
    },
    [setState]
  );

  const handleUpdateMeasureAgg = useCallback(
    (fieldId: string, agg: import('../features/data-explorer/types/dataExplorerTypes').AggFn) => {
      setState((prev) => ({
        ...prev,
        measures: prev.measures.map((m) => (m.fieldId === fieldId ? { ...m, agg } : m)),
      }));
    },
    [setState]
  );

  const handleToggleSort = useCallback(
    (fieldId: string) => {
      setState((prev) => {
        const existing = prev.sorts.find((s) => s.fieldId === fieldId);
        let nextSorts: SortSelection[];
        if (!existing) {
          nextSorts = [...prev.sorts, { fieldId, direction: 'asc' }];
        } else if (existing.direction === 'asc') {
          nextSorts = prev.sorts.map((s) => (s.fieldId === fieldId ? { ...s, direction: 'desc' } : s));
        } else {
          nextSorts = prev.sorts.filter((s) => s.fieldId !== fieldId);
        }
        return { ...prev, sorts: nextSorts };
      });
    },
    [setState]
  );

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

  const handleLimitChange = useCallback(
    (limit: number) => {
      setState((prev) => ({ ...prev, limit }));
    },
    [setState]
  );

  const handleAddFilter = useCallback(
    (filter: FilterSelection) => {
      setState((prev) => {
        if (editingFilterIndex !== null) {
          const updated = [...prev.filters];
          updated[editingFilterIndex] = filter;
          return { ...prev, filters: updated };
        }
        return { ...prev, filters: [...prev.filters, filter] };
      });
      setEditingFilterIndex(null);
    },
    [editingFilterIndex, setState]
  );

  const handleRemoveFilter = useCallback(
    (index: number) => {
      setState((prev) => ({
        ...prev,
        filters: prev.filters.filter((_, i) => i !== index),
      }));
    },
    [setState]
  );

  const [filterModalFieldId, setFilterModalFieldId] = useState<string | undefined>(undefined);

  const handleOpenEditFilter = useCallback((index: number) => {
    setEditingFilterIndex(index);
    setFilterModalFieldId(undefined);
    setFilterModalOpen(true);
  }, []);

  const handleOpenNewFilter = useCallback((fieldId?: string) => {
    setEditingFilterIndex(null);
    setFilterModalFieldId(fieldId);
    setFilterModalOpen(true);
  }, []);

  // Export handlers
  const handleExportCSV = () => {
    if (!result || !result.rows || result.rows.length === 0) return;
    const cols = result.columns.map((c) => c.name);
    const csvContent = [
      cols.join(','),
      ...result.rows.map((row) =>
        cols.map((col) => JSON.stringify(row[col] ?? '')).join(',')
      ),
    ].join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.setAttribute('href', url);
    link.setAttribute('download', `${queryName.toLowerCase().replace(/\s+/g, '_')}_data.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    setExportAnchor(null);
  };

  const handleExportJSON = () => {
    if (!result || !result.rows) return;
    const blob = new Blob([JSON.stringify(result.rows, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.setAttribute('href', url);
    link.setAttribute('download', `${queryName.toLowerCase().replace(/\s+/g, '_')}_data.json`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    setExportAnchor(null);
  };

  const handleSelectModel = useCallback(
    async (
      bo: BusinessObjectSummary,
      bindingId?: string,
      selectedRelatedBOs?: string[],
      _bindingDetails?: any,
      selectedSubtypeKey?: string | null
    ) => {
      try {
        setSourceLoadError(null);
        const loaded = await loadExplorerSource(bo.id, bindingId || bo.defaultBindingId, selectedSubtypeKey);

        if (selectedRelatedBOs && selectedRelatedBOs.length > 0 && loaded.relatedBOs) {
          loaded.relatedBOs = loaded.relatedBOs.filter((r) => selectedRelatedBOs.includes(r.boName));
        }

        loaded.selectedSubtypeKey = selectedSubtypeKey ?? null;

        const displayNameWithSubtype = selectedSubtypeKey && loaded.subtypes && loaded.subtypes[selectedSubtypeKey]
          ? `${loaded.displayName} (${loaded.subtypes[selectedSubtypeKey].displayName})`
          : loaded.displayName;

        if (!source) {
          setSource(loaded);
          setState(emptyExplorerState(loaded.id, loaded.bindingId));
          setQueryName(`${displayNameWithSubtype} Query`);
        } else {
          const nextNum = tabs.length + 1;
          const newTab = createNewQueryTab(undefined, `${displayNameWithSubtype} Query ${nextNum}`, loaded);
          setTabs((prev) => [...prev, newTab]);
          setActiveTabId(newTab.id);
        }
        setShowModelPicker(false);
      } catch (err) {
        setSourceLoadError(err instanceof Error ? err.message : String(err));
      }
    },
    [source, tabs.length, setQueryName, setSource, setState]
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 64px)', bgcolor: 'background.default' }}>
      {/* Top Header Bar */}
      <Box
        sx={{
          px: 2.5,
          py: 1.2,
          borderBottom: 1,
          borderColor: 'divider',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          bgcolor: 'background.paper',
        }}
      >
        <Stack direction="row" spacing={1.5} alignItems="center">
          <Tooltip title="Back to Query Library">
            <IconButton size="small" onClick={() => navigate('/data-explorer')}>
              <BackIcon />
            </IconButton>
          </Tooltip>

          <Box>
            {isEditingName ? (
              <TextField
                size="small"
                value={queryName}
                onChange={(e) => setQueryName(e.target.value)}
                onBlur={() => setIsEditingName(false)}
                onKeyDown={(e) => e.key === 'Enter' && setIsEditingName(false)}
                autoFocus
                sx={{ width: 260 }}
              />
            ) : (
              <Stack direction="row" spacing={1} alignItems="center">
                <Typography variant="h6" sx={{ fontWeight: 700, fontSize: '1.05rem' }}>
                  {queryName || QUERY_NAME_DEFAULT}
                </Typography>
                <IconButton size="small" onClick={() => setIsEditingName(true)}>
                  <EditIcon fontSize="small" sx={{ fontSize: 15 }} />
                </IconButton>
              </Stack>
            )}

            {/* Immutable BO & Binding badge */}
            <Stack direction="row" spacing={1} alignItems="center">
              {source ? (
                <Tooltip title="Business Object and Datasource Binding are immutable for this query tab">
                  <Chip
                    icon={<LockIcon sx={{ fontSize: '12px !important' }} />}
                    label={`${source.displayName} • ${source.bindingId || 'Default Binding'}`}
                    size="small"
                    sx={{
                      height: 20,
                      fontSize: '0.7rem',
                      fontWeight: 600,
                      bgcolor: isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)',
                    }}
                  />
                </Tooltip>
              ) : (
                <Typography variant="caption" color="text.secondary">
                  Select a Business Object to begin
                </Typography>
              )}
            </Stack>
          </Box>
        </Stack>

        {/* View Mode Switching Tabs (Design vs Results vs Filters vs Parameters vs Schedule) */}
        {source && (
          <Tabs
            value={mainTab}
            onChange={(_, val) => setMainTab(val)}
            sx={{
              minHeight: 36,
              bgcolor: isDark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.04)',
              borderRadius: 2,
              p: 0.3,
              '& .MuiTab-root': {
                minHeight: 32,
                py: 0.5,
                px: 2,
                fontSize: '0.8rem',
                fontWeight: 700,
                textTransform: 'none',
                borderRadius: 1.5,
              },
              '& .Mui-selected': {
                bgcolor: 'background.paper',
                boxShadow: 1,
              },
            }}
          >
            <Tab icon={<DesignIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="Design Palette" value="design" />
            <Tab
              icon={<ResultsIcon sx={{ fontSize: 16 }} />}
              iconPosition="start"
              label={`Results ${result ? `(${result.rowCount})` : ''}`}
              value="results"
            />
            <Tab icon={<FilterAltIcon sx={{ fontSize: 16 }} />} iconPosition="start" label={`Filters (${state.filters.length})`} value="filters" />
            <Tab icon={<TuneIcon sx={{ fontSize: 16 }} />} iconPosition="start" label={`Parameters (${(state.parameters || []).length})`} value="parameters" />
            <Tab icon={<ScheduleIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="Schedule & Bursting" value="schedule" />
          </Tabs>
        )}

        {/* Action Buttons */}
        <Stack direction="row" spacing={1.5} alignItems="center">
          {!source && (
            <Button
              variant="contained"
              size="small"
              onClick={() => setShowModelPicker(true)}
              sx={{ textTransform: 'none', borderRadius: 2 }}
            >
              Select Business Object
            </Button>
          )}

          <Button
            variant="outlined"
            size="small"
            startIcon={<ScheduleIcon />}
            onClick={() => setScheduleModalOpen(true)}
            sx={{ textTransform: 'none', borderRadius: 2 }}
          >
            Schedule Workbook
          </Button>

          <Button
            variant="outlined"
            size="small"
            startIcon={<CodeIcon />}
            onClick={() => setSqlDialogOpen(true)}
            disabled={!result?.sql}
            sx={{ textTransform: 'none', borderRadius: 2 }}
          >
            Preview SQL
          </Button>

          <Button
            variant="outlined"
            size="small"
            startIcon={<DownloadIcon />}
            onClick={(e) => setExportAnchor(e.currentTarget)}
            disabled={!result?.rows || result.rows.length === 0}
            sx={{ textTransform: 'none', borderRadius: 2 }}
          >
            Export Data
          </Button>

          <Menu
            anchorEl={exportAnchor}
            open={Boolean(exportAnchor)}
            onClose={() => setExportAnchor(null)}
          >
            <MenuItem onClick={handleExportCSV}>Export as CSV</MenuItem>
            <MenuItem onClick={handleExportJSON}>Export as JSON</MenuItem>
          </Menu>

          <Button
            variant="contained"
            color="primary"
            startIcon={isLoading ? <CircularProgress size={16} color="inherit" /> : <PlayIcon />}
            onClick={handleRunQuery}
            disabled={isLoading || !source || (state.dimensions.length === 0 && state.measures.length === 0)}
            sx={{ textTransform: 'none', fontWeight: 700, borderRadius: 2, px: 2.5 }}
          >
            {isLoading ? 'Running...' : 'Run Query'}
          </Button>
        </Stack>
      </Box>

      {/* Multi-Tab Tab Manager */}
      <QueryTabManager
        tabs={tabs}
        activeTabId={activeTabId}
        onSelectTab={setActiveTabId}
        onAddTab={handleAddTab}
        onCloseTab={handleCloseTab}
        onRenameTab={handleRenameTab}
        onDuplicateTab={handleDuplicateTab}
      />

      {/* Global Workbook Parameters Toolbar */}
      <ParametersToolbar
        parameters={state.parameters || []}
        onAddParameter={(p) => {
          setTabs((prev) =>
            prev.map((t) => ({
              ...t,
              queryState: {
                ...t.queryState,
                parameters: [...(t.queryState.parameters || []), p],
              },
            }))
          );
        }}
        onUpdateParameter={(p) => {
          setTabs((prev) =>
            prev.map((t) => ({
              ...t,
              queryState: {
                ...t.queryState,
                parameters: (t.queryState.parameters || []).map((x) => (x.id === p.id ? p : x)),
              },
            }))
          );
        }}
        onRemoveParameter={(id) => {
          setTabs((prev) =>
            prev.map((t) => ({
              ...t,
              queryState: {
                ...t.queryState,
                parameters: (t.queryState.parameters || []).filter((x) => x.id !== id),
              },
            }))
          );
        }}
        onChangeParamValue={(id, val) => {
          setTabs((prev) =>
            prev.map((t) => ({
              ...t,
              queryState: {
                ...t.queryState,
                parameters: (t.queryState.parameters || []).map((x) =>
                  x.id === id ? { ...x, currentValue: val } : x
                ),
              },
            }))
          );
        }}
      />

      {/* Main Builder Workspace */}
      <Box
        sx={{ flex: 1, display: 'flex', overflow: 'hidden' }}
        onDragOver={(e) => {
          if (e.dataTransfer.types.includes('application/json')) {
            e.preventDefault();
            e.dataTransfer.dropEffect = 'copy';
          }
        }}
        onDrop={(e) => {
          const rawData = e.dataTransfer.getData('application/json');
          if (rawData) {
            try {
              const parsed = JSON.parse(rawData);
              if (parsed.type === 'bofield' || parsed.type === 'bofield_batch') {
                e.preventDefault();
                const fields: BOField[] = parsed.type === 'bofield_batch' ? parsed.fields : [parsed.field];
                fields.forEach((f) => {
                  const normType = (f.dataType || f.type || 'string').toLowerCase();
                  if (['number', 'int', 'float', 'double', 'decimal', 'numeric', 'currency', 'money'].some((k) => normType.includes(k))) {
                    handleToggleMeasure(f.name);
                  } else if (['date', 'time', 'timestamp', 'datetime'].some((k) => normType.includes(k))) {
                    handleAddTimeDimension(f.name);
                  } else {
                    handleToggleDimension(f.name);
                  }
                });
              }
            } catch (err) {
              devError('Failed to parse dropped field', err);
            }
          }
        }}
      >
        {/* Left BO Fields Palette */}
        {source ? (
          <Box sx={{ width: 320, borderRight: 1, borderColor: 'divider', bgcolor: 'background.paper', overflowY: 'auto' }}>
            <BOFieldsPalette
              selectedBO={source.rawBO || {
                name: source.id,
                displayName: source.displayName,
                fields: source.fields,
                subtypes: source.subtypes,
                selectedSubtypeKey: source.selectedSubtypeKey,
              }}
              selectedSubtypeKey={source.selectedSubtypeKey}
              relatedBOs={source.relatedBOs}
              onAddFieldToCanvas={(field) => {
                const normType = (field.dataType || field.type || 'string').toLowerCase();
                if (['number', 'int', 'float', 'double', 'decimal', 'numeric', 'currency', 'money'].some((k) => normType.includes(k))) {
                  handleToggleMeasure(field.name);
                } else if (['date', 'time', 'timestamp', 'datetime'].some((k) => normType.includes(k))) {
                  handleAddTimeDimension(field.name);
                } else {
                  handleToggleDimension(field.name);
                }
              }}
              onAddAllAsTable={(fields) => {
                fields.forEach((field) => {
                  const normType = (field.dataType || field.type || 'string').toLowerCase();
                  if (['number', 'int', 'float', 'double', 'decimal', 'numeric', 'currency', 'money'].some((k) => normType.includes(k))) {
                    handleToggleMeasure(field.name);
                  } else if (['date', 'time', 'timestamp', 'datetime'].some((k) => normType.includes(k))) {
                    handleAddTimeDimension(field.name);
                  } else {
                    handleToggleDimension(field.name);
                  }
                });
              }}
            />
          </Box>
        ) : (
          <Box
            sx={{
              width: 320,
              borderRight: 1,
              borderColor: 'divider',
              p: 3,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              textAlign: 'center',
              bgcolor: 'background.paper',
            }}
          >
            <StorageIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
            <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 2 }}>
              No Business Object Selected
            </Typography>
            <Button
              variant="contained"
              size="small"
              onClick={() => setShowModelPicker(true)}
              sx={{ textTransform: 'none', borderRadius: 2 }}
            >
              Choose Business Object
            </Button>
          </Box>
        )}

        {/* Center / Right Results & Playground Canvas */}
        <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          {source && mainTab === 'design' && (
            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'auto', p: 2 }}>
              {/* Query Definition Bar (Cube Shelves) */}
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
                onOpenFilterModal={() => setMainTab('filters')}
                onOpenParameterModal={() => setMainTab('parameters')}
              />

              {/* Filter Section */}
              <Box sx={{ mt: 2 }}>
                <FilterPillBar
                  source={source}
                  filters={state.filters}
                  onAddFilter={() => setMainTab('filters')}
                  onRemoveFilter={handleRemoveFilter}
                  onEditFilter={() => setMainTab('filters')}
                  onDropField={(fieldKey) => {
                    const f = source.fields.find(
                      (field) => field.id === fieldKey || field.technicalName === fieldKey || field.name === fieldKey
                    );
                    const newCond = {
                      id: `cond_${Date.now()}`,
                      field: f?.name || fieldKey,
                      fieldLabel: f?.displayName || f?.name || fieldKey,
                      dataType: f?.type || 'string',
                      operator: 'equals',
                      value: '',
                      values: [],
                      enabled: true,
                    };
                    setReportFilterGroups((prev) => {
                      if (prev.length === 0) {
                        return [{
                          id: `group_main_${Date.now()}`,
                          combinator: 'AND',
                          category: 'WHERE',
                          conditions: [newCond],
                        }];
                      }
                      return prev.map((g, idx) =>
                        idx === 0 ? { ...g, conditions: [...g.conditions, newCond] } : g
                      );
                    });
                    setState((prev) => ({
                      ...prev,
                      filters: [...prev.filters, { fieldId: f?.name || fieldKey, operator: 'equals', values: [''] }],
                    }));
                    setMainTab('filters');
                  }}
                />
              </Box>

              {/* Visual Query Canvas / Design Palette Board (like Report Builder) */}
              <Paper
                variant="outlined"
                sx={{
                  mt: 2.5,
                  borderRadius: 3,
                  border: `1.5px solid ${theme.palette.divider}`,
                  bgcolor: 'background.paper',
                  overflow: 'hidden',
                }}
              >
                <Box
                  sx={{
                    p: 2,
                    borderBottom: 1,
                    borderColor: 'divider',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    bgcolor: isDark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.02)',
                  }}
                >
                  <Stack direction="row" spacing={1.5} alignItems="center">
                    <TableIcon sx={{ color: 'primary.main', fontSize: 20 }} />
                    <Box>
                      <Typography variant="subtitle2" fontWeight={700}>
                        Query Design Canvas • {source.displayName}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {state.dimensions.length} Dimensions • {state.measures.length} Measures • {state.filters.length} Filters mapped
                      </Typography>
                    </Box>
                  </Stack>
                  <Button
                    variant="contained"
                    color="primary"
                    size="small"
                    startIcon={isLoading ? <CircularProgress size={14} color="inherit" /> : <PlayIcon />}
                    onClick={handleRunQuery}
                    disabled={isLoading || (state.dimensions.length === 0 && state.measures.length === 0)}
                    sx={{ textTransform: 'none', fontWeight: 700, borderRadius: 2 }}
                  >
                    {isLoading ? 'Running…' : 'Run Query & View Results'}
                  </Button>
                </Box>

                {/* Grid Design Table */}
                <Box sx={{ p: 2.5, bgcolor: isDark ? '#0F172A' : '#FAFAFA', minHeight: 180 }}>
                  <Typography variant="caption" sx={{ fontWeight: 700, color: 'text.secondary', textTransform: 'uppercase', letterSpacing: 0.5, mb: 1.5, display: 'block' }}>
                    Assigned Query Columns (Drag or click fields on left to add)
                  </Typography>

                  {state.dimensions.length === 0 && state.measures.length === 0 && state.timeDimensions.length === 0 ? (
                    <Paper
                      elevation={0}
                      sx={{
                        p: 4,
                        textAlign: 'center',
                        borderRadius: 2,
                        border: `2px dashed ${theme.palette.divider}`,
                        bgcolor: 'background.paper',
                      }}
                    >
                      <Typography variant="body2" color="text.secondary" fontWeight={600}>
                        Drop fields here from the left palette to construct your query columns.
                      </Typography>
                    </Paper>
                  ) : (
                    <Box sx={{ overflowX: 'auto' }}>
                      <Box sx={{ display: 'inline-flex', gap: 1.5, minWidth: '100%', pb: 1 }}>
                        {state.dimensions.map((dim) => {
                          const f = source.fields.find(
                            field => field.id === dim.fieldId
                              || field.technicalName === dim.fieldId
                              || field.name === dim.fieldId
                          );
                          return (
                            <Paper
                              key={`dim-${dim.fieldId}`}
                              sx={{
                                p: 1.5,
                                minWidth: 160,
                                borderRadius: 2,
                                border: '1px solid',
                                borderColor: 'primary.main',
                                bgcolor: 'background.paper',
                              }}
                            >
                              <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 0.5 }}>
                                <Chip label="Dimension" size="small" color="primary" sx={{ height: 18, fontSize: 10, fontWeight: 700 }} />
                                <IconButton size="small" onClick={() => handleRemoveDimension(dim.fieldId)} sx={{ p: 0.2 }}>
                                  <EditIcon sx={{ fontSize: 12 }} />
                                </IconButton>
                              </Stack>
                              <Typography variant="body2" fontWeight={700} noWrap>
                                {f?.displayName || dim.fieldId}
                              </Typography>
                              <Typography variant="caption" color="text.secondary" noWrap display="block">
                                Column: {f?.technicalName || f?.name || dim.fieldId}
                              </Typography>
                            </Paper>
                          );
                        })}

                        {state.timeDimensions.map((time) => {
                          const f = source.fields.find(
                            field => field.id === time.fieldId
                              || field.technicalName === time.fieldId
                              || field.name === time.fieldId
                          );
                          return (
                            <Paper
                              key={`time-${time.fieldId}`}
                              sx={{
                                p: 1.5,
                                minWidth: 160,
                                borderRadius: 2,
                                border: '1px solid',
                                borderColor: 'warning.main',
                                bgcolor: 'background.paper',
                              }}
                            >
                              <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 0.5 }}>
                                <Chip label={`Time (${time.granularity || 'month'})`} size="small" color="warning" sx={{ height: 18, fontSize: 10, fontWeight: 700 }} />
                                <IconButton size="small" onClick={() => handleRemoveTimeDimension(time.fieldId)} sx={{ p: 0.2 }}>
                                  <EditIcon sx={{ fontSize: 12 }} />
                                </IconButton>
                              </Stack>
                              <Typography variant="body2" fontWeight={700} noWrap>
                                {f?.displayName || time.fieldId}
                              </Typography>
                              <Typography variant="caption" color="text.secondary" noWrap display="block">
                                Granularity: {time.granularity || 'month'}
                              </Typography>
                            </Paper>
                          );
                        })}

                        {state.measures.map((m) => {
                          const f = source.fields.find(
                            field => field.id === m.fieldId
                              || field.technicalName === m.fieldId
                              || field.name === m.fieldId
                          );
                          return (
                            <Paper
                              key={`meas-${m.fieldId}`}
                              sx={{
                                p: 1.5,
                                minWidth: 160,
                                borderRadius: 2,
                                border: '1px solid',
                                borderColor: 'success.main',
                                bgcolor: 'background.paper',
                              }}
                            >
                              <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 0.5 }}>
                                <Chip label={`Measure • ${m.agg}`} size="small" color="success" sx={{ height: 18, fontSize: 10, fontWeight: 700 }} />
                                <IconButton size="small" onClick={() => handleRemoveMeasure(m.fieldId)} sx={{ p: 0.2 }}>
                                  <EditIcon sx={{ fontSize: 12 }} />
                                </IconButton>
                              </Stack>
                              <Typography variant="body2" fontWeight={700} noWrap>
                                {f?.displayName || m.fieldId}
                              </Typography>
                              <Typography variant="caption" color="text.secondary" noWrap display="block">
                                Agg: {m.agg}({f?.technicalName || f?.name || m.fieldId})
                              </Typography>
                            </Paper>
                          );
                        })}
                      </Box>
                    </Box>
                  )}
                </Box>
              </Paper>
            </Box>
          )}

          {source && mainTab === 'filters' && (
            <Box sx={{ flex: 1, display: 'flex', overflow: 'hidden', p: 1 }}>
              <FilterBuilderPanel
                selectedBO={source.rawBO || {
                  id: source.id,
                  name: source.id,
                  displayName: source.displayName,
                  fields: source.fields,
                  subtypes: source.subtypes,
                }}
                parameters={state.parameters || []}
                onChange={handleFilterGroupsChange}
                onParametersChange={(newParams) => {
                  setState((prev) => ({ ...prev, parameters: newParams }));
                }}
              />
            </Box>
          )}

          {source && mainTab === 'parameters' && (
            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'auto', p: 3 }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
                <Typography variant="h6" fontWeight={700}>
                  Workbook Parameters
                </Typography>
                <Button
                  variant="contained"
                  size="small"
                  startIcon={<TuneIcon />}
                  onClick={() => setParametersDialogOpen(true)}
                  sx={{ textTransform: 'none', borderRadius: 2 }}
                >
                  Manage Parameters
                </Button>
              </Stack>
              <ParametersToolbar
                parameters={state.parameters || []}
                onAddParameter={(p) => {
                  setState((prev) => ({ ...prev, parameters: [...(prev.parameters || []), p] }));
                }}
                onUpdateParameter={(p) => {
                  setState((prev) => ({
                    ...prev,
                    parameters: (prev.parameters || []).map((x) => (x.id === p.id ? p : x)),
                  }));
                }}
                onRemoveParameter={(id) => {
                  setState((prev) => ({
                    ...prev,
                    parameters: (prev.parameters || []).filter((x) => x.id !== id),
                  }));
                }}
                onChangeParamValue={(id, val) => {
                  setState((prev) => ({
                    ...prev,
                    parameters: (prev.parameters || []).map((x) => (x.id === id ? { ...x, currentValue: val } : x)),
                  }));
                }}
              />
            </Box>
          )}

          {source && mainTab === 'schedule' && (
            <Box sx={{ flex: 1, overflow: 'auto', p: 3 }}>
              <ReportScheduleBurstingTab
                reportId={activeTabId}
                reportName={queryName}
              />
            </Box>
          )}

          {source && mainTab === 'results' && (
            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
              {/* View Mode & Output Selector */}
              <Box
                sx={{
                  px: 2.5,
                  py: 1,
                  borderBottom: 1,
                  borderColor: 'divider',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  bgcolor: 'background.paper',
                }}
              >
                <ToggleButtonGroup
                  size="small"
                  value={viewMode}
                  exclusive
                  onChange={(_, val) => val && setViewMode(val)}
                >
                  <ToggleButton value="table">
                    <TableIcon fontSize="small" sx={{ mr: 0.5 }} /> Table
                  </ToggleButton>
                  <ToggleButton value="bar">
                    <BarChartIcon fontSize="small" sx={{ mr: 0.5 }} /> Bar
                  </ToggleButton>
                  <ToggleButton value="line">
                    <LineChartIcon fontSize="small" sx={{ mr: 0.5 }} /> Line
                  </ToggleButton>
                  <ToggleButton value="pie">
                    <PieChartIcon fontSize="small" sx={{ mr: 0.5 }} /> Pie
                  </ToggleButton>
                </ToggleButtonGroup>

                {result && (
                  <Typography variant="caption" color="text.secondary">
                    {result.rowCount.toLocaleString()} rows returned in {result.executionTimeMs}ms
                  </Typography>
                )}
              </Box>

              {/* Results Content Area */}
              <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
                {viewMode === 'table' ? (
                  <ResultsTablePane
                    result={result}
                    isLoading={isLoading}
                    isPreviewing={isPreviewing}
                    error={error}
                  />
                ) : (
                  <VisualizationPane
                    source={source}
                    result={result}
                    mode={viewMode}
                  />
                )}
              </Box>
            </Box>
          )}
        </Box>
      </Box>

      {/* Unified Business Object & Binding Picker Modal */}
      <UnifiedBOPickerModal
        open={showModelPicker}
        businessObjects={allBusinessObjects}
        selectedBoId={source?.id}
        context="query"
        onSelect={handleSelectModel}
        onClose={() => setShowModelPicker(false)}
      />

      {/* Filter Modal with Parameter Binding */}
      {source && (
        <FilterModal
          open={filterModalOpen}
          source={source}
          parameters={state.parameters || []}
          initialFieldId={filterModalFieldId}
          initialFilter={editingFilterIndex !== null ? state.filters[editingFilterIndex] : undefined}
          onApply={handleAddFilter}
          onClose={() => {
            setFilterModalOpen(false);
            setEditingFilterIndex(null);
            setFilterModalFieldId(undefined);
          }}
        />
      )}

      {/* Schedule Workbook Modal */}
      <ScheduleQueryModal
        open={scheduleModalOpen}
        queryName={queryName}
        onClose={() => setScheduleModalOpen(false)}
        onSaveSchedule={(cfg) => {
          devError('Saved schedule:', cfg);
        }}
      />

      {/* SQL Preview Dialog */}
      <SqlPreviewDialog
        open={sqlDialogOpen}
        sql={result?.sql || ''}
        plan={result?.plan}
        onClose={() => setSqlDialogOpen(false)}
      />
    </Box>
  );
};

export default DataQueryBuilderPage;
