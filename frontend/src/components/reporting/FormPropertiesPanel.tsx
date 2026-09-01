import React from 'react';
import {
  Box,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Typography,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Button,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import type { FormReferenceElement } from './formElementModel';
import type { FormTemplateSpec } from './form/FormManagerTypes';
import { ExpressionInputControl } from './ExpressionInputControl';
import { CellStyleEditor } from './tableStyling/CellStyleEditor';
import { ConditionalVisibilityPanel } from './ConditionalVisibilityPanel';

interface FormPropertiesPanelProps {
  element: FormReferenceElement;
  formRegistry: Record<string, FormTemplateSpec>;
  availableFields: Array<{ name: string; type: string; label: string }>;
  onUpdate: (patch: Partial<FormReferenceElement>) => void;
  onNavigateToFormTab: (templateId: string) => void;
}

export const FormPropertiesPanel: React.FC<FormPropertiesPanelProps> = ({
  element,
  formRegistry,
  availableFields,
  onUpdate,
  onNavigateToFormTab,
}) => {
  const registeredForms = Object.values(formRegistry);
  const staticTemplateId = !element.templateId?.isExpression
    ? element.templateId?.value ?? ''
    : '';

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      {/* 1. Dynamic Form Template Selector & Expression Engine */}
      <Accordion
        defaultExpanded
        sx={{ bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', '&::before': { display: 'none' } }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />}>
          <Typography
            variant="caption"
            sx={{ fontWeight: 700, color: '#00D4FF', textTransform: 'uppercase', letterSpacing: '0.05em' }}
          >
            Form Source Binding
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          <ExpressionInputControl<string>
            label="Target Form Template"
            property={element.templateId ?? { isExpression: false, value: '' }}
            onChange={(updated) => onUpdate({ templateId: updated })}
            renderStaticControl={(val, setVal) => (
              <FormControl size="small" fullWidth>
                <InputLabel sx={{ color: '#94A3B8' }}>Select Template</InputLabel>
                <Select
                  value={val || ''}
                  label="Select Template"
                  onChange={(e) => setVal(e.target.value)}
                  sx={{ color: '#F8FAFC', fontSize: 12, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' } }}
                >
                  {registeredForms.length === 0 && (
                    <MenuItem value="" disabled>
                      <em style={{ color: '#64748B' }}>No forms defined yet</em>
                    </MenuItem>
                  )}
                  {registeredForms.map((f) => (
                    <MenuItem key={f.templateId} value={f.templateId}>
                      {f.title}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            )}
          />

          {!element.templateId?.isExpression && staticTemplateId && (
            <Button
              size="small"
              variant="outlined"
              onClick={() => onNavigateToFormTab(staticTemplateId)}
              sx={{
                color: '#38BDF8',
                borderColor: '#1E293B',
                textTransform: 'none',
                fontSize: 11,
                '&:hover': { borderColor: '#00D4FF', bgcolor: 'rgba(56, 189, 248, 0.05)' },
              }}
            >
              Edit Form in Manager
            </Button>
          )}
        </AccordionDetails>
      </Accordion>

      {/* 2. Outer Container Typography & Borders */}
      <Accordion
        defaultExpanded
        sx={{ bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', '&::before': { display: 'none' } }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />}>
          <Typography
            variant="caption"
            sx={{ fontWeight: 700, color: '#38BDF8', textTransform: 'uppercase', letterSpacing: '0.05em' }}
          >
            Container Styling & Padding
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ bgcolor: '#050D1A', borderTop: '1px solid #1E293B' }}>
          <CellStyleEditor
            value={element.containerStyle || {}}
            onChange={(style) => onUpdate({ containerStyle: style })}
          />
        </AccordionDetails>
      </Accordion>

      {/* 3. Conditional Visibility Engine */}
      <Accordion
        sx={{ bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', '&::before': { display: 'none' } }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />}>
          <Typography
            variant="caption"
            sx={{ fontWeight: 700, color: '#A855F7', textTransform: 'uppercase', letterSpacing: '0.05em' }}
          >
            Conditional Visibility
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ bgcolor: '#050D1A', borderTop: '1px solid #1E293B' }}>
          <ConditionalVisibilityPanel
            availableFields={availableFields as any}
            value={element.visibilityCondition}
            onChange={(cond) => onUpdate({ visibilityCondition: cond })}
          />
        </AccordionDetails>
      </Accordion>
    </Box>
  );
};
