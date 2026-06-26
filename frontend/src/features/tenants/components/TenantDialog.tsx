import { useState, useEffect } from 'react';
import {
  Dialog, DialogContent, DialogActions,
  Button, TextField, Box, FormControlLabel, Switch
} from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import type { Tenant } from '../../../types';

interface TenantDialogProps {
  open: boolean;
  tenant: Tenant | null;
  onClose: () => void;
  onSave: (data: Partial<Tenant>) => Promise<void>;
}

export const TenantDialog: React.FC<TenantDialogProps> = ({ open, tenant, onClose, onSave }) => {
  const [displayName, setDisplayName] = useState('');
  const [description, setDescription] = useState('');
  const [isActive, setIsActive] = useState(true);

  const isEditMode = !!tenant;

  useEffect(() => {
    if (open) {
      setDisplayName(tenant?.display_name || '');
      setDescription(tenant?.description || '');
      setIsActive(tenant ? tenant.is_active : true);
    }
  }, [open, tenant]);

  const handleSave = () => {
    const tenantData: Partial<Tenant> = {
      display_name: displayName,
      description: description,
      is_active: isActive,
    };
    if (isEditMode) {
      tenantData.id = tenant!.id;
    }
    onSave(tenantData);
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
  <ModalHeader title={isEditMode ? 'Edit Tenant' : 'Add New Tenant'} onClose={onClose} />
      <DialogContent>
        <Box component="form" sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
          <TextField
            autoFocus
            required
            label="Tenant Display Name"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            fullWidth
          />
          <TextField
            label="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            fullWidth
            multiline
            rows={3}
          />
          <FormControlLabel
            control={<Switch checked={isActive} onChange={(e) => setIsActive(e.target.checked)} />}
            label="Active"
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained" disabled={!displayName.trim()}>
          {isEditMode ? 'Update' : 'Create'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default TenantDialog;