import React from 'react';
import { Box, TextField } from '@mui/material';
import { DynamicFieldUIConfig } from './DynamicPropertyTypes';
import { useEvaluatedField } from './useEvaluatedField';

interface DynamicFormFieldProps {
  label: string;
  fieldKey: string;
  value: any;
  onChange: (val: any) => void;
  uiConfig?: DynamicFieldUIConfig;
  rowData?: Record<string, any>;
}

export const DynamicFormField: React.FC<DynamicFormFieldProps> = ({
  label,
  value,
  onChange,
  uiConfig = {},
  rowData = {},
}) => {
  const evaluated = useEvaluatedField(uiConfig, rowData);

  // 1. Dynamic Visibility Check
  if (!evaluated.isVisible) {
    return null;
  }

  return (
    <Box sx={{ width: '100%' }}>
      <TextField
        fullWidth
        size="small"
        label={label}
        value={value ?? ''}
        disabled={evaluated.isReadOnly}
        required={evaluated.isRequired}
        onChange={(e) => onChange(e.target.value)}
        sx={{
          bgcolor: evaluated.backgroundColor,
          input: {
            color: evaluated.textColor,
            fontWeight: evaluated.fontWeight,
            fontSize: 12,
          },
          label: { color: '#94A3B8', fontSize: 12 },
          '& .MuiOutlinedInput-notchedOutline': {
            borderColor: evaluated.textColor !== '#F8FAFC' ? evaluated.textColor : '#1E293B',
          },
        }}
      />
    </Box>
  );
};
