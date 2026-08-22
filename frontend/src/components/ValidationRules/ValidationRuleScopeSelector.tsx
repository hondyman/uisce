import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  FormHelperText,
  Stack,
  Typography,
  Box,
  Alert,
} from '@mui/material';

interface Field {
  key: string;
  name: string;
  displayName?: string;
  technicalName?: string;
  type?: string;
}

interface Subtype {
  id: string;
  key: string;
  name: string;
  displayName: string;
  subtypeFields?: Field[];
  fields?: Field[];
}

export interface ValidationRuleScope {
  subtype?: string; // subtype key
}

interface ValidationRuleScopeSelectorProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (scope: ValidationRuleScope) => void;
  businessObjectName: string;
  subtypes?: Record<string, Subtype>;
}

export const ValidationRuleScopeSelector: React.FC<ValidationRuleScopeSelectorProps> = ({
  isOpen,
  onClose,
  onConfirm,
  businessObjectName,
  subtypes = {},
}) => {
  const [selectedSubtype, setSelectedSubtype] = useState<string>('');

  const handleConfirm = () => {
    const scope: ValidationRuleScope = {};
    if (selectedSubtype) {
      scope.subtype = selectedSubtype;
    }
    onConfirm(scope);
    handleClose();
  };

  const handleClose = () => {
    setSelectedSubtype('');
    onClose();
  };

  const subtypeOptions = Object.entries(subtypes).map(([key, subtype]) => ({
    key,
    label: subtype.displayName || subtype.name || key,
  }));

  return (
    <Dialog
      open={isOpen}
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
      PaperProps={{
        sx: { bgcolor: 'background.paper', backgroundImage: 'none' }
      }}
    >
      <DialogTitle sx={{ borderBottom: '1px solid rgba(255,255,255,0.1)', bgcolor: 'rgba(15,17,23,0.8)' }}>
        Create Validation Rule - Select Scope
      </DialogTitle>
      <DialogContent sx={{ bgcolor: 'background.paper', backgroundImage: 'none' }}>
        <Stack spacing={3} sx={{ mt: 2 }}>
          <Alert severity="info">
            You're creating a validation rule for <strong>{businessObjectName}</strong>. 
            You can optionally apply it to a specific subtype.
          </Alert>

          <Box>
            <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
              Select Subtype (Optional)
            </Typography>
            <FormControl fullWidth>
              <InputLabel>Subtype</InputLabel>
              <Select
                value={selectedSubtype}
                label="Subtype"
                onChange={(e) => {
                  setSelectedSubtype(e.target.value);
                }}
              >
                <MenuItem value="">
                  <em>Apply to entire {businessObjectName}</em>
                </MenuItem>
                {subtypeOptions.map((option) => (
                  <MenuItem key={option.key} value={option.key}>
                    {option.label}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>
                {selectedSubtype 
                  ? `Selected: ${subtypes[selectedSubtype]?.displayName || selectedSubtype}`
                  : 'Leave empty to apply rule to all subtypes. Fields will be selected when building conditions.'
                }
              </FormHelperText>
            </FormControl>
          </Box>

          <Box sx={{ bgcolor: 'rgba(99,102,241,0.08)', border: '1px solid rgba(99,102,241,0.2)', p: 2, borderRadius: 1 }}>
            <Typography variant="caption" sx={{ color: 'text.secondary' }}>
              <strong>Summary:</strong><br/>
              {selectedSubtype
                ? `Rule will be applied to ${subtypes[selectedSubtype]?.displayName || selectedSubtype}`
                : `Rule will be applied to the entire ${businessObjectName}`
              }
            </Typography>
          </Box>
        </Stack>
      </DialogContent>
      <DialogActions sx={{ borderTop: '1px solid rgba(255,255,255,0.1)', bgcolor: 'rgba(15,17,23,0.8)' }}>
        <Button onClick={handleClose} color="inherit">
          Cancel
        </Button>
        <Button onClick={handleConfirm} variant="contained">
          Continue to Rule Builder
        </Button>
      </DialogActions>
    </Dialog>
  );
};
