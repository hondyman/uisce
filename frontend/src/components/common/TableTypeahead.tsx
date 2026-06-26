import React, { useState, useEffect, useRef, useMemo } from 'react';
import { devDebug } from '../../utils/devLogger';
import { Autocomplete, TextField, Box, CircularProgress, Typography, TextFieldProps } from '@mui/material';
import type { AutocompleteRenderInputParams } from '@mui/material/Autocomplete';
import { useLazyQuery } from '@apollo/client';
import { GET_TABLES_FOR_DATASOURCE, GET_CATALOG_NODE_BY_ID } from '../../graphql/queries/datasourceQueries';
import renderCoreCustomChips from './semanticChips';

// debounce helper
const debounce = (fn: (...args: any[]) => void, wait = 300) => {
  let t: any;
  return (...args: any[]) => {
    clearTimeout(t);
    t = setTimeout(() => fn(...args), wait);
  };
};

export type TableOption = {
  tenant_tenant_instance_id?: string;
  source_name?: string;
  id?: string; // node_id
  node_name?: string;
  catalog_type_name?: string;
  node_type_id?: string;
  description?: string;
  qualified_path?: string;
  properties?: any;
  parent_id?: string | null;
  fetchKey?: string;
};

interface TableTypeaheadProps {
  options?: TableOption[];
  loading?: boolean;
  inputValue?: string;
  onInputChange?: (value: string) => void;
  value?: TableOption | string | null;
  onChange?: (value: TableOption | string | null) => void;
  placeholder?: string;
  disabled?: boolean;
  fetchOptions?: boolean;
  datasourceId?: string; // tenant_tenant_instance_id
  pageSize?: number;
  minQueryLength?: number;
  resolveOnFetch?: boolean;
  className?: string;
  // Passed through to the underlying MUI TextField (optional)
  textFieldProps?: Partial<TextFieldProps>;
}

