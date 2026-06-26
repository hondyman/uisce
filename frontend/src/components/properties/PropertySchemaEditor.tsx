import React, { FC, useState, useEffect, useCallback } from 'react';
import {
  Box,
  Button,
  TextField,
  IconButton,
  Stack,
  MenuItem,
  Checkbox,
  FormControlLabel,
  Paper,
  Typography,
} from '@mui/material';
import { useTenant } from '../../contexts/TenantContext';
import { useLookups, useInfiniteLookups } from '../../api/lookups';
import { useNodeTypes } from '../../api/nodeTypes'; // Import useNodeTypes
import ProfessionalSearchInput from '../ProfessionalSearchInput';
import DeleteIcon from '@mui/icons-material/Delete';

export type PropertyDef = {
  name: string; // machine name
  label?: string;
  data_type: 'string' | 'number' | 'boolean' | 'date';
  input_type: 'text' | 'select' | 'checkbox' | 'date' | 'lookup' | 'code-editor';
  required?: boolean;
  options?: string[]; // for select/lookups
  lookup?: string | null; // optional lookup id
  cascade_from?: string | null; // optional; name of property this field cascades from
  syntax_language?: 'sql' | 'yaml' | 'json' | null; // optional syntax highlighting language for code-editor input type
  is_array?: boolean; // NEW: Allow multiple values
  lookup_node_type_id?: string | null; // NEW: Node Type ID for lookup
  original_data_type?: 'text' | 'json' | 'integer' | 'float' | 'boolean' | 'date' | 'string'; // Preserve original from backend
  original_input_type?: 'text' | 'textarea' | 'number' | 'json-editor' | 'date-picker' | 'select' | 'checkbox' | 'code-editor'; // Preserve original from backend
};

interface Props {
  value?: PropertyDef[];
  onChange: (next: PropertyDef[]) => void;
}

const emptyProp = (): PropertyDef => ({
  name: '',
  label: '',
  data_type: 'string',
  input_type: 'text',
  required: false,
  options: [],
});

