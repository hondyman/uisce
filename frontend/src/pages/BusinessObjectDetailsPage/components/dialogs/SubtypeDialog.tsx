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
interface SubtypeDialogBusinessObject {
  displayName?: string;
  driverTableName?: string;
}

interface SubtypeDialogProps {
  open: boolean;
  mode: 'add' | 'edit';
  businessObject: SubtypeDialogBusinessObject | null;
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
            Subtypes inherit the driving table and binding from {businessObject?.displayName || 'the parent'}
            {businessObject?.driverTableName ? (
              <>
                {' '}(<code>{businessObject.driverTableName}</code>)
              </>
            ) : null}
            , with an STI filter on <code>subtype_code</code> equal to the technical name. They also inherit core fields and can add their own.
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
