import type { FC } from 'react';
import { Grid, Typography, Button, IconButton, TextField, FormControl, InputLabel, Select, MenuItem, Divider } from '@mui/material';
import { Plus, Trash2 } from 'lucide-react';

type Field = any;

type Props = {
  calculatedFields: Field[];
  datasets: any[];
  onAddCalculatedField: () => void;
  onCalculatedFieldChange: (fieldId: string, key: string, value: any) => void;
  onRemoveCalculatedField: (fieldId: string) => void;
};

const CalculatedFieldsEditor: FC<Props> = ({ calculatedFields, datasets, onAddCalculatedField, onCalculatedFieldChange, onRemoveCalculatedField }) => {
  return (
    <>
      <Typography variant="subtitle1" gutterBottom>Calculated Fields, Expressions & Delivery</Typography>
      <Divider sx={{ my: 2 }}>Calculated Fields</Divider>
      {calculatedFields.map((field) => (
        <Grid container spacing={1.5} key={field.id} alignItems="center" sx={{ mb: 1 }}>
          <Grid item xs={12} sm={3}><TextField fullWidth size="small" label="Name" value={field.name} onChange={(e) => onCalculatedFieldChange(field.id, 'name', e.target.value)} /></Grid>
          <Grid item xs={12} sm={5}><TextField fullWidth size="small" label="Expression" value={field.expression} onChange={(e) => onCalculatedFieldChange(field.id, 'expression', e.target.value)} /></Grid>
          <Grid item xs={12} sm={2}><FormControl fullWidth size="small"><InputLabel>Dataset</InputLabel><Select label="Dataset" value={field.datasetId} onChange={(e) => onCalculatedFieldChange(field.id, 'datasetId', e.target.value)}>{datasets.map((ds) => (<MenuItem key={`${field.id}_${ds.id}`} value={ds.id}>{ds.name}</MenuItem>))}</Select></FormControl></Grid>
          <Grid item xs={12} sm={1.5}><TextField fullWidth size="small" label="Format" value={field.format ?? ''} onChange={(e) => onCalculatedFieldChange(field.id, 'format', e.target.value)} /></Grid>
          <Grid item xs={12} sm={0.5} sx={{ textAlign: 'right' }}><IconButton size="small" onClick={() => onRemoveCalculatedField(field.id)}><Trash2 size={16} /></IconButton></Grid>
        </Grid>
      ))}
      <Button size="small" startIcon={<Plus size={14} />} onClick={onAddCalculatedField}>Add Calculated Field</Button>
    </>
  );
};

export default CalculatedFieldsEditor;
