import React, { useReducer, useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Container,
  Paper,
  Stepper,
  Step,
  StepLabel,
  Button,
  Typography,
  TextField,
  MenuItem,
  FormControl,
  InputLabel,
  Select,
  Autocomplete,
  Stack,
  Card,
  CardContent,
  IconButton,
  Divider,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Checkbox,
  Grid,
  CircularProgress,
  Alert
} from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  AutoAwesome as AutoAwesomeIcon,
  CheckCircle as CheckCircleIcon,
  NavigateNext as NextIcon,
  NavigateBefore as BeforeIcon,
  Save as SaveIcon
} from '@mui/icons-material';
import { AICopilotInput } from '../components/AICopilotInput';
import { useTenant } from '../contexts/TenantContext';

// --- Types matching AutoSuggestResponse and SQLGenerationRequest structs ---
interface SuggestedField {
  field_name: string;
  semantic_term_node_id: string;
  semantic_term_name: string;
  source_column_node_id: string;
  source_column_name: string;
  field_role: string; // KEY, DIMENSION, MEASURE, ATTRIBUTE
  aggregation_type: string; // NONE, SUM, AVG, MIN, MAX
  confidence_score: number;
}

interface CopilotDraft {
  suggested_bo_name: string;
  suggested_bo_key: string;
  fields: SuggestedField[];
}

interface FallbackBinding {
  datasourceType: string;
  drivingNodeId: string;
  connectionString?: string;
}

interface WizardState {
  name: string;
  technicalKey: string;
  description: string;
  datasourceType: string;
  drivingNodeId: string;
  fallbackBindings: FallbackBinding[];
  selectedTermIds: string[];
  termsMatrix: SuggestedField[];
}

type WizardAction =
  | { type: 'SET_FIELD'; field: keyof WizardState; value: any }
  | { type: 'HYDRATE_DRAFT'; draft: CopilotDraft }
  | { type: 'ADD_FALLBACK'; fallback: FallbackBinding }
  | { type: 'REMOVE_FALLBACK'; index: number }
  | { type: 'UPDATE_MATRIX_FIELD'; index: number; field: keyof SuggestedField; value: any };

const initialState: WizardState = {
  name: '',
  technicalKey: '',
  description: '',
  datasourceType: 'Postgres',
  drivingNodeId: '',
  fallbackBindings: [],
  selectedTermIds: [],
  termsMatrix: []
};

function wizardReducer(state: WizardState, action: WizardAction): WizardState {
  switch (action.type) {
    case 'SET_FIELD':
      return { ...state, [action.field]: action.value };
    case 'HYDRATE_DRAFT':
      return {
        ...state,
        name: action.draft.suggested_bo_name,
        technicalKey: action.draft.suggested_bo_key,
        termsMatrix: action.draft.fields.map(f => ({
          ...f,
          field_role: f.field_role || 'ATTRIBUTE',
          aggregation_type: f.aggregation_type || 'NONE'
        })),
        selectedTermIds: action.draft.fields.map(f => f.semantic_term_node_id)
      };
    case 'ADD_FALLBACK':
      return { ...state, fallbackBindings: [...state.fallbackBindings, action.fallback] };
    case 'REMOVE_FALLBACK':
      return {
        ...state,
        fallbackBindings: state.fallbackBindings.filter((_, idx) => idx !== action.index)
      };
    case 'UPDATE_MATRIX_FIELD':
      const updatedMatrix = [...state.termsMatrix];
      updatedMatrix[action.index] = {
        ...updatedMatrix[action.index],
        [action.field]: action.value
      };
      return { ...state, termsMatrix: updatedMatrix };
    default:
      return state;
  }
}

const steps = ['Definition', 'Bindings', 'Scope', 'Terms Matrix'];

