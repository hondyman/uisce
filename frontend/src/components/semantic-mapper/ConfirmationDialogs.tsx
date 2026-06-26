import { Button, Dialog, DialogActions, DialogContent, DialogTitle, Typography, Alert } from '@mui/material';
import { Check } from 'lucide-react';

interface ConfirmationDialogsProps {
  confirmOpen: boolean;
  closeConfirm: () => void;
  confirmCreate: () => void;
  selectedCount: number;
  replaceConfirmOpen: boolean;
  closeReplaceConfirm: () => void;
  confirmReplace: () => void;
}

export function ConfirmationDialogs({
  confirmOpen,
  closeConfirm,
  confirmCreate,
  selectedCount,
  replaceConfirmOpen,
  closeReplaceConfirm,
  confirmReplace,
}: ConfirmationDialogsProps) {
  return (
    <>
      <Dialog open={confirmOpen} onClose={closeConfirm} aria-labelledby="confirm-create-edges" maxWidth="sm" fullWidth>
        <DialogTitle id="confirm-create-edges">Create Semantic Edges</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to create <strong>{selectedCount}</strong> edge{selectedCount !== 1 ? 's' : ''}?
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            This will connect the selected mappings to their semantic terms in the knowledge graph.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeConfirm}>Cancel</Button>
          <Button onClick={confirmCreate} variant="contained" color="success" startIcon={<Check width={16} height={16} />}>
            Create Edges
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={replaceConfirmOpen} onClose={closeReplaceConfirm} aria-labelledby="confirm-replace-mapping" maxWidth="sm" fullWidth>
        <DialogTitle id="confirm-replace-mapping" sx={{ color: 'warning.main' }}>
          ⚠️ Replace Mapping
        </DialogTitle>
        <DialogContent>
          <Alert severity="warning" sx={{ mb: 2 }}>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>Warning: This action will cascade deletions</Typography>
          </Alert>
          <Typography>Are you sure you want to replace this mapping?</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeReplaceConfirm}>Cancel</Button>
          <Button onClick={confirmReplace} variant="contained" color="warning" startIcon={<span>🔄</span>}>
            Replace Mapping
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
