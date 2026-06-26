import type { FC } from 'react';
import { Box, Typography, Accordion, AccordionSummary, AccordionDetails, Grid, TextField, FormControl, InputLabel, Select, MenuItem } from '@mui/material';
import { ChevronDown } from 'lucide-react';
import { ELEMENT_TYPES, datasets, sanitizeInput } from './reportingUtils';

const PropertiesPanel: FC<any> = ({ selectedElement, onElementUpdate }) => {
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
        <AccordionSummary expandIcon={<ChevronDown />}>
          <Typography variant="subtitle2">General</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <Grid container spacing={2}>
            <Grid item xs={12}>
              <TextField fullWidth size="small" label="Name" value={selectedElement.properties.name || ''} onChange={(e) => updateProperty('name', e.target.value)} />
            </Grid>
            {selectedElement.type === ELEMENT_TYPES.TEXTBOX && (
              <>
                <Grid item xs={12}>
                  <TextField fullWidth size="small" multiline rows={3} label="Text" value={selectedElement.properties.text || ''} onChange={(e) => updateProperty('text', e.target.value)} />
                </Grid>
                <Grid item xs={6}>
                  <FormControl fullWidth size="small"><InputLabel>Font Size</InputLabel><Select value={selectedElement.properties.fontSize || 12} onChange={(e) => updateProperty('fontSize', e.target.value)}>{[8,9,10,11,12,14,16,18,20,24,28,32].map(size => <MenuItem key={size} value={size}>{size}pt</MenuItem>)}</Select></FormControl>
                </Grid>
                <Grid item xs={6}>
                  <FormControl fullWidth size="small"><InputLabel>Text Align</InputLabel><Select value={selectedElement.properties.textAlign || 'left'} onChange={(e) => updateProperty('textAlign', e.target.value)}><MenuItem value="left">Left</MenuItem><MenuItem value="center">Center</MenuItem><MenuItem value="right">Right</MenuItem></Select></FormControl>
                </Grid>
              </>
            )}
            {selectedElement.type === ELEMENT_TYPES.TABLE && (
              <>
                <Grid item xs={12}>
                  <FormControl fullWidth size="small"><InputLabel>Data Source</InputLabel><Select value={selectedElement.properties.dataSource || ''} onChange={(e) => updateProperty('dataSource', e.target.value)}>{datasets.map(ds => <MenuItem key={ds.id} value={ds.id}>{ds.name}</MenuItem>)}</Select></FormControl>
                </Grid>
                <Grid item xs={12}><TextField fullWidth size="small" label="Columns (comma separated)" value={(selectedElement.properties.columns || []).join(', ')} onChange={(e) => updateProperty('columns', e.target.value.split(', '))} /></Grid>
              </>
            )}
          </Grid>
        </AccordionDetails>
      </Accordion>
      <Accordion>
        <AccordionSummary expandIcon={<ChevronDown />}><Typography variant="subtitle2">Expressions & Formatting</Typography></AccordionSummary>
        <AccordionDetails>
          <Grid container spacing={2}>
            <Grid item xs={12}><TextField fullWidth size="small" multiline minRows={2} label="Value Expression" value={selectedElement.properties.valueExpression || ''} onChange={(e) => updateProperty('valueExpression', e.target.value)} /></Grid>
            <Grid item xs={12}><TextField fullWidth size="small" multiline minRows={2} label="Conditional Expression" helperText="Example: =IIF(Fields!Growth.Value < 0, true, false)" value={selectedElement.properties.conditionalExpression || ''} onChange={(e) => updateProperty('conditionalExpression', e.target.value)} /></Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>
      <Accordion>
        <AccordionSummary expandIcon={<ChevronDown />}><Typography variant="subtitle2">Layout</Typography></AccordionSummary>
        <AccordionDetails>
          <Grid container spacing={2}>
            <Grid item xs={6}><TextField fullWidth size="small" type="number" label="Width" value={selectedElement.size.width} onChange={(e) => onElementUpdate(selectedElement.id, { size: { ...selectedElement.size, width: Number(e.target.value) } })} /></Grid>
            <Grid item xs={6}><TextField fullWidth size="small" type="number" label="Height" value={selectedElement.size.height} onChange={(e) => onElementUpdate(selectedElement.id, { size: { ...selectedElement.size, height: Number(e.target.value) } })} /></Grid>
            <Grid item xs={6}><TextField fullWidth size="small" type="number" label="X Position" value={selectedElement.position.x} onChange={(e) => onElementUpdate(selectedElement.id, { position: { ...selectedElement.position, x: Number(e.target.value) } })} /></Grid>
            <Grid item xs={6}><TextField fullWidth size="small" type="number" label="Y Position" value={selectedElement.position.y} onChange={(e) => onElementUpdate(selectedElement.id, { position: { ...selectedElement.position, y: Number(e.target.value) } })} /></Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>
    </Box>
  );
};

export default PropertiesPanel;
