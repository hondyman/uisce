import React from 'react';
import {
  TextField,
  FormControlLabel,
  Checkbox,
  Box,
  Typography,
  Alert,
} from '@mui/material';

export interface PropertyMetadata {
  name: string;
  label: string;
  order: number;
  nullable?: boolean;
  data_type: string;
  input_type: string;
  description?: string;
  placeholder?: string;
  options?: { label: string; value: any }[];
}

export interface DynamicPropertyFormProps {
  properties: PropertyMetadata[];
  values: Record<string, any>;
  onChange: (field: string, value: any) => void;
  errors?: Record<string, string>;
  disabled?: boolean;
}

const DynamicPropertyForm: React.FC<DynamicPropertyFormProps> = ({
  properties,
  values,
  onChange,
  errors = {},
  disabled = false,
}) => {
  // Sort properties by order
  const sortedProperties = [...properties].sort((a, b) => (a.order ?? 999) - (b.order ?? 999));

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      {sortedProperties.length === 0 && (
         <Typography variant="body2" color="text.secondary">No properties defined.</Typography>
      )}

      {sortedProperties.map((prop) => {
        const error = errors[prop.name];

        // START: Special handling for SQL, Description, etc based on name or input_type
        
        // Checkbox for booleans
        if (prop.input_type === 'checkbox' || prop.data_type === 'boolean') {
          return (
            <FormControlLabel
              key={prop.name}
              control={
                <Checkbox
                  checked={!!values[prop.name]}
                  onChange={(e) => onChange(prop.name, e.target.checked)}
                  disabled={disabled}
                />
              }
              label={
                <Box>
                    <Typography>{prop.label || prop.name}</Typography>
                    {prop.description && (
                         <Typography variant="caption" color="text.secondary">{prop.description}</Typography>
                    )}
                </Box>
              }
            />
          );
        }

        // Text Area for SQL or lengthy text
        const isMultiline = 
            prop.input_type === 'textarea' || 
            prop.name === 'sql' || 
            prop.name === 'description' ||
            prop.name === 'meta' ||
            prop.name === 'filters'; // heuristic

        if (prop.input_type === 'select' && prop.options) {
          return (
            <TextField
              key={prop.name}
              select
              label={prop.label || prop.name}
              value={values[prop.name] ?? ''}
              onChange={(e) => onChange(prop.name, e.target.value)}
              disabled={disabled}
              fullWidth
              required={!prop.nullable}
              error={!!error}
              helperText={error || prop.description}
              SelectProps={{ native: true }}
            >
              <option value="" disabled>Select {prop.label}</option>
              {prop.options.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </TextField>
          );
        }

        return (
          <TextField
            key={prop.name}
            label={prop.label || prop.name}
            value={values[prop.name] ?? ''}
            onChange={(e) => onChange(prop.name, e.target.value)}
            disabled={disabled}
            fullWidth
            required={!prop.nullable}
            error={!!error}
            helperText={error || prop.description}
            placeholder={prop.placeholder}
            type={prop.data_type === 'number' || prop.input_type === 'number' ? 'number' : 'text'}
            multiline={isMultiline}
            rows={isMultiline ? (prop.name === 'sql' ? 4 : 2) : 1}
            InputProps={prop.name === 'sql' ? {
                sx: { fontFamily: 'monospace' }
            } : undefined}
          />
        );
      })}
    </Box>
  );
};

export default DynamicPropertyForm;