export const PropertySchemaEditor: FC<Props> = ({ value = [], onChange }) => {
  const propsList = value || [];
  const { tenant } = useTenant();
  const { data: nodeTypes = [] } = useNodeTypes(tenant?.id || ''); // Fetch Node Types
  const [lookupQuery, setLookupQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  
  // Debounce the search query by 300ms to avoid excessive API calls
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(lookupQuery);
    }, 300);
    return () => clearTimeout(timer);
  }, [lookupQuery]);

  // Infinite results for dropdown — refetches when debouncedQuery changes
  const lookupsInfinite = useInfiniteLookups(tenant?.id, debouncedQuery, 50);
  const flattenedLookups = (lookupsInfinite?.data?.pages || []).flatMap((p: any) => p.items || []);

  const updateAt = (idx: number, patch: Partial<PropertyDef>) => {
    const next = propsList.map((p, i) => (i === idx ? { ...p, ...patch } : p));
    onChange(next);
  };

  const addProp = () => onChange([...propsList, emptyProp()]);
  const removeAt = (idx: number) => onChange(propsList.filter((_, i) => i !== idx));

  const handleSearch = useCallback((q: string) => {
    setLookupQuery(q);
  }, []);

  return (
    <Box>
      <Stack spacing={1}>
        {propsList.map((p, idx) => (
          <Paper key={idx} variant="outlined" sx={{ p: 2 }}>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="center">
              <TextField
                label="Name"
                value={p.name}
                onChange={(e) => updateAt(idx, { name: e.target.value })}
                helperText="Machine name (lowercase, underscores)"
                size="small"
                sx={{ minWidth: 120 }}
              />

              <TextField
                label="Label"
                value={p.label ?? ''}
                onChange={(e) => updateAt(idx, { label: e.target.value })}
                size="small"
                sx={{ minWidth: 140 }}
              />

              <TextField
                select
                label="Data type"
                value={p.data_type}
                onChange={(e) => updateAt(idx, { data_type: e.target.value as PropertyDef['data_type'] })}
                size="small"
                sx={{ width: 110 }}
              >
                <MenuItem value="string">string</MenuItem>
                <MenuItem value="number">number</MenuItem>
                <MenuItem value="boolean">boolean</MenuItem>
                <MenuItem value="date">date</MenuItem>
              </TextField>

              <TextField
                select
                label="Input"
                value={p.input_type}
                onChange={(e) => updateAt(idx, { input_type: e.target.value as PropertyDef['input_type'] })}
                size="small"
                sx={{ width: 120 }}
              >
                <MenuItem value="text">text</MenuItem>
                <MenuItem value="select">select</MenuItem>
                <MenuItem value="checkbox">checkbox</MenuItem>
                <MenuItem value="date">date</MenuItem>
                <MenuItem value="lookup">lookup</MenuItem>
                <MenuItem value="code-editor">code editor</MenuItem>
              </TextField>

              <FormControlLabel
                control={<Checkbox checked={!!p.required} onChange={(e) => updateAt(idx, { required: e.target.checked })} />}
                label="Required"
              />

              <Box sx={{ flex: 1 }} />

              <IconButton onClick={() => removeAt(idx)} aria-label={`remove-property-${idx}`}>
                <DeleteIcon />
              </IconButton>
            </Stack>

            {p.input_type === 'select' && (
              <Box mt={2}>
                <Typography variant="caption" display="block" gutterBottom>
                  Options (comma separated)
                </Typography>
                <TextField
                  fullWidth
                  size="small"
                  value={(p.options || []).join(',')}
                  onChange={(e) => updateAt(idx, { options: e.target.value.split(',').map((s) => s.trim()).filter(Boolean) })}
                />
              </Box>
            )}
            {p.input_type === 'lookup' && (() => {
              // Find the selected lookup if one is specified
              const selectedLookup = p.lookup ? flattenedLookups?.find((l: any) => l.id === p.lookup) : null;
              const initialSelected = selectedLookup
                ? { id: selectedLookup.id, text: selectedLookup.name, subtext: selectedLookup.description || '', payload: selectedLookup }
                : null;

              return (
                <Box mt={2}>
                  <Typography variant="caption" display="block" gutterBottom>
                    Lookup Table (search by name)
                  </Typography>
                  <ProfessionalSearchInput
                    placeholder="Search lookup by name..."
                    data={(flattenedLookups || []).map((l: any) => ({ id: l.id, text: l.name, subtext: l.description || '', payload: l }))}
                    initialSelected={initialSelected}
                    onSelect={(payload: any) => updateAt(idx, { lookup: payload?.id || null })}
                    onSearch={handleSearch}
                    onLoadMore={() => { if (lookupsInfinite && typeof lookupsInfinite.fetchNextPage === 'function') lookupsInfinite.fetchNextPage(); }}
                  />
                  <Typography variant="caption" display="block" gutterBottom sx={{ mt: 1 }}>
                    Cascading: select a property to filter this lookup by (optional)
                  </Typography>
                  <TextField
                    select
                    fullWidth
                    size="small"
                    value={p.cascade_from || ''}
                    onChange={(e) => updateAt(idx, { cascade_from: e.target.value || null })}
                  >
                    <MenuItem value="">(none)</MenuItem>
                    {propsList.filter((_, i) => i !== idx).map((pp, i) => (
                      <MenuItem key={i} value={pp.name}>{pp.name}</MenuItem>
                    ))}
                  </TextField>
                </Box>
              );
            })()}
            {p.input_type === 'code-editor' && (
              <Box mt={2}>
                <Typography variant="caption" display="block" gutterBottom>
                  Syntax Highlighting Language
                </Typography>
                <TextField
                  select
                  fullWidth
                  size="small"
                  value={p.syntax_language || ''}
                  onChange={(e) => updateAt(idx, { syntax_language: (e.target.value as 'sql' | 'yaml' | 'json' | null) || null })}
                >
                  <MenuItem value="">None (plain text)</MenuItem>
                  <MenuItem value="sql">SQL</MenuItem>
                  <MenuItem value="yaml">YAML</MenuItem>
                  <MenuItem value="json">JSON</MenuItem>
                </TextField>
              </Box>
            )}

             {/* NEW: Node Type Lookup Configuration */}
            {p.input_type === 'lookup' && (
                <Box mt={2}>
                    <Typography variant="caption" display="block" gutterBottom>
                        Target Node Type (for Lookup)
                    </Typography>
                    <TextField
                        select
                        fullWidth
                        size="small"
                        value={p.lookup_node_type_id || ''}
                        onChange={(e) => updateAt(idx, { lookup_node_type_id: e.target.value || null })}
                        helperText="Select the Node Type to look up values from"
                     >
                        <MenuItem value="">Select Node Type...</MenuItem>
                        {nodeTypes.map((nt: any) => (
                            <MenuItem key={nt.id} value={nt.id}>
                                {nt.catalog_type_name}
                            </MenuItem>
                        ))}
                    </TextField>
                </Box>
            )}

            {/* NEW: Allow Multiple (Array) Checkbox */}
             <Box mt={1}>
                <FormControlLabel
                    control={
                        <Checkbox
                            checked={!!p.is_array}
                            onChange={(e) => updateAt(idx, { is_array: e.target.checked })}
                            size="small"
                        />
                    }
                    label={<Typography variant="body2">Allow Multiple Values (List)</Typography>}
                />
            </Box>
          </Paper>
        ))}

        <Button variant="outlined" onClick={addProp} sx={{ mt: 1 }}>
          + Add property
        </Button>
      </Stack>
    </Box>
  );
};

export default PropertySchemaEditor;
