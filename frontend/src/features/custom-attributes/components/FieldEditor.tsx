import React, { useEffect, useState } from 'react';
import {
  Box,
  Button,
  Divider,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  Select,
  Switch,
  TextField,
  Typography,
  Alert,
} from '@mui/material';
import type { AttributeDef, UpdateAttributeInput } from '../types';
// UpdateAttributeInput used by save patch

const DATA_TYPES = [
  'string',
  'integer',
  'decimal',
  'boolean',
  'date',
  'enum',
  'json',
];

interface FieldEditorProps {
  field: AttributeDef | null;
  saving?: boolean;
  error?: string | null;
  onSave: (id: string, patch: UpdateAttributeInput) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

export function FieldEditor({ field, saving, error, onSave, onDelete }: FieldEditorProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [dataType, setDataType] = useState('string');
  const [section, setSection] = useState('');
  const [isRequired, setIsRequired] = useState(false);
  const [isSearchable, setIsSearchable] = useState(true);
  const [picklistText, setPicklistText] = useState('');
  const [semanticTermId, setSemanticTermId] = useState('');
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (!field) return;
    setName(field.name);
    setDescription(field.description || '');
    setDataType(field.data_type);
    setSection(field.section || '');
    setIsRequired(field.is_required);
    setIsSearchable(field.is_searchable);
    setPicklistText((field.picklist_values || []).join('\n'));
    setSemanticTermId(field.semantic_term_id || '');
    setDirty(false);
  }, [field]);

  if (!field) {
    return (
      <Box sx={{ p: 2 }}>
        <Typography color="text.secondary">
          Select a field to edit its definition.
        </Typography>
      </Box>
    );
  }

  const isCore = field.origin === 'CORE';

  const handleSave = async () => {
    const picklist_values = picklistText
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean)
      .map((v) => {
        const n = Number(v);
        return Number.isFinite(n) && v !== '' ? n : v;
      });
    const patch: UpdateAttributeInput = {
      name,
      description,
      data_type: dataType,
      section,
      is_required: isRequired,
      is_searchable: isSearchable,
      picklist_values: dataType === 'enum' || picklist_values.length ? picklist_values : [],
    };
    const trimmedTerm = semanticTermId.trim();
    if (!trimmedTerm && field.semantic_term_id) {
      patch.clear_semantic_term = true;
    } else if (trimmedTerm) {
      patch.semantic_term_id = trimmedTerm;
    }
    await onSave(field.id, patch);
    setDirty(false);
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Box sx={{ px: 2, py: 1.5 }}>
        <Typography variant="subtitle1" fontWeight={600}>
          {field.name}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {field.field_cd} · {field.origin || 'CUSTOM'}
        </Typography>
      </Box>
      <Divider />
      <Box sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 2, overflow: 'auto', flex: 1 }}>
        {isCore && (
          <Alert severity="info">
            Core definitions are shared. Soft-delete and edits apply only to tenant-owned copies.
            Create a tenant override by adding a field with the same code if needed.
          </Alert>
        )}
        {error && <Alert severity="error">{error}</Alert>}

        <TextField
          label="Display Name"
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setDirty(true);
          }}
          disabled={isCore}
          fullWidth
          size="small"
        />

        <TextField
          label="Field Code (immutable)"
          value={field.field_cd}
          fullWidth
          size="small"
          InputProps={{ readOnly: true }}
        />

        <TextField
          label="Section"
          value={section}
          onChange={(e) => {
            setSection(e.target.value);
            setDirty(true);
          }}
          disabled={isCore}
          fullWidth
          size="small"
        />

        <FormControl fullWidth size="small" disabled={isCore}>
          <InputLabel>Data Type</InputLabel>
          <Select
            label="Data Type"
            value={dataType}
            onChange={(e) => {
              setDataType(e.target.value);
              setDirty(true);
            }}
          >
            {DATA_TYPES.map((t) => (
              <MenuItem key={t} value={t}>
                {t}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <TextField
          label="Allowed Values (one per line)"
          value={picklistText}
          onChange={(e) => {
            setPicklistText(e.target.value);
            setDirty(true);
          }}
          disabled={isCore}
          fullWidth
          size="small"
          multiline
          minRows={3}
          helperText="Used for enum / picklist fields"
        />

        <FormControlLabel
          control={
            <Switch
              checked={isRequired}
              disabled={isCore}
              onChange={(e) => {
                setIsRequired(e.target.checked);
                setDirty(true);
              }}
            />
          }
          label="Required"
        />
        <FormControlLabel
          control={
            <Switch
              checked={isSearchable}
              disabled={isCore}
              onChange={(e) => {
                setIsSearchable(e.target.checked);
                setDirty(true);
              }}
            />
          }
          label="Searchable"
        />

        <TextField
          label="Description"
          value={description}
          onChange={(e) => {
            setDescription(e.target.value);
            setDirty(true);
          }}
          disabled={isCore}
          fullWidth
          size="small"
          multiline
          minRows={3}
        />

        <TextField
          label="Semantic Term ID"
          value={semanticTermId}
          onChange={(e) => {
            setSemanticTermId(e.target.value);
            setDirty(true);
          }}
          disabled={isCore}
          fullWidth
          size="small"
          helperText="catalog_node id for the semantic term. BO fields using this term resolve to custom_attributes->>'field_cd'."
          placeholder="uuid of semantic_term node"
        />
        {field.semantic_term_id && (
          <Typography variant="caption" color="text.secondary">
            Bound term: {field.semantic_term_name || field.semantic_term_id}
          </Typography>
        )}
      </Box>
      <Divider />
      <Box sx={{ p: 1.5, display: 'flex', gap: 1 }}>
        <Button
          variant="contained"
          disabled={!dirty || saving || isCore}
          onClick={handleSave}
        >
          Save
        </Button>
        <Button
          color="error"
          disabled={saving || isCore}
          onClick={() => onDelete(field.id)}
        >
          Deactivate
        </Button>
      </Box>
    </Box>
  );
}
