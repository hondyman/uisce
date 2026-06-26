import React, { useState, useEffect, useRef } from 'react';
import {
  Drawer,
  Box,
  Typography,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Button,
  Divider,
  IconButton,
  Alert,
  CircularProgress,
  Chip,
  Stack,
  Autocomplete,
  Paper,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
} from '@mui/material';
import {
  Close as CloseIcon,
  Save as SaveIcon,
  Functions as FunctionsIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  DragIndicator as DragIcon,
} from '@mui/icons-material';
import Editor, { useMonaco } from '@monaco-editor/react';
import { useTenant } from '../../contexts/TenantContext';

// ============================================================================
// Types
// ============================================================================

interface CalculationDependency {
  term_id: string;
  term_name?: string;
  is_optional?: boolean;
}

interface Materialization {
  target_type: 'table' | 'view' | 'cube' | 'metric' | 'iceberg';
  target_name: string;
  refresh_schedule?: string;
}

interface CalculationDefinition {
  calc_id?: string;
  name: string;
  display_name: string;
  description?: string;
  expression: string;
  output_type: string;
  precision?: number;
  evaluation_mode: string;
  dependencies: CalculationDependency[];
  materialization?: Materialization;
  default_aggregation?: string;
  owner?: string;
  status?: string;
}

interface CalculationEditorDrawerProps {
  open: boolean;
  onClose: () => void;
  calcId?: string;
  boId?: string;
  onSave?: () => void;
}

const OUTPUT_TYPES = [
  { value: 'number', label: 'Number' },
  { value: 'currency', label: 'Currency' },
  { value: 'percent', label: 'Percentage' },
  { value: 'integer', label: 'Integer' },
  { value: 'string', label: 'String' },
  { value: 'date', label: 'Date' },
  { value: 'boolean', label: 'Boolean' },
];

const EVALUATION_MODES = [
  { value: 'live', label: 'Live', description: 'Computed at query time' },
  { value: 'pre_aggregated', label: 'Pre-Aggregated', description: 'Materialized for performance' },
  { value: 'hybrid', label: 'Hybrid', description: 'Cached when possible' },
  { value: 'on_demand', label: 'On-Demand', description: 'Computed when explicitly requested' },
];

const AGGREGATION_OPTIONS = ['none', 'sum', 'avg', 'min', 'max', 'count'];

const STANDARD_FUNCTIONS = [
    { label: 'sum', insertText: 'sum()', documentation: 'Calculate sum of values' },
    { label: 'avg', insertText: 'avg()', documentation: 'Calculate average of values' },
    { label: 'min', insertText: 'min()', documentation: 'Minimum value' },
    { label: 'max', insertText: 'max()', documentation: 'Maximum value' },
    { label: 'count', insertText: 'count()', documentation: 'Count of values' },
    { label: 'coalesce', insertText: 'coalesce(, )', documentation: 'Return first non-null value' },
    { label: 'round', insertText: 'round(, 2)', documentation: 'Round number to decimals' },
    { label: 'case_when', insertText: 'case_when(cond, val, else)', documentation: 'Conditional logic' },
    { label: 'date_add', insertText: 'date_add(\'day\', , )', documentation: 'Add interval to date' },
    { label: 'abs', insertText: 'abs()', documentation: 'Absolute value' },
    { label: 'cast', insertText: 'cast( as type)', documentation: 'Cast value to type' },
];

// ============================================================================
// Component
// ============================================================================

