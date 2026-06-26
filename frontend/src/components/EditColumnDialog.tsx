import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Box,
  Typography,
  Alert,
  FormControlLabel,
  Checkbox,
} from '@mui/material';

interface EditColumnDialogProps {
  open: boolean;
  onClose: () => void;
  column: {
    id: string;
    name: string;
    description?: string;
    type?: string;
    properties?: any;
    tenant_id?: string;
    tenant_tenant_instance_id?: string;
  };
  onSave?: () => void;
}

const EditColumnDialog: React.FC<EditColumnDialogProps> = ({
  open,
  onClose,
  column,
  onSave
}) => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    type: '',
    nullable: true,
    isPrimaryKey: false,
    isForeignKey: false,
    defaultValue: '',
  });
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open && column) {
      setFormData({
        name: column.name || '',
        description: column.description || '',
        type: column.type || '',
        nullable: column.properties?.nullable ?? true,
        isPrimaryKey: column.properties?.is_primary_key || false,
        isForeignKey: column.properties?.is_foreign_key || false,
        defaultValue: column.properties?.default_value || '',
      });
      setError(null);
    }
  }, [open, column]);

  const handleChange = (field: string, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleSave = async () => {
    setError(null);
    setSaving(true);

    try {
      // Build properties object
      const properties = {
        ...column.properties,
        nullable: formData.nullable,
        is_primary_key: formData.isPrimaryKey,
        is_foreign_key: formData.isForeignKey,
        default_value: formData.defaultValue || undefined,
      };

      // Update column via API
      const response = await fetch(`/api/catalog/nodes/${column.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': column.tenant_id || 'default',
          'X-Tenant-Datasource-ID': column.tenant_tenant_instance_id || '',
        },
        body: JSON.stringify({
          node_name: formData.name,
          description: formData.description,
          properties,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to update column');
      }

      if (onSave) onSave();
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to save changes');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Edit Column</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          {/* Basic Info */}
          <Typography variant="subtitle2" color="text.secondary" sx={{ mt: 1 }}>
            Basic Information
          </Typography>
          
          <TextField
            label="Column Name"
            value={formData.name}
            onChange={(e) => handleChange('name', e.target.value)}
            fullWidth
            required
            helperText="Name of the database column"
          />

          <TextField
            label="Description"
            value={formData.description}
            onChange={(e) => handleChange('description', e.target.value)}
            fullWidth
            multiline
            rows={2}
            helperText="Documentation for this column"
          />

          {/* Column Properties */}
          <Typography variant="subtitle2" color="text.secondary" sx={{ mt: 2 }}>
            Column Properties
          </Typography>

          <TextField
            label="Data Type"
            value={formData.type}
            onChange={(e) => handleChange('type', e.target.value)}
            fullWidth
            helperText="Database data type (e.g., VARCHAR, INTEGER, TIMESTAMP)"
            placeholder="e.g., VARCHAR(255)"
          />

          <TextField
            label="Default Value"
            value={formData.defaultValue}
            onChange={(e) => handleChange('defaultValue', e.target.value)}
            fullWidth
            helperText="Default value for this column"
          />

          <FormControlLabel
            control={
              <Checkbox
                checked={formData.nullable}
                onChange={(e) => handleChange('nullable', e.target.checked)}
              />
            }
            label="Nullable (allows NULL values)"
          />

          <FormControlLabel
            control={
              <Checkbox
                checked={formData.isPrimaryKey}
                onChange={(e) => handleChange('isPrimaryKey', e.target.checked)}
              />
            }
            label="Primary Key"
          />

          <FormControlLabel
            control={
              <Checkbox
                checked={formData.isForeignKey}
                onChange={(e) => handleChange('isForeignKey', e.target.checked)}
              />
            }
            label="Foreign Key"
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={saving}>
          Cancel
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={saving}>
          {saving ? 'Saving...' : 'Save Changes'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default EditColumnDialog;
