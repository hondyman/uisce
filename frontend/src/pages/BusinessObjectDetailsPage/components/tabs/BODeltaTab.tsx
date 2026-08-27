import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Chip,
  Button,
  Card,
  CardContent,
  CardHeader,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  CircularProgress,
  Alert,
  Grid,
  IconButton,
  TextField,
  InputAdornment,
  Divider,
} from '@mui/material';
import {
  CompareArrows as CompareIcon,
  Search as SearchIcon,
  Refresh as RefreshIcon,
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  AddCircle as AddCircleIcon,
  RemoveCircle as RemoveCircleIcon,
  AutoAwesome as AIIcon,
  Handyman as FixIcon,
  VerifiedUser as VerifiedIcon,
  Troubleshoot as RadarIcon,
  FactCheck as ChecksumIcon,
} from '@mui/icons-material';
import { useTenant } from '../../../../contexts/TenantContext';
import { fetchAPI } from '../../../../api';
import apiClient from '../../../../utils/apiClient';
import { useNotification } from '../../../../hooks/useNotification';

interface BODeltaTabProps {
  businessObject: any;
  selectedSubtypeKey?: string | null;
}

interface DeltaField {
  fieldKey: string;
  fieldName: string;
  status: 'INHERITED' | 'OVERRIDDEN' | 'CUSTOM_ADDED' | 'CUSTOM_REMOVED';
  coreField?: any;
  customField?: any;
  overrides?: Record<string, any>;
}

interface DeltaResponse {
  boId: string;
  key: string;
  name: string;
  displayName: string;
  isCore: boolean;
  coreId?: string;
  fieldsDelta: DeltaField[];
  inheritedCount: number;
  overriddenCount: number;
  customCount: number;
}

