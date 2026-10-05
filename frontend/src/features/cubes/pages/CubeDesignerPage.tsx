import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Checkbox,
  Chip,
  CircularProgress,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Tab,
  Tabs,
  TextField,
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';
import RefreshIcon from '@mui/icons-material/Refresh';
import SaveIcon from '@mui/icons-material/Save';
import SpeedIcon from '@mui/icons-material/Speed';
import VerifiedIcon from '@mui/icons-material/Verified';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import { useLocale } from '../../../i18n/useLocale';
import {
  listBusinessObjects,
  fetchBusinessObjectBindings,
  fetchBOTerms,
  type BusinessObjectOption,
} from '../../../studio-core/binding/businessObjectApi';
import type { SemanticTermView } from '../../../features/query-builder/types/queryDef';
import {
  createCube,
  deployCube,
  getCube,
  listCubeMetrics,
  patchCube,
  refreshCube,
  validateCube,
  type CubeMaterializeStartResponse,
} from '../cubeDefinitionApi';
import type {
  CubeDefinition,
  CubeDraft,
  CubeMetricOption,
  CubeValidateResponse,
} from '../types';
import { federationIsActive } from '../types';
import {
  cubeDraftFromDefinition,
  cubeDraftPayload,
  emptyCubeDraft,
} from '../draft';
import FederationEditor from '../FederationEditor';

type TabKey =
  | 'overview'
  | 'dimensions'
  | 'metrics'
  | 'grains'
  | 'federation'
  | 'materialization'
  | 'versions';

const emptyDraft = emptyCubeDraft;
const draftFromCube = cubeDraftFromDefinition;
const draftPayload = cubeDraftPayload;

