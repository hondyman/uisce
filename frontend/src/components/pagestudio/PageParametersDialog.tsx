import React, { useState, useEffect } from 'react';
import {
  Dialog, DialogTitle, DialogContent, DialogActions,
  Button, Box, Typography, Stack, TextField, Select, MenuItem,
  IconButton, Chip, Divider, FormControl, InputLabel,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import EditIcon from '@mui/icons-material/Edit';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import type { PageParameter } from '../pagestudio/pageStudioTypes';

interface PageParametersDialogProps {
  open: boolean;
  onClose: () => void;
  parameters: PageParameter[];
  onAdd: (param: PageParameter) => void;
  onUpdate: (param: PageParameter) => void;
  onDelete: (paramKey: string) => void;
}

const PAGE_PARAMETER_PRESETS: Omit<PageParameter, 'key'>[] = [
  { displayName: 'Selected Account ID', dataType: 'string', defaultValue: '' },
  { displayName: 'Selected Client ID', dataType: 'string', defaultValue: '' },
  { displayName: 'Start Date', dataType: 'date', defaultValue: '' },
  { displayName: 'End Date', dataType: 'date', defaultValue: '' },
  { displayName: 'Year', dataType: 'number', defaultValue: new Date().getFullYear() },
  { displayName: 'Status', dataType: 'string', defaultValue: '' },
  { displayName: 'Portfolio List', dataType: 'list', defaultValue: '' },
  { displayName: 'Branch Code', dataType: 'string', defaultValue: '' },
];

const ParameterEditor: React.FC<{
  param: Partial<PageParameter> | null;
  onSave: (param: PageParameter) => void;
  onCancel: () => void;
  existingKeys: string[];
}> = ({ param, onSave, onCancel, existingKeys: _existingKeys }) => {
  const [formData, setFormData] = useState<Partial<PageParameter>>({
    key: '',
    displayName: '',
    dataType: 'string',
    defaultValue: '',
  });

  useEffect(() => {
    setFormData(param || { key: '', displayName: '', dataType: 'string', defaultValue: '' });
  }, [param]);

  const handleChange = (field: keyof PageParameter, value: unknown) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (field === 'displayName' && !param?.key) {
      const autoKey = value
        .toString()
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '_')
        .replace(/^_+|_+$/g, '') || 'param';
      setFormData((prev) => ({ ...prev, key: autoKey }));
    }
  };

  if (!param) return null;

  return (
    <Box sx={{ p: 2, bgcolor: '#0B1E36', borderRadius: 1, border: '1px solid #1E293B' }}>
      <Typography variant="caption" sx={{ color: '#00D4FF', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1 }}>
        {param.key ? 'Edit Parameter' : 'New Parameter'}
      </Typography>
      <Stack spacing={1.5} sx={{ mt: 1.5 }}>
        <TextField
          label="Parameter Key"
          value={formData.key || ''}
          onChange={(e) => handleChange('key', e.target.value)}
          size="small"
          fullWidth
          disabled={!!param.key}
          placeholder="e.g. selected_account_id"
          inputProps={{ style: { color: '#F8FAFC', fontSize: 13 } }}
          sx={{ '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' }, '& .MuiInputLabel-root': { color: '#94A3B8', fontSize: 12 } }}
        />
        <TextField
          label="Display Name"
          value={formData.displayName || ''}
          onChange={(e) => handleChange('displayName', e.target.value)}
          size="small"
          fullWidth
          placeholder="e.g. Selected Account"
          inputProps={{ style: { color: '#F8FAFC', fontSize: 13 } }}
          sx={{ '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' }, '& .MuiInputLabel-root': { color: '#94A3B8', fontSize: 12 } }}
        />
        <FormControl fullWidth size="small">
          <InputLabel sx={{ color: '#94A3B8', fontSize: 12 }}>Data Type</InputLabel>
          <Select
            value={formData.dataType || 'string'}
            label="Data Type"
            onChange={(e) => handleChange('dataType', e.target.value)}
            sx={{ color: '#F8FAFC', fontSize: 13, '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' } }}
          >
            <MenuItem value="string">String</MenuItem>
            <MenuItem value="number">Number</MenuItem>
            <MenuItem value="date">Date</MenuItem>
            <MenuItem value="boolean">Boolean (Yes/No)</MenuItem>
            <MenuItem value="list">List (Multi-select)</MenuItem>
          </Select>
        </FormControl>
        <TextField
          label="Default Value"
          value={formData.defaultValue ?? ''}
          onChange={(e) => handleChange('defaultValue', e.target.value)}
          size="small"
          fullWidth
          placeholder="Leave empty for no default"
          inputProps={{ style: { color: '#F8FAFC', fontSize: 13 } }}
          sx={{ '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' }, '& .MuiInputLabel-root': { color: '#94A3B8', fontSize: 12 } }}
        />
        <Stack direction="row" spacing={1} justifyContent="flex-end">
          <Button size="small" onClick={onCancel} sx={{ color: '#94A3B8', textTransform: 'none' }}>Cancel</Button>
          <Button
            size="small"
            variant="contained"
            onClick={() => {
              if (!formData.key || !formData.displayName) return;
              onSave(formData as PageParameter);
            }}
            disabled={!formData.key || !formData.displayName}
            sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 700 }}
          >
            {param.key ? 'Update' : 'Add'}
          </Button>
        </Stack>
      </Stack>
    </Box>
  );
};

