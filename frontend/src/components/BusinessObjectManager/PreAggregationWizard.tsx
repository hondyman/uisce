import React, { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Stepper,
  Step,
  StepLabel,
  Typography,
  Box,
  TextField,
  IconButton,
  FormControl,
  FormLabel,
  RadioGroup,
  FormControlLabel,
  Radio,
  Checkbox,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Paper,
  Divider,
  Alert,
  CircularProgress,
  Select,
  MenuItem,
  InputLabel,
  Grid,
  Chip,
  Card,
  CardContent,
} from '@mui/material';
import { Close, Bolt, Storage, Schedule, Code, CheckCircle } from '@mui/icons-material';

// --- Types ---

interface BOTerm {
  id: string;
  name: string;
  type: 'dimension' | 'measure' | 'unknown';
}

interface BOCalc {
  id: string;
  name: string;
}

interface PreAggFilter {
  expression: string;
}

interface MaterializationConfig {
  type: 'materialized_view' | 'table';
  target_name: string;
  incremental_column?: string;
  incremental_window_days?: number;
}

interface PreAggRequest {
  tenant_id: string;
  bo_name: string;
  name: string;
  description: string;
  terms: string[];
  calculations: string[];
  group_by: string[];
  filters: PreAggFilter[];
  materialization: MaterializationConfig;
  refresh_strategy: 'manual' | 'interval' | 'incremental';
  refresh_interval_minutes: number;
}

interface PreAggregationWizardProps {
  open: boolean;
  onClose: () => void;
  boName: string;
  boId: string;
  tenantId: string;
}

// --- Main Component ---

