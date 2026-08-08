import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Stack,
  Box,
  Typography,
  Alert,
} from '@mui/material';
import type { BusinessObject } from '../../types/entity-schema';

interface DeleteObjectConfirmDialogProps {
  open: boolean;
  businessObject: BusinessObject | null;
  onClose: () => void;
  onConfirm: () => void;
}

export function DeleteObjectConfirmDialog({
  open,
  businessObject,
  onClose,
  onConfirm,
}: DeleteObjectConfirmDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ fontWeight: 700, color: 'error.main' }}>
        🗑️ Delete Business Object?
      </DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 2 }}>
          <Alert severity="error">
            Are you sure you want to delete this Business Object? This action cannot be undone and will delete all associated data and configuration.
          </Alert>
          <Box sx={{ bgcolor: 'action.hover', p: 2, borderRadius: 1 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
              {businessObject?.displayName}
            </Typography>
            <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
              {businessObject?.technicalName}
            </Typography>
          </Box>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" color="error" onClick={onConfirm}>
          Delete Permanently
        </Button>
      </DialogActions>
    </Dialog>
  );
}
