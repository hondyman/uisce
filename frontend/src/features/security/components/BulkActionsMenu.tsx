import React, { useState } from 'react';
import {
  Menu,
  MenuItem,
  ListItemIcon,
  ListItemText,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Alert,
} from '@mui/material';
import {
  Delete as DeleteIcon,
  Edit as EditIcon,
  GetApp as ExportIcon,
  CheckCircle as ApproveIcon,
} from '@mui/icons-material';
import { AccessRule, accessRulesApi } from '../../../api/accessRules';

interface BulkActionsMenuProps {
  anchorEl: HTMLElement | null;
  open: boolean;
  onClose: () => void;
  selectedRules: AccessRule[];
  onComplete: () => void;
}

export const BulkActionsMenu: React.FC<BulkActionsMenuProps> = ({
  anchorEl,
  open,
  onClose,
  selectedRules,
  onComplete,
}) => {
  const [confirmDialog, setConfirmDialog] = useState<'delete' | 'approve' | null>(null);
  const [processing, setProcessing] = useState(false);

  const handleExport = () => {
    const dataStr = JSON.stringify(selectedRules, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `access-rules-${new Date().toISOString().split('T')[0]}.json`;
    link.click();
    URL.revokeObjectURL(url);
    onClose();
  };

  const handleBulkDelete = async () => {
    setProcessing(true);
    try {
      await Promise.all(
        selectedRules.map((rule) => accessRulesApi.delete(rule.ruleId))
      );
      setConfirmDialog(null);
      onComplete();
      onClose();
    } catch (error) {
      console.error('Bulk delete failed:', error);
    } finally {
      setProcessing(false);
    }
  };

  const handleBulkApprove = async () => {
    setProcessing(true);
    try {
      await Promise.all(
        selectedRules.map((rule) =>
          accessRulesApi.update(rule.ruleId, { ...rule, status: 'APPROVED' })
        )
      );
      setConfirmDialog(null);
      onComplete();
      onClose();
    } catch (error) {
      console.error('Bulk approve failed:', error);
    } finally {
      setProcessing(false);
    }
  };

  return (
    <>
      <Menu anchorEl={anchorEl} open={open} onClose={onClose}>
        <MenuItem onClick={handleExport}>
          <ListItemIcon>
            <ExportIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>Export Selected ({selectedRules.length})</ListItemText>
        </MenuItem>
        <MenuItem onClick={() => setConfirmDialog('approve')}>
          <ListItemIcon>
            <ApproveIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>Approve Selected</ListItemText>
        </MenuItem>
        <MenuItem onClick={() => setConfirmDialog('delete')}>
          <ListItemIcon>
            <DeleteIcon fontSize="small" color="error" />
          </ListItemIcon>
          <ListItemText>Delete Selected</ListItemText>
        </MenuItem>
      </Menu>

      {/* Delete Confirmation */}
      <Dialog open={confirmDialog === 'delete'} onClose={() => setConfirmDialog(null)}>
        <DialogTitle>Confirm Bulk Delete</DialogTitle>
        <DialogContent>
          <Alert severity="warning" sx={{ mb: 2 }}>
            This action cannot be undone.
          </Alert>
          <Typography>
            Are you sure you want to delete {selectedRules.length} rule(s)?
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmDialog(null)}>Cancel</Button>
          <Button
            variant="contained"
            color="error"
            onClick={handleBulkDelete}
            disabled={processing}
          >
            {processing ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Approve Confirmation */}
      <Dialog open={confirmDialog === 'approve'} onClose={() => setConfirmDialog(null)}>
        <DialogTitle>Confirm Bulk Approve</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to approve {selectedRules.length} rule(s)?
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmDialog(null)}>Cancel</Button>
          <Button
            variant="contained"
            color="success"
            onClick={handleBulkApprove}
            disabled={processing}
          >
            {processing ? 'Approving...' : 'Approve'}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

export default BulkActionsMenu;
