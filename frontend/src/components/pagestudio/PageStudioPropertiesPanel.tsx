import React, { useState } from 'react';
import {
  Box, Typography, Stack, TextField, Select, MenuItem, FormControl,
  InputLabel, Chip, IconButton, Divider, Accordion, AccordionSummary, AccordionDetails,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import AddIcon from '@mui/icons-material/Add';
import { FieldPresentationPropertiesPanel } from '../pagedesigner/FieldPresentationPropertiesPanel';
import { ConditionGroupEditor } from './ConditionGroupEditor';
import { ContainerStyleEditor, type ContainerStyle } from './ContainerStyleEditor';
import type { CanvasWidget, FieldWidget, ContainerWidget, RelatedObjectWidget, PageParameter } from './pageStudioTypes';
import type { DynamicFieldUIConfig } from '../pagedesigner/DynamicPropertyTypes';

interface PageStudioPropertiesPanelProps {
  widget: CanvasWidget | null;
  declaredParameters: PageParameter[];
  onUpdate: (widget: CanvasWidget) => void;
  onDeselect?: () => void;
}

const widgetTypeLabel: Record<string, string> = {
  field: 'Form Field',
  section: 'Section',
  row: 'Row',
  grid: 'Data Grid',
  chart: 'Analytics Chart',
  kpi: 'KPI Tile',
  relatedObject: 'Related Object',
};

export const PageStudioPropertiesPanel: React.FC<PageStudioPropertiesPanelProps> = ({
  widget, declaredParameters, onUpdate, onDeselect: _onDeselect,
}) => {
  if (!widget) {
    return (
      <Box sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
        <Box sx={{ textAlign: 'center', p: 3, bgcolor: 'rgba(0,212,255,0.04)', borderRadius: 2, border: '1px dashed #1E293B' }}>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Select a widget on the canvas to edit its properties.
          </Typography>
          <Typography variant="caption" color="text.secondary">
            Click any field, widget, or container to see its settings here.
          </Typography>
        </Box>
        <Box sx={{ mt: 3, width: '100%' }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1 }}>
            Page Parameters
          </Typography>
          <Typography variant="caption" display="block" sx={{ color: '#64748B', mt: 0.5 }}>
            {declaredParameters.length === 0
              ? 'No parameters defined. Use the "Parameters" toolbar button to add page-level parameters.'
              : `${declaredParameters.length} parameter(s) defined.`}
          </Typography>
        </Box>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 2, overflowY: 'auto', height: '100%' }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 2 }}>
        <Typography variant="caption" sx={{ fontWeight: 700, color: '#00D4FF', textTransform: 'uppercase', letterSpacing: 1 }}>
          Properties
        </Typography>
        <Chip
          label={widgetTypeLabel[widget.type] || widget.type}
          size="small"
          sx={{ bgcolor: 'rgba(0,212,255,0.1)', color: '#00D4FF', fontSize: 10, height: 20 }}
        />
      </Stack>

      {widget.type === 'field' && (
        <FieldWidgetPanel widget={widget as FieldWidget} onUpdate={onUpdate} />
      )}

      {(widget.type === 'grid' || widget.type === 'chart' || widget.type === 'kpi') && (
        <ContainerWidgetPanel widget={widget as ContainerWidget} declaredParameters={declaredParameters} onUpdate={onUpdate} />
      )}

      {(widget.type === 'section' || widget.type === 'row') && (
        <SectionWidgetPanel widget={widget as ContainerWidget} onUpdate={onUpdate} />
      )}

      {widget.type === 'relatedObject' && (
        <RelatedObjectWidgetPanel widget={widget as RelatedObjectWidget} onUpdate={onUpdate} />
      )}
    </Box>
  );
};

