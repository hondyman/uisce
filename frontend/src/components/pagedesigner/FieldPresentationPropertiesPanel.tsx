import React from 'react';
import { Box, Typography, Stack } from '@mui/material';
import { ExpressionInputControl } from '../reporting/ExpressionInputControl';
import { DynamicFieldUIConfig } from './DynamicPropertyTypes';

interface FieldPresentationPropertiesPanelProps {
  fieldLabel: string;
  config: DynamicFieldUIConfig;
  onChange: (updated: DynamicFieldUIConfig) => void;
}

export const FieldPresentationPropertiesPanel: React.FC<FieldPresentationPropertiesPanelProps> = ({
  fieldLabel,
  config,
  onChange,
}) => {
  return (
    <Box sx={{ p: 2, bgcolor: '#071526', color: '#F8FAFC' }}>
      <Typography variant="caption" sx={{ fontWeight: 700, color: '#00D4FF', textTransform: 'uppercase' }}>
        Field UI & Formatting: {fieldLabel}
      </Typography>

      <Stack spacing={2} sx={{ mt: 2 }}>
        {/* 1. Dynamic Text Color */}
        <ExpressionInputControl<string>
          label="Text Color"
          property={config.textColor || { isExpression: false, value: '#F8FAFC' }}
          onChange={(prop) => onChange({ ...config, textColor: prop })}
          renderStaticControl={(val, setVal) => (
            <input
              type="color"
              value={val || '#F8FAFC'}
              onChange={(e) => setVal(e.target.value)}
              style={{ width: '100%', height: 28, background: '#0B1E36', border: '1px solid #1E293B' }}
            />
          )}
        />

        {/* 2. Dynamic Read-Only State */}
        <ExpressionInputControl<boolean>
          label="Read-Only State"
          property={config.isReadOnly || { isExpression: false, value: false }}
          onChange={(prop) => onChange({ ...config, isReadOnly: prop })}
          renderStaticControl={(val, setVal) => (
            <button
              type="button"
              onClick={() => setVal(!val)}
              style={{ padding: '4px 8px', background: val ? '#0284C7' : '#0B1E36', color: '#fff', fontSize: 11, border: '1px solid #1E293B', cursor: 'pointer', borderRadius: 4 }}
            >
              {val ? 'Locked (Read-Only)' : 'Editable'}
            </button>
          )}
        />

        {/* 3. Dynamic Visibility */}
        <ExpressionInputControl<boolean>
          label="Visible"
          property={config.isVisible || { isExpression: false, value: true }}
          onChange={(prop) => onChange({ ...config, isVisible: prop })}
          renderStaticControl={(val, setVal) => (
            <button
              type="button"
              onClick={() => setVal(!val)}
              style={{ padding: '4px 8px', background: val ? '#10B981' : '#EF4444', color: '#fff', fontSize: 11, border: '1px solid #1E293B', cursor: 'pointer', borderRadius: 4 }}
            >
              {val ? 'Visible' : 'Hidden'}
            </button>
          )}
        />
      </Stack>
    </Box>
  );
};
