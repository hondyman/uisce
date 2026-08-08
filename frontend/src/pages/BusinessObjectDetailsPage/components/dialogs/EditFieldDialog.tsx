import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Autocomplete,
} from '@mui/material';
import type { EnhancedSemanticTerm } from '../../hooks/useEnhancedSemanticTerms';

interface EditedFieldData {
  displayName: string;
  description: string;
  semanticTermId: string;
  role: string;
}

interface EditFieldDialogProps {
  open: boolean;
  semanticTerms: EnhancedSemanticTerm[];
  editedFieldData: EditedFieldData;
  onClose: () => void;
  onFieldDataChange: (data: EditedFieldData) => void;
  onSave: () => void;
}

export function EditFieldDialog({
  open,
  semanticTerms,
  editedFieldData,
  onClose,
  onFieldDataChange,
  onSave,
}: EditFieldDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Edit Field</DialogTitle>
      <DialogContent sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
        <TextField
          label="Display Name"
          fullWidth
          value={editedFieldData.displayName}
          onChange={(e) => onFieldDataChange({ ...editedFieldData, displayName: e.target.value })}
          sx={{ mt: 1 }}
        />
        <TextField
          label="Description"
          fullWidth
          multiline
          rows={3}
          value={editedFieldData.description}
          onChange={(e) => onFieldDataChange({ ...editedFieldData, description: e.target.value })}
        />

        <Autocomplete
          options={semanticTerms}
          getOptionLabel={(option) => option.node_name}
          value={semanticTerms.find(t => t.id === editedFieldData.semanticTermId) || null}
          onChange={(_, newValue) => {
            onFieldDataChange({
              ...editedFieldData,
              semanticTermId: newValue?.id || ''
            });
          }}
          renderInput={(params) => <TextField {...params} label="Semantic Term" />}
        />

        <TextField
          id="role-select"
          select
          label="Role"
          fullWidth
          InputLabelProps={{ id: 'role-select-label' }}
          value={editedFieldData.role}
          onChange={(e) => onFieldDataChange({ ...editedFieldData, role: e.target.value })}
          SelectProps={{ native: true, inputProps: { 'aria-label': 'Select field role', title: 'Select the role for this field', id: 'role-select', 'aria-labelledby': 'role-select-label' } }}
        >
          <option value="">None</option>
          <option value="DIMENSION">Dimension</option>
          <option value="MEASURE">Measure</option>
          <option value="ATTRIBUTE">Attribute</option>
        </TextField>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" onClick={onSave}>Save</Button>
      </DialogActions>
    </Dialog>
  );
}
