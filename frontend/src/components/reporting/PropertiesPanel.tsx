import type { FC } from 'react';
import { Box, Typography, Accordion, AccordionSummary, AccordionDetails, Grid, TextField, FormControl, InputLabel, Select, MenuItem } from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { ELEMENT_TYPES, datasets, sanitizeInput } from './reportingUtils';
import ExpressionEditorField from '../ExpressionBuilder/ExpressionEditorField';

const FONT_FAMILIES = ['inherit', 'Arial, sans-serif', 'Georgia, serif', '"Courier New", monospace', 'Roboto, sans-serif'];
const FONT_WEIGHTS = [400, 500, 600, 700];

const PropertiesPanel: FC<any> = ({ selectedElement, onElementUpdate, selectedBO }) => {
  if (!selectedElement) {
    return (
      <Box sx={{ p: 2, textAlign: 'center' }}>
        <Typography variant="body2" color="text.secondary">Select an element to view properties</Typography>
      </Box>
    );
  }

  const updateProperty = (property: string, value: any) => {
    const sanitizedValue = typeof value === 'string' ? sanitizeInput(value) : value;
    if (property === 'text' && !sanitizedValue.trim()) return;
    onElementUpdate(selectedElement.id, { properties: { ...selectedElement.properties, [property]: sanitizedValue } });
  };

  return (
    <Box sx={{ p: 2, maxHeight: '70vh', overflow: 'auto' }}>
      <Typography variant="h6" gutterBottom>{selectedElement.type} Properties</Typography>
      <Accordion defaultExpanded>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2">General</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <Grid container spacing={2}>
            <Grid size={12}>
              <TextField fullWidth size="small" label="Name" value={selectedElement.properties.name || ''} onChange={(e) => updateProperty('name', e.target.value)} />
            </Grid>
            {selectedElement.type === ELEMENT_TYPES.TEXTBOX && (
              <>
                <Grid size={12}>
                  <TextField fullWidth size="small" multiline rows={3} label="Text" value={selectedElement.properties.text || ''} onChange={(e) => updateProperty('text', e.target.value)} />
                </Grid>
                <Grid size={6}>
                  <FormControl fullWidth size="small"><InputLabel>Font Size</InputLabel><Select value={selectedElement.properties.fontSize || 12} onChange={(e) => updateProperty('fontSize', e.target.value)}>{[8,9,10,11,12,14,16,18,20,24,28,32].map(size => <MenuItem key={size} value={size}>{size}pt</MenuItem>)}</Select></FormControl>
                </Grid>
                <Grid size={6}>
                  <FormControl fullWidth size="small"><InputLabel>Text Align</InputLabel><Select value={selectedElement.properties.textAlign || 'left'} onChange={(e) => updateProperty('textAlign', e.target.value)}><MenuItem value="left">Left</MenuItem><MenuItem value="center">Center</MenuItem><MenuItem value="right">Right</MenuItem></Select></FormControl>
                </Grid>
                <Grid size={6}>
                  <FormControl fullWidth size="small"><InputLabel>Font Family</InputLabel><Select value={selectedElement.properties.fontFamily || 'inherit'} onChange={(e) => updateProperty('fontFamily', e.target.value)}>{FONT_FAMILIES.map((f) => <MenuItem key={f} value={f}>{f}</MenuItem>)}</Select></FormControl>
                </Grid>
                <Grid size={6}>
                  <FormControl fullWidth size="small"><InputLabel>Font Weight</InputLabel><Select value={selectedElement.properties.fontWeight || 400} onChange={(e) => updateProperty('fontWeight', e.target.value)}>{FONT_WEIGHTS.map((w) => <MenuItem key={w} value={w}>{w}</MenuItem>)}</Select></FormControl>
                </Grid>
                <Grid size={6}>
                  <TextField fullWidth size="small" type="color" label="Text Color" InputLabelProps={{ shrink: true }} value={selectedElement.properties.textColor || '#111827'} onChange={(e) => updateProperty('textColor', e.target.value)} />
                </Grid>
                <Grid size={6}>
                  <TextField fullWidth size="small" type="color" label="Background Color" InputLabelProps={{ shrink: true }} value={selectedElement.properties.backgroundColor || '#ffffff'} onChange={(e) => updateProperty('backgroundColor', e.target.value)} />
                </Grid>
                <Grid size={6}>
                  <TextField fullWidth size="small" type="number" label="Padding (px)" value={selectedElement.properties.padding ?? ''} onChange={(e) => updateProperty('padding', e.target.value)} />
                </Grid>
                <Grid size={6}>
                  <FormControl fullWidth size="small"><InputLabel>Format</InputLabel><Select value={selectedElement.properties.format || 'Text'} onChange={(e) => updateProperty('format', e.target.value)}><MenuItem value="Text">Text</MenuItem><MenuItem value="Number">Number</MenuItem><MenuItem value="Date">Date</MenuItem><MenuItem value="Boolean">Boolean</MenuItem></Select></FormControl>
                </Grid>
              </>
            )}
            {selectedElement.type === ELEMENT_TYPES.TABLE && (
              <>
                <Grid  size={{ xs: 12 }}>
                  <FormControl fullWidth size="small"><InputLabel>Data Source</InputLabel><Select value={selectedElement.properties.dataSource || ''} onChange={(e) => updateProperty('dataSource', e.target.value)}>{datasets.map(ds => <MenuItem key={ds.id} value={ds.id}>{ds.name}</MenuItem>)}</Select></FormControl>
                </Grid>
                <Grid  size={{ xs: 12 }}><TextField fullWidth size="small" label="Columns (comma separated)" value={(selectedElement.properties.columns || []).join(', ')} onChange={(e) => updateProperty('columns', e.target.value.split(', '))} /></Grid>
              </>
            )}
          </Grid>
        </AccordionDetails>
      </Accordion>
      <Accordion>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}><Typography variant="subtitle2">Expressions & Formatting</Typography></AccordionSummary>
        <AccordionDetails>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12 }}>
              <ExpressionEditorField
                label="Value Expression"
                value={selectedElement.properties.valueExpression || ''}
                onChange={(v) => updateProperty('valueExpression', v)}
                boName={selectedBO?.key || selectedBO?.technicalName || selectedBO?.name}
              />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <ExpressionEditorField
                label="Conditional Expression (e.g. Growth < 0)"
                value={selectedElement.properties.conditionalExpression || ''}
                onChange={(v) => updateProperty('conditionalExpression', v)}
                boName={selectedBO?.key || selectedBO?.technicalName || selectedBO?.name}
              />
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>
      <Accordion>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}><Typography variant="subtitle2">Layout</Typography></AccordionSummary>
        <AccordionDetails>
          <Grid container spacing={2}>
            <Grid  size={{ xs: 6 }}><TextField fullWidth size="small" type="number" label="Width" value={selectedElement.size.width} onChange={(e) => onElementUpdate(selectedElement.id, { size: { ...selectedElement.size, width: Number(e.target.value) } })} /></Grid>
            <Grid  size={{ xs: 6 }}><TextField fullWidth size="small" type="number" label="Height" value={selectedElement.size.height} onChange={(e) => onElementUpdate(selectedElement.id, { size: { ...selectedElement.size, height: Number(e.target.value) } })} /></Grid>
            <Grid  size={{ xs: 6 }}><TextField fullWidth size="small" type="number" label="X Position" value={selectedElement.position.x} onChange={(e) => onElementUpdate(selectedElement.id, { position: { ...selectedElement.position, x: Number(e.target.value) } })} /></Grid>
            <Grid  size={{ xs: 6 }}><TextField fullWidth size="small" type="number" label="Y Position" value={selectedElement.position.y} onChange={(e) => onElementUpdate(selectedElement.id, { position: { ...selectedElement.position, y: Number(e.target.value) } })} /></Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>
    </Box>
  );
};

export default PropertiesPanel;
