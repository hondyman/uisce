/**
 * Saved Query editor - "just a diluted report builder": pick a primary
 * Business Object + binding, then add/remove related Business Objects and
 * the fields (dimensions/measures) you want from each. The primary BO and
 * binding are chosen once, at creation, and are locked forever after -
 * only the related-object set and field selection stay editable (see
 * backend/internal/querybuilder/saved_query_handler.go's
 * HandleUpdateSavedQuery doc comment).
 *
 * No SQL is ever built here - this page only edits a QueryDef-shaped
 * SavedQueryState; the backend's boresolver package does all SQL
 * generation, join resolution, tenant scoping, and field masking, exactly
 * as it does for an ad-hoc /api/query/execute call.
 *
 * The object/binding/related-object/field picker itself is not
 * reimplemented here - it's the shared BusinessObjectSelectorControl (see
 * components/shared/BusinessObjectSelectorControl.tsx and
 * studio-core/binding/useBusinessObjectSelector.ts), the same control
 * Page Studio's Data Binding panel is built on.
 */
import React, { useMemo, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import {
  Box, Paper, Typography, TextField, Button, Stack,
  CircularProgress, Alert, Breadcrumbs, Link as MuiLink,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import SaveIcon from '@mui/icons-material/Save';
import { useTenant } from '../../../contexts/TenantContext';
import { useNotification } from '../../../hooks/useNotification';
import BusinessObjectSelectorControl from '../../../components/shared/BusinessObjectSelectorControl';
import BusinessObjectExplorerPanel from '../../../components/shared/BusinessObjectExplorerPanel';
import QueryShelves, { type EditableFilter } from '../../../components/shared/QueryShelves';
import QueryResultsPanel from '../../../components/shared/QueryResultsPanel';
import { useBusinessObjectSelector } from '../../../studio-core/binding/useBusinessObjectSelector';
import type { BusinessObjectOption, SemanticTermView } from '../../../studio-core/binding/businessObjectApi';
import { previewQuery } from '../services/queryBuilderApi';
import {
  getSavedQuery, createSavedQuery, updateSavedQuery, runSavedQuery,
} from '../services/savedQueryApi';
import type { SavedQuery, SavedQueryState, QueryDef } from '../types/queryDef';
import { friendlyQueryError, savedQueryResultToSet } from '../query-execution';

function useLoadedSavedQuery(id: string | undefined, isNew: boolean) {
  const [savedQuery, setSavedQuery] = useState<SavedQuery | null>(null);
  const [loading, setLoading] = useState(!isNew);
  const [error, setError] = useState<string | null>(null);

  React.useEffect(() => {
    if (isNew || !id) return;
    setLoading(true);
    getSavedQuery(id)
      .then(setSavedQuery)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [isNew, id]);

  return { savedQuery, setSavedQuery, loading, error, setError };
}

export default function SavedQueryEditor() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const notify = useNotification();

  const isNew = !id || id === 'new';
  const returnTo = searchParams.get('return_to');

  const { savedQuery, setSavedQuery, loading, error: loadError } = useLoadedSavedQuery(id, isNew);

  const [name, setName] = useState('New Query');
  const [description, setDescription] = useState('');
  const [saving, setSaving] = useState(false);
  const [creating, setCreating] = useState(false);
  const [runResult, setRunResult] = useState<{ columns: { name: string }[]; rows: Record<string, unknown>[] } | null>(null);
  const [running, setRunning] = useState(false);
  const [runError, setRunError] = useState<string | null>(null);

  // Hydrate name/description/filters/parameters once the saved query loads.
  const [filters, setFilters] = useState<EditableFilter[]>([]);
  const [parameters, setParameters] = useState<SavedQueryState['parameters']>([]);
  React.useEffect(() => {
    if (savedQuery) {
      setName(savedQuery.name);
      setDescription(savedQuery.description);
      setFilters(savedQuery.state.filters.map((f) => ({
        termNodeId: f.termNodeId,
        boId: f.boId,
        operator: f.operator,
        value: f.value == null ? '' : Array.isArray(f.value) ? f.value.join(', ') : String(f.value),
        paramRef: f.paramRef,
      })));
      setParameters(savedQuery.state.parameters || []);
    }
  }, [savedQuery]);

  const addFilterFromField = (boId: string, term: SemanticTermView) => {
    const primaryBoId = selector.primary?.boId;
    setFilters((prev) => [...prev, {
      termNodeId: term.termNodeId,
      boId: boId === primaryBoId ? undefined : boId,
      operator: 'eq',
      value: '',
    }]);
  };
  const updateFilter = (index: number, patch: Partial<EditableFilter>) => {
    setFilters((prev) => prev.map((f, i) => i === index ? { ...f, ...patch } : f));
  };
  const removeFilter = (index: number) => {
    setFilters((prev) => prev.filter((_, i) => i !== index));
  };
  const addParameter = (param: SavedQueryState['parameters'][number]) => {
    setParameters((prev) => [...prev, param]);
  };
  const removeParameter = (name: string) => {
    setParameters((prev) => prev.filter((p) => p.name !== name));
    // A filter bound to a removed parameter would otherwise silently send
    // an undefined paramRef to the backend - fall back it to a literal
    // (empty) value instead, same as picking "Literal value" in the UI.
    setFilters((prev) => prev.map((f) => f.paramRef === name ? { ...f, paramRef: undefined, value: '' } : f));
  };

  const selectedTermIdsByBO = useMemo(() => {
    if (!savedQuery) return undefined;
    const out: Record<string, string[]> = {};
    const add = (boId: string | undefined, termNodeId: string) => {
      const key = boId || savedQuery.boId;
      (out[key] ||= []).push(termNodeId);
    };
    savedQuery.state.dimensions.forEach((d) => add(d.boId, d.termNodeId));
    savedQuery.state.measures.forEach((m) => add(m.boId, m.termNodeId));
    return out;
  }, [savedQuery]);

  const selector = useBusinessObjectSelector({
    initial: savedQuery ? { boId: savedQuery.boId, bindingId: savedQuery.bindingId, relatedBoIds: savedQuery.relatedBoIds } : undefined,
    hydrationKey: savedQuery?.id,
    initialSelectedTermIds: selectedTermIdsByBO,
    lockPrimaryAfterSelect: true,
  });

  const handleCreate = async (bo: BusinessObjectOption, bindingId: string) => {
    setCreating(true);
    try {
      const sq = await createSavedQuery({
        name: name || bo.displayName || bo.name,
        description,
        boId: bo.id,
        bindingId,
        chartType: 'bar',
        state: { dimensions: [], measures: [], filters: [], parameters: [] },
        tags: [],
      });
      notify.success('Query created');
      navigate(`/query-builder/editor/${sq.id}`, { replace: true });
    } catch (e: any) {
      notify.error(e.message);
    } finally {
      setCreating(false);
    }
  };

  const buildState = (): SavedQueryState => {
    const dimensions: SavedQueryState['dimensions'] = [];
    const measures: SavedQueryState['measures'] = [];
    for (const obj of selector.objects) {
      const boId = obj.isPrimary ? undefined : obj.boId;
      for (const t of obj.terms) {
        if (!t.selected) continue;
        if (t.role === 'MEASURE') {
          measures.push({ termNodeId: t.termNodeId, alias: t.termKey, agg: t.defaultAggregation || 'NONE', boId });
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

  const handleSave = async () => {
    if (!savedQuery) return;
    setSaving(true);
    try {
      const updated = await updateSavedQuery(savedQuery.id, {
        name, description,
        boId: savedQuery.boId, bindingId: savedQuery.bindingId,
        relatedBoIds: selector.related.map((r) => r.boId),
        chartType: savedQuery.chartType,
        state: buildState(),
        tags: savedQuery.tags,
        folderId: savedQuery.folderId,
      });
      setSavedQuery(updated);
      notify.success('Query saved');
    } catch (e: any) {
      notify.error(e.message);
    } finally {
      setSaving(false);
    }
  };

  const handleRun = async () => {
    if (!savedQuery) return;
    setRunning(true);
    setRunResult(null);
    setRunError(null);
    try {
      setRunResult(await runSavedQuery(savedQuery.id));
    } catch (e: any) {
      setRunError(friendlyQueryError(e.message));
    } finally {
      setRunning(false);
    }
  };

  const [activeBoId, setActiveBoId] = useState('');
  React.useEffect(() => {
    if (selector.primary && !activeBoId) setActiveBoId(selector.primary.boId);
  }, [selector.primary, activeBoId]);

  const [compiledSql, setCompiledSql] = useState<{ sql: string; dialect?: string } | null>(null);
  const [sqlLoading, setSqlLoading] = useState(false);
  const [sqlError, setSqlError] = useState<string | null>(null);
  const { tenant } = useTenant();

  const handleCompileSql = async () => {
    if (!savedQuery) return;
    setSqlLoading(true);
    setSqlError(null);
    try {
      const state = buildState();
      const qd: QueryDef = {
        context: { boId: savedQuery.boId, bindingId: savedQuery.bindingId, tenantId: tenant?.id || '', relatedBoIds: selector.related.map((r) => r.boId) },
        query: { dimensions: state.dimensions, measures: state.measures, filters: state.filters, limit: state.limit },
      };
      const result = await previewQuery(qd);
      setCompiledSql(result);
    } catch (e: any) {
      setSqlError(friendlyQueryError(e.message));
    } finally {
      setSqlLoading(false);
    }
  };

  if (loading) {
    return <Box p={4} display="flex" justifyContent="center"><CircularProgress /></Box>;
  }

  // --- Creation flow: pick primary BO + binding, then lock them in ---
  if (isNew) {
    return (
      <Box p={3} maxWidth={640}>
        <Typography variant="h5" gutterBottom>New Query</Typography>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          Choose the primary Business Object and binding. These cannot be changed later —
          repointing a query at a different object is a new query, not an edit.
        </Typography>
        <TextField label="Query name" value={name} onChange={(e) => setName(e.target.value)} fullWidth sx={{ my: 2 }} />
        {creating ? (
          <Box display="flex" justifyContent="center" p={2}><CircularProgress /></Box>
        ) : (
          <BusinessObjectSelectorControl
            selector={selector}
            renderFields={false}
            onSelectPrimary={handleCreate}
          />
        )}
      </Box>
    );
  }

  if (!savedQuery) {
    return <Box p={4}><Alert severity="error">{loadError || 'Query not found'}</Alert></Box>;
  }

  // A fixed height (rather than 100vh) so this page composes safely inside
  // the app shell's own header/scroll container instead of fighting it for
  // viewport real estate - panes 1-3 each scroll internally within this
  // budget, which is what keeps the editor from needing a long page-level
  // scroll the way the pre-redesign version did.
  const explorerHeight = 640;

  return (
    <Box sx={{ p: 1.5, display: 'flex', flexDirection: 'column' }}>
      {returnTo && (
        <Breadcrumbs sx={{ mb: 0.5, flexShrink: 0 }}>
          <MuiLink component="button" onClick={() => navigate(returnTo)} underline="hover" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
            <ArrowBackIcon fontSize="small" /> Back
          </MuiLink>
        </Breadcrumbs>
      )}

      <Stack direction="row" spacing={1.5} sx={{ flex: 1, minHeight: 0 }}>
        {/* Panes 1 + 2: the centralized Business Object explorer */}
        <Box sx={{ height: explorerHeight, flexShrink: 0 }}>
          <BusinessObjectExplorerPanel
            selector={selector} activeBoId={activeBoId} onSelectActive={setActiveBoId}
            onToggleField={selector.toggleField} onAddFilter={addFilterFromField}
          />
        </Box>

        {/* Pane 3: title/actions bar, shelves, results tabs - the only internally-scrolling column */}
        <Box sx={{ flex: 1, minWidth: 0, height: explorerHeight, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          <Paper variant="outlined" sx={{ p: 1.5, flexShrink: 0 }}>
            <Stack direction="row" justifyContent="space-between" alignItems="flex-start" flexWrap="wrap" gap={1}>
              <Box sx={{ minWidth: 0, flex: 1 }}>
                <TextField value={name} onChange={(e) => setName(e.target.value)} variant="standard"
                  sx={{ '& input': { fontSize: '1.35rem', fontWeight: 600 } }} fullWidth />
                <TextField
                  value={description} onChange={(e) => setDescription(e.target.value)}
                  variant="standard" placeholder="Add a description…" fullWidth
                  InputProps={{ disableUnderline: true }}
                  sx={{ '& input': { fontSize: '0.8rem', color: 'text.secondary' } }}
                />
              </Box>
              <Stack direction="row" spacing={1} flexShrink={0}>
                <Button
                  startIcon={running ? <CircularProgress size={14} color="inherit" /> : <PlayArrowIcon />}
                  onClick={handleRun} disabled={running} variant="outlined"
                >
                  {running ? 'Running…' : 'Run'}
                </Button>
                <Button variant="contained" startIcon={<SaveIcon />} onClick={handleSave} disabled={saving}>
                  {saving ? 'Saving…' : 'Save'}
                </Button>
              </Stack>
            </Stack>
          </Paper>

          <Box sx={{ flexShrink: 0 }}>
            <QueryShelves
              selector={selector} filters={filters} onUpdateFilter={updateFilter} onRemoveFilter={removeFilter}
              parameters={parameters} onAddParameter={addParameter} onRemoveParameter={removeParameter}
            />
          </Box>

          <QueryResultsPanel
            resultSet={runResult ? savedQueryResultToSet(runResult) : null}
            running={running} runError={runError}
            sql={compiledSql} sqlLoading={sqlLoading} sqlError={sqlError}
            onRequestCompileSql={handleCompileSql}
            extraTabs={[{
              id: 'execution-plan',
              label: 'Execution Plan',
              content: (
                <Typography variant="body2" color="text.secondary">
                  Execution plan / diagnostics aren't exposed by the query API yet - /api/query/preview returns
                  the compiled SQL only, not an EXPLAIN plan or timing breakdown.
                </Typography>
              ),
            }]}
          />
        </Box>
      </Stack>
    </Box>
  );
}
