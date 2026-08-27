import React from 'react';
import {
  Box,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import type { FormFieldItem } from './FormManagerTypes';
import { FormatToolbar } from '../FormatToolbar';
import { CellStyleEditor } from '../tableStyling/CellStyleEditor';
import { FormatMaskInput } from '../tableStyling/FormatMaskInput';
import { ConditionalVisibilityPanel } from '../ConditionalVisibilityPanel';
import { ExpressionInputControl } from '../ExpressionInputControl';

interface FormFieldPropertiesPanelProps {
  field: FormFieldItem;
  availableFields: Array<{ name: string; type: string; label: string }>;
  onChange: (patch: Partial<FormFieldItem>) => void;
}

export const FormFieldPropertiesPanel: React.FC<FormFieldPropertiesPanelProps> = ({
  field,
  availableFields,
  onChange,
}) => {
  const fieldProps = field.style || {};

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      {/* 1. Typography & Alignment */}
      <Accordion
        defaultExpanded
        sx={{ bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', '&::before': { display: 'none' } }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />}>
          <Typography
            variant="caption"
            sx={{ fontWeight: 700, color: '#38BDF8', textTransform: 'uppercase', letterSpacing: '0.05em' }}
          >
            Typography
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ bgcolor: '#050D1A', borderTop: '1px solid #1E293B' }}>
          <FormatToolbar
            properties={fieldProps}
            onUpdate={(key, value) =>
              onChange({ style: { ...fieldProps, [key]: value } })
            }
          />
        </AccordionDetails>
      </Accordion>

      {/* 2. Colors, Borders & Padding */}
      <Accordion
        sx={{ bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', '&::before': { display: 'none' } }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />}>
          <Typography
            variant="caption"
            sx={{ fontWeight: 700, color: '#38BDF8', textTransform: 'uppercase', letterSpacing: '0.05em' }}
          >
            Colors, Borders &amp; Padding
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ bgcolor: '#050D1A', borderTop: '1px solid #1E293B' }}>
          <CellStyleEditor
            value={fieldProps}
            onChange={(style) => onChange({ style })}
          />
        </AccordionDetails>
      </Accordion>

      {/* 3. Value Format Mask */}
      <Accordion
        defaultExpanded
        sx={{ bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', '&::before': { display: 'none' } }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />}>
          <Typography
            variant="caption"
            sx={{ fontWeight: 700, color: '#00D4FF', textTransform: 'uppercase', letterSpacing: '0.05em' }}
          >
            Value Format
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ bgcolor: '#050D1A', borderTop: '1px solid #1E293B' }}>
          <FormatMaskInput
            formatType={(field.formatMask as any) || 'Auto'}
            formatMask={field.formatMask || ''}
            prefix={field.formatPrefix || ''}
            suffix={field.formatSuffix || ''}
            onChange={(p) => onChange({ ...p })}
          />
        </AccordionDetails>
      </Accordion>

      {/* 4. Value Expression */}
      <Accordion
        defaultExpanded
        sx={{ bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', '&::before': { display: 'none' } }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />}>
          <Typography
            variant="caption"
            sx={{ fontWeight: 700, color: '#A855F7', textTransform: 'uppercase', letterSpacing: '0.05em' }}
          >
            Value Expression
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ bgcolor: '#050D1A', borderTop: '1px solid #1E293B' }}>
          <ExpressionInputControl<string>
            label="Value Expression"
            property={
              typeof field.valueExpression === 'object'
                ? field.valueExpression
                : { isExpression: false, value: field.valueExpression || '' }
            }
            onChange={(updated) => onChange({ valueExpression: updated.value })}
            renderStaticControl={(val, setVal) => (
              <Typography
                variant="body2"
                sx={{ fontFamily: 'monospace', fontSize: 12, color: '#38BDF8' }}
              >
                {val || '(static value)'}
              </Typography>
            )}
          />
        </AccordionDetails>
      </Accordion>

      {/* 5. Conditional Visibility */}
      <Accordion
        sx={{ bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', '&::before': { display: 'none' } }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />}>
          <Typography
            variant="caption"
            sx={{ fontWeight: 700, color: '#F59E0B', textTransform: 'uppercase', letterSpacing: '0.05em' }}
          >
            Conditional Visibility
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ bgcolor: '#050D1A', borderTop: '1px solid #1E293B' }}>
          <ConditionalVisibilityPanel
            availableFields={availableFields as any}
            value={field.visibilityCondition}
            onChange={(cond) => onChange({ visibilityCondition: cond })}
          />
        </AccordionDetails>
      </Accordion>
    </Box>
  );
};