const FieldWidgetPanel: React.FC<{ widget: FieldWidget; onUpdate: (w: CanvasWidget) => void }> = ({ widget, onUpdate }) => {
  const [config, setConfig] = useState<DynamicFieldUIConfig>(widget.presentation || {
    textColor: { isExpression: false, value: '#F8FAFC' },
    isVisible: { isExpression: false, value: true },
    isReadOnly: { isExpression: false, value: false },
  });

  const handleChange = (updated: DynamicFieldUIConfig) => {
    setConfig(updated);
    onUpdate({ ...widget, presentation: updated });
  };

  return (
    <Stack spacing={2}>
      <TextField
        label="Field Label"
        value={widget.label}
        size="small"
        fullWidth
        onChange={(e) => onUpdate({ ...widget, label: e.target.value })}
        inputProps={{ style: { color: '#F8FAFC', fontSize: 13 } }}
        sx={{
          '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' },
          '& .MuiInputLabel-root': { color: '#94A3B8', fontSize: 12 },
        }}
      />
      <FormControl fullWidth size="small">
        <InputLabel sx={{ color: '#94A3B8', fontSize: 12 }}>Control Type</InputLabel>
        <Select
          value={widget.controlType}
          label="Control Type"
          onChange={(e) => onUpdate({ ...widget, controlType: e.target.value as FieldWidget['controlType'] })}
          sx={{ color: '#F8FAFC', fontSize: 13, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' } }}
        >
          <MenuItem value="text">Text</MenuItem>
          <MenuItem value="number">Number</MenuItem>
          <MenuItem value="switch">Switch / Toggle</MenuItem>
          <MenuItem value="date">Date</MenuItem>
          <MenuItem value="datetime">Date & Time</MenuItem>
          <MenuItem value="select">Dropdown</MenuItem>
        </Select>
      </FormControl>
      <Divider sx={{ borderColor: '#1E293B' }} />
      <FieldPresentationPropertiesPanel
        fieldLabel={widget.label}
        config={config}
        onChange={handleChange}
      />
    </Stack>
  );
};

const ContainerWidgetPanel: React.FC<{
  widget: ContainerWidget;
  declaredParameters: PageParameter[];
  onUpdate: (w: CanvasWidget) => void;
}> = ({ widget, declaredParameters, onUpdate }) => {
  const [newParam, setNewParam] = useState('');

  const subscribedParams: string[] = widget.subscribedParams || [];

  const addParam = (paramKey: string) => {
    if (paramKey && !subscribedParams.includes(paramKey)) {
      onUpdate({ ...widget, subscribedParams: [...subscribedParams, paramKey] });
    }
    setNewParam('');
  };

  const removeParam = (paramKey: string) => {
    onUpdate({ ...widget, subscribedParams: subscribedParams.filter((p) => p !== paramKey) });
  };

  return (
    <Stack spacing={2}>
      <TextField
        label="Widget Title"
        value={widget.title || ''}
        size="small"
        fullWidth
        onChange={(e) => onUpdate({ ...widget, title: e.target.value })}
        inputProps={{ style: { color: '#F8FAFC', fontSize: 13 } }}
        sx={{
          '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' },
          '& .MuiInputLabel-root': { color: '#94A3B8', fontSize: 12 },
        }}
      />
      <TextField
        label="Bound BO Key"
        value={widget.boKey || ''}
        size="small"
        fullWidth
        placeholder="e.g. account, position"
        onChange={(e) => onUpdate({ ...widget, boKey: e.target.value })}
        inputProps={{ style: { color: '#F8FAFC', fontSize: 13 } }}
        sx={{
          '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' },
          '& .MuiInputLabel-root': { color: '#94A3B8', fontSize: 12 },
        }}
      />
      <FormControl fullWidth size="small">
        <InputLabel sx={{ color: '#94A3B8', fontSize: 12 }}>Subtype</InputLabel>
        <Select
          value={widget.subtypeKey || ''}
          label="Subtype"
          onChange={(e) => onUpdate({ ...widget, subtypeKey: e.target.value || null })}
          sx={{ color: '#F8FAFC', fontSize: 13, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' } }}
        >
          <MenuItem value=""><em>All subtypes</em></MenuItem>
        </Select>
      </FormControl>

      <Divider sx={{ borderColor: '#1E293B' }} />

      <Box>
        <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1, display: 'block', mb: 1 }}>
          Subscribed Parameters
        </Typography>
        <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 1 }}>
          This widget reacts when these page parameters change.
        </Typography>
        <Stack direction="row" gap={0.5} flexWrap="wrap" sx={{ mb: 1 }}>
          {subscribedParams.map((p) => (
            <Chip
              key={p}
              label={p}
              size="small"
              onDelete={() => removeParam(p)}
              sx={{ bgcolor: 'rgba(0,212,255,0.1)', color: '#00D4FF', fontSize: 10 }}
            />
          ))}
          {subscribedParams.length === 0 && (
            <Typography variant="caption" color="text.secondary">None — widget uses static binding.</Typography>
          )}
        </Stack>
        <Stack direction="row" gap={0.5}>
          <Select
            value={newParam}
            onChange={(e) => setNewParam(e.target.value)}
            size="small"
            displayEmpty
            sx={{ flex: 1, color: '#F8FAFC', fontSize: 12, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' } }}
          >
            <MenuItem value="" disabled><em>Add parameter...</em></MenuItem>
            {declaredParameters
              .filter((p) => !subscribedParams.includes(p.key))
              .map((p) => (
                <MenuItem key={p.key} value={p.key}>{p.displayName} ({p.key})</MenuItem>
              ))}
          </Select>
          <IconButton
            size="small"
            onClick={() => addParam(newParam)}
            disabled={!newParam}
            sx={{ color: '#00D4FF' }}
          >
            <AddIcon fontSize="small" />
          </IconButton>
        </Stack>
      </Box>
    </Stack>
  );
};

