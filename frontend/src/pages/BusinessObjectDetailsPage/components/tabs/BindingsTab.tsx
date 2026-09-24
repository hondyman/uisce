import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Button,
  Card,
  CardContent,
  CardHeader,
  Divider,
  CircularProgress,
  Alert,
  Tabs,
  Tab,
  Tooltip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  FormControlLabel,
  Switch,
} from '@mui/material';
import {
  Storage as StorageIcon,
  CheckCircle as ResolvedIcon,
  Warning as UnresolvedIcon,
  AutoAwesome as AIIcon,
  Code as CodeIcon,
  AccountTree as GraphIcon,
  Hub as TierIcon,
  CloudQueue as CloudIcon,
  ContentCopy as CopyIcon,
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import { fetchAPI } from '../../../../api';
import { useNotification } from '../../../../hooks/useNotification';
import { useTenant } from '../../../../contexts/TenantContext';
import {
  fetchPhysicalBackends,
  fetchCatalogTables,
  createBinding,
  updateBinding,
  deleteBinding,
} from '../../../../components/BusinessObjectManager/bindingWizard.service';

const MAX_FIELDS_DISPLAY = 10;

interface BindingsTabProps {
  bindings: any[];
  businessObject?: any;
  // Re-fetches the parent BO (and its bindings array) after an add/edit/
  // delete - the tab has no state of its own for the bindings list, it
  // only ever renders the `bindings` prop, so a mutation needs the parent
  // to reload rather than trying to keep a local copy in sync.
  onBindingsChanged?: () => void | Promise<void>;
}

export function BindingsTab({ bindings, businessObject, onBindingsChanged }: BindingsTabProps) {
  const notification = useNotification();
  const { tenant } = useTenant();
  const [subTab, setSubTab] = useState(0);

  const [loadingScope, setLoadingScope] = useState(false);
  const [scopeData, setScopeData] = useState<any>(null);

  const [multiBackend, setMultiBackend] = useState<any>(null);
  const [artifacts, setArtifacts] = useState<any>(null);
  const [loadingArtifacts, setLoadingArtifacts] = useState(false);

  const boId = businessObject?.id || businessObject?.key;

  // ─── Add / Edit / Delete binding ───────────────────────────────────────
  const [addOpen, setAddOpen] = useState(false);
  const [backends, setBackends] = useState<Array<{ backendId: string; backendName: string; description?: string }>>([]);
  const [loadingBackends, setLoadingBackends] = useState(false);
  const [addBackendId, setAddBackendId] = useState('');
  const [addTables, setAddTables] = useState<Array<{ node_id: string; node_name: string; qualified_path: string }>>([]);
  const [loadingTables, setLoadingTables] = useState(false);
  const [addTableId, setAddTableId] = useState('');
  const [addBindingName, setAddBindingName] = useState('');
  const [addIsDefault, setAddIsDefault] = useState(false);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const [editingBinding, setEditingBinding] = useState<any | null>(null);
  const [editBindingName, setEditBindingName] = useState('');
  const [editIsDefault, setEditIsDefault] = useState(false);
  const [editIsActive, setEditIsActive] = useState(true);

  const [deletingId, setDeletingId] = useState<string | null>(null);

  const openAddDialog = async () => {
    setFormError(null);
    setAddBackendId('');
    setAddTableId('');
    setAddBindingName('');
    setAddIsDefault(bindings.length === 0);
    setAddOpen(true);
    setLoadingBackends(true);
    try {
      const list = await fetchPhysicalBackends();
      setBackends(list);
    } catch (err: any) {
      setFormError(err?.message || 'Failed to load backends');
    } finally {
      setLoadingBackends(false);
    }
  };

  // Driving-table search is scoped to a backend, not a tenant datasource,
  // but in this schema a physical_backend's id doubles as its
  // tenant_product_datasource id (see fix_orphan_orm_backend_datasource.sql's
  // comment for why) - the same id space fetchCatalogTables' datasourceId
  // expects.
  useEffect(() => {
    if (!addOpen || !addBackendId || !tenant?.id) { setAddTables([]); return; }
    let cancelled = false;
    setLoadingTables(true);
    setAddTableId('');
    fetchCatalogTables({ tenantId: tenant.id, datasourceId: addBackendId })
      .then((tables) => { if (!cancelled) setAddTables(tables); })
      .catch((err) => { if (!cancelled) setFormError(err?.message || 'Failed to load tables'); })
      .finally(() => { if (!cancelled) setLoadingTables(false); });
    return () => { cancelled = true; };
  }, [addOpen, addBackendId, tenant?.id]);

  const handleCreateBinding = async () => {
    if (!boId || !addBackendId || !addTableId) return;
    setSaving(true);
    setFormError(null);
    try {
      await createBinding(boId, {
        backendId: addBackendId,
        drivingNodeId: addTableId,
        bindingName: addBindingName || undefined,
        isDefault: addIsDefault,
      });
      notification.success('Binding added');
      setAddOpen(false);
      await onBindingsChanged?.();
    } catch (err: any) {
      setFormError(err?.message || 'Failed to create binding');
    } finally {
      setSaving(false);
    }
  };

  const openEditDialog = (b: any) => {
    setEditingBinding(b);
    setEditBindingName(b.nodeName ? `${b.nodeName} Binding` : '');
    setEditIsDefault(!!b.isDefault);
    setEditIsActive(b.isActive ?? true);
    setFormError(null);
  };

  const handleSaveEdit = async () => {
    if (!boId || !editingBinding?.boBindingId) return;
    setSaving(true);
    setFormError(null);
    try {
      await updateBinding(boId, editingBinding.boBindingId, {
        bindingName: editBindingName || undefined,
        isDefault: editIsDefault,
        isActive: editIsActive,
      });
      notification.success('Binding updated');
      setEditingBinding(null);
      await onBindingsChanged?.();
    } catch (err: any) {
      setFormError(err?.message || 'Failed to update binding');
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteBinding = async (b: any) => {
    if (!boId || !b.boBindingId) return;
    if (!window.confirm(`Delete the "${b.nodeName || b.boBindingId}" binding? This cannot be undone.`)) return;
    setDeletingId(b.boBindingId);
    try {
      await deleteBinding(boId, b.boBindingId);
      notification.success('Binding deleted');
      await onBindingsChanged?.();
    } catch (err: any) {
      notification.error(err?.message || 'Failed to delete binding');
    } finally {
      setDeletingId(null);
    }
  };

  useEffect(() => {
    if (!boId) return;

    const loadData = async () => {
      setLoadingScope(true);
      try {
        const [scopeResp, multiResp, artResp] = await Promise.all([
          fetchAPI<any>(`/business-objects/${boId}/scope`).catch(() => null),
          fetchAPI<any>(`/business-objects/${boId}/multi-backend`).catch(() => null),
          fetchAPI<any>(`/business-objects/${boId}/artifacts`).catch(() => null),
        ]);
        if (scopeResp) setScopeData(scopeResp);
        if (multiResp) setMultiBackend(multiResp);
        if (artResp) setArtifacts(artResp);
      } catch (err) {
        console.error('Failed to load binding details', err);
      } finally {
        setLoadingScope(false);
      }
    };

    loadData();
  }, [boId]);

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    notification.success(`Copied ${label} to clipboard!`);
  };

  return (
    <Box sx={{ p: 3 }}>
      {/* Header Bar */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <StorageIcon color="primary" />
            <Typography variant="h6" sx={{ fontWeight: 700 }}>
              Polymorphic Storage Bindings & Dynamic Scope
            </Typography>
          </Stack>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            Manage physical database mappings across Tier 1 (Postgres OLTP), Tier 2 (StarRocks OLAP), Tier 3 (Iceberg Deep History), and auto-discovered scope fences.
          </Typography>
        </Box>
        <Stack direction="row" spacing={1.5} alignItems="center">
          {scopeData && (
            <Chip
              icon={scopeData.isPublishReady ? <ResolvedIcon /> : <UnresolvedIcon />}
              label={scopeData.isPublishReady ? 'Scope Gate: Resolved' : 'Scope Gate: Blocked'}
              color={scopeData.isPublishReady ? 'success' : 'warning'}
              sx={{ fontWeight: 700 }}
            />
          )}
          <Button variant="contained" size="small" startIcon={<AddIcon />} onClick={openAddDialog}>
            Add Binding
          </Button>
        </Stack>
      </Stack>

      <Tabs value={subTab} onChange={(_, val) => setSubTab(val)} sx={{ mb: 3, borderBottom: 1, borderColor: 'divider' }}>
        <Tab label="Multi-Tier Storage Planes" icon={<TierIcon />} iconPosition="start" />
        <Tab label="Dynamic Scope & Auto-Discovery" icon={<GraphIcon />} iconPosition="start" />
        <Tab label="Zero-Code Artifacts (OpenAPI)" icon={<CodeIcon />} iconPosition="start" />
      </Tabs>

      {/* SUBTAB 0: Multi-Tier Storage Planes */}
      {subTab === 0 && (
        <Stack spacing={3}>
          {multiBackend && (
            <Card variant="outlined" sx={{ bgcolor: 'action.hover' }}>
              <CardContent sx={{ p: 2 }}>
                <Grid container spacing={2} alignItems="center">
                  <Grid size={{ xs: 12, md: 8 }}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                      Hot / Cold Watermark Seam
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Queries automatically route to PostgreSQL / StarRocks for live operational dates, and seam into Apache Iceberg for archival queries ($Date &lt; W_t$).
                    </Typography>
                  </Grid>
                  <Grid size={{ xs: 12, md: 4 }} sx={{ textAlign: { md: 'right' } }}>
                    <Chip
                      label={`Watermark: ${new Date(multiBackend.watermarkDate || Date.now()).toLocaleDateString()}`}
                      color="primary"
                      variant="outlined"
                      sx={{ fontWeight: 700 }}
                    />
                  </Grid>
                </Grid>
              </CardContent>
            </Card>
          )}

          {bindings.length === 0 ? (
            <Alert severity="info">No physical bindings are configured for this business object yet.</Alert>
          ) : (
            <Grid container spacing={2}>
              {bindings.map((b: any, idx: number) => (
                <Grid size={{ xs: 12, md: 4 }} key={b.boBindingId || idx}>
                  <Card variant="outlined" sx={{ height: '100%' }}>
                    <CardHeader
                      avatar={<CloudIcon color="primary" />}
                      title={
                        <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap">
                          <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                            {b.backendType || 'Backend'}
                          </Typography>
                          {b.isDefault && (
                            <Chip label="DEFAULT" size="small" color="primary" sx={{ fontSize: '0.6rem', height: 18 }} />
                          )}
                        </Stack>
                      }
                      subheader={
                        <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                          {b.drivingNodeName || b.nodeName || 'unmapped'}
                        </Typography>
                      }
                      action={
                        <Stack direction="row">
                          <Tooltip title="Edit binding">
                            <IconButton size="small" onClick={() => openEditDialog(b)}>
                              <EditIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title={bindings.length <= 1 ? "Can't delete a BO's only binding" : 'Delete binding'}>
                            <span>
                              <IconButton
                                size="small"
                                color="error"
                                disabled={bindings.length <= 1 || deletingId === b.boBindingId}
                                onClick={() => handleDeleteBinding(b)}
                              >
                                {deletingId === b.boBindingId ? <CircularProgress size={16} /> : <DeleteIcon fontSize="small" />}
                              </IconButton>
                            </span>
                          </Tooltip>
                        </Stack>
                      }
                    />
                    <Divider />
                    <CardContent sx={{ p: 2 }}>
                      <Stack spacing={1}>
                        <Stack direction="row" justifyContent="space-between">
                          <Typography variant="caption" color="text.secondary">Driving Table</Typography>
                          <Typography variant="caption" sx={{ fontWeight: 700 }}>{b.nodeName || '-'}</Typography>
                        </Stack>
                        <Stack direction="row" justifyContent="space-between">
                          <Typography variant="caption" color="text.secondary">Temporal Mode</Typography>
                          <Typography variant="caption" sx={{ fontWeight: 700 }}>{b.temporalOverride || 'NONE'}</Typography>
                        </Stack>
                        <Stack direction="row" justifyContent="space-between">
                          <Typography variant="caption" color="text.secondary">Binding ID</Typography>
                          <Tooltip title={b.boBindingId || ''}>
                            <Typography variant="caption" sx={{ fontFamily: 'monospace', maxWidth: 140, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                              {b.boBindingId || '-'}
                            </Typography>
                          </Tooltip>
                        </Stack>
                      </Stack>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          )}
        </Stack>
      )}

      {/* SUBTAB 1: Dynamic Scope & Auto-Discovery */}
      {subTab === 1 && (
        <Stack spacing={3}>
          {scopeData && (
            <Grid container spacing={2}>
              <Grid size={{ xs: 6, sm: 3 }}>
                <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center' }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>DIRECT TERMS</Typography>
                  <Typography variant="h5" sx={{ fontWeight: 800, color: 'primary.main', my: 0.5 }}>{scopeData.directCount}</Typography>
                  <Typography variant="caption" color="text.secondary">Auto-mapped via columns</Typography>
                </Paper>
              </Grid>
              <Grid size={{ xs: 6, sm: 3 }}>
                <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center' }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>RELATED TERMS</Typography>
                  <Typography variant="h5" sx={{ fontWeight: 800, color: 'info.main', my: 0.5 }}>{scopeData.relatedCount}</Typography>
                  <Typography variant="caption" color="text.secondary">Traversed via foreign keys</Typography>
                </Paper>
              </Grid>
              <Grid size={{ xs: 6, sm: 3 }}>
                <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center' }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>CALCULATED TERMS</Typography>
                  <Typography variant="h5" sx={{ fontWeight: 800, color: 'secondary.main', my: 0.5 }}>{scopeData.calculatedCount}</Typography>
                  <Typography variant="caption" color="text.secondary">USES_INPUT dependency tree</Typography>
                </Paper>
              </Grid>
              <Grid size={{ xs: 6, sm: 3 }}>
                <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center' }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>MANUAL TERMS</Typography>
                  <Typography variant="h5" sx={{ fontWeight: 800, my: 0.5 }}>{scopeData.manualCount}</Typography>
                  <Typography variant="caption" color="text.secondary">Explicit user injections</Typography>
                </Paper>
              </Grid>
            </Grid>
          )}

          {/* Scope Fields Table */}
          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow sx={{ bgcolor: 'action.hover' }}>
                  <TableCell sx={{ fontWeight: 700 }}>Field Name</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Eligibility Level</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Resolution Path</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Physical Column</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Status</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {(scopeData?.eligibleFields || []).map((ef: any, idx: number) => (
                  <TableRow key={idx} hover>
                    <TableCell sx={{ fontWeight: 600 }}>{ef.displayName || ef.fieldName}</TableCell>
                    <TableCell>
                      <Chip
                        label={ef.eligibilityLevel}
                        size="small"
                        color={ef.eligibilityLevel === 'DIRECT' ? 'primary' : ef.eligibilityLevel === 'RELATED' ? 'info' : ef.eligibilityLevel === 'CALCULATED' ? 'secondary' : 'default'}
                        sx={{ fontSize: '0.65rem', height: 20 }}
                      />
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'text.secondary' }}>
                      {ef.resolutionPath}
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                      {ef.physicalColumn}
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={ef.resolutionStatus}
                        size="small"
                        color={ef.resolutionStatus === 'RESOLVED' ? 'success' : 'error'}
                        variant="outlined"
                        sx={{ fontSize: '0.65rem', height: 20 }}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Stack>
      )}

      {/* SUBTAB 2: Zero-Code Generated Artifacts */}
      {subTab === 2 && (
        <Stack spacing={3}>
          {artifacts && (
            <>
              {/* OpenAPI 3.0 */}
              <Card variant="outlined">
                <CardHeader
                  title={<Typography variant="subtitle2" sx={{ fontWeight: 700 }}>OpenAPI 3.0 REST Specification</Typography>}
                  subheader={`Endpoint: ${artifacts.restEndpointUrl}`}
                  action={
                    <Button
                      size="small"
                      startIcon={<CopyIcon />}
                      onClick={() => copyToClipboard(artifacts.openApiSpecJson, 'OpenAPI Spec')}
                    >
                      Copy JSON
                    </Button>
                  }
                />
                <Divider />
                <CardContent sx={{ p: 2, bgcolor: 'action.hover' }}>
                  <Typography variant="body2" component="pre" sx={{ fontFamily: 'monospace', fontSize: '0.75rem', m: 0, overflowX: 'auto' }}>
                    {artifacts.openApiSpecJson}
                  </Typography>
                </CardContent>
              </Card>

              {/* StarRocks Materialized View */}
              <Card variant="outlined">
                <CardHeader
                  title={<Typography variant="subtitle2" sx={{ fontWeight: 700 }}>StarRocks Materialized View DDL</Typography>}
                  action={
                    <Button
                      size="small"
                      startIcon={<CopyIcon />}
                      onClick={() => copyToClipboard(artifacts.starRocksMvDdl, 'StarRocks DDL')}
                    >
                      Copy SQL
                    </Button>
                  }
                />
                <Divider />
                <CardContent sx={{ p: 2, bgcolor: 'action.hover' }}>
                  <Typography variant="body2" component="pre" sx={{ fontFamily: 'monospace', fontSize: '0.75rem', m: 0, overflowX: 'auto' }}>
                    {artifacts.starRocksMvDdl}
                  </Typography>
                </CardContent>
              </Card>
            </>
          )}
        </Stack>
      )}

      {/* Add Binding dialog */}
      <Dialog open={addOpen} onClose={() => !saving && setAddOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Add Binding</DialogTitle>
        <DialogContent>
          <Stack spacing={2.5} sx={{ mt: 1 }}>
            {formError && <Alert severity="error" onClose={() => setFormError(null)}>{formError}</Alert>}
            <FormControl fullWidth size="small">
              <InputLabel id="add-backend-label">Backend</InputLabel>
              <Select
                labelId="add-backend-label"
                label="Backend"
                value={addBackendId}
                onChange={(e) => setAddBackendId(e.target.value)}
                disabled={loadingBackends}
              >
                {backends.map((b) => (
                  <MenuItem key={b.backendId} value={b.backendId}>
                    {b.backendName}{b.description ? ` — ${b.description}` : ''}
                  </MenuItem>
                ))}
              </Select>
              {loadingBackends && (
                <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
                  <CircularProgress size={14} />
                  <Typography variant="caption" color="text.secondary">Loading backends…</Typography>
                </Stack>
              )}
            </FormControl>

            <FormControl fullWidth size="small" disabled={!addBackendId || loadingTables}>
              <InputLabel id="add-table-label">Driving Table</InputLabel>
              <Select
                labelId="add-table-label"
                label="Driving Table"
                value={addTableId}
                onChange={(e) => setAddTableId(e.target.value)}
              >
                {addTables.map((t) => (
                  <MenuItem key={t.node_id} value={t.node_id}>
                    {t.qualified_path || t.node_name}
                  </MenuItem>
                ))}
              </Select>
              {loadingTables && (
                <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
                  <CircularProgress size={14} />
                  <Typography variant="caption" color="text.secondary">Loading tables…</Typography>
                </Stack>
              )}
              {!loadingTables && addBackendId && addTables.length === 0 && (
                <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5 }}>
                  No tables found for this backend.
                </Typography>
              )}
            </FormControl>

            <TextField
              label="Binding Name"
              size="small"
              fullWidth
              placeholder="Defaults to the driving table's name"
              value={addBindingName}
              onChange={(e) => setAddBindingName(e.target.value)}
            />

            <FormControlLabel
              control={<Switch checked={addIsDefault} onChange={(e) => setAddIsDefault(e.target.checked)} />}
              label="Set as default binding"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAddOpen(false)} disabled={saving}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleCreateBinding}
            disabled={saving || !addBackendId || !addTableId}
            startIcon={saving ? <CircularProgress size={16} color="inherit" /> : <AddIcon />}
          >
            {saving ? 'Adding…' : 'Add Binding'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Binding dialog */}
      <Dialog open={!!editingBinding} onClose={() => !saving && setEditingBinding(null)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Binding</DialogTitle>
        <DialogContent>
          <Stack spacing={2.5} sx={{ mt: 1 }}>
            {formError && <Alert severity="error" onClose={() => setFormError(null)}>{formError}</Alert>}
            <Alert severity="info" variant="outlined">
              Backend and driving table aren't editable here — repointing a binding at a different
              physical table needs a reviewed script, not an in-place edit.
            </Alert>
            <TextField
              label="Binding Name"
              size="small"
              fullWidth
              value={editBindingName}
              onChange={(e) => setEditBindingName(e.target.value)}
            />
            <FormControlLabel
              control={<Switch checked={editIsDefault} onChange={(e) => setEditIsDefault(e.target.checked)} />}
              label="Default binding"
            />
            <FormControlLabel
              control={<Switch checked={editIsActive} onChange={(e) => setEditIsActive(e.target.checked)} />}
              label="Active"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditingBinding(null)} disabled={saving}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleSaveEdit}
            disabled={saving}
            startIcon={saving ? <CircularProgress size={16} color="inherit" /> : undefined}
          >
            {saving ? 'Saving…' : 'Save Changes'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
