import React, { useState } from 'react';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  Select,
  Switch,
  TextField,
} from '@mui/material';
import type { CreateAttributeInput } from '../types';

const DATA_TYPES = [
  'string',
  'integer',
  'decimal',
  'boolean',
  'date',
  'enum',
  'json',
];

interface AddFieldDialogProps {
  open: boolean;
  entityType: string;
  tableRef: string;
  onClose: () => void;
  onCreate: (input: CreateAttributeInput) => Promise<void>;
}

export function AddFieldDialog({
  open,
  entityType,
  tableRef,
  onClose,
  onCreate,
}: AddFieldDialogProps) {
  const [name, setName] = useState('');
  const [fieldCd, setFieldCd] = useState('');
  const [dataType, setDataType] = useState('string');
  const [section, setSection] = useState('Custom');
  const [description, setDescription] = useState('');
  const [isRequired, setIsRequired] = useState(false);
  const [isSearchable, setIsSearchable] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reset = () => {
    setName('');
    setFieldCd('');
    setDataType('string');
    setSection('Custom');
    setDescription('');
    setIsRequired(false);
    setIsSearchable(true);
    setError(null);
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  const handleCreate = async () => {
    if (!name.trim()) {
      setError('Display name is required');
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onCreate({
        entity_type: entityType,
        table_ref: tableRef,
        name: name.trim(),
        field_cd: fieldCd.trim() || undefined,
        data_type: dataType,
        section,
        description,
        is_required: isRequired,
        is_searchable: isSearchable,
      });
      handleClose();
    } catch (e: any) {
      setError(e?.message || 'Failed to create field');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>New Custom Field · {entityType}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
        <TextField
          label="Display Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          fullWidth
          autoFocus
        />
        <TextField
          label="Field Code (optional)"
          value={fieldCd}
          onChange={(e) => setFieldCd(e.target.value)}
          helperText="Snake_case. Auto-generated from name if empty."
          fullWidth
        />
        <FormControl fullWidth>
          <InputLabel>Data Type</InputLabel>
          <Select
            label="Data Type"
            value={dataType}
            onChange={(e) => setDataType(e.target.value)}
          >
            {DATA_TYPES.map((t) => (
              <MenuItem key={t} value={t}>
                {t}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <TextField
          label="Section"
          value={section}
          onChange={(e) => setSection(e.target.value)}
          fullWidth
        />
        <TextField
          label="Description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          fullWidth
          multiline
          minRows={2}
        />
        <FormControlLabel
          control={
            <Switch checked={isRequired} onChange={(e) => setIsRequired(e.target.checked)} />
          }
          label="Required"
        />
        <FormControlLabel
          control={
            <Switch
              checked={isSearchable}
              onChange={(e) => setIsSearchable(e.target.checked)}
            />
          }
          label="Searchable"
        />
        {error && <div style={{ color: '#c62828' }}>{error}</div>}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>Cancel</Button>
        <Button variant="contained" onClick={handleCreate} disabled={saving}>
          Create Field
        </Button>
      </DialogActions>
    </Dialog>
  );
}
