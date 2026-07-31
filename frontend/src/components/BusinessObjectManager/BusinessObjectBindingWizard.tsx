import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  MenuItem,
  Paper,
  Select,
  Stack,
  Tab,
  Tabs,
  TextField,
  Typography,
} from '@mui/material';
import {
  Close as CloseIcon,
  Search as SearchIcon,
  Storage as StorageIcon,
  TableChart as TableChartIcon,
} from '@mui/icons-material';

import FieldOverridesMatrix from './FieldOverridesMatrix';

import { useTenant } from '../../contexts/TenantContext';
import { useNotification } from '../../hooks/useNotification';
import type {
  CatalogTableNode,
  EligibilitySource,
  TermColumnMapping,
  WizardBusinessObject,
  WizardField,
  WizardSemanticTerm,
} from './bindingWizard.types';
import {
  buildCreateBusinessObjectPayload,
  createBusinessObject,
  createEmptyBusinessObject,
  createFieldFromTerm,
  fetchAllSemanticTerms,
  fetchCalculatedSemanticTerms,
  fetchCatalogTables,
  fetchRelatedSemanticTerms,
  fetchSemanticTermsByTable,
  saveBindingWithFields,
} from './bindingWizard.service';

interface BusinessObjectBindingWizardProps {
  open: boolean;
  onClose: () => void;
  onSave?: (boId: string) => void;
}

const ELIGIBILITY_TABS: { key: EligibilitySource | 'MANUAL'; label: string }[] = [
  { key: 'DIRECT', label: 'Direct' },
  { key: 'RELATED', label: 'Related' },
  { key: 'CALCULATED', label: 'Calculated' },
  { key: 'MANUAL', label: 'Manual' },
];

function toDisplayName(value: string): string {
  return value
    .split('_')
    .map((w) => (w ? w[0].toUpperCase() + w.slice(1) : ''))
    .join(' ');
}

function toKey(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, '_')
    .replace(/_+/g, '_')
    .replace(/^_+|_+$/g, '');
}

