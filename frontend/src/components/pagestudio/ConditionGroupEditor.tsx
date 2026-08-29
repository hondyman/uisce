import React, { useState } from 'react';
import {
  Dialog, DialogTitle, DialogContent, DialogActions,
  Button, Box, Typography,
} from '@mui/material';
import { AdvancedConditionBuilder, type ConditionGroup, type EntityDefinition, type FieldDefinition } from '../ExpressionBuilder/AdvancedConditionBuilder';

interface ConditionGroupEditorProps {
  open: boolean;
  onClose: () => void;
  value: ConditionGroup | null;
  onChange: (group: ConditionGroup) => void;
  availableFields?: FieldDefinition[];
  entities?: EntityDefinition[];
  primaryEntity?: string;
  readOnly?: boolean;
}

export const ConditionGroupEditor: React.FC<ConditionGroupEditorProps> = ({
  open,
  onClose,
  value,
  onChange,
  availableFields = [],
  entities = [],
  primaryEntity = 'root',
  readOnly = false,
}) => {
  const [draft, setDraft] = useState<ConditionGroup | null>(value);

  const handleApply = () => {
    if (draft) {
      onChange(draft);
    }
    onClose();
  };

  const handleClear = () => {
    setDraft(null);
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="md"
      fullWidth
      PaperProps={{ sx: { bgcolor: '#030B15', maxHeight: '80vh' } }}
    >
      <DialogTitle sx={{ bgcolor: '#071526', borderBottom: '1px solid #1E293B', color: '#F8FAFC' }}>
        <Typography variant="subtitle1" fontWeight={700}>Visibility / Condition</Typography>
        <Typography variant="caption" color="text.secondary">
          Define when this element is visible or active using field-based conditions.
        </Typography>
      </DialogTitle>
      <DialogContent sx={{ bgcolor: '#030B15', p: 0, overflow: 'auto' }}>
        {draft === null ? (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', p: 4, flexDirection: 'column', gap: 2 }}>
            <Typography variant="body2" color="text.secondary">
              No condition set — element will always be visible.
            </Typography>
            <Button
              variant="outlined"
              size="small"
              onClick={handleClear}
              sx={{ color: '#94A3B8', borderColor: '#1E293B', textTransform: 'none' }}
            >
              Add Condition
            </Button>
          </Box>
        ) : (
          <Box sx={{ p: 2 }}>
            <AdvancedConditionBuilder
              value={draft}
              onChange={setDraft}
              availableFields={availableFields}
              entities={entities}
              primaryEntity={primaryEntity}
              compact
              readOnly={readOnly}
            />
          </Box>
        )}
      </DialogContent>
      <DialogActions sx={{ bgcolor: '#071526', borderTop: '1px solid #1E293B', p: 1.5 }}>
        {draft !== null && (
          <Button
            size="small"
            onClick={handleClear}
            sx={{ color: '#EF4444', textTransform: 'none', mr: 'auto' }}
          >
            Clear Condition
          </Button>
        )}
        <Button
          size="small"
          onClick={onClose}
          sx={{ color: '#94A3B8', textTransform: 'none' }}
        >
          Cancel
        </Button>
        <Button
          size="small"
          variant="contained"
          onClick={handleApply}
          sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 700 }}
        >
          Apply
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default ConditionGroupEditor;
