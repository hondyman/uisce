import React, { useState, useEffect } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Box, TableContainer, Paper, Table, TableHead, TableRow, TableCell, TableBody, IconButton, TextField, Select, MenuItem, FormControl, InputLabel, Checkbox, FormControlLabel, FormHelperText } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import SettingsIcon from '@mui/icons-material/Settings';
import DeleteIcon from '@mui/icons-material/Delete';
import type { ParamSpec } from '../../studio-core/params/ParamSpec';
import { fetchBOTerms } from '../../features/query-builder/services/queryBuilderApi';
import type { SemanticTermView } from '../../features/query-builder/types/queryDef';

type ReportParameter = ParamSpec;

type Props = {
  open: boolean;
  onClose: () => void;
  parameters: ReportParameter[];
  onAdd: (param: Omit<ReportParameter, 'id'>) => void;
  onUpdate: (param: ReportParameter) => void;
  onDelete: (paramId: string) => void;
  boId?: string;
  boKey?: string;
  bindingId?: string;
};

const ParameterEditor: React.FC<{
  param: Partial<ReportParameter> | null;
  onSave: (param: any) => void;
  onCancel: () => void;
  boId?: string;
  boKey?: string;
  bindingId?: string;
}> = ({ param, onSave, onCancel, boId, boKey, bindingId }) => {
  const [formData, setFormData] = useState<Partial<ReportParameter>>({});
  const [boFields, setBoFields] = useState<SemanticTermView[]>([]);

  useEffect(() => {
    setFormData(param || { name: '', type: 'string', prompt: '', defaultValue: '', allowBlank: false, allowMultiple: false });
  }, [param]);

  useEffect(() => {
    if (!param || !boId || !bindingId) {
      setBoFields([]);
      return;
    }
    let cancelled = false;
    fetchBOTerms(boId, bindingId)
      .then((t) => {
        if (!cancelled) setBoFields(t);
      })
      .catch(() => {
        if (!cancelled) setBoFields([]);
      });
    return () => {
      cancelled = true;
    };
  }, [param, boId, bindingId]);

  const handleChange = (field: keyof ReportParameter, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleFieldBindingChange = (termKey: string) => {
    setFormData(prev => ({
      ...prev,
      source: termKey ? { kind: 'ref', boKey: boKey || '', termKey } : undefined,
    }));
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
          <FormControl fullWidth>
            <InputLabel>Bind to Field</InputLabel>
            <Select
              value={(formData as any).source?.kind === 'ref' ? (formData as any).source.termKey : ''}
              label="Bind to Field"
              onChange={(e) => handleFieldBindingChange(e.target.value as string)}
            >
              <MenuItem value="">— Not bound —</MenuItem>
              {boFields.map((f) => (
                <MenuItem key={f.termKey} value={f.termKey}>{f.displayName || f.termName}</MenuItem>
              ))}
            </Select>
            <FormHelperText>An unbound parameter has no effect on report results.</FormHelperText>
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

const ParametersDialog: React.FC<Props> = ({ open, onClose, parameters, onAdd, onUpdate, onDelete, boId, boKey, bindingId }) => {
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
            <Button variant="contained" startIcon={<AddIcon />} sx={{ mb: 2 }} onClick={() => setEditingParam({})}>Add Parameter</Button>
          </Box>
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Name</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Bound Field</TableCell>
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
                    <TableCell>
                      {(param as any).source?.kind === 'ref'
                        ? (param as any).source.termKey
                        : <em>not bound - has no effect on results</em>}
                    </TableCell>
                    <TableCell>{param.prompt}</TableCell>
                    <TableCell>{param.defaultValue}</TableCell>
                    <TableCell>
                      <IconButton size="small" onClick={() => setEditingParam(param)}><SettingsIcon fontSize="small" /></IconButton>
                      <IconButton size="small" onClick={() => onDelete(param.id)}><DeleteIcon fontSize="small" /></IconButton>
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
      <ParameterEditor param={editingParam} onSave={handleSave} onCancel={() => setEditingParam(null)} boId={boId} boKey={boKey} bindingId={bindingId} />
    </>
  );
};

export default ParametersDialog;
