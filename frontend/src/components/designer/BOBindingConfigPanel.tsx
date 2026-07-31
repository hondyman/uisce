import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  TextField,
  Button,
  Grid,
  Chip,
  Alert
} from '@mui/material';
import StorageIcon from '@mui/icons-material/Storage';
import SaveIcon from '@mui/icons-material/Save';

export type BindingMode = 'OLTP_CRUD' | 'OLAP_READONLY' | 'BI_TEMPORAL_OLAP';

export interface BusinessObjectBinding {
  binding_id?: string;
  tenant_id?: string;
  bo_id: string;
  binding_name: string;
  binding_mode: BindingMode;
  datasource_id: string;
  physical_table_name: string;
  valid_time_start_col?: string;
  valid_time_end_col?: string;
  transaction_time_start_col?: string;
  transaction_time_end_col?: string;
  is_primary?: boolean;
}

interface BOBindingConfigPanelProps {
  boId?: string;
  tenantId?: string;
  onSaveSuccess?: () => void;
}

export const BOBindingConfigPanel: React.FC<BOBindingConfigPanelProps> = ({
  boId = 'customers',
  tenantId = 'core',
  onSaveSuccess,
}) => {
  const [binding, setBinding] = useState<BusinessObjectBinding>({
    bo_id: boId,
    binding_name: 'primary_bitemporal_binding',
    binding_mode: 'BI_TEMPORAL_OLAP',
    datasource_id: 'ds_iceberg_lakehouse',
    physical_table_name: 'lakehouse.customer_snapshots',
    valid_time_start_col: 'effective_from',
    valid_time_end_col: 'effective_to',
    transaction_time_start_col: 'sys_start_time',
    transaction_time_end_col: 'sys_end_time',
    is_primary: true,
  });

  const [loading, setLoading] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  const fetchBindings = () => {
    fetch(`/api/business-objects/bindings?bo_id=${boId}&tenant_id=${tenantId}`)
      .then((res) => res.json())
      .then((data) => {
        if (data.bindings && data.bindings.length > 0) {
          setBinding(data.bindings[0]);
        }
      })
      .catch(() => {});
  };

  useEffect(() => {
    fetchBindings();
  }, [boId, tenantId]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setStatusMessage(null);

    try {
      const res = await fetch('/api/business-objects/bindings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...binding, tenant_id: tenantId, bo_id: boId }),
      });

      if (res.ok) {
        setStatusMessage('Polyglot binding configuration saved successfully!');
        if (onSaveSuccess) onSaveSuccess();
      } else {
        const errorData = await res.json();
        setStatusMessage(`Error: ${errorData.message || 'Failed to save binding'}`);
      }
    } catch (err: any) {
      setStatusMessage(`Error: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Paper component="form" onSubmit={handleSubmit} sx={{ p: 3, bgcolor: '#1e293b', border: '1px solid #334155', color: '#f8fafc' }}>
      <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
        <Box display="flex" alignItems="center" gap={1}>
          <StorageIcon sx={{ color: '#38bdf8' }} />
          <Typography variant="h6" fontWeight="600">
            Polyglot & Bi-Temporal Binding Designer ({boId})
          </Typography>
        </Box>
        <Chip
          label={binding.binding_mode}
          color={binding.binding_mode === 'BI_TEMPORAL_OLAP' ? 'info' : binding.binding_mode === 'OLTP_CRUD' ? 'success' : 'warning'}
          size="small"
        />
      </Box>

      <Typography variant="body2" color="#94a3b8" mb={3}>
        Configure dual-binding engine routing: map OLTP CRUD for write operations and Bi-Temporal Iceberg/OLAP for point-in-time analytical queries.
      </Typography>

      {statusMessage && (
        <Alert severity={statusMessage.startsWith('Error') ? 'error' : 'success'} sx={{ mb: 2 }} onClose={() => setStatusMessage(null)}>
          {statusMessage}
        </Alert>
      )}

      <Grid container spacing={2} mb={3}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            label="Binding Name"
            value={binding.binding_name}
            onChange={(e) => setBinding({ ...binding, binding_name: e.target.value })}
            required
            fullWidth
            size="small"
            sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
          />
        </Grid>

        <Grid size={{ xs: 12, sm: 6 }}>
          <FormControl fullWidth size="small" sx={{ bgcolor: '#0f172a' }}>
            <InputLabel sx={{ color: '#94a3b8' }}>Binding Mode</InputLabel>
            <Select
              value={binding.binding_mode}
              onChange={(e) => setBinding({ ...binding, binding_mode: e.target.value as BindingMode })}
              sx={{ color: '#fff' }}
            >
              <MenuItem value="OLTP_CRUD">OLTP CRUD (Writeable Operational)</MenuItem>
              <MenuItem value="OLAP_READONLY">OLAP Read-Only (Analytical)</MenuItem>
              <MenuItem value="BI_TEMPORAL_OLAP">Bi-Temporal Datalake (Iceberg/StarRocks)</MenuItem>
            </Select>
          </FormControl>
        </Grid>

        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            label="Datasource ID / Target"
            value={binding.datasource_id}
            onChange={(e) => setBinding({ ...binding, datasource_id: e.target.value })}
            required
            fullWidth
            size="small"
            sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
          />
        </Grid>

        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            label="Physical Table Name"
            value={binding.physical_table_name}
            onChange={(e) => setBinding({ ...binding, physical_table_name: e.target.value })}
            required
            fullWidth
            size="small"
            sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
          />
        </Grid>
      </Grid>

      {/* Bi-Temporal Column Mapping Section */}
      {binding.binding_mode === 'BI_TEMPORAL_OLAP' && (
        <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #0284c7', mb: 3 }}>
          <Typography variant="subtitle2" color="#38bdf8" fontWeight="600" mb={2}>
            Bi-Temporal Boundary Column Mappings
          </Typography>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, sm: 6 }}>
              <TextField
                label="Valid Time Start Column (Effective From)"
                placeholder="e.g. effective_from"
                value={binding.valid_time_start_col || ''}
                onChange={(e) => setBinding({ ...binding, valid_time_start_col: e.target.value })}
                required
                fullWidth
                size="small"
                sx={{ bgcolor: '#1e293b', input: { color: '#fff' } }}
              />
            </Grid>
            <Grid size={{ xs: 12, sm: 6 }}>
              <TextField
                label="Valid Time End Column (Effective To)"
                placeholder="e.g. effective_to"
                value={binding.valid_time_end_col || ''}
                onChange={(e) => setBinding({ ...binding, valid_time_end_col: e.target.value })}
                fullWidth
                size="small"
                sx={{ bgcolor: '#1e293b', input: { color: '#fff' } }}
              />
            </Grid>
            <Grid size={{ xs: 12, sm: 6 }}>
              <TextField
                label="Transaction Time Start Column (System Log)"
                placeholder="e.g. sys_start_time"
                value={binding.transaction_time_start_col || ''}
                onChange={(e) => setBinding({ ...binding, transaction_time_start_col: e.target.value })}
                fullWidth
                size="small"
                sx={{ bgcolor: '#1e293b', input: { color: '#fff' } }}
              />
            </Grid>
            <Grid size={{ xs: 12, sm: 6 }}>
              <TextField
                label="Transaction Time End Column (System End)"
                placeholder="e.g. sys_end_time"
                value={binding.transaction_time_end_col || ''}
                onChange={(e) => setBinding({ ...binding, transaction_time_end_col: e.target.value })}
                fullWidth
                size="small"
                sx={{ bgcolor: '#1e293b', input: { color: '#fff' } }}
              />
            </Grid>
          </Grid>
        </Paper>
      )}

      <Box display="flex" justifyContent="flex-end">
        <Button
          type="submit"
          variant="contained"
          startIcon={<SaveIcon />}
          disabled={loading}
          sx={{ bgcolor: '#0284c7', '&:hover': { bgcolor: '#0369a1' } }}
        >
          Save Binding Configuration
        </Button>
      </Box>
    </Paper>
  );
};
