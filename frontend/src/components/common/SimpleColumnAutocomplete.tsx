import React, { useState, useEffect, useMemo } from 'react';
import { Autocomplete, TextField, CircularProgress } from '@mui/material';
import { useLazyQuery } from '@apollo/client';
import { GET_COLUMNS_FOR_TABLE } from '../../graphql/queries/datasourceQueries';
import { devDebug } from '../../utils/devLogger';

export interface SimpleColumnOption {
  id: string;
  node_name?: string;
  qualified_path?: string;
  description?: string;
  catalog_type_name?: string;
  node_type_id?: string;
  properties?: any;
  catalog_defn?: any; // type-level config
}

interface SimpleColumnAutocompleteProps {
  datasourceId?: string;
  parentId?: string; // table id
  value: string | SimpleColumnOption | null;
  onChange: (v: string | SimpleColumnOption | null) => void;
  limit?: number;
  debounceMs?: number;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  fullWidth?: boolean;
  autoFocus?: boolean;
  zIndex?: number;
  minChars?: number;
  showAllOnFocus?: boolean; // fetch % on focus/open when no query
}

const SimpleColumnAutocomplete: React.FC<SimpleColumnAutocompleteProps> = ({
  datasourceId,
  parentId,
  value,
  onChange,
  limit = 50,
  debounceMs = 250,
  placeholder = 'column_name',
  disabled = false,
  className,
  fullWidth = true,
  autoFocus = false,
  minChars = 2,
  showAllOnFocus = false,
}) => {
  const [input, setInput] = useState('');
  const [options, setOptions] = useState<SimpleColumnOption[]>([]);
  const [runQuery, { loading }] = useLazyQuery(GET_COLUMNS_FOR_TABLE, { fetchPolicy: 'network-only' });

  // Decide how to show current value
  const selectedOption = useMemo(() => {
    if (!value) return null;
    if (typeof value === 'string') {
      return (
        options.find(o => o.id === value || o.node_name === value) ||
        { id: value, node_name: value }
      );
    }
    return value;
  }, [value, options]);

  // Fetch on input change with debounce
  useEffect(() => {
    if (!datasourceId || !parentId) { setOptions([]); return; }
    const q = input.trim();
    let active = true;
    const t = setTimeout(async () => {
      try {
        // Only query when input length meets minimum characters (or empty to fetch all)
        if (q && q.length < minChars) {
          if (active) setOptions([]);
          return;
        }
        const vars: any = { datasourceId, parentId, limit };
        if (q.length > 0) vars.q = `%${q}%`;
        const res: any = await runQuery({ variables: vars });
        if (!active) return;
        const rows = (res?.data?.catalog_node_vw || []).map((r: any) => ({
          id: r.node_id || r.id,
          node_name: r.node_name,
          qualified_path: r.qualified_path,
          description: r.description,
          catalog_type_name: r.catalog_type_name,
          node_type_id: r.node_type_id,
          properties: r.properties,
          catalog_defn: r.catalog_defn,
        })) as SimpleColumnOption[];
        // debug: show the first few rows and the raw response to help trace missing qualified_path
        try {
          devDebug('[SimpleColumnAutocomplete] fetched rows (first 5):', rows.slice(0, 5));
          devDebug('[SimpleColumnAutocomplete] raw response:', res?.data?.catalog_node_vw?.slice(0,5));
        } catch (e) { /* no-op */ }
        setOptions(rows);
      } catch {
        if (active) setOptions([]);
      }
    }, debounceMs);
    return () => { active = false; clearTimeout(t); };
  }, [input, datasourceId, parentId, limit, debounceMs, runQuery]);

  // If we have a string value not yet in options, trigger a broad fetch once when parentId changes.
  useEffect(() => {
    if (!datasourceId || !parentId) return;
    if (!value || typeof value !== 'string') return;
    if (options.some(o => o.id === value || o.node_name === value)) return;
    if (!input) setInput('');
  }, [value, options, datasourceId, parentId, input]);

  const handleChange = (_: any, newVal: any) => {
    if (!newVal) { onChange(null); return; }
    if (typeof newVal === 'string') {
      onChange(newVal);
    } else {
      onChange(newVal);
    }
  };

  return (
    <Autocomplete
      options={options}
      value={selectedOption as any}
      onChange={handleChange}
      getOptionLabel={(o: any) => (typeof o === 'string' ? o : (o.node_name || ''))}
      isOptionEqualToValue={(o: any, v: any) => !!o && !!v && (o.id === v.id || o.node_name === v.node_name)}
      inputValue={input}
      onInputChange={(_, val) => setInput(val)}
      loading={loading}
      className={className}
      disabled={disabled || !datasourceId || !parentId}
      fullWidth={fullWidth}
      filterOptions={(x) => x}
      openOnFocus
      freeSolo
  disablePortal={false}
  onOpen={() => devDebug('SimpleColumnAutocomplete opened')}
  onClose={() => devDebug('SimpleColumnAutocomplete closed')}
      onFocus={() => {
        if (showAllOnFocus && !input && options.length === 0 && !loading) setInput('');
      }}
      slotProps={{ popper: { sx: { zIndex: 999999, backgroundColor: 'white', border: '1px solid red' } } as any }}
      renderOption={(props, opt: any) => (
        <li {...props} key={opt.id || opt.node_name} className="sta-option">
          <span className="sta-option-primary">{opt.node_name || opt.qualified_path}</span>
          {opt.description ? <span className="sta-option-secondary">{opt.description}</span> : null}
        </li>
      )}
      loadingText="Loading columns..."
      noOptionsText={(!datasourceId || !parentId) ? 'Select a table first' : (input ? 'No matches' : 'No columns')}
      renderInput={(params) => (
        <TextField
          {...params}
          size="small"
          autoFocus={autoFocus}
          placeholder={placeholder}
          inputProps={{ ...params.inputProps, autoComplete: 'off' }}
          InputProps={{
            ...params.InputProps,
            endAdornment: (
              <>
                {loading ? <CircularProgress size={16} /> : null}
                {params.InputProps.endAdornment}
              </>
            )
          }}
          helperText={(!datasourceId || !parentId) ? 'Select a table to enable column search' : ''}
          FormHelperTextProps={{ style: { minHeight: 16 } }}
        />
      )}
    />
  );
};

export default SimpleColumnAutocomplete;