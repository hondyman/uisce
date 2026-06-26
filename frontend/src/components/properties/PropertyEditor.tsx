import React from 'react';
import { TextField, FormControl, InputLabel, Select, MenuItem, Checkbox, FormControlLabel, Typography, Box } from '@mui/material';
import JsonMonacoEditor from '../editors/JsonMonacoEditor';
import { getJsonSchemaForProperty } from '../../utils/propertyHelpers';
import { useTenant } from '../../contexts/TenantContext';
import { devDebug } from '../../utils/devLogger';
import { useLookupValues } from '../../api/lookups';
import ArrayChips from '../editors/ArrayChips';
import SyntaxPropertyEditor from './SyntaxPropertyEditor';
import type { NodeProperty } from '../../types/nodeTypes';

export interface PropertyEditorProps {
  property: NodeProperty;
  value: any;
  onChange: (v: any) => void;
  error?: string | null;
  allProperties?: Record<string, any> | null;
}

const PropertyEditor: React.FC<PropertyEditorProps> = ({ property, value, onChange, error, allProperties }) => {
  const key = property.name;
  const label = property.label || property.name;

  const { tenant } = useTenant();
  // Load lookup values when property has a lookup configured
  const lookupId = (property as any).lookup_id;
  const cascadeFrom = (property as any).cascade_from;
  
  // For cascading lookups, get the parent value to pass to the API
  let parentId: string | null = null;
  let parentValue: string | null = null;
  if (cascadeFrom && allProperties) {
    const all = allProperties as any;
    const parentSelected = all ? all[cascadeFrom] : undefined;
    if (parentSelected) {
      // If parentSelected looks like a UUID (simple test), use it as parentId
      const s = String(parentSelected);
      const uuidRegex = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
      if (uuidRegex.test(s)) parentId = s;
      else parentValue = s;
    }
  }
  
  // Fetch lookup values - with parent_id filter for cascading
  const { data: lookupValues, isLoading: _lookupLoading } = useLookupValues(tenant?.id, lookupId, parentId, parentValue);
  
  // Debug logging for lookup loading
  React.useEffect(() => {
    if (lookupId) {
      try { devDebug(`[PropertyEditor] Lookup ${property.name}:`, {
        tenant_id: tenant?.id,
        lookup_id: lookupId,
        cascade_from: cascadeFrom,
        parent_id: parentId,
        values_count: lookupValues?.length || 0,
        values: lookupValues
      }); } catch {};
    }
  }, [lookupId, tenant?.id, lookupValues, property.name, cascadeFrom, parentId]);

  // Boolean
  if (property.input_type === 'checkbox') {
    return (
      <FormControlLabel
        key={key}
        control={
          <Checkbox
            checked={!!value}
            onChange={(e) => onChange(e.target.checked)}
          />
        }
        label={label}
        sx={{ display: 'block', mt: 1 }}
      />
    );
  }

  // Options / Selects
  const opts = property.options || (property as any).enumValues;
  if (Array.isArray(opts) && opts.length > 0) {
    return (
      <FormControl fullWidth key={key} margin="normal">
        <InputLabel>{label}</InputLabel>
        <Select
          value={value ?? ''}
          label={label}
          multiple={!!property.validation?.multiple}
          onChange={(e) => onChange(e.target.value)}
        >
          {opts.map((opt) => <MenuItem key={opt} value={opt}>{opt}</MenuItem>)}
        </Select>
        {error && <Typography color="error" variant="caption">{error}</Typography>}
      </FormControl>
    );
  }

  // Lookup-backed selects (external lookup tables)
  if ((property as any).input_type === 'lookup' || (property as any).input_type === 'Lookup') {
    // When cascading, disable if parent value not selected
    const isDisabled = cascadeFrom && !parentId;
    
    // Filter to show only top-level values (no parent) unless we're in a cascading context
    const displayValues = React.useMemo(() => {
      if (!lookupValues) return [];
      // If this is a cascading field, its values MUST have a parent_id that matches the selected parent.
      if (cascadeFrom && parentId) {
        return lookupValues.filter((lv: any) => lv.parent_id === parentId);
      }
      // Otherwise, show only top-level values (where parent_id is null/undefined)
      return lookupValues.filter((lv: any) => !lv.parent_id);
    }, [lookupValues, cascadeFrom, parentId]);
    
    // Create a map from ID to name for displaying selected values
    const lookupNameMap = React.useMemo(() => {
      const map = new Map<string, string>();
      if (displayValues && displayValues.length > 0) {
        displayValues.forEach((lv: any) => {
          if (lv.id && lv.name) {
            map.set(String(lv.id), lv.name);
          }
        });
      }
      return map;
    }, [displayValues]);
    
    return (
      <FormControl fullWidth key={key} margin="normal" disabled={isDisabled}>
        <InputLabel>{label}</InputLabel>
        <Select
          value={value ?? ''}
          label={label}
          multiple={!!property.validation?.multiple}
          onChange={(e) => onChange(e.target.value)}
          renderValue={(selectedValue) => {
            if (!selectedValue) return '';
            if (Array.isArray(selectedValue)) {
              return selectedValue.map((sv) => lookupNameMap.get(String(sv)) || String(sv)).join(', ');
            }
            return lookupNameMap.get(String(selectedValue)) || String(selectedValue);
          }}
        >
          {(displayValues && displayValues.length > 0) ? 
            displayValues.map((lv: any) => (
              <MenuItem key={lv.id ?? lv} value={lv.id ?? lv}>{lv.name ?? lv}</MenuItem>
            ))
            : (opts || []).map((opt: any) => <MenuItem key={opt} value={opt}>{opt}</MenuItem>)
          }
        </Select>
        {isDisabled && <Typography color="warning" variant="caption">Select {cascadeFrom} first</Typography>}
        {error && <Typography color="error" variant="caption">{error}</Typography>}
      </FormControl>
    );
  }

  // Code editor with syntax highlighting
  if (property.input_type === 'code-editor') {
    const syntaxLanguage = (property as any).syntax_language || null;
    return (
      <Box key={key} sx={{ mt: 1 }}>
        <SyntaxPropertyEditor
          value={value ?? ''}
          onChange={onChange}
          language={syntaxLanguage}
          label={label}
          height="200px"
        />
        {error && <Typography color="error" variant="caption">{error}</Typography>}
      </Box>
    );
  }

  // JSON editor
  if (property.input_type === 'json-editor' || property.data_type === 'json') {
    const schema = getJsonSchemaForProperty(property as any);
    return (
      <Box key={key} sx={{ mt: 1 }}>
        <JsonMonacoEditor value={value ?? ''} onChange={onChange} height="160px" schema={schema || undefined} schemaUrn={`inmemory://schema/${property.name}`} />
        {error && <Typography color="error" variant="caption">{error}</Typography>}
      </Box>
    );
  }

  // Array chips
  if (property.data_type === 'array' || property.input_type === 'chips' || property.validation?.multiple) {
    return (
      <Box key={key} sx={{ mt: 1 }}>
        <ArrayChips value={Array.isArray(value) ? value : (value ? String(value).split(',').map((v: string) => v.trim()).filter(Boolean) : [])}
              onChange={onChange}
              label={label}
              placeholder="Add a value"
              displayMap={lookupValues ? new Map((lookupValues || []).map((v: any) => [v.id, v.name])) : null} />
        {error && <Typography color="error" variant="caption">{error}</Typography>}
      </Box>
    );
  }

  // Numbers
  if (property.input_type === 'number' || property.data_type === 'integer' || property.data_type === 'float') {
    return (
      <TextField
        key={key}
        fullWidth
        margin="normal"
        label={label}
        type="number"
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value)}
        error={!!error}
        helperText={error}
      />
    );
  }

  // long text/textarea
  if (property.input_type === 'textarea' || property.input_type === 'text' || property.data_type === 'text') {
    return (
      <TextField
        key={key}
        fullWidth
        margin="normal"
        label={label}
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value)}
        multiline={property.input_type === 'textarea'}
        rows={property.input_type === 'textarea' ? 4 : undefined}
        error={!!error}
        helperText={error}
      />
    );
  }

  // Default: simple text input
  return (
    <TextField key={key} fullWidth margin="normal" label={label} value={value ?? ''} onChange={(e) => onChange(e.target.value)} error={!!error} helperText={error} />
  );
};

export default PropertyEditor;