export default function BusinessObjectBindingWizard({
  open,
  onClose,
  onSave,
}: BusinessObjectBindingWizardProps) {
  const { tenant, datasource } = useTenant();
  const notification = useNotification();
  const tenantId = tenant?.id || '';
  const datasourceId = (datasource?.id || datasource?.alpha_tenant_instance_id || '') as string;
  const datasourceName = (datasource?.source_name || datasource?.alpha_datasource?.datasource_name || 'Current Backend') as string;

  const [bo, setBo] = useState<WizardBusinessObject>(() =>
    createEmptyBusinessObject(datasourceId, datasourceName)
  );
  const [tables, setTables] = useState<CatalogTableNode[]>([]);
  const [tablesLoading, setTablesLoading] = useState(false);
  const [termsLoading, setTermsLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<EligibilitySource | 'MANUAL'>('DIRECT');
  const [manualSearch, setManualSearch] = useState('');
  const [keyTouched, setKeyTouched] = useState(false);
  const [displayTouched, setDisplayTouched] = useState(false);

  const [directTerms, setDirectTerms] = useState<WizardSemanticTerm[]>([]);
  const [relatedTerms] = useState<WizardSemanticTerm[]>([]);
  const [calculatedTerms] = useState<WizardSemanticTerm[]>([]);
  const [manualTerms, setManualTerms] = useState<WizardSemanticTerm[]>([]);
  const [termMap, setTermMap] = useState<Record<string, WizardSemanticTerm>>({});

  const binding = bo.bindings[0];
  const selectedTable = useMemo(
    () => tables.find((t) => t.node_id === binding.drivingTableId) || null,
    [tables, binding.drivingTableId]
  );

  useEffect(() => {
    if (open) {
      setBo(createEmptyBusinessObject(datasourceId, datasourceName));
      setDirectTerms([]);
      setManualTerms([]);
      setTermMap({});
      setError(null);
      setActiveTab('DIRECT');
      setManualSearch('');
      setKeyTouched(false);
      setDisplayTouched(false);
    }
  }, [open, datasourceId, datasourceName]);

  useEffect(() => {
    if (!open || !datasourceId) return;
    let cancelled = false;
    setTablesLoading(true);
    fetchCatalogTables({ tenantId, datasourceId })
      .then((data) => {
        if (!cancelled) setTables(data);
      })
      .catch((err) => {
        if (!cancelled) setError(err?.message || 'Failed to load tables');
      })
      .finally(() => {
        if (!cancelled) setTablesLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, datasourceId, tenantId]);

  useEffect(() => {
    if (!open || !datasourceId || !binding.drivingTableId) return;
    let cancelled = false;
    setTermsLoading(true);
    Promise.all([
      fetchSemanticTermsByTable(binding.drivingTableId, datasourceId, binding.drivingTableName),
      fetchRelatedSemanticTerms(binding.drivingTableId, datasourceId),
      fetchCalculatedSemanticTerms(binding.drivingTableId, datasourceId, []),
    ])
      .then(([direct, _related, _calculated]) => {
        if (cancelled) return;
        setDirectTerms(direct);
        const map: Record<string, WizardSemanticTerm> = {};
        direct.forEach((t) => {
          map[t.termNodeId] = t;
        });
        setTermMap((prev) => ({ ...prev, ...map }));
      })
      .catch((err) => {
        if (!cancelled) setError(err?.message || 'Failed to load semantic terms');
      })
      .finally(() => {
        if (!cancelled) setTermsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, datasourceId, binding.drivingTableId, binding.drivingTableName]);

  useEffect(() => {
    if (activeTab !== 'MANUAL' || !datasourceId) return;
    let cancelled = false;
    fetchAllSemanticTerms(tenantId, datasourceId, manualSearch)
      .then((terms) => {
        if (cancelled) return;
        setManualTerms(terms);
        const map: Record<string, WizardSemanticTerm> = {};
        terms.forEach((t) => {
          map[t.termNodeId] = t;
        });
        setTermMap((prev) => ({ ...prev, ...map }));
      })
      .catch(() => {
        // Non-fatal; manual tab can stay empty.
      });
    return () => {
      cancelled = true;
    };
  }, [activeTab, datasourceId, tenantId, manualSearch]);

  const handleNameChange = (value: string) => {
    setBo((prev) => {
      const next = { ...prev, name: value };
      if (!keyTouched) next.key = toKey(value);
      if (!displayTouched) next.displayName = toDisplayName(value);
      return next;
    });
  };

  const handleTableChange = (_event: any, value: CatalogTableNode | null) => {
    setBo((prev) => {
      const nextBinding = { ...prev.bindings[0] };
      nextBinding.drivingTableId = value?.node_id;
      nextBinding.drivingTableName = value?.node_name;
      nextBinding.drivingTableQualifiedPath = value?.qualified_path;
      nextBinding.fields = [];
      return { ...prev, bindings: [nextBinding] };
    });
  };

  const isFieldSelected = (termId: string) =>
    binding.fields.some((f) => f.semanticTermId === termId);

  const toggleTerm = (term: WizardSemanticTerm) => {
    if (isFieldSelected(term.termNodeId)) {
      removeField(term.termNodeId);
      return;
    }
    const newField = createFieldFromTerm(term, binding.fields.length);
    setBo((prev) => {
      const nextBinding = { ...prev.bindings[0] };
      nextBinding.fields = [...nextBinding.fields, newField];
      return { ...prev, bindings: [nextBinding] };
    });
    setTermMap((prev) => ({ ...prev, [term.termNodeId]: term }));
  };

  const removeField = (termId: string) => {
    setBo((prev) => {
      const nextBinding = { ...prev.bindings[0] };
      nextBinding.fields = nextBinding.fields.filter((f) => f.semanticTermId !== termId);
      return { ...prev, bindings: [nextBinding] };
    });
  };

  const updateField = (termId: string, updates: Partial<WizardField>) => {
    setBo((prev) => {
      const nextBinding = { ...prev.bindings[0] };
      nextBinding.fields = nextBinding.fields.map((f) =>
        f.semanticTermId === termId ? { ...f, ...updates } : f
      );
      return { ...prev, bindings: [nextBinding] };
    });
  };

  const handleMappingChange = (field: WizardField, mapping: TermColumnMapping | 'unresolved') => {
    const selectedMapping = mapping === 'unresolved' ? undefined : mapping;
    const bindingStatus = field.eligibilitySource === 'MANUAL'
      ? (selectedMapping ? 'PARTIAL' : 'UNRESOLVED')
      : (selectedMapping ? 'RESOLVED' : 'UNRESOLVED');
    updateField(field.semanticTermId, { selectedMapping, bindingStatus });
  };

  const visibleTerms = useMemo(() => {
    switch (activeTab) {
      case 'DIRECT':
        return directTerms;
      case 'RELATED':
        return relatedTerms;
      case 'CALCULATED':
        return calculatedTerms;
      case 'MANUAL':
        return manualTerms;
      default:
        return [];
    }
  }, [activeTab, directTerms, relatedTerms, calculatedTerms, manualTerms]);

  const validation = useMemo(() => {
    const issues: string[] = [];
    if (!bo.name.trim()) issues.push('Business Object name is required.');
    if (!bo.key.trim()) issues.push('Business Object key is required.');
    if (!binding.drivingTableId) issues.push('A driving table must be selected.');
    if (binding.fields.length === 0) issues.push('At least one semantic term must be added.');

    const unresolvedRequired = binding.fields.filter(
      (f) => f.bindingRequirement === 'REQUIRED' && f.bindingStatus === 'UNRESOLVED'
    );
    if (unresolvedRequired.length > 0) {
      issues.push(`${unresolvedRequired.length} REQUIRED field(s) are unresolved.`);
    }

    const missingJustification = binding.fields.filter(
      (f) => f.eligibilitySource === 'OVERRIDE' && !(f.overrideReason || '').trim()
    );
    if (missingJustification.length > 0) {
      issues.push(`${missingJustification.length} overridden field(s) require an audit justification.`);
    }

    const resolvedCount = binding.fields.filter((f) => f.bindingStatus === 'RESOLVED').length;
    return { issues, resolvedCount, totalFields: binding.fields.length, canSave: issues.length === 0 };
  }, [bo, binding]);

  const handleSave = async (publish = false) => {
    if (!validation.canSave) {
      setError(validation.issues.join(' '));
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const result = await saveBindingWithFields(bo, publish);
      if (result.error) {
        // BO was saved but binding/fields persist had a partial failure
        notification.warning?.(result.error) ?? console.warn(result.error);
      } else {
        notification.success(
          publish ? 'Business Object published.' : 'Business Object saved as draft.'
        );
      }
      onSave?.(result.id);
      onClose();
    } catch (err: any) {
      setError(err?.message || 'Failed to save Business Object.');
    } finally {
      setSaving(false);
    }
  };

  const handleRebaseField = async (fieldKey: string) => {
    const field = binding.fields.find((f) => f.key === fieldKey || f.name === fieldKey);
    if (!field || !field.coreReferenceFieldId) return;

    try {
      const targetBoId = boId || bo.key; // Fallback to key or ID
      await rebaseField(targetBoId, field.coreReferenceFieldId);
      notification.success(`Rebased field ${field.displayName} to latest Gold Copy blueprint.`);
      // Update local state to clear drift
      updateField(field.semanticTermId, {
        driftStatus: 'UP_TO_DATE',
        hasLocalOverride: false,
        eligibilitySource: 'INHERITED',
      });
    } catch (err: any) {
      notification.error(err?.message || 'Failed to rebase field.');
    }
  };

  const renderTermList = () => {
    if (termsLoading) {
      return (
        <Box sx={{ p: 4, textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            Loading terms…
          </Typography>
        </Box>
      );
    }

    if (activeTab === 'RELATED' || activeTab === 'CALCULATED') {
      return (
        <Alert severity="info" sx={{ m: 2 }}>
          {activeTab === 'RELATED'
            ? 'Related terms will be discovered from declared Business Object relationships once the backend supports relationship bindings.'
            : 'Calculated terms will be suggested once the backend exposes calculation dependency eligibility.'}
        </Alert>
      );
    }

    if (activeTab === 'MANUAL') {
      return (
        <>
          <TextField
            size="small"
            fullWidth
            placeholder="Search semantic terms"
            value={manualSearch}
            onChange={(e) => setManualSearch(e.target.value)}
            InputProps={{
              startAdornment: <SearchIcon fontSize="small" sx={{ mr: 1, color: 'action.active' }} />,
            }}
            sx={{ mb: 1 }}
          />
          {renderTermCheckboxes(manualTerms)}
        </>
      );
    }

    if (!binding.drivingTableId) {
      return (
        <Alert severity="info" sx={{ m: 2 }}>
          Select a driving table to see eligible semantic terms.
        </Alert>
      );
    }

    return renderTermCheckboxes(visibleTerms);
  };

  const renderTermCheckboxes = (terms: WizardSemanticTerm[]) => {
    if (terms.length === 0) {
      return (
        <Box sx={{ p: 3, textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            No terms available.
          </Typography>
        </Box>
      );
    }
    return (
      <List dense sx={{ maxHeight: 320, overflow: 'auto' }}>
        {terms.map((term) => {
          const selected = isFieldSelected(term.termNodeId);
          const hasMapping = term.mappings && term.mappings.length > 0;
          return (
            <ListItem key={term.termNodeId} disablePadding>
              <ListItemButton onClick={() => toggleTerm(term)} dense>
                <ListItemIcon>
                  <input
                    type="checkbox"
                    checked={selected}
                    onChange={() => {}}
                    aria-label={`Select ${term.termName}`}
                    style={{ pointerEvents: 'none' }}
                  />
                </ListItemIcon>
                <ListItemText
                  primary={term.termName}
                  secondary={
                    <>
                      {term.termKey}
                      {hasMapping && (
                        <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 1 }}>
                          ({term.mappings.length} mapping{term.mappings.length === 1 ? '' : 's'})
                        </Typography>
                      )}
                    </>
                  }
                />
              </ListItemButton>
            </ListItem>
          );
        })}
      </List>
    );
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="xl">
      <DialogTitle sx={{ fontWeight: 600 }}>
        Create Business Object
        <IconButton onClick={onClose} sx={{ position: 'absolute', right: 8, top: 8 }}>
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      <DialogContent dividers>
        <Stack spacing={3}>
          {error && <Alert severity="error">{error}</Alert>}

          {/* BO Definition */}
          <Paper variant="outlined" sx={{ p: 2 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
              Definition
            </Typography>
            <Grid container spacing={2}>
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  label="Name"
                  fullWidth
                  size="small"
                  value={bo.name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder="e.g. Customer"
                  required
                />
              </Grid>
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  label="Key"
                  fullWidth
                  size="small"
                  value={bo.key}
                  onChange={(e) => {
                    setKeyTouched(true);
                    setBo((prev) => ({ ...prev, key: e.target.value }));
                  }}
                  placeholder="e.g. customer"
                  required
                />
              </Grid>
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  label="Display Name"
                  fullWidth
                  size="small"
                  value={bo.displayName}
                  onChange={(e) => {
                    setDisplayTouched(true);
                    setBo((prev) => ({ ...prev, displayName: e.target.value }));
                  }}
                  placeholder="e.g. Customer Profile"
                  required
                />
              </Grid>
              <Grid size={{ xs: 12 }}>
                <TextField
                  label="Description"
                  fullWidth
                  size="small"
                  multiline
                  rows={2}
                  value={bo.description}
                  onChange={(e) => setBo((prev) => ({ ...prev, description: e.target.value }))}
                />
              </Grid>
              <Grid size={{ xs: 12, md: 6 }}>
                <FormControl fullWidth size="small">
                  <InputLabel>Status</InputLabel>
                  <Select
                    value={bo.status}
                    label="Status"
                    onChange={(e) =>
                      setBo((prev) => ({ ...prev, status: e.target.value as WizardBusinessObject['status'] }))
                    }
                  >
                    <MenuItem value="draft">Draft</MenuItem>
                    <MenuItem value="active">Active</MenuItem>
                    <MenuItem value="deprecated">Deprecated</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid size={{ xs: 12, md: 6 }}>
                <FormControl fullWidth size="small">
                  <InputLabel>History Mode</InputLabel>
                  <Select
                    value={bo.historyMode}
                    label="History Mode"
                    onChange={(e) =>
                      setBo((prev) => ({ ...prev, historyMode: e.target.value as WizardBusinessObject['historyMode'] }))
                    }
                  >
                    <MenuItem value="EXPLICIT_RANGE">Explicit Range</MenuItem>
                    <MenuItem value="EVENT_LOG">Event Log</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
            </Grid>
          </Paper>

          {/* Binding Panel */}
          <Paper variant="outlined" sx={{ p: 2 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
              Binding
            </Typography>
            <Grid container spacing={2} alignItems="center">
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  label="Backend"
                  fullWidth
                  size="small"
                  value={datasourceName}
                  InputProps={{ startAdornment: <StorageIcon fontSize="small" sx={{ mr: 1, color: 'action.active' }} /> }}
                  disabled
                />
              </Grid>
              <Grid size={{ xs: 12, md: 8 }}>
                <Autocomplete
                  options={tables}
                  getOptionLabel={(option) => option.node_name || option.qualified_path || ''}
                  value={selectedTable}
                  onChange={handleTableChange}
                  loading={tablesLoading}
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      label="Driving Table"
                      size="small"
                      placeholder="Select a table"
                      InputProps={{
                        ...params.InputProps,
                        startAdornment: <TableChartIcon fontSize="small" sx={{ mr: 1, color: 'action.active' }} />,
                      }}
                      required
                    />
                  )}
                />
              </Grid>
              {selectedTable && (
                <Grid size={{ xs: 12 }}>
                  <Typography variant="body2" color="text.secondary">
                    {selectedTable.qualified_path || selectedTable.node_name}
                  </Typography>
                </Grid>
              )}
            </Grid>
          </Paper>

          {/* Term Picker */}
          <Paper variant="outlined" sx={{ p: 2 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
              Semantic Terms
            </Typography>
            <Tabs
              value={activeTab}
              onChange={(_e, v) => setActiveTab(v)}
              variant="scrollable"
              scrollButtons="auto"
              sx={{ mb: 1 }}
            >
              {ELIGIBILITY_TABS.map((t) => (
                <Tab key={t.key} value={t.key} label={t.label} />
              ))}
            </Tabs>
            {renderTermList()}
          </Paper>

          {/* Selected Fields */}
          <Paper variant="outlined" sx={{ p: 2 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
              Selected Fields ({binding.fields.length})
            </Typography>
            {binding.fields.length === 0 ? (
              <Alert severity="info">Select semantic terms above to build the Business Object.</Alert>
            ) : (
              <FieldOverridesMatrix
                fields={binding.fields}
                termMap={termMap}
                onUpdateField={updateField}
                onMappingChange={handleMappingChange}
                onRemoveField={removeField}
                onRebaseField={handleRebaseField}
              />
            )}
          </Paper>

          {/* Validation Summary */}
          <Paper variant="outlined" sx={{ p: 2, bgcolor: 'action.hover' }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
              Validation Summary
            </Typography>
            <Stack direction="row" spacing={3} flexWrap="wrap">
              <Typography variant="body2">
                Fields: <strong>{validation.totalFields}</strong>
              </Typography>
              <Typography variant="body2">
                Resolved: <strong>{validation.resolvedCount}</strong>
              </Typography>
              {validation.issues.length > 0 ? (
                <Typography variant="body2" color="error">
                  {validation.issues.join(' ')}
                </Typography>
              ) : (
                <Typography variant="body2" color="success.main">
                  Ready to save
                </Typography>
              )}
            </Stack>
          </Paper>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={saving}>
          Cancel
        </Button>
        <Button onClick={() => handleSave(false)} disabled={saving || !validation.canSave} variant="outlined">
          Save Draft
        </Button>
        <Button onClick={() => handleSave(true)} disabled={saving || !validation.canSave} variant="contained">
          Publish
        </Button>
      </DialogActions>
    </Dialog>
  );
}
