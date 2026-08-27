import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Grid,
  Card,
  CardHeader,
  CardContent,
  IconButton,
  Button,
  Chip,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
} from '@mui/material';
import {
  Add as AddIcon,
  DeleteOutline as DeleteIcon,
  Visibility as PreviewIcon,
  DesignServices as EditIcon,
} from '@mui/icons-material';
import { FormTemplateSpec, FormSection, FormFieldItem, ColSpan } from './FormManagerTypes';
import { ReportFormRenderer } from './ReportFormRenderer';
import { FormFieldPropertiesPanel } from './FormFieldPropertiesPanel';

interface ReportFormManagerProps {
  formSpec: FormTemplateSpec | null;
  onFormSpecChange: (spec: FormTemplateSpec) => void;
  availableFields: Array<{ key: string; name: string; type: string }>;
  previewData?: Record<string, any>;
  focusedSectionId?: string | null;
  onFocusSection?: (id: string | null) => void;
  onSectionDeleted?: (snapshot: FormSection) => void;
}

const defaultInitialSpec: FormTemplateSpec = {
  templateId: `form_${Date.now()}`,
  title: 'Institutional Mandate Form',
  sections: [
    {
      id: 'sec_header',
      title: 'General Identification & Metadata',
      description: 'Account classification and core portfolio terms',
      columns: 2,
      items: [],
    },
  ],
};

