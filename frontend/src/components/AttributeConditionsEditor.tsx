import type { FC } from 'react';
import {
  Alert,
  Box,
  Button,
  Grid,
  MenuItem,
  Stack,
  TextField
} from '@mui/material';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import { AttributeCondition } from '../types/bundles';

export interface OperatorOption {
  value: string;
  label: string;
}

export interface AttributeConditionsEditorProps {
  conditions: AttributeCondition[];
  onChange: (conditions: AttributeCondition[]) => void;
  operatorOptions: OperatorOption[];
  emptyHelperText?: string;
  attributePlaceholder?: string;
  valuesPlaceholder?: string;
  basePath?: string;
  fieldErrors?: Record<string, string[]>;
  onFieldEdit?: (path: string) => void;
}

const createCondition = (seed?: Partial<AttributeCondition>): AttributeCondition => ({
  attribute: seed?.attribute ?? '',
  operator: seed?.operator ?? 'equals',
  values: [...(seed?.values ?? [])]
});

const splitList = (input: string) =>
  input
    .split(',')
    .map((value) => value.trim())
    .filter((value) => value.length > 0);

export const AttributeConditionsEditor: FC<AttributeConditionsEditorProps> = ({
  conditions,
  onChange,
  operatorOptions,
  emptyHelperText = 'No attribute conditions. Add one to scope this policy.',
  attributePlaceholder = 'e.g., roles',
  valuesPlaceholder = 'Comma separated values',
  basePath,
  fieldErrors,
  onFieldEdit
}) => {
  const handleAdd = () => {
    notifyFieldEdit('conditions');
    onChange([...(conditions || []), createCondition()]);
  };

  const handleRemove = (index: number) => {
    notifyFieldEdit('conditions');
    onChange((conditions || []).filter((_, conditionIndex) => conditionIndex !== index));
  };

  const updateCondition = (index: number, updater: (condition: AttributeCondition) => AttributeCondition) => {
    onChange((conditions || []).map((condition, conditionIndex) =>
      conditionIndex === index ? updater(condition) : condition
    ));
  };

  const getErrors = (pathSuffix: string) => {
    if (!basePath || !fieldErrors) {
      return [] as string[];
    }
    return fieldErrors[`${basePath}.${pathSuffix}`] || [];
  };

  const notifyFieldEdit = (pathSuffix: string) => {
    if (!onFieldEdit || !basePath) {
      return;
    }
    onFieldEdit(`${basePath}.${pathSuffix}`);
  };

  return (
    <Box sx={{ mt: 2 }}>
      {!conditions || conditions.length === 0 ? (
        <Alert severity="info" sx={{ mb: 1 }}>
          {emptyHelperText}
        </Alert>
      ) : (
        conditions.map((condition, index) => (
          <Box
            key={`condition-${index}`}
            sx={{
              border: '1px dashed',
              borderColor: 'divider',
              borderRadius: 2,
              p: 2,
              mb: 2
            }}
          >
            <Grid container spacing={2}>
              <Grid item xs={12} md={4}>
                <TextField
                  label="Attribute"
                  value={condition.attribute}
                  onChange={(e) => {
                    notifyFieldEdit(`conditions[${index}].attribute`);
                    updateCondition(index, (current) => ({ ...current, attribute: e.target.value }));
                  }}
                  fullWidth
                  placeholder={attributePlaceholder}
                  error={getErrors(`conditions[${index}].attribute`).length > 0}
                  helperText={getErrors(`conditions[${index}].attribute`).join(' ')}
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <TextField
                  select
                  label="Operator"
                  value={condition.operator}
                  onChange={(e) => {
                    notifyFieldEdit(`conditions[${index}].operator`);
                    updateCondition(index, (current) => ({ ...current, operator: e.target.value }));
                  }}
                  fullWidth
                  error={getErrors(`conditions[${index}].operator`).length > 0}
                  helperText={getErrors(`conditions[${index}].operator`).join(' ')}
                >
                  {operatorOptions.map((option) => (
                    <MenuItem key={option.value} value={option.value}>
                      {option.label}
                    </MenuItem>
                  ))}
                </TextField>
              </Grid>
              <Grid item xs={12} md={4}>
                <TextField
                  label="Values (comma separated)"
                  value={(condition.values || []).join(', ')}
                  onChange={(e) => {
                    notifyFieldEdit(`conditions[${index}].values`);
                    updateCondition(index, (current) => ({ ...current, values: splitList(e.target.value) }));
                  }}
                  fullWidth
                  placeholder={valuesPlaceholder}
                  error={getErrors(`conditions[${index}].values`).length > 0}
                  helperText={getErrors(`conditions[${index}].values`).join(' ')}
                />
              </Grid>
            </Grid>
            <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 1 }}>
              <Button
                size="small"
                color="error"
                startIcon={<RemoveCircleOutlineIcon />}
                onClick={() => handleRemove(index)}
              >
                Remove Condition
              </Button>
            </Box>
          </Box>
        ))
      )}

      <Stack direction="row" spacing={1}>
        <Button size="small" startIcon={<AddCircleOutlineIcon />} onClick={handleAdd}>
          Add Condition
        </Button>
      </Stack>
    </Box>
  );
};

export default AttributeConditionsEditor;
