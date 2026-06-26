import React, { useState, useEffect } from 'react';
import {
  Box,
  Stepper,
  Step,
  StepLabel,
  StepContent,
  Button,
  Paper,
  Typography,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Checkbox,
  FormControlLabel,
  Chip,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Autocomplete,
  Alert,
  CircularProgress,
  Divider,
  Card,
  CardContent,
  Grid,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  TableChart as TableIcon,
  Functions as FunctionsIcon,
  Category as CategoryIcon,
  Link as LinkIcon,
  Speed as SpeedIcon,
  Security as SecurityIcon,
  Preview as PreviewIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { useTenant } from '../../contexts/TenantContext';

// Types
interface CatalogTable {
  id: string;
  name: string;
  display_name: string;
  description: string;
  schema: string;
}

interface CatalogColumn {
  id: string;
  name: string;
  display_name: string;
  data_type: string;
  description: string;
  is_primary_key: boolean;
  is_foreign_key: boolean;
}

interface CatalogRelationship {
  id: string;
  source_table: string;
  source_column: string;
  target_table: string;
  target_column: string;
  relation_type: string;
}

interface MeasureConfig {
  id: string;
  name: string;
  type: 'count' | 'sum' | 'avg' | 'min' | 'max' | 'countDistinct' | 'countDistinctApprox' | 'runningTotal';
  sql: string;
  title: string;
  format?: string;
  description?: string;
  filters?: { sql: string }[];
  drillMembers?: string[];
}

interface DimensionConfig {
  id: string;
  name: string;
  type: 'string' | 'number' | 'boolean' | 'time' | 'geo';
  sql: string;
  title: string;
  primaryKey?: boolean;
  description?: string;
  subQuery?: boolean;
}

interface JoinConfig {
  id: string;
  name: string;
  targetCube: string;
  relationship: 'one_to_one' | 'one_to_many' | 'many_to_one' | 'many_to_many';
  sql: string;
}

interface PreAggConfig {
  id: string;
  name: string;
  type: 'rollup' | 'rollupLambda' | 'rollupJoin' | 'originalSql';
  measures: string[];
  dimensions: string[];
  timeDimension?: string;
  granularity?: string;
  refreshKey?: string;
  partitionGranularity?: string;
  buildRangeStart?: string;
  buildRangeEnd?: string;
}

interface SecurityPolicyConfig {
  id: string;
  name: string;
  policyType: 'row' | 'column' | 'access';
  conditions: {
    roles?: string[];
    groups?: string[];
    attributes?: Record<string, any>;
  };
  effects: {
    action: string;
    rowFilters?: any[];
    columnMasks?: any[];
  };
}

interface WizardState {
  // Step 1: Source Selection
  selectedTable: CatalogTable | null;
  selectedColumns: CatalogColumn[];
  cubeName: string;
  cubeDescription: string;
  dataSource: string;
  coreModelId?: string;
  extensionType: 'standalone' | 'extend' | 'override';

  // Step 2: Measures
  measures: MeasureConfig[];

  // Step 3: Dimensions
  dimensions: DimensionConfig[];

  // Step 4: Relationships
  joins: JoinConfig[];

  // Step 5: Pre-aggregations
  preAggregations: PreAggConfig[];

  // Step 6: Security
  securityPolicies: SecurityPolicyConfig[];
}

// API calls
const API_BASE = '/api/cube';

async function fetchWithTenant(url: string, tenantId: string, datasourceId: string, options?: RequestInit) {
  const params = new URLSearchParams({ tenant_id: tenantId, tenant_instance_id: datasourceId });
  const fullUrl = `${url}${url.includes('?') ? '&' : '?'}${params}`;
  const response = await fetch(fullUrl, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId,
      ...options?.headers,
    },
  });
  if (!response.ok) {
    throw new Error(`API error: ${response.statusText}`);
  }
  return response.json();
}

