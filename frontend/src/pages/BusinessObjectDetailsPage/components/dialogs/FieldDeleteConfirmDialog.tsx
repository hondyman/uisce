import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Stack,
  Box,
  Typography,
  Chip,
  Alert,
} from '@mui/material';
import type { Field } from '../../types/entity-schema';

interface FieldDeleteConfirmDialogProps {
  open: boolean;
  fieldPendingDelete: Field | null;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export function FieldDeleteConfirmDialog({
  open,
  fieldPendingDelete,
  isDeleting,
  onClose,
  onConfirm,
}: FieldDeleteConfirmDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ fontWeight: 700, color: 'error.main' }}>
        🗑️ Remove Field?
      </DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 2 }}>
          <Alert severity="error">
            This will permanently remove this field from the business object. This action cannot be undone.
          </Alert>
          {fieldPendingDelete && (
            <Box sx={{ bgcolor: 'action.hover', p: 2, borderRadius: 1 }}>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                Field to remove:
              </Typography>
              <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                {fieldPendingDelete.businessName || fieldPendingDelete.name}
              </Typography>
              <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                {fieldPendingDelete.technicalName || fieldPendingDelete.key}
              </Typography>
              {(fieldPendingDelete.semanticTermName || fieldPendingDelete.semantic_term_name) && (
                <>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
                    Semantic Term:
                  </Typography>
                  <Chip
                    label={fieldPendingDelete.semanticTermName || fieldPendingDelete.semantic_term_name}
                    size="small"
                    variant="outlined"
                    sx={{ mt: 0.5 }}
                  />
                </>
              )}
            </Box>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={isDeleting}>
          Cancel
        </Button>
        <Button variant="contained" color="error" onClick={onConfirm} disabled={isDeleting}>
          {isDeleting ? 'Removing...' : 'Remove Field'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