export const PreAggregationWizard: React.FC<PreAggregationWizardProps> = ({
  open,
  onClose,
  boName,
  boId,
  tenantId,
}) => {
  const [activeStep, setActiveStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Step 1: Basics
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');

  // Step 2: Grain & Fields
  const [boTerms, setBOTerms] = useState<BOTerm[]>([]);
  const [boCalcs, setBOCalcs] = useState<BOCalc[]>([]);
  const [selectedTerms, setSelectedTerms] = useState<string[]>([]);
  const [selectedCalcs, setSelectedCalcs] = useState<string[]>([]);
  const [filterExpression, setFilterExpression] = useState('');

  // Step 3: Materialization
  const [materializationType, setMaterializationType] = useState<'materialized_view' | 'table'>('materialized_view');
  const [targetName, setTargetName] = useState('');
  const [incrementalColumn, setIncrementalColumn] = useState('');
  const [incrementalWindow, setIncrementalWindow] = useState(2);
  const [refreshStrategy, setRefreshStrategy] = useState<'manual' | 'interval' | 'incremental'>('interval');
  const [refreshInterval, setRefreshInterval] = useState(15);

  const [sqlPreview, setSqlPreview] = useState('');
  const [ddlPreview, setDdlPreview] = useState('');

  const steps = ['Basics', 'Grain & Fields', 'Materialization', 'Review & Create'];

  // Derived values
  const targetDatabase = `tenant_${tenantId}`;
  const suggestedTargetName = useMemo(() => {
    return `mv_${boName.toLowerCase()}_${name.toLowerCase().replace(/\s+/g, '_')}`;
  }, [boName, name]);

  // Load BO terms and calculations
  useEffect(() => {
    if (open && boId) {
      fetchBOMetadata();
    }
  }, [open, boId]);

  // Update suggested target name when name changes
  useEffect(() => {
    if (!targetName && name) {
      setTargetName(suggestedTargetName);
    }
  }, [name, suggestedTargetName]);

  const fetchBOMetadata = async () => {
    setLoading(true);
    try {
      // Fetch terms
      const termsRes = await fetch(`/api/semantic-graph/bo/${boId}/terms`);
      const termsData = await termsRes.json();
      setBOTerms(termsData.terms || []);

      // Fetch calculations
      const calcsRes = await fetch(`/api/semantic-graph/bo/${boId}/calculations`);
      const calcsData = await calcsRes.json();
      setBOCalcs(calcsData.calculations || []);
    } catch (e) {
      console.error('Failed to fetch BO metadata', e);
    } finally {
      setLoading(false);
    }
  };

  const fetchSQLPreview = async () => {
    if (selectedTerms.length === 0 && selectedCalcs.length === 0) return;
    try {
      const res = await fetch('/api/semantic-graph/preview-sql', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          bo_id: boId,
          terms: selectedTerms,
          calculations: selectedCalcs,
          filters: filterExpression ? [{ expression: filterExpression }] : [],
          group_by: selectedTerms,
        }),
      });
      const data = await res.json();
      setSqlPreview(data.sql || '-- Unable to generate preview');
    } catch (e) {
      setSqlPreview('-- Error generating preview');
    }
  };

  // Fetch SQL preview when fields change
  useEffect(() => {
    if (activeStep === 1) {
      fetchSQLPreview();
    }
  }, [selectedTerms, selectedCalcs, filterExpression, activeStep]);

  const handleNext = async () => {
    if (activeStep === 2) {
      // Fetch DDL preview before review step
      await fetchDDLPreview();
    }
    setActiveStep((prev) => prev + 1);
  };

  const handleBack = () => setActiveStep((prev) => prev - 1);

  const fetchDDLPreview = async () => {
    // This would call a hypothetical preview endpoint
    // For now, generate client-side
    const ddl = `CREATE MATERIALIZED VIEW ${targetDatabase}.${targetName}
BUILD IMMEDIATE
REFRESH ASYNC
AS
SELECT
    ${[...selectedTerms, ...selectedCalcs].join(',\n    ')}
FROM (
    -- BO SQL for ${boName}
    ${sqlPreview.split('\n').map(l => '    ' + l).join('\n')}
) t
${filterExpression ? `WHERE ${filterExpression}` : ''}
GROUP BY ${selectedTerms.join(', ')};`;
    setDdlPreview(ddl);
  };

  const handleCreate = async (materialize: boolean) => {
    setLoading(true);
    setError(null);

    const payload: PreAggRequest = {
      tenant_id: tenantId,
      bo_name: boName,
      name: name,
      description: description,
      terms: selectedTerms,
      calculations: selectedCalcs,
      group_by: selectedTerms,
      filters: filterExpression ? [{ expression: filterExpression }] : [],
      materialization: {
        type: materializationType,
        target_name: targetName,
        incremental_column: incrementalColumn || undefined,
        incremental_window_days: incrementalWindow,
      },
      refresh_strategy: refreshStrategy,
      refresh_interval_minutes: refreshInterval,
    };

    try {
      // Create the pre-aggregation definition
      const res = await fetch('/api/pre-aggregations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId },
        body: JSON.stringify(payload),
      });

      if (!res.ok) throw new Error(await res.text());

      const created = await res.json();

      if (materialize) {
        // Apply materialization
        const matRes = await fetch(`/api/pre-aggregations/${created.id}/materialize`, {
          method: 'POST',
          headers: { 'X-Tenant-ID': tenantId },
        });

        if (!matRes.ok) throw new Error(await matRes.text());
      }

      onClose();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  const isNextDisabled = () => {
    if (activeStep === 0 && !name.trim()) return true;
    if (activeStep === 1 && selectedTerms.length === 0) return true;
    if (activeStep === 2 && !targetName.trim()) return true;
    return loading;
  };

  const resetState = () => {
    setActiveStep(0);
    setName('');
    setDescription('');
    setSelectedTerms([]);
    setSelectedCalcs([]);
    setFilterExpression('');
    setMaterializationType('materialized_view');
    setTargetName('');
    setIncrementalColumn('');
    setRefreshStrategy('interval');
    setRefreshInterval(15);
    setError(null);
  };

  const handleClose = () => {
    onClose();
    setTimeout(resetState, 300);
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="lg" fullWidth>
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Bolt color="primary" />
        Create Pre-Aggregation for {boName}
        <IconButton onClick={handleClose} sx={{ ml: 'auto' }}>
          <Close />
        </IconButton>
      </DialogTitle>

      <DialogContent dividers sx={{ minHeight: 500 }}>
        <Stepper activeStep={activeStep} sx={{ mb: 4 }}>
          {steps.map((label) => (
            <Step key={label}>
              <StepLabel>{label}</StepLabel>
            </Step>
          ))}
        </Stepper>

        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        {loading && activeStep === 0 ? (
          <Box display="flex" justifyContent="center" py={4}>
            <CircularProgress />
          </Box>
        ) : (
          <>
            {/* Step 1: Basics */}
            {activeStep === 0 && (
              <Step1Basics
                name={name}
                setName={setName}
                description={description}
                setDescription={setDescription}
                boName={boName}
                tenantId={tenantId}
                targetDatabase={targetDatabase}
              />
            )}

            {/* Step 2: Grain & Fields */}
            {activeStep === 1 && (
              <Step2GrainFields
                boTerms={boTerms}
                boCalcs={boCalcs}
                selectedTerms={selectedTerms}
                setSelectedTerms={setSelectedTerms}
                selectedCalcs={selectedCalcs}
                setSelectedCalcs={setSelectedCalcs}
                filterExpression={filterExpression}
                setFilterExpression={setFilterExpression}
                sqlPreview={sqlPreview}
              />
            )}

            {/* Step 3: Materialization */}
            {activeStep === 2 && (
              <Step3Materialization
                materializationType={materializationType}
                setMaterializationType={setMaterializationType}
                targetName={targetName}
                setTargetName={setTargetName}
                suggestedTargetName={suggestedTargetName}
                targetDatabase={targetDatabase}
                boTerms={boTerms}
                selectedTerms={selectedTerms}
                incrementalColumn={incrementalColumn}
                setIncrementalColumn={setIncrementalColumn}
                incrementalWindow={incrementalWindow}
                setIncrementalWindow={setIncrementalWindow}
                refreshStrategy={refreshStrategy}
                setRefreshStrategy={setRefreshStrategy}
                refreshInterval={refreshInterval}
                setRefreshInterval={setRefreshInterval}
              />
            )}

            {/* Step 4: Review */}
            {activeStep === 3 && (
              <Step4Review
                boName={boName}
                tenantId={tenantId}
                name={name}
                description={description}
                targetDatabase={targetDatabase}
                targetName={targetName}
                selectedTerms={selectedTerms}
                selectedCalcs={selectedCalcs}
                filterExpression={filterExpression}
                materializationType={materializationType}
                refreshStrategy={refreshStrategy}
                refreshInterval={refreshInterval}
                incrementalColumn={incrementalColumn}
                incrementalWindow={incrementalWindow}
                ddlPreview={ddlPreview}
              />
            )}
          </>
        )}
      </DialogContent>

      <DialogActions>
        <Button onClick={handleBack} disabled={activeStep === 0 || loading}>
          Back
        </Button>
        {activeStep < steps.length - 1 ? (
          <Button onClick={handleNext} variant="contained" disabled={isNextDisabled()}>
            Next
          </Button>
        ) : (
          <>
            <Button
              onClick={() => handleCreate(false)}
              disabled={loading}
              variant="outlined"
            >
              Create Definition Only
            </Button>
            <Button
              onClick={() => handleCreate(true)}
              disabled={loading}
              variant="contained"
              color="primary"
              startIcon={<Storage />}
            >
              Create & Materialize
            </Button>
          </>
        )}
      </DialogActions>
    </Dialog>
  );
};

