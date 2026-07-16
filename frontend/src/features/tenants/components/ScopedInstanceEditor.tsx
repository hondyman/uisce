import React, { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  CardActions,
  Stack,
  Typography,
  TextField,
  FormControlLabel,
  Switch,
  Button,
  CircularProgress,
  Alert,
  Box,
  Divider,
} from '@mui/material';
import { apiClient } from '../../../utils/apiClient';

export const ScopedInstanceEditor = ({ instance, tenantId, updateMutation }: any) => {
  const [isEditing, setIsEditing] = useState(false);
  const [form, setForm] = useState({
    instance_name: instance.instance_name || '',
    display_name: instance.display_name || '',
    description: instance.description || '',
    url: instance.url || '',
    is_active: instance.is_active,
  });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setForm({
      instance_name: instance.instance_name || '',
      display_name: instance.display_name || '',
      description: instance.description || '',
      url: instance.url || '',
      is_active: instance.is_active,
    });
    // Reset editing state when instance changes
    setIsEditing(false);
  }, [instance]);

  const handleSave = async () => {
    setSaving(true);
    try {
      // 1. Save the instance itself
      await updateMutation({
        ...form,
        id: instance.id,
        tenant_id: tenantId,
      });

      // Note: Backend now atomically handles cascading deactivation of products and datasources
      // when instance.is_active transitions from true to false.
      setIsEditing(false);
    } catch (error) {
      console.error('Failed to save instance', error);
    } finally {
      setSaving(false);
    }
  };

  const handleCancel = () => {
    setForm({
      instance_name: instance.instance_name || '',
      display_name: instance.display_name || '',
      description: instance.description || '',
      url: instance.url || '',
      is_active: instance.is_active,
    });
    setIsEditing(false);
  };

  if (!isEditing) {
    return (
      <Card variant="outlined">
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3 }}>
            <Typography variant="h6">Scoped Instance Properties</Typography>
            <Button variant="outlined" onClick={() => setIsEditing(true)}>
              Edit Properties
            </Button>
          </Box>
          <Divider sx={{ mb: 3 }} />
          <Stack spacing={3}>
            <Box>
              <Typography variant="caption" color="text.secondary">Instance Name</Typography>
              <Typography variant="body1">{instance.instance_name || '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary">Display Name</Typography>
              <Typography variant="body1">{instance.display_name || '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary">URL</Typography>
              <Typography variant="body1">{instance.url || '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary">Description</Typography>
              <Typography variant="body1">{instance.description || '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary">Status</Typography>
              <Typography variant="body1" sx={{ color: instance.is_active ? 'success.main' : 'error.main', fontWeight: 'bold' }}>
                {instance.is_active ? 'Active' : 'Inactive'}
              </Typography>
            </Box>
          </Stack>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={3}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography variant="h6">Edit Scoped Instance Properties</Typography>
          </Box>
          <Divider />
          
          {!form.is_active && instance.is_active && (
            <Alert severity="warning">
              Making this instance inactive will also cascade to deactivate all its products and datasources. We cannot have anything active on an inactive instance.
            </Alert>
          )}

          <Stack spacing={2}>
            <TextField
              label="Instance Name"
              value={form.instance_name}
              onChange={(e) => setForm({ ...form, instance_name: e.target.value })}
              fullWidth
            />
            <TextField
              label="Display Name"
              value={form.display_name}
              onChange={(e) => setForm({ ...form, display_name: e.target.value })}
              fullWidth
            />
            <TextField
              label="URL"
              value={form.url}
              onChange={(e) => setForm({ ...form, url: e.target.value })}
              fullWidth
            />
            <TextField
              label="Description"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              fullWidth
              multiline
              rows={2}
            />
            <FormControlLabel
              control={
                <Switch
                  checked={form.is_active}
                  onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                />
              }
              label="Active"
            />
          </Stack>
        </Stack>
      </CardContent>
      <CardActions sx={{ px: 3, pb: 3, pt: 0, gap: 1, justifyContent: 'flex-end' }}>
        <Button 
          variant="outlined" 
          onClick={handleCancel} 
          disabled={saving}
          color="inherit"
        >
          Cancel
        </Button>
        <Button 
          variant="contained" 
          onClick={handleSave} 
          disabled={saving}
          startIcon={saving ? <CircularProgress size={20} /> : undefined}
        >
          {saving ? 'Saving...' : 'Save Changes'}
        </Button>
      </CardActions>
    </Card>
  );
};