export const ReportFormManager: React.FC<ReportFormManagerProps> = ({
  formSpec,
  onFormSpecChange,
  availableFields = [],
  previewData = {},
  focusedSectionId: externalFocusedSectionId,
  onFocusSection,
  onSectionDeleted,
}) => {
  const currentSpec = formSpec || defaultInitialSpec;
  const [selectedItemId, setSelectedItemId] = useState<string | null>(null);
  const [isPreviewMode, setIsPreviewMode] = useState<boolean>(false);
  const [dragOverSectionId, setDragOverSectionId] = useState<string | null>(null);
  const [internalFocusedSectionId, setInternalFocusedSectionId] = useState<string | null>(null);

  const focusedSectionId = externalFocusedSectionId !== undefined ? externalFocusedSectionId : internalFocusedSectionId;
  const setFocusedSectionId = (id: string | null) => {
    setInternalFocusedSectionId(id);
    onFocusSection?.(id);
  };

  const activeItem = currentSpec.sections
    .flatMap((s) => s.items)
    .find((item) => item.id === selectedItemId);

  const handleUpdateActiveItem = (updates: Partial<FormFieldItem>) => {
    if (!selectedItemId) return;
    const updatedSections = currentSpec.sections.map((sec) => ({
      ...sec,
      items: sec.items.map((item) => (item.id === selectedItemId ? { ...item, ...updates } : item)),
    }));
    onFormSpecChange({ ...currentSpec, sections: updatedSections });
  };

  const handleAddSection = () => {
    const newSection: FormSection = {
      id: `sec_${Date.now()}`,
      title: 'New Form Section',
      description: 'Drag fields or static elements into this section',
      columns: 2,
      items: [],
    };
    onFormSpecChange({ ...currentSpec, sections: [...currentSpec.sections, newSection] });
  };

  const handleUpdateSection = (sectionId: string, patch: Partial<FormSection>) => {
    const updatedSections = currentSpec.sections.map((s) =>
      s.id === sectionId ? { ...s, ...patch } : s
    );
    onFormSpecChange({ ...currentSpec, sections: updatedSections });
  };

  const handleDeleteSection = (sectionId: string) => {
    const sectionSnapshot = currentSpec.sections.find((s) => s.id === sectionId);
    if (!sectionSnapshot) return;
    const updatedSections = currentSpec.sections.filter((s) => s.id !== sectionId);
    onFormSpecChange({ ...currentSpec, sections: updatedSections });
    if (selectedItemId) setSelectedItemId(null);
    setFocusedSectionId(null);
    onSectionDeleted?.(sectionSnapshot);
  };

  const handleDeleteItem = (itemId: string) => {
    const updatedSections = currentSpec.sections.map((sec) => ({
      ...sec,
      items: sec.items.filter((i) => i.id !== itemId),
    }));
    onFormSpecChange({ ...currentSpec, sections: updatedSections });
    if (selectedItemId === itemId) setSelectedItemId(null);
  };

  const handleDropOnSection = (e: React.DragEvent, sectionId: string) => {
    e.preventDefault();
    setDragOverSectionId(null);
    try {
      const payloadText = e.dataTransfer.getData('application/json');
      if (!payloadText) return;
      const data = JSON.parse(payloadText);

      let newItem: FormFieldItem;
      if (data.type === 'PALETTE_STATIC') {
        newItem = {
          id: `static_${Date.now()}`,
          type: data.elementType,
          label: data.label,
          valueExpression: data.defaultExpression || '',
          colSpan: 6,
          labelPlacement: 'TOP',
        };
      } else if (data.type === 'BO_FIELD') {
        newItem = {
          id: `field_${data.key}_${Date.now()}`,
          type: 'SEMANTIC_FIELD',
          label: data.name,
          fieldKey: data.key,
          valueExpression: `=Fields!${data.key}.Value`,
          colSpan: 6,
          labelPlacement: 'TOP',
        };
      } else {
        return;
      }

      const updatedSections = currentSpec.sections.map((sec) =>
        sec.id === sectionId ? { ...sec, items: [...sec.items, newItem] } : sec
      );
      onFormSpecChange({ ...currentSpec, sections: updatedSections });
      setSelectedItemId(newItem.id);
    } catch (err) {
      console.error('Failed to parse dropped element:', err);
    }
  };

  return (
    <Box sx={{ display: 'flex', height: '100%', bgcolor: '#050D1A', color: '#F8FAFC' }}>
      {/* Center Canvas */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* Toolbar */}
        <Box
          sx={{
            p: 2,
            borderBottom: '1px solid #1E293B',
            bgcolor: '#071526',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Typography variant="subtitle2" fontWeight={700}>
              Form Template Designer
            </Typography>
            <Chip
              size="small"
              label={isPreviewMode ? 'Live Preview' : 'Design Scaffolding'}
              color={isPreviewMode ? 'success' : 'primary'}
              sx={{ fontSize: 10, height: 20 }}
            />
          </Stack>

          <Stack direction="row" spacing={1}>
            <Button
              size="small"
              variant="outlined"
              startIcon={isPreviewMode ? <EditIcon /> : <PreviewIcon />}
              onClick={() => setIsPreviewMode(!isPreviewMode)}
              sx={{ color: '#00D4FF', borderColor: '#00D4FF', textTransform: 'none', fontSize: 11 }}
            >
              {isPreviewMode ? 'Switch to Design' : 'Live Data Preview'}
            </Button>
            <Button
              size="small"
              variant="contained"
              startIcon={<AddIcon />}
              onClick={handleAddSection}
              sx={{ bgcolor: '#0284C7', textTransform: 'none', fontSize: 11 }}
            >
              Add Section Card
            </Button>
          </Stack>
        </Box>

        {/* Scrollable Body */}
        <Box sx={{ flex: 1, p: 3, overflowY: 'auto', bgcolor: '#050D1A' }}>
          {isPreviewMode ? (
            <ReportFormRenderer formSpec={currentSpec} previewData={previewData} />
          ) : (
            <Stack spacing={3} sx={{ maxWidth: 1000, mx: 'auto' }}>
              {currentSpec.sections.map((sec) => (
                <Card
                  key={sec.id}
                  onClick={() => setFocusedSectionId(sec.id)}
                  onDragOver={(e) => {
                    e.preventDefault();
                    setDragOverSectionId(sec.id);
                  }}
                  onDragLeave={() => setDragOverSectionId(null)}
                  onDrop={(e) => handleDropOnSection(e, sec.id)}
                  sx={{
                    bgcolor: focusedSectionId === sec.id ? 'rgba(0, 212, 255, 0.03)' : '#071526',
                    color: '#F8FAFC',
                    border: '1.5px solid',
                    borderColor:
                      focusedSectionId === sec.id
                        ? '#00D4FF'
                        : dragOverSectionId === sec.id
                        ? '#00D4FF'
                        : '#1E293B',
                    borderRadius: 2,
                    boxShadow: 'none',
                    transition: 'border-color 0.2s ease, background-color 0.2s ease',
                  }}
                >
                  <CardHeader
                    title={
                      <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#38BDF8' }}>
                        {sec.title}
                      </Typography>
                    }
                    subheader={
                      sec.description && (
                        <Typography variant="caption" sx={{ color: '#64748B' }}>
                          {sec.description}
                        </Typography>
                      )
                    }
                    action={
                      <Stack direction="row" spacing={0.5} alignItems="center" sx={{ pr: 1 }}>
                        <FormControl size="small" sx={{ minWidth: 72 }}>
                          <Select
                            value={sec.columns}
                            onClick={(e) => e.stopPropagation()}
                            onChange={(e) => handleUpdateSection(sec.id, { columns: e.target.value as any })}
                            sx={{
                              color: '#94A3B8',
                              fontSize: 11,
                              height: 24,
                              '& .MuiSelect-select': { py: 0.5, px: 1 },
                              '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' },
                              '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: '#334155' },
                            }}
                          >
                            <MenuItem value={1}>1 col</MenuItem>
                            <MenuItem value={2}>2 col</MenuItem>
                            <MenuItem value={3}>3 col</MenuItem>
                            <MenuItem value={4}>4 col</MenuItem>
                          </Select>
                        </FormControl>
                        <IconButton
                          size="small"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteSection(sec.id);
                          }}
                          sx={{ color: '#64748B', '&:hover': { color: '#EF4444' }, p: 0.3 }}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Stack>
                    }
                    sx={{ pb: 1, borderBottom: '1px solid #1E293B' }}
                  />
                  <CardContent sx={{ p: 2 }}>
                    {sec.items.length === 0 ? (
                      <Box sx={{ p: 3, border: '1px dashed #334155', borderRadius: 1, textAlign: 'center', color: '#64748B', fontSize: 12 }}>
                        Drag form controls or semantic fields from the sidebar into this section
                      </Box>
                    ) : (
                      <Grid container spacing={2}>
                        {sec.items.map((item) => (
                          <Grid
                            item
                            xs={12}
                            sm={Math.round((item.colSpan / 12) * sec.columns)}
                            key={item.id}
                          >
                            <Box
                              onClick={() => setSelectedItemId(item.id)}
                              sx={{
                                p: 1.5,
                                bgcolor: selectedItemId === item.id ? 'rgba(0, 212, 255, 0.06)' : '#0B1E36',
                                border: '1px solid',
                                borderColor: selectedItemId === item.id ? '#00D4FF' : '#1E293B',
                                borderRadius: 1.5,
                                cursor: 'pointer',
                                '&:hover': { borderColor: '#38BDF8' },
                              }}
                            >
                              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                                <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                                  {item.label}
                                </Typography>
                                <IconButton
                                  size="small"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleDeleteItem(item.id);
                                  }}
                                  sx={{ p: 0.2, color: '#64748B', '&:hover': { color: '#EF4444' } }}
                                >
                                  <DeleteIcon sx={{ fontSize: 13 }} />
                                </IconButton>
                              </Box>
                              <Chip
                                size="small"
                                label={item.valueExpression || `[${item.label}]`}
                                sx={{ fontFamily: 'monospace', fontSize: 10, bgcolor: '#071526', color: '#38BDF8' }}
                              />
                            </Box>
                          </Grid>
                        ))}
                      </Grid>
                    )}
                  </CardContent>
                </Card>
              ))}
            </Stack>
          )}
        </Box>
      </Box>

      {/* 3. Right Property Drawer */}
      <Paper
        elevation={0}
        sx={{
          width: 300,
          borderLeft: '1px solid #1E293B',
          bgcolor: '#071526',
          p: 2,
          display: 'flex',
          flexDirection: 'column',
          gap: 2,
          overflowY: 'auto',
        }}
      >
        <Typography variant="caption" sx={{ fontWeight: 700, color: '#00D4FF', textTransform: 'uppercase', display: 'block', mb: 2 }}>
          Element Properties
        </Typography>

        {activeItem ? (
          <FormFieldPropertiesPanel
            field={activeItem}
            availableFields={availableFields.map((f) => ({
              name: f.key,
              type: f.type,
              label: f.name,
            }))}
            onChange={handleUpdateActiveItem}
          />
        ) : (
          <Typography variant="caption" sx={{ color: '#64748B' }}>
            Select an element to configure typography, format masks, and value expressions.
          </Typography>
        )}
      </Paper>
    </Box>
  );
};