export function BODeltaTab({ businessObject, selectedSubtypeKey }: BODeltaTabProps) {
  const { tenant } = useTenant();
  const tenantId = tenant?.id || '';
  const notification = useNotification();

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [delta, setDelta] = useState<DeltaResponse | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>('ALL');
  const [search, setSearch] = useState('');

  // Subtype Scope
  const [activeDeltaSubtype, setActiveDeltaSubtype] = useState<string>(selectedSubtypeKey || 'all');

  useEffect(() => {
    setActiveDeltaSubtype(selectedSubtypeKey || 'all');
  }, [selectedSubtypeKey]);

  // Scoped Counts for Summary KPIs
  const scopedCounts = useMemo(() => {
    if (!delta) return { total: 0, inherited: 0, overridden: 0, custom: 0 };
    const fields = activeDeltaSubtype === 'all'
      ? delta.fieldsDelta
      : delta.fieldsDelta.filter(f => {
          const stFields = businessObject?.subtypes?.[activeDeltaSubtype]?.subtypeFields || [];
          const stKeys = stFields.map((sf: any) => sf.key || sf.technicalName || sf.name);
          return stKeys.includes(f.fieldKey) || stKeys.includes(f.fieldName);
        });
    return {
      total: fields.length,
      inherited: fields.filter(f => f.status === 'INHERITED').length,
      overridden: fields.filter(f => f.status === 'OVERRIDDEN').length,
      custom: fields.filter(f => f.status === 'CUSTOM_ADDED').length,
    };
  }, [delta, activeDeltaSubtype, businessObject]);

  // AI Delta Explainer State
  const [aiLoading, setAiLoading] = useState(false);
  const [aiExplanation, setAiExplanation] = useState<any>(null);

  // Pillar 1 & 5 Sentinel & Drift Maker-Checker State
  const [sentinelLoading, setSentinelLoading] = useState(false);
  const [sentinelData, setSentinelData] = useState<any>(null);
  const [patchActionLoading, setPatchActionLoading] = useState<string | null>(null);

  const boId = businessObject?.id || businessObject?.key;

  const fetchDelta = useCallback(async () => {
    if (!boId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await apiClient<DeltaResponse>(
        `/business-objects/${encodeURIComponent(boId)}/delta`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
          },
        }
      );
      setDelta(res);
    } catch (err: any) {
      console.warn('Could not fetch delta, creating local comparison', err);
      // Fallback local comparison
      const coreFields = businessObject.coreFields || [];
      const customFields = businessObject.customFields || [];
      const allKeys = Array.from(new Set([
        ...coreFields.map((f: any) => f.name || f.key),
        ...customFields.map((f: any) => f.name || f.key),
      ]));

      const localDeltaFields: DeltaField[] = allKeys.map(k => {
        const cf = coreFields.find((f: any) => (f.name || f.key) === k);
        const tf = customFields.find((f: any) => (f.name || f.key) === k);
        if (cf && tf) return { fieldKey: k, fieldName: k, status: 'OVERRIDDEN', coreField: cf, customField: tf };
        if (cf) return { fieldKey: k, fieldName: k, status: 'INHERITED', coreField: cf };
        return { fieldKey: k, fieldName: k, status: 'CUSTOM_ADDED', customField: tf };
      });

      setDelta({
        boId: businessObject.id,
        key: businessObject.key,
        name: businessObject.name,
        displayName: businessObject.displayName,
        isCore: businessObject.isCore,
        fieldsDelta: localDeltaFields,
        inheritedCount: localDeltaFields.filter(f => f.status === 'INHERITED').length,
        overriddenCount: localDeltaFields.filter(f => f.status === 'OVERRIDDEN').length,
        customCount: localDeltaFields.filter(f => f.status === 'CUSTOM_ADDED').length,
      });
    } finally {
      setLoading(false);
    }
  }, [boId, tenantId, businessObject]);

  const fetchSentinel = useCallback(async () => {
    if (!boId) return;
    setSentinelLoading(true);
    try {
      const resp = await fetchAPI<any>(`/business-objects/${boId}/drift-sentinel`);
      if (resp) setSentinelData(resp);
    } catch (err) {
      console.error('Failed to fetch sentinel data', err);
    } finally {
      setSentinelLoading(false);
    }
  }, [boId]);

  useEffect(() => {
    fetchDelta();
    fetchSentinel();
  }, [fetchDelta, fetchSentinel]);

  const handleExplainWithAI = async () => {
    if (!boId) return;
    setAiLoading(true);
    try {
      const resp = await fetchAPI<any>(`/business-objects/${boId}/ai/explain-delta`, {
        method: 'POST',
      });
      setAiExplanation(resp);
      notification.success('AI Delta Assessment generated!');
    } catch (err: any) {
      notification.error(err?.message || 'Failed to generate AI Delta assessment.');
    } finally {
      setAiLoading(false);
    }
  };

  const handleApplyPatch = async (proposalId: string, action: 'APPROVE' | 'REJECT') => {
    if (!boId) return;
    setPatchActionLoading(proposalId);
    try {
      const resp = await fetchAPI<any>(`/business-objects/${boId}/drift-patch`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          proposalId,
          action,
        }),
      });
      notification.success(resp.message || `Patch proposal ${action.toLowerCase()}d successfully.`);
      fetchSentinel();
      fetchDelta();
    } catch (err: any) {
      notification.error(err?.message || 'Failed to process patch action.');
    } finally {
      setPatchActionLoading(null);
    }
  };

  const filteredFields = (delta?.fieldsDelta || []).filter(f => {
    if (filterStatus !== 'ALL' && f.status !== filterStatus) return false;
    if (activeDeltaSubtype !== 'all') {
      const subtypeFields = businessObject?.subtypes?.[activeDeltaSubtype]?.subtypeFields || [];
      const subtypeFieldKeys = subtypeFields.map((sf: any) => sf.key || sf.technicalName || sf.name);
      if (!subtypeFieldKeys.includes(f.fieldKey) && !subtypeFieldKeys.includes(f.fieldName)) {
        return false;
      }
    }
    if (search.trim()) {
      const q = search.toLowerCase();
      return f.fieldName.toLowerCase().includes(q) || f.fieldKey.toLowerCase().includes(q);
    }
    return true;
  });

  return (
    <Box sx={{ p: 3 }}>
      {/* Header Bar */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <CompareIcon color="primary" />
            <Typography variant="h6" sx={{ fontWeight: 700 }}>
              Workday Core vs. Custom Delta & Autonomous Drift Sentinel
            </Typography>
          </Stack>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            Audit differences against the immutable Gold Copy Core baseline (`100.84.50.65`), review self-healing schema drift proposals, and verify financial ISO checksums.
          </Typography>
        </Box>

        <Stack direction="row" spacing={1.5}>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={() => {
              fetchDelta();
              fetchSentinel();
            }}
            disabled={loading}
          >
            Refresh
          </Button>
          <Button
            variant="contained"
            color="secondary"
            startIcon={aiLoading ? <CircularProgress size={18} color="inherit" /> : <AIIcon />}
            onClick={handleExplainWithAI}
            disabled={aiLoading || !delta}
          >
            AI Delta Assessment
          </Button>
        </Stack>
      </Stack>

      {/* Subtype Scope Selector */}
      {Object.keys(businessObject?.subtypes || {}).length > 0 && (
        <Stack direction="row" spacing={1} flexWrap="wrap" alignItems="center" sx={{ mb: 2.5 }}>
          <Typography variant="caption" sx={{ fontWeight: 700, mr: 1, color: 'text.secondary', textTransform: 'uppercase' }}>
            Subtype Lens:
          </Typography>
          <Chip
            label="All Fields"
            size="small"
            variant={activeDeltaSubtype === 'all' ? 'filled' : 'outlined'}
            color={activeDeltaSubtype === 'all' ? 'primary' : 'default'}
            onClick={() => setActiveDeltaSubtype('all')}
          />
          {Object.entries(businessObject.subtypes).map(([key, st]: any) => (
            <Chip
              key={key}
              label={st.displayName || key}
              size="small"
              variant={activeDeltaSubtype === key ? 'filled' : 'outlined'}
              color={activeDeltaSubtype === key ? 'primary' : 'default'}
              onClick={() => setActiveDeltaSubtype(key)}
            />
          ))}
        </Stack>
      )}

      {/* PILLAR 1: Schema Drift Maker-Checker Inbox Card */}
      {sentinelData?.driftProposals && sentinelData.driftProposals.length > 0 && (
        <Card variant="outlined" sx={{ mb: 3, borderColor: 'warning.light', bgcolor: 'warning.50' }}>
          <CardHeader
            avatar={<FixIcon color="warning" />}
            title={<Typography variant="subtitle2" sx={{ fontWeight: 700 }}>Autonomous Schema Drift Maker-Checker Inbox</Typography>}
            subheader="Debezium CDC detected upstream column changes. High-confidence (>95%) pgvector semantic repair candidate patches available."
          />
          <Divider />
          <CardContent sx={{ p: 2 }}>
            <Stack spacing={2}>
              {sentinelData.driftProposals.map((dp: any, idx: number) => (
                <Paper key={dp.proposalId || idx} variant="outlined" sx={{ p: 2, bgcolor: 'background.paper' }}>
                  <Grid container spacing={2} alignItems="center">
                    <Grid size={{ xs: 12, md: 8 }}>
                      <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.5 }}>
                        <Chip label={dp.driftType} size="small" color="warning" sx={{ fontWeight: 700 }} />
                        <Typography variant="body2" sx={{ fontWeight: 700 }}>
                          Column Renamed: <span style={{ textDecoration: 'line-through' }}>{dp.sourceColumn}</span> ➔ <strong>{dp.targetColumn}</strong>
                        </Typography>
                        <Chip label={`Cosine Match: ${(dp.confidenceScore * 100).toFixed(1)}%`} size="small" color="success" variant="outlined" />
                      </Stack>
                      <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace', display: 'block', mt: 0.5 }}>
                        Auto-Repair SQL: {dp.autoRepairScript}
                      </Typography>
                    </Grid>
                    <Grid size={{ xs: 12, md: 4 }} sx={{ textAlign: { md: 'right' } }}>
                      <Stack direction="row" spacing={1} justifyContent={{ md: 'flex-end' }}>
                        <Button
                          size="small"
                          variant="contained"
                          color="success"
                          startIcon={<CheckCircleIcon />}
                          disabled={patchActionLoading === dp.proposalId}
                          onClick={() => handleApplyPatch(dp.proposalId, 'APPROVE')}
                        >
                          Approve Patch
                        </Button>
                        <Button
                          size="small"
                          variant="outlined"
                          color="error"
                          disabled={patchActionLoading === dp.proposalId}
                          onClick={() => handleApplyPatch(dp.proposalId, 'REJECT')}
                        >
                          Reject
                        </Button>
                      </Stack>
                    </Grid>
                  </Grid>
                </Paper>
              ))}
            </Stack>
          </CardContent>
        </Card>
      )}

      {/* PILLAR 5: Passive Anomaly Radar & Financial ISO Checksum Sentinel Card */}
      {sentinelData && (
        <Card variant="outlined" sx={{ mb: 3 }}>
          <CardHeader
            avatar={<RadarIcon color="primary" />}
            title={<Typography variant="subtitle2" sx={{ fontWeight: 700 }}>Passive Anomaly Radar & Financial ISO Checksum Sentinel</Typography>}
            subheader={`${sentinelData.sentinelSummary} (Overall Quality Score: ${sentinelData.overallQualityScore}/100)`}
          />
          <Divider />
          <CardContent sx={{ p: 2 }}>
            <Grid container spacing={2}>
              {(sentinelData.financialVerifications || []).map((fv: any, idx: number) => (
                <Grid size={{ xs: 12, sm: 4 }} key={idx}>
                  <Paper variant="outlined" sx={{ p: 1.5 }}>
                    <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                      <Typography variant="caption" sx={{ fontWeight: 700 }}>{fv.patternType}</Typography>
                      <Chip
                        icon={<ChecksumIcon />}
                        label={`${fv.passRate}% Valid`}
                        size="small"
                        color={fv.passRate === 100 ? 'success' : 'warning'}
                        sx={{ fontWeight: 700, fontSize: '0.7rem' }}
                      />
                    </Stack>
                    <Typography variant="caption" color="text.secondary">
                      Sampled: {fv.sampleCount} rows | Errors: {fv.invalidCount}
                    </Typography>
                  </Paper>
                </Grid>
              ))}
            </Grid>
          </CardContent>
        </Card>
      )}

      {/* AI Delta Explainer Assessment Card */}
      {aiExplanation && (
        <Card variant="outlined" sx={{ mb: 3, bgcolor: 'secondary.50', borderColor: 'secondary.light' }}>
          <CardHeader
            avatar={<AIIcon color="secondary" />}
            title={<Typography variant="subtitle2" sx={{ fontWeight: 700 }}>AI Executive Delta Assessment</Typography>}
            subheader={`Impact Level: ${aiExplanation.impactLevel}`}
            action={
              <Chip
                label={`Impact: ${aiExplanation.impactLevel}`}
                color={aiExplanation.impactLevel === 'HIGH' ? 'error' : aiExplanation.impactLevel === 'MEDIUM' ? 'warning' : 'success'}
                size="small"
                sx={{ fontWeight: 700 }}
              />
            }
          />
          <Divider />
          <CardContent sx={{ p: 2 }}>
            <Typography variant="body2" sx={{ mb: 2 }}>
              {aiExplanation.summary}
            </Typography>

            {aiExplanation.breakingChanges && aiExplanation.breakingChanges.length > 0 && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="error.main" sx={{ fontWeight: 700, display: 'block', mb: 0.5 }}>
                  Breaking Changes:
                </Typography>
                <Stack spacing={0.5}>
                  {aiExplanation.breakingChanges.map((bc: string, idx: number) => (
                    <Typography key={idx} variant="caption" color="error.dark" sx={{ display: 'block' }}>
                      • {bc}
                    </Typography>
                  ))}
                </Stack>
              </Box>
            )}

            {aiExplanation.recommendations && aiExplanation.recommendations.length > 0 && (
              <Box>
                <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700, display: 'block', mb: 0.5 }}>
                  Recommended Actions:
                </Typography>
                <Stack spacing={0.5}>
                  {aiExplanation.recommendations.map((rec: string, idx: number) => (
                    <Typography key={idx} variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                      ✓ {rec}
                    </Typography>
                  ))}
                </Stack>
              </Box>
            )}
          </CardContent>
        </Card>
      )}

      {/* Summary KPI Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center', bgcolor: filterStatus === 'ALL' ? 'action.selected' : 'background.paper', cursor: 'pointer' }} onClick={() => setFilterStatus('ALL')}>
            <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>TOTAL FIELDS</Typography>
            <Typography variant="h5" sx={{ fontWeight: 800, my: 0.5 }}>{scopedCounts.total}</Typography>
            <Typography variant="caption" color="text.secondary">{activeDeltaSubtype === 'all' ? 'All field definitions' : `${activeDeltaSubtype} scoped`}</Typography>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center', bgcolor: filterStatus === 'INHERITED' ? 'primary.50' : 'background.paper', cursor: 'pointer' }} onClick={() => setFilterStatus('INHERITED')}>
            <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>INHERITED FROM CORE</Typography>
            <Typography variant="h5" sx={{ fontWeight: 800, color: 'primary.main', my: 0.5 }}>{scopedCounts.inherited}</Typography>
            <Typography variant="caption" color="text.secondary">Unmodified Gold Copy</Typography>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center', bgcolor: filterStatus === 'OVERRIDDEN' ? 'warning.50' : 'background.paper', cursor: 'pointer' }} onClick={() => setFilterStatus('OVERRIDDEN')}>
            <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>OVERRIDDEN FIELDS</Typography>
            <Typography variant="h5" sx={{ fontWeight: 800, color: 'warning.main', my: 0.5 }}>{scopedCounts.overridden}</Typography>
            <Typography variant="caption" color="text.secondary">Customized properties</Typography>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center', bgcolor: filterStatus === 'CUSTOM_ADDED' ? 'success.50' : 'background.paper', cursor: 'pointer' }} onClick={() => setFilterStatus('CUSTOM_ADDED')}>
            <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>CUSTOM EXTENSIONS</Typography>
            <Typography variant="h5" sx={{ fontWeight: 800, color: 'success.main', my: 0.5 }}>{scopedCounts.custom}</Typography>
            <Typography variant="caption" color="text.secondary">Tenant specific additions</Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* STI Partial Index Callout */}
      {activeDeltaSubtype !== 'all' && (
        <Box sx={{ mb: 3, p: 2, bgcolor: 'grey.900', borderRadius: 1.5, fontFamily: 'monospace', fontSize: '0.8rem', color: 'primary.light', border: '1px solid', borderColor: 'primary.dark' }}>
          <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.5, fontFamily: 'inherit', fontWeight: 700, letterSpacing: 0.5 }}>
            STI PARTIAL DISCRIMINATOR INDEX
          </Typography>
          {`CREATE INDEX CONCURRENTLY idx_${businessObject?.key || 'bo'}_${activeDeltaSubtype}_subtype`}<br />
          {`  ON public.${businessObject?.driverTableName || businessObject?.technicalName || 'data_table'} (subtype_code)`}<br />
          {`  WHERE subtype_code = '${activeDeltaSubtype}';`}
        </Box>
      )}

      {/* Filter and Search Bar */}
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mb: 2 }} alignItems="center" justifyContent="space-between">
        <TextField
          size="small"
          placeholder="Filter fields by name..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" />
                </InputAdornment>
              ),
            },
          }}
          sx={{ width: { xs: '100%', sm: 280 } }}
        />
        <Typography variant="caption" color="text.secondary">
          Showing {filteredFields.length} of {delta?.fieldsDelta.length || 0} fields
        </Typography>
      </Stack>

      {/* Delta Grid Table */}
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow sx={{ bgcolor: 'action.hover' }}>
              <TableCell sx={{ fontWeight: 700 }}>Field Name</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>Delta Status</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>Core Baseline Definition</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>Tenant Custom Definition</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>Differences</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredFields.map((f, idx) => (
              <TableRow key={f.fieldKey || idx} hover>
                <TableCell sx={{ fontWeight: 600 }}>{f.fieldName}</TableCell>
                <TableCell>
                  <Chip
                    label={f.status}
                    size="small"
                    color={f.status === 'INHERITED' ? 'primary' : f.status === 'OVERRIDDEN' ? 'warning' : f.status === 'CUSTOM_ADDED' ? 'success' : 'error'}
                    sx={{ fontSize: '0.65rem', height: 20 }}
                  />
                </TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                  {f.coreField ? `${f.coreField.type || 'string'} (${f.coreField.role || 'ATTRIBUTE'})` : '—'}
                </TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                  {f.customField ? `${f.customField.type || 'string'} (${f.customField.role || 'ATTRIBUTE'})` : '—'}
                </TableCell>
                <TableCell sx={{ fontSize: '0.75rem', color: 'text.secondary' }}>
                  {f.status === 'OVERRIDDEN' ? 'Custom formula or validation override applied' : f.status === 'CUSTOM_ADDED' ? 'Tenant extended field' : 'Exact match with Gold Copy'}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}
