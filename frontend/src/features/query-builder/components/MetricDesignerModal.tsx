import React, { useState, useMemo } from 'react';
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
import FunctionsIcon from '@mui/icons-material/Functions';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import UpgradeIcon from '@mui/icons-material/Upgrade';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import MetricLineageDrawer from './MetricLineageDrawer';
import type { CoreQueryStatus } from '../types/queryDef';

export interface MetricDesignerDefinition {
  id?: string;
  name: string;
  description: string;
  boId: string;
  expression: {
    kind: 'aggregation' | 'formula' | 'derived';
    fn?: 'sum' | 'avg' | 'count' | 'min' | 'max';
    termNodeId?: string;
    formula?: string;
    baseMetricIds?: string[];
  };
  grainAllowlist: string[];
  variables: Array<{
    name: string;
    type: 'number' | 'string' | 'date';
    defaultValue?: any;
    required: boolean;
    description?: string;
  }>;
  formatConfig: {
    type: 'currency' | 'number' | 'percentage' | 'compact';
    precision?: number;
    currencySymbol?: string;
    prefix?: string;
    suffix?: string;
  };
  materializationConfig: {
    strategy: 'on_the_fly' | 'starrocks_mv' | 'iceberg';
    targetTable?: string;
    refreshSchedule?: string;
    stalePolicy?: 'serve_with_flag' | 'force_raw_fallback';
  };
  isCore?: boolean;
  status?: CoreQueryStatus;
}

export interface MetricDesignerModalProps {
  open: boolean;
  onClose: () => void;
  onSave: (metric: MetricDesignerDefinition) => Promise<void>;
  initialMetric?: MetricDesignerDefinition;
  availableTerms?: Array<{ id: string; name: string; type: string }>;
  availableGrains?: string[];
  availableBaseMetrics?: Array<{ id: string; name: string }>;
  activeQueryGrains?: string[]; // Grains currently requested by calling query/tile
}

const ALLOWLISTED_FUNCTIONS = ['SUM', 'AVG', 'COUNT', 'MIN', 'MAX', 'NULLIF'];

