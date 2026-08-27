import React, { useState } from 'react';
import { DynamicProperty } from './evaluateDynamicProperty';
import CodeIcon from '@mui/icons-material/Code';
import {
  Box,
  Typography,
  Tooltip,
  Button,
  TextField,
  Chip,
} from '@mui/material';
import UnifiedExpressionBuilderModal from './UnifiedExpressionBuilderModal';

interface ExpressionInputControlProps<T> {
  label: string;
  property: DynamicProperty<T> | T | undefined;
  onChange: (updated: DynamicProperty<T>) => void;
  renderStaticControl: (value: T, onChange: (val: T) => void) => React.ReactNode;
  defaultFormula?: string;
}

export function ExpressionInputControl<T>({
  label,
  property,
  onChange,
  renderStaticControl,
  defaultFormula,
}: ExpressionInputControlProps<T>) {
  const isDynamic = typeof property === 'object' && property !== null && 'isExpression' in property;
  const propObj: DynamicProperty<T> = isDynamic
    ? (property as DynamicProperty<T>)
    : { isExpression: false, value: property as T };

  const [isModalOpen, setIsModalOpen] = useState(false);

  const handleToggleMode = () => {
    const nextIsExpression = !propObj.isExpression;
    onChange({
      ...propObj,
      isExpression: nextIsExpression,
      formula: propObj.formula || defaultFormula || `=IIF(Fields!nav.Value > 1000000, "#10B981", "#EF4444")`,
    });
  };

  const handleApplyFormula = (newFormula: string) => {
    onChange({ ...propObj, formula: newFormula, isExpression: true });
    setIsModalOpen(false);
  };

  return (
    <Box sx={{ mb: 1.5, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography variant="caption" color="text.secondary" fontWeight="600">
          {label}
        </Typography>
        <Tooltip title="Toggle Single Unified SSRS / Crystal / Multi-Tenant Expression Builder">
          <Chip
            size="small"
            icon={<CodeIcon sx={{ fontSize: 13 }} />}
            label="fx"
            onClick={handleToggleMode}
            color={propObj.isExpression ? 'primary' : 'default'}
            variant={propObj.isExpression ? 'filled' : 'outlined'}
            sx={{
              height: 20,
              fontSize: '0.65rem',
              fontWeight: 700,
              cursor: 'pointer',
              px: 0.5,
              '& .MuiChip-label': { px: 0.5 },
            }}
          />
        </Tooltip>
      </Box>

      {propObj.isExpression ? (
        <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
          <TextField
            size="small"
            fullWidth
            value={propObj.formula || ''}
            onClick={() => setIsModalOpen(true)}
            InputProps={{
              readOnly: true,
              sx: {
                fontFamily: 'monospace',
                fontSize: '0.75rem',
                color: 'primary.main',
                bgcolor: 'rgba(99, 102, 241, 0.08)',
                cursor: 'pointer',
              },
            }}
          />
          <Button
            size="small"
            variant="outlined"
            onClick={() => setIsModalOpen(true)}
            sx={{ textTransform: 'none', minWidth: 44, fontSize: '0.7rem', py: 0.5 }}
          >
            Edit
          </Button>

          {/* Single Unified SSRS / Multi-Tenant Expression Builder */}
          <UnifiedExpressionBuilderModal
            open={isModalOpen}
            onClose={() => setIsModalOpen(false)}
            title={`Expression Builder: ${label}`}
            label={label}
            initialFormula={propObj.formula || defaultFormula || ''}
            onApply={handleApplyFormula}
          />
        </Box>
      ) : (
        renderStaticControl(propObj.value, (val) => onChange({ ...propObj, value: val }))
      )}
    </Box>
  );
}

export default ExpressionInputControl;