// --- Step Components ---

const Step1Basics: React.FC<{
  name: string;
  setName: (v: string) => void;
  description: string;
  setDescription: (v: string) => void;
  boName: string;
  tenantId: string;
  targetDatabase: string;
}> = ({ name, setName, description, setDescription, boName, tenantId, targetDatabase }) => (
  <Grid container spacing={3}>
    <Grid item xs={12} md={8}>
      <TextField
        label="Pre-Aggregation Name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        fullWidth
        required
        placeholder="e.g., fi_security_daily_pricing"
        helperText="Use snake_case for consistency"
      />
      <TextField
        label="Description"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        fullWidth
        multiline
        rows={3}
        sx={{ mt: 2 }}
        placeholder="Describe what this pre-aggregation captures..."
      />
    </Grid>
    <Grid item xs={12} md={4}>
      <Paper variant="outlined" sx={{ p: 2 }}>
        <Typography variant="subtitle2" color="text.secondary">Context</Typography>
        <Divider sx={{ my: 1 }} />
        <Typography variant="body2"><strong>BO:</strong> {boName}</Typography>
        <Typography variant="body2"><strong>Tenant:</strong> {tenantId}</Typography>
        <Typography variant="body2"><strong>Target DB:</strong> {targetDatabase}</Typography>
      </Paper>
    </Grid>
  </Grid>
);

