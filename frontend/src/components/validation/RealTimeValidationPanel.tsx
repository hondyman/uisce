import React, { useState, useEffect as _useEffect } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Grid,
  TextField,
  Typography,
  Alert,
  LinearProgress as _LinearProgress,
  Chip,
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';

const useStyles = makeStyles({
  container: {
    padding: '20px',
  },
  resultCard: {
    marginTop: '20px',
    backgroundColor: '#f5f5f5',
  },
  successResult: {
    backgroundColor: '#e8f5e9',
    borderLeft: '4px solid #4caf50',
  },
  errorResult: {
    backgroundColor: '#ffebee',
    borderLeft: '4px solid #f44336',
  },
  warningResult: {
    backgroundColor: '#fff3e0',
    borderLeft: '4px solid #ff9800',
  },
});

interface ValidationResult {
  passed: boolean;
  errors: string[];
  warnings: string[];
  execution_time_ms: number;
  actions_to_take: string[];
}

const RealTimeValidationPanel: React.FC = () => {
  const classes = useStyles();
  const [bpName, setBpName] = useState('ChangeMaritalStatus');
  const [stepName, setStepName] = useState('Submit');
  const [formData, setFormData] = useState<Record<string, string>>({
    age: '25',
    marital_status: 'single',
    email: 'user@example.com',
  });
  const [formField, setFormField] = useState('');
  const [formFieldValue, setFormFieldValue] = useState('');
  const [result, setResult] = useState<ValidationResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const getTenantContext = () => {
    const tenantId = localStorage.getItem('selected_tenant')
      ? JSON.parse(localStorage.getItem('selected_tenant') || '{}').id
      : null;
    const datasourceId = localStorage.getItem('selected_datasource')
      ? JSON.parse(localStorage.getItem('selected_datasource') || '{}').id
      : null;
    return { tenantId, datasourceId };
  };

  const handleAddFormField = () => {
    if (formField && formFieldValue) {
      setFormData((prev) => ({
        ...prev,
        [formField]: formFieldValue,
      }));
      setFormField('');
      setFormFieldValue('');
    }
  };

  const handleRemoveFormField = (key: string) => {
    setFormData((prev) => {
      const newData = { ...prev };
      delete newData[key];
      return newData;
    });
  };

  const handleValidate = async () => {
    try {
      setLoading(true);
      setError(null);
      const { tenantId, datasourceId } = getTenantContext();

      if (!tenantId || !datasourceId) {
        setError('Please select a tenant and datasource first');
        return;
      }

      const payload = {
        tenant_id: tenantId,
        bp_name: bpName,
        step_name: stepName,
        user_id: 'current_user',
        return_sync: true,
        form_data: formData,
      };

      const response = await fetch(
        `/api/validations/validate?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          body: JSON.stringify(payload),
        }
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || `Validation failed: ${response.statusText}`);
      }

      const data = await response.json();
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Validation failed');
      setResult(null);
    } finally {
      setLoading(false);
    }
  };

  const getResultClassName = () => {
    if (!result) return '';
    if (result.passed) return classes.successResult;
    if (result.errors.length > 0) return classes.errorResult;
    if (result.warnings.length > 0) return classes.warningResult;
    return '';
  };

  return (
    <Box className={classes.container}>
      <Typography variant="h6" sx={{ mb: 3 }}>
        Real-Time Validation
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6}>
          <TextField
            fullWidth
            label="Business Process"
            value={bpName}
            onChange={(e) => setBpName(e.target.value)}
            placeholder="e.g., ChangeMaritalStatus"
          />
        </Grid>
        <Grid item xs={12} sm={6}>
          <TextField
            fullWidth
            label="Process Step"
            value={stepName}
            onChange={(e) => setStepName(e.target.value)}
            placeholder="e.g., Submit"
          />
        </Grid>
      </Grid>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle2" sx={{ mb: 2 }}>
            Form Data
          </Typography>

          <Box sx={{ mb: 2, display: 'flex', gap: 1, flexWrap: 'wrap' }}>
            {Object.entries(formData).map(([key, val]) => (
              <Chip
                key={key}
                label={`${key}: ${val}`}
                onDelete={() => handleRemoveFormField(key)}
                variant="outlined"
              />
            ))}
          </Box>

          <Grid container spacing={1} sx={{ mb: 2 }}>
            <Grid item xs={12} sm={5}>
              <TextField
                fullWidth
                label="Field Name"
                size="small"
                value={formField}
                onChange={(e) => setFormField(e.target.value)}
                placeholder="e.g., age"
              />
            </Grid>
            <Grid item xs={12} sm={5}>
              <TextField
                fullWidth
                label="Field Value"
                size="small"
                value={formFieldValue}
                onChange={(e) => setFormFieldValue(e.target.value)}
                placeholder="e.g., 25"
              />
            </Grid>
            <Grid item xs={12} sm={2}>
              <Button
                fullWidth
                variant="outlined"
                onClick={handleAddFormField}
                sx={{ height: '40px' }}
              >
                Add
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      <Button
        variant="contained"
        size="large"
        startIcon={loading ? <CircularProgress size={20} /> : <PlayArrowIcon />}
        onClick={handleValidate}
        disabled={loading || !bpName || !stepName || Object.keys(formData).length === 0}
      >
        {loading ? 'Validating...' : 'Run Validation'}
      </Button>

      {result && (
        <Card className={`${classes.resultCard} ${getResultClassName()}`}>
          <CardContent>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="h6">
                {result.passed ? '✓ Validation Passed' : '✗ Validation Failed'}
              </Typography>
              <Chip
                label={`${result.execution_time_ms}ms`}
                size="small"
                variant="outlined"
              />
            </Box>

            {result.errors.length > 0 && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" sx={{ color: '#d32f2f', mb: 1 }}>
                  Errors ({result.errors.length})
                </Typography>
                {result.errors.map((err, idx) => (
                  <Typography key={idx} variant="body2" sx={{ ml: 2, color: '#d32f2f' }}>
                    • {err}
                  </Typography>
                ))}
              </Box>
            )}

            {result.warnings.length > 0 && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" sx={{ color: '#f57c00', mb: 1 }}>
                  Warnings ({result.warnings.length})
                </Typography>
                {result.warnings.map((warn, idx) => (
                  <Typography key={idx} variant="body2" sx={{ ml: 2, color: '#f57c00' }}>
                    • {warn}
                  </Typography>
                ))}
              </Box>
            )}

            {result.actions_to_take.length > 0 && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" sx={{ color: '#1976d2', mb: 1 }}>
                  Actions to Take
                </Typography>
                {result.actions_to_take.map((action, idx) => (
                  <Chip
                    key={idx}
                    label={action}
                    size="small"
                    variant="outlined"
                    sx={{ mr: 1, mb: 1 }}
                  />
                ))}
              </Box>
            )}
          </CardContent>
        </Card>
      )}
    </Box>
  );
};

export default RealTimeValidationPanel;
