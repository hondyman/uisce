/**
 * ConditionRow - Single condition with dynamic inputs
 * Part of the Uisce Visual Rule Builder
 */
import React from 'react';
import {
  Box,
  FormControl,
  Select,
  MenuItem,
  TextField,
  IconButton,
  Typography,
  Autocomplete,
  Chip,
} from '@mui/material';
import { Delete as DeleteIcon } from '@mui/icons-material';

export interface FieldDefinition {
  name: string;
  type: 'number' | 'enum' | 'string' | 'date' | 'boolean';
  label: string;
  options?: string[];
}

export interface Condition {
  id: string;
  field: string;
  operator: string;
  value: any;
}

interface ConditionRowProps {
  condition: Condition;
  fields: FieldDefinition[];
  onChange: (condition: Condition) => void;
  onDelete: () => void;
  showPrefix?: boolean;
  prefixLabel?: string;
}

const OPERATORS_BY_TYPE: Record<string, { value: string; label: string }[]> = {
  number: [
    { value: '>', label: 'is greater than' },
    { value: '>=', label: 'is greater than or equal' },
    { value: '<', label: 'is less than' },
    { value: '<=', label: 'is less than or equal' },
    { value: '==', label: 'equals' },
    { value: '!=', label: 'does not equal' },
  ],
  string: [
    { value: '==', label: 'equals' },
    { value: '!=', label: 'does not equal' },
    { value: 'contains', label: 'contains' },
    { value: 'startsWith', label: 'starts with' },
    { value: 'endsWith', label: 'ends with' },
  ],
  enum: [
    { value: '==', label: 'equals' },
    { value: '!=', label: 'does not equal' },
    { value: 'IN', label: 'is one of' },
    { value: 'NOT_IN', label: 'is not one of' },
  ],
  boolean: [
    { value: '==', label: 'equals' },
  ],
  date: [
    { value: '>', label: 'is after' },
    { value: '<', label: 'is before' },
    { value: '>=', label: 'is on or after' },
    { value: '<=', label: 'is on or before' },
  ],
};

export const ConditionRow: React.FC<ConditionRowProps> = ({
  condition,
  fields,
  onChange,
  onDelete,
  showPrefix = true,
  prefixLabel = 'IF',
}) => {
  const selectedField = fields.find(f => f.name === condition.field);
  const operators = OPERATORS_BY_TYPE[selectedField?.type || 'string'] || OPERATORS_BY_TYPE.string;

  const handleFieldChange = (event: any) => {
    const newField = event.target.value;
    const field = fields.find(f => f.name === newField);
    // Reset operator and value when field changes
    const newOperator = OPERATORS_BY_TYPE[field?.type || 'string'][0]?.value || '==';
    onChange({
      ...condition,
      field: newField,
      operator: newOperator,
      value: field?.type === 'boolean' ? true : field?.type === 'number' ? 0 : '',
    });
  };

  const handleOperatorChange = (event: any) => {
    onChange({ ...condition, operator: event.target.value });
  };

  const handleValueChange = (value: any) => {
    onChange({ ...condition, value });
  };

  const renderValueInput = () => {
    if (!selectedField) return null;

    switch (selectedField.type) {
      case 'number':
        return (
          <TextField
            type="number"
            size="small"
            value={condition.value || ''}
            onChange={(e) => handleValueChange(Number(e.target.value))}
            placeholder="Enter value..."
            sx={{ minWidth: 150 }}
          />
        );

      case 'enum':
        if (condition.operator === 'IN' || condition.operator === 'NOT_IN') {
          return (
            <Autocomplete
              multiple
              size="small"
              options={selectedField.options || []}
              value={Array.isArray(condition.value) ? condition.value : []}
              onChange={(_, newValue) => handleValueChange(newValue)}
              renderTags={(value, getTagProps) =>
                value.map((option: string, index: number) => (
                  <Chip size="small" label={option} {...getTagProps({ index })} key={option} />
                ))
              }
              renderInput={(params) => (
                <TextField {...params} placeholder="Select values..." />
              )}
              sx={{ minWidth: 200 }}
            />
          );
        }
        return (
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <Select
              value={condition.value || ''}
              onChange={(e) => handleValueChange(e.target.value)}
            >
              {selectedField.options?.map((opt) => (
                <MenuItem key={opt} value={opt}>
                  {opt}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        );

      case 'boolean':
        return (
          <FormControl size="small" sx={{ minWidth: 100 }}>
            <Select
              value={String(condition.value)}
              onChange={(e) => handleValueChange(e.target.value === 'true')}
            >
              <MenuItem value="true">True</MenuItem>
              <MenuItem value="false">False</MenuItem>
            </Select>
          </FormControl>
        );

      case 'date':
        return (
          <TextField
            type="date"
            size="small"
            value={condition.value || ''}
            onChange={(e) => handleValueChange(e.target.value)}
            sx={{ minWidth: 150 }}
          />
        );

      default:
        return (
          <TextField
            size="small"
            value={condition.value || ''}
            onChange={(e) => handleValueChange(e.target.value)}
            placeholder="Enter value..."
            sx={{ minWidth: 200 }}
          />
        );
    }
  };

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1.5,
        py: 1,
        px: 2,
        bgcolor: 'action.hover',
        borderRadius: 1,
        flexWrap: 'wrap',
      }}
    >
      {showPrefix && (
        <Typography
          variant="body2"
          fontWeight="bold"
          sx={{ color: 'primary.main', minWidth: 30 }}
        >
          {prefixLabel}
        </Typography>
      )}

      {/* Field Selector */}
      <FormControl size="small" sx={{ minWidth: 180 }}>
        <Select value={condition.field} onChange={handleFieldChange}>
          {fields.map((field) => (
            <MenuItem key={field.name} value={field.name}>
              {field.label}
              <Typography
                component="span"
                variant="caption"
                sx={{ ml: 1, color: 'text.secondary' }}
              >
                ({field.type})
              </Typography>
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      {/* Operator Selector */}
      <FormControl size="small" sx={{ minWidth: 160 }}>
        <Select value={condition.operator} onChange={handleOperatorChange}>
          {operators.map((op) => (
            <MenuItem key={op.value} value={op.value}>
              {op.label}
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      {/* Value Input */}
      {renderValueInput()}

      {/* Delete Button */}
      <IconButton size="small" onClick={onDelete} color="error">
        <DeleteIcon fontSize="small" />
      </IconButton>
    </Box>
  );
};

export default ConditionRow;