const Step2GrainFields: React.FC<{
  boTerms: BOTerm[];
  boCalcs: BOCalc[];
  selectedTerms: string[];
  setSelectedTerms: (v: string[]) => void;
  selectedCalcs: string[];
  setSelectedCalcs: (v: string[]) => void;
  filterExpression: string;
  setFilterExpression: (v: string) => void;
  sqlPreview: string;
}> = ({
  boTerms, boCalcs, selectedTerms, setSelectedTerms,
  selectedCalcs, setSelectedCalcs, filterExpression, setFilterExpression, sqlPreview
}) => {
  const toggleTerm = (name: string) => {
    setSelectedTerms(
      selectedTerms.includes(name)
        ? selectedTerms.filter((t) => t !== name)
        : [...selectedTerms, name]
    );
  };

  const toggleCalc = (name: string) => {
    setSelectedCalcs(
      selectedCalcs.includes(name)
        ? selectedCalcs.filter((c) => c !== name)
        : [...selectedCalcs, name]
    );
  };

  return (
    <Grid container spacing={3}>
      <Grid item xs={12} md={6}>
        <Typography variant="subtitle2" gutterBottom>
          Grain (Group By Dimensions)
        </Typography>
        <Paper variant="outlined" sx={{ maxHeight: 200, overflow: 'auto' }}>
          <List dense>
            {boTerms.map((term) => (
              <ListItem key={term.id} button onClick={() => toggleTerm(term.name)}>
                <ListItemIcon>
                  <Checkbox checked={selectedTerms.includes(term.name)} edge="start" />
                </ListItemIcon>
                <ListItemText primary={term.name} />
              </ListItem>
            ))}
            {boTerms.length === 0 && (
              <ListItem><ListItemText primary="No terms found" /></ListItem>
            )}
          </List>
        </Paper>

        <Typography variant="subtitle2" sx={{ mt: 2 }} gutterBottom>
          Measures (Calculations)
        </Typography>
        <Paper variant="outlined" sx={{ maxHeight: 200, overflow: 'auto' }}>
          <List dense>
            {boCalcs.map((calc) => (
              <ListItem key={calc.id} button onClick={() => toggleCalc(calc.name)}>
                <ListItemIcon>
                  <Checkbox checked={selectedCalcs.includes(calc.name)} edge="start" />
                </ListItemIcon>
                <ListItemText primary={calc.name} />
              </ListItem>
            ))}
            {boCalcs.length === 0 && (
              <ListItem><ListItemText primary="No calculations found" /></ListItem>
            )}
          </List>
        </Paper>

        <TextField
          label="Filter Expression (optional)"
          value={filterExpression}
          onChange={(e) => setFilterExpression(e.target.value)}
          fullWidth
          sx={{ mt: 2 }}
          placeholder="e.g., pricing_date >= current_date - interval '90' day"
          helperText="SQL WHERE clause expression"
        />
      </Grid>

      <Grid item xs={12} md={6}>
        <Typography variant="subtitle2" gutterBottom>
          <Code sx={{ fontSize: 16, mr: 0.5, verticalAlign: 'text-bottom' }} />
          SQL Preview
        </Typography>
        <Paper
          variant="outlined"
          sx={{
            p: 2,
            bgcolor: 'grey.900',
            color: 'grey.100',
            fontFamily: 'monospace',
            fontSize: 12,
            whiteSpace: 'pre-wrap',
            overflow: 'auto',
            maxHeight: 400,
          }}
        >
          {sqlPreview || '-- Select terms and calculations to preview SQL'}
        </Paper>
      </Grid>
    </Grid>
  );
};

