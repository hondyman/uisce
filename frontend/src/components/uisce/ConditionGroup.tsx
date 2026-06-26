/**
 * ConditionGroup - Compound logic grouping (AND/OR)
 * Part of the Uisce Visual Rule Builder
 */
import React from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  ToggleButton,
  ToggleButtonGroup,
  IconButton,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import ConditionRow, { Condition, FieldDefinition } from './ConditionRow';

export interface ConditionGroupType {
  id: string;
  logic: 'AND' | 'OR';
  conditions: Condition[];
}

interface ConditionGroupProps {
  group: ConditionGroupType;
  fields: FieldDefinition[];
  onChange: (group: ConditionGroupType) => void;
  onDelete?: () => void;
  isNested?: boolean;
}

export const ConditionGroup: React.FC<ConditionGroupProps> = ({
  group,
  fields,
  onChange,
  onDelete,
  isNested = false,
}) => {
  const handleLogicChange = (_: any, newLogic: 'AND' | 'OR') => {
    if (newLogic) {
      onChange({ ...group, logic: newLogic });
    }
  };

  const handleConditionChange = (index: number, condition: Condition) => {
    const newConditions = [...group.conditions];
    newConditions[index] = condition;
    onChange({ ...group, conditions: newConditions });
  };

  const handleDeleteCondition = (index: number) => {
    const newConditions = group.conditions.filter((_, i) => i !== index);
    onChange({ ...group, conditions: newConditions });
  };

  const handleAddCondition = () => {
    const defaultField = fields[0];
    const newCondition: Condition = {
      id: `cond-${Date.now()}`,
      field: defaultField?.name || '',
      operator: '==',
      value: defaultField?.type === 'number' ? 0 : '',
    };
    onChange({ ...group, conditions: [...group.conditions, newCondition] });
  };

  return (
    <Paper
      variant={isNested ? 'outlined' : 'elevation'}
      elevation={isNested ? 0 : 1}
      sx={{
        p: 2,
        bgcolor: isNested ? 'action.hover' : 'background.paper',
        borderLeft: isNested ? 4 : 0,
        borderColor: group.logic === 'AND' ? 'primary.main' : 'secondary.main',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Typography variant="subtitle2" color="text.secondary">
            Match
          </Typography>
          <ToggleButtonGroup
            size="small"
            value={group.logic}
            exclusive
            onChange={handleLogicChange}
          >
            <ToggleButton value="AND" sx={{ px: 2 }}>
              <Typography variant="caption" fontWeight="bold">ALL</Typography>
            </ToggleButton>
            <ToggleButton value="OR" sx={{ px: 2 }}>
              <Typography variant="caption" fontWeight="bold">ANY</Typography>
            </ToggleButton>
          </ToggleButtonGroup>
          <Typography variant="body2" color="text.secondary">
            of the following conditions
          </Typography>
        </Box>

        {onDelete && (
          <IconButton size="small" onClick={onDelete} color="error">
            <DeleteIcon fontSize="small" />
          </IconButton>
        )}
      </Box>

      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {group.conditions.map((condition, index) => (
          <ConditionRow
            key={condition.id}
            condition={condition}
            fields={fields}
            onChange={(c) => handleConditionChange(index, c)}
            onDelete={() => handleDeleteCondition(index)}
            showPrefix={true}
            prefixLabel={index === 0 ? 'IF' : group.logic}
          />
        ))}

        {group.conditions.length === 0 && (
          <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
            No conditions added yet. Click "Add Condition" to begin.
          </Typography>
        )}
      </Box>

      <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
        <Button
          size="small"
          startIcon={<AddIcon />}
          onClick={handleAddCondition}
          variant="outlined"
        >
          Add Condition
        </Button>
      </Box>
    </Paper>
  );
};

export default ConditionGroup;
