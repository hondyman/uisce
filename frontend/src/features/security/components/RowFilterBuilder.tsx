import React, { useState } from 'react';
import {
  Box,
  Paper,
  Stack,
  Button,
  IconButton,
  Select,
  MenuItem,
  TextField,
  Typography,
  Chip,
  FormControl,
  InputLabel,
  Divider,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  DragIndicator as DragIcon,
} from '@mui/icons-material';

type Operator = '=' | '!=' | '>' | '<' | '>=' | '<=' | 'LIKE' | 'IN' | 'NOT IN';
type LogicalOperator = 'AND' | 'OR';

interface FilterCondition {
  id: string;
  field: string;
  operator: Operator;
  value: string;
}

interface FilterGroup {
  id: string;
  logicalOperator: LogicalOperator;
  conditions: FilterCondition[];
}

interface RowFilterBuilderProps {
  value: string;
  onChange: (dsl: string) => void;
  availableFields?: Array<{ name: string; type: string; displayName: string }>;
}

const operators: Operator[] = ['=', '!=', '>', '<', '>=', '<=', 'LIKE', 'IN', 'NOT IN'];

export const RowFilterBuilder: React.FC<RowFilterBuilderProps> = ({
  value,
  onChange,
  availableFields = [
    { name: 'region', type: 'string', displayName: 'Region' },
    { name: 'status', type: 'string', displayName: 'Status' },
    { name: 'amount', type: 'number', displayName: 'Amount' },
    { name: 'client_type', type: 'string', displayName: 'Client Type' },
    { name: 'created_date', type: 'date', displayName: 'Created Date' },
  ],
}) => {
  const [filterGroup, setFilterGroup] = useState<FilterGroup>({
    id: '1',
    logicalOperator: 'AND',
    conditions: [],
  });

  const addCondition = () => {
    const newCondition: FilterCondition = {
      id: Date.now().toString(),
      field: '',
      operator: '=',
      value: '',
    };
    setFilterGroup({
      ...filterGroup,
      conditions: [...filterGroup.conditions, newCondition],
    });
  };

  const removeCondition = (id: string) => {
    setFilterGroup({
      ...filterGroup,
      conditions: filterGroup.conditions.filter((c) => c.id !== id),
    });
    updateDSL(filterGroup.conditions.filter((c) => c.id !== id), filterGroup.logicalOperator);
  };

  const updateCondition = (id: string, updates: Partial<FilterCondition>) => {
    const updatedConditions = filterGroup.conditions.map((c) =>
      c.id === id ? { ...c, ...updates } : c
    );
    setFilterGroup({
      ...filterGroup,
      conditions: updatedConditions,
    });
    updateDSL(updatedConditions, filterGroup.logicalOperator);
  };

  const updateLogicalOperator = (operator: LogicalOperator) => {
    setFilterGroup({
      ...filterGroup,
      logicalOperator: operator,
    });
    updateDSL(filterGroup.conditions, operator);
  };

  const updateDSL = (conditions: FilterCondition[], logicalOp: LogicalOperator) => {
    if (conditions.length === 0) {
      onChange('');
      return;
    }

    const dslParts = conditions
      .filter((c) => c.field && c.value)
      .map((c) => {
        const needsQuotes = !['>', '<', '>=', '<='].includes(c.operator);
        const formattedValue = needsQuotes && isNaN(Number(c.value)) ? `'${c.value}'` : c.value;
        return `${c.field} ${c.operator} ${formattedValue}`;
      });

    const dsl = dslParts.join(` ${logicalOp} `);
    onChange(dsl);
  };

  const getFieldType = (fieldName: string) => {
    return availableFields.find((f) => f.name === fieldName)?.type || 'string';
  };

  return (
    <Paper elevation={1} sx={{ p: 3, bgcolor: 'grey.50' }}>
      <Stack spacing={3}>
        {/* Header */}
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
            Filter Conditions
          </Typography>
          <Stack direction="row" spacing={2} alignItems="center">
            <FormControl size="small" sx={{ minWidth: 100 }}>
              <InputLabel>Combine with</InputLabel>
              <Select
                value={filterGroup.logicalOperator}
                onChange={(e) => updateLogicalOperator(e.target.value as LogicalOperator)}
                label="Combine with"
              >
                <MenuItem value="AND">AND</MenuItem>
                <MenuItem value="OR">OR</MenuItem>
              </Select>
            </FormControl>
            <Button startIcon={<AddIcon />} onClick={addCondition} size="small" variant="outlined">
              Add Condition
            </Button>
          </Stack>
        </Stack>

        {/* Conditions */}
        {filterGroup.conditions.length === 0 ? (
          <Box sx={{ textAlign: 'center', py: 4 }}>
            <Typography variant="body2" color="text.secondary">
              No conditions defined. Click "Add Condition" to get started.
            </Typography>
          </Box>
        ) : (
          <Stack spacing={2}>
            {filterGroup.conditions.map((condition, index) => (
              <Paper key={condition.id} elevation={0} sx={{ p: 2, bgcolor: 'background.paper' }}>
                <Stack direction="row" spacing={2} alignItems="center">
                  <DragIcon sx={{ color: 'action.disabled', cursor: 'grab' }} />
                  
                  {index > 0 && (
                    <Chip
                      label={filterGroup.logicalOperator}
                      size="small"
                      color="primary"
                      variant="outlined"
                    />
                  )}

                  <FormControl size="small" sx={{ minWidth: 150 }}>
                    <InputLabel>Field</InputLabel>
                    <Select
                      value={condition.field}
                      onChange={(e) => updateCondition(condition.id, { field: e.target.value })}
                      label="Field"
                    >
                      {availableFields.map((field) => (
                        <MenuItem key={field.name} value={field.name}>
                          {field.displayName}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>

                  <FormControl size="small" sx={{ minWidth: 120 }}>
                    <InputLabel>Operator</InputLabel>
                    <Select
                      value={condition.operator}
                      onChange={(e) => updateCondition(condition.id, { operator: e.target.value as Operator })}
                      label="Operator"
                    >
                      {operators.map((op) => (
                        <MenuItem key={op} value={op}>
                          {op}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>

                  <TextField
                    size="small"
                    label="Value"
                    value={condition.value}
                    onChange={(e) => updateCondition(condition.id, { value: e.target.value })}
                    placeholder={getFieldType(condition.field) === 'number' ? '1000' : 'value'}
                    sx={{ flex: 1 }}
                  />

                  <IconButton
                    size="small"
                    color="error"
                    onClick={() => removeCondition(condition.id)}
                  >
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </Stack>
              </Paper>
            ))}
          </Stack>
        )}

        {/* Preview */}
        {value && (
          <>
            <Divider />
            <Box>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                Generated Expression:
              </Typography>
              <Paper
                elevation={0}
                sx={{
                  p: 2,
                  bgcolor: 'primary.light',
                  fontFamily: 'monospace',
                  fontSize: '0.875rem',
                  color: 'primary.dark',
                }}
              >
                {value}
              </Paper>
            </Box>
          </>
        )}
      </Stack>
    </Paper>
  );
};

export default RowFilterBuilder;