const CubeDesignerPage: React.FC = () => {
  const locale = useLocale();
  const navigate = useNavigate();
  const { cubeId } = useParams<{ cubeId: string }>();
  const isNew = cubeId === 'new' || !cubeId;

  const [tab, setTab] = useState<TabKey>('overview');
  const [draft, setDraft] = useState<CubeDraft>(emptyDraft);
  const [saved, setSaved] = useState<CubeDefinition | null>(null);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [validating, setValidating] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [info, setInfo] = useState<string | null>(null);
  const [validation, setValidation] = useState<CubeValidateResponse | null>(null);

  const [bos, setBos] = useState<BusinessObjectOption[]>([]);
  const [terms, setTerms] = useState<SemanticTermView[]>([]);
  const [metrics, setMetrics] = useState<CubeMetricOption[]>([]);
  const [termsLoading, setTermsLoading] = useState(false);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [dimInput, setDimInput] = useState('');

  useEffect(() => {
    listBusinessObjects()
      .then(setBos)
      .catch(() => setBos([]));
  }, []);

  useEffect(() => {
    if (isNew || !cubeId) {
      setLoading(false);
      setSaved(null);
      setDraft(emptyDraft());
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    getCube(cubeId)
      .then((c) => {
        if (cancelled) return;
        setSaved(c);
        setDraft(draftFromCube(c));
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [cubeId, isNew]);

  const loadTerms = useCallback(async (boKey: string) => {
    if (!boKey) {
      setTerms([]);
      return;
    }
    setTermsLoading(true);
    try {
      const bindings = await fetchBusinessObjectBindings(boKey);
      const preferred =
        bindings.find((b) => b.isDefault) || bindings[0];
      if (!preferred?.bindingId) {
        setTerms([]);
        return;
      }
      const t = await fetchBOTerms(boKey, preferred.bindingId);
      setTerms(t.filter((x) => x.role === 'DIMENSION' || !x.role));
    } catch {
      setTerms([]);
    } finally {
      setTermsLoading(false);
    }
  }, []);

  const loadMetrics = useCallback(async (boKey: string) => {
    setMetricsLoading(true);
    try {
      const rows = await listCubeMetrics(boKey || undefined);
      setMetrics(rows);
    } catch {
      setMetrics([]);
    } finally {
      setMetricsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadTerms(draft.boId);
    void loadMetrics(draft.boId);
  }, [draft.boId, loadTerms, loadMetrics]);

  const selectedDimIds = useMemo(
    () => new Set(draft.dimensions.map((d) => d.termNodeId)),
    [draft.dimensions],
  );

  const addDimension = (termNodeId: string) => {
    const id = termNodeId.trim();
    if (!id || selectedDimIds.has(id)) return;
    setDraft((d) => ({
      ...d,
      dimensions: [...d.dimensions, { termNodeId: id }],
    }));
    setDimInput('');
  };

  const removeDimension = (termNodeId: string) => {
    setDraft((d) => ({
      ...d,
      dimensions: d.dimensions.filter((x) => x.termNodeId !== termNodeId),
      grains: d.grains.map((g) => g.filter((m) => m !== termNodeId)).filter((g) => g.length > 0),
    }));
  };

  const toggleMetric = (id: string) => {
    setDraft((d) => ({
      ...d,
      metricIds: d.metricIds.includes(id)
        ? d.metricIds.filter((x) => x !== id)
        : [...d.metricIds, id],
    }));
  };

  const ensurePrimaryGrain = () => {
    const dims = draft.dimensions.map((d) => d.termNodeId);
    if (dims.length === 0) return;
    setDraft((d) => {
      if (d.grains.length > 0) return d;
      return { ...d, grains: [dims] };
    });
  };

  const onSave = async () => {
    setSaving(true);
    setError(null);
    setInfo(null);
    setValidation(null);
    try {
      if (!draft.name.trim() || !draft.boId.trim()) {
        throw new Error('Name and Business Object are required');
      }
      if (draft.metricIds.length === 0) {
        throw new Error('Select at least one governed metric');
      }
      let grains = draft.grains;
      if (grains.length === 0 && draft.dimensions.length > 0) {
        grains = [draft.dimensions.map((d) => d.termNodeId)];
        setDraft((d) => ({ ...d, grains }));
      }
      if (grains.length === 0) {
        throw new Error('Add at least one grain (covered by your dimensions)');
      }
      const body = draftPayload({ ...draft, grains });
      if (isNew || !saved) {
        const created = await createCube(body);
        setSaved(created);
        setDraft(draftFromCube(created));
        setInfo(`Created cube v${created.contractVersion}`);
        navigate(`/${locale}/build/cubes/${created.id}`, { replace: true });
      } else {
        const res = await patchCube(saved.id, body);
        setSaved(res.cube);
        setDraft(draftFromCube(res.cube));
        if (res.noop) {
          setInfo(`No-op save — content hash unchanged (${res.reason || 'identical surface'})`);
        } else {
          setInfo(`Saved cube v${res.cube.contractVersion}`);
        }
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  const onValidate = async () => {
    if (!saved?.id) {
      setError('Save the cube before validate (POST /api/cubes/{id}/validate)');
      return;
    }
    setValidating(true);
    setError(null);
    setInfo(null);
    try {
      const res = await validateCube(saved.id, draftPayload(draft, { includeKeySamples: true }));
      setValidation(res);
      if (res.ok) setInfo('Validate OK — structural, metrics, and federation gates pass');
      else setError('Validate failed — see details below');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setValidating(false);
    }
  };

  const summarizeStarts = (res: CubeMaterializeStartResponse, verb: string) => {
    const running = (res.starts || []).filter((s) => s.already_running);
    const ok = (res.starts || []).filter((s) => s.workflow_id && !s.already_running && !s.error);
    const failed = (res.starts || []).filter((s) => s.error && !s.already_running);
    if (running.length && !ok.length) {
      return {
        kind: 'running' as const,
        message: `Already running — ${running.length} grain(s) in flight (${running.map((s) => s.workflow_id || s.grain_hash).join(', ')})`,
      };
    }
    const parts = [`${verb}: started ${ok.length}`];
    if (running.length) parts.push(`${running.length} already running`);
    if (failed.length) parts.push(`${failed.length} failed`);
    return { kind: failed.length && !ok.length ? ('error' as const) : ('ok' as const), message: parts.join(' · ') };
  };

  const onDeploy = async () => {
    if (!saved?.id) {
      setError('Save the cube before Deploy');
      return;
    }
    setDeploying(true);
    setError(null);
    setInfo(null);
    try {
      const res = await deployCube(saved.id);
      const summary = summarizeStarts(res, 'Deploy');
      if (summary.kind === 'running' || summary.kind === 'error') setError(summary.message);
      else setInfo(summary.message);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      if (/already running|409/i.test(msg)) setError(`Already running — ${msg}`);
      else setError(msg);
    } finally {
      setDeploying(false);
    }
  };

  const onRefresh = async () => {
    if (!saved?.id) {
      setError('Save the cube before Refresh');
      return;
    }
    setRefreshing(true);
    setError(null);
    setInfo(null);
    try {
      const res = await refreshCube(saved.id);
      const summary = summarizeStarts(res, 'Refresh');
      if (summary.kind === 'running' || summary.kind === 'error') setError(summary.message);
      else setInfo(summary.message);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      if (/already running|409/i.test(msg)) setError(`Already running — ${msg}`);
      else setError(msg);
    } finally {
      setRefreshing(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3, maxWidth: 1100, mx: 'auto' }}>
      <Stack direction="row" spacing={1} mb={2} flexWrap="wrap" useFlexGap>
        <Button component={RouterLink} to={`/${locale}/build/cubes`} startIcon={<ArrowBackIcon />} size="small">
          Cubes
        </Button>
        <Button
          component={RouterLink}
          to={`/${locale}/fabric/preaggregations`}
          startIcon={<SpeedIcon />}
          size="small"
          variant="outlined"
        >
          Preaggregations ops
        </Button>
      </Stack>

      <Stack direction={{ xs: 'column', md: 'row' }} justifyContent="space-between" spacing={2} mb={2}>
        <Box>
          <Typography variant="h4" component="h1">
            {isNew ? 'New Cube' : draft.name || 'Cube Designer'}
          </Typography>
          <Stack direction="row" spacing={1} mt={1} flexWrap="wrap" useFlexGap>
            {saved && <Chip label={`v${saved.contractVersion}`} size="small" />}
            {saved && (
              <Chip
                label={saved.status}
                size="small"
                color={saved.status === 'active' ? 'success' : 'default'}
              />
            )}
            {draft.boId && <Chip label={draft.boId} size="small" variant="outlined" />}
            {federationIsActive(draft.federation) && (
              <Chip
                label={`federated · ${draft.federation.sources?.length || 0} sources`}
                size="small"
                color="secondary"
                variant="outlined"
              />
            )}
          </Stack>
        </Box>
        <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
          <Button
            variant="outlined"
            startIcon={<VerifiedIcon />}
            disabled={validating || !saved}
            onClick={() => void onValidate()}
          >
            {validating ? 'Validating…' : 'Validate'}
          </Button>
          <Button
            variant="outlined"
            startIcon={<CloudUploadIcon />}
            disabled={deploying || !saved}
            onClick={() => void onDeploy()}
          >
            {deploying ? 'Deploying…' : 'Deploy'}
          </Button>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            disabled={refreshing || !saved}
            onClick={() => void onRefresh()}
          >
            {refreshing ? 'Refreshing…' : 'Refresh'}
          </Button>
          <Button
            variant="contained"
            startIcon={<SaveIcon />}
            disabled={saving}
            onClick={() => void onSave()}
          >
            {saving ? 'Saving…' : 'Save'}
          </Button>
        </Stack>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}
      {info && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setInfo(null)}>
          {info}
        </Alert>
      )}
      {validation && (
        <Alert severity={validation.ok ? 'success' : 'warning'} sx={{ mb: 2 }}>
          structural={String(validation.structuralOk)}
          {validation.structuralError ? ` (${validation.structuralError})` : ''} · metrics=
          {String(validation.metricsOk)}
          {validation.metricsError ? ` (${validation.metricsError})` : ''}
          {typeof validation.federationOk === 'boolean'
            ? ` · federation=${String(validation.federationOk)}`
            : ''}
          {validation.federationError ? ` (${validation.federationError})` : ''}
          {validation.orphanReport
            ? ` · orphan max=${validation.orphanReport.maxPercent}% observed=${validation.orphanReport.observedMaxPercent}%${
                validation.orphanReport.skipped ? ' (skipped)' : ''
              }`
            : validation.orphanRateMaxPct != null
              ? ` · orphanRateMaxPct=${validation.orphanRateMaxPct}`
              : ''}
          {validation.federationTransforms?.length
            ? ` · transforms=${validation.federationTransforms.length}`
            : ''}
          {validation.breakReasons?.length
            ? ` · breakReasons: ${validation.breakReasons.join(', ')}`
            : ''}
        </Alert>
      )}

      <Paper variant="outlined" sx={{ p: 2 }}>
        <Tabs
          value={tab}
          onChange={(_, v) => setTab(v as TabKey)}
          variant="scrollable"
          scrollButtons="auto"
          sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}
        >
          <Tab label="Overview" value="overview" />
          <Tab label="Dimensions" value="dimensions" />
          <Tab label="Metrics" value="metrics" />
          <Tab label="Grains" value="grains" />
          <Tab
            label={
              federationIsActive(draft.federation)
                ? `Federation (${draft.federation.sources?.length || 0})`
                : 'Federation'
            }
            value="federation"
          />
          <Tab label="Materialization" value="materialization" />
          <Tab label="Versions" value="versions" />
        </Tabs>

        {tab === 'overview' && (
          <Stack spacing={2} maxWidth={560}>
            <TextField
              label="Name"
              required
              value={draft.name}
              onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
              fullWidth
            />
            <TextField
              label="Description"
              value={draft.description}
              onChange={(e) => setDraft((d) => ({ ...d, description: e.target.value }))}
              fullWidth
              multiline
              minRows={2}
            />
            <Autocomplete
              options={bos}
              getOptionLabel={(o) => `${o.displayName || o.name} (${o.key})`}
              value={bos.find((b) => b.key === draft.boId) || null}
              onChange={(_, v) =>
                setDraft((d) => ({
                  ...d,
                  boId: v?.key || '',
                  dimensions: [],
                  metricIds: [],
                  grains: [],
                }))
              }
              renderInput={(params) => (
                <TextField {...params} label="Business Object" required helperText="Logical BO key stored on the cube" />
              )}
            />
            <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
              contentHash {saved?.contentHash || '—'}
            </Typography>
          </Stack>
        )}

        {tab === 'dimensions' && (
          <Stack spacing={2}>
            <Typography variant="body2" color="text.secondary">
              Ordered dimension surface. Each axis must appear in at least one grain.
            </Typography>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
              <Autocomplete
                freeSolo
                fullWidth
                loading={termsLoading}
                options={terms.map((t) => t.termNodeId || t.termKey).filter(Boolean)}
                inputValue={dimInput}
                onInputChange={(_, v) => setDimInput(v)}
                onChange={(_, v) => {
                  if (typeof v === 'string') addDimension(v);
                }}
                renderInput={(params) => (
                  <TextField {...params} label="Add dimension term" placeholder="term node id" />
                )}
              />
              <Button variant="outlined" onClick={() => addDimension(dimInput)}>
                Add
              </Button>
            </Stack>
            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
              {draft.dimensions.map((d, i) => (
                <Chip
                  key={`${d.termNodeId}-${i}`}
                  label={`${i + 1}. ${d.termNodeId}`}
                  onDelete={() => removeDimension(d.termNodeId)}
                />
              ))}
              {draft.dimensions.length === 0 && (
                <Typography variant="body2" color="text.secondary">
                  No dimensions yet.
                </Typography>
              )}
            </Stack>
          </Stack>
        )}

        {tab === 'metrics' && (
          <Stack spacing={1}>
            <Typography variant="body2" color="text.secondary">
              Governed metrics only (data_explorer.metric_definition). Filter follows the selected BO.
            </Typography>
            {metricsLoading ? (
              <CircularProgress size={24} />
            ) : metrics.length === 0 ? (
              <Alert severity="info">
                No active metrics for this BO. Seed or author a metric_definition row first.
              </Alert>
            ) : (
              metrics.map((m) => (
                <FormControlLabel
                  key={m.id}
                  control={
                    <Checkbox
                      checked={draft.metricIds.includes(m.id)}
                      onChange={() => toggleMetric(m.id)}
                    />
                  }
                  label={`${m.name}${m.isCore ? ' (core)' : ''} — ${m.id.slice(0, 8)}…`}
                />
              ))
            )}
          </Stack>
        )}

        {tab === 'grains' && (
          <Stack spacing={2}>
            <Typography variant="body2" color="text.secondary">
              Each grain is one physical materialization shape (subset of dimensions).
            </Typography>
            <Button variant="outlined" onClick={ensurePrimaryGrain} disabled={draft.dimensions.length === 0}>
              Use all dimensions as primary grain
            </Button>
            {draft.grains.map((g, gi) => (
              <Paper key={gi} variant="outlined" sx={{ p: 1.5 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Grain {gi + 1}
                </Typography>
                <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                  {draft.dimensions.map((d) => {
                    const on = g.includes(d.termNodeId);
                    return (
                      <Chip
                        key={d.termNodeId}
                        label={d.termNodeId}
                        color={on ? 'primary' : 'default'}
                        variant={on ? 'filled' : 'outlined'}
                        onClick={() => {
                          setDraft((prev) => {
                            const grains = prev.grains.map((gg, i) => {
                              if (i !== gi) return gg;
                              return on
                                ? gg.filter((x) => x !== d.termNodeId)
                                : [...gg, d.termNodeId];
                            });
                            return { ...prev, grains };
                          });
                        }}
                      />
                    );
                  })}
                </Stack>
                <Button
                  size="small"
                  color="inherit"
                  sx={{ mt: 1 }}
                  onClick={() =>
                    setDraft((prev) => ({
                      ...prev,
                      grains: prev.grains.filter((_, i) => i !== gi),
                    }))
                  }
                >
                  Remove grain
                </Button>
              </Paper>
            ))}
            <Button
              onClick={() =>
                setDraft((d) => ({
                  ...d,
                  grains: [...d.grains, d.dimensions.slice(0, 1).map((x) => x.termNodeId)],
                }))
              }
              disabled={draft.dimensions.length === 0}
            >
              Add grain
            </Button>
          </Stack>
        )}

        {tab === 'federation' && (
          <FederationEditor
            primaryBoId={draft.boId}
            bos={bos}
            federation={draft.federation || {}}
            keySamples={draft.federationKeySamples || []}
            onChange={(federation) => setDraft((d) => ({ ...d, federation }))}
            onKeySamplesChange={(federationKeySamples) =>
              setDraft((d) => ({ ...d, federationKeySamples }))
            }
          />
        )}

        {tab === 'materialization' && (
          <Stack spacing={2} maxWidth={480}>
            <FormControl fullWidth>
              <InputLabel id="mat-strategy">Strategy</InputLabel>
              <Select
                labelId="mat-strategy"
                label="Strategy"
                value={draft.materialization.strategy || 'starrocks_mv'}
                onChange={(e) =>
                  setDraft((d) => ({
                    ...d,
                    materialization: { ...d.materialization, strategy: e.target.value },
                  }))
                }
              >
                <MenuItem value="starrocks_mv">starrocks_mv</MenuItem>
                <MenuItem value="aggregate_table">aggregate_table</MenuItem>
              </Select>
            </FormControl>
            <FormControl fullWidth>
              <InputLabel id="mat-stale">Stale policy</InputLabel>
              <Select
                labelId="mat-stale"
                label="Stale policy"
                value={draft.materialization.stalePolicy || 'serve_with_flag'}
                onChange={(e) =>
                  setDraft((d) => ({
                    ...d,
                    materialization: { ...d.materialization, stalePolicy: e.target.value },
                  }))
                }
              >
                <MenuItem value="serve_with_flag">serve_with_flag</MenuItem>
                <MenuItem value="force_raw_fallback">force_raw_fallback</MenuItem>
              </Select>
            </FormControl>
            <Typography variant="body2" color="text.secondary">
              Hot engine: {draft.materialization.hotEngine || 'starrocks'} · Cold engine:{' '}
              {draft.materialization.coldEngine || 'iceberg'}
            </Typography>
          </Stack>
        )}

        {tab === 'versions' && (
          <Stack spacing={1}>
            <Typography>
              Current contract_version: <strong>{saved?.contractVersion ?? 1}</strong>
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Additive edits use Save (PATCH). Breaking grain/dimension/metric/federation changes
              require POST /api/cubes/{'{id}'}/versions (CUBE-0.2 API).
            </Typography>
          </Stack>
        )}
      </Paper>
    </Box>
  );
};

export default CubeDesignerPage;
