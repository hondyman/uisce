import React, { useState, useEffect } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Box, TableContainer, Paper, Table, TableHead, TableRow, TableCell, TableBody, IconButton, TextField, Select, MenuItem, FormControl, InputLabel, Checkbox, FormControlLabel } from '@mui/material';
import { Plus, Settings, Trash2 } from 'lucide-react';

type ReportParameter = {
  id: string;
  name: string;
  type: 'string' | 'number' | 'date' | 'boolean';
  prompt: string;
  defaultValue?: string;
  allowBlank?: boolean;
  allowMultiple?: boolean;
};

type Props = {
  open: boolean;
  onClose: () => void;
  parameters: ReportParameter[];
  onAdd: (param: Omit<ReportParameter, 'id'>) => void;
  onUpdate: (param: ReportParameter) => void;
  onDelete: (paramId: string) => void;
};

const ParameterEditor: React.FC<{
  param: Partial<ReportParameter> | null;
  onSave: (param: any) => void;
  onCancel: () => void;
}> = ({ param, onSave, onCancel }) => {
  const [formData, setFormData] = useState<Partial<ReportParameter>>({});

  useEffect(() => {
    setFormData(param || { name: '', type: 'string', prompt: '', defaultValue: '', allowBlank: false, allowMultiple: false });
  }, [param]);

  const handleChange = (field: keyof ReportParameter, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  if (!param) return null;

  return (
    <Dialog open={!!param} onClose={onCancel} maxWidth="xs" fullWidth>
      <DialogTitle>{param.id ? 'Edit Parameter' : 'Add Parameter'}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
          <TextField label="Name" value={formData.name || ''} onChange={(e) => handleChange('name', e.target.value)} />
          <FormControl fullWidth>
            <InputLabel>Type</InputLabel>
            <Select value={formData.type || 'string'} label="Type" onChange={(e) => handleChange('type', e.target.value)}>
              <MenuItem value="string">String</MenuItem>
              <MenuItem value="number">Number</MenuItem>
              <MenuItem value="date">Date</MenuItem>
              <MenuItem value="boolean">Boolean</MenuItem>
            </Select>
          </FormControl>
          <TextField label="Prompt" value={formData.prompt || ''} onChange={(e) => handleChange('prompt', e.target.value)} />
          <TextField label="Default Value" value={formData.defaultValue || ''} onChange={(e) => handleChange('defaultValue', e.target.value)} />
          <FormControlLabel control={<Checkbox checked={!!formData.allowBlank} onChange={(e) => handleChange('allowBlank', e.target.checked)} />} label="Allow Blank" />
          <FormControlLabel control={<Checkbox checked={!!formData.allowMultiple} onChange={(e) => handleChange('allowMultiple', e.target.checked)} />} label="Allow Multiple Values" />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onCancel}>Cancel</Button>
        <Button onClick={() => onSave(formData)} variant="contained">Save</Button>
      </DialogActions>
    </Dialog>
  );
};

const ParametersDialog: React.FC<Props> = ({ open, onClose, parameters, onAdd, onUpdate, onDelete }) => {
  const [editingParam, setEditingParam] = useState<Partial<ReportParameter> | null>(null);

  const handleSave = (paramData: ReportParameter) => {
    if (paramData.id) {
      onUpdate(paramData);
    } else {
      onAdd(paramData);
    }
    setEditingParam(null);
  };

  return (
    <>
      <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
        <DialogTitle>Report Parameters</DialogTitle>
        <DialogContent>
          <Box sx={{ mb: 2 }}>
            <Button variant="contained" startIcon={<Plus />} sx={{ mb: 2 }} onClick={() => setEditingParam({})}>Add Parameter</Button>
          </Box>
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Name</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Prompt</TableCell>
                  <TableCell>Default Value</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {parameters.map((param) => (
                  <TableRow key={param.id}>
                    <TableCell>{param.name}</TableCell>
                    <TableCell>{param.type}</TableCell>
                    <TableCell>{param.prompt}</TableCell>
                    <TableCell>{param.defaultValue}</TableCell>
                    <TableCell>
                      <IconButton size="small" onClick={() => setEditingParam(param)}><Settings size={16} /></IconButton>
                      <IconButton size="small" onClick={() => onDelete(param.id)}><Trash2 size={16} /></IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>Close</Button>
        </DialogActions>
      </Dialog>
      <ParameterEditor param={editingParam} onSave={handleSave} onCancel={() => setEditingParam(null)} />
    </>
  );
};

export default ParametersDialog;