const TableTypeahead: React.FC<TableTypeaheadProps> = ({
  options = [],
  loading = false,
  inputValue,
  onInputChange,
  value,
  onChange,
  placeholder,
  disabled = false,
  fetchOptions = false,
  datasourceId,
  pageSize = 100,
  minQueryLength = 0,
  resolveOnFetch = true,
  className,
  textFieldProps,
}) => {
  const [fetchedOptions, setFetchedOptions] = useState<TableOption[]>([]);
  const [internalLoading, setInternalLoading] = useState(false);
  const abortRef = useRef<AbortController | null>(null);
  // local input state when caller doesn't control inputValue/onInputChange
  const [localInputValue, setLocalInputValue] = useState<string>(inputValue ?? '');

  const [runSearch] = useLazyQuery(GET_TABLES_FOR_DATASOURCE, { fetchPolicy: 'network-only' });
  const [runGetById] = useLazyQuery(GET_CATALOG_NODE_BY_ID, { fetchPolicy: 'network-only' });

  const normalizedOptions = useMemo(() => {
    const source = fetchOptions ? fetchedOptions : options;
    return source.map((opt) => ({
      ...opt,
      fetchKey: opt.fetchKey || opt.id || opt.node_name || opt.qualified_path,
    }));
  }, [fetchOptions, fetchedOptions, options]);

  const effectiveLoading = fetchOptions ? internalLoading : loading;

  const getOptionLabel = (opt: TableOption | string) => {
    if (!opt) return '';
    if (typeof opt === 'string') {
      const lowered = opt.toLowerCase();
      const found = normalizedOptions.find((o) =>
        (o.id && String(o.id).toLowerCase() === lowered) ||
        (o.node_name && o.node_name.toLowerCase() === lowered) ||
        (o.fetchKey && String(o.fetchKey).toLowerCase() === lowered)
      );
      return found ? (found.node_name || String(found.id ?? opt)) : opt;
    }
    try {
      const asAny = opt as any;
      return (asAny.node_name || asAny.qualified_path || asAny.fetchKey || String(asAny.id || '')).toString();
    } catch (e) {
      return String(opt as any || '');
    }
  };

  const isOptionEqualToValue = (option: TableOption, valueIn: TableOption | string | null) => {
    if (!valueIn) return false;
    if (typeof valueIn === 'string') {
      const lowered = valueIn.toLowerCase();
      return Boolean(
        (option.id && String(option.id).toLowerCase() === lowered) ||
        (option.fetchKey && String(option.fetchKey).toLowerCase() === lowered) ||
        (option.node_name && option.node_name.toLowerCase() === lowered)
      );
    }
    const v: any = valueIn as any;
    const keys = ['id', 'fetchKey', 'node_name', 'qualified_path'];
    for (const k of keys) {
      if ((option as any)[k] && v[k] && String((option as any)[k]) === String(v[k])) return true;
    }
    return false;
  };

  const findOption = (identifier?: string | null) => {
    if (!identifier) return undefined;
    const lowered = identifier.toLowerCase();
    return normalizedOptions.find((o) =>
      (o.id && String(o.id).toLowerCase() === lowered) ||
      (o.node_name && o.node_name.toLowerCase() === lowered) ||
      (o.qualified_path && o.qualified_path.toLowerCase() === lowered)
    );
  };

  const displayValue = useMemo(() => {
    if (!value) return null;
    if (typeof value === 'string') return findOption(value) || value;
    const possible = [value.id, (value as any).id, value.node_name]
      .map((v) => (typeof v === 'string' ? v : typeof v === 'number' ? String(v) : undefined))
      .filter((v): v is string => Boolean(v));
    for (const id of possible) {
      const resolved = findOption(id);
      if (resolved) return resolved;
    }
    return value as TableOption;
  }, [value, normalizedOptions]);

  const selectedOption = typeof displayValue === 'string' ? findOption(displayValue) : displayValue;

  // Use controlled inputValue if provided by parent, otherwise fall back to localInputValue
  const currentInput = inputValue !== undefined ? inputValue : localInputValue;

  const effectiveInputValue = useMemo(() => {
    if (!displayValue) return currentInput;
    if (typeof displayValue === 'string') return currentInput;
    try {
      return getOptionLabel(displayValue as TableOption) ?? currentInput;
    } catch (e) {
      return currentInput;
    }
  }, [displayValue, currentInput, normalizedOptions]);

  const renderChips = (option: Partial<TableOption> | null | undefined) => renderCoreCustomChips(option as any);

  // internal fetch logic (debounced) using Apollo lazy query
  useEffect(() => {
    if (!fetchOptions) return;
    const q = (currentInput || '').trim();
    if (q.length < (minQueryLength || 0) && q.length > 0) {
      setFetchedOptions([]);
      return;
    }

    const doFetch = async (query: string) => {
      if (!datasourceId) {
        setFetchedOptions([]);
        return;
      }
      // DEV-LOG: indicate a fetch is about to run (helps debug in browser console)
  devDebug('[TableTypeahead] fetching tables for datasource', datasourceId, 'q=', query);
      setInternalLoading(true);
      try {
        const variables: any = { datasourceId, limit: pageSize };
        if (query && query.length > 0) variables.q = `%${query}%`;
        const res: any = await runSearch({ variables });
        const rows = res?.data?.catalog_node || [];
        const normalized = (rows || []).map((r: any) => ({
          tenant_tenant_instance_id: r.tenant_tenant_instance_id,
          source_name: r.source_name,
          id: r.node_id ?? r.id,
          node_name: r.node_name,
          catalog_type_name: r.catalog_type_name,
          node_type_id: r.node_type_id,
          description: r.description,
          qualified_path: r.qualified_path,
          properties: r.properties,
          parent_id: r.parent_id,
          fetchKey: r.node_id || r.qualified_path || r.node_name,
        } as TableOption));
        setFetchedOptions(normalized);
      } catch (e) {
        setFetchedOptions([]);
      } finally {
        setInternalLoading(false);
      }
    };

  const debounced = debounce(doFetch, 300);
  debounced(q);

    return () => { abortRef.current?.abort?.(); };
  }, [fetchOptions, currentInput, datasourceId, pageSize, minQueryLength, runSearch]);

  // Resolve single string value by fetching catalog node by id
  const resolvingValueRef = useRef<string | null>(null);
  useEffect(() => {
    if (!fetchOptions) return;
    if (!datasourceId) return;
    if (!value || typeof value !== 'string') return;
    const asStr = value.trim();
    if (!asStr) return;
    if (findOption(asStr)) return;
    if (resolvingValueRef.current === asStr) return;

    resolvingValueRef.current = asStr;
    let active = true;

    (async () => {
      try {
        // DEV-LOG: resolving single string value to catalog node
  devDebug('[TableTypeahead] resolving value by id', asStr, 'datasource', datasourceId);
        const res: any = await runGetById({ variables: { datasourceId, nodeId: asStr } });
        const rows = res?.data?.catalog_node || [];
        const v = rows && rows.length > 0 ? rows[0] : null;
        if (!v) {
          resolvingValueRef.current = null;
          return;
        }
        const resolved: TableOption = {
          tenant_tenant_instance_id: v.tenant_tenant_instance_id,
          source_name: v.source_name,
          id: v.node_id ?? v.id,
          node_name: v.node_name,
          catalog_type_name: v.catalog_type_name,
          node_type_id: v.node_type_id,
          description: v.description,
          qualified_path: v.qualified_path,
          properties: v.properties,
          parent_id: v.parent_id,
          fetchKey: v.node_id || v.qualified_path || v.node_name,
        };
        if (!active) return;
        setFetchedOptions((prev) => {
          const exists = prev.some((p) => String(p.id) === String(resolved.id) || String(p.fetchKey) === String(resolved.fetchKey));
          return exists ? prev.map((p) => (String(p.id) === String(resolved.id) ? { ...p, ...resolved } : p)) : [resolved, ...prev];
        });
        if (resolveOnFetch) onChange?.(resolved as any);
      } catch (e) {
        // ignore
      } finally {
        resolvingValueRef.current = null;
      }
    })();

    return () => { active = false; resolvingValueRef.current = null; };
  }, [value, fetchOptions, datasourceId, runGetById, resolveOnFetch, onChange]);

  return (
    <Autocomplete
      options={normalizedOptions}
      loading={effectiveLoading}
      getOptionLabel={getOptionLabel as any}
      isOptionEqualToValue={isOptionEqualToValue as any}
      filterOptions={(x) => x}
      inputValue={effectiveInputValue}
      onInputChange={(_, newInput) => {
        // update local state when uncontrolled
        if (inputValue === undefined) setLocalInputValue(newInput || '');
        onInputChange?.(newInput || '');
      }}
      value={(displayValue ?? null) as any}
      onChange={(_, newValue) => onChange?.(newValue)}
      disabled={disabled}
      freeSolo
      renderOption={(props, option: TableOption) => (
        <li {...props} key={option.id || option.node_name || option.qualified_path}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%', justifyContent: 'space-between' }}>
            <Box>
              <Typography variant="body2">{option.node_name || option.qualified_path}</Typography>
              {option.description ? (
                <Typography variant="caption" color="text.secondary">{option.description}</Typography>
              ) : null}
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>{renderChips(option)}</Box>
          </Box>
        </li>
      )}
      renderInput={(params: AutocompleteRenderInputParams) => (
        <TextField
          {...params}
          size="small"
          placeholder={placeholder}
          className={className}
          {...(textFieldProps || {})}
          InputProps={{
            ...params.InputProps,
            startAdornment: params.InputProps.startAdornment,
            endAdornment: (
              <>
                {renderChips(selectedOption as any)}
                {effectiveLoading ? <CircularProgress color="inherit" size={16} /> : null}
                {params.InputProps.endAdornment}
              </>
            ),
          }}
        />
      )}
      // Ensure the popper (dropdown) renders above other UI chrome
      componentsProps={{ popper: { sx: { zIndex: 1400 } } }}
    />
  );
};

export default TableTypeahead;
