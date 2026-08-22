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

export interface EditedFieldData {
  displayName: string;
  description: string;
  semanticTermId: string;
  role: string;
  targetScope?: string; // 'root' or subtypeKey
}

interface EditFieldDialogProps {
  open: boolean;
  semanticTerms: EnhancedSemanticTerm[];
  editedFieldData: EditedFieldData;
  subtypes?: Record<string, any>;
  businessObjectName?: string;
  onClose: () => void;
  onFieldDataChange: (data: EditedFieldData) => void;
  onSave: () => void;
}

export function EditFieldDialog({
  open,
  semanticTerms,
  editedFieldData,
  subtypes,
  businessObjectName,
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

        <TextField
          id="target-scope-select"
          select
          label="Belongs To (Scope)"
          fullWidth
          value={editedFieldData.targetScope || 'root'}
          onChange={(e) => onFieldDataChange({ ...editedFieldData, targetScope: e.target.value })}
          SelectProps={{ native: true, inputProps: { 'aria-label': 'Select field scope target', id: 'target-scope-select' } }}
          helperText="Move or assign this field to the root business object or a specific subtype"
        >
          <option value="root">Root Business Object ({businessObjectName || 'Main'})</option>
          {subtypes && Object.entries(subtypes).map(([key, st]: [string, any]) => (
            <option key={key} value={key}>Subtype: {st.displayName || st.name || key}</option>
          ))}
        </TextField>

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
