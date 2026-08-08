import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Stack,
  TextField,
  Typography,
  Box,
  Alert,
} from '@mui/material';
import type { BusinessObject, Subtype } from '../../types/entity-schema';

interface DeleteSubtypeDialogProps {
  open: boolean;
  businessObject: BusinessObject | null;
  deletingSubtypeKey: string | null;
  deleteConfirmInput: string;
  onClose: () => void;
  onInputChange: (value: string) => void;
  onConfirm: () => void;
}

export function DeleteSubtypeDialog({
  open,
  businessObject,
  deletingSubtypeKey,
  deleteConfirmInput,
  onClose,
  onInputChange,
  onConfirm,
}: DeleteSubtypeDialogProps) {
  const subtype = deletingSubtypeKey && businessObject?.subtypes?.[deletingSubtypeKey];
  const subtypeTechName = subtype?.technicalName || deletingSubtypeKey || '';
  const isConfirmValid = deleteConfirmInput === subtypeTechName;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ fontWeight: 700, color: 'error.main' }}>
        🗑️ Delete Subtype?
      </DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 2 }}>
          <Alert severity="error">
            This will permanently delete this subtype and cannot be undone.
          </Alert>
          {deletingSubtypeKey && subtype && (
            <>
              <Box sx={{ bgcolor: 'action.hover', p: 2, borderRadius: 1 }}>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                  Deleting:
                </Typography>
                <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                  {subtype.displayName || subtype.name}
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                  {subtype.technicalName || deletingSubtypeKey}
                </Typography>
              </Box>

              <Box>
                <Typography variant="body2" sx={{ fontWeight: 600, mb: 1 }}>
                  To confirm, type the technical name:
                </Typography>
                <Typography
                  variant="body2"
                  sx={{
                    fontFamily: 'monospace',
                    bgcolor: 'background.paper',
                    p: 1,
                    borderRadius: 1,
                    border: '1px solid',
                    borderColor: 'divider',
                    mb: 2
                  }}
                >
                  {subtype.technicalName || deletingSubtypeKey}
                </Typography>
                <TextField
                  fullWidth
                  size="small"
                  placeholder="Enter technical name to confirm"
                  value={deleteConfirmInput}
                  onChange={(e) => onInputChange(e.target.value)}
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      '&.Mui-focused fieldset': {
                        borderColor: 'error.main',
                      }
                    }
                  }}
                />
              </Box>
            </>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          color="error"
          disabled={!isConfirmValid}
          onClick={onConfirm}
        >
          Delete Permanently
        </Button>
      </DialogActions>
    </Dialog>
  );
}