export const PageParametersDialog: React.FC<PageParametersDialogProps> = ({
  open, onClose, parameters, onAdd, onUpdate, onDelete,
}) => {
  const [editingParam, setEditingParam] = useState<Partial<PageParameter> | null>(null);
  const [showPresets] = useState(true);

  const existingKeys = parameters.map((p) => p.key);

  const handleSave = (param: PageParameter) => {
    if (existingKeys.includes(param.key) && !parameters.find((p) => p.key === param.key)) {
      return;
    }
    const existing = parameters.find((p) => p.key === param.key);
    if (existing) {
      onUpdate(param);
    } else {
      onAdd(param);
    }
    setEditingParam(null);
  };

  const handleClone = (param: PageParameter) => {
    const base = param.key;
    let counter = 1;
    let newKey = `${base}_copy`;
    while (existingKeys.includes(newKey)) {
      counter++;
      newKey = `${base}_copy${counter}`;
    }
    setEditingParam({ ...param, key: newKey, displayName: `${param.displayName} (Copy)` });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ bgcolor: '#071526', borderBottom: '1px solid #1E293B', color: '#F8FAFC' }}>
        <Typography variant="subtitle1" fontWeight={700}>Page Parameters</Typography>
        <Typography variant="caption" color="text.secondary">
          Define parameters that widgets on this page can subscribe to for reactive updates.
        </Typography>
      </DialogTitle>
      <DialogContent sx={{ bgcolor: '#030B15', p: 2 }}>
        {editingParam !== null ? (
          <ParameterEditor
            param={editingParam}
            onSave={handleSave}
            onCancel={() => setEditingParam(null)}
            existingKeys={existingKeys}
          />
        ) : (
          <>
            {showPresets && PAGE_PARAMETER_PRESETS.length > 0 && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1 }}>
                  Quick Add Presets
                </Typography>
                <Stack direction="row" gap={0.5} flexWrap="wrap" sx={{ mt: 0.5 }}>
                  {PAGE_PARAMETER_PRESETS.map((preset) => {
                    const autoKey = preset.displayName
                      .toLowerCase()
                      .replace(/[^a-z0-9]+/g, '_')
                      .replace(/^_+|_+$/g, '');
                    const alreadyExists = existingKeys.includes(autoKey);
                    return (
                      <Chip
                        key={preset.displayName}
                        label={preset.displayName}
                        size="small"
                        onClick={() => {
                          if (!alreadyExists) {
                            onAdd({ ...preset, key: autoKey });
                          }
                        }}
                        disabled={alreadyExists}
                        sx={{
                          bgcolor: alreadyExists ? 'transparent' : 'rgba(2,132,199,0.15)',
                          color: alreadyExists ? '#64748B' : '#00D4FF',
                          border: '1px solid',
                          borderColor: alreadyExists ? '#1E293B' : '#0284C7',
                          fontSize: 10,
                          cursor: alreadyExists ? 'default' : 'pointer',
                        }}
                      />
                    );
                  })}
                </Stack>
              </Box>
            )}

            <Divider sx={{ borderColor: '#1E293B', mb: 2 }} />

            {parameters.length === 0 ? (
              <Box sx={{ textAlign: 'center', py: 4 }}>
                <Typography variant="body2" color="text.secondary">
                  No parameters defined yet.
                </Typography>
                <Typography variant="caption" color="text.secondary" display="block" sx={{ mt: 0.5 }}>
                  Add parameters above or click the + button below.
                </Typography>
              </Box>
            ) : (
              <Stack spacing={1}>
                {parameters.map((param) => (
                  <Paper
                    key={param.key}
                    sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid #1E293B', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}
                  >
                    <Box sx={{ flex: 1, minWidth: 0 }}>
                      <Stack direction="row" alignItems="center" gap={1}>
                        <Typography variant="body2" fontWeight={700} sx={{ color: '#F8FAFC' }}>{param.displayName}</Typography>
                        <Chip label={param.dataType} size="small" sx={{ bgcolor: 'rgba(0,212,255,0.1)', color: '#00D4FF', fontSize: 9, height: 16 }} />
                      </Stack>
                      <Typography variant="caption" sx={{ color: '#64748B', fontFamily: 'monospace' }}>key: {param.key}</Typography>
                      {param.defaultValue !== undefined && param.defaultValue !== '' && (
                        <Typography variant="caption" display="block" sx={{ color: '#64748B' }}>default: {String(param.defaultValue)}</Typography>
                      )}
                    </Box>
                    <Stack direction="row">
                      <Tooltip title="Edit">
                        <IconButton size="small" onClick={() => setEditingParam(param)} sx={{ color: '#94A3B8' }}>
                          <EditIcon sx={{ fontSize: 14 }} />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Clone">
                        <IconButton size="small" onClick={() => handleClone(param)} sx={{ color: '#94A3B8' }}>
                          <ContentCopyIcon sx={{ fontSize: 14 }} />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Delete">
                        <IconButton size="small" onClick={() => onDelete(param.key)} sx={{ color: '#EF4444' }}>
                          <DeleteOutlineIcon sx={{ fontSize: 14 }} />
                        </IconButton>
                      </Tooltip>
                    </Stack>
                  </Paper>
                ))}
              </Stack>
            )}

            <Button
              size="small"
              startIcon={<AddIcon />}
              onClick={() => setEditingParam({ key: '', displayName: '', dataType: 'string', defaultValue: '' })}
              sx={{ mt: 2, color: '#00D4FF', textTransform: 'none' }}
            >
              Add Parameter
            </Button>
          </>
        )}
      </DialogContent>
      <DialogActions sx={{ bgcolor: '#071526', borderTop: '1px solid #1E293B', p: 1.5 }}>
        <Button onClick={onClose} sx={{ color: '#94A3B8', textTransform: 'none' }}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};

export default PageParametersDialog;