// Wizard Component
export const CubeModelWizard: React.FC<{
  onComplete?: (modelId: string) => void;
  onCancel?: () => void;
  initialCoreModelId?: string;
}> = ({ onComplete, onCancel, initialCoreModelId }) => {
  const { tenant: selectedTenant, datasource: selectedDatasource } = useTenant();
  const [activeStep, setActiveStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [yamlPreview, setYamlPreview] = useState<string>('');
  const [showPreview, setShowPreview] = useState(false);

  // Catalog data
  const [tables, setTables] = useState<CatalogTable[]>([]);
  const [columns, setColumns] = useState<CatalogColumn[]>([]);
  const [relationships, setRelationships] = useState<CatalogRelationship[]>([]);
  const [coreModels, setCoreModels] = useState<any[]>([]);

  // Wizard state
  const [state, setState] = useState<WizardState>({
    selectedTable: null,
    selectedColumns: [],
    cubeName: '',
    cubeDescription: '',
    dataSource: 'default',
    extensionType: initialCoreModelId ? 'extend' : 'standalone',
    coreModelId: initialCoreModelId,
    measures: [],
    dimensions: [],
    joins: [],
    preAggregations: [],
    securityPolicies: [],
  });

  const tenantId = selectedTenant?.id || '';
  const datasourceId = selectedDatasource?.id || '';

  // Load catalog data
  useEffect(() => {
    if (!tenantId || !datasourceId) return;

    const loadCatalogData = async () => {
      try {
        setLoading(true);
        const [tablesData, relationshipsData, coreModelsData] = await Promise.all([
          fetchWithTenant(`${API_BASE}/catalog/tables`, tenantId, datasourceId),
          fetchWithTenant(`${API_BASE}/catalog/relationships`, tenantId, datasourceId),
          fetchWithTenant(`${API_BASE}/models/core`, tenantId, datasourceId),
        ]);
        setTables(tablesData);
        setRelationships(relationshipsData);
        setCoreModels(coreModelsData);
      } catch (err) {
        setError('Failed to load catalog data');
      } finally {
        setLoading(false);
      }
    };

    loadCatalogData();
  }, [tenantId, datasourceId]);

  // Load columns when table is selected
  useEffect(() => {
    if (!state.selectedTable) return;

    const loadColumns = async () => {
      try {
        const columnsData = await fetchWithTenant(
          `${API_BASE}/catalog/tables/${state.selectedTable!.id}/columns`,
          tenantId,
          datasourceId
        );
        setColumns(columnsData);
      } catch (err) {
        console.error('Failed to load columns', err);
      }
    };

    loadColumns();
  }, [state.selectedTable, tenantId, datasourceId]);

  // Create wizard session
  useEffect(() => {
    if (!tenantId || !datasourceId) return;

    const createSession = async () => {
      try {
        const session = await fetchWithTenant(
          `${API_BASE}/wizard/sessions`,
          tenantId,
          datasourceId,
          {
            method: 'POST',
            body: JSON.stringify({
              tenant_id: tenantId,
              tenant_instance_id: datasourceId,
              session_type: initialCoreModelId ? 'extension' : 'custom',
              created_by: '00000000-0000-0000-0000-000000000000', // TODO: Get from auth
            }),
          }
        );
        setSessionId(session.id);
      } catch (err) {
        console.error('Failed to create wizard session', err);
      }
    };

    createSession();
  }, [tenantId, datasourceId, initialCoreModelId]);

  // Step definitions
  const steps = [
    { label: 'Source Selection', icon: <TableIcon /> },
    { label: 'Configure Measures', icon: <FunctionsIcon /> },
    { label: 'Configure Dimensions', icon: <CategoryIcon /> },
    { label: 'Define Relationships', icon: <LinkIcon /> },
    { label: 'Pre-aggregations', icon: <SpeedIcon /> },
    { label: 'Security Policies', icon: <SecurityIcon /> },
  ];

  // Save step progress
  const saveStepProgress = async (stepNum: number, stepType: string, stepData: any) => {
    if (!sessionId) return;
    try {
      await fetchWithTenant(
        `${API_BASE}/wizard/sessions/${sessionId}/steps/${stepNum}`,
        tenantId,
        datasourceId,
        {
          method: 'PUT',
          body: JSON.stringify({
            step_type: stepType,
            step_data: stepData,
            completed: true,
          }),
        }
      );
    } catch (err) {
      console.error('Failed to save step progress', err);
    }
  };

  // Navigation
  const handleNext = async () => {
    // Save current step
    const stepTypes = ['source_selection', 'measures_config', 'dimensions_config', 'relationships_config', 'preagg_config', 'security_config'];
    await saveStepProgress(activeStep + 1, stepTypes[activeStep], getStepData(activeStep));
    
    setActiveStep((prev) => prev + 1);
  };

  const handleBack = () => {
    setActiveStep((prev) => prev - 1);
  };

  const getStepData = (step: number) => {
    switch (step) {
      case 0:
        return {
          table_id: state.selectedTable?.id,
          cube_name: state.cubeName,
          description: state.cubeDescription,
          data_source: state.dataSource,
          core_model_id: state.coreModelId,
          extension_type: state.extensionType,
          selected_columns: state.selectedColumns.map(c => c.id),
        };
      case 1:
        return { measures: state.measures };
      case 2:
        return { dimensions: state.dimensions };
      case 3:
        return { joins: state.joins };
      case 4:
        return { pre_aggregations: state.preAggregations };
      case 5:
        return { security_policies: state.securityPolicies };
      default:
        return {};
    }
  };

  // Preview YAML
  const handlePreview = async () => {
    try {
      setLoading(true);
      const response = await fetch(`${API_BASE}/models/preview-yaml?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          core_model_id: state.coreModelId,
          extension_type: state.extensionType,
          custom_config: {
            measures: state.measures,
            dimensions: state.dimensions,
            joins: state.joins,
            pre_aggregations: state.preAggregations,
          },
        }),
      });
      const yaml = await response.text();
      setYamlPreview(yaml);
      setShowPreview(true);
    } catch (err) {
      setError('Failed to generate preview');
    } finally {
      setLoading(false);
    }
  };

  // Complete wizard
  const handleComplete = async () => {
    if (!sessionId) return;
    try {
      setLoading(true);
      const result = await fetchWithTenant(
        `${API_BASE}/wizard/sessions/${sessionId}/complete`,
        tenantId,
        datasourceId,
        { method: 'POST' }
      );
      onComplete?.(result.result_model_id);
    } catch (err) {
      setError('Failed to complete wizard');
    } finally {
      setLoading(false);
    }
  };

  // Render step content
  const renderStepContent = (step: number) => {
    switch (step) {
      case 0:
        return <SourceSelectionStep state={state} setState={setState} tables={tables} columns={columns} coreModels={coreModels} />;
      case 1:
        return <MeasuresStep state={state} setState={setState} columns={columns} />;
      case 2:
        return <DimensionsStep state={state} setState={setState} columns={columns} />;
      case 3:
        return <RelationshipsStep state={state} setState={setState} relationships={relationships} tables={tables} />;
      case 4:
        return <PreAggregationsStep state={state} setState={setState} />;
      case 5:
        return <SecurityStep state={state} setState={setState} />;
      default:
        return null;
    }
  };

  if (!tenantId || !datasourceId) {
    return (
      <Alert severity="warning">
        Please select a tenant and datasource before creating a Cube model.
      </Alert>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h5">
          {initialCoreModelId ? 'Extend Cube Model' : 'Create Cube Model'}
        </Typography>
        <Box>
          <Button startIcon={<PreviewIcon />} onClick={handlePreview} disabled={loading}>
            Preview YAML
          </Button>
          <Button onClick={onCancel} sx={{ ml: 1 }}>
            Cancel
          </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" onClose={() => setError(null)} sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Stepper activeStep={activeStep} orientation="vertical">
        {steps.map((step, index) => (
          <Step key={step.label}>
            <StepLabel
              StepIconComponent={() => (
                <Box
                  sx={{
                    width: 32,
                    height: 32,
                    borderRadius: '50%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    bgcolor: index <= activeStep ? 'primary.main' : 'grey.300',
                    color: 'white',
                  }}
                >
                  {step.icon}
                </Box>
              )}
            >
              {step.label}
            </StepLabel>
            <StepContent>
              <Box sx={{ py: 2 }}>
                {renderStepContent(index)}
              </Box>
              <Box sx={{ display: 'flex', gap: 1, mt: 2 }}>
                <Button disabled={index === 0} onClick={handleBack}>
                  Back
                </Button>
                {index < steps.length - 1 ? (
                  <Button variant="contained" onClick={handleNext} disabled={loading}>
                    {loading ? <CircularProgress size={20} /> : 'Continue'}
                  </Button>
                ) : (
                  <Button variant="contained" color="success" onClick={handleComplete} disabled={loading}>
                    {loading ? <CircularProgress size={20} /> : 'Create Model'}
                  </Button>
                )}
              </Box>
            </StepContent>
          </Step>
        ))}
      </Stepper>

      {/* YAML Preview Dialog */}
      <Dialog open={showPreview} onClose={() => setShowPreview(false)} maxWidth="lg" fullWidth>
        <DialogTitle>YAML Preview</DialogTitle>
        <DialogContent>
          <Paper
            sx={{
              p: 2,
              bgcolor: 'grey.900',
              color: 'grey.100',
              fontFamily: 'monospace',
              fontSize: 12,
              overflow: 'auto',
              maxHeight: 500,
            }}
          >
            <pre>{yamlPreview}</pre>
          </Paper>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowPreview(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

// Step 1: Source Selection
const SourceSelectionStep: React.FC<{
  state: WizardState;
  setState: React.Dispatch<React.SetStateAction<WizardState>>;
  tables: CatalogTable[];
  columns: CatalogColumn[];
  coreModels: any[];
}> = ({ state, setState, tables, columns, coreModels }) => {
  return (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <FormControl fullWidth>
          <InputLabel>Extension Type</InputLabel>
          <Select
            value={state.extensionType}
            label="Extension Type"
            onChange={(e) => setState(s => ({ ...s, extensionType: e.target.value as any }))}
          >
            <MenuItem value="standalone">Standalone (New Model)</MenuItem>
            <MenuItem value="extend">Extend Core Model</MenuItem>
            <MenuItem value="override">Override Core Model</MenuItem>
          </Select>
        </FormControl>
      </Grid>

      {state.extensionType !== 'standalone' && (
        <Grid item xs={12}>
          <Autocomplete
            options={coreModels}
            getOptionLabel={(option) => option.name || ''}
            value={coreModels.find(m => m.id === state.coreModelId) || null}
            onChange={(_, value) => setState(s => ({ ...s, coreModelId: value?.id }))}
            renderInput={(params) => <TextField {...params} label="Select Core Model to Extend" />}
          />
        </Grid>
      )}

      <Grid item xs={12}>
        <Autocomplete
          options={tables}
          getOptionLabel={(option) => `${option.schema}.${option.name}`}
          value={state.selectedTable}
          onChange={(_, value) => setState(s => ({ ...s, selectedTable: value, selectedColumns: [] }))}
          renderInput={(params) => <TextField {...params} label="Source Table" />}
          renderOption={(props, option) => (
            <li {...props}>
              <Box>
                <Typography variant="body1">{option.display_name || option.name}</Typography>
                <Typography variant="caption" color="text.secondary">
                  {option.schema}.{option.name}
                </Typography>
              </Box>
            </li>
          )}
        />
      </Grid>

      <Grid item xs={12} md={6}>
        <TextField
          fullWidth
          label="Cube Name"
          value={state.cubeName}
          onChange={(e) => setState(s => ({ ...s, cubeName: e.target.value }))}
          placeholder={state.selectedTable ? toPascalCase(state.selectedTable.name) : 'MyCube'}
          helperText="PascalCase naming convention"
        />
      </Grid>

      <Grid item xs={12} md={6}>
        <TextField
          fullWidth
          label="Data Source"
          value={state.dataSource}
          onChange={(e) => setState(s => ({ ...s, dataSource: e.target.value }))}
          helperText="Database connection name"
        />
      </Grid>

      <Grid item xs={12}>
        <TextField
          fullWidth
          multiline
          rows={2}
          label="Description"
          value={state.cubeDescription}
          onChange={(e) => setState(s => ({ ...s, cubeDescription: e.target.value }))}
          placeholder="Describe what this cube represents"
        />
      </Grid>

      {state.selectedTable && columns.length > 0 && (
        <Grid item xs={12}>
          <Typography variant="subtitle2" gutterBottom>
            Select Columns to Include
          </Typography>
          <Paper variant="outlined" sx={{ maxHeight: 300, overflow: 'auto' }}>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell padding="checkbox">
                      <Checkbox
                        checked={state.selectedColumns.length === columns.length}
                        indeterminate={state.selectedColumns.length > 0 && state.selectedColumns.length < columns.length}
                        onChange={(e) => {
                          setState(s => ({
                            ...s,
                            selectedColumns: e.target.checked ? columns : [],
                          }));
                        }}
                      />
                    </TableCell>
                    <TableCell>Column</TableCell>
                    <TableCell>Type</TableCell>
                    <TableCell>Attributes</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {columns.map((col) => (
                    <TableRow key={col.id} hover>
                      <TableCell padding="checkbox">
                        <Checkbox
                          checked={state.selectedColumns.some(c => c.id === col.id)}
                          onChange={(e) => {
                            setState(s => ({
                              ...s,
                              selectedColumns: e.target.checked
                                ? [...s.selectedColumns, col]
                                : s.selectedColumns.filter(c => c.id !== col.id),
                            }));
                          }}
                        />
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2">{col.display_name || col.name}</Typography>
                        <Typography variant="caption" color="text.secondary">{col.name}</Typography>
                      </TableCell>
                      <TableCell>
                        <Chip label={col.data_type} size="small" />
                      </TableCell>
                      <TableCell>
                        {col.is_primary_key && <Chip label="PK" size="small" color="primary" sx={{ mr: 0.5 }} />}
                        {col.is_foreign_key && <Chip label="FK" size="small" color="secondary" />}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>
        </Grid>
      )}
    </Grid>
  );
};

// Step 2: Measures Configuration
const MeasuresStep: React.FC<{
  state: WizardState;
  setState: React.Dispatch<React.SetStateAction<WizardState>>;
  columns: CatalogColumn[];
}> = ({ state, setState, columns: _columns }) => {
  const measureTypes = ['count', 'sum', 'avg', 'min', 'max', 'countDistinct', 'countDistinctApprox', 'runningTotal'];
  const formatOptions = ['number', 'currency', 'percent', 'duration'];

  const addMeasure = () => {
    setState(s => ({
      ...s,
      measures: [
        ...s.measures,
        {
          id: `measure_${Date.now()}`,
          name: '',
          type: 'count',
          sql: '',
          title: '',
        },
      ],
    }));
  };

  const updateMeasure = (index: number, updates: Partial<MeasureConfig>) => {
    setState(s => ({
      ...s,
      measures: s.measures.map((m, i) => (i === index ? { ...m, ...updates } : m)),
    }));
  };

  const removeMeasure = (index: number) => {
    setState(s => ({
      ...s,
      measures: s.measures.filter((_, i) => i !== index),
    }));
  };

  // Auto-suggest measures from numeric columns
  const suggestMeasures = () => {
    const numericColumns = state.selectedColumns.filter(c => 
      ['integer', 'bigint', 'decimal', 'numeric', 'float', 'double', 'number'].includes(c.data_type.toLowerCase())
    );

    const suggestions: MeasureConfig[] = [
      {
        id: `measure_count_${Date.now()}`,
        name: 'count',
        type: 'count',
        sql: '*',
        title: 'Total Count',
      },
      ...numericColumns.flatMap(col => [
        {
          id: `measure_sum_${col.id}`,
          name: `total_${col.name}`,
          type: 'sum' as const,
          sql: `{CUBE}.${col.name}`,
          title: `Total ${col.display_name || col.name}`,
        },
        {
          id: `measure_avg_${col.id}`,
          name: `avg_${col.name}`,
          type: 'avg' as const,
          sql: `{CUBE}.${col.name}`,
          title: `Average ${col.display_name || col.name}`,
        },
      ]),
    ];

    setState(s => ({
      ...s,
      measures: [...s.measures, ...suggestions],
    }));
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="subtitle1">Define Measures</Typography>
        <Box>
          <Button startIcon={<RefreshIcon />} onClick={suggestMeasures} size="small" sx={{ mr: 1 }}>
            Auto-Suggest
          </Button>
          <Button startIcon={<AddIcon />} onClick={addMeasure} variant="outlined" size="small">
            Add Measure
          </Button>
        </Box>
      </Box>

      {state.measures.length === 0 ? (
        <Alert severity="info">
          Click "Auto-Suggest" to generate measures from your selected columns, or add measures manually.
        </Alert>
      ) : (
        state.measures.map((measure, index) => (
          <Card key={measure.id} variant="outlined" sx={{ mb: 2 }}>
            <CardContent>
              <Grid container spacing={2}>
                <Grid item xs={12} md={3}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Name"
                    value={measure.name}
                    onChange={(e) => updateMeasure(index, { name: e.target.value })}
                    placeholder="snake_case"
                  />
                </Grid>
                <Grid item xs={12} md={2}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Type</InputLabel>
                    <Select
                      value={measure.type}
                      label="Type"
                      onChange={(e) => updateMeasure(index, { type: e.target.value as any })}
                    >
                      {measureTypes.map(t => (
                        <MenuItem key={t} value={t}>{t}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} md={3}>
                  <TextField
                    fullWidth
                    size="small"
                    label="SQL"
                    value={measure.sql}
                    onChange={(e) => updateMeasure(index, { sql: e.target.value })}
                    placeholder="{CUBE}.column_name"
                  />
                </Grid>
                <Grid item xs={12} md={2}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Title"
                    value={measure.title}
                    onChange={(e) => updateMeasure(index, { title: e.target.value })}
                  />
                </Grid>
                <Grid item xs={12} md={1}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Format</InputLabel>
                    <Select
                      value={measure.format || ''}
                      label="Format"
                      onChange={(e) => updateMeasure(index, { format: e.target.value })}
                    >
                      <MenuItem value="">None</MenuItem>
                      {formatOptions.map(f => (
                        <MenuItem key={f} value={f}>{f}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} md={1}>
                  <IconButton color="error" onClick={() => removeMeasure(index)}>
                    <DeleteIcon />
                  </IconButton>
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        ))
      )}
    </Box>
  );
};

// Step 3: Dimensions Configuration
const DimensionsStep: React.FC<{
  state: WizardState;
  setState: React.Dispatch<React.SetStateAction<WizardState>>;
  columns: CatalogColumn[];
}> = ({ state, setState, columns: _columns }) => {
  const dimensionTypes = ['string', 'number', 'boolean', 'time', 'geo'];

  const addDimension = () => {
    setState(s => ({
      ...s,
      dimensions: [
        ...s.dimensions,
        {
          id: `dim_${Date.now()}`,
          name: '',
          type: 'string',
          sql: '',
          title: '',
        },
      ],
    }));
  };

  const updateDimension = (index: number, updates: Partial<DimensionConfig>) => {
    setState(s => ({
      ...s,
      dimensions: s.dimensions.map((d, i) => (i === index ? { ...d, ...updates } : d)),
    }));
  };

  const removeDimension = (index: number) => {
    setState(s => ({
      ...s,
      dimensions: s.dimensions.filter((_, i) => i !== index),
    }));
  };

  // Map data types to Cube dimension types
  const mapDataType = (dataType: string): DimensionConfig['type'] => {
    const lower = dataType.toLowerCase();
    if (['integer', 'bigint', 'decimal', 'numeric', 'float', 'double', 'number'].includes(lower)) return 'number';
    if (['boolean', 'bool'].includes(lower)) return 'boolean';
    if (['date', 'datetime', 'timestamp', 'time'].includes(lower)) return 'time';
    return 'string';
  };

  const suggestDimensions = () => {
    const suggestions: DimensionConfig[] = state.selectedColumns.map(col => ({
      id: `dim_${col.id}`,
      name: col.name,
      type: mapDataType(col.data_type),
      sql: `{CUBE}.${col.name}`,
      title: col.display_name || col.name,
      primaryKey: col.is_primary_key,
    }));

    setState(s => ({
      ...s,
      dimensions: [...s.dimensions, ...suggestions],
    }));
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="subtitle1">Define Dimensions</Typography>
        <Box>
          <Button startIcon={<RefreshIcon />} onClick={suggestDimensions} size="small" sx={{ mr: 1 }}>
            Auto-Suggest
          </Button>
          <Button startIcon={<AddIcon />} onClick={addDimension} variant="outlined" size="small">
            Add Dimension
          </Button>
        </Box>
      </Box>

      {state.dimensions.length === 0 ? (
        <Alert severity="info">
          Click "Auto-Suggest" to generate dimensions from your selected columns, or add dimensions manually.
        </Alert>
      ) : (
        state.dimensions.map((dimension, index) => (
          <Card key={dimension.id} variant="outlined" sx={{ mb: 2 }}>
            <CardContent>
              <Grid container spacing={2} alignItems="center">
                <Grid item xs={12} md={3}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Name"
                    value={dimension.name}
                    onChange={(e) => updateDimension(index, { name: e.target.value })}
                    placeholder="snake_case"
                  />
                </Grid>
                <Grid item xs={12} md={2}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Type</InputLabel>
                    <Select
                      value={dimension.type}
                      label="Type"
                      onChange={(e) => updateDimension(index, { type: e.target.value as any })}
                    >
                      {dimensionTypes.map(t => (
                        <MenuItem key={t} value={t}>{t}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} md={3}>
                  <TextField
                    fullWidth
                    size="small"
                    label="SQL"
                    value={dimension.sql}
                    onChange={(e) => updateDimension(index, { sql: e.target.value })}
                    placeholder="{CUBE}.column_name"
                  />
                </Grid>
                <Grid item xs={12} md={2}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Title"
                    value={dimension.title}
                    onChange={(e) => updateDimension(index, { title: e.target.value })}
                  />
                </Grid>
                <Grid item xs={12} md={1}>
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={dimension.primaryKey || false}
                        onChange={(e) => updateDimension(index, { primaryKey: e.target.checked })}
                        size="small"
                      />
                    }
                    label="PK"
                  />
                </Grid>
                <Grid item xs={12} md={1}>
                  <IconButton color="error" onClick={() => removeDimension(index)}>
                    <DeleteIcon />
                  </IconButton>
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        ))
      )}
    </Box>
  );
};

// Step 4: Relationships/Joins
const RelationshipsStep: React.FC<{
  state: WizardState;
  setState: React.Dispatch<React.SetStateAction<WizardState>>;
  relationships: CatalogRelationship[];
  tables: CatalogTable[];
}> = ({ state, setState, relationships, tables }) => {
  const relationshipTypes = ['one_to_one', 'one_to_many', 'many_to_one', 'many_to_many'];

  const addJoin = () => {
    setState(s => ({
      ...s,
      joins: [
        ...s.joins,
        {
          id: `join_${Date.now()}`,
          name: '',
          targetCube: '',
          relationship: 'many_to_one',
          sql: '',
        },
      ],
    }));
  };

  const updateJoin = (index: number, updates: Partial<JoinConfig>) => {
    setState(s => ({
      ...s,
      joins: s.joins.map((j, i) => (i === index ? { ...j, ...updates } : j)),
    }));
  };

  const removeJoin = (index: number) => {
    setState(s => ({
      ...s,
      joins: s.joins.filter((_, i) => i !== index),
    }));
  };

  const suggestJoins = () => {
    const sourceTable = state.selectedTable?.name;
    if (!sourceTable) return;

    const suggestions: JoinConfig[] = relationships
      .filter(r => r.source_table === sourceTable || r.target_table === sourceTable)
      .map(r => ({
        id: `join_${r.id}`,
        name: r.source_table === sourceTable ? r.target_table : r.source_table,
        targetCube: toPascalCase(r.source_table === sourceTable ? r.target_table : r.source_table),
        relationship: 'many_to_one' as const,
        sql: `{CUBE}.${r.source_column} = {${toPascalCase(r.target_table)}}.${r.target_column}`,
      }));

    setState(s => ({
      ...s,
      joins: [...s.joins, ...suggestions],
    }));
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="subtitle1">Define Relationships</Typography>
        <Box>
          <Button startIcon={<RefreshIcon />} onClick={suggestJoins} size="small" sx={{ mr: 1 }}>
            Auto-Suggest from Catalog
          </Button>
          <Button startIcon={<AddIcon />} onClick={addJoin} variant="outlined" size="small">
            Add Join
          </Button>
        </Box>
      </Box>

      {state.joins.length === 0 ? (
        <Alert severity="info">
          Click "Auto-Suggest from Catalog" to discover relationships, or add joins manually.
        </Alert>
      ) : (
        state.joins.map((join, index) => (
          <Card key={join.id} variant="outlined" sx={{ mb: 2 }}>
            <CardContent>
              <Grid container spacing={2} alignItems="center">
                <Grid item xs={12} md={2}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Name"
                    value={join.name}
                    onChange={(e) => updateJoin(index, { name: e.target.value })}
                  />
                </Grid>
                <Grid item xs={12} md={3}>
                  <Autocomplete
                    options={tables.map(t => toPascalCase(t.name))}
                    value={join.targetCube}
                    onChange={(_, value) => updateJoin(index, { targetCube: value || '' })}
                    renderInput={(params) => <TextField {...params} label="Target Cube" size="small" />}
                    freeSolo
                  />
                </Grid>
                <Grid item xs={12} md={2}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Relationship</InputLabel>
                    <Select
                      value={join.relationship}
                      label="Relationship"
                      onChange={(e) => updateJoin(index, { relationship: e.target.value as any })}
                    >
                      {relationshipTypes.map(t => (
                        <MenuItem key={t} value={t}>{t.replace(/_/g, ' ')}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} md={4}>
                  <TextField
                    fullWidth
                    size="small"
                    label="SQL Condition"
                    value={join.sql}
                    onChange={(e) => updateJoin(index, { sql: e.target.value })}
                    placeholder="{CUBE}.fk_id = {TargetCube}.id"
                  />
                </Grid>
                <Grid item xs={12} md={1}>
                  <IconButton color="error" onClick={() => removeJoin(index)}>
                    <DeleteIcon />
                  </IconButton>
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        ))
      )}
    </Box>
  );
};

// Step 5: Pre-aggregations
const PreAggregationsStep: React.FC<{
  state: WizardState;
  setState: React.Dispatch<React.SetStateAction<WizardState>>;
}> = ({ state, setState }) => {
  const preAggTypes = ['rollup', 'rollupLambda', 'rollupJoin', 'originalSql'];
  const granularities = ['second', 'minute', 'hour', 'day', 'week', 'month', 'quarter', 'year'];

  const addPreAgg = () => {
    setState(s => ({
      ...s,
      preAggregations: [
        ...s.preAggregations,
        {
          id: `preagg_${Date.now()}`,
          name: '',
          type: 'rollup',
          measures: [],
          dimensions: [],
        },
      ],
    }));
  };

  const updatePreAgg = (index: number, updates: Partial<PreAggConfig>) => {
    setState(s => ({
      ...s,
      preAggregations: s.preAggregations.map((p, i) => (i === index ? { ...p, ...updates } : p)),
    }));
  };

  const removePreAgg = (index: number) => {
    setState(s => ({
      ...s,
      preAggregations: s.preAggregations.filter((_, i) => i !== index),
    }));
  };

  const measureOptions = state.measures.map(m => m.name);
  const dimensionOptions = state.dimensions.map(d => d.name);
  const timeDimensionOptions = state.dimensions.filter(d => d.type === 'time').map(d => d.name);

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="subtitle1">Pre-aggregations</Typography>
        <Button startIcon={<AddIcon />} onClick={addPreAgg} variant="outlined" size="small">
          Add Pre-aggregation
        </Button>
      </Box>

      <Alert severity="info" sx={{ mb: 2 }}>
        Pre-aggregations improve query performance by pre-computing aggregations. Configure them based on your most common query patterns.
      </Alert>

      {state.preAggregations.map((preAgg, index) => (
        <Card key={preAgg.id} variant="outlined" sx={{ mb: 2 }}>
          <CardContent>
            <Grid container spacing={2}>
              <Grid item xs={12} md={3}>
                <TextField
                  fullWidth
                  size="small"
                  label="Name"
                  value={preAgg.name}
                  onChange={(e) => updatePreAgg(index, { name: e.target.value })}
                  placeholder="main"
                />
              </Grid>
              <Grid item xs={12} md={2}>
                <FormControl fullWidth size="small">
                  <InputLabel>Type</InputLabel>
                  <Select
                    value={preAgg.type}
                    label="Type"
                    onChange={(e) => updatePreAgg(index, { type: e.target.value as any })}
                  >
                    {preAggTypes.map(t => (
                      <MenuItem key={t} value={t}>{t}</MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12} md={3}>
                <Autocomplete
                  multiple
                  options={measureOptions}
                  value={preAgg.measures}
                  onChange={(_, value) => updatePreAgg(index, { measures: value })}
                  renderInput={(params) => <TextField {...params} label="Measures" size="small" />}
                  renderTags={(value, getTagProps) =>
                    value.map((option, i) => (
                      <Chip label={option} size="small" {...getTagProps({ index: i })} />
                    ))
                  }
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <Autocomplete
                  multiple
                  options={dimensionOptions}
                  value={preAgg.dimensions}
                  onChange={(_, value) => updatePreAgg(index, { dimensions: value })}
                  renderInput={(params) => <TextField {...params} label="Dimensions" size="small" />}
                  renderTags={(value, getTagProps) =>
                    value.map((option, i) => (
                      <Chip label={option} size="small" {...getTagProps({ index: i })} />
                    ))
                  }
                />
              </Grid>
              <Grid item xs={12} md={1}>
                <IconButton color="error" onClick={() => removePreAgg(index)}>
                  <DeleteIcon />
                </IconButton>
              </Grid>

              {/* Time dimension settings */}
              <Grid item xs={12} md={4}>
                <Autocomplete
                  options={timeDimensionOptions}
                  value={preAgg.timeDimension || null}
                  onChange={(_, value) => updatePreAgg(index, { timeDimension: value || undefined })}
                  renderInput={(params) => <TextField {...params} label="Time Dimension" size="small" />}
                />
              </Grid>
              <Grid item xs={12} md={2}>
                <FormControl fullWidth size="small">
                  <InputLabel>Granularity</InputLabel>
                  <Select
                    value={preAgg.granularity || ''}
                    label="Granularity"
                    onChange={(e) => updatePreAgg(index, { granularity: e.target.value })}
                  >
                    <MenuItem value="">None</MenuItem>
                    {granularities.map(g => (
                      <MenuItem key={g} value={g}>{g}</MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12} md={3}>
                <FormControl fullWidth size="small">
                  <InputLabel>Partition Granularity</InputLabel>
                  <Select
                    value={preAgg.partitionGranularity || ''}
                    label="Partition Granularity"
                    onChange={(e) => updatePreAgg(index, { partitionGranularity: e.target.value })}
                  >
                    <MenuItem value="">None</MenuItem>
                    {granularities.map(g => (
                      <MenuItem key={g} value={g}>{g}</MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12} md={3}>
                <TextField
                  fullWidth
                  size="small"
                  label="Refresh Key"
                  value={preAgg.refreshKey || ''}
                  onChange={(e) => updatePreAgg(index, { refreshKey: e.target.value })}
                  placeholder="every 1 hour"
                />
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      ))}
    </Box>
  );
};

// Step 6: Security Policies
const SecurityStep: React.FC<{
  state: WizardState;
  setState: React.Dispatch<React.SetStateAction<WizardState>>;
}> = ({ state, setState }) => {
  const policyTypes = ['row', 'column', 'access'];
  const actionTypes = ['allow', 'deny', 'filter', 'mask'];

  const addPolicy = () => {
    setState(s => ({
      ...s,
      securityPolicies: [
        ...s.securityPolicies,
        {
          id: `policy_${Date.now()}`,
          name: '',
          policyType: 'row',
          conditions: {},
          effects: { action: 'filter' },
        },
      ],
    }));
  };

  const updatePolicy = (index: number, updates: Partial<SecurityPolicyConfig>) => {
    setState(s => ({
      ...s,
      securityPolicies: s.securityPolicies.map((p, i) => (i === index ? { ...p, ...updates } : p)),
    }));
  };

  const removePolicy = (index: number) => {
    setState(s => ({
      ...s,
      securityPolicies: s.securityPolicies.filter((_, i) => i !== index),
    }));
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="subtitle1">Security Policies</Typography>
        <Button startIcon={<AddIcon />} onClick={addPolicy} variant="outlined" size="small">
          Add Policy
        </Button>
      </Box>

      <Alert severity="info" sx={{ mb: 2 }}>
        Security policies define row-level and column-level access control. Policies are evaluated for each query based on the user's security context.
      </Alert>

      {state.securityPolicies.length === 0 ? (
        <Alert severity="warning">
          No security policies configured. Data will be accessible to all authenticated users.
        </Alert>
      ) : (
        state.securityPolicies.map((policy, index) => (
          <Card key={policy.id} variant="outlined" sx={{ mb: 2 }}>
            <CardContent>
              <Grid container spacing={2}>
                <Grid item xs={12} md={4}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Policy Name"
                    value={policy.name}
                    onChange={(e) => updatePolicy(index, { name: e.target.value })}
                    placeholder="tenant_isolation"
                  />
                </Grid>
                <Grid item xs={12} md={3}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Policy Type</InputLabel>
                    <Select
                      value={policy.policyType}
                      label="Policy Type"
                      onChange={(e) => updatePolicy(index, { policyType: e.target.value as any })}
                    >
                      {policyTypes.map(t => (
                        <MenuItem key={t} value={t}>{t}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} md={3}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Action</InputLabel>
                    <Select
                      value={policy.effects.action}
                      label="Action"
                      onChange={(e) => updatePolicy(index, {
                        effects: { ...policy.effects, action: e.target.value }
                      })}
                    >
                      {actionTypes.map(t => (
                        <MenuItem key={t} value={t}>{t}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} md={2}>
                  <IconButton color="error" onClick={() => removePolicy(index)}>
                    <DeleteIcon />
                  </IconButton>
                </Grid>

                <Grid item xs={12}>
                  <Divider sx={{ my: 1 }} />
                  <Typography variant="subtitle2" gutterBottom>Conditions</Typography>
                </Grid>

                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Required Roles (comma-separated)"
                    value={policy.conditions.roles?.join(', ') || ''}
                    onChange={(e) => updatePolicy(index, {
                      conditions: {
                        ...policy.conditions,
                        roles: e.target.value.split(',').map(r => r.trim()).filter(Boolean),
                      }
                    })}
                    placeholder="admin, analyst, viewer"
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Required Groups (comma-separated)"
                    value={policy.conditions.groups?.join(', ') || ''}
                    onChange={(e) => updatePolicy(index, {
                      conditions: {
                        ...policy.conditions,
                        groups: e.target.value.split(',').map(g => g.trim()).filter(Boolean),
                      }
                    })}
                    placeholder="finance, sales"
                  />
                </Grid>

                {policy.policyType === 'row' && (
                  <>
                    <Grid item xs={12}>
                      <Typography variant="subtitle2" gutterBottom>Row Filters</Typography>
                    </Grid>
                    <Grid item xs={12} md={4}>
                      <Autocomplete
                        options={state.dimensions.map(d => d.name)}
                        renderInput={(params) => <TextField {...params} label="Filter Dimension" size="small" />}
                        onChange={(_, value) => {
                          if (value) {
                            updatePolicy(index, {
                              effects: {
                                ...policy.effects,
                                rowFilters: [
                                  ...(policy.effects.rowFilters || []),
                                  { dimension: value, operator: 'equals', values: [], dynamic: true }
                                ]
                              }
                            });
                          }
                        }}
                      />
                    </Grid>
                  </>
                )}

                {policy.policyType === 'column' && (
                  <>
                    <Grid item xs={12}>
                      <Typography variant="subtitle2" gutterBottom>Column Masks</Typography>
                    </Grid>
                    <Grid item xs={12} md={4}>
                      <Autocomplete
                        options={[...state.measures.map(m => m.name), ...state.dimensions.map(d => d.name)]}
                        renderInput={(params) => <TextField {...params} label="Masked Member" size="small" />}
                        onChange={(_, value) => {
                          if (value) {
                            updatePolicy(index, {
                              effects: {
                                ...policy.effects,
                                columnMasks: [
                                  ...(policy.effects.columnMasks || []),
                                  { member: value, maskType: 'redact' }
                                ]
                              }
                            });
                          }
                        }}
                      />
                    </Grid>
                  </>
                )}
              </Grid>
            </CardContent>
          </Card>
        ))
      )}
    </Box>
  );
};

// Helper functions
function toPascalCase(str: string): string {
  return str
    .split(/[_-]/)
    .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join('');
}

export default CubeModelWizard;