const SectionWidgetPanel: React.FC<{ widget: ContainerWidget; onUpdate: (w: CanvasWidget) => void }> = ({ widget, onUpdate }) => {
  const [conditionDialogOpen, setConditionDialogOpen] = useState(false);
  const containerStyle: ContainerStyle = widget.containerStyle || {};

  return (
    <Stack spacing={0}>
      <Accordion defaultExpanded disableGutters sx={{ bgcolor: 'transparent', boxShadow: 'none' }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />} sx={{ minHeight: 36, py: 0 }}>
          <Typography variant="caption" sx={{ fontWeight: 700, color: '#94A3B8', textTransform: 'uppercase', letterSpacing: 0.5, fontSize: '0.65rem' }}>
            General
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0, pb: 1.5 }}>
          <Stack spacing={1.5} sx={{ px: 0 }}>
            <TextField
              label="Container Title"
              value={widget.title || ''}
              size="small"
              fullWidth
              onChange={(e) => onUpdate({ ...widget, title: e.target.value })}
              inputProps={{ style: { color: '#F8FAFC', fontSize: 13 } }}
              sx={{ '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' }, '& .MuiInputLabel-root': { color: '#94A3B8', fontSize: 12 } }}
            />
            <FormControl fullWidth size="small">
              <InputLabel sx={{ color: '#94A3B8', fontSize: 12 }}>Layout Flow</InputLabel>
              <Select
                value={widget.flow || (widget.type === 'row' ? 'row' : 'column')}
                label="Layout Flow"
                onChange={(e) => onUpdate({ ...widget, flow: e.target.value as 'column' | 'row' })}
                sx={{ color: '#F8FAFC', fontSize: 13, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' } }}
              >
                <MenuItem value="column">Column (stacked)</MenuItem>
                <MenuItem value="row">Row (side-by-side)</MenuItem>
              </Select>
            </FormControl>
            <TextField
              label="Gap (px)"
              type="number"
              value={(widget as any).gap ?? 8}
              size="small"
              fullWidth
              onChange={(e) => onUpdate({ ...widget, gap: Number(e.target.value) })}
              inputProps={{ style: { color: '#F8FAFC', fontSize: 13 } }}
              sx={{ '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' }, '& .MuiInputLabel-root': { color: '#94A3B8', fontSize: 12 } }}
            />
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography variant="caption" color="text.secondary">Collapsed</Typography>
              <Chip
                label={widget.collapsed ? 'Yes' : 'No'}
                size="small"
                onClick={() => onUpdate({ ...widget, collapsed: !widget.collapsed })}
                sx={{ bgcolor: widget.collapsed ? 'rgba(239,68,68,0.15)' : 'rgba(16,185,129,0.15)', color: widget.collapsed ? '#EF4444' : '#10B981', fontSize: 10, cursor: 'pointer' }}
              />
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary">
                {widget.children.length} item(s) in this container.
              </Typography>
            </Box>
          </Stack>
        </AccordionDetails>
      </Accordion>

      <Divider sx={{ borderColor: '#1E293B' }} />

      <Accordion disableGutters sx={{ bgcolor: 'transparent', boxShadow: 'none' }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />} sx={{ minHeight: 36, py: 0 }}>
          <Typography variant="caption" sx={{ fontWeight: 700, color: '#94A3B8', textTransform: 'uppercase', letterSpacing: 0.5, fontSize: '0.65rem' }}>
            Style
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0, pb: 1.5 }}>
          <ContainerStyleEditor
            style={containerStyle}
            onChange={(cs) => onUpdate({ ...widget, containerStyle: cs })}
          />
        </AccordionDetails>
      </Accordion>

      <Divider sx={{ borderColor: '#1E293B' }} />

      <Accordion disableGutters sx={{ bgcolor: 'transparent', boxShadow: 'none' }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748B' }} />} sx={{ minHeight: 36, py: 0 }}>
          <Typography variant="caption" sx={{ fontWeight: 700, color: '#94A3B8', textTransform: 'uppercase', letterSpacing: 0.5, fontSize: '0.65rem' }}>
            Visibility Condition
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0, pb: 1.5 }}>
          <Stack spacing={1} sx={{ px: 0 }}>
            {((widget as any).visibilityCondition) ? (
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Chip
                  label="Condition active"
                  size="small"
                  sx={{ bgcolor: 'rgba(0,212,255,0.1)', color: '#00D4FF', fontSize: 10 }}
                />
                <Box sx={{ display: 'flex', gap: 0.5 }}>
                  <Chip
                    label="Edit"
                    size="small"
                    onClick={() => setConditionDialogOpen(true)}
                    sx={{ bgcolor: 'rgba(0,212,255,0.15)', color: '#00D4FF', fontSize: 10, cursor: 'pointer' }}
                  />
                  <Chip
                    label="Clear"
                    size="small"
                    onClick={() => onUpdate({ ...widget, presentation: { ...widget.presentation, visibilityCondition: null } })}
                    sx={{ bgcolor: 'rgba(239,68,68,0.15)', color: '#EF4444', fontSize: 10, cursor: 'pointer' }}
                  />
                </Box>
              </Box>
            ) : (
              <Button
                size="small"
                variant="outlined"
                onClick={() => setConditionDialogOpen(true)}
                sx={{ color: '#94A3B8', borderColor: '#1E293B', textTransform: 'none', fontSize: 11 }}
              >
                + Set visibility condition…
              </Button>
            )}
          </Stack>
        </AccordionDetails>
      </Accordion>

      <ConditionGroupEditor
        open={conditionDialogOpen}
        onClose={() => setConditionDialogOpen(false)}
        value={((widget as any).visibilityCondition) || null}
        onChange={(group) => { const w = widget as any; w.visibilityCondition = group; onUpdate(w as CanvasWidget); }}
      />
    </Stack>
  );
};

