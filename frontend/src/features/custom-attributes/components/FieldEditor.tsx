import React, { useEffect, useMemo, useState } from 'react';
import {
  Autocomplete,
  Box,
  Button,
  CircularProgress,
  Divider,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Switch,
  TextField,
  Typography,
  Alert,
} from '@mui/material';
import { CoreIcon, CustomIcon } from '../../../components/common/CoreCustomIcons';
import { apiFetch } from '../../../lib/apiClient';
import type { AttributeDef, UpdateAttributeInput } from '../types';

const DATA_TYPES = [
  'string',
  'integer',
  'decimal',
  'boolean',
  'date',
  'enum',
  'json',
];

interface SemanticTermOption {
  id: string;
  name: string;
  displayName?: string;
}

interface FieldEditorProps {
  field: AttributeDef | null;
  saving?: boolean;
  error?: string | null;
  onSave: (id: string, patch: UpdateAttributeInput) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

function isCoreOrigin(field: AttributeDef): boolean {
  return (field.origin || '').toUpperCase() === 'CORE';
}

async function searchSemanticTerms(query: string): Promise<SemanticTermOption[]> {
  if (!query || query.trim().length < 2) return [];
  try {
    const res = await apiFetch('/api/semantic-terms/search', {
      method: 'POST',
      body: JSON.stringify({ query: query.trim(), limit: 20 }),
    });
    if (!res.ok) return [];
    const data = await res.json();
    const rows = Array.isArray(data) ? data : data.results || data.terms || data.items || [];
    return rows
      .map((t: any) => ({
        id: String(t.id || t.node_id || t.term_id || ''),
        name: String(t.name || t.term_name || t.node_name || t.display_name || ''),
        displayName: t.display_name || t.displayName || t.name || t.term_name,
      }))
      .filter((t: SemanticTermOption) => t.id && t.name);
  } catch {
    return [];
  }
}

export function FieldEditor({ field, saving, error, onSave, onDelete }: FieldEditorProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [dataType, setDataType] = useState('string');
  const [section, setSection] = useState('');
  const [isRequired, setIsRequired] = useState(false);
  const [isSearchable, setIsSearchable] = useState(true);
  const [picklistText, setPicklistText] = useState('');
  const [semanticTerm, setSemanticTerm] = useState<SemanticTermOption | null>(null);
  const [termOptions, setTermOptions] = useState<SemanticTermOption[]>([]);
  const [termQuery, setTermQuery] = useState('');
  const [termLoading, setTermLoading] = useState(false);
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
    if (field.semantic_term_id) {
      setSemanticTerm({
        id: field.semantic_term_id,
        name: field.semantic_term_name || field.semantic_term_id,
        displayName: field.semantic_term_name || undefined,
      });
    } else {
      setSemanticTerm(null);
    }
    setDirty(false);
  }, [field]);

  useEffect(() => {
    let cancelled = false;
    if (termQuery.trim().length < 2) {
      setTermOptions(semanticTerm ? [semanticTerm] : []);
      return;
    }
    setTermLoading(true);
    const handle = window.setTimeout(async () => {
      const opts = await searchSemanticTerms(termQuery);
      if (!cancelled) {
        setTermOptions(opts);
        setTermLoading(false);
      }
    }, 250);
    return () => {
      cancelled = true;
      window.clearTimeout(handle);
    };
  }, [termQuery, semanticTerm]);

  const isCore = useMemo(() => (field ? isCoreOrigin(field) : false), [field]);

  if (!field) {
    return (
      <Box sx={{ p: 2 }}>
        <Typography color="text.secondary">
          Select a field to edit its definition and link a semantic term.
        </Typography>
      </Box>
    );
  }

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
    if (!semanticTerm && field.semantic_term_id) {
      patch.clear_semantic_term = true;
    } else if (semanticTerm?.id) {
      patch.semantic_term_id = semanticTerm.id;
    }
    await onSave(field.id, patch);
    setDirty(false);
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Box sx={{ px: 2, py: 1.5 }}>
        <Stack direction="row" spacing={1} alignItems="center">
          {isCore ? <CoreIcon fontSize="small" /> : <CustomIcon fontSize="small" />}
          <Typography variant="subtitle1" fontWeight={600}>
            {field.name}
          </Typography>
        </Stack>
        <Typography variant="caption" color="text.secondary">
          {field.field_cd} · {isCore ? 'Core' : 'Custom'}
          {field.semantic_term_id ? ' · semantic term linked' : ''}
        </Typography>
      </Box>
      <Divider />
      <Box sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 2, overflow: 'auto', flex: 1 }}>
        {isCore && (
          <Alert severity="info">
            Core definitions are shared. Soft-delete and edits apply only to tenant-owned copies.
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

        <Autocomplete
          options={termOptions}
          loading={termLoading}
          value={semanticTerm}
          disabled={isCore}
          filterOptions={(x) => x}
          getOptionLabel={(o) => o.displayName || o.name || o.id}
          isOptionEqualToValue={(a, b) => a.id === b.id}
          onChange={(_, value) => {
            setSemanticTerm(value);
            setDirty(true);
          }}
          onInputChange={(_, value, reason) => {
            if (reason === 'input') setTermQuery(value);
          }}
          renderInput={(params) => (
            <TextField
              {...params}
              label="Semantic Term"
              size="small"
              helperText="Search and link a glossary semantic term. BO fields using this term resolve to custom_attributes->>'field_cd'."
              InputProps={{
                ...params.InputProps,
                endAdornment: (
                  <>
                    {termLoading ? <CircularProgress color="inherit" size={16} /> : null}
                    {params.InputProps.endAdornment}
                  </>
                ),
              }}
            />
          )}
        />
      </Box>
      <Divider />
      <Box sx={{ p: 1.5, display: 'flex', gap: 1 }}>
        <Button variant="contained" disabled={!dirty || saving || isCore} onClick={handleSave}>
          Save
        </Button>
        <Button color="error" disabled={saving || isCore} onClick={() => onDelete(field.id)}>
          Deactivate
        </Button>
      </Box>
    </Box>
  );
}