export const BusinessObjectWizardPage: React.FC<{ tenantId?: string }> = ({ tenantId }) => {
  const navigate = useNavigate();
  const { tenant } = useTenant();
  const activeTenantId = tenantId || tenant?.id || '00000000-0000-0000-0000-000000000000';
  const [activeStep, setActiveStep] = useState(0);
  const [state, dispatch] = useReducer(wizardReducer, initialState);
  
  // API states
  const [drivingTables, setDrivingTables] = useState<{ id: string; label: string }[]>([]);
  const [eligibleTerms, setEligibleTerms] = useState<SuggestedField[]>([]);
  const [loadingTerms, setLoadingTerms] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  // Auto-format technical key to snake_case
  const handleNameChange = (nameVal: string) => {
    dispatch({ type: 'SET_FIELD', field: 'name', value: nameVal });
    const formattedKey = nameVal
      .toLowerCase()
      .replace(/[^a-z0-9_]/g, '_')
      .replace(/_+/g, '_');
    dispatch({ type: 'SET_FIELD', field: 'technicalKey', value: formattedKey });
  };

  // Fetch tables on load
  useEffect(() => {
    const fetchDrivingTables = async () => {
      try {
        const response = await fetch(`/api/v1/bo/driving-tables?tenant_id=${activeTenantId}`);
        if (response.ok) {
          const data = await response.json();
          setDrivingTables(data || []);
        } else {
          // Fallback static options if API is not fully running
          setDrivingTables([
            { id: '11111111-1111-1111-1111-111111111111', label: 'sales_ledger' },
            { id: '22222222-2222-2222-2222-222222222222', label: 'customer_dim' },
            { id: '33333333-3333-3333-3333-333333333333', label: 'product_catalog' }
          ]);
        }
      } catch {
        setDrivingTables([
          { id: '11111111-1111-1111-1111-111111111111', label: 'sales_ledger' },
          { id: '22222222-2222-2222-2222-222222222222', label: 'customer_dim' },
          { id: '33333333-3333-3333-3333-333333333333', label: 'product_catalog' }
        ]);
      }
    };
    fetchDrivingTables();
  }, [activeTenantId]);

  // Fetch eligible terms when driving table is selected
  useEffect(() => {
    if (!state.drivingNodeId) {
      setEligibleTerms([]);
      return;
    }

    const fetchEligibleTerms = async () => {
      setLoadingTerms(true);
      try {
        const response = await fetch(`/api/v1/bo/eligible-terms?driving_node_id=${state.drivingNodeId}`);
        if (response.ok) {
          const data = await response.json();
          setEligibleTerms(data || []);
        } else {
          // Fallbacks for testing/design demonstration
          setEligibleTerms([
            {
              field_name: 'total_revenue',
              semantic_term_node_id: 'term-rev-100',
              semantic_term_name: 'Revenue Amount',
              source_column_node_id: 'col-rev-01',
              source_column_name: 'amount',
              field_role: 'MEASURE',
              aggregation_type: 'SUM',
              confidence_score: 0.95
            },
            {
              field_name: 'region_code',
              semantic_term_node_id: 'term-reg-101',
              semantic_term_name: 'Region Identifier',
              source_column_node_id: 'col-reg-02',
              source_column_name: 'region_id',
              field_role: 'DIMENSION',
              aggregation_type: 'NONE',
              confidence_score: 0.88
            },
            {
              field_name: 'created_date',
              semantic_term_node_id: 'term-date-102',
              semantic_term_name: 'Creation Date',
              source_column_node_id: 'col-date-03',
              source_column_name: 'created_at',
              field_role: 'ATTRIBUTE',
              aggregation_type: 'NONE',
              confidence_score: 0.91
            }
          ]);
        }
      } catch {
        // Fallbacks
        setEligibleTerms([
          {
            field_name: 'total_revenue',
            semantic_term_node_id: 'term-rev-100',
            semantic_term_name: 'Revenue Amount',
            source_column_node_id: 'col-rev-01',
            source_column_name: 'amount',
            field_role: 'MEASURE',
            aggregation_type: 'SUM',
            confidence_score: 0.95
          },
          {
            field_name: 'region_code',
            semantic_term_node_id: 'term-reg-101',
            semantic_term_name: 'Region Identifier',
            source_column_node_id: 'col-reg-02',
            source_column_name: 'region_id',
            field_role: 'DIMENSION',
            aggregation_type: 'NONE',
            confidence_score: 0.88
          }
        ]);
      } finally {
        setLoadingTerms(false);
      }
    };

    fetchEligibleTerms();
  }, [state.drivingNodeId]);

  // Handle Copilot Draft hydration
  const handleCopilotDraft = (draft: CopilotDraft) => {
    dispatch({ type: 'HYDRATE_DRAFT', draft });
    // Auto-select sales_ledger or another matching demo table if draft is hydrated
    if (draft.fields.length > 0 && drivingTables.length > 0) {
      dispatch({ type: 'SET_FIELD', field: 'drivingNodeId', value: drivingTables[0].id });
    }
    setActiveStep(2); // Jump straight to scope definition / terms check
  };

  const handleNext = () => {
    if (activeStep === 2) {
      // Sync terms Matrix from selected term IDs before stepping to matrix view
      const selected = eligibleTerms.filter(t => state.selectedTermIds.includes(t.semantic_term_node_id));
      dispatch({ type: 'SET_FIELD', field: 'termsMatrix', value: selected });
    }
    setActiveStep((prevActiveStep) => prevActiveStep + 1);
  };

  const handleBack = () => {
    setActiveStep((prevActiveStep) => prevActiveStep - 1);
  };

  const handleToggleTerm = (termId: string) => {
    const currentIndex = state.selectedTermIds.indexOf(termId);
    const newSelected = [...state.selectedTermIds];

    if (currentIndex === -1) {
      newSelected.push(termId);
    } else {
      newSelected.splice(currentIndex, 1);
    }

    dispatch({ type: 'SET_FIELD', field: 'selectedTermIds', value: newSelected });
  };

  const handleSave = async () => {
    setSubmitError(null);
    try {
      const payload = {
        name: state.name,
        technical_key: state.technicalKey,
        description: state.description,
        tenant_id: activeTenantId,
        datasource_type: state.datasourceType,
        driving_node_id: state.drivingNodeId,
        fallback_bindings: state.fallbackBindings,
        fields: state.termsMatrix
      };

      const response = await fetch('/api/business-objects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || 'Failed to create business object');
      }

      navigate('/business-objects');
    } catch (err: any) {
      setSubmitError(err.message || 'An error occurred while saving the Business Object');
    }
  };

  // Matrix Grid column definitions
  const matrixColumns: GridColDef[] = [
    { field: 'semantic_term_name', headerName: 'Semantic Term', flex: 1, minWidth: 150 },
    { field: 'source_column_name', headerName: 'Physical Column', flex: 1, minWidth: 150 },
    {
      field: 'field_role',
      headerName: 'Field Role',
      width: 180,
      renderCell: (params: GridRenderCellParams) => {
        const idx = state.termsMatrix.findIndex(item => item.semantic_term_node_id === params.row.semantic_term_node_id);
        return (
          <FormControl fullWidth size="small" variant="standard" sx={{ mt: 0.5 }}>
            <Select
              value={params.value || 'ATTRIBUTE'}
              onChange={(e) => dispatch({ type: 'UPDATE_MATRIX_FIELD', index: idx, field: 'field_role', value: e.target.value })}
            >
              <MenuItem value="KEY">KEY</MenuItem>
              <MenuItem value="DIMENSION">DIMENSION</MenuItem>
              <MenuItem value="MEASURE">MEASURE</MenuItem>
              <MenuItem value="ATTRIBUTE">ATTRIBUTE</MenuItem>
            </Select>
          </FormControl>
        );
      }
    },
    {
      field: 'aggregation_type',
      headerName: 'Aggregation',
      width: 180,
      renderCell: (params: GridRenderCellParams) => {
        const idx = state.termsMatrix.findIndex(item => item.semantic_term_node_id === params.row.semantic_term_node_id);
        return (
          <FormControl fullWidth size="small" variant="standard" sx={{ mt: 0.5 }}>
            <Select
              value={params.value || 'NONE'}
              onChange={(e) => dispatch({ type: 'UPDATE_MATRIX_FIELD', index: idx, field: 'aggregation_type', value: e.target.value })}
            >
              <MenuItem value="NONE">NONE</MenuItem>
              <MenuItem value="SUM">SUM</MenuItem>
              <MenuItem value="AVG">AVG</MenuItem>
              <MenuItem value="MIN">MIN</MenuItem>
              <MenuItem value="MAX">MAX</MenuItem>
            </Select>
          </FormControl>
        );
      }
    }
  ];

  return (
    <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
      <AICopilotInput tenantId={activeTenantId} onDraftGenerated={handleCopilotDraft} />

      <Paper sx={{ p: 4, borderRadius: 3, boxShadow: '0 4px 20px rgba(0,0,0,0.08)' }}>
        <Typography variant="h5" component="h1" gutterBottom sx={{ fontWeight: 'bold', mb: 3 }}>
          Create Business Object
        </Typography>

        <Stepper activeStep={activeStep} sx={{ mb: 4 }}>
          {steps.map((label) => (
            <Step key={label}>
              <StepLabel>{label}</StepLabel>
            </Step>
          ))}
        </Stepper>

        <Divider sx={{ mb: 4 }} />

        {/* STEP 1: DEFINITION */}
        {activeStep === 0 && (
          <Stack spacing={3} sx={{ maxWith: 600, mx: 'auto' }}>
            <Typography variant="h6">Step 1: Core Definition</Typography>
            <TextField
              required
              label="Business Object Name"
              value={state.name}
              onChange={(e) => handleNameChange(e.target.value)}
              fullWidth
            />
            <TextField
              required
              label="Technical Key (auto-formatted)"
              value={state.technicalKey}
              onChange={(e) => dispatch({ type: 'SET_FIELD', field: 'technicalKey', value: e.target.value })}
              fullWidth
              helperText="Must be snake_case (e.g. monthly_revenue_by_region)"
            />
            <TextField
              label="Description"
              value={state.description}
              onChange={(e) => dispatch({ type: 'SET_FIELD', field: 'description', value: e.target.value })}
              multiline
              rows={4}
              fullWidth
            />
          </Stack>
        )}

        {/* STEP 2: BINDINGS */}
        {activeStep === 1 && (
          <Stack spacing={4}>
            <Typography variant="h6">Step 2: Source Physical Bindings</Typography>
            <Grid container spacing={3}>
              <Grid size={{ 'xs': 12, 'md': 6 }}>
                <FormControl fullWidth>
                  <InputLabel>Datasource Type</InputLabel>
                  <Select
                    value={state.datasourceType}
                    label="Datasource Type"
                    onChange={(e) => dispatch({ type: 'SET_FIELD', field: 'datasourceType', value: e.target.value })}
                  >
                    <MenuItem value="Postgres">Postgres</MenuItem>
                    <MenuItem value="Iceberg">Iceberg</MenuItem>
                    <MenuItem value="StarRocks">StarRocks</MenuItem>
                  </Select>
                </FormControl>
              </Grid>

              <Grid size={{ 'xs': 12, 'md': 6 }}>
                <Autocomplete
                  options={drivingTables}
                  getOptionLabel={(option) => option.label}
                  value={drivingTables.find(t => t.id === state.drivingNodeId) || null}
                  onChange={(_, newValue) => {
                    dispatch({ type: 'SET_FIELD', field: 'drivingNodeId', value: newValue ? newValue.id : '' });
                  }}
                  renderInput={(params) => <TextField {...params} label="Anchor/Driving Table" required />}
                />
              </Grid>
            </Grid>

            <Divider sx={{ my: 2 }} />

            <Box>
              <Typography variant="subtitle2" gutterBottom sx={{ fontWeight: 'bold' }}>
                Multi-Dialect Fallback Bindings (Iceberg / StarRocks replication)
              </Typography>
              <Button
                variant="outlined"
                startIcon={<AddIcon />}
                onClick={() => dispatch({
                  type: 'ADD_FALLBACK',
                  fallback: { datasourceType: 'StarRocks', drivingNodeId: state.drivingNodeId }
                })}
                disabled={!state.drivingNodeId}
                sx={{ mb: 2 }}
              >
                Add Fallback Replication Endpoint
              </Button>

              {state.fallbackBindings.length > 0 && (
                <Stack spacing={2}>
                  {state.fallbackBindings.map((fb, idx) => (
                    <Paper key={idx} variant="outlined" sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Box>
                        <Typography variant="body2" sx={{ fontWeight: 'bold' }}>Fallback #{idx + 1}</Typography>
                        <Typography variant="caption" color="text.secondary">
                          Type: {fb.datasourceType} | Driving Table Node: {fb.drivingNodeId}
                        </Typography>
                      </Box>
                      <IconButton onClick={() => dispatch({ type: 'REMOVE_FALLBACK', index: idx })} color="error">
                        <DeleteIcon />
                      </IconButton>
                    </Paper>
                  ))}
                </Stack>
              )}
            </Box>
          </Stack>
        )}

        {/* STEP 3: SCOPE / ELIGIBLE TERMS */}
        {activeStep === 2 && (
          <Stack spacing={3}>
            <Box>
              <Typography variant="h6">Step 3: Relationship Scope & Glossary Mapping</Typography>
              <Typography variant="body2" color="text.secondary">
                Select eligible terms discovered via semantic graph traversal on your driving table.
              </Typography>
            </Box>

            {loadingTerms ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
                <CircularProgress />
              </Box>
            ) : eligibleTerms.length === 0 ? (
              <Alert severity="warning">No eligible terms found. Please select an active driving table in Step 2.</Alert>
            ) : (
              <Paper variant="outlined" sx={{ maxHeight: 400, overflow: 'auto', p: 1 }}>
                <List>
                  {eligibleTerms.map((term) => {
                    const labelId = `checkbox-list-label-${term.semantic_term_node_id}`;
                    return (
                      <ListItem
                        key={term.semantic_term_node_id}
                        dense
                        button
                        onClick={() => handleToggleTerm(term.semantic_term_node_id)}
                      >
                        <ListItemIcon>
                          <Checkbox
                            edge="start"
                            checked={state.selectedTermIds.indexOf(term.semantic_term_node_id) !== -1}
                            tabIndex={-1}
                            disableRipple
                            inputProps={{ 'aria-labelledby': labelId }}
                          />
                        </ListItemIcon>
                        <ListItemText
                          id={labelId}
                          primary={term.semantic_term_name}
                          secondary={`Maps to physical column: ${term.source_column_name} (Confidence: ${(term.confidence_score * 100).toFixed(0)}%)`}
                        />
                      </ListItem>
                    );
                  })}
                </List>
              </Paper>
            )}
          </Stack>
        )}

        {/* STEP 4: TERMS MATRIX */}
        {activeStep === 3 && (
          <Stack spacing={3}>
            <Typography variant="h6">Step 4: Terms Matrix & Aggregations</Typography>
            <Box sx={{ height: 400, width: '100%' }}>
              <DataGrid
                rows={state.termsMatrix}
                columns={matrixColumns}
                getRowId={(row) => row.semantic_term_node_id}
                pageSizeOptions={[5, 10, 20]}
                disableRowSelectionOnClick
              />
            </Box>

            {submitError && (
              <Alert severity="error" sx={{ mt: 2 }}>{submitError}</Alert>
            )}
          </Stack>
        )}

        <Box sx={{ display: 'flex', justifyContent: 'space-between', mt: 4 }}>
          <Button
            disabled={activeStep === 0}
            onClick={handleBack}
            startIcon={<BeforeIcon />}
            variant="text"
          >
            Back
          </Button>

          {activeStep === steps.length - 1 ? (
            <Button
              variant="contained"
              onClick={handleSave}
              endIcon={<SaveIcon />}
              color="success"
              disabled={state.termsMatrix.length === 0}
            >
              Publish Business Object
            </Button>
          ) : (
            <Button
              variant="contained"
              onClick={handleNext}
              endIcon={<NextIcon />}
              disabled={
                (activeStep === 0 && (!state.name || !state.technicalKey)) ||
                (activeStep === 1 && !state.drivingNodeId) ||
                (activeStep === 2 && state.selectedTermIds.length === 0)
              }
            >
              Next
            </Button>
          )}
        </Box>
      </Paper>
    </Container>
  );
};

export default BusinessObjectWizardPage;