const RelatedObjectWidgetPanel: React.FC<{ widget: RelatedObjectWidget; onUpdate: (w: CanvasWidget) => void }> = ({ widget, onUpdate }) => {
  return (
    <Stack spacing={2}>
      <TextField
        label="Title"
        value={widget.title}
        size="small"
        fullWidth
        onChange={(e) => onUpdate({ ...widget, title: e.target.value })}
        inputProps={{ style: { color: '#F8FAFC', fontSize: 13 } }}
        sx={{
          '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' },
          '& .MuiInputLabel-root': { color: '#94A3B8', fontSize: 12 },
        }}
      />
      <Box>
        <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1, display: 'block', mb: 1 }}>
          Target Business Object
        </Typography>
        <Typography variant="body2" sx={{ color: '#F8FAFC' }}>{widget.targetBoKey}</Typography>
        <Typography variant="caption" color="text.secondary">ID: {widget.targetBoId}</Typography>
      </Box>
      <Box>
        <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1, display: 'block', mb: 1 }}>
          Cardinality
        </Typography>
        <Typography variant="body2" sx={{ color: '#F8FAFC' }}>{widget.cardinality}</Typography>
      </Box>
      <Box>
        <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1, display: 'block', mb: 1 }}>
          Display Columns
        </Typography>
        <Stack direction="row" gap={0.5} flexWrap="wrap">
          {widget.displayColumns.map((col) => (
            <Chip key={col} label={col} size="small" sx={{ bgcolor: 'rgba(0,212,255,0.1)', color: '#00D4FF', fontSize: 10 }} />
          ))}
          {widget.displayColumns.length === 0 && (
            <Typography variant="caption" color="text.secondary">None selected</Typography>
          )}
        </Stack>
      </Box>
    </Stack>
  );
};

export default PageStudioPropertiesPanel;
