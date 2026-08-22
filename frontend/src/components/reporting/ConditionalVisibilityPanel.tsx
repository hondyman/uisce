import React from 'react';
import { Box, Typography, Switch, FormControlLabel } from '@mui/material';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import VisibilityIcon from '@mui/icons-material/Visibility';
import AdvancedConditionBuilder, {
  ConditionGroup,
  FieldDefinition,
  evaluateCondition
} from '../ExpressionBuilder/AdvancedConditionBuilder';

interface ConditionalVisibilityPanelProps {
  availableFields?: FieldDefinition[];
  value?: ConditionGroup | null;
  onChange: (tree: ConditionGroup | null) => void;
  entityName?: string;
  testData?: Record<string, any>;
}

export const ConditionalVisibilityPanel: React.FC<ConditionalVisibilityPanelProps> = ({
  availableFields = [],
  value,
  onChange,
  entityName = 'Entity',
  testData = {},
}) => {
  const isEnabled = Boolean(value);

  const handleToggle = (checked: boolean) => {
    if (checked) {
      onChange({
        id: `vis_grp_${Date.now()}`,
        type: 'group',
        operator: 'AND',
        conditions: [],
      });
    } else {
      onChange(null);
    }
  };

  const handleTreeChange = (newTree: ConditionGroup) => {
    onChange(newTree);
  };

  const isCurrentlyHidden = isEnabled && value && Object.keys(testData).length > 0 
    ? evaluateCondition(value, testData)
    : false;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <FormControlLabel
          control={
            <Switch
              size="small"
              checked={isEnabled}
              onChange={(e) => handleToggle(e.target.checked)}
            />
          }
          label={
            <Typography variant="body2" fontWeight="600">
              Conditional Hide Expression
            </Typography>
          }
        />
        {isEnabled && (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
            {isCurrentlyHidden ? (
              <VisibilityOffIcon sx={{ fontSize: 16, color: 'warning.main' }} />
            ) : (
              <VisibilityIcon sx={{ fontSize: 16, color: 'success.main' }} />
            )}
            <Typography variant="caption" color={isCurrentlyHidden ? 'warning.main' : 'text.secondary'}>
              {isCurrentlyHidden ? 'Hidden by condition' : 'Visible'}
            </Typography>
          </Box>
        )}
      </Box>

      {isEnabled && value && (
        <Box sx={{ border: '1px solid rgba(255, 255, 255, 0.08)', borderRadius: 1.5, p: 1, bgcolor: 'rgba(0,0,0,0.15)' }}>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
            Hide element/section when this condition evaluates to <strong>TRUE</strong>:
          </Typography>
          <AdvancedConditionBuilder
            value={value}
            onChange={handleTreeChange}
            availableFields={availableFields}
            entityName={entityName}
          />
        </Box>
      )}
    </Box>
  );
};

export default ConditionalVisibilityPanel;