const Step3Materialization: React.FC<{
  materializationType: 'materialized_view' | 'table';
  setMaterializationType: (v: 'materialized_view' | 'table') => void;
  targetName: string;
  setTargetName: (v: string) => void;
  suggestedTargetName: string;
  targetDatabase: string;
  boTerms: BOTerm[];
  selectedTerms: string[];
  incrementalColumn: string;
  setIncrementalColumn: (v: string) => void;
  incrementalWindow: number;
  setIncrementalWindow: (v: number) => void;
  refreshStrategy: 'manual' | 'interval' | 'incremental';
  setRefreshStrategy: (v: 'manual' | 'interval' | 'incremental') => void;
  refreshInterval: number;
  setRefreshInterval: (v: number) => void;
}> = (props) => (
  <Grid container spacing={3}>
    <Grid item xs={12} md={6}>
      <FormControl component="fieldset">
        <FormLabel>Materialization Type</FormLabel>
        <RadioGroup
          value={props.materializationType}
          onChange={(e) => props.setMaterializationType(e.target.value as any)}
        >
          <FormControlLabel value="materialized_view" control={<Radio />} label="Materialized View" />
          <FormControlLabel value="table" control={<Radio />} label="Table" />
        </RadioGroup>
      </FormControl>

      <TextField
        label="Target Name"
        value={props.targetName}
        onChange={(e) => props.setTargetName(e.target.value)}
        fullWidth
        sx={{ mt: 2 }}
        helperText={`Full path: ${props.targetDatabase}.${props.targetName || props.suggestedTargetName}`}
      />

      <FormControl fullWidth sx={{ mt: 2 }}>
        <InputLabel>Incremental Column</InputLabel>
        <Select
          value={props.incrementalColumn}
          onChange={(e) => props.setIncrementalColumn(e.target.value)}
          label="Incremental Column"
        >
          <MenuItem value="">None</MenuItem>
          {props.selectedTerms.map((t) => (
            <MenuItem key={t} value={t}>{t}</MenuItem>
          ))}
        </Select>
      </FormControl>

      {props.incrementalColumn && (
        <TextField
          label="Incremental Window (days)"
          type="number"
          value={props.incrementalWindow}
          onChange={(e) => props.setIncrementalWindow(Number(e.target.value))}
          fullWidth
          sx={{ mt: 2 }}
        />
      )}
    </Grid>

    <Grid item xs={12} md={6}>
      <FormControl component="fieldset">
        <FormLabel>Refresh Strategy</FormLabel>
        <RadioGroup
          value={props.refreshStrategy}
          onChange={(e) => props.setRefreshStrategy(e.target.value as any)}
        >
          <FormControlLabel value="manual" control={<Radio />} label="Manual" />
          <FormControlLabel value="interval" control={<Radio />} label="Interval" />
          <FormControlLabel value="incremental" control={<Radio />} label="Incremental" />
        </RadioGroup>
      </FormControl>

      {props.refreshStrategy !== 'manual' && (
        <TextField
          label="Refresh Interval (minutes)"
          type="number"
          value={props.refreshInterval}
          onChange={(e) => props.setRefreshInterval(Number(e.target.value))}
          fullWidth
          sx={{ mt: 2 }}
        />
      )}

      <Alert severity="info" sx={{ mt: 3 }} icon={<Storage />}>
        This pre-aggregation will be created in StarRocks database&nbsp;
        <strong>{props.targetDatabase}</strong> as <strong>{props.targetName || props.suggestedTargetName}</strong>.
      </Alert>
    </Grid>
  </Grid>
);