export const MetricDesignerModal: React.FC<MetricDesignerModalProps> = ({
  open,
  onClose,
  onSave,
  initialMetric,
  availableTerms = [],
  availableGrains = ['region', 'desk', 'currency', 'trade_date', 'account_id'],
  availableBaseMetrics = [],
  activeQueryGrains = [],
}) => {
  const [tabIndex, setTabIndex] = useState(0);
  const [name, setName] = useState(initialMetric?.name || '');
  const [description, setDescription] = useState(initialMetric?.description || '');
  const [boId, setBoId] = useState(initialMetric?.boId || 'order');
  const [kind, setKind] = useState<'aggregation' | 'formula' | 'derived'>(initialMetric?.expression.kind || 'aggregation');
  const [aggFn, setAggFn] = useState<'sum' | 'avg' | 'count' | 'min' | 'max'>(initialMetric?.expression.fn || 'sum');
  const [termNodeId, setTermNodeId] = useState(initialMetric?.expression.termNodeId || (availableTerms[0]?.id || 'amount'));
  const [formula, setFormula] = useState(initialMetric?.expression.formula || 'SUM(amount) * @fx_rate');
  const [baseMetricIds, setBaseMetricIds] = useState<string[]>(initialMetric?.expression.baseMetricIds || []);
  const [grainAllowlist, setGrainAllowlist] = useState<string[]>(initialMetric?.grainAllowlist || ['region', 'desk']);
  const [formatType, setFormatType] = useState<'currency' | 'number' | 'percentage' | 'compact'>(initialMetric?.formatConfig.type || 'currency');
  const [currencySymbol, setCurrencySymbol] = useState(initialMetric?.formatConfig.currencySymbol || '$');
  const [strategy, setStrategy] = useState<'on_the_fly' | 'starrocks_mv' | 'iceberg'>(initialMetric?.materializationConfig.strategy || 'on_the_fly');
  const [stalePolicy, setStalePolicy] = useState<'serve_with_flag' | 'force_raw_fallback'>(initialMetric?.materializationConfig.stalePolicy || 'serve_with_flag');

  const [lineageOpen, setLineageOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // AST Validation: verify formula only contains allowlisted operators and identifiers
  const formulaValidationError = useMemo(() => {
    if (kind !== 'formula') return null;
    const trimmed = formula.trim();
    if (!trimmed) return 'Formula cannot be empty';

    // Disallow raw DDL/DML keywords
    const dangerousPatterns = [/DROP\s+/i, /DELETE\s+/i, /UPDATE\s+/i, /INSERT\s+/i, /ALTER\s+/i, /EXEC\s+/i];
    for (const pat of dangerousPatterns) {
      if (pat.test(trimmed)) {
        return 'Formula contains invalid or unauthorized SQL keywords';
      }
    }
    return null;
  }, [kind, formula]);

  // Inline Grain-422 Error Surfacing: Check if active query requests grains outside grainAllowlist
  const grainViolationError = useMemo(() => {
    if (!activeQueryGrains.length || !grainAllowlist.length) return null;
    const allowedMap = new Set(grainAllowlist.map((g) => g.toLowerCase().trim()));
    const unallowed = activeQueryGrains.filter((g) => !allowedMap.has(g.toLowerCase().trim()));
    if (unallowed.length > 0) {
      return `ErrGrainNotAllowed: Requested grain(s) [${unallowed.join(', ')}] are not permitted by this metric's allowlist (allowed grains: [${grainAllowlist.join(', ')}])`;
    }
    return null;
  }, [activeQueryGrains, grainAllowlist]);

  const handleSave = async () => {
    if (!name.trim()) {
      setSaveError('Metric name is required');
      return;
    }
    if (formulaValidationError) {
      setSaveError(formulaValidationError);
      return;
    }

    setSaving(true);
    setSaveError(null);

    try {
      await onSave({
        id: initialMetric?.id,
        name,
        description,
        boId,
        expression: {
          kind,
          fn: kind === 'aggregation' ? aggFn : undefined,
          termNodeId: kind === 'aggregation' ? termNodeId : undefined,
          formula: kind === 'formula' ? formula : undefined,
          baseMetricIds: kind === 'derived' ? baseMetricIds : undefined,
        },
        grainAllowlist,
        variables: [
          { name: 'fx_rate', type: 'number', defaultValue: 1.0, required: false },
        ],
        formatConfig: {
          type: formatType,
          currencySymbol: formatType === 'currency' ? currencySymbol : undefined,
        },
        materializationConfig: {
          strategy,
          stalePolicy,
        },
        isCore: initialMetric?.isCore,
        status: initialMetric?.status || 'custom',
      });
      onClose();
    } catch (err: any) {
      setSaveError(err.message || 'Failed to save metric definition');
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center">
            <Stack direction="row" alignItems="center" spacing={1}>
              <FunctionsIcon color="primary" />
              <Typography variant="h6" fontWeight={700}>
                {initialMetric?.id ? 'Edit Semantic Metric' : 'Create Semantic Metric'}
              </Typography>
              {initialMetric?.isCore && (
                <Chip size="small" label="CORE" color="primary" sx={{ height: 20, fontSize: '0.7rem' }} />
              )}
            </Stack>
            <Stack direction="row" spacing={1}>
              {initialMetric?.id && (
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<AccountTreeIcon />}
                  onClick={() => setLineageOpen(true)}
                >
                  View Lineage
                </Button>
              )}
              <IconButton size="small" onClick={onClose}>
                <CloseIcon fontSize="small" />
              </IconButton>
            </Stack>
          </Stack>

          <Tabs value={tabIndex} onChange={(_, idx) => setTabIndex(idx)} sx={{ mt: 1 }}>
            <Tab label="1. Calculation & Formula" />
            <Tab label="2. Grains & Governance" />
            <Tab label="3. Materialization & Format" />
          </Tabs>
        </DialogTitle>

        <DialogContent dividers>
          {grainViolationError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {grainViolationError}
            </Alert>
          )}

          {saveError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {saveError}
            </Alert>
          )}

          {tabIndex === 0 && (
            <Stack spacing={2.5}>
              <TextField
                label="Metric Name"
                fullWidth
                size="small"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />

              <TextField
                label="Description"
                fullWidth
                size="small"
                multiline
                rows={2}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />

              <FormControl fullWidth size="small">
                <InputLabel>Calculation Kind</InputLabel>
                <Select
                  value={kind}
                  label="Calculation Kind"
                  onChange={(e) => setKind(e.target.value as any)}
                  disabled={initialMetric?.isCore}
                >
                  <MenuItem value="aggregation">Aggregation (SUM, AVG, COUNT, etc.)</MenuItem>
                  <MenuItem value="formula">Safe Formula AST (Math + Bound Variables)</MenuItem>
                  <MenuItem value="derived">Derived Metric (Combine Existing Metrics)</MenuItem>
                </Select>
              </FormControl>

              {kind === 'aggregation' && (
                <Stack direction="row" spacing={2}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Aggregation Function</InputLabel>
                    <Select
                      value={aggFn}
                      label="Aggregation Function"
                      onChange={(e) => setAggFn(e.target.value as any)}
                      disabled={initialMetric?.isCore}
                    >
                      <MenuItem value="sum">SUM</MenuItem>
                      <MenuItem value="avg">AVG</MenuItem>
                      <MenuItem value="count">COUNT</MenuItem>
                      <MenuItem value="min">MIN</MenuItem>
                      <MenuItem value="max">MAX</MenuItem>
                    </Select>
                  </FormControl>

                  <FormControl fullWidth size="small">
                    <InputLabel>Underlying Term / Column</InputLabel>
                    <Select
                      value={termNodeId}
                      label="Underlying Term / Column"
                      onChange={(e) => setTermNodeId(e.target.value)}
                      disabled={initialMetric?.isCore}
                    >
                      {availableTerms.length > 0 ? (
                        availableTerms.map((t) => (
                          <MenuItem key={t.id} value={t.id}>
                            {t.name} ({t.type})
                          </MenuItem>
                        ))
                      ) : (
                        <MenuItem value="amount">amount (numeric)</MenuItem>
                      )}
                    </Select>
                  </FormControl>
                </Stack>
              )}

              {kind === 'formula' && (
                <Box>
                  <TextField
                    label="Formula Expression"
                    fullWidth
                    size="small"
                    value={formula}
                    onChange={(e) => setFormula(e.target.value)}
                    error={!!formulaValidationError}
                    helperText={formulaValidationError || 'Allowlisted AST functions: SUM, AVG, COUNT, MIN, MAX, NULLIF, @variables'}
                    disabled={initialMetric?.isCore}
                  />
                  <Stack direction="row" spacing={0.5} sx={{ mt: 1 }}>
                    <Typography variant="caption" color="text.secondary">
                      Suggested functions:
                    </Typography>
                    {ALLOWLISTED_FUNCTIONS.map((fn) => (
                      <Chip
                        key={fn}
                        size="small"
                        label={fn}
                        onClick={() => setFormula((prev) => `${prev} ${fn}()`)}
                        clickable={!initialMetric?.isCore}
                        sx={{ height: 18, fontSize: '0.65rem' }}
                      />
                    ))}
                  </Stack>
                </Box>
              )}

              {kind === 'derived' && (
                <FormControl fullWidth size="small">
                  <InputLabel>Base Metrics (DERIVED_FROM)</InputLabel>
                  <Select
                    multiple
                    value={baseMetricIds}
                    label="Base Metrics (DERIVED_FROM)"
                    onChange={(e) => setBaseMetricIds(typeof e.target.value === 'string' ? e.target.value.split(',') : e.target.value)}
                    renderValue={(selected) => (
                      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                        {selected.map((val) => (
                          <Chip key={val} size="small" label={val} />
                        ))}
                      </Box>
                    )}
                  >
                    {availableBaseMetrics.map((bm) => (
                      <MenuItem key={bm.id} value={bm.id}>
                        {bm.name} ({bm.id})
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              )}
            </Stack>
          )}

          {tabIndex === 1 && (
            <Stack spacing={2.5}>
              <Box>
                <Typography variant="subtitle2" fontWeight={600} gutterBottom>
                  Grain Allowlist (Permitted Grouping Dimensions)
                </Typography>
                <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 1 }}>
                  Client queries can only group by dimensions explicitly enabled in this allowlist. Unauthorized groupings return HTTP 422 (ErrGrainNotAllowed).
                </Typography>
                <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                  {availableGrains.map((grain) => {
                    const isSelected = grainAllowlist.includes(grain);
                    return (
                      <Chip
                        key={grain}
                        label={grain}
                        color={isSelected ? 'primary' : 'default'}
                        variant={isSelected ? 'filled' : 'outlined'}
                        onClick={() => {
                          if (isSelected) {
                            // Cannot remove core grains in extended mode
                            if (initialMetric?.isCore) return;
                            setGrainAllowlist(grainAllowlist.filter((g) => g !== grain));
                          } else {
                            setGrainAllowlist([...grainAllowlist, grain]);
                          }
                        }}
                      />
                    );
                  })}
                </Stack>
              </Box>

              <Divider />

              <Box>
                <Typography variant="subtitle2" fontWeight={600} gutterBottom>
                  Bound Variables
                </Typography>
                <Paper variant="outlined" sx={{ p: 1.5 }}>
                  <Stack direction="row" justifyContent="space-between" alignItems="center">
                    <Typography variant="body2" fontWeight={500}>
                      @fx_rate (number, default: 1.0)
                    </Typography>
                    <Chip size="small" label="PARAMETERIZED" color="info" />
                  </Stack>
                </Paper>
              </Box>
            </Stack>
          )}

          {tabIndex === 2 && (
            <Stack spacing={2.5}>
              <Box>
                <Typography variant="subtitle2" fontWeight={600} gutterBottom>
                  Materialization Tier
                </Typography>
                <FormControl fullWidth size="small">
                  <InputLabel>Storage Strategy</InputLabel>
                  <Select
                    value={strategy}
                    label="Storage Strategy"
                    onChange={(e) => setStrategy(e.target.value as any)}
                  >
                    <MenuItem value="on_the_fly">On-The-Fly (Dynamic pushdown + watermark result cache)</MenuItem>
                    <MenuItem value="starrocks_mv">StarRocks Materialized View (OLAP pre-aggregation)</MenuItem>
                    <MenuItem value="iceberg">Apache Iceberg (Cold tier federated lake persistence)</MenuItem>
                  </Select>
                </FormControl>
              </Box>

              {strategy === 'starrocks_mv' && (
                <FormControl fullWidth size="small">
                  <InputLabel>Stale-MV Fallback Policy</InputLabel>
                  <Select
                    value={stalePolicy}
                    label="Stale-MV Fallback Policy"
                    onChange={(e) => setStalePolicy(e.target.value as any)}
                  >
                    <MenuItem value="serve_with_flag">Serve with Stale Flag (Fast dashboard aggregates)</MenuItem>
                    <MenuItem value="force_raw_fallback">Force Raw Table Fallback (Strict compliance)</MenuItem>
                  </Select>
                </FormControl>
              )}

              <Divider />

              <Box>
                <Typography variant="subtitle2" fontWeight={600} gutterBottom>
                  Display & Formatting
                </Typography>
                <Stack direction="row" spacing={2}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Format Type</InputLabel>
                    <Select
                      value={formatType}
                      label="Format Type"
                      onChange={(e) => setFormatType(e.target.value as any)}
                    >
                      <MenuItem value="currency">Currency</MenuItem>
                      <MenuItem value="percentage">Percentage</MenuItem>
                      <MenuItem value="compact">Compact (1.2M, 500K)</MenuItem>
                      <MenuItem value="number">Number</MenuItem>
                    </Select>
                  </FormControl>

                  {formatType === 'currency' && (
                    <TextField
                      label="Currency Symbol"
                      size="small"
                      value={currencySymbol}
                      onChange={(e) => setCurrencySymbol(e.target.value)}
                      sx={{ width: 140 }}
                    />
                  )}
                </Stack>
              </Box>
            </Stack>
          )}
        </DialogContent>

        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button onClick={onClose} disabled={saving}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSave}
            disabled={saving || !name.trim() || !!formulaValidationError}
          >
            {saving ? 'Saving...' : initialMetric?.id ? 'Update Metric' : 'Create Metric'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Read-Only Lineage Drawer */}
      <MetricLineageDrawer
        open={lineageOpen}
        onClose={() => setLineageOpen(false)}
        metricId={initialMetric?.id || null}
        metricName={name}
      />
    </>
  );
};

export default MetricDesignerModal;
