import React, { useState } from 'react';
import {
  Box,
  Typography,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  List,
  ListItemButton,
  ListItemText,
  Paper,
} from '@mui/material';
import DynamicFormIcon from '@mui/icons-material/DynamicForm';
import type { FormTemplateSpec } from './form/FormManagerTypes';

interface ToolboxFormBlockProps {
  formRegistry: Record<string, FormTemplateSpec>;
  onAddItem: (type: string, payload?: Record<string, unknown>) => void;
}

export const ToolboxFormBlock: React.FC<ToolboxFormBlockProps> = ({
  formRegistry,
  onAddItem,
}) => {
  const [pickerOpen, setPickerOpen] = useState(false);
  const registeredForms = Object.values(formRegistry);
  const isEnabled = registeredForms.length > 0;

  const handleDragStart = (e: React.DragEvent) => {
    if (!isEnabled) {
      e.preventDefault();
      return;
    }
    if (registeredForms.length === 1) {
      e.dataTransfer.setData(
        'application/json',
        JSON.stringify({
          type: 'form-block',
          mode: 'reference',
          templateId: registeredForms[0].templateId,
        })
      );
    } else {
      e.dataTransfer.setData(
        'application/json',
        JSON.stringify({
          type: 'form-block-multi',
          availableTemplateIds: registeredForms.map((f) => f.templateId),
        })
      );
    }
  };

  const handleClick = () => {
    if (!isEnabled) return;
    if (registeredForms.length === 1) {
      onAddItem('formReference', { templateId: registeredForms[0].templateId });
    } else {
      setPickerOpen(true);
    }
  };

  return (
    <>
      <Tooltip
        title={
          isEnabled
            ? 'Drag Form Block into any section'
            : 'Define at least one Form in the Form Manager to unlock'
        }
      >
        <Paper
          elevation={0}
          onDragStart={handleDragStart}
          onClick={handleClick}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 1.5,
            p: 1.2,
            borderRadius: 1,
            bgcolor: isEnabled ? '#0B1E36' : '#040B14',
            border: '1px solid',
            borderColor: isEnabled ? '#1E293B' : '#0F172A',
            color: isEnabled ? '#F8FAFC' : '#475569',
            cursor: isEnabled ? 'grab' : 'not-allowed',
            opacity: isEnabled ? 1 : 0.45,
            transition: 'all 0.15s ease',
            '&:hover': isEnabled
              ? { borderColor: '#00D4FF', bgcolor: '#0E2442' }
              : {},
          }}
        >
          <DynamicFormIcon sx={{ fontSize: 18, color: isEnabled ? '#00D4FF' : '#475569' }} />
          <Typography variant="body2" sx={{ fontSize: 12, fontWeight: 600 }}>
            Form Block
          </Typography>
        </Paper>
      </Tooltip>

      {/* Disambiguation Modal for Multiple Forms */}
      <Dialog
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        PaperProps={{
          sx: { bgcolor: '#071526', border: '1px solid #1E293B', minWidth: 320 },
        }}
      >
        <DialogTitle
          sx={{
            bgcolor: '#071526',
            color: '#F8FAFC',
            fontSize: 14,
            fontWeight: 700,
            borderBottom: '1px solid #1E293B',
            py: 1.5,
          }}
        >
          Select Form Template to Embed
        </DialogTitle>
        <DialogContent sx={{ bgcolor: '#071526', p: 0 }}>
          <List dense>
            {registeredForms.map((form) => (
              <ListItemButton
                key={form.templateId}
                onClick={() => {
                  onAddItem('formReference', { templateId: form.templateId });
                  setPickerOpen(false);
                }}
                sx={{
                  borderBottom: '1px solid #1E293B',
                  '&:hover': { bgcolor: '#0B1E36' },
                }}
              >
                <ListItemText
                  primary={form.title}
                  secondary={`${form.sections.length} Section${form.sections.length !== 1 ? 's' : ''}`}
                  primaryTypographyProps={{ color: '#F8FAFC', fontSize: 13, fontWeight: 600 }}
                  secondaryTypographyProps={{ color: '#64748B', fontSize: 11 }}
                />
              </ListItemButton>
            ))}
          </List>
        </DialogContent>
      </Dialog>
    </>
  );
};
