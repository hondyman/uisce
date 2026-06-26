import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Typography,
  Box,
  Alert,
  Stack,
  Chip,
  Divider,
} from '@mui/material';
import {
  Warning as WarningIcon,
  Restore as RestoreIcon,
} from '@mui/icons-material';
import { EntitySnapshot } from '../../../api/auditApi';
import { format, parseISO } from 'date-fns';

interface Props {
  open: boolean;
  version: EntitySnapshot | null;
  onClose: () => void;
  onConfirm: (reason: string) => void;
}

const RestoreDialog: React.FC<Props> = ({ open, version, onClose, onConfirm }) => {
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');

  const handleConfirm = () => {
    if (!reason.trim()) {
      setError('Reason is required for compliance purposes');
      return;
    }

    if (reason.trim().length < 10) {
      setError('Please provide a more detailed reason (at least 10 characters)');
      return;
    }

    onConfirm(reason);
    setReason('');
    setError('');
  };

  const handleClose = () => {
    setReason('');
    setError('');
    onClose();
  };

  if (!version) return null;

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>
        <Stack direction="row" alignItems="center" spacing={1}>
          <RestoreIcon color="primary" />
          <Typography variant="h6">Restore Entity to Previous State</Typography>
        </Stack>
      </DialogTitle>

      <DialogContent>
        <Alert severity="warning" icon={<WarningIcon />} sx={{ mb: 3 }}>
          <Typography variant="body2">
            <strong>Important:</strong> Restoring will create a new version with the data from the
            selected historical state. The current state will be preserved in the audit history.
            This action is tracked and cannot be undone.
          </Typography>
        </Alert>

        {/* Version Info */}
        <Box sx={{ mb: 3, p: 2, bgcolor: 'background.default', borderRadius: 1 }}>
          <Typography variant="subtitle2" gutterBottom>
            Restore Point Details
          </Typography>

          <Stack spacing={1}>
            <Stack direction="row" spacing={1} alignItems="center">
              <Typography variant="body2" color="text.secondary">
                Change Type:
              </Typography>
              <Chip label={version.change_type} size="small" color="info" />
            </Stack>

            <Typography variant="body2" color="text.secondary">
              <strong>System Time:</strong> {format(parseISO(version.system_from), 'PPpp')}
            </Typography>

            <Typography variant="body2" color="text.secondary">
              <strong>Valid Time:</strong> {format(parseISO(version.valid_from), 'PPpp')}
            </Typography>

            <Typography variant="body2" color="text.secondary">
              <strong>Changed By:</strong> {version.changed_by}
            </Typography>

            {version.change_reason && (
              <Typography variant="body2" color="text.secondary">
                <strong>Original Reason:</strong> {version.change_reason}
              </Typography>
            )}

            <Typography variant="caption" color="text.disabled">
              Version ID: {version.version_id}
            </Typography>
          </Stack>
        </Box>

        <Divider sx={{ my: 2 }} />

        {/* Reason Input */}
        <TextField
          fullWidth
          multiline
          rows={4}
          label="Reason for Restoration *"
          placeholder="Explain why you are restoring this entity to this previous state..."
          value={reason}
          onChange={(e) => {
            setReason(e.target.value);
            setError('');
          }}
          error={!!error}
          helperText={
            error ||
            'This reason will be recorded in the audit trail for compliance purposes. Please be specific.'
          }
          required
        />
      </DialogContent>

      <DialogActions>
        <Button onClick={handleClose} color="inherit">
          Cancel
        </Button>
        <Button
          onClick={handleConfirm}
          variant="contained"
          color="primary"
          startIcon={<RestoreIcon />}
          disabled={!reason.trim()}
        >
          Confirm Restore
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default RestoreDialog;
