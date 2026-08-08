import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Stack,
  TextField,
  Typography,
  Alert,
} from '@mui/material';
import { Info as InfoIcon } from '@mui/icons-material';
import type { BusinessObject } from '../../types/entity-schema';

interface SubtypeDialogProps {
  open: boolean;
  mode: 'add' | 'edit';
  businessObject: BusinessObject | null;
  editingSubtypeKey: string | null;
  subtypeDisplayName: string;
  subtypeName: string;
  subtypeDescription: string;
  subtypeSaving: boolean;
  onClose: () => void;
  onDisplayNameChange: (value: string) => void;
  onTechnicalNameChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onSave: () => void;
}

export function SubtypeDialog({
  open,
  mode,
  businessObject,
  editingSubtypeKey,
  subtypeDisplayName,
  subtypeName,
  subtypeDescription,
  subtypeSaving,
  onClose,
  onDisplayNameChange,
  onTechnicalNameChange,
  onDescriptionChange,
  onSave,
}: SubtypeDialogProps) {
  const suggestedTechName = !subtypeName.trim() && subtypeDisplayName.trim()
    ? subtypeDisplayName.trim().toLowerCase().replace(/\s+/g, '_')
    : '';

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ fontWeight: 700, fontSize: '1.25rem' }}>
        {editingSubtypeKey ? '✏️ Edit Subtype' : '➕ Add New Subtype'}
      </DialogTitle>
      <DialogContent>
        <Stack spacing={3} sx={{ mt: 2 }}>
          <TextField
            fullWidth
            label="Display Name"
            placeholder="e.g., Commercial Customer"
            value={subtypeDisplayName}
            onChange={(e) => onDisplayNameChange(e.target.value)}
            helperText="Human-readable name for this subtype"
            variant="outlined"
            autoFocus
          />
          <TextField
            fullWidth
            label="Technical Name"
            placeholder="e.g., commercial_customer"
            value={subtypeName}
            onChange={(e) => onTechnicalNameChange(e.target.value)}
            helperText="Lowercase letters, numbers, and underscores only. Leave empty to auto-generate from display name."
            variant="outlined"
          />
          {!subtypeName.trim() && subtypeDisplayName.trim() && (
            <Typography variant="body2" color="primary" sx={{ p: 1.5, bgcolor: 'action.hover', borderRadius: 1 }}>
              <strong>Suggested technical name:</strong> <code>{suggestedTechName}</code>
            </Typography>
          )}
          <TextField
            fullWidth
            label="Description"
            placeholder="Describe what this subtype represents..."
            value={subtypeDescription}
            onChange={(e) => onDescriptionChange(e.target.value)}
            helperText="Optional. Helps other team members understand this variation"
            multiline
            rows={3}
            variant="outlined"
          />
          <Alert severity="info" icon={<InfoIcon />}>
            Subtypes inherit all core fields from {businessObject?.displayName} and can have their own additional fields.
          </Alert>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          onClick={onSave}
          disabled={subtypeSaving || !subtypeDisplayName.trim()}
        >
          {subtypeSaving ? 'Saving...' : editingSubtypeKey ? 'Update Subtype' : 'Create Subtype'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