export const CalculationEditorDrawer: React.FC<CalculationEditorDrawerProps> = ({
  open,
  onClose,
  calcId,
  boId,
  onSave,
}) => {
  const { tenant } = useTenant();
  const tenantId = tenant?.id || '';
  const monaco = useMonaco();

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [availableTerms, setAvailableTerms] = useState<any[]>([]);

  const [calc, setCalc] = useState<CalculationDefinition>({
    name: '',
    display_name: '',
    description: '',
    expression: '',
    output_type: 'number',
    precision: 2,
    evaluation_mode: 'live',
    dependencies: [],
    default_aggregation: 'sum',
    status: 'draft',
  });

  const [explaining, setExplaining] = useState(false);
  const [explanation, setExplanation] = useState<any>(null);

  // Load calculation if editing
  useEffect(() => {
    if (open && calcId) {
      setLoading(true);
      fetch(`/api/calculation/${calcId}`, {
        headers: { 'X-Tenant-ID': tenantId },
      })
        .then((r) => r.json())
        .then((data) => setCalc(data))
        .catch((err) => setError(err.message))
        .finally(() => setLoading(false));
    } else if (open && !calcId) {
      // Reset for new calculation
      setCalc({
        name: '',
        display_name: '',
        description: '',
        expression: '',
        output_type: 'number',
        precision: 2,
        evaluation_mode: 'live',
        dependencies: [],
        default_aggregation: 'sum',
        status: 'draft',
      });
      setExplanation(null);
    }
  }, [open, calcId, tenantId]);

  // Load available terms for dependencies and autocomplete
  useEffect(() => {
    if (open && boId) {
      fetch(`/api/bo/${boId}/terms`, {
        headers: { 'X-Tenant-ID': tenantId },
      })
        .then((r) => r.json())
        .then((data) => setAvailableTerms(data.terms || []))
        .catch(() => setAvailableTerms([]));
    }
  }, [open, boId, tenantId]);

  // Register Monco Autocomplete
  useEffect(() => {
    if (monaco && availableTerms.length > 0) {
        const disposable = monaco.languages.registerCompletionItemProvider('sql', {
            provideCompletionItems: (model, position) => {
                const word = model.getWordUntilPosition(position);
                const range = {
                    startLineNumber: position.lineNumber,
                    endLineNumber: position.lineNumber,
                    startColumn: word.startColumn,
                    endColumn: word.endColumn,
                };

                const suggestions = [
                    ...availableTerms.map(t => ({
                        label: t.term_name,
                        kind: monaco.languages.CompletionItemKind.Field,
                        insertText: t.term_name,
                        documentation: t.display_name || t.description,
                        detail: t.data_type,
                        range: range,
                    })),
                    ...STANDARD_FUNCTIONS.map(f => ({
                        label: f.label,
                        kind: monaco.languages.CompletionItemKind.Function,
                        insertText: f.insertText,
                        documentation: f.documentation,
                        range: range,
                    }))
                ];

                return { suggestions };
            }
        });
        return () => disposable.dispose();
    }
  }, [monaco, availableTerms]);

  const handleChange = (field: keyof CalculationDefinition, value: any) => {
    setCalc((prev) => ({ ...prev, [field]: value }));
  };

  const handleAddDependency = (term: any) => {
    if (!term || calc.dependencies.some((d) => d.term_id === term.term_id)) return;
    
    setCalc((prev) => ({
      ...prev,
      dependencies: [
        ...prev.dependencies,
        { term_id: term.term_id, term_name: term.term_name || term.display_name },
      ],
    }));
  };

  const handleRemoveDependency = (termId: string) => {
    setCalc((prev) => ({
      ...prev,
      dependencies: prev.dependencies.filter((d) => d.term_id !== termId),
    }));
  };

  const handleExplain = async () => {
    if (!calc.expression) return;
    setExplaining(true);
    setExplanation(null);
    try {
      const response = await fetch('/api/calculation/explain', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({
           expression: calc.expression,
           bo_id: boId
        }),
      });
      if (!response.ok) throw new Error('Explanation failed');
      const data = await response.json();
      setExplanation(data);
    } catch (e) {
      // ignore
    } finally {
      setExplaining(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);

    try {
      const method = calcId ? 'PATCH' : 'POST';
      const url = calcId ? `/api/calculation/${calcId}` : '/api/calculation';
      
      const payload = { ...calc, domain_id: boId };

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        try {
          const errData = await response.json();
          if (errData.errors && Array.isArray(errData.errors)) {
             setError(errData.errors.join("; "));
             return;
          }
           throw new Error(errData.message || 'Failed to save');
        } catch (e) {
           const errText = await response.text();
           throw new Error(errText || 'Failed to save calculation');
        }
      }

      onSave?.();
      onClose();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  const isNumericOutput = ['number', 'currency', 'percent', 'integer'].includes(calc.output_type);

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      PaperProps={{ sx: { width: 520, p: 0 } }}
    >
      {/* Header */}
      <Box
        sx={{
          p: 2,
          borderBottom: 1,
          borderColor: 'divider',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          bgcolor: 'background.default',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <FunctionsIcon color="primary" />
          <Typography variant="h6">
            {calcId ? 'Edit Calculation' : 'New Calculation'}
          </Typography>
        </Box>
        <IconButton onClick={onClose}>
          <CloseIcon />
        </IconButton>
      </Box>

      {/* Content */}
      <Box sx={{ p: 3, overflowY: 'auto', flex: 1 }}>
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
            <CircularProgress />
          </Box>
        )}

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {!loading && (
          <Stack spacing={3}>
            {/* Basic Info */}
            <Typography variant="subtitle2" color="text.secondary">
              Basic Information
            </Typography>

            <TextField
              label="Technical Name"
              value={calc.name}
              onChange={(e) => handleChange('name', e.target.value)}
              fullWidth
              helperText="Alphanumeric with underscores (e.g., profit_margin)"
            />

            <TextField
              label="Display Name"
              value={calc.display_name}
              onChange={(e) => handleChange('display_name', e.target.value)}
              fullWidth
            />

            <TextField
              label="Description"
              value={calc.description || ''}
              onChange={(e) => handleChange('description', e.target.value)}
              fullWidth
              multiline
              rows={2}
            />

            <Divider />

            {/* Expression */}
            <Typography variant="subtitle2" color="text.secondary">
              Semantic Expression
            </Typography>

            <Box>
                <Paper variant="outlined" sx={{ border: '1px solid #ccc', borderRadius: 1, overflow: 'hidden' }}>
                    <Editor
                        height="150px"
                        defaultLanguage="sql"
                        value={calc.expression}
                        onChange={(value) => handleChange('expression', value || '')}
                        options={{
                            minimap: { enabled: false },
                            lineNumbers: 'off',
                            scrollBeyondLastLine: false,
                            fontSize: 14,
                            padding: { top: 8, bottom: 8 },
                        }}
                    />
                </Paper>
                <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
                    Use semantic term names (e.g., revenue, cost). Ctrl+Space for suggestions.
                </Typography>
             
             <Box sx={{ mt: 1 }}>
                <Button 
                    size="small" 
                    variant="outlined" 
                    onClick={handleExplain}
                    disabled={explaining || !calc.expression}
                    startIcon={explaining ? <CircularProgress size={12}/> : <FunctionsIcon/>}
                >
                    Check Syntax & Type
                </Button>
             </Box>
            </Box>
            
            {explanation && (
                <Paper variant="outlined" sx={{ p: 2, bgcolor: 'action.hover' }}>
                    <Typography variant="subtitle2" color="primary">Analysis Result</Typography>
                    <Typography variant="body2"><strong>Inferred Type:</strong> {explanation.inferred_type}</Typography>
                    <Typography variant="body2"><strong>Is Aggregate:</strong> {explanation.is_aggregate ? 'Yes' : 'No'}</Typography>
                </Paper>
            )}

            {/* Dependencies */}
            <Typography variant="subtitle2" color="text.secondary">
              Dependencies
            </Typography>

            <Autocomplete
              options={availableTerms}
              getOptionLabel={(opt) => opt.display_name || opt.term_name}
              onChange={(_, value) => handleAddDependency(value)}
              renderInput={(params) => (
                <TextField {...params} label="Add Dependent Term" placeholder="Search terms..." />
              )}
            />

            {calc.dependencies.length > 0 && (
              <Paper variant="outlined" sx={{ p: 1 }}>
                <List dense>
                  {calc.dependencies.map((dep) => (
                    <ListItem
                      key={dep.term_id}
                      secondaryAction={
                        <IconButton size="small" onClick={() => handleRemoveDependency(dep.term_id)}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      }
                    >
                      <ListItemIcon><DragIcon /></ListItemIcon>
                      <ListItemText
                        primary={dep.term_name || dep.term_id}
                        secondary={dep.is_optional ? 'Optional' : undefined}
                      />
                    </ListItem>
                  ))}
                </List>
              </Paper>
            )}

            <Divider />

            {/* Output & Evaluation */}
            <Typography variant="subtitle2" color="text.secondary">
              Output Configuration
            </Typography>

            <FormControl fullWidth>
              <InputLabel>Output Type</InputLabel>
              <Select
                value={calc.output_type}
                label="Output Type"
                onChange={(e) => handleChange('output_type', e.target.value)}
              >
                {OUTPUT_TYPES.map((opt) => (
                  <MenuItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            {isNumericOutput && (
              <TextField
                label="Precision"
                type="number"
                value={calc.precision}
                onChange={(e) => handleChange('precision', parseInt(e.target.value) || 0)}
                inputProps={{ min: 0, max: 10 }}
                fullWidth
              />
            )}

            <FormControl fullWidth>
              <InputLabel>Evaluation Mode</InputLabel>
              <Select
                value={calc.evaluation_mode}
                label="Evaluation Mode"
                onChange={(e) => handleChange('evaluation_mode', e.target.value)}
              >
                {EVALUATION_MODES.map((opt) => (
                  <MenuItem key={opt.value} value={opt.value}>
                    <Box>
                      <Typography variant="body2">{opt.label}</Typography>
                      <Typography variant="caption" color="text.secondary">
                        {opt.description}
                      </Typography>
                    </Box>
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            {isNumericOutput && (
              <FormControl fullWidth>
                <InputLabel>Default Aggregation</InputLabel>
                <Select
                  value={calc.default_aggregation}
                  label="Default Aggregation"
                  onChange={(e) => handleChange('default_aggregation', e.target.value)}
                >
                  {AGGREGATION_OPTIONS.map((opt) => (
                    <MenuItem key={opt} value={opt}>
                      {opt.charAt(0).toUpperCase() + opt.slice(1)}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            )}
          </Stack>
        )}
      </Box>

      {/* Footer */}
      <Box
        sx={{
          p: 2,
          borderTop: 1,
          borderColor: 'divider',
          display: 'flex',
          justifyContent: 'flex-end',
          gap: 2,
        }}
      >
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          onClick={handleSave}
          disabled={saving || !calc.name || !calc.display_name || !calc.expression}
          startIcon={saving ? <CircularProgress size={16} /> : <SaveIcon />}
        >
          {saving ? 'Saving...' : 'Save Calculation'}
        </Button>
      </Box>
    </Drawer>
  );
};

export default CalculationEditorDrawer;
