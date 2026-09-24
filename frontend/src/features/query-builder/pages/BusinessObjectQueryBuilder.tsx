import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Box,
  Typography,
  Button,
  Paper,
  List,
  ListItemButton,
  ListItemText,
  ListItemIcon,
  CircularProgress,
  Alert,
  TextField,
  Chip,
  IconButton,
  InputAdornment,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Stack,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Tooltip,
} from '@mui/material';
import {
  Search as SearchIcon,
  FilterList as FilterIcon,
  PlayArrow as RunIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  TableChart as TableIcon,
  Code as CodeIcon,
  Api as ApiIcon,
  AccountTree as PlanIcon,
  Numbers as NumberIcon,
  Abc as StringIcon,
  CalendarToday as DateIcon,
  CheckCircle as CheckCircleIcon,
  Functions as FunctionsIcon,
  Speed as SpeedIcon,
} from '@mui/icons-material';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  horizontalListSortingStrategy,
} from '@dnd-kit/sortable';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

import { useTenant } from '../../../contexts/TenantContext';
import { getRequiredTenantScope } from '../../../utils/tenantScope';
import { devError } from '../../../utils/devLogger';
import apiClient from '../../../utils/apiClient';
import {
  fetchBusinessObjectBindings,
  fetchBOTerms,
  fetchBOSchema,
  previewQuery,
  executeQuery,
} from '../services/queryBuilderApi';
import { QueryEngineIndicator, EngineRouteState } from '../../../components/BusinessObjectManager/QueryEngineIndicator';
import {
  createEmptyQueryDef,
  makeAlias,
  getUsedAliases,
} from '../types/queryDef';
import type {
  QueryDef,
  SemanticTermView,
  BindingView,
  QueryExecuteResult,
  FilterDef,
  FilterOperator,
  PreviewResult,
  BOSchema,
  BOSchemaField,
  SavedQueryParameter,
  SavedQueryChartType,
  SavedQuery,
  SavedQueryState,
} from '../types/queryDef';
import ExplainPlanVisualizer from '../components/ExplainPlanVisualizer';
import ScrollAreaResultView from '../components/ScrollAreaResultView';
import QueryPerformanceSummary from '../components/QueryPerformanceSummary';
import AutoFormRenderer from '../components/AutoFormRenderer';
import { JWTInspector } from '../../../components/BusinessObjectManager/JWTInspector';
import { listSavedQueries, createSavedQuery, updateSavedQuery, getSavedQuery, deleteSavedQuery, runSavedQuery, savedQueryRestUrl } from '../services/savedQueryApi';
import { buildChartOption } from '../utils/chartOption';
import ReactECharts from 'echarts-for-react';
import {
  Save as SaveIcon,
  BarChart as ChartTabIcon,
  ContentCopy as CopyIcon,
  FolderOpen as OpenSavedIcon,
} from '@mui/icons-material';
import { Dialog, DialogTitle, DialogContent, DialogActions, Switch, FormControlLabel } from '@mui/material';

// ─── Types ───────────────────────────────────────────────────────────────────

interface BusinessObject {
  id: string;
  name: string;
  display_name: string;
}

// Shape returned by GET /api/business-objects/{id}/relationships
// (backend/internal/metadata/businessobject_service.go RelationshipResult)
interface RelatedBusinessObject {
  id: string;
  relatedObjectName: string;
  targetObjectId: string;
  relationshipType: string;
  cardinality: string;
  joinCondition: string;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const FieldIcon: React.FC<{ type?: string }> = ({ type }) => {
  const t = (type || 'string').toLowerCase();
  if (['integer', 'decimal', 'number', 'float', 'double'].includes(t)) {
    return <NumberIcon fontSize="small" sx={{ color: '#4caf50' }} />;
  }
  if (['date', 'datetime', 'timestamp', 'time'].includes(t)) {
    return <DateIcon fontSize="small" sx={{ color: '#ff9800' }} />;
  }
  return <StringIcon fontSize="small" sx={{ color: '#2196f3' }} />;
};

const RoleBadge: React.FC<{ role: SemanticTermView['role'] }> = ({ role }) => {
  const colors: Record<SemanticTermView['role'], string> = {
    DIMENSION: '#2196f3',
    MEASURE: '#4caf50',
    CALCULATED: '#9c27b0',
  };
  return (
    <Box
      component="span"
      sx={{
        fontSize: '0.65rem',
        textTransform: 'uppercase',
        fontWeight: 700,
        color: colors[role],
        ml: 1,
      }}
    >
      {role}
    </Box>
  );
};

function debounce(fn: (qd: QueryDef) => void, ms: number) {
  let timer: ReturnType<typeof setTimeout> | null = null;
  return (qd: QueryDef) => {
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => fn(qd), ms);
  };
}


// ─── Sortable Chip ───────────────────────────────────────────────────────────

interface SortableChipProps {
  id: string;
  label: React.ReactNode;
  onDelete: () => void;
  color?: 'primary' | 'secondary' | 'default';
}

const SortableChip: React.FC<SortableChipProps> = ({ id, label, onDelete, color = 'primary' }) => {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <Chip
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      label={label}
      onDelete={onDelete}
      color={color}
      variant="outlined"
      size="small"
      sx={{ cursor: 'grab', '&:active': { cursor: 'grabbing' } }}
    />
  );
};

// ─── Main Component ──────────────────────────────────────────────────────────

