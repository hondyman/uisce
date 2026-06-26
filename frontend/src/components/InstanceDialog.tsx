import { useState, useEffect } from 'react';
import {
  Dialog, DialogContent, DialogActions, Button,
  TextField, FormControlLabel, Switch, Box, Typography
} from '@mui/material';
import ModalHeader from '@/components/ModalHeader';
import MonacoCodeEditor from './UnifiedSemanticBuilder/MonacoCodeEditor.lazy';
import { TenantInstance } from '../types';

export interface InstanceFormData {
  id?: string;
  tenant_id?: string;
  display_name: string;
  description: string;
  url: string;
  is_active: boolean;
  config: string;
}

interface InstanceDialogProps {
  open: boolean;
  instance: Partial<TenantInstance> & { tenant_id?: string } | null;
  onClose: () => void;
  onSave: (data: InstanceFormData) => Promise<void>;
}

export const InstanceDialog: React.FC<InstanceDialogProps> = ({ open, instance, onClose, onSave }) => {
  const [formData, setFormData] = useState<Omit<InstanceFormData, 'config'>>({
    display_name: '',
    description: '',
    url: '',
    is_active: true,
  });
  const [config, setConfig] = useState<string>('{}');

  const isEditMode = !!instance && !!instance.id;

  useEffect(() => {
    if (open) {
      setFormData({
        // Use `display_name` consistently (backend returns `display_name`)
        display_name: instance?.display_name ?? instance?.instance_name ?? '',
        description: instance?.description || '',
        url: instance?.url || '',
        // CORRECTED: Safely handles potentially undefined is_active with a fallback
        is_active: instance?.is_active ?? true,
      });
      const configString = instance?.config ? JSON.stringify(instance.config, null, 2) : '{}';
      setConfig(configString);
    }
  }, [open, instance]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value, checked, type } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }));
  };

  const handleSave = () => {
    const saveData: InstanceFormData = {
      ...formData,
      config: config,
    };
    if (isEditMode) {
      saveData.id = instance!.id;
    } else {
      saveData.tenant_id = instance!.tenant_id;
    }
    onSave(saveData);
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <ModalHeader title={isEditMode ? 'Edit Tenant Instance' : 'Add New Instance'} onClose={onClose} />
      <DialogContent>
        <Box component="form" sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
          <TextField
            autoFocus
            required
            name="display_name"
            label="Instance Display Name"
            value={formData.display_name}
            onChange={handleChange}
            fullWidth
          />
          <TextField
            name="url"
            label="URL"
            value={formData.url}
            onChange={handleChange}
            fullWidth
          />
          <TextField
            name="description"
            label="Description"
            value={formData.description}
            onChange={handleChange}
            fullWidth
            multiline
            rows={3}
          />
          <FormControlLabel
            control={
              <Switch
                name="is_active"
                checked={formData.is_active}
                onChange={handleChange}
              />
            }
            label="Active"
          />
          <Typography variant="subtitle1" sx={{ mt: 1 }}>Configuration (JSON)</Typography>
          <Box border={1} borderColor="grey.400" borderRadius={1} sx={{ height: 300 }}>
            <div className="editor-wrapper-full editor-h-400">
              <MonacoCodeEditor value={config} language="json" onChange={(val: string) => setConfig(val)} />
            </div>
          </Box>
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained" disabled={!formData.display_name.trim()}>
          {isEditMode ? 'Update' : 'Create'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default InstanceDialog;