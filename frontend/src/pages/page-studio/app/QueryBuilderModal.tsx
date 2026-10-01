import React, { useEffect, useMemo, useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Tabs,
  Tab,
  Box,
  Typography,
  TextField,
  Button,
  Stack,
  Chip,
  Paper,
  CircularProgress,
  Alert,
  IconButton,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  Tooltip,
  Divider,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import UpgradeIcon from '@mui/icons-material/Upgrade';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import BusinessObjectExplorerPanel from '../../../components/shared/BusinessObjectExplorerPanel';
import BusinessObjectSelectorControl from '../../../components/shared/BusinessObjectSelectorControl';
import QueryShelves, { type EditableFilter } from '../../../components/shared/QueryShelves';
import QueryResultsPanel from '../../../components/shared/QueryResultsPanel';
import DrillThroughTargetsEditor from '../../../features/query-builder/components/DrillThroughTargetsEditor';
import type { DrillThroughTarget } from '../../../features/query-builder/utils/drilldown';
import { useBusinessObjectSelector } from '../../../studio-core/binding/useBusinessObjectSelector';
import type { BusinessObjectOption, SemanticTermView } from '../../../studio-core/binding/businessObjectApi';
import {
  listSavedQueries,
  createSavedQuery,
  runSavedQuery,
  type SavedQueryRunResult,
} from '../../../features/query-builder/services/savedQueryApi';
import type {
  SavedQuery,
  SavedQueryState,
  QueryDef,
  SavedQueryParameter,
  CoreQueryStatus,
} from '../../../features/query-builder/types/queryDef';
import { previewQuery } from '../../../features/query-builder/services/queryBuilderApi';
import { friendlyQueryError, savedQueryResultToSet, type SavedQueryRunShape } from '../../../features/query-execution';
import { useTenant } from '../../../contexts/TenantContext';
import type { PageQuery, ParamBinding, PageVariable } from './appModel';

export interface QueryBuilderModalProps {
  open: boolean;
  onClose: () => void;
  onSave: (query: PageQuery) => void;
  initialQuery?: PageQuery;
  pageVariables?: PageVariable[];
  editingQueryId?: string;
}

export function CoreStatusBadge({ status }: { status?: CoreQueryStatus }) {
  if (!status || status === 'custom') {
    return <Chip size="small" label="CUSTOM" color="success" variant="outlined" />;
  }
  switch (status) {
    case 'core':
      return <Chip size="small" label="CORE (Master)" color="primary" variant="filled" />;
    case 'vanilla':
      return <Chip size="small" label="CORE (Vanilla)" color="info" variant="outlined" />;
    case 'extended':
      return <Chip size="small" label="CORE (Extended)" color="secondary" variant="outlined" />;
    case 'upgrade_available':
      return (
        <Tooltip title="A new core version is available to merge">
          <Chip size="small" icon={<UpgradeIcon />} label="UPGRADE AVAILABLE" color="warning" variant="filled" />
        </Tooltip>
      );
    case 'cloned':
      return <Chip size="small" label="CLONED" variant="outlined" />;
    default:
      return <Chip size="small" label={status.toUpperCase()} variant="outlined" />;
  }
}

export function validateParamBinding(
  param: SavedQueryParameter,
  binding?: ParamBinding,
  pageVariables: PageVariable[] = []
): { valid: boolean; message?: string } {
  if (!binding) {
    if (param.required) {
      return { valid: false, message: 'Required parameter has no binding' };
    }
    return { valid: true };
  }

  if (binding.mode === 'pageVar') {
    const v = pageVariables.find((pv) => pv.name === binding.varName);
    if (!v) {
      return { valid: false, message: `Page variable "${binding.varName}" not found` };
    }
    // Type check if variable default is present
    if (v.default !== undefined && param.type) {
      const actualType = typeof v.default;
      if (param.type === 'number' && actualType !== 'number') {
        return { valid: false, message: `Type mismatch: parameter expects number, variable default is ${actualType}` };
      }
      if (param.type === 'boolean' && actualType !== 'boolean') {
        return { valid: false, message: `Type mismatch: parameter expects boolean, variable default is ${actualType}` };
      }
    }
  }

  if (binding.mode === 'staticLiteral') {
    if (param.type === 'number' && isNaN(Number(binding.value))) {
      return { valid: false, message: `Literal value "${binding.value}" is not a valid number` };
    }
  }

  return { valid: true };
}

function useSafeTenant() {
  try {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    return useTenant();
  } catch {
    return { tenant: null };
  }
}

export default function QueryBuilderModal({
  open,
  onClose,
  onSave,
  initialQuery,
  pageVariables = [],
  editingQueryId,
}: QueryBuilderModalProps) {
  const [tab, setTab] = useState<'visual' | 'library'>(initialQuery?.kind === 'savedQuery' ? 'library' : 'visual');
  const { tenant } = useSafeTenant();

  // ---------------------------------------------------------------------------
  // Tab 1: Visual Query Studio State
  // ---------------------------------------------------------------------------
  const [queryName, setQueryName] = useState(editingQueryId || 'query1');
  const [queryDescription, setQueryDescription] = useState('');
  const [filters, setFilters] = useState<EditableFilter[]>([]);
  const [parameters, setParameters] = useState<SavedQueryState['parameters']>([]);
  const [drillThroughTargets, setDrillThroughTargets] = useState<DrillThroughTarget[]>([]);
  const [activeBoId, setActiveBoId] = useState('');
  const [running, setRunning] = useState(false);
  const [runResult, setRunResult] = useState<SavedQueryRunShape | null>(null);
  const [runError, setRunError] = useState<string | null>(null);
  const [compiledSql, setCompiledSql] = useState<{ sql: string; dialect?: string } | null>(null);
  const [sqlLoading, setSqlLoading] = useState(false);
  const [sqlError, setSqlError] = useState<string | null>(null);

  const selector = useBusinessObjectSelector({
    lockPrimaryAfterSelect: false,
  });

  useEffect(() => {
    if (selector.primary && !activeBoId) {
      setActiveBoId(selector.primary.boId);
    }
  }, [selector.primary, activeBoId]);

  const addFilterFromField = (boId: string, term: SemanticTermView) => {
    const primaryBoId = selector.primary?.boId;
    setFilters((prev) => [
      ...prev,
      {
        termNodeId: term.termNodeId,
        boId: boId === primaryBoId ? undefined : boId,
        operator: 'eq',
        value: '',
      },
    ]);
  };

  const updateFilter = (index: number, patch: Partial<EditableFilter>) => {
    setFilters((prev) => prev.map((f, i) => (i === index ? { ...f, ...patch } : f)));
  };

  const removeFilter = (index: number) => {
    setFilters((prev) => prev.filter((_, i) => i !== index));
  };

  const addParameter = (param: SavedQueryParameter) => {
    setParameters((prev) => [...prev, param]);
  };

  const removeParameter = (name: string) => {
    setParameters((prev) => prev.filter((p) => p.name !== name));
    setFilters((prev) => prev.map((f) => (f.paramRef === name ? { ...f, paramRef: undefined, value: '' } : f)));
  };

  const buildVisualState = (): SavedQueryState => {
    const dimensions: SavedQueryState['dimensions'] = [];
    const measures: SavedQueryState['measures'] = [];
    for (const obj of selector.objects) {
      const boId = obj.isPrimary ? undefined : obj.boId;
      for (const t of obj.terms) {
        if (!t.selected) continue;
        if (t.role === 'MEASURE') {
          measures.push({
            termNodeId: t.termNodeId,
            alias: t.termKey,
            agg: t.termType === 'calculated' ? 'NONE' : t.defaultAggregation || 'NONE',
            boId,
          });
        } else {
          dimensions.push({ termNodeId: t.termNodeId, alias: t.termKey, boId });
        }
      }
    }
    const filterDefs: SavedQueryState['filters'] = filters.map((f) => ({
      termNodeId: f.termNodeId,
      boId: f.boId,
      operator: f.operator,
      value: f.paramRef ? undefined : f.value,
      paramRef: f.paramRef,
    }));
    return { dimensions, measures, filters: filterDefs, parameters };
  };

  const handleCompileSql = async () => {
    if (!selector.primary) return;
    setSqlLoading(true);
    setSqlError(null);
    try {
      const state = buildVisualState();
      const qd: QueryDef = {
        context: {
          boId: selector.primary.boId,
          bindingId: selector.primary.bindingId,
          tenantId: tenant?.id || '',
          relatedBoIds: selector.related.map((r) => r.boId),
        },
        query: {
          dimensions: state.dimensions,
          measures: state.measures,
          filters: state.filters,
          limit: state.limit,
        },
      };
      const result = await previewQuery(qd);
      setCompiledSql(result);
    } catch (e: any) {
      setSqlError(friendlyQueryError(e.message));
    } finally {
      setSqlLoading(false);
    }
  };

  const handleVisualSave = async () => {
    if (!selector.primary) return;
    setRunning(true);
    try {
      const state = buildVisualState();
      // Create saved query in data_explorer
      const sq = await createSavedQuery({
        name: queryName,
        description: queryDescription,
        boId: selector.primary.boId,
        bindingId: selector.primary.bindingId,
        relatedBoIds: selector.related.map((r) => r.boId),
        chartType: 'bar',
        state,
        tags: [],
      });

      // Pass saved query as PageQuery
      onSave({
        id: queryName,
        kind: 'savedQuery',
        savedQueryId: sq.id,
        paramBindings: {},
      });
      onClose();
    } catch (e: any) {
      setRunError(e.message);
    } finally {
      setRunning(false);
    }
  };

  // ---------------------------------------------------------------------------
  // Tab 2: Saved & Core Query Library State
  // ---------------------------------------------------------------------------
  const [savedQueries, setSavedQueries] = useState<SavedQuery[]>([]);
  const [loadingQueries, setLoadingQueries] = useState(false);
  const [selectedQueryId, setSelectedQueryId] = useState<string>(
    initialQuery?.kind === 'savedQuery' ? initialQuery.savedQueryId : ''
  );
  const [searchFilter, setSearchFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [paramBindings, setParamBindings] = useState<Record<string, ParamBinding>>(
    initialQuery?.kind === 'savedQuery' && initialQuery.paramBindings ? initialQuery.paramBindings : {}
  );
  const [previewResult, setPreviewResult] = useState<SavedQueryRunResult | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);

  useEffect(() => {
    if (open) {
      setLoadingQueries(true);
      listSavedQueries()
        .then((list) => {
          setSavedQueries(list);
          if (!selectedQueryId && list.length > 0) {
            setSelectedQueryId(list[0].id);
          }
        })
        .catch(() => setSavedQueries([]))
        .finally(() => setLoadingQueries(false));
    }
  }, [open]);

  const selectedQuery = useMemo(
    () => savedQueries.find((q) => q.id === selectedQueryId),
    [savedQueries, selectedQueryId]
  );

  const filteredQueries = useMemo(() => {
    return savedQueries.filter((q) => {
      const matchSearch =
        q.name.toLowerCase().includes(searchFilter.toLowerCase()) ||
        (q.description && q.description.toLowerCase().includes(searchFilter.toLowerCase()));
      if (!matchSearch) return false;
      if (statusFilter === 'all') return true;
      if (statusFilter === 'core') return q.isCore || q.coreStatus === 'core' || q.coreStatus === 'vanilla';
      if (statusFilter === 'extended') return q.coreStatus === 'extended' || q.coreStatus === 'upgrade_available';
      if (statusFilter === 'custom') return !q.isCore && q.coreStatus !== 'core';
      return true;
    });
  }, [savedQueries, searchFilter, statusFilter]);

  const handleParamBindingChange = (paramName: string, binding: ParamBinding) => {
    setParamBindings((prev) => ({
      ...prev,
      [paramName]: binding,
    }));
  };

  const handleLibraryPreview = async () => {
    if (!selectedQuery) return;
    setPreviewLoading(true);
    try {
      const resolvedMap: Record<string, string | number | boolean> = {};
      Object.entries(paramBindings).forEach(([pName, b]) => {
        if (b.mode === 'staticLiteral') resolvedMap[pName] = b.value as string;
        else if (b.mode === 'pageVar') {
          const v = pageVariables.find((pv) => pv.name === b.varName);
          if (v?.default !== undefined) resolvedMap[pName] = v.default as string;
        }
      });
      const res = await runSavedQuery(selectedQuery.id, resolvedMap);
      setPreviewResult(res);
    } catch (e: any) {
      setRunError(e.message);
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleLibrarySave = () => {
    if (!selectedQuery) return;
    onSave({
      id: editingQueryId || selectedQuery.name.replace(/[^\w]/g, '_').toLowerCase(),
      kind: 'savedQuery',
      savedQueryId: selectedQuery.id,
      paramBindings,
    });
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xl" fullWidth PaperProps={{ sx: { height: '88vh', display: 'flex', flexDirection: 'column' } }}>
      <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', pb: 1 }}>
        <Stack direction="row" spacing={2} alignItems="center">
          <Typography variant="h6" fontWeight={700}>
            Query Studio
          </Typography>
          <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ minHeight: 36 }}>
            <Tab label="Visual Query Designer" value="visual" sx={{ py: 0.5, minHeight: 36 }} />
            <Tab label="Public & Core Query Library" value="library" sx={{ py: 0.5, minHeight: 36 }} />
          </Tabs>
        </Stack>
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent dividers sx={{ p: 2, flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        {tab === 'visual' ? (
          <Stack direction="row" spacing={2} sx={{ flex: 1, minHeight: 0 }}>
            {/* Visual Builder Column 1 & 2 */}
            <Box sx={{ width: 340, flexShrink: 0, height: '100%', overflowY: 'auto' }}>
              {!selector.primary ? (
                <Paper variant="outlined" sx={{ p: 2 }}>
                  <Typography variant="subtitle2" gutterBottom fontWeight={600}>
                    Choose Primary Business Object
                  </Typography>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
                    Pick the primary business object entry point to start querying.
                  </Typography>
                  <BusinessObjectSelectorControl
                    selector={selector}
                    renderFields={false}
                    onSelectPrimary={(_, bid) => {
                      if (selector.primary) selector.setPrimaryBinding(bid);
                    }}
                  />
                </Paper>
              ) : (
                <BusinessObjectExplorerPanel
                  selector={selector}
                  activeBoId={activeBoId}
                  onSelectActive={setActiveBoId}
                  onToggleField={selector.toggleField}
                  onAddFilter={addFilterFromField}
                />
              )}
            </Box>

            {/* Visual Builder Column 3 */}
            <Box sx={{ flex: 1, minWidth: 0, height: '100%', overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 1.5 }}>
              <Paper variant="outlined" sx={{ p: 1.5 }}>
                <Stack direction="row" spacing={2} alignItems="center">
                  <TextField
                    size="small"
                    label="Query ID / Name"
                    value={queryName}
                    onChange={(e) => setQueryName(e.target.value.replace(/[^\w]/g, ''))}
                    sx={{ width: 260 }}
                  />
                  <TextField
                    size="small"
                    label="Description"
                    value={queryDescription}
                    onChange={(e) => setQueryDescription(e.target.value)}
                    fullWidth
                  />
                  <Button
                    variant="outlined"
                    startIcon={sqlLoading ? <CircularProgress size={14} /> : <AutoFixHighIcon />}
                    onClick={handleCompileSql}
                    disabled={!selector.primary || sqlLoading}
                  >
                    Compile SQL
                  </Button>
                </Stack>
              </Paper>

              <QueryShelves
                selector={selector}
                filters={filters}
                onUpdateFilter={updateFilter}
                onRemoveFilter={removeFilter}
                parameters={parameters}
                onAddParameter={addParameter}
                onRemoveParameter={removeParameter}
              />

              <DrillThroughTargetsEditor
                targets={drillThroughTargets}
                onChange={setDrillThroughTargets}
                outputColumns={selector.objects.flatMap((o) => o.terms.filter((t) => t.selected).map((t) => t.termKey))}
                pageVariables={pageVariables}
              />

              <QueryResultsPanel
                resultSet={runResult ? savedQueryResultToSet(runResult) : null}
                running={running}
                runError={runError}
                sql={compiledSql}
                sqlLoading={sqlLoading}
                sqlError={sqlError}
                onRequestCompileSql={handleCompileSql}
              />
            </Box>
          </Stack>
        ) : (
          /* Tab 2: Public & Core Query Library */
          <Stack direction="row" spacing={2} sx={{ flex: 1, minHeight: 0 }}>
            {/* Library Query List */}
            <Box sx={{ width: 380, flexShrink: 0, height: '100%', overflowY: 'auto', pr: 1 }}>
              <Stack spacing={1.5}>
                <TextField
                  size="small"
                  placeholder="Search queries…"
                  value={searchFilter}
                  onChange={(e) => setSearchFilter(e.target.value)}
                  fullWidth
                />
                <FormControl size="small" fullWidth>
                  <InputLabel>Filter by Status</InputLabel>
                  <Select value={statusFilter} label="Filter by Status" onChange={(e) => setStatusFilter(e.target.value)}>
                    <MenuItem value="all">All Queries</MenuItem>
                    <MenuItem value="core">Core Gold-Copy</MenuItem>
                    <MenuItem value="extended">Extended Core</MenuItem>
                    <MenuItem value="custom">Tenant Custom</MenuItem>
                  </Select>
                </FormControl>

                {loadingQueries ? (
                  <Box display="flex" justifyContent="center" p={3}>
                    <CircularProgress size={24} />
                  </Box>
                ) : filteredQueries.length === 0 ? (
                  <Alert severity="info">No queries match your criteria.</Alert>
                ) : (
                  filteredQueries.map((q) => {
                    const isSelected = q.id === selectedQueryId;
                    return (
                      <Paper
                        key={q.id}
                        variant="outlined"
                        onClick={() => setSelectedQueryId(q.id)}
                        sx={{
                          p: 1.5,
                          cursor: 'pointer',
                          borderColor: isSelected ? 'primary.main' : 'divider',
                          bgcolor: isSelected ? 'action.selected' : 'background.paper',
                          '&:hover': { bgcolor: 'action.hover' },
                        }}
                      >
                        <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                          <Typography variant="body2" fontWeight={700}>
                            {q.name}
                          </Typography>
                          <CoreStatusBadge status={q.coreStatus} />
                        </Stack>
                        {q.description && (
                          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                            {q.description}
                          </Typography>
                        )}
                        <Stack direction="row" spacing={0.5} sx={{ mt: 1 }}>
                          <Chip size="small" label={q.chartType} variant="outlined" />
                          <Chip size="small" label={`${q.state.dimensions.length} dims`} variant="outlined" />
                          <Chip size="small" label={`${q.state.measures.length} meas`} variant="outlined" />
                          {q.state.parameters?.length > 0 && (
                            <Chip size="small" label={`${q.state.parameters.length} params`} color="info" variant="outlined" />
                          )}
                        </Stack>
                      </Paper>
                    );
                  })
                )}
              </Stack>
            </Box>

            <Divider orientation="vertical" flexItem />

            {/* Selected Query Details & Parameter Binding Matrix */}
            <Box sx={{ flex: 1, minWidth: 0, height: '100%', overflowY: 'auto', pl: 1 }}>
              {selectedQuery ? (
                <Stack spacing={2}>
                  <Paper variant="outlined" sx={{ p: 2 }}>
                    <Stack direction="row" justifyContent="space-between" alignItems="center">
                      <Box>
                        <Typography variant="h6" fontWeight={700}>
                          {selectedQuery.name}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          {selectedQuery.description || 'No description provided.'}
                        </Typography>
                      </Box>
                      <CoreStatusBadge status={selectedQuery.coreStatus} />
                    </Stack>
                  </Paper>

                  {/* Parameter Binding Matrix */}
                  <Paper variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="subtitle2" fontWeight={700} gutterBottom>
                      Parameter Bindings
                    </Typography>
                    {(!selectedQuery.state.parameters || selectedQuery.state.parameters.length === 0) ? (
                      <Typography variant="caption" color="text.secondary">
                        This query has no parameters; it will execute as-is without runtime bindings.
                      </Typography>
                    ) : (
                      <Stack spacing={2} sx={{ mt: 1 }}>
                        {selectedQuery.state.parameters.map((param) => {
                          const currentBinding = paramBindings[param.name];
                          const validation = validateParamBinding(param, currentBinding, pageVariables);

                          return (
                            <Box key={param.name} sx={{ border: 1, borderColor: 'divider', borderRadius: 1, p: 1.5 }}>
                              <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 1 }}>
                                <Typography variant="body2" fontWeight={600}>
                                  {param.label || param.name}
                                  {param.required && ' *'}
                                </Typography>
                                <Chip size="small" label={param.type || 'string'} variant="outlined" />
                                {!validation.valid && (
                                  <Alert severity="error" sx={{ py: 0, px: 1, fontSize: '0.75rem' }}>
                                    {validation.message}
                                  </Alert>
                                )}
                              </Stack>

                              <Stack direction="row" spacing={1.5} alignItems="center">
                                <FormControl size="small" sx={{ width: 180 }}>
                                  <InputLabel>Source</InputLabel>
                                  <Select
                                    value={currentBinding?.mode || 'staticLiteral'}
                                    label="Source"
                                    onChange={(e) => {
                                      const mode = e.target.value as ParamBinding['mode'];
                                      if (mode === 'pageVar') {
                                        handleParamBindingChange(param.name, {
                                          mode: 'pageVar',
                                          varName: pageVariables[0]?.name || '',
                                        });
                                      } else if (mode === 'selectionContext') {
                                        handleParamBindingChange(param.name, {
                                          mode: 'selectionContext',
                                          field: 'recordId',
                                        });
                                      } else if (mode === 'urlParam') {
                                        handleParamBindingChange(param.name, {
                                          mode: 'urlParam',
                                          paramName: param.name,
                                        });
                                      } else {
                                        handleParamBindingChange(param.name, {
                                          mode: 'staticLiteral',
                                          value: param.default !== undefined ? (param.default as string) : '',
                                        });
                                      }
                                    }}
                                  >
                                    <MenuItem value="pageVar">Page Variable</MenuItem>
                                    <MenuItem value="selectionContext">Selection Context</MenuItem>
                                    <MenuItem value="staticLiteral">Static Literal</MenuItem>
                                    <MenuItem value="urlParam">URL Parameter</MenuItem>
                                  </Select>
                                </FormControl>

                                {currentBinding?.mode === 'pageVar' && (
                                  <FormControl size="small" sx={{ flex: 1 }}>
                                    <InputLabel>Page Variable</InputLabel>
                                    <Select
                                      value={currentBinding.varName}
                                      label="Page Variable"
                                      onChange={(e) =>
                                        handleParamBindingChange(param.name, {
                                          mode: 'pageVar',
                                          varName: e.target.value,
                                        })
                                      }
                                    >
                                      {pageVariables.map((pv) => (
                                        <MenuItem key={pv.name} value={pv.name}>
                                          {pv.name} {pv.default !== undefined ? `(default: ${pv.default})` : ''}
                                        </MenuItem>
                                      ))}
                                    </Select>
                                  </FormControl>
                                )}

                                {currentBinding?.mode === 'staticLiteral' && (
                                  <TextField
                                    size="small"
                                    label="Literal Value"
                                    value={currentBinding.value ?? ''}
                                    onChange={(e) =>
                                      handleParamBindingChange(param.name, {
                                        mode: 'staticLiteral',
                                        value: param.type === 'number' ? Number(e.target.value) : e.target.value,
                                      })
                                    }
                                    fullWidth
                                  />
                                )}

                                {currentBinding?.mode === 'urlParam' && (
                                  <TextField
                                    size="small"
                                    label="URL Query Param Name"
                                    value={currentBinding.paramName}
                                    onChange={(e) =>
                                      handleParamBindingChange(param.name, {
                                        mode: 'urlParam',
                                        paramName: e.target.value,
                                      })
                                    }
                                    fullWidth
                                  />
                                )}
                              </Stack>
                            </Box>
                          );
                        })}
                      </Stack>
                    )}
                  </Paper>

                  {/* Live Run Preview Button */}
                  <Stack direction="row" spacing={1}>
                    <Button
                      variant="outlined"
                      startIcon={previewLoading ? <CircularProgress size={14} /> : <PlayArrowIcon />}
                      onClick={handleLibraryPreview}
                      disabled={previewLoading}
                    >
                      Test Run with Bindings
                    </Button>
                  </Stack>

                  {previewResult && (
                    <Paper variant="outlined" sx={{ p: 2 }}>
                      <Typography variant="subtitle2" gutterBottom fontWeight={700}>
                        Preview Result ({previewResult.rowCount} rows)
                      </Typography>
                      <Box sx={{ maxHeight: 200, overflowY: 'auto' }}>
                        <pre style={{ fontSize: '0.75rem' }}>{JSON.stringify(previewResult.rows.slice(0, 5), null, 2)}</pre>
                      </Box>
                    </Paper>
                  )}
                </Stack>
              ) : (
                <Alert severity="info">Select a query on the left to configure bindings.</Alert>
              )}
            </Box>
          </Stack>
        )}
      </DialogContent>

      <DialogActions sx={{ p: 2, justifyContent: 'space-between' }}>
        <Button onClick={onClose}>Cancel</Button>
        {tab === 'visual' ? (
          <Button
            variant="contained"
            startIcon={<CheckCircleIcon />}
            onClick={handleVisualSave}
            disabled={!selector.primary}
          >
            Save & Use in Page
          </Button>
        ) : (
          <Button
            variant="contained"
            startIcon={<CheckCircleIcon />}
            onClick={handleLibrarySave}
            disabled={!selectedQuery}
          >
            Select & Bind Query
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}
