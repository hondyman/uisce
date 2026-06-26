// React import not required with the new JSX transform
import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Alert,
  Box,
  Chip
} from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import { Warning } from '@mui/icons-material';
import { IPWhitelistEntry, Tenant } from '../types/ipWhitelist';

interface IPDeleteDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => Promise<void>;
  entry: IPWhitelistEntry | null;
  tenants: Tenant[];
}

const IPDeleteDialog: React.FC<IPDeleteDialogProps> = ({
  open,
  onClose,
  onConfirm,
  entry,
  tenants
}) => {
  const [loading, setLoading] = useState(false);

  const handleConfirm = async () => {
    setLoading(true);
    try {
      await onConfirm();
      onClose();
    } catch (err) {
      // Error handling is done by parent component
    } finally {
      setLoading(false);
    }
  };

  if (!entry) return null;

  const assignedTenants = entry.tenantIds.map(id => 
    tenants.find(t => t.id === id)?.displayName || id
  );

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <ModalHeader
        title={(
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Warning color="warning" />
            <span>Delete IP Address</span>
          </Box>
        )}
        onClose={onClose}
      />
      
      <DialogContent>
        <Alert severity="warning" sx={{ mb: 2 }}>
          This action cannot be undone. The IP address will be removed from all assigned tenants.
        </Alert>

        <Box sx={{ mb: 2 }}>
          <Typography variant="h6" gutterBottom>
            {entry.ipAddress}
          </Typography>
          {entry.label && (
            <Typography variant="body2" color="text.secondary" gutterBottom>
              Label: {entry.label}
            </Typography>
          )}
          {entry.description && (
            <Typography variant="body2" color="text.secondary" gutterBottom>
              Description: {entry.description}
            </Typography>
          )}
        </Box>

        <Box>
          <Typography variant="subtitle2" gutterBottom>
            Currently assigned to:
          </Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
            {assignedTenants.length > 0 ? (
              assignedTenants.map((tenantName, index) => (
                <Chip
                  key={index}
                  label={tenantName}
                  size="small"
                  color="primary"
                  variant="outlined"
                />
              ))
            ) : (
              <Typography variant="body2" color="text.secondary">
                No tenants assigned
              </Typography>
            )}
          </Box>
        </Box>
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose} disabled={loading}>
          Cancel
        </Button>
        <Button 
          onClick={handleConfirm} 
          color="error" 
          variant="contained"
          disabled={loading}
        >
          {loading ? 'Deleting...' : 'Delete'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default IPDeleteDialog;
