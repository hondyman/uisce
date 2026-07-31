import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Grid,
  Chip,
  Alert
} from '@mui/material';
import LockIcon from '@mui/icons-material/Lock';
import SaveIcon from '@mui/icons-material/Save';
import { PageComponent } from '../../types/pageDesigner';
import { usePageContextStore } from '../../store/usePageContextStore';

interface DynamicBOFormProps {
  component: PageComponent;
}

export const DynamicBOForm: React.FC<DynamicBOFormProps> = ({ component }) => {
  const contextMap = usePageContextStore((state) => state.contextMap);
  const overrideMap = usePageContextStore((state) => state.overrideMap);
  const activeSelectedId = contextMap['selected_account_id'];

  // Capability inspection: OLTP mutable vs OLAP read-only
  const isMutable = component.config?.is_mutable !== false;
  const isRuleDisabled = overrideMap[component.id]?.disabled;
  const isRuleHidden = overrideMap[component.id]?.hidden;

  const [formData, setFormData] = useState({
    account_id: activeSelectedId || 'ACC-99812',
    name: 'Acme Wealth Management',
    region: 'North America',
    status: contextMap['selected_account_status'] || 'ACTIVE',
    notes: 'Primary institutional wealth account.',
  });

  useEffect(() => {
    if (activeSelectedId) {
      setFormData((prev) => ({
        ...prev,
        account_id: activeSelectedId,
        status: contextMap['selected_account_status'] || prev.status,
      }));
    }
  }, [activeSelectedId, contextMap]);

  if (isRuleHidden) {
    return null;
  }

  return (
    <Paper sx={{ p: 3, bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
      <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
        <Typography variant="h6" fontWeight="600">
          {component.title}
        </Typography>
        {!isMutable ? (
          <Chip icon={<LockIcon />} label="OLAP Read-Only Detail View" color="warning" size="small" />
        ) : (
          <Chip label="OLTP Mutable Form" color="success" size="small" />
        )}
      </Box>

      {isRuleDisabled && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          Editing is disabled by active Page Governance Rules.
        </Alert>
      )}

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            label="Account ID"
            value={formData.account_id}
            disabled
            fullWidth
            size="small"
            sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            label="Entity Name"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            disabled={!isMutable || isRuleDisabled}
            fullWidth
            size="small"
            sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            label="Region"
            value={formData.region}
            onChange={(e) => setFormData({ ...formData, region: e.target.value })}
            disabled={!isMutable || isRuleDisabled}
            fullWidth
            size="small"
            sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            label="Status"
            value={formData.status}
            onChange={(e) => setFormData({ ...formData, status: e.target.value })}
            disabled={!isMutable || isRuleDisabled}
            fullWidth
            size="small"
            sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
          />
        </Grid>
        <Grid size={{ xs: 12 }}>
          <TextField
            label="Notes"
            value={formData.notes}
            onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
            disabled={!isMutable || isRuleDisabled}
            multiline
            rows={2}
            fullWidth
            size="small"
            sx={{ bgcolor: '#0f172a', textarea: { color: '#fff' } }}
          />
        </Grid>
      </Grid>

      {isMutable && !isRuleDisabled && (
        <Box display="flex" justifyContent="flex-end" mt={3}>
          <Button variant="contained" startIcon={<SaveIcon />} sx={{ bgcolor: '#0284c7', '&:hover': { bgcolor: '#0369a1' } }}>
            Save Changes
          </Button>
        </Box>
      )}
    </Paper>
  );
};