const Step4Review: React.FC<{
  boName: string;
  tenantId: string;
  name: string;
  description: string;
  targetDatabase: string;
  targetName: string;
  selectedTerms: string[];
  selectedCalcs: string[];
  filterExpression: string;
  materializationType: string;
  refreshStrategy: string;
  refreshInterval: number;
  incrementalColumn: string;
  incrementalWindow: number;
  ddlPreview: string;
}> = (props) => (
  <Grid container spacing={3}>
    <Grid item xs={12} md={6}>
      <Card variant="outlined">
        <CardContent>
          <Typography variant="h6" gutterBottom>
            <CheckCircle color="success" sx={{ mr: 1, verticalAlign: 'text-bottom' }} />
            Summary
          </Typography>
          <Divider sx={{ mb: 2 }} />
          
          <Box display="grid" gridTemplateColumns="1fr 2fr" gap={1} sx={{ fontSize: 14 }}>
            <Typography color="text.secondary">BO:</Typography>
            <Typography>{props.boName}</Typography>
            
            <Typography color="text.secondary">Name:</Typography>
            <Typography>{props.name}</Typography>
            
            <Typography color="text.secondary">Tenant:</Typography>
            <Typography>{props.tenantId}</Typography>
            
            <Typography color="text.secondary">Target:</Typography>
            <Typography sx={{ fontFamily: 'monospace' }}>
              {props.targetDatabase}.{props.targetName}
            </Typography>
            
            <Typography color="text.secondary">Grain:</Typography>
            <Box>
              {props.selectedTerms.map((t) => (
                <Chip key={t} label={t} size="small" sx={{ mr: 0.5, mb: 0.5 }} />
              ))}
            </Box>
            
            <Typography color="text.secondary">Measures:</Typography>
            <Box>
              {props.selectedCalcs.map((c) => (
                <Chip key={c} label={c} size="small" color="primary" sx={{ mr: 0.5, mb: 0.5 }} />
              ))}
            </Box>
            
            {props.filterExpression && (
              <>
                <Typography color="text.secondary">Filter:</Typography>
                <Typography sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  {props.filterExpression}
                </Typography>
              </>
            )}
            
            <Typography color="text.secondary">Type:</Typography>
            <Typography>{props.materializationType}</Typography>
            
            <Typography color="text.secondary">Refresh:</Typography>
            <Typography>
              {props.refreshStrategy}
              {props.refreshStrategy !== 'manual' && ` (every ${props.refreshInterval} min)`}
              {props.incrementalColumn && `, incremental on ${props.incrementalColumn}`}
            </Typography>
          </Box>
        </CardContent>
      </Card>
    </Grid>
    
    <Grid item xs={12} md={6}>
      <Typography variant="subtitle2" gutterBottom>
        <Code sx={{ fontSize: 16, mr: 0.5, verticalAlign: 'text-bottom' }} />
        Generated DDL
      </Typography>
      <Paper
        variant="outlined"
        sx={{
          p: 2,
          bgcolor: 'grey.900',
          color: 'grey.100',
          fontFamily: 'monospace',
          fontSize: 11,
          whiteSpace: 'pre-wrap',
          overflow: 'auto',
          maxHeight: 350,
        }}
      >
        {props.ddlPreview}
      </Paper>
    </Grid>
  </Grid>
);

export default PreAggregationWizard;