const BusinessObjectQueryBuilder: React.FC = () => {
  const { tenant, datasource } = useTenant();

  // Scope
  const [tenantId, setTenantId] = useState<string>('');

  // Data
  const [businessObjects, setBusinessObjects] = useState<BusinessObject[]>([]);
  const [selectedBO, setSelectedBO] = useState<BusinessObject | null>(null);
  const [bindings, setBindings] = useState<BindingView[]>([]);
  const [selectedBindingId, setSelectedBindingId] = useState<string>('');
  const [terms, setTerms] = useState<SemanticTermView[]>([]);
  const [boSchema, setBoSchema] = useState<BOSchema | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [relatedBOs, setRelatedBOs] = useState<RelatedBusinessObject[]>([]);
  // BOs the user has added to this query as joins, keyed by their catalog
  // node id. Their terms are merged into the term browser tagged with boId,
  // and queryDef.context.relatedBoIds is derived from these keys.
  const [includedRelatedBOs, setIncludedRelatedBOs] = useState<
    Record<string, { name: string; cardinalityHint: 'one' | 'many'; terms: SemanticTermView[] }>
  >({});
  const [relatedTermsLoading, setRelatedTermsLoading] = useState<string | null>(null);

  // Query state
  const [queryDef, setQueryDef] = useState<QueryDef | null>(null);

  // UI state
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [previewResult, setPreviewResult] = useState<PreviewResult | null>(null);
  const [executeResult, setExecuteResult] = useState<QueryExecuteResult | null>(null);

  // Saved query: parameters (named placeholders a filter can defer its
  // value to instead of a literal - see FilterDef.paramRef), a visual, and
  // persistence. Kept separate from `queryDef` (the ad-hoc QueryDef this
  // page already sends to /api/query/execute) since parameters/chartType/
  // save-identity aren't part of that wire contract at all.
  const [parameters, setParameters] = useState<SavedQueryParameter[]>([]);
  const [chartType, setChartType] = useState<SavedQueryChartType>('bar');
  const [savedQueryId, setSavedQueryId] = useState<string | null>(null);
  const [savedQueryName, setSavedQueryName] = useState('');
  const [savedQueryDescription, setSavedQueryDescription] = useState('');
  const [savedQueries, setSavedQueries] = useState<SavedQuery[]>([]);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [chartDimTerm, setChartDimTerm] = useState<string>('');
  const [chartMeasureTerm, setChartMeasureTerm] = useState<string>('');
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);

  // Drag sensors
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  // Resolve tenant id once
  useEffect(() => {
    let id = tenant?.id;
    if (!id) {
      try {
        id = getRequiredTenantScope().tenantId;
      } catch (e) {
        setError('Tenant scope is not selected');
      }
    }
    if (id) setTenantId(id);
  }, [tenant]);

  // Fetch business objects
  useEffect(() => {
    if (!tenantId) return;

    const load = async () => {
      setLoading(true);
      try {
        const data = await apiClient<unknown>('/business-objects', {
          headers: {
            'X-Tenant-ID': tenantId,
            ...(datasource?.id ? { 'X-Tenant-Datasource-ID': datasource.id } : {}),
          },
        });

        let rawList: unknown[] = [];
        if (Array.isArray(data)) rawList = data;
        else if (data && typeof data === 'object') {
          if (Array.isArray((data as any).businessObjects)) rawList = (data as any).businessObjects;
          else if (Array.isArray((data as any).business_objects)) rawList = (data as any).business_objects;
          else if (Array.isArray((data as any).items)) rawList = (data as any).items;
          else if (Array.isArray((data as any).data)) rawList = (data as any).data;
          else rawList = Object.values(data);
        }

        const normalized = rawList
          .filter((item): item is Record<string, unknown> => !!item && typeof item === 'object')
          .map((item) => ({
            id: String(item.id ?? item.name),
            name: String(item.name ?? ''),
            display_name: String(item.displayName ?? item.display_name ?? item.name ?? ''),
          }));

        setBusinessObjects(normalized);
        if (normalized.length === 0) {
          setError('No Business Objects available for the current tenant/datasource');
        }
      } catch (err) {
        devError('Failed to load BOs', err);
        setError('Failed to load Business Objects');
      } finally {
        setLoading(false);
      }
    };

    load();
  }, [tenantId, datasource?.id]);

  // Fetch bindings when BO changes
  useEffect(() => {
    setIncludedRelatedBOs({});
    if (!selectedBO || !tenantId) {
      setBindings([]);
      setSelectedBindingId('');
      setTerms([]);
      setQueryDef(null);
      return;
    }

    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const b = await fetchBusinessObjectBindings(selectedBO.id);
        setBindings(b);

        const defaultBinding = b.find((x) => x.isDefault) || b[0];
        if (defaultBinding) {
          setSelectedBindingId(defaultBinding.bindingId);
        } else {
          setError('No binding found for this Business Object');
        }
      } catch (err) {
        devError('Failed to load bindings', err);
        setError('Failed to load bindings for the selected Business Object');
      } finally {
        setLoading(false);
      }
    };

    load();
  }, [selectedBO, tenantId]);

  // Fetch related business objects when the subject area changes, so the
  // user can see (and jump to) BOs reachable from this one via its driving
  // table's relationship graph.
  useEffect(() => {
    if (!selectedBO || !tenantId) {
      setRelatedBOs([]);
      return;
    }

    let cancelled = false;
    apiClient<{ relatedObjects?: RelatedBusinessObject[] }>(
      `/business-objects/${selectedBO.id}/relationships`,
      { headers: { 'X-Tenant-ID': tenantId, ...(datasource?.id ? { 'X-Tenant-Datasource-ID': datasource.id } : {}) } }
    )
      .then((data) => {
        if (!cancelled) setRelatedBOs(data?.relatedObjects || []);
      })
      .catch((err) => {
        devError('Failed to load related business objects', err);
        if (!cancelled) setRelatedBOs([]);
      });

    return () => {
      cancelled = true;
    };
  }, [selectedBO, tenantId, datasource?.id]);

  // Fetch terms when binding changes
  useEffect(() => {
    if (!selectedBO || !selectedBindingId || !tenantId) {
      setTerms([]);
      setQueryDef(null);
      return;
    }

    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const [t, schema] = await Promise.all([
          fetchBOTerms(selectedBO.id, selectedBindingId),
          fetchBOSchema(selectedBO.id, tenantId),
        ]);
        setTerms(t.filter((term) => term.bindingStatus === 'RESOLVED'));
        setBoSchema(schema);
        setQueryDef(createEmptyQueryDef({
          boId: selectedBO.id,
          bindingId: selectedBindingId,
          tenantId,
        }));
        setPreviewResult(null);
        setExecuteResult(null);
        // Switching BO/binding starts a fresh query, not an edit of
        // whatever saved query happened to be loaded before.
        setSavedQueryId(null);
        setSavedQueryName('');
        setSavedQueryDescription('');
        setParameters([]);
        setChartDimTerm('');
        setChartMeasureTerm('');
      } catch (err) {
        devError('Failed to load terms/schema', err);
        setError('Failed to load semantic terms for the selected binding');
      } finally {
        setLoading(false);
      }
    };

    load();
  }, [selectedBO, selectedBindingId, tenantId]);

  // Saved queries for the selected BO - the "Cube Playground"-style list of
  // reusable queries a Page Studio widget (or an external REST caller) can
  // already point at, shown alongside the ad-hoc builder above it.
  const refreshSavedQueries = useCallback(() => {
    if (!selectedBO) { setSavedQueries([]); return; }
    listSavedQueries(selectedBO.id).then(setSavedQueries).catch(() => setSavedQueries([]));
  }, [selectedBO]);
  useEffect(() => { refreshSavedQueries(); }, [refreshSavedQueries]);

  // Default the Chart tab's category/value pickers to the first
  // dimension/measure whenever the query shape changes and nothing (or a
  // now-removed field) is selected - a query with data should never open
  // the Chart tab to an empty picker.
  useEffect(() => {
    if (!queryDef) return;
    if (!queryDef.query.dimensions.some((d) => d.alias === chartDimTerm)) {
      setChartDimTerm(queryDef.query.dimensions[0]?.alias || '');
    }
    if (!queryDef.query.measures.some((m) => m.alias === chartMeasureTerm)) {
      setChartMeasureTerm(queryDef.query.measures[0]?.alias || '');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [queryDef?.query.dimensions, queryDef?.query.measures]);

  // Debounced SQL preview
  const runPreview = useCallback(async (qd: QueryDef) => {
    if (!qd.query.dimensions.length && !qd.query.measures.length) {
      setPreviewResult({ sql: '-- Add dimensions or measures to generate SQL' });
      return;
    }
    try {
      const result = await previewQuery(qd);
      setPreviewResult(result);
    } catch (err) {
      devError('Preview failed', err);
      setPreviewResult({
        sql: `-- Preview failed: ${err instanceof Error ? err.message : String(err)}`,
      });
    }
  }, []);

  const debouncedPreview = useMemo(() => debounce(runPreview, 400), [runPreview]);

  useEffect(() => {
    if (!queryDef) return;
    debouncedPreview(queryDef);
  }, [queryDef, debouncedPreview]);

  // Handlers
  const handleSelectBO = (boId: string) => {
    const bo = businessObjects.find((b) => b.id === boId) || null;
    setSelectedBO(bo);
  };

  const cardinalityHintFromString = (c: string): 'one' | 'many' => {
    const norm = (c || '').toUpperCase().replace(/\s/g, '');
    return norm === '1:M' || norm === '1:N' || norm === 'M:M' || norm === 'N:N' ? 'many' : 'one';
  };

  // Joins a related BO into the query (or removes it): fetches its terms so
  // they appear in the field browser tagged with boId, and keeps
  // queryDef.context.relatedBoIds in sync so the backend resolves the join.
  // Server-computed cardinality (from the actual resolved join path) always
  // wins at render time — this hint only badges fields before a query runs.
  const handleToggleRelatedBO = async (rel: RelatedBusinessObject) => {
    if (!rel.targetObjectId || !tenantId) return;

    if (includedRelatedBOs[rel.targetObjectId]) {
      setIncludedRelatedBOs((prev) => {
        const next = { ...prev };
        delete next[rel.targetObjectId];
        return next;
      });
      setQueryDef((prev) => {
        if (!prev) return prev;
        return {
          ...prev,
          context: {
            ...prev.context,
            relatedBoIds: (prev.context.relatedBoIds || []).filter((id) => id !== rel.targetObjectId),
          },
          query: {
            ...prev.query,
            dimensions: prev.query.dimensions.filter((d) => d.boId !== rel.targetObjectId),
            measures: prev.query.measures.filter((m) => m.boId !== rel.targetObjectId),
            filters: prev.query.filters.filter((f) => f.boId !== rel.targetObjectId),
          },
        };
      });
      return;
    }

    setRelatedTermsLoading(rel.targetObjectId);
    try {
      const relBindings = await fetchBusinessObjectBindings(rel.targetObjectId);
      const defaultBinding = relBindings.find((b) => b.isDefault) || relBindings[0];
      if (!defaultBinding) {
        devError('No binding found for related BO', rel.targetObjectId);
        return;
      }
      const relTerms = await fetchBOTerms(rel.targetObjectId, defaultBinding.bindingId);

      setIncludedRelatedBOs((prev) => ({
        ...prev,
        [rel.targetObjectId]: {
          name: rel.relatedObjectName,
          cardinalityHint: cardinalityHintFromString(rel.cardinality),
          terms: relTerms,
        },
      }));
      setQueryDef((prev) => {
        if (!prev) return prev;
        const existing = prev.context.relatedBoIds || [];
        if (existing.includes(rel.targetObjectId)) return prev;
        return { ...prev, context: { ...prev.context, relatedBoIds: [...existing, rel.targetObjectId] } };
      });
    } catch (err) {
      devError('Failed to load related BO terms', err);
    } finally {
      setRelatedTermsLoading(null);
    }
  };

  const handleAddTerm = (term: SemanticTermView, boId?: string) => {
    if (!queryDef) return;

    setQueryDef((prev) => {
      if (!prev) return prev;
      const used = getUsedAliases(prev.query);
      const alias = makeAlias(term.displayName || term.termName, used);

      if (term.role === 'MEASURE' || term.role === 'CALCULATED') {
        const exists = prev.query.measures.some((m) => m.termNodeId === term.termNodeId && m.boId === boId);
        if (exists) return prev;
        return {
          ...prev,
          query: {
            ...prev.query,
            measures: [
              ...prev.query.measures,
              {
                termNodeId: term.termNodeId,
                alias,
                agg: term.defaultAggregation || 'SUM',
                ...(boId ? { boId } : {}),
              },
            ],
          },
        };
      }

      const exists = prev.query.dimensions.some((d) => d.termNodeId === term.termNodeId && d.boId === boId);
      if (exists) return prev;
      return {
        ...prev,
        query: {
          ...prev.query,
          dimensions: [...prev.query.dimensions, { termNodeId: term.termNodeId, alias, ...(boId ? { boId } : {}) }],
        },
      };
    });
  };

  const handleRemoveDimension = (termNodeId: string) => {
    setQueryDef((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        query: {
          ...prev.query,
          dimensions: prev.query.dimensions.filter((d) => d.termNodeId !== termNodeId),
          groupBy: prev.query.groupBy?.filter((g) =>
            prev.query.dimensions.some((d) => d.alias === g && d.termNodeId !== termNodeId)
          ) || [],
        },
      };
    });
  };

  const handleRemoveMeasure = (termNodeId: string) => {
    setQueryDef((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        query: {
          ...prev.query,
          measures: prev.query.measures.filter((m) => m.termNodeId !== termNodeId),
        },
      };
    });
  };

  const handleAddFilter = (term: SemanticTermView) => {
    if (!queryDef) return;
    const newFilter: FilterDef = {
      termNodeId: term.termNodeId,
      operator: 'eq',
      value: '',
    };
    setQueryDef((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        query: {
          ...prev.query,
          filters: [...prev.query.filters, newFilter],
        },
      };
    });
  };

  const schemaFieldToTerm = (field: BOSchemaField): SemanticTermView => ({
    termNodeId: field.id,
    termKey: field.name,
    termName: field.name,
    displayName: field.displayName || field.name,
    dataType: field.type,
    role: ['measure', 'calculated'].includes((field.type || '').toLowerCase()) ? 'MEASURE' : 'DIMENSION',
    bindingStatus: 'RESOLVED',
    defaultAggregation: field.aggregation as any,
  });

  const handleAddSchemaField = (field: BOSchemaField) => {
    handleAddTerm(schemaFieldToTerm(field));
  };

  const handleAddSchemaFilter = (field: BOSchemaField) => {
    handleAddFilter(schemaFieldToTerm(field));
  };

  const handleUpdateFilter = (index: number, patch: Partial<FilterDef>) => {
    setQueryDef((prev) => {
      if (!prev) return prev;
      const filters = [...prev.query.filters];
      filters[index] = { ...filters[index], ...patch };
      return { ...prev, query: { ...prev.query, filters } };
    });
  };

  const handleRemoveFilter = (index: number) => {
    setQueryDef((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        query: {
          ...prev.query,
          filters: prev.query.filters.filter((_, i) => i !== index),
        },
      };
    });
  };

  const handleMeasureAggChange = (termNodeId: string, agg: string) => {
    setQueryDef((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        query: {
          ...prev.query,
          measures: prev.query.measures.map((m) =>
            m.termNodeId === termNodeId ? { ...m, agg: agg as any } : m
          ),
        },
      };
    });
  };

  const handleLimitChange = (limit: number) => {
    setQueryDef((prev) => {
      if (!prev) return prev;
      return { ...prev, query: { ...prev.query, limit } };
    });
  };

  const handleDimensionDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id && queryDef) {
      const items = queryDef.query.dimensions.map((d) => d.termNodeId);
      const oldIndex = items.indexOf(active.id as string);
      const newIndex = items.indexOf(over.id as string);
      const reordered = arrayMove(queryDef.query.dimensions, oldIndex, newIndex);
      setQueryDef({ ...queryDef, query: { ...queryDef.query, dimensions: reordered } });
    }
  };

  const handleMeasureDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id && queryDef) {
      const items = queryDef.query.measures.map((m) => m.termNodeId);
      const oldIndex = items.indexOf(active.id as string);
      const newIndex = items.indexOf(over.id as string);
      const reordered = arrayMove(queryDef.query.measures, oldIndex, newIndex);
      setQueryDef({ ...queryDef, query: { ...queryDef.query, measures: reordered } });
    }
  };

  const handleRunQuery = async () => {
    if (!queryDef) return;
    setLoading(true);
    setError(null);
    setActiveTab(0);
    try {
      const result = await executeQuery(queryDef);
      setExecuteResult(result);
      setPreviewResult((prev) =>
        prev ? { ...prev, sql: result.sql } : { sql: result.sql }
      );
    } catch (err) {
      devError('Query execution failed', err);
      setError(err instanceof Error ? err.message : 'Failed to execute query');
      setExecuteResult(null);
    } finally {
      setLoading(false);
    }
  };

  // ─── Parameters ──────────────────────────────────────────────────────────
  // A filter's value can be deferred to one of these instead of a literal
  // (FilterDef.paramRef) - see handleToggleFilterParam below.
  const handleAddParameter = () => {
    setParameters((prev) => {
      let n = prev.length + 1;
      while (prev.some((p) => p.name === `param${n}`)) n++;
      return [...prev, { name: `param${n}`, label: '', type: 'string', required: false }];
    });
  };
  const handleUpdateParameter = (index: number, patch: Partial<SavedQueryParameter>) => {
    setParameters((prev) => prev.map((p, i) => (i === index ? { ...p, ...patch } : p)));
  };
  const handleRemoveParameter = (index: number) => {
    const removed = parameters[index];
    setParameters((prev) => prev.filter((_, i) => i !== index));
    // A filter referencing a deleted parameter would silently never match -
    // detach it back to an editable literal instead.
    if (removed && queryDef) {
      setQueryDef((prev) => prev && {
        ...prev,
        query: { ...prev.query, filters: prev.query.filters.map((f) => (f.paramRef === removed.name ? { ...f, paramRef: undefined, value: '' } : f)) },
      });
    }
  };
  // Toggling a filter to "parameterized" clears its literal value and picks
  // (or creates) a parameter for it to reference; toggling off does the
  // reverse. Kept as one function since the two states are mutually
  // exclusive on FilterDef (paramRef vs value).
  const handleToggleFilterParam = (index: number) => {
    const filter = queryDef?.query.filters[index];
    if (!filter) return;
    if (filter.paramRef) {
      handleUpdateFilter(index, { paramRef: undefined, value: '' });
      return;
    }
    let param = parameters[0];
    if (!param) {
      param = { name: 'param1', label: '', type: 'string', required: false };
      setParameters([param]);
    }
    handleUpdateFilter(index, { paramRef: param.name, value: undefined });
  };

  // ─── Save / load / run as a saved query ─────────────────────────────────
  const currentSavedQueryState = (): SavedQueryState | null => {
    if (!queryDef) return null;
    return {
      dimensions: queryDef.query.dimensions,
      measures: queryDef.query.measures,
      filters: queryDef.query.filters,
      parameters,
      limit: queryDef.query.limit,
    };
  };

  const handleSaveQuery = async () => {
    const state = currentSavedQueryState();
    if (!state || !selectedBO || !savedQueryName.trim()) return;
    setSaving(true);
    setSaveError(null);
    try {
      const input = {
        name: savedQueryName.trim(),
        description: savedQueryDescription,
        boId: selectedBO.id,
        bindingId: selectedBindingId,
        chartType,
        state,
      };
      const saved = savedQueryId ? await updateSavedQuery(savedQueryId, input) : await createSavedQuery(input);
      setSavedQueryId(saved.id);
      refreshSavedQueries();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save query');
    } finally {
      setSaving(false);
    }
  };

  const handleLoadSavedQuery = async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const sq = await getSavedQuery(id);
      const bo = businessObjects.find((b) => b.id === sq.boId);
      if (bo) setSelectedBO(bo);
      setSelectedBindingId(sq.bindingId);
      // The BO/binding effect above resets queryDef/parameters as soon as
      // selectedBO/selectedBindingId change, so populate the actual saved
      // shape on the next tick once that reset has happened.
      setTimeout(() => {
        setQueryDef((prev) => prev && {
          ...prev,
          query: { ...prev.query, dimensions: sq.state.dimensions, measures: sq.state.measures, filters: sq.state.filters, limit: sq.state.limit || prev.query.limit },
        });
        setParameters(sq.state.parameters || []);
        setChartType(sq.chartType);
        setSavedQueryId(sq.id);
        setSavedQueryName(sq.name);
        setSavedQueryDescription(sq.description);
      }, 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load saved query');
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteSavedQueryClick = async (id: string) => {
    try {
      await deleteSavedQuery(id);
      if (savedQueryId === id) setSavedQueryId(null);
      refreshSavedQueries();
    } catch (err) {
      devError('Failed to delete saved query', err);
    }
  };

  // Derived state
  const visibleTerms = useMemo(() => {
    const q = searchTerm.toLowerCase();
    return terms.filter(
      (t) =>
        t.displayName.toLowerCase().includes(q) ||
        t.termName.toLowerCase().includes(q) ||
        t.termKey.toLowerCase().includes(q)
    );
  }, [terms, searchTerm]);

  const groupedTerms = useMemo(() => {
    const groups: Record<SemanticTermView['role'], SemanticTermView[]> = {
      DIMENSION: [],
      MEASURE: [],
      CALCULATED: [],
    };
    visibleTerms.forEach((t) => groups[t.role].push(t));
    return groups;
  }, [visibleTerms]);

  const isInQuery = (termNodeId: string, role: SemanticTermView['role']) => {
    if (!queryDef) return false;
    if (role === 'MEASURE' || role === 'CALCULATED') {
      return queryDef.query.measures.some((m) => m.termNodeId === termNodeId);
    }
    return queryDef.query.dimensions.some((d) => d.termNodeId === termNodeId);
  };

  const isSchemaFieldInQuery = (fieldId: string) => {
    if (!queryDef) return false;
    return (
      queryDef.query.dimensions.some((d) => d.termNodeId === fieldId) ||
      queryDef.query.measures.some((m) => m.termNodeId === fieldId)
    );
  };

  const termById = useMemo(() => {
    const map = new Map<string, SemanticTermView>();
    terms.forEach((t) => map.set(t.termNodeId, t));
    return map;
  }, [terms]);

  const isRunDisabled = !queryDef || (!queryDef.query.dimensions.length && !queryDef.query.measures.length);

  return (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 64px)', bgcolor: '#f5f5f5' }}>
      {/* ── Left Sidebar ── */}
      <Paper
        sx={{
          width: 340,
          display: 'flex',
          flexDirection: 'column',
          borderRight: '1px solid #ddd',
          borderRadius: 0,
        }}
      >
        {/* BO Selector */}
        <Box sx={{ p: 2, borderBottom: '1px solid #eee' }}>
          <Typography variant="overline" color="text.secondary">
            Subject Area
          </Typography>
          <TextField
            select
            fullWidth
            size="small"
            value={selectedBO?.id || ''}
            onChange={(e) => handleSelectBO(e.target.value)}
            SelectProps={{ native: true }}
            inputProps={{ 'aria-label': 'Subject Area' }}
            sx={{ mt: 1 }}
          >
            <option value="" disabled>
              Select Business Object...
            </option>
            {businessObjects.map((bo) => (
              <option key={bo.id} value={bo.id}>
                {bo.display_name}
              </option>
            ))}
          </TextField>
        </Box>

        {/* Binding Selector */}
        {selectedBO && bindings.length > 0 && (
          <Box sx={{ p: 2, borderBottom: '1px solid #eee' }}>
            <Typography variant="overline" color="text.secondary">
              Binding
            </Typography>
            <TextField
              select
              fullWidth
              size="small"
              value={selectedBindingId}
              onChange={(e) => setSelectedBindingId(e.target.value)}
              SelectProps={{ native: true }}
              inputProps={{ 'aria-label': 'Binding' }}
              sx={{ mt: 1 }}
            >
              {bindings.map((b) => (
                <option key={b.bindingId} value={b.bindingId}>
                  {b.bindingName} ({b.backendName})
                </option>
              ))}
            </TextField>
          </Box>
        )}

        {/* Saved Queries - reusable, parameterized QueryDefs already exposed
            as their own REST endpoint (see savedQueryApi.ts) and pickable
            from Page Studio's Chart/Slicer/KPI "use a saved query" binding. */}
        {selectedBO && (
          <Box sx={{ p: 2, borderBottom: '1px solid #eee' }}>
            <Typography variant="overline" color="text.secondary" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <OpenSavedIcon fontSize="small" /> Saved Queries
            </Typography>
            {savedQueries.length === 0 ? (
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                None yet for this Business Object.
              </Typography>
            ) : (
              <List dense disablePadding sx={{ mt: 0.5 }}>
                {savedQueries.map((sq) => (
                  <ListItemButton
                    key={sq.id}
                    selected={sq.id === savedQueryId}
                    onClick={() => handleLoadSavedQuery(sq.id)}
                    sx={{ borderRadius: 1 }}
                  >
                    <ListItemText
                      primary={sq.name}
                      secondary={`${sq.state.dimensions.length}d · ${sq.state.measures.length}m · ${sq.state.parameters.length} params`}
                      primaryTypographyProps={{ variant: 'body2' }}
                      secondaryTypographyProps={{ variant: 'caption' }}
                    />
                    <IconButton size="small" onClick={(e) => { e.stopPropagation(); handleDeleteSavedQueryClick(sq.id); }}>
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </ListItemButton>
                ))}
              </List>
            )}
          </Box>
        )}

        {/* Related Business Objects */}
        {selectedBO && relatedBOs.length > 0 && (
          <Box sx={{ p: 2, borderBottom: '1px solid #eee' }}>
            <Typography variant="overline" color="text.secondary">
              Related Business Objects
            </Typography>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
              Click to join into this query and browse its fields
            </Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
              {relatedBOs.map((rel) => {
                const included = !!includedRelatedBOs[rel.targetObjectId];
                const isMany = cardinalityHintFromString(rel.cardinality) === 'many';
                return (
                  <Tooltip
                    key={rel.id}
                    title={`${rel.relationshipType} (${rel.cardinality})${rel.joinCondition ? ` — ${rel.joinCondition}` : ''}${
                      isMany ? ' — one-to-many: fields will render as a child scroll area' : ''
                    }`}
                  >
                    <Chip
                      size="small"
                      icon={isMany ? <PlanIcon fontSize="small" /> : undefined}
                      label={rel.relatedObjectName}
                      onClick={() => handleToggleRelatedBO(rel)}
                      color={included ? 'primary' : 'default'}
                      variant={included ? 'filled' : 'outlined'}
                      disabled={relatedTermsLoading === rel.targetObjectId}
                    />
                  </Tooltip>
                );
              })}
            </Box>
          </Box>
        )}

        {/* Fields from joined related Business Objects */}
        {Object.entries(includedRelatedBOs).map(([boId, rel]) => (
          <Box key={boId} sx={{ p: 2, borderBottom: '1px solid #eee' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <Typography variant="overline" color="text.secondary">
                {rel.name} fields
              </Typography>
              {rel.cardinalityHint === 'many' && (
                <Tooltip title="One-to-many: added fields render as a child scroll area under each row">
                  <Chip size="small" label="many" color="warning" sx={{ height: 18, fontSize: '0.6rem' }} />
                </Tooltip>
              )}
            </Box>
            <List dense disablePadding sx={{ mt: 0.5 }}>
              {rel.terms.map((term) => {
                const selected = [...(queryDef?.query.dimensions || []), ...(queryDef?.query.measures || [])].some(
                  (f) => f.termNodeId === term.termNodeId && f.boId === boId
                );
                return (
                  <ListItemButton
                    key={term.termNodeId}
                    selected={selected}
                    sx={{ borderRadius: 1, py: 0.25 }}
                    onClick={() => handleAddTerm(term, boId)}
                    disabled={selected}
                  >
                    <ListItemIcon sx={{ minWidth: 28 }}>
                      <FieldIcon type={term.dataType} />
                    </ListItemIcon>
                    <ListItemText
                      primary={term.displayName}
                      secondary={term.termKey}
                      primaryTypographyProps={{ variant: 'body2' }}
                      secondaryTypographyProps={{ variant: 'caption', sx: { fontSize: '0.65rem' } }}
                    />
                  </ListItemButton>
                );
              })}
            </List>
          </Box>
        ))}

        {/* Search */}
        {selectedBO && (
          <Box sx={{ p: 2, pb: 1 }}>
            <TextField
              fullWidth
              size="small"
              placeholder="Search terms..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon fontSize="small" />
                  </InputAdornment>
                ),
              }}
            />
          </Box>
        )}

        {/* Term List */}
        <List dense sx={{ flex: 1, overflow: 'auto', px: 1 }}>
          {loading && !selectedBO && (
            <Box sx={{ p: 2, textAlign: 'center' }}>
              <CircularProgress size={20} />
            </Box>
          )}

          {selectedBO && terms.length === 0 && !loading && (
            <Box sx={{ p: 2, textAlign: 'center', color: 'text.secondary' }}>
              <Typography variant="body2">No resolved terms for this binding</Typography>
            </Box>
          )}

          {(['DIMENSION', 'MEASURE', 'CALCULATED'] as const).map((role) =>
            groupedTerms[role].length > 0 ? (
              <React.Fragment key={role}>
                <Typography
                  variant="caption"
                  sx={{
                    px: 2,
                    pt: 1,
                    pb: 0.5,
                    display: 'block',
                    fontWeight: 700,
                    color: 'text.secondary',
                    textTransform: 'uppercase',
                  }}
                >
                  {role}s
                </Typography>
                {groupedTerms[role].map((term) => {
                  const selected = isInQuery(term.termNodeId, role);
                  return (
                    <ListItemButton
                      key={term.termNodeId}
                      selected={selected}
                      sx={{ borderRadius: 1, mb: 0.5 }}
                    >
                      <ListItemIcon sx={{ minWidth: 32 }}>
                        <FieldIcon type={term.dataType} />
                      </ListItemIcon>
                      <ListItemText
                        primary={
                          <Box sx={{ display: 'flex', alignItems: 'center' }}>
                            {term.displayName}
                            <RoleBadge role={term.role} />
                          </Box>
                        }
                        secondary={term.termKey}
                        primaryTypographyProps={{ variant: 'body2' }}
                        secondaryTypographyProps={{ variant: 'caption', sx: { fontSize: '0.65rem' } }}
                      />
                      <Tooltip title="Add to query">
                        <IconButton
                          size="small"
                          onClick={() => handleAddTerm(term)}
                          disabled={selected}
                        >
                          <AddIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Add as filter">
                        <IconButton size="small" onClick={() => handleAddFilter(term)}>
                          <FilterIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      {selected && (
                        <CheckCircleIcon
                          fontSize="small"
                          color="primary"
                          sx={{ ml: 0.5 }}
                        />
                      )}
                    </ListItemButton>
                  );
                })}
              </React.Fragment>
            ) : null
          )}
        </List>

        {boSchema && (
          <Box sx={{ borderTop: '1px solid #eee', p: 2, flex: 1, overflow: 'auto', minHeight: 200 }}>
            <Typography variant="overline" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
              Meta-API Schema
            </Typography>
            <AutoFormRenderer
              schema={boSchema}
              onAddField={handleAddSchemaField}
              onAddFilter={handleAddSchemaFilter}
              isInQuery={isSchemaFieldInQuery}
            />
          </Box>
        )}
      </Paper>

      {/* ── Center Canvas ── */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', p: 2, overflow: 'hidden' }}>
        {/* Header */}
        <Paper
          sx={{
            p: 2,
            mb: 2,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Box>
            <Typography variant="h6">Alpha Query Builder</Typography>
            <Typography variant="caption" color="text.secondary">
              {queryDef
                ? `${queryDef.query.dimensions.length} dimensions · ${queryDef.query.measures.length} measures · ${queryDef.query.filters.length} filters`
                : 'Select a Business Object and binding to begin'}
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <TextField
              label="Limit"
              type="number"
              size="small"
              value={queryDef?.query.limit || ''}
              onChange={(e) => handleLimitChange(Math.max(1, parseInt(e.target.value, 10) || 0))}
              sx={{ width: 100 }}
              inputProps={{ min: 1 }}
            />
            <Button
              variant="contained"
              color="primary"
              startIcon={<RunIcon />}
              onClick={handleRunQuery}
              disabled={isRunDisabled || loading}
            >
              Run Query
            </Button>
            <Button
              variant="outlined"
              startIcon={<SaveIcon />}
              onClick={() => setSaveDialogOpen(true)}
              disabled={isRunDisabled}
            >
              {savedQueryId ? 'Update Saved Query' : 'Save Query'}
            </Button>
          </Box>
        </Paper>

        {/* Dimensions */}
        {queryDef && (
          <Paper sx={{ p: 2, mb: 2 }}>
            <Typography
              variant="subtitle2"
              color="text.secondary"
              sx={{ mb: 1, display: 'flex', alignItems: 'center', gap: 0.5 }}
            >
              <TableIcon fontSize="small" /> Dimensions
            </Typography>
            {queryDef.query.dimensions.length === 0 ? (
              <Typography variant="body2" color="text.secondary" fontStyle="italic">
                Click a dimension term from the sidebar to add it here.
              </Typography>
            ) : (
              <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                onDragEnd={handleDimensionDragEnd}
              >
                <SortableContext
                  items={queryDef.query.dimensions.map((d) => d.termNodeId)}
                  strategy={horizontalListSortingStrategy}
                >
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                    {queryDef.query.dimensions.map((d) => (
                      <SortableChip
                        key={d.termNodeId}
                        id={d.termNodeId}
                        label={
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                            {termById.get(d.termNodeId)?.displayName || d.alias}
                            <Typography variant="caption" color="text.secondary">
                              AS {d.alias}
                            </Typography>
                          </Box>
                        }
                        onDelete={() => handleRemoveDimension(d.termNodeId)}
                        color="primary"
                      />
                    ))}
                  </Box>
                </SortableContext>
              </DndContext>
            )}
          </Paper>
        )}

        {/* Measures */}
        {queryDef && (
          <Paper sx={{ p: 2, mb: 2 }}>
            <Typography
              variant="subtitle2"
              color="text.secondary"
              sx={{ mb: 1, display: 'flex', alignItems: 'center', gap: 0.5 }}
            >
              <FunctionsIcon fontSize="small" /> Measures
            </Typography>
            {queryDef.query.measures.length === 0 ? (
              <Typography variant="body2" color="text.secondary" fontStyle="italic">
                Click a measure or calculated term from the sidebar to add it here.
              </Typography>
            ) : (
              <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                onDragEnd={handleMeasureDragEnd}
              >
                <SortableContext
                  items={queryDef.query.measures.map((m) => m.termNodeId)}
                  strategy={horizontalListSortingStrategy}
                >
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, alignItems: 'center' }}>
                    {queryDef.query.measures.map((m) => (
                      <Box key={m.termNodeId} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                        <FormControl size="small" sx={{ minWidth: 90 }}>
                          <Select
                            value={m.agg}
                            onChange={(e) => handleMeasureAggChange(m.termNodeId, e.target.value)}
                            variant="standard"
                          >
                            <MenuItem value="SUM">SUM</MenuItem>
                            <MenuItem value="AVG">AVG</MenuItem>
                            <MenuItem value="MIN">MIN</MenuItem>
                            <MenuItem value="MAX">MAX</MenuItem>
                            <MenuItem value="COUNT">COUNT</MenuItem>
                            <MenuItem value="COUNT_DISTINCT">COUNT DISTINCT</MenuItem>
                            <MenuItem value="NONE">NONE</MenuItem>
                          </Select>
                        </FormControl>
                        <SortableChip
                          id={m.termNodeId}
                          label={
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                              {termById.get(m.termNodeId)?.displayName || m.alias}
                              <Typography variant="caption" color="text.secondary">
                                AS {m.alias}
                              </Typography>
                            </Box>
                          }
                          onDelete={() => handleRemoveMeasure(m.termNodeId)}
                          color="secondary"
                        />
                      </Box>
                    ))}
                  </Box>
                </SortableContext>
              </DndContext>
            )}
          </Paper>
        )}

        {/* Filters */}
        {queryDef && queryDef.query.filters.length > 0 && (
          <Paper sx={{ p: 2, mb: 2 }}>
            <Typography
              variant="subtitle2"
              color="text.secondary"
              sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 0.5 }}
            >
              <FilterIcon fontSize="small" /> Filters
            </Typography>
            <Stack spacing={2}>
              {queryDef.query.filters.map((filter, index) => {
                const term = termById.get(filter.termNodeId);
                return (
                  <Box
                    key={`${filter.termNodeId}-${index}`}
                    sx={{ display: 'flex', alignItems: 'center', gap: 1 }}
                  >
                    <Chip
                      label={term?.displayName || filter.termNodeId}
                      size="small"
                      variant="outlined"
                    />
                    <FormControl size="small" sx={{ minWidth: 120 }}>
                      <InputLabel id={`op-label-${index}`}>Operator</InputLabel>
                      <Select
                        labelId={`op-label-${index}`}
                        value={filter.operator}
                        label="Operator"
                        onChange={(e) =>
                          handleUpdateFilter(index, { operator: e.target.value as FilterOperator })
                        }
                      >
                        <MenuItem value="eq">=</MenuItem>
                        <MenuItem value="neq">!=</MenuItem>
                        <MenuItem value="gt">&gt;</MenuItem>
                        <MenuItem value="gte">&gt;=</MenuItem>
                        <MenuItem value="lt">&lt;</MenuItem>
                        <MenuItem value="lte">&lt;=</MenuItem>
                        <MenuItem value="contains">contains</MenuItem>
                        <MenuItem value="starts_with">starts with</MenuItem>
                        <MenuItem value="ends_with">ends with</MenuItem>
                        <MenuItem value="in">in</MenuItem>
                        <MenuItem value="not_in">not in</MenuItem>
                        <MenuItem value="is_null">is null</MenuItem>
                        <MenuItem value="is_not_null">is not null</MenuItem>
                        <MenuItem value="between">between</MenuItem>
                      </Select>
                    </FormControl>
                    {!['is_null', 'is_not_null'].includes(filter.operator) && (
                      filter.paramRef ? (
                        <FormControl size="small" sx={{ flex: 1 }}>
                          <Select
                            value={filter.paramRef}
                            onChange={(e) => handleUpdateFilter(index, { paramRef: e.target.value })}
                          >
                            {parameters.map((p) => (
                              <MenuItem key={p.name} value={p.name}>{p.label || p.name}</MenuItem>
                            ))}
                          </Select>
                        </FormControl>
                      ) : (
                        <TextField
                          size="small"
                          placeholder="Value"
                          value={filter.value ?? ''}
                          onChange={(e) => handleUpdateFilter(index, { value: e.target.value })}
                          sx={{ flex: 1 }}
                        />
                      )
                    )}
                    {!['is_null', 'is_not_null'].includes(filter.operator) && (
                      <Tooltip title={filter.paramRef ? 'Using a parameter - click to use a fixed value instead' : 'Use a parameter instead of a fixed value, so this filter can vary per request (REST caller or Page Studio binding)'}>
                        <IconButton size="small" onClick={() => handleToggleFilterParam(index)} color={filter.paramRef ? 'primary' : 'default'}>
                          <FunctionsIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    )}
                    <IconButton size="small" onClick={() => handleRemoveFilter(index)}>
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Box>
                );
              })}
            </Stack>
          </Paper>
        )}

        {/* Parameters - named placeholders a filter above can defer its
            value to (click the ƒ icon on a filter) instead of a literal. */}
        {queryDef && (
          <Paper sx={{ p: 2, mb: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: parameters.length ? 2 : 0 }}>
              <Typography variant="subtitle2" color="text.secondary" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <FunctionsIcon fontSize="small" /> Parameters
              </Typography>
              <Button size="small" startIcon={<AddIcon />} onClick={handleAddParameter}>Add parameter</Button>
            </Box>
            {parameters.length > 0 && (
              <Stack spacing={1.5}>
                {parameters.map((p, i) => (
                  <Box key={i} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <TextField
                      size="small" label="Name" value={p.name}
                      onChange={(e) => handleUpdateParameter(i, { name: e.target.value })}
                      sx={{ width: 140 }}
                    />
                    <TextField
                      size="small" label="Label" value={p.label || ''}
                      onChange={(e) => handleUpdateParameter(i, { label: e.target.value })}
                      sx={{ flex: 1 }}
                    />
                    <FormControl size="small" sx={{ width: 110 }}>
                      <InputLabel>Type</InputLabel>
                      <Select value={p.type || 'string'} label="Type" onChange={(e) => handleUpdateParameter(i, { type: e.target.value as SavedQueryParameter['type'] })}>
                        <MenuItem value="string">string</MenuItem>
                        <MenuItem value="number">number</MenuItem>
                        <MenuItem value="date">date</MenuItem>
                        <MenuItem value="boolean">boolean</MenuItem>
                      </Select>
                    </FormControl>
                    <TextField
                      size="small" label="Default" value={p.default ?? ''}
                      onChange={(e) => handleUpdateParameter(i, { default: e.target.value })}
                      sx={{ width: 120 }}
                    />
                    <FormControlLabel
                      control={<Switch size="small" checked={!!p.required} onChange={(e) => handleUpdateParameter(i, { required: e.target.checked })} />}
                      label={<Typography variant="caption">Required</Typography>}
                    />
                    <IconButton size="small" onClick={() => handleRemoveParameter(i)}>
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Box>
                ))}
              </Stack>
            )}
          </Paper>
        )}

        {/* Results */}
        <Paper sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
            <Tabs
              value={activeTab}
              onChange={(_, v) => setActiveTab(v)}
              textColor="primary"
              indicatorColor="primary"
            >
              <Tab icon={<TableIcon fontSize="small" />} label="Results" iconPosition="start" />
              <Tab icon={<CodeIcon fontSize="small" />} label="SQL" iconPosition="start" />
              <Tab icon={<PlanIcon fontSize="small" />} label="Plan" iconPosition="start" />
              <Tab icon={<ApiIcon fontSize="small" />} label="QueryDef" iconPosition="start" />
              <Tab icon={<ChartTabIcon fontSize="small" />} label="Chart" iconPosition="start" />
            </Tabs>
          </Box>

          <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <QueryEngineIndicator
              state={
                queryDef?.query?.filters?.some(f => String(f.value).includes('2025') || String(f.value).includes('2024'))
                  ? 'HYBRID_SEAM'
                  : 'OLAP'
              }
              estimatedLatencyMs={executeResult ? 240 : 180}
              estimatedRows={executeResult ? executeResult.rows.length : 847}
            />
          </Box>

          <Box sx={{ flex: 1, overflow: 'auto', p: 0, position: 'relative' }}>
            {loading && (
              <Box
                sx={{
                  position: 'absolute',
                  inset: 0,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  bgcolor: 'rgba(255,255,255,0.7)',
                  zIndex: 10,
                }}
              >
                <CircularProgress />
              </Box>
            )}

            {error && (
              <Box sx={{ p: 2 }}>
                <Alert severity="error">{error}</Alert>
              </Box>
            )}

            {/* Results tab */}
            {activeTab === 0 && (
              <Box sx={{ p: 0, height: '100%' }}>
                {executeResult && executeResult.rows.length > 0 ? (
                  <ScrollAreaResultView columns={executeResult.columns} rows={executeResult.rows} />
                ) : (
                  <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
                    <Typography>No results to display. Build a query and run it.</Typography>
                  </Box>
                )}
              </Box>
            )}

            {/* SQL tab */}
            {activeTab === 1 && (
              <Box sx={{ height: '100%', bgcolor: '#1e1e1e', fontSize: '13px' }}>
                <SyntaxHighlighter
                  language="sql"
                  style={vscDarkPlus}
                  customStyle={{
                    margin: 0,
                    height: '100%',
                    padding: '16px',
                    background: 'transparent',
                  }}
                >
                  {previewResult?.sql || '-- No SQL generated yet'}
                </SyntaxHighlighter>
              </Box>
            )}

            {/* Plan tab */}
            {activeTab === 2 && (
              <Box sx={{ height: '100%', width: '100%', overflow: 'auto', p: 2 }}>
                {previewResult?.plan ? (
                  <>
                    <QueryPerformanceSummary plan={previewResult.plan} />
                    <Box sx={{ height: 500, width: '100%' }}>
                      <ExplainPlanVisualizer plan={previewResult.plan} />
                    </Box>
                  </>
                ) : (
                  <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
                    <Typography>No explain plan available.</Typography>
                  </Box>
                )}
              </Box>
            )}

            {/* QueryDef tab */}
            {activeTab === 3 && (
              <Box sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column', gap: 2, overflow: 'auto' }}>
                <Box sx={{ flexShrink: 0 }}>
                  <JWTInspector />
                </Box>
                <SyntaxHighlighter
                  language="json"
                  style={vscDarkPlus}
                  customStyle={{ borderRadius: '4px', flex: 1, minHeight: 0 }}
                >
                  {queryDef ? JSON.stringify(queryDef, null, 2) : '// Select a Business Object first'}
                </SyntaxHighlighter>
              </Box>
            )}

            {/* Chart tab - the same visual a Page Studio Chart widget would
                render for this query once saved (see PageComponentRenderer's
                "use a saved query" mode), previewed here against whatever
                the last Run Query returned. */}
            {activeTab === 4 && (
              <Box sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column', gap: 2 }}>
                {!executeResult || executeResult.rows.length === 0 ? (
                  <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
                    <Typography>Run the query to preview a visual.</Typography>
                  </Box>
                ) : (
                  <>
                    <Stack direction="row" spacing={2}>
                      <FormControl size="small" sx={{ minWidth: 140 }}>
                        <InputLabel>Visual type</InputLabel>
                        <Select value={chartType} label="Visual type" onChange={(e) => setChartType(e.target.value as SavedQueryChartType)}>
                          <MenuItem value="bar">Bar</MenuItem>
                          <MenuItem value="line">Line</MenuItem>
                          <MenuItem value="pie">Pie</MenuItem>
                        </Select>
                      </FormControl>
                      <FormControl size="small" sx={{ minWidth: 160 }}>
                        <InputLabel>Category</InputLabel>
                        <Select value={chartDimTerm} label="Category" onChange={(e) => setChartDimTerm(e.target.value)}>
                          {queryDef?.query.dimensions.map((d) => (
                            <MenuItem key={d.termNodeId} value={d.alias}>{termById.get(d.termNodeId)?.displayName || d.alias}</MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                      <FormControl size="small" sx={{ minWidth: 160 }}>
                        <InputLabel>Value</InputLabel>
                        <Select value={chartMeasureTerm} label="Value" onChange={(e) => setChartMeasureTerm(e.target.value)}>
                          {queryDef?.query.measures.map((m) => (
                            <MenuItem key={m.termNodeId} value={m.alias}>{termById.get(m.termNodeId)?.displayName || m.alias}</MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                    </Stack>
                    <Box sx={{ flex: 1, minHeight: 0 }}>
                      <ReactECharts style={{ height: '100%', width: '100%' }} option={buildChartOption(executeResult.rows, executeResult.columns, chartType, chartDimTerm, chartMeasureTerm)} notMerge />
                    </Box>
                  </>
                )}
              </Box>
            )}
          </Box>

          {executeResult && (
            <Box
              sx={{
                p: 1,
                borderTop: '1px solid #eee',
                display: 'flex',
                alignItems: 'center',
                gap: 2,
                color: 'text.secondary',
              }}
            >
              <SpeedIcon fontSize="small" />
              <Typography variant="caption">
                {executeResult.rowCount ?? executeResult.rows.length} rows ·{' '}
                {executeResult.executionTimeMs ?? '-'} ms
              </Typography>
            </Box>
          )}
        </Paper>
      </Box>

      {/* Save Query dialog - name/description + the parameters already
          defined below in the Filters section, then (once saved) the
          stable REST endpoint a Page Studio widget or external caller
          hits to re-run this exact query. */}
      <Dialog open={saveDialogOpen} onClose={() => setSaveDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{savedQueryId ? 'Update Saved Query' : 'Save Query'}</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            {saveError && <Alert severity="error">{saveError}</Alert>}
            <TextField
              label="Name"
              size="small"
              fullWidth
              value={savedQueryName}
              onChange={(e) => setSavedQueryName(e.target.value)}
              autoFocus
            />
            <TextField
              label="Description"
              size="small"
              fullWidth
              multiline
              minRows={2}
              value={savedQueryDescription}
              onChange={(e) => setSavedQueryDescription(e.target.value)}
            />
            {parameters.length > 0 && (
              <Alert severity="info" sx={{ fontSize: '0.8rem' }}>
                {parameters.length} parameter{parameters.length > 1 ? 's' : ''} will be part of this
                query's REST endpoint - a caller (or a Page Studio widget) supplies them as
                <code> ?paramName=value</code>.
              </Alert>
            )}
            {savedQueryId && (
              <Box>
                <Typography variant="caption" color="text.secondary">REST endpoint</Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <TextField
                    size="small"
                    fullWidth
                    value={savedQueryRestUrl(savedQueryId)}
                    InputProps={{ readOnly: true, sx: { fontFamily: 'monospace', fontSize: '0.8rem' } }}
                  />
                  <IconButton size="small" onClick={() => navigator.clipboard.writeText(savedQueryRestUrl(savedQueryId))}>
                    <CopyIcon fontSize="small" />
                  </IconButton>
                </Box>
                {parameters.length > 0 && (
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5, fontFamily: 'monospace' }}>
                    ?{parameters.map((p) => `${p.name}=...`).join('&')}
                  </Typography>
                )}
              </Box>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSaveDialogOpen(false)}>Close</Button>
          <Button variant="contained" onClick={handleSaveQuery} disabled={!savedQueryName.trim() || saving}>
            {saving ? 'Saving…' : savedQueryId ? 'Update' : 'Save'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default BusinessObjectQueryBuilder;
